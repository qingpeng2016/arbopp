package statcollect

import "encoding/json"

// marshalStatJSON 写入 stat_kv 的 JSON（统一 2 空格缩进，便于人工查看）。
func marshalStatJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}