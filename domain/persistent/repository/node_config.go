package repository

import (
	"context"

	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

type NodeConfigRepo interface {
	FindByNodeCode(ctx context.Context, tx *gorm.DB, nodeCode string) (entity.NodeConfig, error)
}

type NodeRouteRepo interface {
	ListBySrcNode(ctx context.Context, tx *gorm.DB, srcNode string) ([]entity.NodeRoute, error)
}