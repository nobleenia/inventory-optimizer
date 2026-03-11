// Inventory Optimizer — main entry point.
//
// Usage:
//
//	inventory-optimizer \
//	  -sales   data/sales_history.csv \
//	  -params  data/sku_parameters.csv \
//	  -output  output/report.csv
//
// The program loads CSV inputs, computes demand statistics, derives
// optimal inventory policies, runs Monte-Carlo simulations, and
// presents the results on the terminal and (optionally) in a CSV file.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/demand"
	"github.com/noble-ch/inventory-optimizer/internal/inventory"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/parser"
	"github.com/noble-ch/inventory-optimizer/internal/reporting"
	"github.com/noble-ch/inventory-optimizer/internal/simulation"
)

func main() {
	// ── CLI flags ──────────────────────────────────────────────────────
	salesPath := flag.String("sales", "data/sales_history.csv",
		"Path to weekly sales history CSV")
	paramsPath := flag.String("params", "data/sku_parameters.csv",
		"Path to SKU parameters CSV")
	outputPath := flag.String("output", "",
		"Path for CSV export (optional; leave blank to skip)")
	serviceLevel := flag.Float64("service-level", inventory.DefaultServiceLevel,
		"Target service level (0.90, 0.95, or 0.99)")
	simRuns := flag.Int("sim-runs", simulation.DefaultRuns,
		"Number of Monte-Carlo simulation runs per SKU")
	simWeeks := flag.Int("sim-weeks", simulation.DefaultWeeks,
		"Simulation horizon in weeks")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("inventory-optimizer v%s\n", models.Version)
		os.Exit(0)
	}

	start := time.Now()

	// ── Phase 1: Data Ingestion ───────────────────────────────────────
	log.Println("Loading sales history …")
	sales, errs := parser.LoadSalesHistory(*salesPath)
	reportParseErrors("sales history", errs)
	log.Printf("  → %d sales records loaded.\n", len(sales))

	log.Println("Loading SKU parameters …")
	paramSlice, errs := parser.LoadSKUParameters(*paramsPath)
	reportParseErrors("SKU parameters", errs)
	log.Printf("  → %d SKUs configured.\n", len(paramSlice))

	// Index parameters by SKU for O(1) lookup.
	paramsMap := make(map[string]models.SKUParameters, len(paramSlice))
	for _, p := range paramSlice {
		paramsMap[p.SKU] = p
	}

	// ── Phase 2: Demand Analysis ──────────────────────────────────────
	log.Println("Analysing demand …")
	demandStats, err := demand.Analyze(sales, paramsMap)
	if err != nil {
		log.Fatalf("Demand analysis failed: %v\n", err)
	}

	// Index demand stats by SKU.
	statsMap := make(map[string]models.DemandStats, len(demandStats))
	for _, d := range demandStats {
		statsMap[d.SKU] = d
	}

	// ── Phase 3: Inventory Optimisation ───────────────────────────────
	log.Println("Computing inventory policies …")
	policies := make(map[string]models.InventoryPolicy, len(paramSlice))
	for _, p := range paramSlice {
		stats, ok := statsMap[p.SKU]
		if !ok {
			log.Fatalf("No demand data for SKU %s\n", p.SKU)
		}
		pol, err := inventory.ComputePolicy(stats, p, *serviceLevel)
		if err != nil {
			log.Fatalf("Policy computation failed for %s: %v\n", p.SKU, err)
		}
		policies[p.SKU] = pol
	}

	// ── Phase 4: Monte-Carlo Simulation ───────────────────────────────
	log.Println("Running simulations …")
	simResults := make(map[string]models.SimulationResult, len(paramSlice))
	for _, p := range paramSlice {
		stats := statsMap[p.SKU]
		pol := policies[p.SKU]
		cfg := simulation.Config{
			Runs:  *simRuns,
			Weeks: *simWeeks,
			Seed:  time.Now().UnixNano(),
		}
		simResults[p.SKU] = simulation.Run(p, stats, pol, cfg)
	}

	// ── Phase 5: Reporting ────────────────────────────────────────────
	reports := assembleReports(paramSlice, statsMap, policies, simResults)

	reporting.PrintCLI(os.Stdout, reports)

	if *outputPath != "" {
		if err := reporting.ExportCSV(*outputPath, reports); err != nil {
			log.Fatalf("CSV export failed: %v\n", err)
		}
		log.Printf("CSV report saved to %s\n", *outputPath)
	}

	log.Printf("Done in %s.\n", time.Since(start).Round(time.Millisecond))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// assembleReports builds a sorted slice of SKUReport in the same order
// as the parameters file (preserves the user's SKU ordering).
func assembleReports(
	paramSlice []models.SKUParameters,
	stats map[string]models.DemandStats,
	policies map[string]models.InventoryPolicy,
	simResults map[string]models.SimulationResult,
) []models.SKUReport {
	reports := make([]models.SKUReport, 0, len(paramSlice))
	for _, p := range paramSlice {
		reports = append(reports, models.SKUReport{
			Parameters: p,
			Demand:     stats[p.SKU],
			Policy:     policies[p.SKU],
			Simulation: simResults[p.SKU],
		})
	}
	return reports
}

// reportParseErrors logs non-fatal parse warnings and aborts if the
// error count exceeds 10 (likely a bad file).
func reportParseErrors(label string, errs []error) {
	if len(errs) == 0 {
		return
	}
	for _, e := range errs {
		log.Printf("  ⚠  %s: %v\n", label, e)
	}
	if len(errs) > 10 {
		log.Fatalf("Too many errors in %s (%d) — aborting.\n", label, len(errs))
	}
}
