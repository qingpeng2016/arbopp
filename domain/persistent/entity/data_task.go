package entity

import "time"

const (
	DataTaskIngestPmdata    = "pmdata"
	DataTaskIngestSynthetic = "synthetic"

	DataTaskStatusPending   = "pending"
	DataTaskStatusRunning   = "running"
	DataTaskStatusCompleted = "completed"
	DataTaskStatusFailed    = "failed"
)

// DataTask 按 map_key 的历史订单簿拉取/伪造任务（回测用 backtest_task）。
type DataTask struct {
	ID           uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	IngestSource string    `gorm:"column:ingest_source;type:varchar(16);not null;default:pmdata"`
	MapKey       string    `gorm:"column:map_key;type:varchar(192);not null;default:''"`
	Params       *string   `gorm:"column:params;type:text"`
	Status       string    `gorm:"column:status;type:varchar(24);not null;default:pending"`
	ErrorText string    `gorm:"column:error_text;type:varchar(512);not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (DataTask) TableName() string {
	return "data_task"
}