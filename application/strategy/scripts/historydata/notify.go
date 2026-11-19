package historydata

import (
	"context"
	"strconv"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/common/notification"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

// NotifyTaskFinished 单条 data_task 拉取结束（completed / failed）时发 Lark 信息通知。
func NotifyTaskFinished(ctx context.Context, task *persistententity.DataTask, filesOK, failed int, errText string) {
	if task == nil {
		return
	}
	mapKey := strings.TrimSpace(task.MapKey)
	if mapKey == "" {
		return
	}
	title := "history_data_sync 任务完成"
	if task.Status == persistententity.DataTaskStatusFailed {
		title = "history_data_sync 任务失败"
	}
	fields := []notification.FieldPair{
		{Key: "task_id", Value: strconv.FormatUint(uint64(task.ID), 10)},
		{Key: "map_key", Value: mapKey},
		{Key: "status", Value: task.Status},
		{Key: "files_ok", Value: strconv.Itoa(filesOK)},
		{Key: "download_failed", Value: strconv.Itoa(failed)},
	}
	if strings.TrimSpace(errText) != "" {
		fields = append(fields, notification.FieldPair{Key: "说明", Value: errText})
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, title, fields)
}

// NotifySyntheticTaskFinished 秒级伪造任务结束。
func NotifySyntheticTaskFinished(ctx context.Context, task *persistententity.DataTask, rows int64, totalSeconds, outcomes int) {
	if task == nil {
		return
	}
	title := "synthetic_history_sync 任务完成"
	if task.Status == persistententity.DataTaskStatusFailed {
		title = "synthetic_history_sync 任务失败"
	}
	_ = notification.GetGlobalNotificationManager().SendInfoAlert(ctx, title, []notification.FieldPair{
		{Key: "task_id", Value: strconv.FormatUint(uint64(task.ID), 10)},
		{Key: "map_key", Value: strings.TrimSpace(task.MapKey)},
		{Key: "status", Value: task.Status},
		{Key: "rows", Value: strconv.FormatInt(rows, 10)},
		{Key: "seconds", Value: strconv.Itoa(totalSeconds)},
		{Key: "outcomes", Value: strconv.Itoa(outcomes)},
	})
}