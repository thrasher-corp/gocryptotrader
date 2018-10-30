package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

var (
	priceMap        map[string]float64
	displayCurrency string
)

func printSummary(msg string, amount float64) {
	log.Println()
	log.Println(fmt.Sprintf("%s in USD: $%.2f", msg, amount))

	if displayCurrency != "USD" {
		conv, err := currency.ConvertCurrency(amount, "USD", displayCurrency)
		if err != nil {
			log.Println(err)
		} else {
			symb, err := symbol.GetSymbolByCurrencyName(displayCurrency)
			if err != nil {
				log.Println(fmt.Sprintf("%s in %s: %.2f", msg, displayCurrency, conv))
			} else {
				log.Println(fmt.Sprintf("%s in %s: %s%.2f", msg, displayCurrency, symb, conv))
			}

		}
	}
	log.Println()
}

func getOnlineOfflinePortfolio(coins []portfolio.Coin, online bool) {
	var totals float64
	for _, x := range coins {
		value := priceMap[x.Coin] * x.Balance
		totals += value
		log.Printf("\t%v %v Subtotal: $%.2f Coin percentage: %.2f%%\n", x.Coin,
			x.Balance, value, x.Percentage)
	}
	if !online {
		printSummary("\tOffline balance", totals)
	} else {
		printSummary("\tOnline balance", totals)
	}
}

func main() {
	var inFile, key string

	defaultCfg, err := config.GetFilePath("")
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&inFile, "infile", defaultCfg, "The config input file to process.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: portfolio tool.")

	var cfg config.Config
	err = cfg.LoadConfig(inFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Loaded config file.")

	displayCurrency = cfg.FiatDisplayCurrency
	port := portfolio.Base{}
	port.Seed(cfg.Portfolio)
	result := port.GetPortfolioSummary()

	log.Println("Fetched portfolio data.")

	type PortfolioTemp struct {
		Balance  float64
		Subtotal float64
	}

	cfg.RetrieveConfigCurrencyPairs(true)
	portfolioMap := make(map[string]PortfolioTemp)
	total := float64(0)

	log.Println("Fetching currency data..")
	var fiatCurrencies []string
	for _, y := range result.Totals {
		if currency.IsFiatCurrency(y.Coin) {
			fiatCurrencies = append(fiatCurrencies, y.Coin)
		}
	}
	err = currency.Seed(common.JoinStrings(fiatCurrencies, ","))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Fetched currency data.")
	log.Println("Fetching ticker data and calculating totals..")
	priceMap = make(map[string]float64)
	priceMap["USD"] = 1

	for _, y := range result.Totals {
		pf := PortfolioTemp{}
		pf.Balance = y.Balance
		pf.Subtotal = 0

		if currency.IsDefaultCurrency(y.Coin) {
			if y.Coin != "USD" {
				conv, err := currency.ConvertCurrency(y.Balance, y.Coin, "USD")
				if err != nil {
					log.Println(err)
				} else {
					priceMap[y.Coin] = conv / y.Balance
					pf.Subtotal = conv
				}
			} else {
				pf.Subtotal = y.Balance
			}
		} else {
			bf := bitfinex.Bitfinex{}
			ticker, errf := bf.GetTicker(y.Coin + "USD")
			if errf != nil {
				log.Println(errf)
			} else {
				priceMap[y.Coin] = ticker.Last
				pf.Subtotal = ticker.Last * y.Balance
			}
		}
		portfolioMap[y.Coin] = pf
		total += pf.Subtotal
	}
	log.Println("Done.")
	log.Println()
	log.Println("PORTFOLIO TOTALS:")
	for x, y := range portfolioMap {
		log.Printf("\t%s Amount: %f Subtotal: $%.2f USD (1 %s = $%.2f USD). Percentage of portfolio %.3f%%", x, y.Balance, y.Subtotal, x, y.Subtotal/y.Balance, y.Subtotal/total*100/1)
	}
	printSummary("\tTotal balance", total)

	log.Println("OFFLINE COIN TOTALS:")
	getOnlineOfflinePortfolio(result.Offline, false)

	log.Println("ONLINE COIN TOTALS:")
	getOnlineOfflinePortfolio(result.Online, true)

	log.Println("OFFLINE COIN SUMMARY:")
	var totals float64
	for x, y := range result.OfflineSummary {
		log.Printf("\t%s:", x)
		totals = 0
		for z := range y {
			value := priceMap[x] * y[z].Balance
			totals += value
			log.Printf("\t %s Amount: %f Subtotal: $%.2f Coin percentage: %.2f%%\n",
				y[z].Address, y[z].Balance, value, y[z].Percentage)
		}
		printSummary(fmt.Sprintf("\t %s balance", x), totals)
	}

	log.Println("ONLINE COINS SUMMARY:")
	for x, y := range result.OnlineSummary {
		log.Printf("\t%s:", x)
		totals = 0
		for z, w := range y {
			value := priceMap[z] * w.Balance
			totals += value
			log.Printf("\t %s Amount: %f Subtotal $%.2f Coin percentage: %.2f%%",
				z, w.Balance, value, w.Percentage)
		}
		printSummary("\t Exchange balance", totals)
	}
}
