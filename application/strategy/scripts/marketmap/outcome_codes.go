package marketmap

import (
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
)

// polySlugOutcomeCode 末段队缩写，如 epl-ast-sun-2026-11-21-ast → ast
func polySlugOutcomeCode(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	parts := strings.Split(slug, "-")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// polyOutcomeCodeNorm Poly slug 队码 → 与 Kalshi「X wins」归一化一致的 outcomeNorm
var polyOutcomeCodeNorm = map[string]string{
	// 英超
	"ast": "aston_villa", "sun": "sunderland", "cry": "crystal_palace", "cov": "coventry",
	"mac": "manchester_city", "ful": "fulham", "ars": "arsenal", "new": "newcastle",
	"ips": "ipswich", "tot": "tottenham", "bou": "bournemouth", "not": "nottingham_forest",
	"che": "chelsea", "lee": "leeds", "bri": "brighton", "hul": "hull", "liv": "liverpool", "mun": "manchester_united",
	// 意甲
	"tor": "torino", "juv": "juventus", "ven": "venezia", "nap": "napoli", "com": "como", "cag": "cagliari",
	"laz": "lazio", "lec": "lecce", "par": "parma", "rom": "roma", "mil": "milan", "fro": "frosinone",
	"mon": "monza", "fio": "fiorentina", "ata": "atalanta", "int": "inter", "bol": "bologna",
	"udi": "udinese", "sas": "sassuolo", "gen": "genoa",
	// 德甲
	"bay": "bayern_munich", "koe": "koln", "moe": "monchengladbach", "dor": "dortmund",
	"b04": "leverkusen", "scp": "paderborn", "s04": "schalke", "stu": "stuttgart",
	"ein": "frankfurt", "hof": "hoffenheim", "elv": "elversberg", "fre": "freiburg",
	"lei": "leipzig", "uni": "union_berlin", "aug": "augsburg", "mai": "mainz", "wer": "bremen", "hsv": "hamburg",
	// 西甲
	"bar": "barcelona", "vil": "villarreal", "elc": "elche", "lev": "levante",
	"bil": "bilbao", "esp": "espanyol", "ala": "alaves", "dep": "deportivo",
	"rea": "real_madrid", "cel": "celta_vigo", "ray": "vallecano", "val": "valencia",
	"sev": "sevilla", "bet": "betis", "get": "getafe", "mad": "atletico",
	"osa": "osasuna", "mala": "malaga", "rrc": "santander", "rso": "real_sociedad",
}

// outcomeNormAliases 全名归一化后与 Kalshi 短名对齐
var outcomeNormAliases = map[string]string{
	"m_borussia_monchengladbach": "monchengladbach",
	"borussia_monchengladbach":   "monchengladbach",
	"mgladbach":                  "monchengladbach",
	"monchengladbach":            "monchengladbach",
	"bayer_04_leverkusen":        "leverkusen",
	"bv_borussia_09_dortmund":    "dortmund",
	"borussia_dortmund":          "dortmund",
	"1_fc_koln":                  "koln",
	"fc_bayern_munchen":          "bayern_munich",
	"fc_bayern_munich":           "bayern_munich",
	"newcastle_united":           "newcastle",
	"tottenham":                  "tottenham",
	"tottenham_hotspur":          "tottenham",
	"ipswich":                    "ipswich",
	"ipswich_town":               "ipswich",
	"crystal_palace":             "crystal_palace",
	"brighton_hove_albion":       "brighton",
	"brighton":                   "brighton",
	"hull":                       "hull",
	"coventry":                   "coventry",
	"parma_calcio":               "parma",
	"parma_calcio_1913":          "parma",
	"ac_milan":                   "milan",
	"fc_internazionale_milano":   "inter",
	"atalanta_bc":                "atalanta",
	"bologna_fc_1909":            "bologna",
	"udinese_calcio":             "udinese",
	"deportivo_alaves":           "alaves",
	"rc_deportivo_a_coruna":      "deportivo",
	"athletic_club":              "bilbao",
	"athletic_bilbao":            "bilbao",
	"club_atletico_de_madrid":    "atletico",
	"atletico_madrid":            "atletico",
	"real_racing_club":           "santander",
	"hull_city":                  "hull",
	"leeds_united":               "leeds",
	"fc_koln":                    "koln",
	"ssc_napoli":                 "napoli",
	"real_sociedad_de_futbol":    "real_sociedad",
	"rayo_vallecano_de_madrid":   "vallecano",
	"afc_bournemouth":            "bournemouth",
	"nottingham_forest":          "nottingham_forest",
	"leeds":                      "leeds",
	"manchester_united":          "manchester_united",
	"manchester_city":            "manchester_city",
	"vallecano":                  "vallecano",
	"deportivo_de_la_coruna":     "deportivo",
	"real_betis_balompie":        "betis",
	"real_betis":                 "betis",
	"crystal":                    "crystal_palace",
	"nottingham":                 "nottingham_forest",
}

func canonicalOutcomeNorm(platform, team, polySlug string) string {
	if strings.ToUpper(strings.TrimSpace(platform)) == appcommon.PlatformPolymarket {
		if code := polySlugOutcomeCode(polySlug); code != "" {
			if n, ok := polyOutcomeCodeNorm[code]; ok {
				return n
			}
		}
	}
	n := normalizeTeam(team)
	if n == "draw" {
		return n
	}
	if a, ok := outcomeNormAliases[n]; ok {
		return a
	}
	return n
}
