// Package parser handles the ingestion and validation of CSV input files.
// It converts raw CSV rows into strongly-typed models that the rest of the
// engine can rely on without additional nil-checks or format concerns.
//
// Two file formats are supported:
//   - Sales history   (sku, week, units_sold)
//   - SKU parameters  (sku, current_inventory, lead_time_days, unit_cost,
//     order_cost, holding_cost_rate)
package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// LoadSalesHistory reads and validates a sales history CSV file at the
// given path. Rows that fail validation are collected into the returned
// error slice so the caller can decide whether to continue.
func LoadSalesHistory(path string) ([]models.SalesRecord, []error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []error{fmt.Errorf("cannot open sales history file: %w", err)}
	}
	defer f.Close()
	return LoadSalesHistoryFromReader(f)
}

// LoadSalesHistoryFromReader parses sales history CSV data from any
// io.Reader (file, HTTP upload body, etc.).
func LoadSalesHistoryFromReader(r io.Reader) ([]models.SalesRecord, []error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("cannot read header row: %w", err)}
	}
	if err := validateHeader(header, []string{"sku", "week", "units_sold"}); err != nil {
		return nil, []error{err}
	}

	var (
		records []models.SalesRecord
		errs    []error
		row     int = 1
	)

	for {
		row++
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("row %d: %w", row, err))
			continue
		}

		rec, parseErr := parseSalesRow(line, row)
		if parseErr != nil {
			errs = append(errs, parseErr)
			continue
		}
		records = append(records, rec)
	}

	return records, errs
}

// LoadSKUParameters reads and validates a SKU parameters CSV file.
func LoadSKUParameters(path string) ([]models.SKUParameters, []error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []error{fmt.Errorf("cannot open SKU parameters file: %w", err)}
	}
	defer f.Close()
	return LoadSKUParametersFromReader(f)
}

// LoadSKUParametersFromReader parses SKU parameters CSV data from any
// io.Reader.
func LoadSKUParametersFromReader(r io.Reader) ([]models.SKUParameters, []error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("cannot read header row: %w", err)}
	}
	expectedCols := []string{
		"sku", "current_inventory", "lead_time_days",
		"unit_cost", "order_cost", "holding_cost_rate",
	}
	if err := validateHeader(header, expectedCols); err != nil {
		return nil, []error{err}
	}

	var (
		params []models.SKUParameters
		errs   []error
		row    int = 1
	)

	for {
		row++
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("row %d: %w", row, err))
			continue
		}

		p, parseErr := parseParamsRow(line, row)
		if parseErr != nil {
			errs = append(errs, parseErr)
			continue
		}
		params = append(params, p)
	}

	return params, errs
}

// ---------------------------------------------------------------------------
// Row-level parsing
// ---------------------------------------------------------------------------

func parseSalesRow(fields []string, row int) (models.SalesRecord, error) {
	if len(fields) < 3 {
		return models.SalesRecord{}, fmt.Errorf("row %d: expected 3 columns, got %d", row, len(fields))
	}

	sku := strings.TrimSpace(fields[0])
	if sku == "" {
		return models.SalesRecord{}, fmt.Errorf("row %d: SKU is blank", row)
	}

	week, err := time.Parse("2006-01-02", strings.TrimSpace(fields[1]))
	if err != nil {
		return models.SalesRecord{}, fmt.Errorf("row %d: invalid date %q (expected YYYY-MM-DD): %w", row, fields[1], err)
	}

	units, err := strconv.Atoi(strings.TrimSpace(fields[2]))
	if err != nil {
		return models.SalesRecord{}, fmt.Errorf("row %d: units_sold %q is not an integer: %w", row, fields[2], err)
	}
	if units < 0 {
		return models.SalesRecord{}, fmt.Errorf("row %d: units_sold cannot be negative (%d)", row, units)
	}

	return models.SalesRecord{SKU: sku, Week: week, UnitsSold: units}, nil
}

func parseParamsRow(fields []string, row int) (models.SKUParameters, error) {
	if len(fields) < 6 {
		return models.SKUParameters{}, fmt.Errorf("row %d: expected 6 columns, got %d", row, len(fields))
	}

	sku := strings.TrimSpace(fields[0])
	if sku == "" {
		return models.SKUParameters{}, fmt.Errorf("row %d: SKU is blank", row)
	}

	currentInv, err := strconv.Atoi(strings.TrimSpace(fields[1]))
	if err != nil {
		return models.SKUParameters{}, fmt.Errorf("row %d: current_inventory %q is not an integer: %w", row, fields[1], err)
	}

	leadTime, err := strconv.Atoi(strings.TrimSpace(fields[2]))
	if err != nil {
		return models.SKUParameters{}, fmt.Errorf("row %d: lead_time_days %q is not an integer: %w", row, fields[2], err)
	}
	if leadTime <= 0 {
		return models.SKUParameters{}, fmt.Errorf("row %d: lead_time_days must be positive (%d)", row, leadTime)
	}

	unitCost, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
	if err != nil {
		return models.SKUParameters{}, fmt.Errorf("row %d: unit_cost %q is not a number: %w", row, fields[3], err)
	}

	orderCost, err := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
	if err != nil {
		return models.SKUParameters{}, fmt.Errorf("row %d: order_cost %q is not a number: %w", row, fields[4], err)
	}

	holdingRate, err := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
	if err != nil {
		return models.SKUParameters{}, fmt.Errorf("row %d: holding_cost_rate %q is not a number: %w", row, fields[5], err)
	}
	if holdingRate < 0 || holdingRate > 1 {
		return models.SKUParameters{}, fmt.Errorf("row %d: holding_cost_rate should be between 0 and 1 (got %.4f)", row, holdingRate)
	}

	return models.SKUParameters{
		SKU:              sku,
		CurrentInventory: currentInv,
		LeadTimeDays:     leadTime,
		UnitCost:         unitCost,
		OrderCost:        orderCost,
		HoldingCostRate:  holdingRate,
	}, nil
}

// ---------------------------------------------------------------------------
// Header validation
// ---------------------------------------------------------------------------

func validateHeader(got []string, want []string) error {
	if len(got) < len(want) {
		return fmt.Errorf("CSV header has %d columns, expected at least %d: %v", len(got), len(want), want)
	}
	for i, w := range want {
		g := strings.ToLower(strings.TrimSpace(got[i]))
		if g != w {
			return fmt.Errorf("column %d: expected %q, got %q", i+1, w, g)
		}
	}
	return nil
}
