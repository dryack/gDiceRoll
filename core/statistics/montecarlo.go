package statistics

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
)

// SimulationFunc represents a function that generates a single simulation result
type SimulationFunc func() int

// MonteCarloSimulation performs a Monte Carlo simulation
func MonteCarloSimulation(ctx context.Context, simFunc SimulationFunc, iterations int) *Result {
	results := make([]int, iterations)
	numWorkers := runtime.NumCPU() - 4
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	resultChan := make(chan int, iterations)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					result := simFunc()
					select {
					case resultChan <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for i := 0; i < iterations; i++ {
		select {
		case result, ok := <-resultChan:
			if !ok {
				return Calculate(results[:i])
			}
			results[i] = result
		case <-ctx.Done():
			return Calculate(results[:i])
		}
	}

	return Calculate(results)
}

// RandomDiceRoll simulates rolling a die with a given number of sides
func RandomDiceRoll(sides int) SimulationFunc {
	return func() int {
		return rand.Intn(sides) + 1
	}
}
