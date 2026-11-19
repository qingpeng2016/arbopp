package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StatKVImpl struct {
	db *gorm.DB
}

func NewStatKVImpl(db *gorm.DB) repository.StatKVRepo {
	return &StatKVImpl{db: db}
}

func (r *StatKVImpl) UpsertBatch(ctx context.Context, tx *gorm.DB, list []*entity.StatKV) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.StatKV{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "group_name"}, {Name: "item_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"item_value", "value_type", "description", "sort_no", "is_enabled", "updated_at",
		}),
	}).CreateInBatches(list, 100).Error
}

func (r *StatKVImpl) DeleteByGroup(ctx context.Context, tx *gorm.DB, groupName string) error {
	if tx == nil {
		tx = r.db
	}
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return fmt.Errorf("delete stat_kv: empty group_name")
	}
	return tx.WithContext(ctx).Where("group_name = ?", groupName).Delete(&entity.StatKV{}).Error
}

func (r *StatKVImpl) BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.StatKV) error {
	if tx == nil {
		tx = r.db
	}
	if len(list) == 0 {
		return nil
	}
	return tx.WithContext(ctx).Model(entity.StatKV{}).CreateInBatches(list, 100).Error
}