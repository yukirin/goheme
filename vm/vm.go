package vm

import (
	"fmt"
	"io"

	"github.com/yukirin/goheme/parser"
)

func Run(r io.Reader) {
	ast, err := parser.Parse(r)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(ast)
}
