package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type HistoryMarketEventRepo interface {
	CountWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) (int64, error)
	BatchCreateIgnoreDup(ctx context.Context, tx *gorm.DB, list []*entity.HistoryMarketEvent) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
}