package entity

import "time"

// ConfigMarketCategory 市场同步：一行 = 运动子类 + 一个平台（如 Football + KALSHI）。
type ConfigMarketCategory struct {
	ID               uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	SportCN          string    `gorm:"column:sport_cn;type:varchar(64);not null"`
	SubCategory      string    `gorm:"column:sub_category;type:varchar(64);not null"`
	Platform         string    `gorm:"column:platform;type:varchar(32);not null"`
	Concurrency      int       `gorm:"column:concurrency;type:int;not null;default:1"`
	ConfigJSON       string    `gorm:"column:config_json;type:longtext"`
	OutcomeTypesJSON string    `gorm:"column:outcome_types_json;type:varchar(512);not null;default:'[]'"`
	SortNo           int       `gorm:"column:sort_no;type:int;not null;default:0"`
	IsEnabled        int       `gorm:"column:is_enabled;type:tinyint(1);not null;default:1"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime;not null"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime;not null"`
}

func (ConfigMarketCategory) TableName() string { return "market_category" }