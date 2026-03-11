// Package inventory provides inventory policy calculations.
//
// This file implements Safety Stock computation using the service-level
// (Z-score) approach. Safety stock acts as a buffer against demand
// variability during the supplier lead time.
//
// Formula:
//
//	SafetyStock = Z × σ_LT
//
// where Z is the standard normal quantile for the target service level
// and σ_LT is the standard deviation of demand during lead time.
package inventory

import (
	"fmt"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ServiceLevelZ maps commonly used service levels to their Z-scores.
var ServiceLevelZ = map[float64]float64{
	0.90: 1.28,
	0.95: 1.65,
	0.99: 2.33,
}

// DefaultServiceLevel is the target service level for V1.
const DefaultServiceLevel = 0.95

// ComputeSafetyStock calculates the safety stock for a single SKU.
// It returns the buffer quantity (in units) required to achieve the
// target service level.
func ComputeSafetyStock(stats models.DemandStats, serviceLevel float64) (float64, error) {
	z, ok := ServiceLevelZ[serviceLevel]
	if !ok {
		return 0, fmt.Errorf("unsupported service level %.2f; choose from 0.90, 0.95, or 0.99", serviceLevel)
	}

	if stats.LeadTimeStdDev < 0 {
		return 0, fmt.Errorf("lead-time std-dev cannot be negative (SKU %s)", stats.SKU)
	}

	return z * stats.LeadTimeStdDev, nil
}
