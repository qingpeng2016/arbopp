package conf

import (
	"flag"
	"os"
	"strings"
)

var runNode = flag.String("node", "ALL", "node role: ALL (legacy all WS) | POLY | PRED | SXBET")

// GetRunNode returns --node or env RUN_NODE (default ALL).
func GetRunNode() string {
	if v := strings.TrimSpace(os.Getenv("RUN_NODE")); v != "" {
		return strings.ToUpper(v)
	}
	if runNode == nil {
		return "ALL"
	}
	return strings.ToUpper(strings.TrimSpace(*runNode))
}

// IsNodeAllMode true when all platform WS and direct broker (no cross-node relay).
func IsNodeAllMode() bool {
	n := GetRunNode()
	return n == "" || strings.EqualFold(n, "ALL")
}