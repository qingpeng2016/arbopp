package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

// BacktestOrderRepo 回测订单（backtest_orders）。
type BacktestOrderRepo interface {
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.BacktestOrder) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
}