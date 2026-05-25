// Package models defines the core data structures used throughout the
// inventory-optimizer engine. Every other package communicates through
// these types, keeping coupling low and intent clear.
package models

import "time"

// Version is the semantic version of the engine, set at build time.
const Version = "0.7.0"

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
	SKU              string  `json:"sku"`
	CurrentInventory int     `json:"current_inventory"`
	LeadTimeDays     int     `json:"lead_time_days"`
	UnitCost         float64 `json:"unit_cost"`
	OrderCost        float64 `json:"order_cost"`
	HoldingCostRate  float64 `json:"holding_cost_rate"` // annual, e.g. 0.25 = 25 %
}

// ---------------------------------------------------------------------------
// Analysis layer — computed demand statistics per SKU
// ---------------------------------------------------------------------------

// DemandStats holds the statistical profile of a SKU's historical demand.
type DemandStats struct {
	SKU             string  `json:"sku"`
	WeeklyMean      float64 `json:"weekly_mean"`
	WeeklyStdDev    float64 `json:"weekly_std_dev"`
	DailyMean       float64 `json:"daily_mean"`
	DailyStdDev     float64 `json:"daily_std_dev"`
	AnnualDemand    float64 `json:"annual_demand"`
	DataPointsCount int     `json:"data_points_count"`
	LeadTimeMean    float64 `json:"lead_time_mean"`    // mean demand during lead time
	LeadTimeStdDev  float64 `json:"lead_time_std_dev"` // std-dev of demand during lead time
}

// ---------------------------------------------------------------------------
// Policy layer — inventory control recommendations
// ---------------------------------------------------------------------------

// InventoryPolicy contains the computed reorder policy for a single SKU.
type InventoryPolicy struct {
	SKU          string  `json:"sku"`
	EOQ          float64 `json:"eoq"`           // economic order quantity (units)
	SafetyStock  float64 `json:"safety_stock"`  // buffer inventory (units)
	ReorderPoint float64 `json:"reorder_point"` // trigger level for placing an order (units)
	ServiceLevel float64 `json:"service_level"` // target service level used (e.g. 0.95)
}

// ---------------------------------------------------------------------------
// Simulation layer — Monte-Carlo output
// ---------------------------------------------------------------------------

// SimulationResult holds the aggregated outcome of all Monte-Carlo runs
// for a single SKU.
type SimulationResult struct {
	SKU                  string  `json:"sku"`
	Runs                 int     `json:"runs"`
	WeeksPerRun          int     `json:"weeks_per_run"`
	AvgStockouts         float64 `json:"avg_stockouts"`        // mean stockout events per year
	StockoutProbability  float64 `json:"stockout_probability"` // fraction of weeks with a stockout
	AvgInventoryLevel    float64 `json:"avg_inventory_level"`  // mean units on hand
	AvgAnnualHoldingCost float64 `json:"avg_annual_holding_cost"`
	AvgAnnualOrderCost   float64 `json:"avg_annual_order_cost"`
	AvgTotalAnnualCost   float64 `json:"avg_total_annual_cost"`
}

// ---------------------------------------------------------------------------
// Forecast layer — demand forecasting outputs
// ---------------------------------------------------------------------------

// ForecastResult holds the output of demand forecasting for a single SKU.
type ForecastResult struct {
	SKU string `json:"sku"`

	// Historical weekly sales used as input (in chronological order).
	WeeklySales []float64 `json:"weekly_sales"`

	// Simple Moving Average (window = min(4, len(sales))).
	SMA []float64 `json:"sma"`

	// Single Exponential Smoothing fitted values.
	SES []float64 `json:"ses"`

	// Forecasted demand for the next N weeks (default 8).
	ForecastWeeks int       `json:"forecast_weeks"`
	ForecastedSMA []float64 `json:"forecasted_sma"` // flat projection of last SMA value
	ForecastedSES []float64 `json:"forecasted_ses"` // flat projection of last SES value

	// Trend: linear regression slope of weekly demand.
	TrendSlope     float64 `json:"trend_slope"`
	TrendIntercept float64 `json:"trend_intercept"`
	TrendLabel     string  `json:"trend_label"` // "rising", "falling", "stable"

	// Seasonality: coefficient of variation (stddev/mean).
	// High CV (>0.5) suggests possible seasonality or erratic demand.
	CoeffOfVariation float64 `json:"coeff_of_variation"`
	SeasonalityFlag  string  `json:"seasonality_flag"` // "stable", "variable", "erratic"

	// Smoothing parameter used for SES.
	Alpha float64 `json:"alpha"`
}

// ---------------------------------------------------------------------------
// Report layer — unified per-SKU summary
// ---------------------------------------------------------------------------

// SKUReport combines all computed outputs for a single SKU into one
// structure that the reporting package can render.
type SKUReport struct {
	Parameters SKUParameters    `json:"parameters"`
	Demand     DemandStats      `json:"demand"`
	Policy     InventoryPolicy  `json:"policy"`
	Simulation SimulationResult `json:"simulation"`
	Forecast   ForecastResult   `json:"forecast"`
}
