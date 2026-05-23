package analytics

import (
	"github.com/noble-ch/inventory-optimizer/internal/models"
	"github.com/noble-ch/inventory-optimizer/internal/store"
	"sort"
)

type BudgetRecommendation struct {
	SKUID           string  `json:"sku_id"`
	OrderQuantity   int     `json:"order_quantity"`
	Cost            float64 `json:"cost"`
	StockoutAverted float64 `json:"stockout_averted"` // Estimated stockouts reduced
}

// OptimizeBudget allocates a limited cash budget across SKUs based on expected stockout impact.
func OptimizeBudget(budget float64, skus []store.SKU, results map[string]models.SKUReport) []BudgetRecommendation {
	type candidate struct {
		sku  store.SKU
		roi  float64
		eoq  float64
		need int
	}

	var candidates []candidate
	for _, s := range skus {
		rep, ok := results[s.SKUID]
		if !ok {
			continue
		}

		if float64(s.CurrentStock) < rep.Policy.ReorderPoint {
			need := int(rep.Policy.EOQ)
			if need == 0 {
				need = 1
			}

			margin := s.SellingPrice - s.UnitCost
			if margin <= 0 {
				margin = s.UnitCost
			}
			impact := rep.Simulation.AvgStockouts * margin
			roi := impact / (float64(need) * s.UnitCost)

			candidates = append(candidates, candidate{
				sku:  s,
				roi:  roi,
				eoq:  rep.Policy.EOQ,
				need: need,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].roi > candidates[j].roi
	})

	var recs []BudgetRecommendation
	remaining := budget

	for _, c := range candidates {
		cost := float64(c.need) * c.sku.UnitCost
		if remaining >= cost {
			recs = append(recs, BudgetRecommendation{
				SKUID:           c.sku.SKUID,
				OrderQuantity:   c.need,
				Cost:            cost,
				StockoutAverted: c.roi * cost,
			})
			remaining -= cost
		} else if remaining > c.sku.UnitCost {
			partialQty := int(remaining / c.sku.UnitCost)
			if partialQty > 0 {
				partialCost := float64(partialQty) * c.sku.UnitCost
				recs = append(recs, BudgetRecommendation{
					SKUID:           c.sku.SKUID,
					OrderQuantity:   partialQty,
					Cost:            partialCost,
					StockoutAverted: c.roi * partialCost,
				})
				remaining -= partialCost
			}
		}
		if remaining <= 0 {
			break
		}
	}

	return recs
}
