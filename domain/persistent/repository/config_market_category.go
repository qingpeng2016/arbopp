package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

// ConfigMarketCategoryRepo 市场同步运动分类配置（一行一类）。
type ConfigMarketCategoryRepo interface {
	ListCategories(ctx context.Context, tx *gorm.DB, enabledOnly bool) ([]entity.ConfigMarketCategory, error)
}