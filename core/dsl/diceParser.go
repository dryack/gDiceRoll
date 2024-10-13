package dsl

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type DiceRoll struct {
	Expression string
	Result     int
	Breakdown  []int
}

func Parse(expression string) (*DiceRoll, error) {
	parts := strings.Split(expression, "d")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid dice expression")
	}

	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid dice count")
	}

	sides, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid dice sides")
	}

	result := 0
	breakdown := make([]int, count)
	for i := 0; i < count; i++ {
		roll := rand.Intn(sides) + 1
		result += roll
		breakdown[i] = roll
	}

	return &DiceRoll{
		Expression: expression,
		Result:     result,
		Breakdown:  breakdown,
	}, nil
}
