package predict

import "strings"

// Predict 让分/大小等挂在独立 category（如 …-more-markets，marketVariant=SPORTS_PROPS），
// 与主胜负盘 category（SPORTS_MATCH）slug 不同；库内统一归到主场次 slug。
var predictCategorySlugSuffixes = []string{
	"-more-markets",
	"-halftime-result",
	"-first-to-score",
	"-total-corners",
	"-exact-score",
	"-second-half-result",
}

func predictMainFixtureID(categorySlug string) string {
	slug := strings.TrimSpace(categorySlug)
	if slug == "" {
		return ""
	}
	for _, suf := range predictCategorySlugSuffixes {
		if strings.HasSuffix(slug, suf) {
			return strings.TrimSuffix(slug, suf)
		}
	}
	return slug
}

func predictIsMainMatchCategory(categorySlug string) bool {
	slug := strings.TrimSpace(categorySlug)
	if slug == "" {
		return false
	}
	return slug == predictMainFixtureID(slug)
}