package dsl

import (
	"fmt"
	"log"
	"math/rand"
)

// Parse takes a string input and returns a parsed Expression
func Parse(input string) (*Expression, error) {
	log.Printf("Attempting to parse input: %s", input)
	expr, err := DiceParser.ParseString("", input)
	if err != nil {
		log.Printf("Parsing error: %v", err)
		return nil, fmt.Errorf("parsing error: %w", err)
	}
	log.Printf("Successfully parsed input. Result: %+v", expr)
	if expr.Dice != nil {
		log.Printf("Dice details - Count: %v, Sides: %s", expr.Dice.Count, expr.Dice.Sides.Value)
	} else if expr.Number != nil {
		log.Printf("Number: %d", *expr.Number)
	}
	return expr, nil
}

// EvaluateExpression evaluates a parsed Expression and returns the result
func EvaluateExpression(expr *Expression) (int, error) {
	if expr.Number != nil {
		return *expr.Number, nil
	}
	if expr.Dice != nil {
		return evaluateDice(expr.Dice)
	}
	return 0, fmt.Errorf("invalid expression")
}

// evaluateDice evaluates a Dice structure and returns the result
func evaluateDice(dice *Dice) (int, error) {
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
			return 0, fmt.Errorf("invalid dice sides: %s", dice.Sides.Value)
		}
	}

	result := 0
	for i := 0; i < count; i++ {
		result += rand.Intn(sides) + 1
	}

	return result, nil
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
