package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type ArbBinaryOrderImpl struct {
	db *gorm.DB
}

func NewArbBinaryOrderImpl(db *gorm.DB) repository.ArbBinaryOrderRepo {
	return &ArbBinaryOrderImpl{db: db}
}

func (r *ArbBinaryOrderImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.ArbBinaryOrder) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).CreateInBatches(list, 100).Error
}

func (r *ArbBinaryOrderImpl) DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	if len(where) == 0 {
		return nil
	}
	db := tx.WithContext(ctx).Model(&entity.ArbBinaryOrder{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Delete(&entity.ArbBinaryOrder{}).Error
}