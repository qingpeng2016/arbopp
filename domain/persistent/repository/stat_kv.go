package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type StatKVRepo interface {
	UpsertBatch(ctx context.Context, tx *gorm.DB, list []*entity.StatKV) error
	DeleteByGroup(ctx context.Context, tx *gorm.DB, groupName string) error
	DeleteByGroupItems(ctx context.Context, tx *gorm.DB, groupName string, itemKeys []string) error
	BatchCreate(ctx context.Context, tx *gorm.DB, list []*entity.StatKV) error
}