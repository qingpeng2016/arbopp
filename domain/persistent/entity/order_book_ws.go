package entity

import "time"

// OrderBookWS strategy_entry WebSocket 写入的盘口快照（表 market_order_book_ws）。
type OrderBookWS struct {
	ID                      int64     `gorm:"primaryKey;column:id;type:bigint;not null;autoIncrement"`
	MapKey                  *string   `gorm:"column:map_key;type:varchar(192)"`
	Category                *string   `gorm:"column:category;type:varchar(200)"`
	SubCategory             *string   `gorm:"column:sub_category;type:varchar(200)"`
	Competition             *string   `gorm:"column:competition;type:varchar(64)"`
	CompetitionEn           *string   `gorm:"column:competition_en;type:varchar(128)"`
	Platform                string    `gorm:"column:platform;type:varchar(32);not null;default:''"`
	MarketRowID             uint      `gorm:"column:market_row_id;type:bigint unsigned;not null;default:0"`
	PlatformMarketID        string    `gorm:"column:platform_market_id;type:varchar(255);not null;default:''"`
	MarketName              *string   `gorm:"column:market_name;type:text"`
	OutcomeName             *string   `gorm:"column:outcome_name;type:text"`
	TokenName               *string   `gorm:"column:token_name;type:varchar(255)"`
	MarketURL               string    `gorm:"column:market_url;type:varchar(255);default:''"`
	PlatformMarketOutcomeID string    `gorm:"column:platform_market_outcome_id;type:varchar(255);not null;default:''"`
	MarketAddress           string    `gorm:"column:market_address;type:varchar(255);default:''"`
	TokenID                 string    `gorm:"column:token_id;type:varchar(160);not null;default:''"`
	SelectionID             *string   `gorm:"column:selection_id;type:varchar(255)"`
	OrderBookAsks           string    `gorm:"column:order_book_asks;type:text;not null"`
	OrderBookBids           string    `gorm:"column:order_book_bids;type:text;not null"`
	CreatedAt               time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt               time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (OrderBookWS) TableName() string {
	return "market_order_book_ws"
}

// OrderBookWSCopy 从 REST 盘口行拷贝为 WS 行（深拷贝字符串指针字段）。
func OrderBookWSCopy(b *OrderBook) *OrderBookWS {
	if b == nil {
		return nil
	}
	out := &OrderBookWS{
		Platform:                b.Platform,
		MarketRowID:             b.MarketRowID,
		PlatformMarketID:        b.PlatformMarketID,
		MarketURL:               b.MarketURL,
		PlatformMarketOutcomeID: b.PlatformMarketOutcomeID,
		MarketAddress:           b.MarketAddress,
		TokenID:                 b.TokenID,
		OrderBookAsks:           b.OrderBookAsks,
		OrderBookBids:           b.OrderBookBids,
	}
	if b.MapKey != nil {
		v := *b.MapKey
		out.MapKey = &v
	}
	if b.Category != nil {
		v := *b.Category
		out.Category = &v
	}
	if b.SubCategory != nil {
		v := *b.SubCategory
		out.SubCategory = &v
	}
	if b.Competition != nil {
		v := *b.Competition
		out.Competition = &v
	}
	if b.CompetitionEn != nil {
		v := *b.CompetitionEn
		out.CompetitionEn = &v
	}
	if b.MarketName != nil {
		v := *b.MarketName
		out.MarketName = &v
	}
	if b.OutcomeName != nil {
		v := *b.OutcomeName
		out.OutcomeName = &v
	}
	if b.TokenName != nil {
		v := *b.TokenName
		out.TokenName = &v
	}
	if b.SelectionID != nil {
		v := *b.SelectionID
		out.SelectionID = &v
	}
	return out
}