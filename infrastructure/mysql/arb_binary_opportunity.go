package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ArbBinaryOpportunityImpl struct {
	db *gorm.DB
}

func NewArbBinaryOpportunityImpl(db *gorm.DB) repository.ArbBinaryOpportunityRepo {
	return &ArbBinaryOpportunityImpl{db: db}
}

func (r *ArbBinaryOpportunityImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.ArbBinaryOpportunity, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.ArbBinaryOpportunity{})
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

func (r *ArbBinaryOpportunityImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.ArbBinaryOpportunity) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).CreateInBatches(list, 100).Error
}

func (r *ArbBinaryOpportunityImpl) UpdateArbProfitAmount(ctx context.Context, id uint, amount float64) error {
	if id == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&entity.ArbBinaryOpportunity{}).
		Where("id = ?", id).
		Update("arb_profit_amount", decimal.NewFromFloat(amount).Round(8)).Error
}
