package main

import (
	"log"

	"github.com/Patrick-Ivann/AIM-Q/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("❌ Command failed: %v", err)
	}
}
