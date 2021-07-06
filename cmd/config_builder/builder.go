package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func main() {
	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}

	log.Printf("Loading exchanges..")
	var wg sync.WaitGroup
	for x := range exchange.Exchanges {
		name := exchange.Exchanges[x]
		err = engine.Bot.LoadExchange(name, &wg)
		if err != nil {
			log.Printf("Failed to load exchange %s. Err: %s", name, err)
			continue
		}
	}
	wg.Wait()
	log.Println("Done.")

	var cfgs []config.ExchangeConfig
	exchanges := engine.Bot.GetExchanges()
	for x := range exchanges {
		var cfg *config.ExchangeConfig
		cfg, err = exchanges[x].GetDefaultConfig()
		if err != nil {
			log.Printf("Failed to get exchanges default config. Err: %s", err)
			continue
		}
		log.Printf("Adding %s", exchanges[x].GetName())
		cfgs = append(cfgs, *cfg)
	}

	data, err := json.MarshalIndent(cfgs, "", " ")
	if err != nil {
		log.Fatalf("Unable to marshal cfgs. Err: %s", err)
	}

	log.Println(string(data))
}
