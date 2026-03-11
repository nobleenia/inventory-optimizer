package simulation

import (
"math"
"testing"

"github.com/noble-ch/inventory-optimizer/internal/models"
)

func almostEqual(a, b, tol float64) bool {
return math.Abs(a-b) < tol
}

func baseInputs() (models.SKUParameters, models.DemandStats, models.InventoryPolicy) {
params := models.SKUParameters{
SKU:              "SKU001",
CurrentInventory: 100,
LeadTimeDays:     14,
UnitCost:         10.0,
OrderCost:        50.0,
HoldingCostRate:  0.25,
}
stats := models.DemandStats{
SKU:        "SKU001",
DailyMean:  2.0,
DailyStdDev: 0.5,
}
policy := models.InventoryPolicy{
SKU:          "SKU001",
EOQ:          200.0,
SafetyStock:  16.5,
ReorderPoint: 44.5,
ServiceLevel: 0.95,
}
return params, stats, policy
}

func TestRun_ReturnsCorrectMetadata(t *testing.T) {
params, stats, policy := baseInputs()
cfg := Config{Runs: 100, Weeks: 52, Seed: 42}

result := Run(params, stats, policy, cfg)

if result.SKU != "SKU001" {
t.Errorf("SKU = %q, want SKU001", result.SKU)
}
if result.Runs != 100 {
t.Errorf("Runs = %d, want 100", result.Runs)
}
if result.WeeksPerRun != 52 {
t.Errorf("WeeksPerRun = %d, want 52", result.WeeksPerRun)
}
}

func TestRun_Deterministic(t *testing.T) {
params, stats, policy := baseInputs()
cfg := Config{Runs: 50, Weeks: 26, Seed: 123}

r1 := Run(params, stats, policy, cfg)
r2 := Run(params, stats, policy, cfg)

if r1.AvgStockouts != r2.AvgStockouts {
t.Errorf("non-deterministic: stockouts %.2f vs %.2f", r1.AvgStockouts, r2.AvgStockouts)
}
if r1.AvgTotalAnnualCost != r2.AvgTotalAnnualCost {
t.Errorf("non-deterministic: cost %.2f vs %.2f", r1.AvgTotalAnnualCost, r2.AvgTotalAnnualCost)
}
}

func TestRun_CostBreakdown(t *testing.T) {
params, stats, policy := baseInputs()
cfg := Config{Runs: 200, Weeks: 52, Seed: 99}

result := Run(params, stats, policy, cfg)

// Total should equal holding + ordering
sum := result.AvgAnnualHoldingCost + result.AvgAnnualOrderCost
if !almostEqual(result.AvgTotalAnnualCost, sum, 0.01) {
t.Errorf("total %.2f != holding %.2f + ordering %.2f",
result.AvgTotalAnnualCost, result.AvgAnnualHoldingCost, result.AvgAnnualOrderCost)
}
}

func TestRun_StockoutProbabilityRange(t *testing.T) {
params, stats, policy := baseInputs()
cfg := Config{Runs: 200, Weeks: 52, Seed: 7}

result := Run(params, stats, policy, cfg)

if result.StockoutProbability < 0 || result.StockoutProbability > 1 {
t.Errorf("stockout probability %.4f out of [0,1] range", result.StockoutProbability)
}
}

func TestRun_HighInventoryNoStockouts(t *testing.T) {
params, stats, policy := baseInputs()
params.CurrentInventory = 10000 // massive starting inventory
policy.ReorderPoint = 5000
policy.EOQ = 5000
cfg := Config{Runs: 100, Weeks: 52, Seed: 1}

result := Run(params, stats, policy, cfg)

if result.AvgStockouts > 0.1 {
t.Errorf("with huge inventory, expected near-zero stockouts, got %.2f", result.AvgStockouts)
}
}

func TestRun_DefaultsApplied(t *testing.T) {
params, stats, policy := baseInputs()
cfg := Config{Seed: 42} // Runs and Weeks left at 0

result := Run(params, stats, policy, cfg)

if result.Runs != DefaultRuns {
t.Errorf("Runs = %d, want default %d", result.Runs, DefaultRuns)
}
if result.WeeksPerRun != DefaultWeeks {
t.Errorf("WeeksPerRun = %d, want default %d", result.WeeksPerRun, DefaultWeeks)
}
}
