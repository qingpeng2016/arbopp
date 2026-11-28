package notification

import (
	"fmt"
	"strings"

	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

// runNodeLabel 用于 Lark 正文首行（--node / RUN_NODE）。
func runNodeLabel() string {
	node := strings.ToUpper(strings.TrimSpace(conf.GetRunNode()))
	if node == "" {
		return "ALL"
	}
	return node
}

func larkContentWithPlatform(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Sprintf("**平台:** %s", runNodeLabel())
	}
	return fmt.Sprintf("**平台:** %s\n%s", runNodeLabel(), body)
}

func prependPlatformField(fields []FieldPair) []FieldPair {
	label := runNodeLabel()
	if len(fields) > 0 && fields[0].Key == "平台" {
		fields[0].Value = label
		return fields
	}
	out := make([]FieldPair, 0, len(fields)+1)
	out = append(out, FieldPair{Key: "平台", Value: label})
	out = append(out, fields...)
	return out
}