package synthetic

import (
	"fmt"
	"math"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketmap"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
)

const defaultTick = 0.01

func levelsJSON(bids, asks [][2]float64) (bidsS, asksS string) {
	return marshalLevelsJSON(bids), marshalLevelsJSON(asks)
}

func marshalLevelsJSON(levels [][2]float64) string {
	if len(levels) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, lv := range levels {
		if i > 0 {
			b.WriteByte(',')
		}
		p := roundTick(lv[0], defaultTick)
		s := roundSize(lv[1])
		fmt.Fprintf(&b, "[%.2f,%.2f]", p, s)
	}
	b.WriteByte(']')
	return b.String()
}

// fakeBookAtSecond 生成简易 L2；各平台在同一秒使用相同时间戳，价格略错开便于触发套利检测。
func fakeBookAtSecond(sec int, platform string, outcomeIdx int) (bidsJSON, asksJSON string) {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	base := 0.46 + 0.03*math.Sin(float64(sec)/80.0) + 0.01*float64(outcomeIdx%3)
	platSkew := 0.0
	switch platform {
	case string(cryptoPredictionEnum.PlatformPolymarket):
		platSkew = 0.0
	case string(cryptoPredictionEnum.PlatformPredict):
		platSkew = 0.012
	case string(cryptoPredictionEnum.PlatformKalshi):
		platSkew = -0.008
	}
	bid := roundTick(base+platSkew, defaultTick)
	spread := roundTick(0.02+0.003*math.Cos(float64(sec)/40.0), defaultTick)
	if spread < defaultTick {
		spread = defaultTick
	}
	ask := roundTick(bid+spread, defaultTick)
	if ask <= bid {
		ask = roundTick(bid+defaultTick, defaultTick)
	}
	size := roundSize(800 + 200*math.Mod(float64(sec*7+outcomeIdx), 50))
	bids := [][2]float64{{bid, size}, {roundTick(bid-defaultTick, defaultTick), roundSize(size * 0.4)}}
	asks := [][2]float64{{ask, roundSize(size * 0.9)}, {roundTick(ask+defaultTick, defaultTick), roundSize(size * 0.35)}}
	return levelsJSON(bids, asks)
}

func roundTick(p, tick float64) float64 {
	if tick <= 0 {
		tick = defaultTick
	}
	scale := 1 / tick
	units := int64(math.Round(p * scale))
	return boundPrice(float64(units) / scale)
}

func roundSize(s float64) float64 {
	if s < 0 {
		return 0
	}
	return math.Round(s*100) / 100
}

func boundPrice(p float64) float64 {
	if p < defaultTick {
		return defaultTick
	}
	if p > 0.99 {
		return 0.99
	}
	return p
}

// 在 1000 秒合成序列里嵌入少量秒级窗口，便于回测 yesno（秒 200–206）与 onextwo（秒 500–506）。
const (
	binaryArbStartSec = 200
	binaryArbSecLen   = 7
	oneX2ArbStartSec  = 500
	oneX2ArbSecLen    = 7
)

type arbPlan struct {
	binaryGroup string
	drawKey     string
	teamAKey    string
	teamBKey    string
}

func buildArbPlan(outcomes []persistententity.MarketOutcome) arbPlan {
	type platSet map[string]struct{}
	byGroup := map[string]platSet{}
	moneyline := map[string]bool{}
	for i := range outcomes {
		oc := &outcomes[i]
		gk := marketmap.OutcomeGroupKey(oc)
		if gk == "" {
			continue
		}
		if byGroup[gk] == nil {
			byGroup[gk] = platSet{}
		}
		byGroup[gk][strings.ToUpper(strings.TrimSpace(oc.Platform))] = struct{}{}
		if strings.EqualFold(strings.TrimSpace(oc.OutcomeType), "moneyline") {
			moneyline[gk] = true
		}
	}
	var plan arbPlan
	for gk, plats := range byGroup {
		if len(plats) < 2 {
			continue
		}
		if plan.binaryGroup == "" && !isDrawGroupKey(gk) {
			plan.binaryGroup = gk
		}
	}
	var draw string
	var teams []string
	for gk := range byGroup {
		if !moneyline[gk] {
			continue
		}
		if isDrawGroupKey(gk) {
			draw = gk
			continue
		}
		teams = append(teams, gk)
	}
	if draw != "" && len(teams) >= 2 {
		plan.drawKey = draw
		plan.teamAKey = teams[0]
		plan.teamBKey = teams[1]
		if strings.Compare(plan.teamAKey, plan.teamBKey) > 0 {
			plan.teamAKey, plan.teamBKey = plan.teamBKey, plan.teamAKey
		}
	}
	return plan
}

func isDrawGroupKey(gk string) bool {
	parts := strings.Split(gk, "|")
	if len(parts) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(parts[len(parts)-1]), "draw")
}

func inSecWindow(sec, start, span int) bool {
	return sec >= start && sec < start+span
}

func oneX2Leg(plan arbPlan, groupKey string) string {
	switch groupKey {
	case plan.drawKey:
		return "draw"
	case plan.teamAKey:
		return "teamA"
	case plan.teamBKey:
		return "teamB"
	default:
		return ""
	}
}

func bookForOutcome(sec int, oc persistententity.MarketOutcome, outcomeIdx int, plan arbPlan) (bidsJSON, asksJSON string) {
	gk := marketmap.OutcomeGroupKey(&oc)
	platform := strings.ToUpper(strings.TrimSpace(oc.Platform))
	if plan.binaryGroup != "" && gk == plan.binaryGroup && inSecWindow(sec, binaryArbStartSec, binaryArbSecLen) {
		return binaryArbLevelsJSON(platform)
	}
	if leg := oneX2Leg(plan, gk); leg != "" && inSecWindow(sec, oneX2ArbStartSec, oneX2ArbSecLen) {
		return oneX2ArbLevelsJSON(platform, leg)
	}
	return fakeBookAtSecond(sec, platform, outcomeIdx)
}

func binaryArbLevelsJSON(platform string) (string, string) {
	var bids, asks [][2]float64
	switch platform {
	case string(cryptoPredictionEnum.PlatformPolymarket):
		bids = [][2]float64{{0.36, 500}, {0.35, 300}}
		asks = [][2]float64{{0.38, 800}, {0.39, 400}}
	case string(cryptoPredictionEnum.PlatformPredict):
		bids = [][2]float64{{0.40, 500}, {0.39, 200}}
		asks = [][2]float64{{0.48, 500}, {0.49, 200}}
	case string(cryptoPredictionEnum.PlatformKalshi):
		bids = [][2]float64{{0.56, 800}, {0.55, 300}}
		asks = [][2]float64{{0.30, 500}, {0.31, 200}}
	default:
		bids, asks = [][2]float64{{0.45, 400}}, [][2]float64{{0.47, 400}}
	}
	return levelsJSON(bids, asks)
}

func oneX2ArbLevelsJSON(platform, leg string) (string, string) {
	var yesAsk float64
	switch leg {
	case "draw":
		yesAsk = 0.32
	case "teamA":
		yesAsk = 0.30
	case "teamB":
		yesAsk = 0.28
	default:
		yesAsk = 0.33
	}
	if platform == string(cryptoPredictionEnum.PlatformPolymarket) {
		bid := roundTick(yesAsk-defaultTick, defaultTick)
		asks := [][2]float64{{yesAsk, 600}, {roundTick(yesAsk+defaultTick, defaultTick), 300}}
		bids := [][2]float64{{bid, 500}, {roundTick(bid-defaultTick, defaultTick), 200}}
		return levelsJSON(bids, asks)
	}
	return fakeBookAtSecond(oneX2ArbStartSec, platform, 0)
}