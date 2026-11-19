package mysql

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type PlatformImpl struct {
	db *gorm.DB
}

func NewPlatformImpl(db *gorm.DB) repository.PlatformRepo {
	return &PlatformImpl{db: db}
}

func (r *PlatformImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.Platform, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Platform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.Platform
	err := db.First(&e).Error
	return e, err
}

func (r *PlatformImpl) ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.Platform, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Platform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	for key, values := range or {
		db = db.Or(key, values...)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	db = db.Count(&total)
	if size != 0 {
		db = db.Offset(size * (page - 1)).Limit(size)
	}
	err = db.Find(&list).Error
	return total, list, err
}

func (r *PlatformImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.Platform, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Platform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	err = db.Count(&total).Find(&list).Error
	return total, list, err
}

func (r *PlatformImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Platform{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}

func (r *PlatformImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.Platform) (uint, error) {
	if tx == nil {
		tx = r.db
	}
	if err := tx.WithContext(ctx).Model(entity.Platform{}).Create(e).Error; err != nil {
		return 0, fmt.Errorf("create platform: %w", err)
	}
	return e.ID, nil
}

func (r *PlatformImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.Platform) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.Platform{}).CreateInBatches(list, 100).Error
}
