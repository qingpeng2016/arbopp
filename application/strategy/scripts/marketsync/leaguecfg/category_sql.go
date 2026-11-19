package leaguecfg

import (
	"fmt"
	"io"
	"strings"
)

func WriteSeedSQL(w io.Writer, docsSQLDir string) error {
	rows, err := categoryRowsFromDocs(docsSQLDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "INSERT INTO `market_category` (`id`, `sport_cn`, `sub_category`, `platform`, `concurrency`, `config_json`, `outcome_types_json`, `sort_no`, `is_enabled`, `created_at`, `updated_at`)")
	fmt.Fprintln(w, "VALUES")
	for i, row := range rows {
		comma := ","
		if i == len(rows)-1 {
			comma = ""
		}
		fmt.Fprintf(w, "\t(%d, %s, %s, %s, %d, %s, %s, %d, 1, NOW(), NOW())%s\n",
			i+1, sqlStr(row.SportCN), sqlStr(row.SubCategory), sqlStr(row.Platform), row.Concurrency,
			sqlStr(row.ConfigJSON), sqlStr(row.OutcomeTypesJSON), row.SortNo, comma)
	}
	fmt.Fprintln(w, "ON DUPLICATE KEY UPDATE")
	fmt.Fprintln(w, "`sport_cn`=VALUES(`sport_cn`), `concurrency`=VALUES(`concurrency`), `config_json`=VALUES(`config_json`),")
	fmt.Fprintln(w, "`outcome_types_json`=VALUES(`outcome_types_json`),")
	fmt.Fprintln(w, "`sort_no`=VALUES(`sort_no`), `is_enabled`=1, `updated_at`=NOW();")
	return nil
}

func sqlStr(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "''")
	return "'" + s + "'"
}