package main

import (
	"os"

	"github.com/arigustisj/rangkai/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
