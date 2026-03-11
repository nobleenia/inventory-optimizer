// Package engine provides a high-level API that orchestrates the full
// analysis pipeline (parse → demand → policy → simulation → report).
//
// Both the CLI (cmd/main.go) and the web server call into this package,
// ensuring the core logic lives in one place. The engine never touches
// HTTP, templates, or stdout — it only returns data.
package engine

import (
	"fmt"
	"io"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/demand"
	"github.com/noble-ch/inventory-optimizer/internal/inventory"
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/parser"
	"github.com/noble-ch/inventory-optimizer/internal/simulation"
)

// Options configures an engine run.
type Options struct {
	ServiceLevel float64
	SimRuns      int
	SimWeeks     int
}

// DefaultOptions returns production defaults.
func DefaultOptions() Options {
	return Options{
		ServiceLevel: inventory.DefaultServiceLevel,
		SimRuns:      simulation.DefaultRuns,
		SimWeeks:     simulation.DefaultWeeks,
	}
}

// RunFromFiles orchestrates the full pipeline using file paths.
func RunFromFiles(salesPath, paramsPath string, opts Options) ([]models.SKUReport, []string, error) {
	sales, errs := parser.LoadSalesHistory(salesPath)
	warnings := errStrings(errs)
	if len(sales) == 0 {
		return nil, warnings, fmt.Errorf("no valid sales records loaded")
	}

	paramSlice, errs := parser.LoadSKUParameters(paramsPath)
	warnings = append(warnings, errStrings(errs)...)
	if len(paramSlice) == 0 {
		return nil, warnings, fmt.Errorf("no valid SKU parameters loaded")
	}

	paramsMap := indexParams(paramSlice)
	return runPipeline(sales, paramSlice, paramsMap, opts, warnings)
}

// RunFromReaders orchestrates the pipeline from io.Reader sources
// (used by the web upload handler — no files on disk required).
func RunFromReaders(salesReader, paramsReader io.Reader, opts Options) ([]models.SKUReport, []string, error) {
	sales, errs := parser.LoadSalesHistoryFromReader(salesReader)
	warnings := errStrings(errs)
	if len(sales) == 0 {
		return nil, warnings, fmt.Errorf("no valid sales records loaded")
	}

	paramSlice, errs := parser.LoadSKUParametersFromReader(paramsReader)
	warnings = append(warnings, errStrings(errs)...)
	if len(paramSlice) == 0 {
		return nil, warnings, fmt.Errorf("no valid SKU parameters loaded")
	}

	paramsMap := indexParams(paramSlice)
	return runPipeline(sales, paramSlice, paramsMap, opts, warnings)
}

// ---------------------------------------------------------------------------
// Internal pipeline
// ---------------------------------------------------------------------------

func runPipeline(
	sales []models.SalesRecord,
	paramSlice []models.SKUParameters,
	paramsMap map[string]models.SKUParameters,
	opts Options,
	warnings []string,
) ([]models.SKUReport, []string, error) {

	demandStats, err := demand.Analyze(sales, paramsMap)
	if err != nil {
		return nil, warnings, fmt.Errorf("demand analysis: %w", err)
	}

	statsMap := make(map[string]models.DemandStats, len(demandStats))
	for _, d := range demandStats {
		statsMap[d.SKU] = d
	}

	policies := make(map[string]models.InventoryPolicy, len(paramSlice))
	for _, p := range paramSlice {
		stats, ok := statsMap[p.SKU]
		if !ok {
			return nil, warnings, fmt.Errorf("no demand data for SKU %s", p.SKU)
		}
		pol, err := inventory.ComputePolicy(stats, p, opts.ServiceLevel)
		if err != nil {
			return nil, warnings, fmt.Errorf("policy for %s: %w", p.SKU, err)
		}
		policies[p.SKU] = pol
	}

	simResults := make(map[string]models.SimulationResult, len(paramSlice))
	for _, p := range paramSlice {
		cfg := simulation.Config{
			Runs:  opts.SimRuns,
			Weeks: opts.SimWeeks,
			Seed:  time.Now().UnixNano(),
		}
		simResults[p.SKU] = simulation.Run(p, statsMap[p.SKU], policies[p.SKU], cfg)
	}

	reports := make([]models.SKUReport, 0, len(paramSlice))
	for _, p := range paramSlice {
		reports = append(reports, models.SKUReport{
			Parameters: p,
			Demand:     statsMap[p.SKU],
			Policy:     policies[p.SKU],
			Simulation: simResults[p.SKU],
		})
	}

	return reports, warnings, nil
}

func indexParams(slice []models.SKUParameters) map[string]models.SKUParameters {
	m := make(map[string]models.SKUParameters, len(slice))
	for _, p := range slice {
		m[p.SKU] = p
	}
	return m
}

func errStrings(errs []error) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, e.Error())
	}
	return out
}
