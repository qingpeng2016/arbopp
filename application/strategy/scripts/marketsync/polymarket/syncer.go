package polymarket

import (
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/batch"
	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/syncopt"
	httprepo "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/repository"
)

// Syncer Polymarket 场次/盘口同步。
type Syncer struct {
	API               httprepo.PolymarketRepo
	W                 batch.Writer
	OutcomeTypes      syncopt.OutcomeTypeSet
	LeagueConcurrency int
}

func NewSyncer(api httprepo.PolymarketRepo, w batch.Writer) *Syncer {
	return &Syncer{API: api, W: w}
}
