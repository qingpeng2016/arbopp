package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ConfigStrategyFee 按平台和运动配置的吃单手续费系数。
// 费用 = taker_fee_coef * 价格 * (1-价格)，按每个份额计算。
// sub_category 为空表示该平台、该 category 下没单独配置的运动。
type ConfigStrategyFee struct {
	ID           uint            `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Platform     string          `gorm:"column:platform;type:varchar(32);not null;default:''"`
	Category     string          `gorm:"column:category;type:varchar(64);not null;default:''"`
	SubCategory  string          `gorm:"column:sub_category;type:varchar(64);not null;default:''"`
	TakerFeeCoef decimal.Decimal `gorm:"column:taker_fee_coef;type:decimal(20,8);not null"`
	Description  string          `gorm:"column:description;type:varchar(500);not null;default:''"`
	Status       int             `gorm:"column:status;type:tinyint;not null;default:1"`
	CreatedAt    time.Time       `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (ConfigStrategyFee) TableName() string {
	return "config_strategy_fee"
}
