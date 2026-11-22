package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// BacktestArbBinaryOpportunity 回测命中的跨平台套利机会（腿槽位同 ArbBinaryOpportunity）。
type BacktestArbBinaryOpportunity struct {
	ID             uint             `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	BacktestTaskID uint             `gorm:"column:backtest_task_id;type:bigint unsigned;not null"`
	OppKind        string           `gorm:"column:opp_kind;type:varchar(32);not null"`
	MapKey              *string          `gorm:"column:map_key;type:varchar(192)"`
	Category            *string          `gorm:"column:category;type:varchar(200)"`
	SubCategory         *string          `gorm:"column:sub_category;type:varchar(200)"`
	Competition         *string          `gorm:"column:competition;type:varchar(64)"`
	CompetitionEn       *string          `gorm:"column:competition_en;type:varchar(128)"`
	Leg1Info            string           `gorm:"column:leg_1_info;type:mediumtext;not null"`
	Leg1OrderBooks      string           `gorm:"column:leg_1_order_books;type:mediumtext;not null"`
	Leg2Info            string           `gorm:"column:leg_2_info;type:mediumtext;not null"`
	Leg2OrderBooks      string           `gorm:"column:leg_2_order_books;type:mediumtext;not null"`
	Leg3Info            string           `gorm:"column:leg_3_info;type:mediumtext;not null"`
	Leg3OrderBooks      string           `gorm:"column:leg_3_order_books;type:mediumtext;not null"`
	Leg1BestAsk         *decimal.Decimal `gorm:"column:leg_1_best_ask;type:decimal(20,8)"`
	Leg2BestAsk         *decimal.Decimal `gorm:"column:leg_2_best_ask;type:decimal(20,8)"`
	Leg3BestAsk         *decimal.Decimal `gorm:"column:leg_3_best_ask;type:decimal(20,8)"`
	TotalCost      decimal.Decimal  `gorm:"column:total_cost;type:decimal(20,8);not null"`
	ProfitSpace    decimal.Decimal  `gorm:"column:profit_space;type:decimal(20,8);not null"`
	ArbQty         decimal.Decimal  `gorm:"column:arb_qty;type:decimal(20,8);not null;default:0"`
	ArbProfit      decimal.Decimal  `gorm:"column:arb_profit;type:decimal(20,8);not null;default:0"`
	ThresholdKey   string           `gorm:"column:threshold_key;type:varchar(64);not null"`
	ThresholdValue decimal.Decimal  `gorm:"column:threshold_value;type:decimal(20,8);not null"`
	BookUpdatedAt  *time.Time       `gorm:"column:book_updated_at;type:datetime"`
	SnapshotAt     time.Time        `gorm:"column:snapshot_at;type:datetime;not null"`
	SnapshotAtMs   int64            `gorm:"column:snapshot_at_ms;type:bigint;not null;default:0"`
	CreatedAt      time.Time        `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time        `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (BacktestArbBinaryOpportunity) TableName() string {
	return "backtest_arb_binary_opportunity"
}