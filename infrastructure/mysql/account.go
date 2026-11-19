package mysql

import (
	"context"
	"fmt"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type AccountImpl struct {
	db *gorm.DB
}

func NewAccountImpl(db *gorm.DB) repository.AccountRepo {
	return &AccountImpl{db: db}
}

func (r *AccountImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.Account, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Account{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.Account
	err := db.First(&e).Error
	return e, err
}

func (r *AccountImpl) ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.Account, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Account{})
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

func (r *AccountImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.Account, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Account{})
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

func (r *AccountImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Account{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}

func (r *AccountImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.Account) (uint, error) {
	if tx == nil {
		tx = r.db
	}
	if err := tx.WithContext(ctx).Model(entity.Account{}).Create(e).Error; err != nil {
		return 0, fmt.Errorf("create account: %w", err)
	}
	return e.ID, nil
}

func (r *AccountImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.Account) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.Account{}).CreateInBatches(list, 100).Error
}
