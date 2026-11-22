package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ArbBinaryOpportunity 跨平台二元 YES+NO 命中（一行 = 同 map_key 下两边组合）。
type ArbBinaryOpportunity struct {
	ID             uint             `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	OppKind        string           `gorm:"column:opp_kind;type:varchar(32);not null"`
	MapKey              *string          `gorm:"column:map_key;type:varchar(192)"`
	Category            *string          `gorm:"column:category;type:varchar(200)"`
	SubCategory         *string          `gorm:"column:sub_category;type:varchar(200)"`
	Competition         *string          `gorm:"column:competition;type:varchar(64)"`
	CompetitionEn       *string          `gorm:"column:competition_en;type:varchar(128)"`
	YesInfo             string           `gorm:"column:yes_info;type:mediumtext;not null"`
	NoInfo              string           `gorm:"column:no_info;type:mediumtext;not null"`
	YesOrderBooks       string           `gorm:"column:yes_order_books;type:mediumtext;not null"`
	NoOrderBooks        string           `gorm:"column:no_order_books;type:mediumtext;not null"`
	DrawInfo            string           `gorm:"column:draw_info;type:mediumtext;not null"`
	DrawOrderBooks      string           `gorm:"column:draw_order_books;type:mediumtext;not null"`
	YesBestAsk          *decimal.Decimal `gorm:"column:yes_best_ask;type:decimal(20,8)"`
	NoBestAsk           *decimal.Decimal `gorm:"column:no_best_ask;type:decimal(20,8)"`
	DrawBestAsk         *decimal.Decimal `gorm:"column:draw_best_ask;type:decimal(20,8)"`
	TotalCost      decimal.Decimal  `gorm:"column:total_cost;type:decimal(20,8);not null"`
	ProfitSpace    decimal.Decimal  `gorm:"column:profit_space;type:decimal(20,8);not null"`
	ThresholdKey   string           `gorm:"column:threshold_key;type:varchar(64);not null"`
	ThresholdValue decimal.Decimal  `gorm:"column:threshold_value;type:decimal(20,8);not null"`
	BookUpdatedAt  *time.Time       `gorm:"column:book_updated_at;type:datetime"`
	SnapshotAt     time.Time        `gorm:"column:snapshot_at;type:datetime;not null"`
	SnapshotAtMs   int64            `gorm:"column:snapshot_at_ms;type:bigint;not null;default:0"`
	CreatedAt      time.Time        `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time        `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (ArbBinaryOpportunity) TableName() string {
	return "arb_binary_opportunity"
}