package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type OrderBookWSRepo interface {
	BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.OrderBookWS) error
}