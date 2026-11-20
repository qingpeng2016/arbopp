package entity

import "time"

// OrderBook 报价快照（预测订单簿 / 赔率线）
type OrderBook struct {
	ID            int64     `gorm:"primaryKey;column:id;type:bigint;not null;autoIncrement"`
	MapKey           *string `gorm:"column:map_key;type:varchar(192)"`
	Category         *string `gorm:"column:category;type:varchar(200)"`
	SubCategory      *string `gorm:"column:sub_category;type:varchar(200)"`
	Competition      *string `gorm:"column:competition;type:varchar(64)"`
	CompetitionEn    *string `gorm:"column:competition_en;type:varchar(128)"`
	Platform         string  `gorm:"column:platform;type:varchar(32);not null;default:''"`
	MarketRowID      uint    `gorm:"column:market_row_id;type:bigint unsigned;not null;default:0"` // 父场次 market.id
	PlatformMarketID string  `gorm:"column:platform_market_id;type:varchar(255);not null;default:''"`
	MarketName       *string `gorm:"column:market_name;type:text"`
	OutcomeName      *string `gorm:"column:outcome_name;type:text"`
	TokenName        *string `gorm:"column:token_name;type:varchar(255)"`
	MarketURL               string `gorm:"column:market_url;type:varchar(255);default:''"`
	PlatformMarketOutcomeID string `gorm:"column:platform_market_outcome_id;type:varchar(255);not null;default:''"` // 平台盘口 ID
	MarketAddress string    `gorm:"column:market_address;type:varchar(255);default:''"`
	TokenID       string    `gorm:"column:token_id;type:varchar(160);not null;default:''"`
	SelectionID   *string   `gorm:"column:selection_id;type:varchar(255)"`
	OrderBookAsks string    `gorm:"column:order_book_asks;type:text;not null"`
	OrderBookBids string    `gorm:"column:order_book_bids;type:text;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (OrderBook) TableName() string {
	return "market_order_book"
}
