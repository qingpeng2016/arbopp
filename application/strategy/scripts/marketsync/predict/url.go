package predict

import (
	"net/url"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/common"
)

const predictWebOrigin = "https://predict.fun"

// buildPredictFixtureURL 官网场次页：/market/{categorySlug}（非 /category/、非数字 market id）。
func buildPredictFixtureURL(categorySlug string) string {
	slug := strings.Trim(strings.ToLower(strings.TrimSpace(categorySlug)), "/")
	if slug == "" {
		return predictWebOrigin
	}
	u := predictWebOrigin + "/market/" + url.PathEscape(slug)
	return common.Truncate(u, 255)
}