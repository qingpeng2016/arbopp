package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type ArbBinaryOrderRepo interface {
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.ArbBinaryOrder) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
}