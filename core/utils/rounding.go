package utils

import "math"

func Round(x float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(x*shift) / shift
}

// RoundMap rounds all float64 values in a map to a specified number of decimal places
func RoundMap(m map[int]float64, places int) map[int]float64 {
	rounded := make(map[int]float64)
	for k, v := range m {
		rounded[k] = Round(v, places)
	}
	return rounded
}
