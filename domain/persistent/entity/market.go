package entity

import (
	"encoding/json"
	"time"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
)

// Market 平台场次/Event（fixture）；可交易盘口见 market_outcome。
type Market struct {
	ID               uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Platform         string    `gorm:"column:platform;type:varchar(32);not null;default:''"`
	Name             *string   `gorm:"column:name;type:varchar(2000)"`
	Desc             *string   `gorm:"column:desc;type:varchar(2000)"`
	PlatformMarketID string    `gorm:"column:platform_market_id;type:varchar(255);not null;default:''"` // 平台场次/Event ID
	MarketURL        string    `gorm:"column:market_url;type:varchar(255);not null;default:''"`
	EventStartTime   *string   `gorm:"column:event_start_time;type:varchar(255)"`
	EventEndTime     *string   `gorm:"column:event_end_time;type:varchar(255)"`
	Status           string    `gorm:"column:status;type:varchar(20);not null;default:ACTIVE"`
	Category         *string   `gorm:"column:category;type:varchar(200)"`
	SubCategory      *string   `gorm:"column:sub_category;type:varchar(200)"`
	SportCN          *string   `gorm:"column:sport_cn;type:varchar(64)"` // 运动分类中文（足球/篮球）
	VolumeTotal      *float64  `gorm:"column:volume_total;type:double"`
	Volume24hr       *float64  `gorm:"column:volume24hr;type:double"`
	Slug             *string   `gorm:"column:slug;type:varchar(255)"`
	League           *string   `gorm:"column:league;type:varchar(128)"`
	Competition      *string   `gorm:"column:competition;type:varchar(64)"`     // 联赛中文名，来自配置 name
	CompetitionEn    *string   `gorm:"column:competition_en;type:varchar(128)"` // 统一英文名，来自配置 name_en
	MapKey           *string   `gorm:"column:map_key;type:varchar(192)"`        // 跨平台场次对齐键（联赛+场次）
	MapVerified      int       `gorm:"column:map_verified;type:tinyint(1);not null;default:0"`
	Config           *string   `gorm:"column:config;type:text"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (Market) TableName() string {
	return "market"
}

type PolymarketMarketConfig struct {
	NegRisk bool `json:"negRisk"`
}

// BuildMarketPlatformConfig 根据平台构建 config JSON
func BuildMarketPlatformConfig(platform string, negRisk bool) string {
	if platform == string(cryptoPredictionEnum.PlatformPolymarket) {
		b, _ := json.Marshal(map[string]bool{"negRisk": negRisk})
		return string(b)
	}
	return ""
}
