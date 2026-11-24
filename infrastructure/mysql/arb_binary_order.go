package mysql

import (
	"context"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"github.com/shopspring/decimal"
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

func (r *ArbBinaryOrderImpl) ListForFillSync(ctx context.Context, limit int, onlyMissing bool) ([]*entity.ArbBinaryOrder, error) {
	q := r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).
		Where("platform_order_id <> ''").
		Order("id asc")
	if onlyMissing {
		q = q.Where("(fill_qty = 0 OR status IN ? OR fee = 0)", []string{"NO_FILL", "PARTIAL", "ERROR", ""})
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var list []*entity.ArbBinaryOrder
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ArbBinaryOrderImpl) UpdateFill(ctx context.Context, id uint, fillQty, fee float64, status, errMsg string) error {
	up := map[string]interface{}{
		"fill_qty": decimal.NewFromFloat(fillQty).Round(8),
		"fee":      decimal.NewFromFloat(fee).Round(8),
		"status":   strings.TrimSpace(status),
	}
	if msg := strings.TrimSpace(errMsg); msg != "" {
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		up["err_msg"] = msg
	}
	return r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).Where("id = ?", id).Updates(up).Error
}