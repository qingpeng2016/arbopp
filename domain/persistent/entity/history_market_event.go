package entity

import "time"

const (
	HistoryEventKindBook         = "book"
	HistoryEventKindPriceChange  = "price_change"
	HistoryEventKindOrderbook    = "orderbook"
)

// HistoryMarketEvent PMData Parquet 原始行（WS 事件形态）。
type HistoryMarketEvent struct {
	ID              uint64    `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	DataPlatformID  uint      `gorm:"column:data_platform_id;type:bigint unsigned;not null;default:0"`
	MapKey          string    `gorm:"column:map_key;type:varchar(192);not null;default:''"`
	MarketRowID     uint      `gorm:"column:market_row_id;type:bigint unsigned;not null;default:0"`
	MarketOutcomeID uint      `gorm:"column:market_outcome_id;type:bigint unsigned;not null;default:0"`
	VenuePlatform   string    `gorm:"column:venue_platform;type:varchar(32);not null;default:''"`
	VendorExchange  string    `gorm:"column:vendor_exchange;type:varchar(32);not null;default:''"`
	VendorSymbol    string    `gorm:"column:vendor_symbol;type:varchar(255);not null;default:''"`
	FileCalendarDay string    `gorm:"column:file_calendar_day;not null"`
	DataType        string    `gorm:"column:data_type;type:varchar(32);not null;default:l2"`
	EventKind       string    `gorm:"column:event_kind;type:varchar(32);not null;default:''"`
	EventType       string    `gorm:"column:event_type;type:varchar(64);not null;default:''"`
	EventAtMs       int64     `gorm:"column:event_at_ms;type:bigint;not null;default:0"`
	LocalEventAtMs  int64     `gorm:"column:local_event_at_ms;type:bigint;not null;default:0"`
	RowIndex        uint32    `gorm:"column:row_index;type:int unsigned;not null;default:0"`
	PayloadJSON     string    `gorm:"column:payload_json;type:mediumtext;not null"`
	LocalFilePath   string    `gorm:"column:local_file_path;type:varchar(512);not null;default:''"`
	FileBytes       uint64    `gorm:"column:file_bytes;type:bigint unsigned;not null;default:0"`
	FileSHA256      string    `gorm:"column:file_sha256;type:char(64);not null;default:''"`
	IngestStatus    string    `gorm:"column:ingest_status;type:varchar(24);not null;default:ok"`
	CreatedAt       time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (HistoryMarketEvent) TableName() string {
	return "data_history_market_event"
}