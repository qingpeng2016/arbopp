package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type DataPlatformImpl struct {
	db *gorm.DB
}

func NewDataPlatformImpl(db *gorm.DB) repository.DataPlatformRepo {
	return &DataPlatformImpl{db: db}
}

func (r *DataPlatformImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.DataPlatform, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataPlatform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.DataPlatform
	err := db.First(&e).Error
	return e, err
}

func (r *DataPlatformImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.DataPlatform, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataPlatform{})
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

func (r *DataPlatformImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.DataPlatform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}