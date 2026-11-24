package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type ArbBinaryOrderRepo interface {
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.ArbBinaryOrder) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
	// ListForFillSync 待回补成交的实盘单（有 platform_order_id）。
	ListForFillSync(ctx context.Context, limit int, onlyMissing bool) ([]*entity.ArbBinaryOrder, error)
	UpdateFill(ctx context.Context, id uint, fillQty, fee float64, status, errMsg string) error
	// HasNonTerminalByMapKey 该 map_key 是否仍有未终态订单（阻塞新开仓）。
	HasNonTerminalByMapKey(ctx context.Context, mapKey string) (bool, error)
}