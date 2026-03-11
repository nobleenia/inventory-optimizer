// Package inventory — Economic Order Quantity (EOQ) calculation.
//
// EOQ determines the optimal order size that minimises the sum of
// ordering costs and holding costs over a year.
//
// Formula:
//
//	EOQ = √( (2 × D × S) / H )
//
// where
//
//	D = annual demand (units)
//	S = fixed cost per order
//	H = annual holding cost per unit  (= unit_cost × holding_cost_rate)
package inventory

import (
	"fmt"
	"math"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ComputeEOQ calculates the Economic Order Quantity for a single SKU.
func ComputeEOQ(stats models.DemandStats, params models.SKUParameters) (float64, error) {
	D := stats.AnnualDemand
	S := params.OrderCost
	H := params.UnitCost * params.HoldingCostRate

	if D <= 0 {
		return 0, fmt.Errorf("SKU %s: annual demand must be positive (got %.2f)", params.SKU, D)
	}
	if S <= 0 {
		return 0, fmt.Errorf("SKU %s: order cost must be positive (got %.2f)", params.SKU, S)
	}
	if H <= 0 {
		return 0, fmt.Errorf("SKU %s: annual holding cost per unit must be positive (got %.2f)", params.SKU, H)
	}

	return math.Sqrt((2 * D * S) / H), nil
}
