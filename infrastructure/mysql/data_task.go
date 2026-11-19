package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type DataTaskImpl struct {
	db *gorm.DB
}

func NewDataTaskImpl(db *gorm.DB) repository.DataTaskRepo {
	return &DataTaskImpl{db: db}
}

func (r *DataTaskImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.DataTask, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataTask{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.DataTask
	err := db.First(&e).Error
	return e, err
}

func (r *DataTaskImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.DataTask, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataTask{})
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

func (r *DataTaskImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.DataTask) (uint, error) {
	if tx == nil {
		tx = r.db
	}
	err := tx.WithContext(ctx).Create(e).Error
	return e.ID, err
}

func (r *DataTaskImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataTask{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}