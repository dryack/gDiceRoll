package statistics

import (
	"github.com/dryack/gDiceRoll/core/utils"
	"math"
	"sort"
)

// Result represents the outcome of a statistical calculation
type Result struct {
	Min               int
	Max               int
	Mean              float64
	Variance          float64
	StandardDeviation float64
	Skewness          float64
	Kurtosis          float64
	Percentiles       map[int]float64
}

// Calculate computes statistical measures for a given set of integers
func Calculate(data []int) *Result {
	sort.Ints(data)
	n := float64(len(data))

	min := data[0]
	max := data[len(data)-1]
	sum := 0
	for _, v := range data {
		sum += v
	}
	mean := float64(sum) / n

	// Calculate variance and higher moments
	m2 := 0.0
	m3 := 0.0
	m4 := 0.0
	for _, v := range data {
		diff := float64(v) - mean
		m2 += diff * diff
		m3 += diff * diff * diff
		m4 += diff * diff * diff * diff
	}
	variance := m2 / n
	stdDev := math.Sqrt(variance)

	// Calculate skewness and kurtosis
	skewness := (m3 / n) / math.Pow(stdDev, 3)
	kurtosis := (m4/n)/math.Pow(variance, 2) - 3 // Excess kurtosis

	// Calculate percentiles
	percentiles := map[int]float64{
		0:   float64(min),
		5:   percentile(data, 5),
		10:  percentile(data, 10),
		25:  percentile(data, 25),
		50:  percentile(data, 50),
		75:  percentile(data, 75),
		90:  percentile(data, 90),
		95:  percentile(data, 95),
		100: float64(max),
	}

	return &Result{
		Min:               min,
		Max:               max,
		Mean:              utils.Round(mean, 2),
		Variance:          utils.Round(variance, 2),
		StandardDeviation: utils.Round(stdDev, 2),
		Skewness:          utils.Round(skewness, 2),
		Kurtosis:          utils.Round(kurtosis, 2),
		Percentiles:       utils.RoundMap(percentiles, 2),
	}
}

func percentile(data []int, p int) float64 {
	index := float64(len(data)-1) * float64(p) / 100
	i := int(index)
	if i == len(data)-1 {
		return float64(data[i])
	}
	return float64(data[i]) + (float64(data[i+1])-float64(data[i]))*(index-float64(i))
}
