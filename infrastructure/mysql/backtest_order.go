package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type BacktestOrderImpl struct {
	db *gorm.DB
}

func NewBacktestOrderImpl(db *gorm.DB) repository.BacktestOrderRepo {
	return &BacktestOrderImpl{db: db}
}

func (r *BacktestOrderImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.BacktestOrder) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).CreateInBatches(list, 100).Error
}

func (r *BacktestOrderImpl) DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	if len(where) == 0 {
		return nil
	}
	db := tx.WithContext(ctx).Model(&entity.BacktestOrder{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Delete(&entity.BacktestOrder{}).Error
}