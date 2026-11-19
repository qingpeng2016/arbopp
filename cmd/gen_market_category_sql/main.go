package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gph-tech/fgmm-strategy-bitfinex/application/strategy/scripts/marketsync/leaguecfg"
)

func main() {
	docsDir := "docs/sql"
	outPath := "docs/sql/migration_config_market_category_data.sql"
	if len(os.Args) > 1 {
		docsDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}
	docsDir, _ = filepath.Abs(docsDir)
	outPath, _ = filepath.Abs(outPath)

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := leaguecfg.WriteSeedSQL(f, docsDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", outPath)

	installPath := filepath.Join(filepath.Dir(outPath), "config_market_category_install.sql")
	ddl, err := os.ReadFile(filepath.Join(filepath.Dir(outPath), "migration_config_market_category.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "skip install merge: %v\n", err)
		return
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return
	}
	inst, err := os.Create(installPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "install file: %v\n", err)
		return
	}
	defer inst.Close()
	_, _ = inst.Write(ddl)
	_, _ = inst.WriteString("\n")
	_, _ = inst.Write(data)
	fmt.Fprintf(os.Stderr, "wrote %s\n", installPath)
}