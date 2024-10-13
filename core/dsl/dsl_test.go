package dsl

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Expression
		wantErr  bool
	}{
		{
			name:  "Integer",
			input: "42",
			expected: &Expression{
				Number: intPtr(42),
			},
		},
		{
			name:  "Basic dice",
			input: "d20",
			expected: &Expression{
				Dice: &Dice{
					Sides: &Sides{Value: "20"},
				},
			},
		},
		{
			name:  "Multiple dice",
			input: "3d6",
			expected: &Expression{
				Dice: &Dice{
					Count: intPtr(3),
					Sides: &Sides{Value: "6"},
				},
			},
		},
		{
			name:  "Percentile dice",
			input: "d%",
			expected: &Expression{
				Dice: &Dice{
					Sides: &Sides{Value: "%"},
				},
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
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name        string
		expr        *Expression
		expectedMin int
		expectedMax int
	}{
		{
			name: "Simple number",
			expr: &Expression{
				Number: intPtr(42),
			},
			expectedMin: 42,
			expectedMax: 42,
		},
		{
			name: "Single die",
			expr: &Expression{
				Dice: &Dice{
					Sides: &Sides{Value: "6"},
				},
			},
			expectedMin: 1,
			expectedMax: 6,
		},
		{
			name: "Multiple dice",
			expr: &Expression{
				Dice: &Dice{
					Count: intPtr(3),
					Sides: &Sides{Value: "6"},
				},
			},
			expectedMin: 3,
			expectedMax: 18,
		},
		{
			name: "Percentile dice",
			expr: &Expression{
				Dice: &Dice{
					Sides: &Sides{Value: "%"},
				},
			},
			expectedMin: 1,
			expectedMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
		})
	}
}

func intPtr(i int) *int {
	return &i
}
