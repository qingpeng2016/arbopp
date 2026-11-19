package entity

import "time"

// DataPlatform 第三方历史数据供应配置（API Key、基址等）。
type DataPlatform struct {
	ID           uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	Code         string    `gorm:"column:code;type:varchar(64);not null;default:''"`
	DisplayName  string    `gorm:"column:display_name;type:varchar(128);not null;default:''"`
	VendorType   string    `gorm:"column:vendor_type;type:varchar(32);not null;default:third_party"`
	APIBaseURL   string    `gorm:"column:api_base_url;type:varchar(512);not null;default:''"`
	APIKey       string    `gorm:"column:api_key;type:varchar(512);not null;default:''"`
	ExtraConfig  *string   `gorm:"column:extra_config;type:text"`
	Status       string    `gorm:"column:status;type:varchar(20);not null;default:ACTIVE"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (DataPlatform) TableName() string {
	return "data_platform"
}