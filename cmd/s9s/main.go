package main

import (
	"log"
	"os"

	"github.com/jontk/s9s/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
