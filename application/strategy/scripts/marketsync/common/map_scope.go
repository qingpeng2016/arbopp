package common

import "strings"

// MapMatchScopeKey 跨平台场次对齐桶：同一 category + sub_category + competition_en 内才允许配对。
func MapMatchScopeKey(category, subCategory, competitionEn string) string {
	category = strings.TrimSpace(category)
	subCategory = strings.TrimSpace(subCategory)
	competitionEn = strings.TrimSpace(competitionEn)
	if category == "" || subCategory == "" || competitionEn == "" {
		return ""
	}
	return category + "/" + subCategory + "/" + competitionEn
}

// MapMatchScopeOK 是否具备跨平台对齐所需维度。
func MapMatchScopeOK(category, subCategory, competitionEn string) bool {
	return MapMatchScopeKey(category, subCategory, competitionEn) != ""
}