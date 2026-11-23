// 从每条腿自己的 leg_N_info 回填实时机会表的平台。
// 预览：go run ./cmd/backfill_arb_opp_platform -run_conf=local -dry-run
// 写入：go run ./cmd/backfill_arb_opp_platform -run_conf=local
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const opportunityTable = "arb_binary_opportunity"

func main() {
	dry := flag.Bool("dry-run", false, "只统计能从 leg_N_info 确认的平台")
	flag.Parse()
	configPath := conf.Get_RUN_CONF()
	switch configPath {
	case "":
		log.Fatal("请用 -run_conf 或 RUN_CONF 指定数据库配置")
	case "local", "dev", "test", "prod":
		configPath = "conf/" + configPath + ".conf.yaml"
	}
	cfg, err := conf.LoadSettingByFilePath(configPath)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.MysqlMasterConf == nil {
		log.Fatal("配置缺少 db")
	}
	source := cfg.MysqlMasterConf
	dsn := driver.NewConfig()
	dsn.Net, dsn.Addr, dsn.DBName = "tcp", source.Host, source.DbName
	dsn.User, dsn.Passwd = source.User, source.Password
	dsn.Timeout = 10 * time.Second
	dsn.Params = map[string]string{"charset": "utf8mb4"}
	db, err := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatal(err)
	}
	var maxID uint64
	if err := db.Table(opportunityTable).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("database=%s table=%s max_id=%d dry_run=%t\n", source.DbName, opportunityTable, maxID, *dry)
	for slot := 1; slot <= 3; slot++ {
		if err := backfillSlot(db, slot, maxID, *dry); err != nil {
			log.Fatalf("leg_%d: %v", slot, err)
		}
	}
}

type legRow struct {
	ID   uint64
	Info string
}

func backfillSlot(db *gorm.DB, slot int, maxID uint64, dry bool) error {
	infoCol := fmt.Sprintf("leg_%d_info", slot)
	platformCol := fmt.Sprintf("leg_%d_platform", slot)
	missing := "COALESCE(TRIM(" + platformCol + "), '') = ''"
	var after uint64
	var scanned, empty, unresolved, resolved, updated int64
	report := func(id uint64, err error) {
		unresolved++
		if unresolved <= 10 {
			fmt.Printf("unresolved id=%d leg=%d: %v\n", id, slot, err)
		}
	}
	for {
		var rows []legRow
		err := db.Table(opportunityTable).Select("id, "+infoCol+" AS info").
			Where("id > ? AND id <= ?", after, maxID).Where(missing).
			Order("id").Limit(256).Find(&rows).Error
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		after = rows[len(rows)-1].ID
		scanned += int64(len(rows))
		candidates := make(map[uint64]legInfo, len(rows))
		var ids []uint64
		for _, row := range rows {
			leg, err := parseLegInfo(row.Info)
			if err != nil {
				report(row.ID, err)
				continue
			}
			if leg == (legInfo{}) {
				empty++
				continue
			}
			candidates[row.ID] = leg
			if leg.OutcomeID != 0 {
				ids = append(ids, leg.OutcomeID)
			}
		}
		platforms, err := lookupPlatforms(db, ids)
		if err != nil {
			return err
		}
		// 只在该腿仍缺平台且 leg_N_info 与解析时逐字节相同的情况下写入。
		byPlatform := make(map[string][][]any)
		for _, row := range rows {
			leg, ok := candidates[row.ID]
			if !ok {
				continue
			}
			platform, err := leg.resolve(platforms)
			if err != nil {
				report(row.ID, err)
				continue
			}
			resolved++
			byPlatform[platform] = append(byPlatform[platform], []any{row.ID, row.Info})
		}
		if dry {
			continue
		}
		for platform, keys := range byPlatform {
			result := db.Table(opportunityTable).Where(missing).
				Where("(id, BINARY "+infoCol+") IN ?", keys).
				UpdateColumns(map[string]any{platformCol: platform, "updated_at": gorm.Expr("updated_at")})
			if result.Error != nil {
				return result.Error
			}
			updated += result.RowsAffected
		}
	}
	fmt.Printf("leg=%d scanned=%d empty=%d unresolved=%d resolved=%d updated=%d\n",
		slot, scanned, empty, unresolved, resolved, updated)
	return nil
}

func lookupPlatforms(db *gorm.DB, ids []uint64) (map[uint64]string, error) {
	platforms := make(map[uint64]string)
	if len(ids) == 0 {
		return platforms, nil
	}
	var outcomes []struct {
		ID       uint64
		Platform string
	}
	if err := db.Table("market_outcome").Select("id, platform").Where("id IN ?", ids).Find(&outcomes).Error; err != nil {
		return nil, err
	}
	for _, outcome := range outcomes {
		platforms[outcome.ID] = outcome.Platform
	}
	return platforms, nil
}
