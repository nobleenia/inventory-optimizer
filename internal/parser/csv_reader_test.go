package parser

import (
"os"
"path/filepath"
"testing"
)

func TestLoadSalesHistory_ValidFile(t *testing.T) {
path := writeTempFile(t, "sales.csv", "sku,week,units_sold\nSKU001,2024-01-01,12\nSKU001,2024-01-08,15\nSKU002,2024-01-01,6\n")
records, errs := LoadSalesHistory(path)
if len(errs) != 0 {
t.Fatalf("unexpected errors: %v", errs)
}
if len(records) != 3 {
t.Fatalf("expected 3 records, got %d", len(records))
}
if records[0].SKU != "SKU001" || records[0].UnitsSold != 12 {
t.Errorf("first record mismatch: %+v", records[0])
}
if records[2].SKU != "SKU002" || records[2].UnitsSold != 6 {
t.Errorf("third record mismatch: %+v", records[2])
}
}

func TestLoadSalesHistory_MissingFile(t *testing.T) {
_, errs := LoadSalesHistory("/nonexistent/path.csv")
if len(errs) == 0 {
t.Fatal("expected an error for missing file")
}
}

func TestLoadSalesHistory_BadHeader(t *testing.T) {
path := writeTempFile(t, "bad_header.csv", "product,date,quantity\nSKU001,2024-01-01,12\n")
_, errs := LoadSalesHistory(path)
if len(errs) == 0 {
t.Fatal("expected error for wrong header columns")
}
}

func TestLoadSalesHistory_InvalidDate(t *testing.T) {
path := writeTempFile(t, "bad_date.csv", "sku,week,units_sold\nSKU001,01-01-2024,12\n")
records, errs := LoadSalesHistory(path)
if len(errs) == 0 {
t.Fatal("expected error for invalid date format")
}
if len(records) != 0 {
t.Errorf("expected 0 valid records, got %d", len(records))
}
}

func TestLoadSalesHistory_NegativeUnits(t *testing.T) {
path := writeTempFile(t, "neg.csv", "sku,week,units_sold\nSKU001,2024-01-01,-5\n")
records, errs := LoadSalesHistory(path)
if len(errs) == 0 {
t.Fatal("expected error for negative units")
}
if len(records) != 0 {
t.Errorf("expected 0 valid records, got %d", len(records))
}
}

func TestLoadSalesHistory_BlankSKU(t *testing.T) {
path := writeTempFile(t, "blank_sku.csv", "sku,week,units_sold\n,2024-01-01,5\n")
records, errs := LoadSalesHistory(path)
if len(errs) == 0 {
t.Fatal("expected error for blank SKU")
}
if len(records) != 0 {
t.Errorf("expected 0 valid records, got %d", len(records))
}
}

func TestLoadSalesHistory_ZeroUnitsAllowed(t *testing.T) {
path := writeTempFile(t, "zero.csv", "sku,week,units_sold\nSKU001,2024-01-01,0\n")
records, errs := LoadSalesHistory(path)
if len(errs) != 0 {
t.Fatalf("unexpected errors: %v", errs)
}
if len(records) != 1 || records[0].UnitsSold != 0 {
t.Errorf("zero units should be valid: %+v", records)
}
}

func TestLoadSKUParameters_ValidFile(t *testing.T) {
path := writeTempFile(t, "params.csv", "sku,current_inventory,lead_time_days,unit_cost,order_cost,holding_cost_rate\nSKU001,120,21,8.50,40.00,0.25\nSKU002,80,14,12.00,35.00,0.20\n")
params, errs := LoadSKUParameters(path)
if len(errs) != 0 {
t.Fatalf("unexpected errors: %v", errs)
}
if len(params) != 2 {
t.Fatalf("expected 2 params, got %d", len(params))
}
if params[0].SKU != "SKU001" || params[0].LeadTimeDays != 21 {
t.Errorf("first param mismatch: %+v", params[0])
}
if params[1].HoldingCostRate != 0.20 {
t.Errorf("second param holding rate: got %.2f, want 0.20", params[1].HoldingCostRate)
}
}

func TestLoadSKUParameters_InvalidLeadTime(t *testing.T) {
path := writeTempFile(t, "bad_lt.csv", "sku,current_inventory,lead_time_days,unit_cost,order_cost,holding_cost_rate\nSKU001,120,0,8.50,40.00,0.25\n")
_, errs := LoadSKUParameters(path)
if len(errs) == 0 {
t.Fatal("expected error for zero lead time")
}
}

func TestLoadSKUParameters_HoldingRateOutOfRange(t *testing.T) {
path := writeTempFile(t, "bad_rate.csv", "sku,current_inventory,lead_time_days,unit_cost,order_cost,holding_cost_rate\nSKU001,120,21,8.50,40.00,1.50\n")
_, errs := LoadSKUParameters(path)
if len(errs) == 0 {
t.Fatal("expected error for holding_cost_rate > 1")
}
}

func TestLoadSKUParameters_MissingColumns(t *testing.T) {
path := writeTempFile(t, "short.csv", "sku,current_inventory\nSKU001,120\n")
_, errs := LoadSKUParameters(path)
if len(errs) == 0 {
t.Fatal("expected error for missing columns in header")
}
}

func writeTempFile(t *testing.T, name, content string) string {
t.Helper()
dir := t.TempDir()
path := filepath.Join(dir, name)
if err := os.WriteFile(path, []byte(content), 0644); err != nil {
t.Fatalf("writing temp file: %v", err)
}
return path
}
