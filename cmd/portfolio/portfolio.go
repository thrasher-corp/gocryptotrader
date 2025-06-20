package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

var (
	priceMap        map[*currency.Item]float64
	displayCurrency currency.Code
)

func printSummary(msg string, amount float64) {
	log.Println()
	log.Printf("%s in USD: $%.2f", msg, amount)

	if !displayCurrency.Equal(currency.USD) {
		conv, err := currency.ConvertFiat(amount, currency.USD, displayCurrency)
		if err != nil {
			log.Println(err)
		} else {
			var symb string
			symb, err = currency.GetSymbolByCurrencyName(displayCurrency)
			if err != nil {
				log.Printf("%s in %s: %.2f",
					msg,
					displayCurrency,
					conv)
			} else {
				log.Printf("%s in %s: %s%.2f",
					msg,
					displayCurrency,
					symb,
					conv)
			}
		}
	}
}

func getOnlineOfflinePortfolio(coins []portfolio.Coin, online bool) {
	var totals float64
	for _, x := range coins {
		value := priceMap[x.Coin.Item] * x.Balance
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
	flag.StringVar(&inFile, "config", config.DefaultFilePath(), "The config input file to process.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: portfolio tool.")

	var cfg config.Config
	err := cfg.LoadConfig(inFile, true)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println("Loaded config file.")

	displayCurrency = cfg.Currency.FiatDisplayCurrency
	port := cfg.Portfolio
	result := port.GetPortfolioSummary()

	log.Println("Fetched portfolio data.")

	type PortfolioTemp struct {
		Balance  float64
		Subtotal float64
	}

	portfolioMap := make(map[*currency.Item]PortfolioTemp)
	total := float64(0)

	log.Println("Fetching currency data..")
	var fiatCurrencies []currency.Code
	for _, y := range result.Totals {
		if y.Coin.IsFiatCurrency() {
			fiatCurrencies = append(fiatCurrencies, y.Coin)
		}
	}
	err = currency.SeedForeignExchangeData(fiatCurrencies)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	log.Println("Fetched currency data.")
	log.Println("Fetching ticker data and calculating totals..")
	priceMap = make(map[*currency.Item]float64)
	priceMap[currency.USD.Item] = 1

	for _, y := range result.Totals {
		pf := PortfolioTemp{}
		pf.Balance = y.Balance
		pf.Subtotal = 0

		if y.Coin.IsFiatCurrency() {
			if !y.Coin.Equal(currency.USD) {
				conv, err := currency.ConvertFiat(y.Balance, y.Coin, currency.USD)
				if err != nil {
					log.Println(err)
				} else {
					priceMap[y.Coin.Item] = conv / y.Balance
					pf.Subtotal = conv
				}
			} else {
				pf.Subtotal = y.Balance
			}
		} else {
			bf := bitfinex.Exchange{}
			bf.SetDefaults()
			bf.Verbose = false
			pair := "t" + y.Coin.String() + currency.USD.String()
			ticker, errf := bf.GetTicker(context.TODO(), pair)
			if errf != nil {
				log.Println(errf)
			} else {
				priceMap[y.Coin.Item] = ticker.Last
				pf.Subtotal = ticker.Last * y.Balance
			}
		}
		portfolioMap[y.Coin.Item] = pf
		total += pf.Subtotal
	}
	log.Println("Done.")
	log.Println()
	log.Println("PORTFOLIO TOTALS:")
	for x, y := range portfolioMap {
		code := currency.Code{Item: x}
		log.Printf("\t%s Amount: %f Subtotal: $%.2f USD (1 %s = $%.2f USD). Percentage of portfolio %.3f%%",
			code,
			y.Balance,
			y.Subtotal,
			code,
			y.Subtotal/y.Balance,
			y.Subtotal/total*100/1)
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
			value := priceMap[x.Item] * y[z].Balance
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
			value := priceMap[z.Item] * w.Balance
			totals += value
			log.Printf("\t %s Amount: %f Subtotal $%.2f Coin percentage: %.2f%%",
				z, w.Balance, value, w.Percentage)
		}
		printSummary("\t Exchange balance", totals)
	}
}
