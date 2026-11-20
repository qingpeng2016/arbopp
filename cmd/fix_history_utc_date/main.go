// 从 local_file_path 修正 utc_date，并打印验收（一次性运维）。
package main

import (
	"fmt"
	"os"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

const mapKey = "v1|英超|epl-eve-che-2026-10-17"

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	res := db.Exec(`
UPDATE ` + "`data_history_order_book`" + `
SET ` + "`utc_date`" + ` = STR_TO_DATE(
  SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(local_file_path, '\\', '/'), '/l2/', -1), '/', 1),
  '%Y-%m-%d'
)
WHERE record_kind IN ('snapshot', 'file')
  AND local_file_path LIKE '%/l2/20%/%'
  AND SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(local_file_path, '\\', '/'), '/l2/', -1), '/', 1) REGEXP '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'`)
	if res.Error != nil {
		fmt.Println("UPDATE ERR:", res.Error)
		os.Exit(1)
	}
	fmt.Printf("rows updated: %d\n", res.RowsAffected)

	type row struct {
		UD   string
		Path string
		N    int64
	}
	var rows []row
	_ = db.Raw(`SELECT DATE_FORMAT(`+"`utc_date`"+`,'%Y-%m-%d') AS ud,
    SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(local_file_path, '\\', '/'), '/l2/', -1), '/', 1) AS path,
    COUNT(*) n
  FROM data_history_order_book
  WHERE map_key=? AND record_kind='snapshot'
  GROUP BY 1,2 ORDER BY path, ud`, mapKey).Scan(&rows)
	fmt.Println("=== snapshot utc_date × path ===")
	var bad int64
	for _, r := range rows {
		if r.UD != r.Path {
			bad += r.N
		}
		fmt.Printf("  utc=%s path=%s n=%d%s\n", r.UD, r.Path, r.N, mark(r.UD == r.Path))
	}
	var mismatch int64
	_ = db.Raw(`SELECT COUNT(*) FROM data_history_order_book
    WHERE map_key=? AND record_kind='snapshot'
    AND DATE_FORMAT(`+"`utc_date`"+`,'%Y-%m-%d') <> SUBSTRING_INDEX(SUBSTRING_INDEX(REPLACE(local_file_path, '\\', '/'), '/l2/', -1), '/', 1)`, mapKey).Scan(&mismatch)
	fmt.Printf("\nmismatch snapshot rows: %d\n", mismatch)
	if mismatch > 0 {
		os.Exit(1)
	}
	fmt.Println("OK")
}

func mark(ok bool) string {
	if ok {
		return " OK"
	}
	return " MISMATCH"
}