package dsl

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/dryack/gDiceRoll/core/statistics"
)

// Expression represents the top-level structure of our DSL
type Expression struct {
	Dice   *Dice `@@`
	Number *int  `| @Int`
}

// Dice represents a dice roll in our DSL
type Dice struct {
	Count *int   `@Int?`
	Sides *Sides `"d" @@`
}

// Sides represents the sides of a die
type Sides struct {
	Value string `@("%" | Int)`
}

// ResultSource represents where the result came from
type ResultSource string

const (
	SourceFreshCalculation ResultSource = "fresh_calculation"
	SourceCache            ResultSource = "cache"
	SourceDatabase         ResultSource = "database"
)

type Result struct {
	Expression string      // original expression string
	Parsed     *Expression // parsed expression string
	Value      int
	Breakdown  []int // The individual die rolls
	Statistics *statistics.Result
	Source     ResultSource
}

type CachedResult struct {
	Statistics *statistics.Result `json:"statistics"`
}

// dslLexer defines the lexer rules for our DSL
var dslLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Int", `\d+`},
	{"Dice", `d`},
	{"Percent", `%`},
	{"whitespace", `\s+`},
})

// DiceParser is the exported participle parser for our DSL
var DiceParser = participle.MustBuild[Expression](
	participle.Lexer(dslLexer),
)
