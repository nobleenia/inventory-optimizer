// Inventory Optimizer — main entry point.
//
// Modes:
//
//	CLI mode (default):
//	  inventory-optimizer -sales data/sales_history.csv -params data/sku_parameters.csv
//
//	Web mode:
//	  inventory-optimizer -web
//	  inventory-optimizer -web -port :3000
//
// The program loads CSV inputs, computes demand statistics, derives
// optimal inventory policies, runs Monte-Carlo simulations, and
// presents the results either on the terminal or in a web browser.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/engine"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/web"
)

func main() {
	// ── CLI flags ──────────────────────────────────────────────────────
	webMode := flag.Bool("web", false,
		"Start the web server instead of running CLI analysis")
	port := flag.String("port", ":8080",
		"Port for the web server (used with -web)")
	salesPath := flag.String("sales", "data/sales_history.csv",
		"Path to weekly sales history CSV")
	paramsPath := flag.String("params", "data/sku_parameters.csv",
		"Path to SKU parameters CSV")
	outputPath := flag.String("output", "",
		"Path for CSV export (optional; leave blank to skip)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("inventory-optimizer v%s\n", models.Version)
		os.Exit(0)
	}

	if *webMode {
		runWeb(*port)
	} else {
		runCLI(*salesPath, *paramsPath, *outputPath)
	}
}

// ---------------------------------------------------------------------------
// Web mode
// ---------------------------------------------------------------------------

func runWeb(port string) {
	server := web.NewServer(port)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}

// ---------------------------------------------------------------------------
// CLI mode
// ---------------------------------------------------------------------------

func runCLI(salesPath, paramsPath, outputPath string) {
	start := time.Now()
	opts := engine.DefaultOptions()

	log.Println("Loading data and running analysis …")
	reports, warnings, err := engine.RunFromFiles(salesPath, paramsPath, opts)
	if err != nil {
		log.Fatalf("Analysis failed: %v\n", err)
	}

	for _, w := range warnings {
		log.Printf("  ⚠  %s\n", w)
	}

	reporting.PrintCLI(os.Stdout, reports)

	if outputPath != "" {
		if err := reporting.ExportCSV(outputPath, reports); err != nil {
			log.Fatalf("CSV export failed: %v\n", err)
		}
		log.Printf("CSV report saved to %s\n", outputPath)
	}

	log.Printf("Done in %s.\n", time.Since(start).Round(time.Millisecond))
}
