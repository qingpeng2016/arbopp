package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type HistoryDataOrderBookRepo interface {
	FindOne(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy string) (entity.HistoryDataOrderBook, error)
	List(ctx context.Context, tx *gorm.DB, where map[string]interface{}, orderBy, groupBy string) (total int64, list []entity.HistoryDataOrderBook, err error)
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.HistoryDataOrderBook) error
	BatchUpsert(ctx context.Context, tx *gorm.DB, list []*entity.HistoryDataOrderBook) error
	// PatchUTCDateForPath 修正 utc_date（当前库 INSERT 该列会被写成 CURDATE，UPDATE 可写对日历日）。
	PatchUTCDateForPath(ctx context.Context, tx *gorm.DB, outcomeID uint, localPath, recordKind, utcDate string) error
	DeleteWhere(ctx context.Context, tx *gorm.DB, where map[string]interface{}) error
}