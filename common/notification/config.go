package notification

import (
	"context"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"go.uber.org/zap"
)

// NotificationManager 通知管理器
type NotificationManager struct {
	alertClient  *LarkClient // 异常报警机器人
	notifyClient *LarkClient // 通知消息机器人
}

// NewNotificationManager 创建通知管理器
func NewNotificationManager() *NotificationManager {
	larkConf := conf.GetLarkNotificationConf()
	if larkConf == nil {
		return &NotificationManager{alertClient: nil, notifyClient: nil}
	}

	var alertClient, notifyClient *LarkClient

	// 创建异常报警机器人客户端
	if larkConf.AlertWebhookURL != "" {
		alertClient = NewLarkClient(larkConf.AlertWebhookURL)
	}

	// 创建通知消息机器人客户端
	if larkConf.NotifyWebhookURL != "" {
		notifyClient = NewLarkClient(larkConf.NotifyWebhookURL)
	}

	return &NotificationManager{
		alertClient:  alertClient,
		notifyClient: notifyClient,
	}
}

// FieldPair 字段对
type FieldPair struct {
	Key   string
	Value string
}

// SendErrorAlert 发送错误报警（使用异常报警机器人）
func (nm *NotificationManager) SendErrorAlert(ctx context.Context, errorMsg string, fields map[string]string) error {
	if nm.alertClient == nil {
		logger.InfoZ(ctx, "alert-notification-disabled", zap.String("type", "error"))
		return nil
	}

	larkConf := conf.GetLarkNotificationConf()
	if larkConf == nil {
		return nil
	}
	return nm.alertClient.SendErrorAlert(ctx, larkConf.ServiceName, errorMsg, fields)
}

// SendErrorAlertWithOrder 发送错误报警（保持字段顺序）
func (nm *NotificationManager) SendErrorAlertWithOrder(ctx context.Context, errorMsg string, fields []FieldPair) error {
	if nm.alertClient == nil {
		logger.InfoZ(ctx, "alert-notification-disabled", zap.String("type", "error"))
		return nil
	}

	larkConf := conf.GetLarkNotificationConf()
	if larkConf == nil {
		return nil
	}
	return nm.alertClient.SendErrorAlertWithOrder(ctx, larkConf.ServiceName, errorMsg, fields)
}

// SendInfoAlert 发送信息通知（使用通知消息机器人）
func (nm *NotificationManager) SendInfoAlert(ctx context.Context, infoMsg string, fields []FieldPair) error {
	if nm.notifyClient == nil {
		logger.InfoZ(ctx, "notify-notification-disabled", zap.String("type", "info"))
		return nil
	}

	larkConf := conf.GetLarkNotificationConf()
	if larkConf == nil {
		return nil
	}
	return nm.notifyClient.SendInfoAlert(ctx, larkConf.ServiceName, infoMsg, fields)
}

// IsAlertEnabled 检查异常报警是否启用
func (nm *NotificationManager) IsAlertEnabled() bool {
	return nm.alertClient != nil
}

// IsNotifyEnabled 检查通知消息是否启用
func (nm *NotificationManager) IsNotifyEnabled() bool {
	return nm.notifyClient != nil
}
