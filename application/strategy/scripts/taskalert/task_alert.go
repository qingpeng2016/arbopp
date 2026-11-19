package taskalert

import (
	"context"
)

// TaskAlert 产品「内部告警」：同币对并发/名义、单任务 IOC 次数（只告警，不拦发单）。
type TaskAlert struct {
}

// NewTaskAlert 注入依赖11。
func NewTaskAlert() *TaskAlert {
	return &TaskAlert{}
}

// Run 扫描全部 liquidating 任务，按 symbol / 任务维度检查策略阈值。
func (j *TaskAlert) Run(ctx context.Context) {

}
