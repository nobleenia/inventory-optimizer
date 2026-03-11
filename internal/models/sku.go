// Package models defines the core data structures used throughout the
// inventory-optimizer engine. Every other package communicates through
// these types, keeping coupling low and intent clear.
package models

import "time"

// Version is the semantic version of the engine, set at build time.
const Version = "0.4.0"

// ---------------------------------------------------------------------------
// Input layer — raw data ingested from CSV files
// ---------------------------------------------------------------------------

// SalesRecord represents a single row from the sales history CSV.
type SalesRecord struct {
	SKU       string
	Week      time.Time
	UnitsSold int
}

// SKUParameters represents a single row from the SKU parameters CSV.
type SKUParameters struct {
	SKU              string
	CurrentInventory int
	LeadTimeDays     int
	UnitCost         float64
	OrderCost        float64
	HoldingCostRate  float64 // annual, e.g. 0.25 = 25 %
}

// ---------------------------------------------------------------------------
// Analysis layer — computed demand statistics per SKU
// ---------------------------------------------------------------------------

// DemandStats holds the statistical profile of a SKU's historical demand.
type DemandStats struct {
	SKU             string
	WeeklyMean      float64
	WeeklyStdDev    float64
	DailyMean       float64
	DailyStdDev     float64
	AnnualDemand    float64
	DataPointsCount int
	LeadTimeMean    float64 // mean demand during lead time
	LeadTimeStdDev  float64 // std-dev of demand during lead time
}

// ---------------------------------------------------------------------------
// Policy layer — inventory control recommendations
// ---------------------------------------------------------------------------

// InventoryPolicy contains the computed reorder policy for a single SKU.
type InventoryPolicy struct {
	SKU          string
	EOQ          float64 // economic order quantity (units)
	SafetyStock  float64 // buffer inventory (units)
	ReorderPoint float64 // trigger level for placing an order (units)
	ServiceLevel float64 // target service level used (e.g. 0.95)
}

// ---------------------------------------------------------------------------
// Simulation layer — Monte-Carlo output
// ---------------------------------------------------------------------------

// SimulationResult holds the aggregated outcome of all Monte-Carlo runs
// for a single SKU.
type SimulationResult struct {
	SKU                  string
	Runs                 int
	WeeksPerRun          int
	AvgStockouts         float64 // mean stockout events per year
	StockoutProbability  float64 // fraction of weeks with a stockout
	AvgInventoryLevel    float64 // mean units on hand
	AvgAnnualHoldingCost float64
	AvgAnnualOrderCost   float64
	AvgTotalAnnualCost   float64
}

// ---------------------------------------------------------------------------
// Forecast layer — demand forecasting outputs
// ---------------------------------------------------------------------------

// ForecastResult holds the output of demand forecasting for a single SKU.
type ForecastResult struct {
	SKU string

	// Historical weekly sales used as input (in chronological order).
	WeeklySales []float64

	// Simple Moving Average (window = min(4, len(sales))).
	SMA []float64

	// Single Exponential Smoothing fitted values.
	SES []float64

	// Forecasted demand for the next N weeks (default 8).
	ForecastWeeks int
	ForecastedSMA []float64 // flat projection of last SMA value
	ForecastedSES []float64 // flat projection of last SES value

	// Trend: linear regression slope of weekly demand.
	TrendSlope     float64
	TrendIntercept float64
	TrendLabel     string // "rising", "falling", "stable"

	// Seasonality: coefficient of variation (stddev/mean).
	// High CV (>0.5) suggests possible seasonality or erratic demand.
	CoeffOfVariation float64
	SeasonalityFlag  string // "stable", "variable", "erratic"

	// Smoothing parameter used for SES.
	Alpha float64
}

// ---------------------------------------------------------------------------
// Report layer — unified per-SKU summary
// ---------------------------------------------------------------------------

// SKUReport combines all computed outputs for a single SKU into one
// structure that the reporting package can render.
type SKUReport struct {
	Parameters SKUParameters
	Demand     DemandStats
	Policy     InventoryPolicy
	Simulation SimulationResult
	Forecast   ForecastResult
}
