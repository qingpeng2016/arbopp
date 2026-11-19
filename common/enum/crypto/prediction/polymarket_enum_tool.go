package prediction

import (
	"fmt"
)

// ToPolymarketSportIDs 将业务层 SubMarketCategory 转换为 Polymarket 的 sport 标识列表。
// 不依赖 DB：tag 列表由 GET /sports 动态拉取；此处仅做「子类→sport 白名单」映射，Polymarket 新增运动时在此处维护即可。
func ToPolymarketSportIDs(subCat SubMarketCategory) ([]string, error) {
	switch subCat {
	case SubMarketCategoryDefault:
		return nil, nil // 空表示不过滤，拉取全部
	case SubMarketCategoryFootball:
		return polymarketSportIDsFootball(), nil
	case SubMarketCategoryBasketball:
		return polymarketSportIDsBasketball(), nil
	case SubMarketCategoryNFL:
		return polymarketSportIDsNFL(), nil
	case SubMarketCategoryMLB:
		return polymarketSportIDsMLB(), nil
	case SubMarketCategoryNHL:
		return polymarketSportIDsNHL(), nil
	case SubMarketCategoryTennis:
		return polymarketSportIDsTennis(), nil
	case SubMarketCategoryMMA:
		return polymarketSportIDsMMA(), nil
	case SubMarketCategoryBoxing:
		return polymarketSportIDsBoxing(), nil
	case SubMarketCategoryGolf:
		return polymarketSportIDsGolf(), nil
	case SubMarketCategoryEsports:
		return polymarketSportIDsEsports(), nil
	case SubMarketCategoryRugby:
		return polymarketSportIDsRugby(), nil
	case SubMarketCategoryCricket:
		return polymarketSportIDsCricket(), nil
	case SubMarketCategoryWinterOlympics:
		return polymarketSportIDsWinterOlympics(), nil
	case SubMarketCategoryTableTennis:
		return polymarketSportIDsTableTennis(), nil
	case SubMarketCategoryFormula1:
		return polymarketSportIDsFormula1(), nil
	case SubMarketCategoryChess:
		return polymarketSportIDsChess(), nil
	case SubMarketCategoryPickleball:
		return polymarketSportIDsPickleball(), nil
	default:
		return nil, fmt.Errorf("不支持的 SubMarketCategory: %s", subCat)
	}
}

// polymarketSportIDsFootball 足球 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsFootball() []string {
	return []string{
		"epl", "lal", "bun", "fl1", "sea", "ucl", "uel", "mls", "fif", "uef",
		"con", "caf", "cof", "bra", "arg", "mex", "lib", "sud", "tur", "rus",
		"efa", "efl", "ere", "itc", "cdr", "dfb", "cde", "col", "afc",
		"ofc", "jap", "ja2", "kor", "spl", "chi", "aus", "idn1", "nor", "den",
		"por", "acn", "ssc",
	}
}

// polymarketSportIDsBasketball 篮球 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsBasketball() []string {
	return []string{
		"nba", "ncaab", "cbb", "wnba", "cwbb",
		"bkligend", "bknbl", "bkcl", "bkseriea", "bkcba", "bkfr1", "bkarg", "bkkbl",
	}
}

// polymarketSportIDsNFL 美式橄榄球 sport 标识（来自 Polymarket GET /sports 实际返回，含 NFL+CFB）
func polymarketSportIDsNFL() []string {
	return []string{"nfl", "cfb"}
}

// polymarketSportIDsMLB 棒球 sport 标识（来自 Polymarket GET /sports 实际返回，含 MLB+KBO）
func polymarketSportIDsMLB() []string {
	return []string{"mlb", "kbo"}
}

// polymarketSportIDsNHL 冰球 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsNHL() []string {
	return []string{"nhl", "ahl", "shl", "cehl", "dehl", "snhl", "khl"}
}

// polymarketSportIDsTennis 网球 sport 标识（来自 Polymarket GET /sports 实际返回：atp/wta）
func polymarketSportIDsTennis() []string {
	return []string{"atp", "wta"}
}

// polymarketSportIDsMMA 综合格斗 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsMMA() []string {
	return []string{"ufc"}
}

// polymarketSportIDsBoxing 拳击 sport 标识（当前 GET /sports 响应中未看到，若需过滤请核对 API 后更新）
func polymarketSportIDsBoxing() []string {
	return []string{"boxing"}
}

// polymarketSportIDsGolf 高尔夫 sport 标识（当前 GET /sports 响应中未看到，若需过滤请核对 API 后更新）
func polymarketSportIDsGolf() []string {
	return []string{"golf"}
}

// polymarketSportIDsEsports 电竞/游戏 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsEsports() []string {
	return []string{
		"dota2", "lol", "cs2", "val", "mlbb", "ow", "codmw", "fifa",
		"pubg", "r6siege", "rl", "hok", "wildrift", "sc2", "sc",
	}
}

// polymarketSportIDsRugby 橄榄球 sport 标识（来自 Polymarket GET /sports 实际返回，含 Six Nations 等）
func polymarketSportIDsRugby() []string {
	return []string{
		"rusixnat", "ruprem", "rutopft", "rueuchamp", "ruurc", "rusrp", "ruchamp",
	}
}

// polymarketSportIDsCricket 板球 sport 标识（来自 Polymarket GET /sports 实际返回）
func polymarketSportIDsCricket() []string {
	return []string{
		"ipl", "odi", "t20", "abb", "csa", "crban", "crint", "craus", "creng",
		"crnew", "crind", "crsou", "crpak", "cruae", "test", "she", "sasa", "lpl", "psp",
		"cru19wc", "crwpl20",
	}
}

// polymarketSportIDsWinterOlympics 冬季奥运会（GET /sports 若有返回请填入实际 sport 标识）
func polymarketSportIDsWinterOlympics() []string {
	return []string{"wo", "winteroly", "oly"}
}

// polymarketSportIDsTableTennis 乒乓球 WTT（GET /sports 若有返回请填入实际 sport 标识）
func polymarketSportIDsTableTennis() []string {
	return []string{"wtt", "wttm", "wttw"}
}

// polymarketSportIDsFormula1 F1（GET /sports 若有返回请填入实际 sport 标识）
func polymarketSportIDsFormula1() []string {
	return []string{"f1", "formula1"}
}

// polymarketSportIDsChess 国际象棋（GET /sports 若有返回请填入实际 sport 标识）
func polymarketSportIDsChess() []string {
	return []string{"chess"}
}

// polymarketSportIDsPickleball 匹克球（GET /sports 若有返回请填入实际 sport 标识）
func polymarketSportIDsPickleball() []string {
	return []string{"pickleball", "pickle"}
}
