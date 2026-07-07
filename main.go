package main

import (
	"os"

	"github.com/tymbaca/spin/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
