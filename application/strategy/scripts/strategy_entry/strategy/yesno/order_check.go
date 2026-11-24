package yesno

import (
	"encoding/json"
	"math"
)

// OrderSizeCheckInput 多平台限价 FAK 下单前份额检查入参。
//
// 口径：
//   - BestAsks[i]：平台 i 下单限价（= 该腿 best_ask）
//   - AskSizes[i]：该限价档挂单份额
//   - MinAmount / MaxAmount：config_strategy_threshold.min_amount / max_amount（金额）
type OrderSizeCheckInput struct {
	Platforms []string
	BestAsks  []float64
	AskSizes  []float64
	MinAmount float64
	MaxAmount float64
}

// OrderSizeCheckResult 下单前份额检查结果；FailReason/Snapshot 写入 arb_binary_opportunity.ext_data。
type OrderSizeCheckResult struct {
	OK               bool
	Qty              float64
	PlatformMinPrice float64
	MinQty           float64
	MaxQty           float64
	PlatformMinQty   float64
	FailReason       string
	// Snapshot 完整检查现场，便于排查。
	Snapshot map[string]any
}

// CheckOrderSize 多平台下单份额检查（完整函数，便于排查）。
//
// 步骤：
//  1. platform_min_price = min(各平台 best_ask)
//  2. min_qty = min_amount / platform_min_price
//     max_qty = max_amount / platform_min_price
//  3. platform_min_qty = min(各平台 best_ask 档挂单份额)
//  4. platform_min_qty < min_qty → 不下单
//     min_qty ≤ platform_min_qty ≤ max_qty → 下单份额 = platform_min_qty
//     platform_min_qty > max_qty → 下单份额 = max_qty
func CheckOrderSize(in OrderSizeCheckInput) OrderSizeCheckResult {
	n := len(in.BestAsks)
	legs := make([]map[string]any, 0, n)
	for i := 0; i < n; i++ {
		plat := ""
		if i < len(in.Platforms) {
			plat = in.Platforms[i]
		}
		ask, sz := 0.0, 0.0
		if i < len(in.BestAsks) {
			ask = in.BestAsks[i]
		}
		if i < len(in.AskSizes) {
			sz = in.AskSizes[i]
		}
		legs = append(legs, map[string]any{
			"platform":  plat,
			"best_ask":  ask,
			"ask_size":  sz,
			"limit_px":  ask,
		})
	}
	base := map[string]any{
		"check":      "order_size",
		"min_amount": in.MinAmount,
		"max_amount": in.MaxAmount,
		"legs":       legs,
	}

	fail := func(reason string, extra map[string]any) OrderSizeCheckResult {
		snap := cloneMap(base)
		for k, v := range extra {
			snap[k] = v
		}
		snap["fail_reason"] = reason
		snap["ok"] = false
		return OrderSizeCheckResult{FailReason: reason, Snapshot: snap}
	}

	if n == 0 || len(in.AskSizes) != n {
		return fail("invalid_legs", map[string]any{
			"best_asks_len": len(in.BestAsks),
			"ask_sizes_len": len(in.AskSizes),
		})
	}
	if !(in.MinAmount > 0) || !(in.MaxAmount > 0) {
		return fail("amount_config_invalid", nil)
	}
	if in.MaxAmount+1e-12 < in.MinAmount {
		return fail("max_amount_lt_min_amount", nil)
	}

	minPrice := math.Inf(1)
	minQtyAtAsk := math.Inf(1)
	for i := 0; i < n; i++ {
		ask, sz := in.BestAsks[i], in.AskSizes[i]
		if !(ask > 0) || math.IsNaN(ask) || math.IsInf(ask, 0) {
			return fail("best_ask_invalid", map[string]any{"leg_index": i, "best_ask": ask})
		}
		if !(sz > 0) || math.IsNaN(sz) || math.IsInf(sz, 0) {
			return fail("ask_size_invalid", map[string]any{"leg_index": i, "ask_size": sz})
		}
		if ask < minPrice {
			minPrice = ask
		}
		if sz < minQtyAtAsk {
			minQtyAtAsk = sz
		}
	}

	minQty := in.MinAmount / minPrice
	maxQty := in.MaxAmount / minPrice
	extra := map[string]any{
		"platform_min_price": minPrice,
		"min_qty":            minQty,
		"max_qty":            maxQty,
		"platform_min_qty":   minQtyAtAsk,
	}

	if minQtyAtAsk+1e-12 < minQty {
		return fail("platform_min_qty_lt_min_qty", extra)
	}

	qty := minQtyAtAsk
	rule := "use_platform_min_qty"
	if minQtyAtAsk > maxQty+1e-12 {
		qty = maxQty
		rule = "cap_at_max_qty"
	}
	extra["order_qty"] = qty
	extra["qty_rule"] = rule
	extra["ok"] = true

	snap := cloneMap(base)
	for k, v := range extra {
		snap[k] = v
	}
	return OrderSizeCheckResult{
		OK:               true,
		Qty:              qty,
		PlatformMinPrice: minPrice,
		MinQty:           minQty,
		MaxQty:           maxQty,
		PlatformMinQty:   minQtyAtAsk,
		Snapshot:         snap,
	}
}

// ExtDataJSON 将检查快照序列化为 ext_data 文本。
func ExtDataJSON(parts ...map[string]any) string {
	out := map[string]any{}
	for _, p := range parts {
		for k, v := range p {
			out[k] = v
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return `{"error":"ext_data_marshal_failed"}`
	}
	return string(b)
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+8)
	for k, v := range in {
		out[k] = v
	}
	return out
}
