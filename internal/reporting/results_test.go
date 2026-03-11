package reporting

import (
"bytes"
"os"
"path/filepath"
"strings"
"testing"

"github.com/noble-ch/inventory-optimizer/internal/models"
)

func sampleReports() []models.SKUReport {
return []models.SKUReport{
{
Parameters: models.SKUParameters{
SKU: "SKU001", CurrentInventory: 120, LeadTimeDays: 21,
UnitCost: 8.5, OrderCost: 40, HoldingCostRate: 0.25,
},
Demand: models.DemandStats{
SKU: "SKU001", WeeklyMean: 17.6, WeeklyStdDev: 5.9,
AnnualDemand: 913, DataPointsCount: 52,
},
Policy: models.InventoryPolicy{
SKU: "SKU001", EOQ: 186, SafetyStock: 17,
ReorderPoint: 70, ServiceLevel: 0.95,
},
Simulation: models.SimulationResult{
SKU: "SKU001", Runs: 500, WeeksPerRun: 52,
AvgStockouts: 0.1, StockoutProbability: 0.001,
AvgInventoryLevel: 108, AvgAnnualHoldingCost: 230.18,
AvgAnnualOrderCost: 205.20, AvgTotalAnnualCost: 435.38,
},
},
}
}

func TestPrintCLI_ContainsKeyInfo(t *testing.T) {
var buf bytes.Buffer
PrintCLI(&buf, sampleReports())
output := buf.String()

checks := []string{
"SKU001",
"INVENTORY OPTIMIZATION REPORT",
"Reorder point",
"Order quantity",
"Safety stock",
"Estimated Annual Costs",
"End of report",
}
for _, c := range checks {
if !strings.Contains(output, c) {
t.Errorf("CLI output missing %q", c)
}
}
}

func TestPrintCLI_MultiSKU(t *testing.T) {
reports := sampleReports()
second := reports[0]
second.Parameters.SKU = "SKU002"
reports = append(reports, second)

var buf bytes.Buffer
PrintCLI(&buf, reports)
output := buf.String()

if !strings.Contains(output, "SKU001") || !strings.Contains(output, "SKU002") {
t.Error("multi-SKU report should contain both SKU identifiers")
}
if !strings.Contains(output, "SKUs analysed: 2") {
t.Error("should show correct SKU count")
}
}

func TestExportCSV_CreatesFile(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "out.csv")

err := ExportCSV(path, sampleReports())
if err != nil {
t.Fatalf("ExportCSV error: %v", err)
}

data, err := os.ReadFile(path)
if err != nil {
t.Fatalf("reading output: %v", err)
}

content := string(data)
if !strings.Contains(content, "sku,") {
t.Error("CSV should have header")
}
if !strings.Contains(content, "SKU001") {
t.Error("CSV should contain SKU data")
}
}

func TestExportCSV_BadPath(t *testing.T) {
err := ExportCSV("/nonexistent/dir/file.csv", sampleReports())
if err == nil {
t.Fatal("expected error for bad path")
}
}

func TestCurrency(t *testing.T) {
got := currency(1234.56)
want := "€1234.56"
if got != want {
t.Errorf("currency = %q, want %q", got, want)
}
}
