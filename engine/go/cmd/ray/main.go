package main

import (
	"os"

	"github.com/Algo2147483647/ray/engine/go/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
