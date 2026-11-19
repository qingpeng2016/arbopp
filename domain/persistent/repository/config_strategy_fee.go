package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type ConfigStrategyFeeRepo interface {
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) ([]entity.ConfigStrategyFee, error)
}
