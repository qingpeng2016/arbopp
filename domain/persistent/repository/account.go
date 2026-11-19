package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type AccountRepo interface {
	FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.Account, error)
	ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.Account, err error)
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.Account, err error)
	Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error
	Create(ctx context.Context, tx *gorm.DB, e *entity.Account) (id uint, err error)
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.Account) error
}
