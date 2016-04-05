package main

import (
	"os"

	"github.com/yukirin/goheme/vm"
)

func main() {
	vm.Run(os.Stdin)
}
