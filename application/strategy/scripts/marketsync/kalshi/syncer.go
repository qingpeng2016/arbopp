package kalshi

import (
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

// Syncer Kalshi 场次/盘口同步。
type Syncer struct {
	API               httprepo.KalshiRepo
	W                 batch.Writer
	LeagueConcurrency int
}

func NewSyncer(api httprepo.KalshiRepo, w batch.Writer) *Syncer {
	return &Syncer{API: api, W: w}
}
