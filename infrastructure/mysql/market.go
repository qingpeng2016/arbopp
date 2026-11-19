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

type MarketImpl struct {
	db *gorm.DB
}

func NewMarketImpl(db *gorm.DB) repository.MarketRepo {
	return &MarketImpl{db: db}
}

func (r *MarketImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.Market, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Market{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.Market
	err := db.First(&e).Error
	return e, err
}

func (r *MarketImpl) ListByPage(ctx context.Context, tx *gorm.DB, where map[string]interface{}, or map[string][]interface{}, page, size int, orderBy, groupBy string) (total int64, list []entity.Market, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Market{})
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

func (r *MarketImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.Market, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Market{})
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

func (r *MarketImpl) Update(ctx context.Context, tx *gorm.DB, where map[string]interface{}, updateData map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.Market{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Updates(updateData).Error
}

func (r *MarketImpl) Create(ctx context.Context, tx *gorm.DB, e *entity.Market) (uint, error) {
	if tx == nil {
		tx = r.db
	}
	if err := tx.WithContext(ctx).Model(entity.Market{}).Create(e).Error; err != nil {
		return 0, fmt.Errorf("create market: %w", err)
	}
	return e.ID, nil
}

func (r *MarketImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.Market) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.Market{}).CreateInBatches(list, 100).Error
}

// BatchUpsert 按 uk_platform_market_sub；同步任务不覆盖 map_key、map_verified
func (r *MarketImpl) BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.Market) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	sortMarketsForUpsert(list)
	return withDeadlockRetry(func() error {
		return tx.WithContext(ctx).Model(entity.Market{}).Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "platform"},
				{Name: "platform_market_id"},
				{Name: "category"},
				{Name: "sub_category"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "desc", "market_url",
				"event_start_time", "event_end_time",
				"status", "volume_total", "volume24hr", "slug", "league", "competition", "competition_en", "sport_cn", "config",
			}),
		}).CreateInBatches(list, 100).Error
	})
}

func sortMarketsForUpsert(list []*entity.Market) {
	sort.Slice(list, func(i, j int) bool {
		return marketUpsertLockKey(list[i]) < marketUpsertLockKey(list[j])
	})
}

func marketUpsertLockKey(m *entity.Market) string {
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
	return m.Platform + "\x00" + m.PlatformMarketID + "\x00" + cat + "\x00" + sub
}
