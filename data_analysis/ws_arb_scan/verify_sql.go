package main

import (
	"fmt"
	"strings"
)

func hitOneLiner(h yesnoHit) string {
	switch strings.ToLower(strings.TrimSpace(h.Mode)) {
	case "replay":
		if h.TriggerWSID > 0 {
			return fmt.Sprintf("回放触发 ws.id=%d（毫秒%d，仅本笔WS落库；对侧用更早缓存）：%s买YES + %s买NO，合成成本%.4f",
				h.TriggerWSID, h.SnapshotMs, platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
		}
		return fmt.Sprintf("回放触发（毫秒%d，非两边同毫秒）：%s买YES + %s买NO，合成成本%.4f",
			h.SnapshotMs, platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
	case "snapshot":
		if h.SnapshotMs > 0 {
			return fmt.Sprintf("同毫秒%d：%s买YES + %s买NO，成本%.4f",
				h.SnapshotMs, platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
		}
	}
	if h.SnapshotMs > 0 {
		return fmt.Sprintf("时刻%d：%s买YES + %s买NO，成本%.4f",
			h.SnapshotMs, platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
	}
	return fmt.Sprintf("%s买YES + %s买NO，成本%.4f（无固定时刻，勿当真）",
		platformLabel(h.YesPlatform), platformLabel(h.NoPlatform), h.Per)
}

// sqlVerifyHit 生成核对 SQL；replay 不能用「同毫秒两边」过滤。
func sqlVerifyHit(mapKey string, h yesnoHit) string {
	mk := sqlLiteral(mapKey)
	yesP := sqlLiteral(strings.ToUpper(strings.TrimSpace(h.YesPlatform)))
	noP := sqlLiteral(strings.ToUpper(strings.TrimSpace(h.NoPlatform)))
	mlIDs := fmt.Sprintf("(SELECT platform_market_outcome_id FROM market_outcome mo JOIN market m ON m.id=mo.market_row_id WHERE m.map_key=%s AND mo.outcome_type='moneyline')", mk)

	if strings.EqualFold(strings.TrimSpace(h.Mode), "replay") && h.TriggerWSID > 0 {
		return fmt.Sprintf("-- 回放核对：① 触发行 ② 各平台独赢在 id<=%d 前各 token 最后一笔（含触发侧本次更新）\n"+
			"SELECT id,platform,platform_market_outcome_id,token_name,outcome_name,snapshot_at_ms,order_book_asks,order_book_bids FROM "+orderBookTable+" WHERE id=%d;\n"+
			"SELECT w.id,w.platform,w.platform_market_outcome_id,w.token_name,w.outcome_name,w.snapshot_at_ms,w.order_book_asks,w.order_book_bids FROM "+orderBookTable+" w JOIN (SELECT platform,token_name,MAX(id) AS max_id FROM "+orderBookTable+" WHERE map_key=%s AND id<=%d AND platform_market_outcome_id IN %s AND platform IN (%s,%s) GROUP BY platform,token_name) t ON w.id=t.max_id ORDER BY w.platform,w.token_name;",
			h.TriggerWSID, h.TriggerWSID, mk, h.TriggerWSID, mlIDs, yesP, noP)
	}

	if h.SnapshotMs <= 0 {
		return fmt.Sprintf("SELECT id,platform,platform_market_outcome_id,outcome_name,snapshot_at_ms,order_book_asks,order_book_bids FROM %s w JOIN market_outcome o ON o.platform=w.platform AND o.platform_market_outcome_id=w.platform_market_outcome_id WHERE w.map_key=%s AND o.outcome_type='moneyline' AND w.platform IN (%s,%s) ORDER BY w.id DESC LIMIT 20;", orderBookTable, mk, yesP, noP)
	}
	if strings.EqualFold(strings.TrimSpace(h.Mode), "snapshot") {
		return fmt.Sprintf("SELECT id,platform,platform_market_outcome_id,token_name,outcome_name,snapshot_at_ms,order_book_asks,order_book_bids FROM %s WHERE map_key=%s AND snapshot_at_ms=%d AND platform IN (%s,%s) AND platform_market_outcome_id IN %s;", orderBookTable, mk, h.SnapshotMs, yesP, noP, mlIDs)
	}
	return fmt.Sprintf("SELECT id,platform,platform_market_outcome_id,outcome_name,snapshot_at_ms,order_book_asks,order_book_bids FROM %s WHERE map_key=%s AND snapshot_at_ms=%d AND platform IN (%s,%s) AND platform_market_outcome_id IN %s;", orderBookTable, mk, h.SnapshotMs, yesP, noP, mlIDs)
}

func sqlLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}