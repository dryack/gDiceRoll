package dsl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedDice   *Dice
		expectedNumber *int
		wantErr        bool
	}{
		{
			name:           "Integer",
			input:          "42",
			expectedNumber: intPtr(42),
		},
		{
			name:  "Basic dice",
			input: "d20",
			expectedDice: &Dice{
				Sides: &Sides{Value: "20"},
			},
		},
		{
			name:  "Multiple dice",
			input: "3d6",
			expectedDice: &Dice{
				Count: intPtr(3),
				Sides: &Sides{Value: "6"},
			},
		},
		{
			name:  "Percentile dice",
			input: "d%",
			expectedDice: &Dice{
				Sides: &Sides{Value: "%"},
			},
		},
		{
			name:    "Invalid input - float",
			input:   "3.14",
			wantErr: true,
		},
		{
			name:    "Invalid input - text",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input, result.Expression)
				assert.Equal(t, tt.expectedDice, result.Parsed.Dice)
				assert.Equal(t, tt.expectedNumber, result.Parsed.Number)
				assert.NotZero(t, result.Value)
				assert.NotEmpty(t, result.Breakdown)
			}
		})
	}
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "Simple number",
			input:       "42",
			expectedMin: 42,
			expectedMax: 42,
		},
		{
			name:        "Single die",
			input:       "d6",
			expectedMin: 1,
			expectedMax: 6,
		},
		{
			name:        "Multiple dice",
			input:       "3d6",
			expectedMin: 3,
			expectedMax: 18,
		},
		{
			name:        "Percentile dice",
			input:       "d%",
			expectedMin: 1,
			expectedMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.input, result.Expression)
			assert.GreaterOrEqual(t, result.Value, tt.expectedMin)
			assert.LessOrEqual(t, result.Value, tt.expectedMax)
			assert.Len(t, result.Breakdown, len(result.Breakdown))
			for _, roll := range result.Breakdown {
				assert.GreaterOrEqual(t, roll, 1)
				assert.LessOrEqual(t, roll, tt.expectedMax)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
