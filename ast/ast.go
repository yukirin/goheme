package ast

// Token is scheme token
type Token struct {
	TokType int
	Lit     string
}

// Expr is Expression
type Expr interface{}

// Num is number
type Num struct {
	Lit string
}

// Char is charcter
type Char struct {
	Lit string
}

// String is string
type String struct {
	Lit string
}

// Boolean is boolean
type Boolean struct {
	Lit string
}

// List is list
type List []Expr

// Vector is array {
type Vector []Expr

// Symbol is symbol
type Symbol struct {
	Lit string
}
