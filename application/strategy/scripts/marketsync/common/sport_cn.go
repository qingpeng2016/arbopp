package common

import "strings"

// SportCNForSubCategory 与 market_category.sport_cn 一致。
func SportCNForSubCategory(sub string) string {
	switch strings.TrimSpace(sub) {
	case "Football":
		return "足球"
	case "Cricket":
		return "板球"
	case "Tennis":
		return "网球"
	case "Basketball":
		return "篮球"
	case "NFL":
		return "美式足球"
	case "NHL":
		return "冰球"
	case "MMA":
		return "UFC"
	default:
		return ""
	}
}