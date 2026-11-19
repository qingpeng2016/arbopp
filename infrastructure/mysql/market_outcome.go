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

type MarketOutcomeImpl struct {
	db *gorm.DB
}

func NewMarketOutcomeImpl(db *gorm.DB) repository.MarketOutcomeRepo {
	return &MarketOutcomeImpl{db: db}
}

func (r *MarketOutcomeImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.MarketOutcome, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.MarketOutcome{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.MarketOutcome
	err := db.First(&e).Error
	return e, err
}

func (r *MarketOutcomeImpl) ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.MarketOutcome, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.MarketOutcome{})
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

func (r *MarketOutcomeImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.MarketOutcome, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.MarketOutcome{})
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

func (r *MarketOutcomeImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.MarketOutcome{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}

func (r *MarketOutcomeImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.MarketOutcome) (uint, error) {
	if tx == nil {
		tx = r.db
	}
	if err := tx.WithContext(ctx).Model(entity.MarketOutcome{}).Create(e).Error; err != nil {
		return 0, fmt.Errorf("create market_outcome: %w", err)
	}
	return e.ID, nil
}

func (r *MarketOutcomeImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.MarketOutcome) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.MarketOutcome{}).CreateInBatches(list, 100).Error
}

func (r *MarketOutcomeImpl) BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.MarketOutcome) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	sortMarketOutcomesForUpsert(list)
	return withDeadlockRetry(func() error {
		return tx.WithContext(ctx).Model(entity.MarketOutcome{}).Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "platform"},
				{Name: "platform_market_outcome_id"},
				{Name: "category"},
				{Name: "sub_category"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"market_row_id", "platform_market_id", "name", "desc", "market_address", "market_url",
				"yes_token_id", "no_token_id", "selection_id",
				"market_start_time", "market_end_time", "outcome_type",
				"volume_total", "volume24hr", "slug", "competition", "config",
			}),
		}).CreateInBatches(list, 100).Error
	})
}

func sortMarketOutcomesForUpsert(list []*entity.MarketOutcome) {
	sort.Slice(list, func(i, j int) bool {
		return marketOutcomeUpsertLockKey(list[i]) < marketOutcomeUpsertLockKey(list[j])
	})
}

func marketOutcomeUpsertLockKey(m *entity.MarketOutcome) string {
	if m == nil {
		return ""
	}
	cat, sub := "", ""
	if m.Category != nil {
		cat = *m.Category
	}
	if m.SubCategory != nil {
		sub = *m.SubCategory
	}
	return m.Platform + "\x00" + m.PlatformMarketOutcomeID + "\x00" + cat + "\x00" + sub
}
