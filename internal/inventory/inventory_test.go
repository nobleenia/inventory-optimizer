package inventory

import (
"math"
"testing"

"github.com/noble-ch/inventory-optimizer/internal/models"
)

func almostEqual(a, b, tol float64) bool {
return math.Abs(a-b) < tol
}

// ---------------------------------------------------------------------------
// Safety Stock
// ---------------------------------------------------------------------------

func TestComputeSafetyStock_95(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", LeadTimeStdDev: 10.0}
ss, err := ComputeSafetyStock(stats, 0.95)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
// Z(95%) = 1.65 → 1.65 * 10 = 16.5
if !almostEqual(ss, 16.5, 0.01) {
t.Errorf("safety stock = %.2f, want 16.50", ss)
}
}

func TestComputeSafetyStock_90(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", LeadTimeStdDev: 10.0}
ss, err := ComputeSafetyStock(stats, 0.90)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if !almostEqual(ss, 12.8, 0.01) {
t.Errorf("safety stock = %.2f, want 12.80", ss)
}
}

func TestComputeSafetyStock_99(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", LeadTimeStdDev: 10.0}
ss, err := ComputeSafetyStock(stats, 0.99)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if !almostEqual(ss, 23.3, 0.01) {
t.Errorf("safety stock = %.2f, want 23.30", ss)
}
}

func TestComputeSafetyStock_UnsupportedLevel(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", LeadTimeStdDev: 10.0}
_, err := ComputeSafetyStock(stats, 0.85)
if err == nil {
t.Fatal("expected error for unsupported service level")
}
}

// ---------------------------------------------------------------------------
// Reorder Point
// ---------------------------------------------------------------------------

func TestComputeReorderPoint(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", LeadTimeMean: 42.0}
rop := ComputeReorderPoint(stats, 16.5)
if !almostEqual(rop, 58.5, 0.01) {
t.Errorf("ROP = %.2f, want 58.50", rop)
}
}

// ---------------------------------------------------------------------------
// EOQ
// ---------------------------------------------------------------------------

func TestComputeEOQ(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", AnnualDemand: 1000}
params := models.SKUParameters{SKU: "SKU001", OrderCost: 50, UnitCost: 10, HoldingCostRate: 0.25}

eoq, err := ComputeEOQ(stats, params)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
// EOQ = sqrt(2*1000*50 / (10*0.25)) = sqrt(100000/2.5) = sqrt(40000) = 200
if !almostEqual(eoq, 200.0, 0.01) {
t.Errorf("EOQ = %.2f, want 200.00", eoq)
}
}

func TestComputeEOQ_ZeroDemand(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", AnnualDemand: 0}
params := models.SKUParameters{SKU: "SKU001", OrderCost: 50, UnitCost: 10, HoldingCostRate: 0.25}

_, err := ComputeEOQ(stats, params)
if err == nil {
t.Fatal("expected error for zero demand")
}
}

func TestComputeEOQ_ZeroOrderCost(t *testing.T) {
stats := models.DemandStats{SKU: "SKU001", AnnualDemand: 1000}
params := models.SKUParameters{SKU: "SKU001", OrderCost: 0, UnitCost: 10, HoldingCostRate: 0.25}

_, err := ComputeEOQ(stats, params)
if err == nil {
t.Fatal("expected error for zero order cost")
}
}

// ---------------------------------------------------------------------------
// Unified Policy
// ---------------------------------------------------------------------------

func TestComputePolicy(t *testing.T) {
stats := models.DemandStats{
SKU:            "SKU001",
AnnualDemand:   1000,
LeadTimeMean:   42.0,
LeadTimeStdDev: 10.0,
}
params := models.SKUParameters{
SKU: "SKU001", OrderCost: 50, UnitCost: 10, HoldingCostRate: 0.25,
}

pol, err := ComputePolicy(stats, params, 0.95)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}

if pol.SKU != "SKU001" {
t.Errorf("SKU = %q, want SKU001", pol.SKU)
}
if !almostEqual(pol.SafetyStock, 16.5, 0.01) {
t.Errorf("SafetyStock = %.2f, want 16.50", pol.SafetyStock)
}
if !almostEqual(pol.ReorderPoint, 58.5, 0.01) {
t.Errorf("ReorderPoint = %.2f, want 58.50", pol.ReorderPoint)
}
if !almostEqual(pol.EOQ, 200.0, 0.01) {
t.Errorf("EOQ = %.2f, want 200.00", pol.EOQ)
}
if pol.ServiceLevel != 0.95 {
t.Errorf("ServiceLevel = %.2f, want 0.95", pol.ServiceLevel)
}
}
