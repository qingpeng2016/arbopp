package entity

import "time"

const (
	HistoryRecordKindFile     = "file"
	HistoryRecordKindSnapshot = "snapshot"

	HistoryIngestOK        = "ok"
	HistoryIngestSkipped   = "skipped"
	HistoryIngestFailed    = "failed"
	HistoryIngestSynthetic = "synthetic"

	HistorySourcePmdata    = "pmdata"
	HistorySourceSynthetic = "synthetic"
)

// HistoryDataOrderBook 第三方历史订单簿（文件元数据或快照行）。
type HistoryDataOrderBook struct {
	ID               uint64    `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	DataPlatformID   uint      `gorm:"column:data_platform_id;type:bigint unsigned;not null;default:0"`
	MapKey           string    `gorm:"column:map_key;type:varchar(192);not null;default:''"`
	MarketRowID      uint      `gorm:"column:market_row_id;type:bigint unsigned;not null;default:0"`
	MarketOutcomeID  uint      `gorm:"column:market_outcome_id;type:bigint unsigned;not null;default:0"`
	VenuePlatform    string    `gorm:"column:venue_platform;type:varchar(32);not null;default:''"`
	VendorExchange   string    `gorm:"column:vendor_exchange;type:varchar(32);not null;default:''"`
	VendorSymbol     string    `gorm:"column:vendor_symbol;type:varchar(255);not null;default:''"`
	UTCDate          time.Time `gorm:"column:utc_date;type:date;not null"`
	DataType         string    `gorm:"column:data_type;type:varchar(32);not null;default:l2"`
	Source           string    `gorm:"column:source;type:varchar(16);not null;default:pmdata"`
	RecordKind       string    `gorm:"column:record_kind;type:varchar(16);not null;default:snapshot"`
	SnapshotAtMs     int64     `gorm:"column:snapshot_at_ms;type:bigint;not null;default:0"`
	OrderBookBids    string    `gorm:"column:order_book_bids;type:mediumtext;not null"`
	OrderBookAsks    string    `gorm:"column:order_book_asks;type:mediumtext;not null"`
	LocalFilePath    string    `gorm:"column:local_file_path;type:varchar(512);not null;default:''"`
	FileBytes        uint64    `gorm:"column:file_bytes;type:bigint unsigned;not null;default:0"`
	FileSHA256       string    `gorm:"column:file_sha256;type:char(64);not null;default:''"`
	IngestStatus     string    `gorm:"column:ingest_status;type:varchar(24);not null;default:ok"`
	ErrorText        string    `gorm:"column:error_text;type:varchar(512);not null;default:''"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (HistoryDataOrderBook) TableName() string {
	return "data_history_order_book"
}