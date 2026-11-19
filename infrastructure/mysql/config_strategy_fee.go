package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type ConfigStrategyFeeImpl struct {
	db *gorm.DB
}

func NewConfigStrategyFeeImpl(db *gorm.DB) repository.ConfigStrategyFeeRepo {
	return &ConfigStrategyFeeImpl{db: db}
}

func (r *ConfigStrategyFeeImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) ([]entity.ConfigStrategyFee, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.ConfigStrategyFee{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var list []entity.ConfigStrategyFee
	err := db.Find(&list).Error
	return list, err
}
