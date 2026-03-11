package demand

import (
"math"
"testing"

"github.com/noble-ch/inventory-optimizer/internal/models"
)

func makeRecords(sku string, units []int) []models.SalesRecord {
var recs []models.SalesRecord
for _, u := range units {
recs = append(recs, models.SalesRecord{SKU: sku, UnitsSold: u})
}
return recs
}

func makeParams(sku string, leadTimeDays int) map[string]models.SKUParameters {
return map[string]models.SKUParameters{
sku: {SKU: sku, LeadTimeDays: leadTimeDays, UnitCost: 10, OrderCost: 50, HoldingCostRate: 0.25},
}
}

func almostEqual(a, b, tol float64) bool {
return math.Abs(a-b) < tol
}

func TestAnalyze_BasicStats(t *testing.T) {
// 5 weeks of data: 10, 20, 30, 40, 50 → mean=30, stddev≈15.81
records := makeRecords("SKU001", []int{10, 20, 30, 40, 50})
params := makeParams("SKU001", 14)

stats, err := Analyze(records, params)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(stats) != 1 {
t.Fatalf("expected 1 stat, got %d", len(stats))
}

s := stats[0]
if !almostEqual(s.WeeklyMean, 30.0, 0.01) {
t.Errorf("WeeklyMean: got %.2f, want 30.00", s.WeeklyMean)
}
if !almostEqual(s.WeeklyStdDev, 15.8114, 0.01) {
t.Errorf("WeeklyStdDev: got %.4f, want ≈15.8114", s.WeeklyStdDev)
}
if !almostEqual(s.AnnualDemand, 30.0*52, 0.01) {
t.Errorf("AnnualDemand: got %.2f, want %.2f", s.AnnualDemand, 30.0*52)
}
if s.DataPointsCount != 5 {
t.Errorf("DataPointsCount: got %d, want 5", s.DataPointsCount)
}
}

func TestAnalyze_DailyConversion(t *testing.T) {
records := makeRecords("SKU001", []int{14, 14, 14, 14, 14})
params := makeParams("SKU001", 7)

stats, err := Analyze(records, params)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
s := stats[0]

// Weekly mean = 14, daily mean = 14/7 = 2.0
if !almostEqual(s.DailyMean, 2.0, 0.01) {
t.Errorf("DailyMean: got %.2f, want 2.00", s.DailyMean)
}
}

func TestAnalyze_LeadTimeDemand(t *testing.T) {
// Constant demand → stddev = 0, so LT stddev should also be 0
records := makeRecords("SKU001", []int{21, 21, 21})
params := makeParams("SKU001", 14)

stats, err := Analyze(records, params)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
s := stats[0]

expectedLTMean := (21.0 / 7.0) * 14.0 // daily_mean * lead_time_days = 3 * 14 = 42
if !almostEqual(s.LeadTimeMean, expectedLTMean, 0.01) {
t.Errorf("LeadTimeMean: got %.2f, want %.2f", s.LeadTimeMean, expectedLTMean)
}
}

func TestAnalyze_MissingSKUInParams(t *testing.T) {
records := makeRecords("SKU999", []int{10, 20, 30})
params := makeParams("SKU001", 14) // different SKU

_, err := Analyze(records, params)
if err == nil {
t.Fatal("expected error when SKU is missing from params")
}
}

func TestAnalyze_InsufficientDataPoints(t *testing.T) {
records := makeRecords("SKU001", []int{10}) // only 1 point
params := makeParams("SKU001", 14)

_, err := Analyze(records, params)
if err == nil {
t.Fatal("expected error for insufficient data points")
}
}

func TestMean(t *testing.T) {
tests := []struct {
vals []float64
want float64
}{
{[]float64{1, 2, 3, 4, 5}, 3.0},
{[]float64{10}, 10.0},
{[]float64{}, 0},
}
for _, tt := range tests {
got := mean(tt.vals)
if !almostEqual(got, tt.want, 0.001) {
t.Errorf("mean(%v) = %.3f, want %.3f", tt.vals, got, tt.want)
}
}
}

func TestStddev(t *testing.T) {
vals := []float64{2, 4, 4, 4, 5, 5, 7, 9}
mu := mean(vals)
got := stddev(vals, mu)
// sample stddev ≈ 2.0
if !almostEqual(got, 2.1381, 0.01) {
t.Errorf("stddev = %.4f, want ≈2.0", got)
}
}
