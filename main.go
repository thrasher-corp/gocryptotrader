package main

import (
	"log"

	"github.com/thrasher-/gocryptotrader/cmd"
)

func main() {
	if err := cmd.GocryptotraderCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
