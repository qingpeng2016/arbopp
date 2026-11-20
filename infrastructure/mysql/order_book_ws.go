package mysql

import (
	"context"
	"sort"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderBookWSImpl struct {
	db *gorm.DB
}

func NewOrderBookWSImpl(db *gorm.DB) repository.OrderBookWSRepo {
	return &OrderBookWSImpl{db: db}
}

func (r *OrderBookWSImpl) BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.OrderBookWS) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	sort.Slice(list, func(i, j int) bool {
		return orderBookWSUpsertLockKey(list[i]) < orderBookWSUpsertLockKey(list[j])
	})
	return withDeadlockRetry(func() error {
		return tx.WithContext(ctx).Model(entity.OrderBookWS{}).Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "platform"},
				{Name: "platform_market_outcome_id"},
				{Name: "token_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"market_row_id", "map_key", "category", "sub_category", "competition", "competition_en",
				"platform_market_id",
				"market_name", "outcome_name", "token_name", "market_url", "market_address", "selection_id",
				"order_book_asks", "order_book_bids", "snapshot_second", "updated_at",
			}),
		}).CreateInBatches(list, 100).Error
	})
}

func orderBookWSUpsertLockKey(b *entity.OrderBookWS) string {
	if b == nil {
		return ""
	}
	return b.Platform + "\x00" + b.PlatformMarketOutcomeID + "\x00" + b.TokenID
}