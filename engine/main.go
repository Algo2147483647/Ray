package main

import (
	"os"

	"github.com/Algo2147483647/ray/engine/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
