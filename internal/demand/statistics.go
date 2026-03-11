// Package demand computes statistical profiles from historical sales data.
// It produces a DemandStats struct per SKU that downstream packages
// (inventory, simulation) consume without needing access to the raw records.
package demand

import (
	"fmt"
	"math"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// weeksPerYear is used to annualize weekly demand figures.
const weeksPerYear = 52

// daysPerWeek converts weekly statistics to daily equivalents.
const daysPerWeek = 7

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Analyze groups sales records by SKU, computes weekly demand statistics,
// converts them to daily / lead-time equivalents, and returns a DemandStats
// per SKU. The lead-time for each SKU is looked up from the params map.
//
// An error is returned if a SKU in the sales data has no matching entry in
// the params map, or if a SKU has fewer than 2 data points (insufficient
// for standard deviation).
func Analyze(records []models.SalesRecord, params map[string]models.SKUParameters) ([]models.DemandStats, error) {
	// Group weekly sales by SKU.
	grouped := groupBySKU(records)

	var results []models.DemandStats

	for sku, weeklyUnits := range grouped {
		p, ok := params[sku]
		if !ok {
			return nil, fmt.Errorf("SKU %q found in sales data but missing from parameters file", sku)
		}

		n := len(weeklyUnits)
		if n < 2 {
			return nil, fmt.Errorf("SKU %q has only %d data point(s); need at least 2 for variability estimation", sku, n)
		}

		weeklyMean := mean(weeklyUnits)
		weeklyStd := stddev(weeklyUnits, weeklyMean)

		dailyMean := weeklyMean / daysPerWeek
		dailyStd := weeklyStd / math.Sqrt(daysPerWeek)

		ltDays := float64(p.LeadTimeDays)
		ltMean := dailyMean * ltDays
		ltStd := dailyStd * math.Sqrt(ltDays)

		results = append(results, models.DemandStats{
			SKU:             sku,
			WeeklyMean:      weeklyMean,
			WeeklyStdDev:    weeklyStd,
			DailyMean:       dailyMean,
			DailyStdDev:     dailyStd,
			AnnualDemand:    weeklyMean * weeksPerYear,
			DataPointsCount: n,
			LeadTimeMean:    ltMean,
			LeadTimeStdDev:  ltStd,
		})
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// groupBySKU collects weekly unit sales into slices keyed by SKU.
func groupBySKU(records []models.SalesRecord) map[string][]float64 {
	m := make(map[string][]float64)
	for _, r := range records {
		m[r.SKU] = append(m[r.SKU], float64(r.UnitsSold))
	}
	return m
}

// mean returns the arithmetic mean of a float64 slice.
func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// stddev computes the sample standard deviation (Bessel-corrected, n-1).
func stddev(vals []float64, mu float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var ss float64
	for _, v := range vals {
		d := v - mu
		ss += d * d
	}
	return math.Sqrt(ss / float64(len(vals)-1))
}
