package main

import (
	"os"

	"github.com/VitorAllux/devtools/internal/app"
)

func main() {
	os.Exit(exitCode(os.Args[1:]))
}

func exitCode(args []string) int {
	return app.Run(args)
}
