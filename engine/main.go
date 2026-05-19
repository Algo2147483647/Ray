package main

import (
	"os"

	"github.com/Algo2147483647/ray/engine/controller"
)

func main() {
	os.Exit(controller.Run(os.Args[1:]))
}
