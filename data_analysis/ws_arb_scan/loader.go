package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type wsRow struct {
	MapKey                  string `json:"map_key"`
	Platform                string `json:"platform"`
	PlatformMarketOutcomeID string `json:"platform_market_outcome_id"`
	TokenID                 string `json:"token_id"`
	Asks                    string `json:"order_book_asks"`
	Bids                    string `json:"order_book_bids"`
	SnapshotAtMs            int64  `json:"snapshot_at_ms"`
	ID                      int64  `json:"id"`
}

func loadMapKeys(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var keys []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k := strings.TrimSpace(sc.Text())
		if k == "" || strings.HasPrefix(k, "#") {
			continue
		}
		keys = append(keys, k)
	}
	return keys, sc.Err()
}

func readJSONL(path string, decode func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("缺少文件 %s（见 data/README.md）", path)
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 256*1024), 10*1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if err := decode([]byte(line)); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
	}
	return sc.Err()
}

func loadOutcomesFile(path string, want map[string]struct{}) (map[string][]outcomeRec, error) {
	out := map[string][]outcomeRec{}
	err := readJSONL(path, func(b []byte) error {
		var row outcomeRec
		if err := json.Unmarshal(b, &row); err != nil {
			return err
		}
		mk := strings.TrimSpace(row.MapKey)
		if mk == "" {
			return nil
		}
		if len(want) > 0 {
			if _, ok := want[mk]; !ok {
				return nil
			}
		}
		out[mk] = append(out[mk], row)
		return nil
	})
	return out, err
}

func loadWSFile(path string, want map[string]struct{}) (map[string][]wsRow, error) {
	out := map[string][]wsRow{}
	err := readJSONL(path, func(b []byte) error {
		var row wsRow
		if err := json.Unmarshal(b, &row); err != nil {
			return err
		}
		mk := strings.TrimSpace(row.MapKey)
		if mk == "" {
			return nil
		}
		if len(want) > 0 {
			if _, ok := want[mk]; !ok {
				return nil
			}
		}
		out[mk] = append(out[mk], row)
		return nil
	})
	return out, err
}

func splitLatestHist(rows []wsRow) (latest, hist []wsRow) {
	type key struct{ plat, hash, tok string }
	seen := map[key]bool{}
	for _, r := range rows {
		k := key{strings.ToUpper(r.Platform), r.PlatformMarketOutcomeID, r.TokenID}
		if !seen[k] {
			seen[k] = true
			latest = append(latest, r)
		}
	}
	hist = rows
	if len(hist) > 8000 {
		hist = hist[:8000]
	}
	return latest, hist
}

func filepathDir(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return ""
}

func runScan(dataDir, keysFile, outPath string) error {
	if keysFile == "" {
		keysFile = filepath.Join(dataDir, "map_keys.txt")
	}
	keys, err := loadMapKeys(keysFile)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return fmt.Errorf("map_keys 为空: %s", keysFile)
	}
	want := map[string]struct{}{}
	for _, k := range keys {
		want[k] = struct{}{}
	}

	outcomePath := filepath.Join(dataDir, "market_outcome.jsonl")
	wsPath := filepath.Join(dataDir, "ws_book.jsonl")

	outcomesByKey, err := loadOutcomesFile(outcomePath, want)
	if err != nil {
		return err
	}
	wsByKey, err := loadWSFile(wsPath, want)
	if err != nil {
		return err
	}

	var reports []fixtureReport
	for _, mk := range keys {
		latest, hist := splitLatestHist(wsByKey[mk])
		reports = append(reports, scanFixture(mk, outcomesByKey[mk], latest, hist))
	}

	if dir := filepathDir(outPath); dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	outF, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outF.Close()
	enc := json.NewEncoder(outF)
	for _, r := range reports {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	sumPath := strings.TrimSuffix(outPath, ".jsonl") + "_summary.txt"
	_ = os.WriteFile(sumPath, []byte(summarize(reports)+"\n"), 0644)
	fmt.Println(summarize(reports))
	fmt.Println("written:", outPath, sumPath)
	return nil
}