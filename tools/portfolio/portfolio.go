package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
)

func main() {
	var inFile, key string
	var err error
	flag.StringVar(&inFile, "infile", "config.dat", "The config input file to process.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: portfolio tool.")

	var data []byte
	var cfg config.Config

	data, err = common.ReadFile(inFile)
	if err != nil {
		log.Fatalf("Unable to read input file %s. Error: %s.", inFile, err)
	}

	if config.ConfirmECS(data) {
		if key == "" {
			result, err := config.PromptForConfigKey()
			if err != nil {
				log.Fatal("Unable to obtain encryption/decryption key.")
			}
			key = string(result)
		}
		data, err = config.DecryptConfigFile(data, []byte(key))
		if err != nil {
			log.Fatalf("Unable to decrypt config data. Error: %s.", err)
		}

	}
	err = config.ConfirmConfigJSON(data, &cfg)
	if err != nil {
		log.Fatal("File isn't in JSON format")
	}

	result := make(map[string]float64)
	for _, x := range cfg.Portfolio.Addresses {
		balance, ok := result[x.CoinType]
		if !ok {
			result[x.CoinType] = x.Balance
		} else {
			result[x.CoinType] = x.Balance + balance
		}
	}

	type Portfolio struct {
		Balance  float64
		Subtotal float64
	}

	stuff := make(map[string]Portfolio)
	total := float64(0)

	for x, y := range result {
		if x == "ETH" {
			y = y / common.WEI_PER_ETHER
		}

		pf := Portfolio{}
		pf.Balance = y
		pf.Subtotal = 0

		bf := bitfinex.Bitfinex{}
		ticker, err := bf.GetTicker(x+"USD", url.Values{})
		if err != nil {
			log.Println(err)
		} else {
			pf.Subtotal = ticker.Last * y
		}
		stuff[x] = pf
		total += pf.Subtotal
	}

	for x, y := range stuff {
		log.Printf("%s %f subtotal: %f USD. Percentage of portfolio %f", x, y.Balance, y.Subtotal, y.Subtotal/total*100/1)
	}

	log.Printf("Total balance in USD: %f.\n", total)

	conv, err := currency.ConvertCurrency(total, "USD", "AUD")
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("Total balance in AUD: %f.\n", conv)
	}
}
