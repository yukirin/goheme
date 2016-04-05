%{
	package parser

	import (
		"fmt"
		"io"
		"github.com/yukirin/goheme/ast"
	)
%}

%union{
	tok ast.Token
	expr ast.Expr
}

%type<expr> program
%token<tok> ExprType BooleanType CharType StringType NumberType SymbolType ShortStatType VectorType

%%
program
	: '(' SymbolType NumberType NumberType ')'
	{
		$$ = ast.List{ast.Symbol{$2.Lit}, ast.Num{$3.Lit}, ast.Num{$4.Lit}}
		if l, ok := yylex.(*LexerWrapper); ok {
			l.Ast = $$
		}
	}
%%

func Parse(r io.Reader) (ast.List, error) {
	lexer := LexerWrapper{
		Lexer: NewLexer(r),
	}
	if yyParse(&lexer) != 0 {
		return nil, fmt.Errorf("parse error")
	}

	return lexer.Ast.(ast.List), nil
}
