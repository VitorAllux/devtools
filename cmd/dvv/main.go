package main

import (
	"os"

	"github.com/VitorAllux/devtools/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
