// Package inventory — reorder point calculation.
//
// The Reorder Point (ROP) indicates the inventory level at which a
// replenishment order should be triggered.
//
// Formula:
//
//	ROP = μ_LT + SafetyStock
//
// where μ_LT is the mean demand during the supplier lead time.
package inventory

import "github.com/noble-ch/inventory-optimizer/internal/models"

// ComputeReorderPoint calculates the reorder point for a single SKU.
// It requires the demand statistics and a pre-computed safety stock value.
func ComputeReorderPoint(stats models.DemandStats, safetyStock float64) float64 {
	return stats.LeadTimeMean + safetyStock
}
