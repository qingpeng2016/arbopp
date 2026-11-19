package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type BacktestArbBinaryOpportunityImpl struct {
	db *gorm.DB
}

func NewBacktestArbBinaryOpportunityImpl(db *gorm.DB) repository.BacktestArbBinaryOpportunityRepo {
	return &BacktestArbBinaryOpportunityImpl{db: db}
}

func (r *BacktestArbBinaryOpportunityImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.BacktestArbBinaryOpportunity) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).CreateInBatches(list, 100).Error
}

func (r *BacktestArbBinaryOpportunityImpl) DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	if len(where) == 0 {
		return nil
	}
	db := tx.WithContext(ctx).Model(entity.BacktestArbBinaryOpportunity{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Delete(&entity.BacktestArbBinaryOpportunity{}).Error
}