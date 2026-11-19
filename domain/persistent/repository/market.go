package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type MarketRepo interface {
	FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.Market, error)
	ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.Market, err error)
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.Market, err error)
	Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error
	Create(ctx context.Context, tx *gorm.DB, e *entity.Market) (id uint, err error)
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.Market) error
	// BatchUpsert 按 uk_platform_market_sub；不覆盖 map_key、map_verified
	BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.Market) error
}
