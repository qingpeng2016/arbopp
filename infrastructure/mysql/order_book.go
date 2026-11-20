package mysql

import (
	"context"
	"fmt"
	"sort"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderBookImpl struct {
	db *gorm.DB
}

func NewOrderBookImpl(db *gorm.DB) repository.OrderBookRepo {
	return &OrderBookImpl{db: db}
}

func (r *OrderBookImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.OrderBook, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.OrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.OrderBook
	err := db.First(&e).Error
	return e, err
}

func (r *OrderBookImpl) ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.OrderBook, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.OrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	for key, values := range or {
		db = db.Or(key, values...)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	db = db.Count(&total)
	if size != 0 {
		db = db.Offset(size * (page - 1)).Limit(size)
	}
	err = db.Find(&list).Error
	return total, list, err
}

func (r *OrderBookImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.OrderBook, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.OrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	err = db.Count(&total).Find(&list).Error
	return total, list, err
}

func (r *OrderBookImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.OrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}

func (r *OrderBookImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.OrderBook) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	if err := tx.WithContext(ctx).Model(entity.OrderBook{}).Create(e).Error; err != nil {
		return 0, fmt.Errorf("create market_order_book: %w", err)
	}
	return e.ID, nil
}

func (r *OrderBookImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.OrderBook) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.OrderBook{}).CreateInBatches(list, 100).Error
}

// BatchUpsert 遗留：历史快照请用 BatchCreate（表无 token 唯一键）。
func (r *OrderBookImpl) BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.OrderBook) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	sortOrderBooksForUpsert(list)
	return withDeadlockRetry(func() error {
		return tx.WithContext(ctx).Model(entity.OrderBook{}).Clauses(clause.OnConflict{
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

func sortOrderBooksForUpsert(list []*entity.OrderBook) {
	sort.Slice(list, func(i, j int) bool {
		return orderBookUpsertLockKey(list[i]) < orderBookUpsertLockKey(list[j])
	})
}

func orderBookUpsertLockKey(b *entity.OrderBook) string {
	if b == nil {
		return ""
	}
	return b.Platform + "\x00" + b.PlatformMarketOutcomeID + "\x00" + b.TokenID
}
