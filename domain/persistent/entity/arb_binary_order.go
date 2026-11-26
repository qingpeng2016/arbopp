package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ArbBinaryOrder 实盘 strategy_entry 组合腿下单（表 orders；回测见 BacktestOrder）。
type ArbBinaryOrder struct {
	ID                      uint            `gorm:"primaryKey;column:id;type:bigint unsigned;not null;autoIncrement"`
	BacktestTaskID          uint            `gorm:"column:backtest_task_id;type:bigint unsigned;not null;default:0"`
	StrategyType            string          `gorm:"column:strategy_type;type:varchar(32);not null;default:''"`
	ArbOppID                uint            `gorm:"column:arb_opp_id;type:bigint unsigned;not null"`
	MapKey                  string          `gorm:"column:map_key;type:varchar(192);not null"`
	SideNo                  uint8           `gorm:"column:side_no;type:tinyint unsigned;not null"`
	Platform                string          `gorm:"column:platform;type:varchar(32);not null"`
	PlatformMarketOutcomeID string          `gorm:"column:platform_market_outcome_id;type:varchar(255);not null;default:''"`
	TokenID                 string          `gorm:"column:token_id;type:varchar(255);not null;default:''"`
	ClientOrderID           string          `gorm:"column:client_order_id;type:varchar(64);not null;default:''"`
	PlatformOrderID         string          `gorm:"column:platform_order_id;type:varchar(128);not null;default:''"`
	Side                    string          `gorm:"column:side;type:varchar(8);not null"`
	Action                  string          `gorm:"column:action;type:varchar(16);not null;default:BUY"`
	Price                   decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null"`
	Qty                     decimal.Decimal `gorm:"column:qty;type:decimal(20,8);not null"`
	TradeQty                decimal.Decimal `gorm:"column:trade_qty;type:decimal(20,8);not null;default:0"`     // 实际成交份额（fill sync 回填）
	TradePrice              decimal.Decimal `gorm:"column:trade_price;type:decimal(20,8);not null;default:0"`   // 实际成交均价
	TradeAmount             decimal.Decimal `gorm:"column:trade_amount;type:decimal(20,8);not null;default:0"` // trade_price*trade_qty
	Fee                     decimal.Decimal `gorm:"column:fee;type:decimal(20,8);not null;default:0"`
	Status                  string          `gorm:"column:status;type:varchar(32);not null"`
	ErrMsg                  string          `gorm:"column:err_msg;type:varchar(1024);not null;default:''"`
	SentAt                  time.Time       `gorm:"column:sent_at;type:datetime;not null"`
	CreatedAt               time.Time       `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt               time.Time       `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

func (ArbBinaryOrder) TableName() string {
	return "orders"
}