package synthetic

import (
	"fmt"
	"math"
	"strings"

	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
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