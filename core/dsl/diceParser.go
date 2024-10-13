package dsl

import (
	"fmt"
	"log"
	"math/rand"
)

// Parse takes a string input and returns a Result
func Parse(input string) (*Result, error) {
	log.Printf("Attempting to parse input: %s", input)
	expr, err := DiceParser.ParseString("", input)
	if err != nil {
		log.Printf("Parsing error: %v", err)
		return nil, fmt.Errorf("parsing error: %w", err)
	}
	log.Printf("Successfully parsed input. Expression: %+v", expr)

	// Evaluate the expression
	value, breakdown, err := evaluateExpression(expr)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}

	result := &Result{
		Expression: input,
		Parsed:     expr,
		Value:      value,
		Breakdown:  breakdown,
	}

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
