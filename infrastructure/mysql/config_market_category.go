package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type ConfigMarketCategoryImpl struct {
	db *gorm.DB
}

func NewConfigMarketCategoryImpl(db *gorm.DB) repository.ConfigMarketCategoryRepo {
	return &ConfigMarketCategoryImpl{db: db}
}

func (r *ConfigMarketCategoryImpl) ListCategories(ctx context.Context, tx *gorm.DB, enabledOnly bool) ([]entity.ConfigMarketCategory, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(&entity.ConfigMarketCategory{})
	if enabledOnly {
		db = db.Where("is_enabled = ?", 1)
	}
	var list []entity.ConfigMarketCategory
	err := db.Order("platform asc, sort_no asc, sub_category asc").Find(&list).Error
	return list, err
}