package mysql

import (
	"context"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
	"gorm.io/gorm"
)

type NodeConfigImpl struct {
	db *gorm.DB
}

func NewNodeConfigImpl(db *gorm.DB) repository.NodeConfigRepo {
	return &NodeConfigImpl{db: db}
}

func (r *NodeConfigImpl) FindByNodeCode(ctx context.Context, tx *gorm.DB, nodeCode string) (entity.NodeConfig, error) {
	if tx == nil {
		tx = r.db
	}
	var e entity.NodeConfig
	err := tx.WithContext(ctx).Model(entity.NodeConfig{}).
		Where("node_code = ? AND is_enabled = 1", strings.TrimSpace(nodeCode)).
		First(&e).Error
	return e, err
}

type NodeRouteImpl struct {
	db *gorm.DB
}

func NewNodeRouteImpl(db *gorm.DB) repository.NodeRouteRepo {
	return &NodeRouteImpl{db: db}
}

func (r *NodeRouteImpl) ListBySrcNode(ctx context.Context, tx *gorm.DB, srcNode string) ([]entity.NodeRoute, error) {
	if tx == nil {
		tx = r.db
	}
	var list []entity.NodeRoute
	err := tx.WithContext(ctx).Model(entity.NodeRoute{}).
		Where("src_node = ? AND is_enabled = 1", strings.TrimSpace(srcNode)).
		Order("id asc").
		Find(&list).Error
	return list, err
}