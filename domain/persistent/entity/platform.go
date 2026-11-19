package entity

import "time"

// Platform 预测市场 + 赔率平台连接配置
type Platform struct {
	ID               uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Slug             string    `gorm:"column:slug;type:varchar(50);not null;comment:POLYMARKET, KALSHI …"`
	PlatformType     string    `gorm:"column:platform_type;type:varchar(20);not null;default:PREDICTION"`
	QuoteStyle       string    `gorm:"column:quote_style;type:varchar(24);not null;default:PROBABILITY"`
	DisplayName      string    `gorm:"column:display_name;type:varchar(64);not null;default:''"`
	Timeout          int       `gorm:"column:timeout;type:int;not null;default:30"`
	RetryTimes       int       `gorm:"column:retry_times;type:int;not null;default:3"`
	RetryInterval    int       `gorm:"column:retry_interval;type:int;not null;default:1"`
	MarketAPIBaseURL *string   `gorm:"column:market_api_base_url;type:varchar(512)"`
	OrderAPIBaseURL  *string   `gorm:"column:order_api_base_url;type:varchar(512)"`
	WebsocketURL     *string   `gorm:"column:websocket_url;type:varchar(512)"`
	UserWebsocketURL *string   `gorm:"column:user_websocket_url;type:varchar(512)"`
	GrpcHost         *string   `gorm:"column:grpc_host;type:varchar(255)"`
	FgAPIHost        *string   `gorm:"column:fg_api_host;type:varchar(255)"`
	ExtraConfig      *string   `gorm:"column:extra_config;type:text"`
	Status           string    `gorm:"column:status;type:varchar(20);not null;default:ACTIVE"`
	Sort             int       `gorm:"column:sort;type:int;not null;default:0"`
	Remark           string    `gorm:"column:remark;type:varchar(500);not null;default:''"`
	CreatedAt        time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (Platform) TableName() string {
	return "platform"
}
