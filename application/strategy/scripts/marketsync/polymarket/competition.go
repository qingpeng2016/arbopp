package polymarket

import (
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
	httpentity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

func polymarketCompetitionPtrs(name, nameEn string) (comp, compEn *string) {
	if n := strings.TrimSpace(name); n != "" {
		comp = common.StrPtr128(n)
	}
	if n := strings.TrimSpace(nameEn); n != "" {
		compEn = common.StrPtr128(n)
	}
	return comp, compEn
}

func polymarketCompetitionName(m *httpentity.Market, bySport map[string]string) (name, sportID string) {
	if m == nil || len(bySport) == 0 {
		return "", ""
	}
	slug := strings.ToLower(strings.TrimSpace(m.Slug))
	for sid, n := range bySport {
		if slug == sid || strings.HasPrefix(slug, sid+"-") {
			return n, sid
		}
	}
	for _, e := range m.Events {
		for _, s := range e.Series {
			ss := strings.ToLower(strings.TrimSpace(s.Slug))
			for sid, n := range bySport {
				if ss == sid || strings.HasPrefix(ss, sid+"-") || strings.Contains(ss, sid) {
					return n, sid
				}
			}
		}
	}
	return "", ""
}
