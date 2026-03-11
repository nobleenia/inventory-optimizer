// forecast.go implements lightweight demand forecasting methods:
//
//   - Simple Moving Average (SMA)
//   - Single Exponential Smoothing (SES)
//   - Linear trend detection
//   - Demand variability classification
//
// These are intentionally simple, well-understood techniques appropriate
// for small e-commerce sellers with limited data (20–104 weeks).
package demand

import (
	"math"

	"github.com/noble-ch/inventory-optimizer/internal/models"
)

// ForecastOptions configures the forecast computation.
type ForecastOptions struct {
	SMAWindow     int     // moving-average window size (default 4)
	SESAlpha      float64 // exponential smoothing factor 0–1 (default 0.3)
	ForecastWeeks int     // how many weeks ahead to project (default 8)
}

// DefaultForecastOptions returns production defaults.
func DefaultForecastOptions() ForecastOptions {
	return ForecastOptions{
		SMAWindow:     4,
		SESAlpha:      0.3,
		ForecastWeeks: 8,
	}
}

// Forecast computes SMA, SES, trend, and variability metrics for one SKU.
// The weeklySales slice must be in chronological order with ≥ 2 data points.
func Forecast(sku string, weeklySales []float64, opts ForecastOptions) models.ForecastResult {
	n := len(weeklySales)
	if n == 0 {
		return models.ForecastResult{SKU: sku}
	}

	// Clamp window to available data.
	window := opts.SMAWindow
	if window < 1 {
		window = 1
	}
	if window > n {
		window = n
	}

	// ── Simple Moving Average ─────────────────────────────────────────
	sma := computeSMA(weeklySales, window)

	// ── Single Exponential Smoothing ──────────────────────────────────
	ses := computeSES(weeklySales, opts.SESAlpha)

	// ── Forecast projection ───────────────────────────────────────────
	fw := opts.ForecastWeeks
	if fw < 1 {
		fw = 1
	}

	lastSMA := sma[len(sma)-1]
	lastSES := ses[len(ses)-1]

	forecastSMA := make([]float64, fw)
	forecastSES := make([]float64, fw)
	for i := 0; i < fw; i++ {
		forecastSMA[i] = lastSMA
		forecastSES[i] = lastSES
	}

	// ── Trend detection (linear regression) ───────────────────────────
	slope, intercept := linearRegression(weeklySales)
	trendLabel := classifyTrend(slope, mean(weeklySales))

	// ── Variability / seasonality flag ────────────────────────────────
	mu := mean(weeklySales)
	sd := stddev(weeklySales, mu)
	cv := 0.0
	if mu > 0 {
		cv = sd / mu
	}
	seasonFlag := classifyVariability(cv)

	return models.ForecastResult{
		SKU:              sku,
		WeeklySales:      weeklySales,
		SMA:              sma,
		SES:              ses,
		ForecastWeeks:    fw,
		ForecastedSMA:    forecastSMA,
		ForecastedSES:    forecastSES,
		TrendSlope:       slope,
		TrendIntercept:   intercept,
		TrendLabel:       trendLabel,
		CoeffOfVariation: cv,
		SeasonalityFlag:  seasonFlag,
		Alpha:            opts.SESAlpha,
	}
}

// ---------------------------------------------------------------------------
// Algorithm implementations
// ---------------------------------------------------------------------------

// computeSMA returns the simple moving average series.
// Output length = len(data) - window + 1.
func computeSMA(data []float64, window int) []float64 {
	n := len(data)
	if n < window {
		return []float64{mean(data)}
	}

	out := make([]float64, 0, n-window+1)
	var windowSum float64
	for i := 0; i < window; i++ {
		windowSum += data[i]
	}
	out = append(out, windowSum/float64(window))

	for i := window; i < n; i++ {
		windowSum += data[i] - data[i-window]
		out = append(out, windowSum/float64(window))
	}
	return out
}

// computeSES returns the single exponential smoothing series.
// Output length = len(data). ses[0] = data[0].
func computeSES(data []float64, alpha float64) []float64 {
	if len(data) == 0 {
		return nil
	}
	out := make([]float64, len(data))
	out[0] = data[0]
	for i := 1; i < len(data); i++ {
		out[i] = alpha*data[i] + (1-alpha)*out[i-1]
	}
	return out
}

// linearRegression fits y = slope*x + intercept via ordinary least squares
// where x = 0, 1, 2, … (week index).
func linearRegression(y []float64) (slope, intercept float64) {
	n := float64(len(y))
	if n < 2 {
		if n == 1 {
			return 0, y[0]
		}
		return 0, 0
	}

	var sx, sy, sxx, sxy float64
	for i, v := range y {
		x := float64(i)
		sx += x
		sy += v
		sxx += x * x
		sxy += x * v
	}

	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0, sy / n
	}

	slope = (n*sxy - sx*sy) / denom
	intercept = (sy - slope*sx) / n
	return
}

// classifyTrend labels the trend based on slope relative to mean demand.
func classifyTrend(slope, meanDemand float64) string {
	if meanDemand == 0 {
		return "stable"
	}
	// Normalize slope: a change of >2% per week relative to the mean
	// is considered significant.
	rel := math.Abs(slope) / meanDemand
	switch {
	case rel < 0.02:
		return "stable"
	case slope > 0:
		return "rising"
	default:
		return "falling"
	}
}

// classifyVariability labels demand volatility using the coefficient of variation.
func classifyVariability(cv float64) string {
	switch {
	case cv < 0.3:
		return "stable"
	case cv < 0.6:
		return "variable"
	default:
		return "erratic"
	}
}
