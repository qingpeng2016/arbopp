package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ConfigStrategyThreshold 扫描阈值（0.95/1.9 等）
type ConfigStrategyThreshold struct {
	ID          uint            `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	ConfigKey   string          `gorm:"column:config_key;type:varchar(64);not null"`
	ConfigValue decimal.Decimal `gorm:"column:config_value;type:decimal(20,8);not null"`
	ValueType   string          `gorm:"column:value_type;type:varchar(16);not null;default:DECIMAL"`
	Description *string         `gorm:"column:description;type:varchar(500)"`
	Status      int             `gorm:"column:status;type:tinyint;not null;default:1"`
	CreatedAt   time.Time       `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (ConfigStrategyThreshold) TableName() string {
	return "config_strategy_threshold"
}
