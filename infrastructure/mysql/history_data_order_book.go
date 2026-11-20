package mysql

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HistoryDataOrderBookImpl struct {
	db *gorm.DB
}

func NewHistoryDataOrderBookImpl(db *gorm.DB) repository.HistoryDataOrderBookRepo {
	return &HistoryDataOrderBookImpl{db: db}
}

func (r *HistoryDataOrderBookImpl) List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.HistoryDataOrderBook, err error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.HistoryDataOrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if groupBy != "" {
		db = db.Group(groupBy)
	}
	if err = db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	err = db.Find(&list).Error
	return total, list, err
}

func (r *HistoryDataOrderBookImpl) FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.HistoryDataOrderBook, error) {
	if tx == nil {
		tx = r.db
	}
	db := tx.WithContext(ctx).Model(entity.HistoryDataOrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	if orderBy != "" {
		db = db.Order(orderBy)
	}
	var e entity.HistoryDataOrderBook
	err := db.First(&e).Error
	return e, err
}

func (r *HistoryDataOrderBookImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.HistoryDataOrderBook) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.HistoryDataOrderBook{}).CreateInBatches(list, 100).Error
}

func (r *HistoryDataOrderBookImpl) BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.HistoryDataOrderBook) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.HistoryDataOrderBook{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "data_platform_id"},
			{Name: "market_outcome_id"},
			{Name: "utc_date"},
			{Name: "data_type"},
			{Name: "record_kind"},
			{Name: "snapshot_at_ms"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"map_key", "market_row_id", "venue_platform", "vendor_exchange", "vendor_symbol",
			"source", "order_book_bids", "order_book_asks", "local_file_path", "file_bytes", "file_sha256",
			"ingest_status", "error_text", "updated_at",
		}),
	}).CreateInBatches(list, 100).Error
}

func (r *HistoryDataOrderBookImpl) PatchUTCDateForPath(ctx context.Context, tx *gorm.DB, outcomeID uint, localPath, recordKind, utcDate string) error {
	if tx == nil {
		tx = r.db
	}
	if outcomeID == 0 || localPath == "" || recordKind == "" || utcDate == "" {
		return nil
	}
	return tx.WithContext(ctx).Exec(
		"UPDATE `data_history_order_book` SET `utc_date`=? WHERE `market_outcome_id`=? AND `local_file_path`=? AND `record_kind`=?",
		utcDate, outcomeID, localPath, recordKind,
	).Error
}

func (r *HistoryDataOrderBookImpl) DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error {
	if tx == nil {
		tx = r.db
	}
	if len(where) == 0 {
		return nil
	}
	db := tx.WithContext(ctx).Model(entity.HistoryDataOrderBook{})
	for condition, value := range where {
		db = db.Where(condition, value)
	}
	return db.Delete(&entity.HistoryDataOrderBook{}).Error
}