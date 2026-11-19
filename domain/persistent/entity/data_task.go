package entity

import "time"

const (
	DataTaskTypeHistoryL2   = "history_l2"
	DataTaskTypeSyntheticL2 = "synthetic_l2"

	DataTaskStatusPending   = "pending"
	DataTaskStatusRunning   = "running"
	DataTaskStatusCompleted = "completed"
	DataTaskStatusFailed    = "failed"
)

// DataTask 按 map_key 的数据拉取或回测任务。
type DataTask struct {
	ID        uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	TaskType  string    `gorm:"column:task_type;type:varchar(32);not null;default:history_l2"`
	MapKey    string    `gorm:"column:map_key;type:varchar(192);not null;default:''"`
	Status    string    `gorm:"column:status;type:varchar(24);not null;default:pending"`
	ErrorText string    `gorm:"column:error_text;type:varchar(512);not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (DataTask) TableName() string {
	return "data_task"
}