package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type MarketOutcomeRepo interface {
	FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.MarketOutcome, error)
	ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.MarketOutcome, err error)
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.MarketOutcome, err error)
	Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error
	Create(ctx context.Context, tx *gorm.DB, e *entity.MarketOutcome) (id uint, err error)
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.MarketOutcome) error
	// BatchUpsert 按 uk_platform_outcome_sub；不覆盖 map_key（由 market_map 从 market 同步到 outcome）
	BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.MarketOutcome) error
}