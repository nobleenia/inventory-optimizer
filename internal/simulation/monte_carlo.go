// Package simulation implements a Monte-Carlo engine that evaluates
// inventory performance under demand uncertainty.
//
// For each SKU it runs N simulations of W weeks. In every simulated
// week the engine:
//
//  1. Generates random demand from a normal distribution.
//  2. Subtracts demand from current inventory.
//  3. Records a stockout if inventory goes negative.
//  4. Places a replenishment order when inventory ≤ ROP.
//  5. Receives ordered units after the lead time elapses.
//  6. Accumulates holding and ordering costs.
//
// The package exposes only the Run function; all internal mechanics are
// unexported, keeping the API surface minimal.
package simulation

import (
	"math"
	"math/rand"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// DefaultRuns is the number of Monte-Carlo iterations per SKU.
const DefaultRuns = 500

// DefaultWeeks is the simulation horizon in weeks.
const DefaultWeeks = 52

// daysPerWeek is used to convert daily demand parameters back to weekly.
const daysPerWeek = 7

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Config lets callers override simulation defaults.
type Config struct {
	Runs  int // number of simulation runs (default 500)
	Weeks int // weeks per run (default 52)
	Seed  int64
}

// Run executes the Monte-Carlo simulation for a single SKU and returns
// the aggregated result. It is safe for concurrent use when each call
// receives its own Config (each call creates its own RNG).
func Run(
	params models.SKUParameters,
	stats models.DemandStats,
	policy models.InventoryPolicy,
	cfg Config,
) models.SimulationResult {
	runs := cfg.Runs
	if runs <= 0 {
		runs = DefaultRuns
	}
	weeks := cfg.Weeks
	if weeks <= 0 {
		weeks = DefaultWeeks
	}

	rng := rand.New(rand.NewSource(cfg.Seed))

	// Accumulators across all runs.
	var (
		totalStockouts     float64
		totalStockoutWeeks float64
		totalAvgInventory  float64
		totalHoldingCost   float64
		totalOrderCost     float64
	)

	weeklyMean := stats.DailyMean * daysPerWeek
	weeklyStd := stats.DailyStdDev * math.Sqrt(daysPerWeek)
	holdingCostPerUnit := params.UnitCost * params.HoldingCostRate / DefaultWeeks // per unit per week
	leadTimeWeeks := int(math.Ceil(float64(params.LeadTimeDays) / daysPerWeek))

	for r := 0; r < runs; r++ {
		result := simulateOneRun(rng, weeks, weeklyMean, weeklyStd,
			float64(params.CurrentInventory), policy.ReorderPoint, policy.EOQ,
			holdingCostPerUnit, params.OrderCost, leadTimeWeeks)

		totalStockouts += float64(result.stockouts)
		totalStockoutWeeks += float64(result.stockoutWeeks)
		totalAvgInventory += result.avgInventory
		totalHoldingCost += result.holdingCost
		totalOrderCost += result.orderCost
	}

	n := float64(runs)
	totalWeeks := float64(runs) * float64(weeks)

	return models.SimulationResult{
		SKU:                  params.SKU,
		Runs:                 runs,
		WeeksPerRun:          weeks,
		AvgStockouts:         totalStockouts / n,
		StockoutProbability:  totalStockoutWeeks / totalWeeks,
		AvgInventoryLevel:    totalAvgInventory / n,
		AvgAnnualHoldingCost: totalHoldingCost / n,
		AvgAnnualOrderCost:   totalOrderCost / n,
		AvgTotalAnnualCost:   (totalHoldingCost + totalOrderCost) / n,
	}
}

// ---------------------------------------------------------------------------
// Single-run mechanics (unexported)
// ---------------------------------------------------------------------------

type runResult struct {
	stockouts     int
	stockoutWeeks int
	avgInventory  float64
	holdingCost   float64
	orderCost     float64
}

func simulateOneRun(
	rng *rand.Rand,
	weeks int,
	weeklyMean, weeklyStd float64,
	startInventory, rop, eoq float64,
	holdingCostPerUnitWeek, orderCostPerOrder float64,
	leadTimeWeeks int,
) runResult {
	inventory := startInventory

	// pendingOrders tracks orders in transit: key = arrival week.
	pendingOrders := make(map[int]float64)

	var (
		stockouts     int
		stockoutWeeks int
		invSum        float64
		holdingCost   float64
		orderCost     float64
	)

	for w := 0; w < weeks; w++ {
		// 1. Receive any pending deliveries arriving this week.
		if qty, ok := pendingOrders[w]; ok {
			inventory += qty
			delete(pendingOrders, w)
		}

		// 2. Generate random demand (floored at zero — can't have negative demand).
		demand := rng.NormFloat64()*weeklyStd + weeklyMean
		if demand < 0 {
			demand = 0
		}

		// 3. Subtract demand.
		inventory -= demand

		// 4. Check for stockout.
		if inventory < 0 {
			stockouts++
			stockoutWeeks++
			inventory = 0 // unfulfilled demand is lost (lost-sales model)
		}

		// 5. Place an order if inventory is at or below the reorder point
		//    and there is no outstanding order for this SKU.
		if inventory <= rop && !hasPending(pendingOrders) {
			arrivalWeek := w + leadTimeWeeks
			if arrivalWeek < weeks { // only track if it arrives within horizon
				pendingOrders[arrivalWeek] = eoq
			}
			orderCost += orderCostPerOrder
		}

		// 6. Accumulate holding cost on ending inventory.
		if inventory > 0 {
			holdingCost += inventory * holdingCostPerUnitWeek
		}

		invSum += inventory
	}

	return runResult{
		stockouts:     stockouts,
		stockoutWeeks: stockoutWeeks,
		avgInventory:  invSum / float64(weeks),
		holdingCost:   holdingCost,
		orderCost:     orderCost,
	}
}

// hasPending reports whether any orders are still in transit.
func hasPending(orders map[int]float64) bool {
	return len(orders) > 0
}
