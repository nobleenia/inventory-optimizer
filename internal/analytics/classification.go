package analytics

import (
	"github.com/noble-ch/inventory-optimizer/internal/store"
	"math"
	"sort"
)

type SKUClassification struct {
	SKUID       string  `json:"sku_id"`
	ABCClass    string  `json:"abc_class"`
	XYZClass    string  `json:"xyz_class"`
	AnnualValue float64 `json:"annual_value"`
	CoV         float64 `json:"cov"`
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stddev(vals []float64, mu float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var ss float64
	for _, v := range vals {
		ss += (v - mu) * (v - mu)
	}
	return math.Sqrt(ss / float64(len(vals)-1))
}

func ClassifyCatalogue(skus []store.SKU, sales []store.SalesEntry) map[string]SKUClassification {
	// Group sales by SKU
	salesBySKU := make(map[string][]store.SalesEntry)
	for _, entry := range sales {
		salesBySKU[entry.SKUID] = append(salesBySKU[entry.SKUID], entry)
	}

	results := make(map[string]SKUClassification)
	var valueRanks []SKUClassification
	var totalValue float64

	for _, sku := range skus {
		skuSales := salesBySKU[sku.SKUID]
		if len(skuSales) == 0 {
			results[sku.SKUID] = SKUClassification{SKUID: sku.SKUID, ABCClass: "C", XYZClass: "Z", AnnualValue: 0, CoV: 0}
			continue
		}

		weeklyDemands := make(map[string]int)
		for _, s := range skuSales {
			year, week := s.Date.ISOWeek()
			key := string(rune(year)) + "-" + string(rune(week))
			weeklyDemands[key] += s.Quantity
		}

		var demands []float64
		for _, v := range weeklyDemands {
			demands = append(demands, float64(v))
		}

		mu := mean(demands)
		stdDev := stddev(demands, mu)

		annualDemand := mu * 52
		annualValue := annualDemand * sku.UnitCost
		totalValue += annualValue

		cov := 0.0
		if mu > 0 {
			cov = stdDev / mu
		}

		xyz := "X"
		if cov > 1.0 {
			xyz = "Z"
		} else if cov > 0.5 {
			xyz = "Y"
		}

		classif := SKUClassification{SKUID: sku.SKUID, XYZClass: xyz, AnnualValue: annualValue, CoV: cov}
		valueRanks = append(valueRanks, classif)
	}

	sort.Slice(valueRanks, func(i, j int) bool { return valueRanks[i].AnnualValue > valueRanks[j].AnnualValue })

	cumulative := 0.0
	for _, item := range valueRanks {
		if totalValue > 0 {
			cumulative += item.AnnualValue / totalValue
		}
		abc := "C"
		if cumulative <= 0.80 {
			abc = "A"
		} else if cumulative <= 0.95 {
			abc = "B"
		}

		item.ABCClass = abc
		results[item.SKUID] = item
	}

	return results
}
