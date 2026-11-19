package predict

import (
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

// Syncer Predict.fun 场次/盘口同步。
type Syncer struct {
	API               httprepo.PredictRepo
	W                 batch.Writer
	LeagueConcurrency int
}

func NewSyncer(api httprepo.PredictRepo, w batch.Writer) *Syncer {
	return &Syncer{API: api, W: w}
}