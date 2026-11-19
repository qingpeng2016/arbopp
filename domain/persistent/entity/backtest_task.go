package entity

import "time"

const (
	// BacktestStrategyStrategyEntry 回放 strategy_entry：二元 yesno + 足球 1X2 onextwo。
	BacktestStrategyStrategyEntry = "strategy_entry"
	// BacktestStrategyYesNo 历史任务码，与 strategy_entry 等价（双策略）。
	BacktestStrategyYesNo = "yesno"
	// BacktestStrategyOneX2 仅回放 1X2。
	BacktestStrategyOneX2 = "onextwo"

	BacktestTaskStatusPending   = "pending"
	BacktestTaskStatusRunning   = "running"
	BacktestTaskStatusCompleted = "completed"
	BacktestTaskStatusFailed    = "failed"
)

// BacktestTask 历史订单簿回放回测任务。
type BacktestTask struct {
	ID           uint      `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	MapKey       string    `gorm:"column:map_key;type:varchar(192);not null;default:''"`
	StrategyCode string    `gorm:"column:strategy_code;type:varchar(32);not null;default:strategy_entry"`
	Status       string    `gorm:"column:status;type:varchar(24);not null;default:pending"`
	Progress     uint8     `gorm:"column:progress;type:tinyint unsigned;not null;default:0"`
	ErrorText    string    `gorm:"column:error_text;type:varchar(512);not null;default:''"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (BacktestTask) TableName() string {
	return "backtest_task"
}