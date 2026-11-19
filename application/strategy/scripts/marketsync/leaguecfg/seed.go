package leaguecfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SeedFromDocsJSON(db *gorm.DB, docsSQLDir string) error {
	rows, err := categoryRowsFromDocs(docsSQLDir)
	if err != nil {
		return err
	}
	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			row := rows[i]
			row.CreatedAt = now
			row.UpdatedAt = now
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "sub_category"}, {Name: "platform"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"sport_cn", "concurrency", "config_json", "outcome_types_json", "sort_no", "is_enabled", "updated_at",
				}),
			}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type manifestRow struct {
	SortNo              int      `json:"sort_no"`
	SportCN             string   `json:"sport_cn"`
	SubCategory         string   `json:"sub_category"`
	OutcomeTypes        []string `json:"outcome_types"`
	PolymarketConc      int      `json:"polymarket_concurrency"`
	PredictConc         int      `json:"predict_concurrency"`
}

type sportGroup struct {
	SportCN     string          `json:"sport_cn"`
	SubCategory string          `json:"sub_category"`
	Raw         json.RawMessage `json:"-"`
}

func categoryRowsFromDocs(docsSQLDir string) ([]entity.ConfigMarketCategory, error) {
	manifestPath := filepath.Join(docsSQLDir, "market_category_manifest.json")
	polyPath := filepath.Join(docsSQLDir, "market_category_polymarket.json")
	predictPath := filepath.Join(docsSQLDir, "market_category_predict.json")

	manifest, err := readManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	polyGroups, err := readSportGroups(polyPath)
	if err != nil {
		return nil, err
	}
	predictGroups, err := readSportGroups(predictPath)
	if err != nil {
		return nil, err
	}
	polyBySub := indexGroupsBySub(polyGroups)
	predictBySub := indexGroupsBySub(predictGroups)

	var out []entity.ConfigMarketCategory
	for _, m := range manifest {
		sub := strings.TrimSpace(m.SubCategory)
		poly, okP := polyBySub[sub]
		prd, okR := predictBySub[sub]
		if !okP || !okR {
			return nil, fmt.Errorf("manifest %s: missing polymarket=%v predict=%v group", sub, okP, okR)
		}
		otypes, err := json.Marshal(m.OutcomeTypes)
		if err != nil {
			return nil, err
		}
		sportCN := strings.TrimSpace(m.SportCN)
		if sportCN == "" {
			sportCN = strings.TrimSpace(poly.SportCN)
		}
		concP := m.PolymarketConc
		if concP <= 0 {
			concP = 3
		}
		concR := m.PredictConc
		if concR <= 0 {
			concR = 2
		}
		out = append(out,
			entity.ConfigMarketCategory{
				SportCN: sportCN, SubCategory: sub, Platform: appcommon.PlatformPolymarket,
				Concurrency: concP, ConfigJSON: string(poly.Raw), OutcomeTypesJSON: string(otypes),
				SortNo: m.SortNo, IsEnabled: 1,
			},
			entity.ConfigMarketCategory{
				SportCN: sportCN, SubCategory: sub, Platform: appcommon.PlatformPredict,
				Concurrency: concR, ConfigJSON: string(prd.Raw), OutcomeTypesJSON: string(otypes),
				SortNo: m.SortNo, IsEnabled: 1,
			},
		)
	}
	return out, nil
}

func readManifest(path string) ([]manifestRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var rows []manifestRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return rows, nil
}

func readSportGroups(path string) ([]sportGroup, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make([]sportGroup, 0, len(arr))
	for _, block := range arr {
		var head struct {
			SportCN     string `json:"sport_cn"`
			SubCategory string `json:"sub_category"`
		}
		if err := json.Unmarshal(block, &head); err != nil {
			return nil, fmt.Errorf("parse group in %s: %w", path, err)
		}
		cfg, lc := sportGroupFromBlock(block)
		out = append(out, sportGroup{
			SportCN: head.SportCN, SubCategory: head.SubCategory, Raw: json.RawMessage(cfg),
		})
		_ = lc
	}
	return out, nil
}

func indexGroupsBySub(groups []sportGroup) map[string]sportGroup {
	m := make(map[string]sportGroup, len(groups))
	for _, g := range groups {
		sub := strings.TrimSpace(g.SubCategory)
		if sub != "" {
			m[sub] = g
		}
	}
	return m
}