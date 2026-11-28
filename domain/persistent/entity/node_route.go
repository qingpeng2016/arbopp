package entity

import "time"

type NodeRoute struct {
	ID          uint           `gorm:"primaryKey;column:id"`
	SrcNode     string         `gorm:"column:src_node"`
	DstNode     string         `gorm:"column:dst_node"`
	RouteJSON   []byte `gorm:"column:route_json"`
	IsEnabled   int            `gorm:"column:is_enabled"`
	Description string         `gorm:"column:description"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
}

func (NodeRoute) TableName() string { return "node_route" }