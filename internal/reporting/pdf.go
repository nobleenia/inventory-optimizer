// Package reporting renders SKU analysis results for human consumption.
//
// pdf.go generates a branded PDF report containing per-SKU tables,
// demand statistics, policy recommendations, simulation results, and
// forecast insights. It uses go-pdf/fpdf (pure Go, no CGo).
package reporting

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// WritePDF renders all SKU reports as a multi-page PDF to dest.
func WritePDF(dest io.Writer, reports []models.SKUReport) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	// ── Cover page ─────────────────────────────────────────────────
	pdf.AddPage()
	renderCoverPage(pdf, reports)

	// ── Per-SKU pages ──────────────────────────────────────────────
	for _, r := range reports {
		pdf.AddPage()
		renderSKUPage(pdf, r)
	}

	return pdf.Output(dest)
}

// ExportPDF writes the PDF to a file at the given path.
func ExportPDF(path string, reports []models.SKUReport) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)

	pdf.AddPage()
	renderCoverPage(pdf, reports)

	for _, r := range reports {
		pdf.AddPage()
		renderSKUPage(pdf, r)
	}

	return pdf.OutputFileAndClose(path)
}

// ---------------------------------------------------------------------------
// Cover page
// ---------------------------------------------------------------------------

func renderCoverPage(pdf *fpdf.Fpdf, reports []models.SKUReport) {
	w, _ := pdf.GetPageSize()
	skuCount := len(reports)

	// Blue header band
	pdf.SetFillColor(37, 99, 235) // --color-primary
	pdf.Rect(0, 0, w, 60, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 28)
	pdf.SetXY(20, 18)
	pdf.CellFormat(w-40, 12, "Inventory Optimizer", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 13)
	pdf.SetXY(20, 35)
	pdf.CellFormat(w-40, 8, "Inventory Analysis Report", "", 1, "L", false, 0, "")

	// Reset text color
	pdf.SetTextColor(30, 41, 59) // --color-text

	pdf.SetY(75)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetX(20)
	pdf.CellFormat(0, 7, fmt.Sprintf("SKUs analysed: %d", skuCount), "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.CellFormat(0, 7, fmt.Sprintf("Engine version: %s", models.Version), "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.CellFormat(0, 7, "Service level: 95%", "", 1, "L", false, 0, "")
	pdf.SetX(20)
	pdf.CellFormat(0, 7, "Simulation: 500 runs x 52 weeks per SKU", "", 1, "L", false, 0, "")

	// Summary table
	pdf.Ln(10)
	pdf.SetX(20)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, "Summary", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	// Table header
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(241, 245, 249) // light gray
	colW := []float64{30, 30, 25, 28, 28, 28}
	headers := []string{"SKU", "ROP", "EOQ", "Safety", "Stockouts", "Annual Cost"}
	pdf.SetX(20)
	for i, h := range headers {
		pdf.CellFormat(colW[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table body
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetFillColor(255, 255, 255)
	for _, r := range reports {
		pdf.SetX(20)
		pdf.CellFormat(colW[0], 6, r.Parameters.SKU, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[1], 6, fmt.Sprintf("%.0f", math.Ceil(r.Policy.ReorderPoint)), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[2], 6, fmt.Sprintf("%.0f", math.Ceil(r.Policy.EOQ)), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], 6, fmt.Sprintf("%.0f", math.Ceil(r.Policy.SafetyStock)), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[4], 6, fmt.Sprintf("%.1f/yr", r.Simulation.AvgStockouts), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[5], 6, fmt.Sprintf("%.2f", r.Simulation.AvgTotalAnnualCost), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}
}

// ---------------------------------------------------------------------------
// Per-SKU page
// ---------------------------------------------------------------------------

func renderSKUPage(pdf *fpdf.Fpdf, r models.SKUReport) {
	w, _ := pdf.GetPageSize()

	// Blue header bar
	pdf.SetFillColor(37, 99, 235)
	pdf.Rect(0, pdf.GetY()-10, w, 14, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetX(20)
	pdf.CellFormat(0, 10, r.Parameters.SKU, "", 1, "L", false, 0, "")
	pdf.SetTextColor(30, 41, 59)
	pdf.Ln(4)

	// Badges line
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetX(20)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(0, 5,
		fmt.Sprintf("%d units on hand  |  %d-day lead time  |  Trend: %s  |  Variability: %s",
			r.Parameters.CurrentInventory, r.Parameters.LeadTimeDays,
			r.Forecast.TrendLabel, r.Forecast.SeasonalityFlag),
		"", 1, "L", false, 0, "")
	pdf.SetTextColor(30, 41, 59)
	pdf.Ln(4)

	// ── Recommended Policy ─────────────────────────────────────────
	sectionHeading(pdf, "Recommended Policy")
	kvTable(pdf, []kv{
		{"Reorder Point (ROP)", fmt.Sprintf("%.0f units", math.Ceil(r.Policy.ReorderPoint))},
		{"Order Quantity (EOQ)", fmt.Sprintf("%.0f units", math.Ceil(r.Policy.EOQ))},
		{"Safety Stock", fmt.Sprintf("%.0f units", math.Ceil(r.Policy.SafetyStock))},
		{"Service Level", fmt.Sprintf("%.0f%%", r.Policy.ServiceLevel*100)},
	})

	// ── Demand Analysis ────────────────────────────────────────────
	sectionHeading(pdf, "Demand Analysis")
	kvTable(pdf, []kv{
		{"Weekly Mean Demand", fmt.Sprintf("%.1f units/wk", r.Demand.WeeklyMean)},
		{"Weekly Std Dev", fmt.Sprintf("%.1f units/wk", r.Demand.WeeklyStdDev)},
		{"Annual Demand", fmt.Sprintf("%.0f units/yr", r.Demand.AnnualDemand)},
		{"Data Points", fmt.Sprintf("%d weeks", r.Demand.DataPointsCount)},
	})

	// ── Simulation Results ─────────────────────────────────────────
	sectionHeading(pdf, "Simulation Results")
	kvTable(pdf, []kv{
		{"Expected Stockouts", fmt.Sprintf("%.1f events/year", r.Simulation.AvgStockouts)},
		{"Stockout Probability", fmt.Sprintf("%.1f%%", r.Simulation.StockoutProbability*100)},
		{"Average Inventory", fmt.Sprintf("%.0f units", r.Simulation.AvgInventoryLevel)},
	})

	// ── Costs ──────────────────────────────────────────────────────
	sectionHeading(pdf, "Estimated Annual Costs")
	kvTable(pdf, []kv{
		{"Holding Cost", fmt.Sprintf("EUR %.2f", r.Simulation.AvgAnnualHoldingCost)},
		{"Ordering Cost", fmt.Sprintf("EUR %.2f", r.Simulation.AvgAnnualOrderCost)},
		{"Total Annual Cost", fmt.Sprintf("EUR %.2f", r.Simulation.AvgTotalAnnualCost)},
	})

	// ── Recommendation ─────────────────────────────────────────────
	pdf.Ln(4)
	pdf.SetFillColor(239, 246, 255) // light blue
	pdf.SetX(20)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(170, 7, "Recommendation", "", 1, "L", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetX(20)
	rec := fmt.Sprintf(
		"Order %d units when stock drops to %d. Maintain %d units safety stock. "+
			"Expected stockouts: %.1f/year. Estimated annual cost: EUR %.2f. "+
			"Demand trend is %s; variability is %s.",
		int(math.Ceil(r.Policy.EOQ)),
		int(math.Ceil(r.Policy.ReorderPoint)),
		int(math.Ceil(r.Policy.SafetyStock)),
		r.Simulation.AvgStockouts,
		r.Simulation.AvgTotalAnnualCost,
		r.Forecast.TrendLabel,
		r.Forecast.SeasonalityFlag,
	)
	pdf.MultiCell(170, 5, rec, "", "L", false)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type kv struct {
	key, val string
}

func sectionHeading(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetX(20)
	pdf.SetTextColor(37, 99, 235)
	pdf.CellFormat(0, 7, strings.ToUpper(title), "", 1, "L", false, 0, "")
	pdf.SetTextColor(30, 41, 59)
}

func kvTable(pdf *fpdf.Fpdf, rows []kv) {
	pdf.SetFont("Helvetica", "", 9)
	for i, r := range rows {
		fill := i%2 == 0
		if fill {
			pdf.SetFillColor(248, 250, 252)
		}
		pdf.SetX(20)
		pdf.CellFormat(80, 6, r.key, "", 0, "L", fill, 0, "")
		pdf.CellFormat(90, 6, r.val, "", 1, "L", fill, 0, "")
	}
	pdf.Ln(3)
}
