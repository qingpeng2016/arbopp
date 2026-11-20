package main

import (
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	var dbName string
	_ = db.Raw("SELECT DATABASE()").Scan(&dbName)
	fmt.Println("database:", dbName)

	var cols []struct {
		Field string `gorm:"column:Field"`
		Type  string `gorm:"column:Type"`
	}
	_ = db.Raw("SHOW COLUMNS FROM data_task").Scan(&cols)
	fmt.Println("data_task columns:")
	for _, c := range cols {
		fmt.Println(" ", c.Field, c.Type)
	}
	var hasTaskType, hasIngest, hasParams int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name='data_task' AND column_name='task_type'`).Scan(&hasTaskType)
	db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name='data_task' AND column_name='ingest_source'`).Scan(&hasIngest)
	db.Raw(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name='data_task' AND column_name='params'`).Scan(&hasParams)
	fmt.Printf("task_type exists=%v ingest_source exists=%v params exists=%v\n", hasTaskType > 0, hasIngest > 0, hasParams > 0)

	var rows []struct {
		ID           uint
		IngestSource string
		MapKey       string
		Status       string
	}
	_ = db.Raw("SELECT id, ingest_source, map_key, status FROM data_task").Scan(&rows)
	fmt.Println("rows:", len(rows))
	for _, r := range rows {
		fmt.Printf("  id=%d ingest_source=%s map_key=%q status=%s\n", r.ID, r.IngestSource, r.MapKey, r.Status)
	}
}