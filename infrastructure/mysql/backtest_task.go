package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type BacktestTaskImpl struct {
	db *gorm.DB
}

func NewBacktestTaskImpl(db *gorm.DB) repository.BacktestTaskRepo {
	return &BacktestTaskImpl{db: db}
}

func (r *BacktestTaskImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.BacktestTask, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.BacktestTask{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	if err = db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	err = db.Find(&list).Error
	return total, list, err
}

func (r *BacktestTaskImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.BacktestTask{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}