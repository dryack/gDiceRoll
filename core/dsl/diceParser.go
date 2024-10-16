package dsl

import (
	"context"
	"fmt"
	"github.com/dryack/gDiceRoll/core/statistics"
	"log"
	"math/rand"
	"time"
)

// Parse takes a string input and returns a Result
func Parse(ctx context.Context, input string, cache Cache, db Database) (*Result, error) {
	log.Printf("Attempting to parse input: %s", input)
	expr, err := DiceParser.ParseString("", input)
	if err != nil {
		log.Printf("Parsing error: %v", err)
		return nil, fmt.Errorf("parsing error: %w", err)
	}
	log.Printf("Successfully parsed input. Expression: %+v", expr)

	// Check cache first
	cachedResult, err := cache.Get(ctx, input)
	if err == nil {
		// Cache hit
		value, breakdown, _ := evaluateExpression(expr)
		return &Result{
			Expression: input,
			Parsed:     expr,
			Value:      value,
			Breakdown:  breakdown,
			Statistics: cachedResult.Statistics,
			Source:     SourceCache,
		}, nil
	}

	// Check database if not in cache
	dbResult, err := db.Get(ctx, input)
	if err == nil {
		// Database hit
		value, breakdown, _ := evaluateExpression(expr)
		// Asynchronously update cache
		go cache.Set(ctx, input, dbResult)
		return &Result{
			Expression: input,
			Parsed:     expr,
			Value:      value,
			Breakdown:  breakdown,
			Statistics: dbResult.Statistics,
			Source:     SourceDatabase,
		}, nil
	}

	// If not found in cache or database, calculate
	value, breakdown, err := evaluateExpression(expr)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}

	// Perform Monte Carlo simulation
	simCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	simFunc := func() int {
		simValue, _, _ := evaluateExpression(expr)
		return simValue
	}

	stats := statistics.MonteCarloSimulation(simCtx, simFunc, 1000000)

	result := &Result{
		Expression: input,
		Parsed:     expr,
		Value:      value,
		Breakdown:  breakdown,
		Statistics: stats,
		Source:     SourceFreshCalculation,
	}

	// Asynchronously update cache and database
	cachedResult = &CachedResult{Statistics: stats}
	go func() {
		cache.Set(context.Background(), input, cachedResult)
		db.Set(context.Background(), input, cachedResult)
	}()

	log.Printf("Evaluation result: %+v", result)
	return result, nil
}

// evaluateExpression evaluates a parsed Expression and returns the result and breakdown
func evaluateExpression(expr *Expression) (int, []int, error) {
	if expr.Number != nil {
		return *expr.Number, []int{*expr.Number}, nil
	}
	if expr.Dice != nil {
		return evaluateDice(expr.Dice)
	}
	return 0, nil, fmt.Errorf("invalid expression")
}

// evaluateDice evaluates a Dice structure and returns the result and breakdown
func evaluateDice(dice *Dice) (int, []int, error) {
	count := 1
	if dice.Count != nil {
		count = *dice.Count
	}

	var sides int
	if dice.Sides.Value == "%" {
		sides = 100
	} else {
		var err error
		sides, err = ParseInt(dice.Sides.Value)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid dice sides: %s", dice.Sides.Value)
		}
	}

	result := 0
	breakdown := make([]int, count)
	for i := 0; i < count; i++ {
		roll := rand.Intn(sides) + 1
		result += roll
		breakdown[i] = roll
	}

	return result, breakdown, nil
}

// ParseInt is a helper function to parse a string to an integer
func ParseInt(s string) (int, error) {
	var result int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(ch-'0')
	}
	return result, nil
}
