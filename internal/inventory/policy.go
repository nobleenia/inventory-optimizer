// Package inventory — policy.go ties the three inventory calculations
// (safety stock, reorder point, EOQ) into a single InventoryPolicy per SKU.
//
// This keeps the individual formula files focused and gives callers a
// single entry point for the complete policy computation.
package inventory

import (
	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ComputePolicy calculates the full inventory policy for one SKU.
func ComputePolicy(stats models.DemandStats, params models.SKUParameters, serviceLevel float64) (models.InventoryPolicy, error) {
	ss, err := ComputeSafetyStock(stats, serviceLevel)
	if err != nil {
		return models.InventoryPolicy{}, err
	}

	rop := ComputeReorderPoint(stats, ss)

	eoq, err := ComputeEOQ(stats, params)
	if err != nil {
		return models.InventoryPolicy{}, err
	}

	return models.InventoryPolicy{
		SKU:          stats.SKU,
		EOQ:          eoq,
		SafetyStock:  ss,
		ReorderPoint: rop,
		ServiceLevel: serviceLevel,
	}, nil
}
