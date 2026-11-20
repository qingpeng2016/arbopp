package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type OrderBookWSImpl struct {
	db *gorm.DB
}

func NewOrderBookWSImpl(db *gorm.DB) repository.OrderBookWSRepo {
	return &OrderBookWSImpl{db: db}
}

// BatchCreate 追加 WS 快照行（历史表，不做 upsert）。
func (r *OrderBookWSImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.OrderBookWS) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.OrderBookWS{}).CreateInBatches(list, 100).Error
}