// This tool will initiaite an authenticated request generating an initial
// nonce/timestamp across all supported exchanges and then sleep for 1 minute
// which will allow us to determine if there are any timestamp issues.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()

	log.Println("Checking GCT supported exchanges for timestamp issues on authenticated endpoints.")
	log.Println()

	if !LoadConfig() {
		os.Exit(0)
	}

	var wg sync.WaitGroup
	for i := range SupportedExchanges {
		SupportedExchanges[i].SetDefaults()
		cfg, ok := configs[SupportedExchanges[i].GetName()]
		if !ok {
			cfg.Report = &Report{errors.New("cannot find configuration for exchange")}
			continue
		}

		if !cfg.Enabled {
			cfg.Report = &Report{fmt.Errorf("not enabled in %s", configFilename)}
			continue
		}

		if cfg.Keys == (Keys{}) {
			cfg.Report = &Report{fmt.Errorf("API Keys not set in %s", configFilename)}
			continue
		}

		err := SupportedExchanges[i].Setup(&config.ExchangeConfig{
			Enabled: true,
			Verbose: verbose,
			API: config.APIConfig{
				AuthenticatedSupport: true,
				Credentials: config.APICredentialsConfig{
					Key:       cfg.Key,
					Secret:    cfg.Secret,
					ClientID:  cfg.ClientID,
					PEMKey:    cfg.PEMKey,
					OTPSecret: cfg.OTP,
				},
				CredentialsValidator: &config.APICredentialsValidatorConfig{},
			},
		})
		if err != nil {
			cfg.Report = &Report{err}
			continue
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, exch exchange.IBotExchange, cfg *TimestampConfiguration) {
			defer wg.Done()
			_, err := exch.FetchAccountInfo()
			if err != nil {
				if err != nil {
					cfg.Report = &Report{err}
					return
				}
			}

			log.Printf("%s initial nonce/timestamp created, sleeping for %s\n",
				exch.GetName(),
				defaultSleepTime)

			time.Sleep(defaultSleepTime)
			_, err = exch.FetchAccountInfo()
			if err != nil {
				if err != nil {
					cfg.Report = &Report{err}
					return
				}
			}
			cfg.Report = &Report{}
		}(&wg, SupportedExchanges[i], cfg)
	}

	interupted := make(chan struct{})
	go func(ch chan struct{}) {
		signaler.WaitForInterrupt()
		ch <- struct{}{}
	}(interupted)

	finished := make(chan struct{})
	go func(ch chan struct{}, wg *sync.WaitGroup) {
		wg.Wait()
		ch <- struct{}{}
	}(finished, &wg)

	select {
	case <-interupted:
		log.Println("Interruption caught, shutting down...")
	case <-finished:
	}
	log.Println()
	log.Println("Report:")
	for key, val := range configs {
		if val.Report != nil {
			if val.Report.Error != nil {
				log.Printf("%s has FAILED authentication validation: %s",
					key,
					val.Report.Error)
				continue
			}
			log.Printf("%s has PASSED authentication validation", key)
			continue
		}
		log.Printf("%s was not able to test authentication validation", key)
	}
}

// LoadConfig loads configuration
func LoadConfig() bool {
	fileData, err := ioutil.ReadFile(configFilename)
	if err != nil {
		log.Printf("Config file not found, creating %s configuration file.",
			configFilename)
		log.Println("Please set exchange `API Keys` and set `enabled` to `true` to verify authenticated endpoints.")
		configs = make(map[string]*TimestampConfiguration)
		for i := range SupportedExchanges {
			SupportedExchanges[i].SetDefaults()
			configs[SupportedExchanges[i].GetName()] = &TimestampConfiguration{}
		}
		var data []byte
		data, err = json.MarshalIndent(configs, "", " ")
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(configFilename, data, 0770)
		if err != nil {
			log.Fatal(err)
		}

		return false
	}
	err = json.Unmarshal(fileData, &configs)
	if err != nil {
		log.Fatal(err)
	}
	return true
}
