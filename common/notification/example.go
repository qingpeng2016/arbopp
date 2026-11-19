package notification

import (
	"context"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/logger"
	"go.uber.org/zap"
)

// ExampleUsage 示例用法
func ExampleUsage() {
	// 1. 创建通知管理器（自动从配置中读取）
	notificationManager := NewNotificationManager()

	ctx := context.Background()

	// 3. 发送不同类型的通知
	// 发送错误报警
	err := notificationManager.SendErrorAlert(ctx, "数据库连接失败", map[string]string{
		"错误类型": "数据库连接",
		"错误详情": "MySQL连接超时",
		"影响范围": "所有服务",
	})
	if err != nil {
		logger.ErrorZ(ctx, "Failed to send error alert", zap.Error(err))
	}

	// 发送信息通知
	err = notificationManager.SendInfoAlert(ctx, "服务启动成功", []FieldPair{
		{Key: "服务名称", Value: "fgmm"},
		{Key: "启动时间", Value: "2025-01-04 10:00:00"},
		{Key: "运行环境", Value: "test"},
	})
	if err != nil {
		logger.ErrorZ(ctx, "Failed to send info alert", zap.Error(err))
	}
}

// SendPolymarketAlert 发送Polymarket相关报警的便捷方法
func SendPolymarketAlert(ctx context.Context, alertType, message string, fields []FieldPair) error {
	notificationManager := NewNotificationManager()

	switch alertType {
	case "error":
		// 需要将 []FieldPair 转换为 map[string]string
		fieldsMap := make(map[string]string)
		for _, f := range fields {
			fieldsMap[f.Key] = f.Value
		}
		return notificationManager.SendErrorAlert(ctx, message, fieldsMap)
	case "info":
		return notificationManager.SendInfoAlert(ctx, message, fields)
	default:
		return notificationManager.SendInfoAlert(ctx, message, fields)
	}
}

// SendHedgeAlert 发送对冲相关报警
func SendHedgeAlert(ctx context.Context, alertType, message string, marketID string, additionalFields []FieldPair) error {
	fields := []FieldPair{
		{Key: "市场ID", Value: marketID},
		{Key: "模块", Value: "对冲"},
	}

	// 合并额外字段
	fields = append(fields, additionalFields...)

	return SendPolymarketAlert(ctx, alertType, message, fields)
}

// SendOrderAlert 发送订单相关报警
func SendOrderAlert(ctx context.Context, alertType, message string, orderID string, additionalFields []FieldPair) error {
	fields := []FieldPair{
		{Key: "订单ID", Value: orderID},
		{Key: "模块", Value: "订单"},
	}

	// 合并额外字段
	fields = append(fields, additionalFields...)

	return SendPolymarketAlert(ctx, alertType, message, fields)
}
