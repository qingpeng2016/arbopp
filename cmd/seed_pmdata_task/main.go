// 写入 10 月英超 Poly+Predict 场次 + data_task（免费 PMData：UTC 日 ≤ 2026-10-30）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	persistententity "github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
)

const (
	fixtureSlug   = "epl-eve-che-2026-10-17"
	mapKey        = "v1|英超|" + fixtureSlug
	competition   = "英超"
	competitionEn = "Premier League"
	fixtureDay    = "2026-10-17"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()
	ctx := context.Background()

	predictKey := predictAPIKey(ctx, db)
	if predictKey == "" {
		fmt.Println("ERR: no Predict API key (platform/account)")
		os.Exit(1)
	}

	pe, err := fetchPolyEvent(fixtureSlug)
	if err != nil {
		fmt.Println("ERR poly:", err)
		os.Exit(1)
	}
	predCat, err := fetchPredictCategory(ctx, predictKey, fixtureSlug)
	if err != nil {
		fmt.Println("ERR predict:", err)
		os.Exit(1)
	}

	polyURL := "https://polymarket.com/event/" + fixtureSlug
	predURL := "https://predict.fun/market/" + fixtureSlug

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM market_outcome WHERE market_row_id IN (
			SELECT id FROM market WHERE map_key = ? AND UPPER(platform) IN ('POLYMARKET','PREDICT'))`, mapKey).Error; err != nil {
			return err
		}
		if err := tx.Where("map_key = ? AND platform IN ?", mapKey, []string{"POLYMARKET", "PREDICT"}).
			Delete(&persistententity.Market{}).Error; err != nil {
			return err
		}

		polyM := persistPolyMarket(tx, pe, polyURL)
		predM := persistPredictMarket(tx, predCat, predURL)
		for _, m := range pe.Markets {
			if !strings.Contains(m.Slug, fixtureSlug) {
				continue
			}
			if strings.Contains(m.Slug, "halftime") || strings.Contains(m.Slug, "exact-score") {
				continue
			}
			if !isMoneyline3(m.Slug, fixtureSlug) {
				continue
			}
			persistPolyOutcome(tx, polyM, m)
		}
		for _, mk := range predCat.Markets {
			if strings.EqualFold(strings.TrimSpace(mk.Variant), "") {
				continue
			}
			if !strings.EqualFold(mk.Variant, "SPORTS_MATCH") && !strings.Contains(strings.ToUpper(mk.Variant), "MATCH") {
				continue
			}
			persistPredictOutcome(tx, predM, mk)
		}

		params := map[string]interface{}{
			"fixture_slug":     fixtureSlug,
			"min_utc_age_days": 21,
			"utc_days_before":  1,
			"utc_days_after":   1,
			"utc_dates":        []string{"2026-10-16", fixtureDay, "2026-10-18"},
			"outcome_types":    []string{"moneyline"},
			"market_url": map[string]string{
				"POLYMARKET": polyURL,
				"PREDICT":    predURL,
			},
			"note": "Everton vs Chelsea 2026-10-17; free PMData OK",
		}
		pj, _ := json.Marshal(params)
		ps := string(pj)
		task := persistententity.DataTask{
			IngestSource: persistententity.DataTaskIngestPmdata,
			MapKey:       "",
			Params:       &ps,
			Status:       persistententity.DataTaskStatusPending,
		}
		_ = tx.Exec(`DELETE FROM data_task WHERE ingest_source = ? AND params LIKE ?`,
			persistententity.DataTaskIngestPmdata, "%"+fixtureSlug+"%")
		return tx.Create(&task).Error
	}); err != nil {
		fmt.Println("ERR tx:", err)
		os.Exit(1)
	}

	fmt.Println("OK seeded pmdata task params.fixture_slug=", fixtureSlug, "market map_key=", mapKey)
	fmt.Println("  POLYMARKET", polyURL)
	fmt.Println("  PREDICT", predURL)
	fmt.Println("  data_task pending params:", fixtureDay)
}

func isMoneyline3(slug, base string) bool {
	for _, suf := range []string{"-eve", "-draw", "-che"} {
		if slug == base+suf {
			return true
		}
	}
	return false
}

func predictAPIKey(ctx context.Context, db *gorm.DB) string {
	var plat persistententity.Platform
	if err := db.WithContext(ctx).Where("slug = ? AND status = ?", "PREDICT", "ACTIVE").First(&plat).Error; err == nil {
		if plat.ExtraConfig != nil {
			var m map[string]interface{}
			if json.Unmarshal([]byte(*plat.ExtraConfig), &m) == nil {
				if k, _ := m["api_key"].(string); strings.TrimSpace(k) != "" {
					return strings.TrimSpace(k)
				}
			}
		}
	}
	var acc struct {
		APIKey string `gorm:"column:api_key"`
	}
	_ = db.WithContext(ctx).Raw(`SELECT api_key FROM account WHERE UPPER(platform)='PREDICT' AND status='ACTIVE' AND TRIM(api_key)<>'' LIMIT 1`).Scan(&acc)
	return strings.TrimSpace(acc.APIKey)
}

type polyEvent struct {
	Slug      string
	Title     string
	StartDate string
	EndDate   string
	Markets   []polyMarket
}

type polyMarket struct {
	Slug         string
	Question     string
	ConditionID  string `json:"conditionId"`
	ClobTokenIds string `json:"clobTokenIds"`
}

func fetchPolyEvent(slug string) (*polyEvent, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://gamma-api.polymarket.com/events?slug="+slug, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var list []polyEvent
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("poly event not found")
	}
	return &list[0], nil
}

type predCategory struct {
	Slug    string
	Title   string
	Markets []predMarket
}

type predMarket struct {
	ID      json.Number `json:"id"`
	Variant string      `json:"marketVariant"`
	Title   string      `json:"title"`
}

func fetchPredictCategory(ctx context.Context, apiKey, slug string) (*predCategory, error) {
	url := "https://api.predict.fun/v1/categories/" + slug
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("x-api-key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("predict http %d: %s", resp.StatusCode, string(body)[:min(200, len(body))])
	}
	var wrap struct {
		Success bool `json:"success"`
		Data    struct {
			Category predCategory `json:"category"`
			Markets  []predMarket `json:"markets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	c := wrap.Data.Category
	c.Markets = wrap.Data.Markets
	if c.Slug == "" {
		c.Slug = slug
	}
	return &c, nil
}

func strPtr(s string) *string { return &s }

func persistPolyMarket(tx *gorm.DB, pe *polyEvent, url string) persistententity.Market {
	name := pe.Title
	slug := fixtureSlug
	cat := string(cryptoPredictionEnum.MarketCategorySports)
	sub := string(cryptoPredictionEnum.SubMarketCategoryFootball)
	sc := "足球"
	mv := 1
	m := persistententity.Market{
		Platform:       string(cryptoPredictionEnum.PlatformPolymarket),
		Name:           &name,
		PlatformMarketID: fixtureSlug,
		MarketURL:      url,
		EventStartTime: &pe.StartDate,
		EventEndTime:   &pe.EndDate,
		Status:         string(cryptoPredictionEnum.MarketStatusActive),
		Category:       &cat,
		SubCategory:    &sub,
		SportCN:        &sc,
		Slug:           &slug,
		Competition:    strPtr(competition),
		CompetitionEn:  strPtr(competitionEn),
		MapKey:         strPtr(mapKey),
		MapVerified:    mv,
	}
	tx.Create(&m)
	return m
}

func persistPredictMarket(tx *gorm.DB, pc *predCategory, url string) persistententity.Market {
	title := pc.Title
	if title == "" {
		title = "Everton FC vs. Chelsea FC"
	}
	slug := fixtureSlug
	cat := string(cryptoPredictionEnum.MarketCategorySports)
	sub := string(cryptoPredictionEnum.SubMarketCategoryFootball)
	sc := "足球"
	es, ee := fixtureDay+"T15:00:00Z", fixtureDay+"T17:00:00Z"
	mv := 1
	m := persistententity.Market{
		Platform:         string(cryptoPredictionEnum.PlatformPredict),
		Name:             &title,
		PlatformMarketID: pc.Slug,
		MarketURL:        url,
		EventStartTime:   &es,
		EventEndTime:     &ee,
		Status:           string(cryptoPredictionEnum.MarketStatusActive),
		Category:         &cat,
		SubCategory:      &sub,
		SportCN:          &sc,
		Slug:             &slug,
		Competition:      strPtr(competition),
		CompetitionEn:    strPtr(competitionEn),
		MapKey:           strPtr(mapKey),
		MapVerified:      mv,
	}
	tx.Create(&m)
	return m
}

func persistPolyOutcome(tx *gorm.DB, parent persistententity.Market, m polyMarket) {
	yes, no := parseClobTokens(m.ClobTokenIds)
	n := m.Question
	sl := m.Slug
	cat := string(cryptoPredictionEnum.MarketCategorySports)
	sub := string(cryptoPredictionEnum.SubMarketCategoryFootball)
	oc := persistententity.MarketOutcome{
		Platform:                parent.Platform,
		MarketRowID:             parent.ID,
		PlatformMarketID:        parent.PlatformMarketID,
		OutcomeName:             &n,
		MarketName:              parent.Name,
		PlatformMarketOutcomeID: m.ConditionID,
		MarketURL:               parent.MarketURL,
		YesTokenID:              yes,
		NoTokenID:               no,
		Category:                &cat,
		SubCategory:             &sub,
		OutcomeType:             "moneyline",
		Slug:                    &sl,
		Competition:             strPtr(competition),
	}
	tx.Create(&oc)
}

func persistPredictOutcome(tx *gorm.DB, parent persistententity.Market, mk predMarket) {
	title := mk.Title
	cat := string(cryptoPredictionEnum.MarketCategorySports)
	sub := string(cryptoPredictionEnum.SubMarketCategoryFootball)
	sl := fixtureSlug
	oc := persistententity.MarketOutcome{
		Platform:                parent.Platform,
		MarketRowID:             parent.ID,
		PlatformMarketID:        parent.PlatformMarketID,
		OutcomeName:             &title,
		MarketName:              parent.Name,
		PlatformMarketOutcomeID: mk.ID.String(),
		MarketURL:               parent.MarketURL,
		OutcomeType:             "moneyline",
		Slug:                    &sl,
		Category:                &cat,
		SubCategory:             &sub,
		Competition:             strPtr(competition),
	}
	tx.Create(&oc)
}

func parseClobTokens(raw string) (yes, no string) {
	raw = strings.TrimSpace(raw)
	var ids []string
	if json.Unmarshal([]byte(raw), &ids) == nil && len(ids) >= 2 {
		return ids[0], ids[1]
	}
	return "", ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}