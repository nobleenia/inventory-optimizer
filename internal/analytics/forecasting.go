package analytics

import (
	"math"
)

// ForecastPoint represents a single projected point in time.
type ForecastPoint struct {
	WeekIndex  int     `json:"week_index"`
	Expected   float64 `json:"expected"`
	LowerBound float64 `json:"lower_bound"`
	UpperBound float64 `json:"upper_bound"`
}

// SimpleExponentialSmoothing fits finding SES and projects N weeks forward.
func ForecastSES(demands []float64, periods int, alpha float64) []ForecastPoint {
	if len(demands) == 0 {
		return nil
	}

	// Initialize level L_0 = first demand
	level := demands[0]

	// Fit history to find standard error
	var squaredError float64
	for t := 1; t < len(demands); t++ {
		forecast := level
		err := demands[t] - forecast
		squaredError += err * err
		level = alpha*demands[t] + (1-alpha)*level
	}

	stdErr := 0.0
	if len(demands) > 1 {
		stdErr = math.Sqrt(squaredError / float64(len(demands)-1))
	}

	zScore := 1.96 // 95% confidence approximately

	var forecasts []ForecastPoint
	for i := 1; i <= periods; i++ {
		// For simple exponential smoothing, the forecast is flat
		expected := level
		// Confidence interval expands over time: stdErr * sqrt(i)
		margin := zScore * stdErr * math.Sqrt(float64(i))

		forecasts = append(forecasts, ForecastPoint{
			WeekIndex:  i,
			Expected:   expected,
			LowerBound: math.Max(0, expected-margin),
			UpperBound: expected + margin,
		})
	}

	return forecasts
}
