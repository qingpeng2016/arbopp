package entity

import "time"

type NodeConfig struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	NodeCode       string    `gorm:"column:node_code"`
	LocalPlatform  string    `gorm:"column:local_platform"`
	OfficialWS     string    `gorm:"column:official_ws"`
	NodeName       string    `gorm:"column:node_name"`
	Region         string    `gorm:"column:region"`
	VpcCidr        string    `gorm:"column:vpc_cidr"`
	RelayListen    string    `gorm:"column:relay_listen"`
	PeerIP         string    `gorm:"column:peer_ip"`
	RunConf        string    `gorm:"column:run_conf"`
	Description    string    `gorm:"column:description"`
	IsEnabled      int       `gorm:"column:is_enabled"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (NodeConfig) TableName() string { return "node_config" }