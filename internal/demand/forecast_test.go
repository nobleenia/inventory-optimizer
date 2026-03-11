package demand

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// SMA
// ---------------------------------------------------------------------------

func TestComputeSMA_Basic(t *testing.T) {
	data := []float64{2, 4, 6, 8, 10}
	got := computeSMA(data, 3)
	want := []float64{4, 6, 8}
	if len(got) != len(want) {
		t.Fatalf("SMA length: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("SMA[%d] = %f, want %f", i, got[i], want[i])
		}
	}
}

func TestComputeSMA_WindowLargerThanData(t *testing.T) {
	data := []float64{3, 7}
	got := computeSMA(data, 4)
	if len(got) != 1 {
		t.Fatalf("expected single-element SMA, got %d", len(got))
	}
	if math.Abs(got[0]-5.0) > 1e-9 {
		t.Errorf("SMA[0] = %f, want 5.0", got[0])
	}
}

// ---------------------------------------------------------------------------
// SES
// ---------------------------------------------------------------------------

func TestComputeSES_Basic(t *testing.T) {
	data := []float64{10, 20, 30}
	alpha := 0.5
	got := computeSES(data, alpha)

	// ses[0] = 10
	// ses[1] = 0.5*20 + 0.5*10 = 15
	// ses[2] = 0.5*30 + 0.5*15 = 22.5
	want := []float64{10, 15, 22.5}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Errorf("SES[%d] = %f, want %f", i, got[i], want[i])
		}
	}
}

func TestComputeSES_AlphaZero(t *testing.T) {
	data := []float64{5, 10, 15}
	got := computeSES(data, 0)
	// With alpha=0 every value stays at the first observation.
	for i, v := range got {
		if math.Abs(v-5.0) > 1e-9 {
			t.Errorf("SES[%d] = %f, want 5.0 (alpha=0)", i, v)
		}
	}
}

func TestComputeSES_AlphaOne(t *testing.T) {
	data := []float64{5, 10, 15}
	got := computeSES(data, 1)
	// With alpha=1 SES just tracks the actual data.
	for i, v := range got {
		if math.Abs(v-data[i]) > 1e-9 {
			t.Errorf("SES[%d] = %f, want %f (alpha=1)", i, v, data[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Linear regression
// ---------------------------------------------------------------------------

func TestLinearRegression_PerfectLine(t *testing.T) {
	// y = 2x + 3
	y := []float64{3, 5, 7, 9, 11}
	slope, intercept := linearRegression(y)
	if math.Abs(slope-2.0) > 1e-9 {
		t.Errorf("slope = %f, want 2.0", slope)
	}
	if math.Abs(intercept-3.0) > 1e-9 {
		t.Errorf("intercept = %f, want 3.0", intercept)
	}
}

func TestLinearRegression_Flat(t *testing.T) {
	y := []float64{5, 5, 5, 5}
	slope, _ := linearRegression(y)
	if math.Abs(slope) > 1e-9 {
		t.Errorf("slope = %f, want ≈0 for flat data", slope)
	}
}

func TestLinearRegression_SinglePoint(t *testing.T) {
	slope, intercept := linearRegression([]float64{7})
	if slope != 0 || intercept != 7 {
		t.Errorf("got slope=%f intercept=%f, want 0 and 7", slope, intercept)
	}
}

// ---------------------------------------------------------------------------
// Trend classification
// ---------------------------------------------------------------------------

func TestClassifyTrend(t *testing.T) {
	tests := []struct {
		slope, mean float64
		want        string
	}{
		{0.01, 10, "stable"},  // 0.1% per week
		{0.5, 10, "rising"},   // 5% per week
		{-0.3, 10, "falling"}, // 3% per week
		{0, 0, "stable"},      // zero mean edge case
	}
	for _, tt := range tests {
		got := classifyTrend(tt.slope, tt.mean)
		if got != tt.want {
			t.Errorf("classifyTrend(%f, %f) = %q, want %q", tt.slope, tt.mean, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Variability classification
// ---------------------------------------------------------------------------

func TestClassifyVariability(t *testing.T) {
	tests := []struct {
		cv   float64
		want string
	}{
		{0.1, "stable"},
		{0.4, "variable"},
		{0.8, "erratic"},
	}
	for _, tt := range tests {
		got := classifyVariability(tt.cv)
		if got != tt.want {
			t.Errorf("classifyVariability(%f) = %q, want %q", tt.cv, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration: Forecast()
// ---------------------------------------------------------------------------

func TestForecast_Integration(t *testing.T) {
	sales := []float64{10, 12, 11, 13, 15, 14, 16, 18}
	opts := DefaultForecastOptions()
	res := Forecast("TEST001", sales, opts)

	if res.SKU != "TEST001" {
		t.Errorf("SKU = %q, want TEST001", res.SKU)
	}
	if len(res.SMA) != len(sales)-opts.SMAWindow+1 {
		t.Errorf("SMA length = %d, want %d", len(res.SMA), len(sales)-opts.SMAWindow+1)
	}
	if len(res.SES) != len(sales) {
		t.Errorf("SES length = %d, want %d", len(res.SES), len(sales))
	}
	if len(res.ForecastedSMA) != opts.ForecastWeeks {
		t.Errorf("ForecastedSMA length = %d, want %d", len(res.ForecastedSMA), opts.ForecastWeeks)
	}
	if res.TrendLabel != "rising" {
		t.Errorf("TrendLabel = %q, want 'rising' for upward data", res.TrendLabel)
	}
	if res.SeasonalityFlag != "stable" {
		t.Errorf("SeasonalityFlag = %q, want 'stable' for low-variance data", res.SeasonalityFlag)
	}
}

func TestForecast_EmptyData(t *testing.T) {
	res := Forecast("EMPTY", nil, DefaultForecastOptions())
	if res.SKU != "EMPTY" {
		t.Errorf("SKU = %q, want EMPTY", res.SKU)
	}
	if len(res.WeeklySales) != 0 {
		t.Errorf("expected empty WeeklySales")
	}
}
