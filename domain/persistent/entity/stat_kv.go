package entity

import "time"

// StatKV 运行统计键值（横向扩展）。
type StatKV struct {
	ID          uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	GroupName   string    `gorm:"column:group_name;type:varchar(64);not null"`
	ItemKey     string    `gorm:"column:item_key;type:varchar(128);not null"`
	ItemValue   string    `gorm:"column:item_value;type:text;not null"`
	ValueType   string    `gorm:"column:value_type;type:varchar(32);not null;default:string"`
	Description string    `gorm:"column:description;type:varchar(500);not null;default:''"`
	SortNo      int       `gorm:"column:sort_no;type:int;not null;default:0"`
	IsEnabled   int       `gorm:"column:is_enabled;type:tinyint(1);not null;default:1"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (StatKV) TableName() string {
	return "stat_kv"
}