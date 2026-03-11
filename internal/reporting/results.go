// Package reporting renders SKU analysis results for human consumption.
//
// It provides two output modes:
//   - CLI: a formatted table printed to stdout.
//   - CSV: a machine-readable file that can be opened in any spreadsheet app.
//
// The package never performs calculations; it only presents data that
// has already been computed by other packages.
package reporting

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ---------------------------------------------------------------------------
// CLI output
// ---------------------------------------------------------------------------

// PrintCLI writes a human-friendly summary to w (typically os.Stdout).
func PrintCLI(w io.Writer, reports []models.SKUReport) {
	divider := strings.Repeat("─", 72)

	fmt.Fprintf(w, "\n%s\n", divider)
	fmt.Fprintf(w, "  INVENTORY OPTIMIZATION REPORT  (v%s)\n", models.Version)
	fmt.Fprintf(w, "%s\n\n", divider)
	fmt.Fprintf(w, "  SKUs analysed: %d\n\n", len(reports))

	for i, r := range reports {
		if i > 0 {
			fmt.Fprintf(w, "\n%s\n", divider)
		}
		printSKU(w, r)
	}

	fmt.Fprintf(w, "\n%s\n", divider)
	fmt.Fprintf(w, "  End of report.\n")
	fmt.Fprintf(w, "%s\n\n", divider)
}

func printSKU(w io.Writer, r models.SKUReport) {
	fmt.Fprintf(w, "  SKU: %s\n", r.Parameters.SKU)
	fmt.Fprintf(w, "  %-36s %10d units\n", "Current inventory on hand:",
		r.Parameters.CurrentInventory)
	fmt.Fprintf(w, "  %-36s %10d days\n", "Supplier lead time:",
		r.Parameters.LeadTimeDays)
	fmt.Fprintln(w)

	// Demand
	fmt.Fprintf(w, "  ── Demand Analysis ──\n")
	fmt.Fprintf(w, "  %-36s %10.1f units/week\n", "Average weekly demand:",
		r.Demand.WeeklyMean)
	fmt.Fprintf(w, "  %-36s %10.1f units/week\n", "Weekly demand variability (σ):",
		r.Demand.WeeklyStdDev)
	fmt.Fprintf(w, "  %-36s %10.0f units/year\n", "Estimated annual demand:",
		r.Demand.AnnualDemand)
	fmt.Fprintf(w, "  %-36s %10d weeks\n", "Data points used:",
		r.Demand.DataPointsCount)
	fmt.Fprintln(w)

	// Policy
	fmt.Fprintf(w, "  ── Recommended Policy ──\n")
	fmt.Fprintf(w, "  %-36s %10.0f units\n", "▸ Reorder point (ROP):",
		math.Ceil(r.Policy.ReorderPoint))
	fmt.Fprintf(w, "  %-36s %10.0f units\n", "▸ Order quantity (EOQ):",
		math.Ceil(r.Policy.EOQ))
	fmt.Fprintf(w, "  %-36s %10.0f units\n", "▸ Safety stock:",
		math.Ceil(r.Policy.SafetyStock))
	fmt.Fprintf(w, "  %-36s %9.0f%%\n", "  Target service level:",
		r.Policy.ServiceLevel*100)
	fmt.Fprintln(w)

	// Simulation
	fmt.Fprintf(w, "  ── Simulation Results (%d runs × %d weeks) ──\n",
		r.Simulation.Runs, r.Simulation.WeeksPerRun)
	fmt.Fprintf(w, "  %-36s %10.1f events/year\n", "Expected stockouts:",
		r.Simulation.AvgStockouts)
	fmt.Fprintf(w, "  %-36s %9.1f%%\n", "Stockout probability:",
		r.Simulation.StockoutProbability*100)
	fmt.Fprintf(w, "  %-36s %10.0f units\n", "Average inventory level:",
		r.Simulation.AvgInventoryLevel)
	fmt.Fprintln(w)

	// Costs
	fmt.Fprintf(w, "  ── Estimated Annual Costs ──\n")
	fmt.Fprintf(w, "  %-36s %10s\n", "Holding cost:",
		currency(r.Simulation.AvgAnnualHoldingCost))
	fmt.Fprintf(w, "  %-36s %10s\n", "Ordering cost:",
		currency(r.Simulation.AvgAnnualOrderCost))
	fmt.Fprintf(w, "  %-36s %10s\n", "Total annual cost:",
		currency(r.Simulation.AvgTotalAnnualCost))
}

// currency formats a float as a Euro string.
func currency(v float64) string {
	return fmt.Sprintf("€%.2f", v)
}

// ---------------------------------------------------------------------------
// CSV export
// ---------------------------------------------------------------------------

// ExportCSV writes all SKU reports to a CSV file at the given path.
func ExportCSV(path string, reports []models.SKUReport) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"sku",
		"current_inventory",
		"lead_time_days",
		"weekly_mean_demand",
		"weekly_std_demand",
		"annual_demand",
		"reorder_point",
		"order_quantity_eoq",
		"safety_stock",
		"service_level",
		"expected_stockouts",
		"stockout_probability",
		"avg_inventory_level",
		"annual_holding_cost",
		"annual_ordering_cost",
		"total_annual_cost",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for _, r := range reports {
		row := []string{
			r.Parameters.SKU,
			fmt.Sprintf("%d", r.Parameters.CurrentInventory),
			fmt.Sprintf("%d", r.Parameters.LeadTimeDays),
			fmt.Sprintf("%.2f", r.Demand.WeeklyMean),
			fmt.Sprintf("%.2f", r.Demand.WeeklyStdDev),
			fmt.Sprintf("%.0f", r.Demand.AnnualDemand),
			fmt.Sprintf("%.0f", math.Ceil(r.Policy.ReorderPoint)),
			fmt.Sprintf("%.0f", math.Ceil(r.Policy.EOQ)),
			fmt.Sprintf("%.0f", math.Ceil(r.Policy.SafetyStock)),
			fmt.Sprintf("%.0f%%", r.Policy.ServiceLevel*100),
			fmt.Sprintf("%.1f", r.Simulation.AvgStockouts),
			fmt.Sprintf("%.1f%%", r.Simulation.StockoutProbability*100),
			fmt.Sprintf("%.0f", r.Simulation.AvgInventoryLevel),
			fmt.Sprintf("%.2f", r.Simulation.AvgAnnualHoldingCost),
			fmt.Sprintf("%.2f", r.Simulation.AvgAnnualOrderCost),
			fmt.Sprintf("%.2f", r.Simulation.AvgTotalAnnualCost),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("writing CSV row for %s: %w", r.Parameters.SKU, err)
		}
	}

	return nil
}
