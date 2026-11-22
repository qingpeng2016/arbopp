package main

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
)

type level struct {
	price float64
	size  float64
}

func parseLevels(js string) []level {
	var raw [][]string
	if json.Unmarshal([]byte(js), &raw) != nil {
		return nil
	}
	out := make([]level, 0, len(raw))
	for _, r := range raw {
		if len(r) < 2 {
			continue
		}
		p, _ := strconv.ParseFloat(r[0], 64)
		s, _ := strconv.ParseFloat(r[1], 64)
		if validPrice(p) && s > 0 {
			out = append(out, level{p, s})
		}
	}
	return out
}

func bestAsk(levels []level) (float64, float64) {
	if len(levels) == 0 {
		return 0, 0
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i].price < levels[j].price })
	return levels[0].price, levels[0].size
}

func bestBid(levels []level) (float64, float64) {
	if len(levels) == 0 {
		return 0, 0
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i].price > levels[j].price })
	return levels[0].price, levels[0].size
}

type legQuote struct {
	YesPx, YesSz float64
	NoPx, NoSz   float64
}

func (lq *legQuote) ready() bool {
	return validPrice(lq.YesPx) && validPrice(lq.NoPx) && lq.YesSz > 0 && lq.NoSz > 0
}

// useWSRowForTopBook 与 yesno.TopFromStore 一致：Poly 按 token；Predict/SX/Kalshi 只吃 YES 簿那一行。
func useWSRowForTopBook(plat, tok string, mo outcomeRec) bool {
	plat = strings.ToUpper(strings.TrimSpace(plat))
	tok = strings.TrimSpace(tok)
	yesTok := strings.TrimSpace(mo.YesTokenID)
	switch plat {
	case platPoly:
		return true
	default:
		if tok == "" {
			return true
		}
		if yesTok != "" && tok != yesTok {
			return false
		}
		return true
	}
}

func round4(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	return math.Round(x*10000) / 10000
}