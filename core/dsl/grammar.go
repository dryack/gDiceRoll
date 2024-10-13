package dsl

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
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
