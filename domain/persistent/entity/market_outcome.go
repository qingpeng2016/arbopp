package entity

import (
	"time"
)

// MarketOutcome 单场下的可交易盘口（Polymarket condition / Kalshi market ticker）。
type MarketOutcome struct {
	ID                      uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Platform                string    `gorm:"column:platform;type:varchar(32);not null;default:''"`
	MarketRowID             uint      `gorm:"column:market_row_id;type:bigint unsigned;not null;default:0"`    // 父场次 market.id
	PlatformMarketID        string    `gorm:"column:platform_market_id;type:varchar(255);not null;default:''"` // 父场次平台 ID，对应 market.platform_market_id
	Name                    *string   `gorm:"column:name;type:varchar(2000)"`
	Desc                    *string   `gorm:"column:desc;type:varchar(2000)"`
	PlatformMarketOutcomeID string    `gorm:"column:platform_market_outcome_id;type:varchar(255);not null;default:''"` // 平台盘口 ID
	MarketAddress           *string   `gorm:"column:market_address;type:varchar(255)"`
	MarketURL               string    `gorm:"column:market_url;type:varchar(255);not null;default:''"`
	YesTokenID              string    `gorm:"column:yes_token_id;type:varchar(255);not null;default:''"`
	NoTokenID               string    `gorm:"column:no_token_id;type:varchar(255);not null;default:''"`
	SelectionID             *string   `gorm:"column:selection_id;type:varchar(255)"`
	MarketStartTime         *string   `gorm:"column:market_start_time;type:varchar(255)"`
	MarketEndTime           *string   `gorm:"column:market_end_time;type:varchar(255)"`
	Category                *string   `gorm:"column:category;type:varchar(200)"`
	SubCategory             *string   `gorm:"column:sub_category;type:varchar(200)"`
	OutcomeType             string    `gorm:"column:outcome_type;type:varchar(32);not null;default:moneyline"`
	VolumeTotal             *float64  `gorm:"column:volume_total;type:double"`
	Volume24hr              *float64  `gorm:"column:volume24hr;type:double"`
	Slug                    *string   `gorm:"column:slug;type:varchar(255)"`
	Competition             *string   `gorm:"column:competition;type:varchar(64)"`
	CompetitionEn           *string   `gorm:"column:competition_en;type:varchar(128)"`
	Config                  *string   `gorm:"column:config;type:text"`
	CreatedAt               time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt               time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (MarketOutcome) TableName() string {
	return "market_outcome"
}
