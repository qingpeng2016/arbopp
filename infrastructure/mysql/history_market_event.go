package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HistoryMarketEventImpl struct {
	db *gorm.DB
}

func NewHistoryMarketEventImpl(db *gorm.DB) repository.HistoryMarketEventRepo {
	return &HistoryMarketEventImpl{db: db}
}

func (r *HistoryMarketEventImpl) CountWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(&entity.HistoryMarketEvent{})
	for cond, val := range where {
		db = db.Where(cond, val)
	}
	var n int64
	return n, db.Count(&n).Error
}

func (r *HistoryMarketEventImpl) BatchCreateIgnoreDup(ctx context.Context, tx *gorm.DB, list []*entity.HistoryMarketEvent) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(list, 200).Error
}

func (r *HistoryMarketEventImpl) DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	if len(where) == 0 {
		return nil
	}
	db := tx.WithContext(ctx).Model(&entity.HistoryMarketEvent{})
	for cond, val := range where {
		db = db.Where(cond, val)
	}
	return db.Delete(&entity.HistoryMarketEvent{}).Error
}