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
		// 有平台单号，或 ERROR 的 err_msg 里藏了 orderID（Poly FAK 400）
		Where("(platform_order_id <> '' OR (platform_order_id = '' AND err_msg LIKE ?))", "%orderID%").
		Order("id asc")
	if onlyMissing {
		// 未成交/部分/错误/挂起需回查；FILLED 且 trade_qty>0 仅在 fee 仍为 0 时补手续费。
		q = q.Where(`(
			trade_qty = 0
			OR fee = 0
			OR UPPER(TRIM(status)) IN ?
		)`, []string{"NO_FILL", "PARTIAL", "ERROR", "PENDING", "OPEN", "LIVE", ""})
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

func (r *ArbBinaryOrderImpl) UpdateFill(ctx context.Context, id uint, tradeQty, fee, tradePrice float64, status, errMsg string) error {
	tq := decimal.NewFromFloat(tradeQty).Round(8)
	tp := decimal.NewFromFloat(tradePrice).Round(8)
	var ta decimal.Decimal
	if tradeQty > 0 && tradePrice > 0 {
		ta = decimal.NewFromFloat(tradePrice * tradeQty).Round(8)
	}
	up := map[string]interface{}{
		"trade_qty":    tq,
		"trade_price":  tp,
		"trade_amount": ta,
		"fee":          decimal.NewFromFloat(fee).Round(8),
		"status":       strings.TrimSpace(status),
	}
	if msg := strings.TrimSpace(errMsg); msg != "" {
		if len(msg) > 1000 {
			msg = msg[:1000]
		}
		up["err_msg"] = msg
	}
	return r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).Where("id = ?", id).Updates(up).Error
}

func (r *ArbBinaryOrderImpl) ListByArbOppID(ctx context.Context, arbOppID uint) ([]*entity.ArbBinaryOrder, error) {
	if arbOppID == 0 {
		return nil, nil
	}
	var list []*entity.ArbBinaryOrder
	err := r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).
		Where("arb_opp_id = ?", arbOppID).
		Order("side_no asc, id asc").
		Find(&list).Error
	return list, err
}

// 终态：FAK/IOC 路径下成交完成或明确失败；其余（含空串）视为未终态。
var orderTerminalStatuses = []string{"FILLED", "PARTIAL", "NO_FILL", "ERROR"}

func (r *ArbBinaryOrderImpl) HasNonTerminalByMapKey(ctx context.Context, mapKey string) (bool, error) {
	mapKey = strings.TrimSpace(mapKey)
	if mapKey == "" {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).
		Where("map_key = ?", mapKey).
		Where("UPPER(TRIM(status)) NOT IN ?", orderTerminalStatuses).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ArbBinaryOrderImpl) ListNonTerminalMapKeys(ctx context.Context) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).Model(&entity.ArbBinaryOrder{}).
		Select("DISTINCT map_key").
		Where("map_key <> ''").
		Where("UPPER(TRIM(status)) NOT IN ?", orderTerminalStatuses).
		Pluck("map_key", &keys).Error
	if err != nil {
		return nil, err
	}
	return keys, nil
}