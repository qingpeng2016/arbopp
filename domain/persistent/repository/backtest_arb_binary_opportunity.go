package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type BacktestArbBinaryOpportunityRepo interface {
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.BacktestArbBinaryOpportunity) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
}