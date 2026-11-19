package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leaguecfg"
	"github.com/gph-tech/fgmm-strategy-bitfinex/boot"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
)

func main() {
	_ = conf.NewCfg()
	db := boot.NewDBClient()

	docsDir := "docs/sql"
	if len(os.Args) > 1 {
		docsDir = os.Args[1]
	}
	docsDir, _ = filepath.Abs(docsDir)

	if err := leaguecfg.SeedFromDocsJSON(db, docsDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("seeded market_category from %s (one row per platform)\n", docsDir)
}