package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type ArbBinaryOpportunityRepo interface {
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.ArbBinaryOpportunity, err error)
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.ArbBinaryOpportunity) error
	// UpdateArbProfitAmount 写入实盘成交回补利润。
	UpdateArbProfitAmount(ctx context.Context, id uint, amount float64) error
}
