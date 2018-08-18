package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	"github.com/thrasher-/gocryptotrader/decimal"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

var (
	priceMap        map[string]decimal.Decimal
	displayCurrency string
)

func printSummary(msg string, amount decimal.Decimal) {
	log.Println()
	log.Println(fmt.Sprintf("%s in USD: $%v", msg, amount.StringFixed(2)))

	if displayCurrency != "USD" {
		conv, err := currency.ConvertCurrency(amount, "USD", displayCurrency)
		if err != nil {
			log.Println(err)
		} else {
			symb, err := symbol.GetSymbolByCurrencyName(displayCurrency)
			if err != nil {
				log.Println(fmt.Sprintf("%s in %s: %v", msg, displayCurrency, conv.StringFixed(2)))
			} else {
				log.Println(fmt.Sprintf("%s in %s: %s%v", msg, displayCurrency, symb, conv.StringFixed(2)))
			}

		}
	}
	log.Println()
}

func getOnlineOfflinePortfolio(coins []portfolio.Coin, online bool) {
	var totals decimal.Decimal
	for _, x := range coins {
		value := priceMap[x.Coin].Mul(x.Balance)
		totals = totals.Add(value)
		log.Printf("\t%v %v Subtotal: $%v Coin percentage: %v%%\n", x.Coin,
			x.Balance.StringFixed(2), value.StringFixed(2), x.Percentage.StringFixed(2))
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
	port.SeedPortfolio(cfg.Portfolio)
	result := port.GetPortfolioSummary()

	log.Println("Fetched portfolio data.")

	type PortfolioTemp struct {
		Balance  decimal.Decimal
		Subtotal decimal.Decimal
	}

	cfg.RetrieveConfigCurrencyPairs(true)
	portfolioMap := make(map[string]PortfolioTemp)
	total := decimal.Zero

	log.Println("Fetching currency data..")
	var fiatCurrencies []string
	for _, y := range result.Totals {
		if currency.IsFiatCurrency(y.Coin) {
			fiatCurrencies = append(fiatCurrencies, y.Coin)
		}
	}
	err = currency.SeedCurrencyData(common.JoinStrings(fiatCurrencies, ","))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Fetched currency data.")
	log.Println("Fetching ticker data and calculating totals..")
	priceMap = make(map[string]decimal.Decimal)
	priceMap["USD"] = decimal.One

	for _, y := range result.Totals {
		pf := PortfolioTemp{}
		pf.Balance = y.Balance
		pf.Subtotal = decimal.Zero

		if currency.IsDefaultCurrency(y.Coin) {
			if y.Coin != "USD" {
				conv, err := currency.ConvertCurrency(y.Balance, y.Coin, "USD")
				if err != nil {
					log.Println(err)
				} else {
					priceMap[y.Coin] = conv.Div(y.Balance)
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
				pf.Subtotal = ticker.Last.Mul(y.Balance)
			}
		}
		portfolioMap[y.Coin] = pf
		total = total.Add(pf.Subtotal)
	}
	log.Println("Done.")
	log.Println()
	log.Println("PORTFOLIO TOTALS:")
	for x, y := range portfolioMap {
		log.Printf("\t%s Amount: %v Subtotal: $%v USD (1 %s = $%v USD). Percentage of portfolio %v%%", x, y.Balance.String(), y.Subtotal.StringFixed(2), x, y.Subtotal.Div(y.Balance).StringFixed(2), y.Subtotal.Percentage(total).StringFixed(3))
	}
	printSummary("\tTotal balance", total)

	log.Println("OFFLINE COIN TOTALS:")
	getOnlineOfflinePortfolio(result.Offline, false)

	log.Println("ONLINE COIN TOTALS:")
	getOnlineOfflinePortfolio(result.Online, true)

	log.Println("OFFLINE COIN SUMMARY:")
	var totals decimal.Decimal
	for x, y := range result.OfflineSummary {
		log.Printf("\t%s:", x)
		totals = decimal.Zero
		for z := range y {
			value := priceMap[x].Mul(y[z].Balance)
			totals = totals.Add(value)
			log.Printf("\t %s Amount: %v Subtotal: $%v Coin percentage: %v%%\n",
				y[z].Address, y[z].Balance.String(), value.StringFixed(2), y[z].Percentage.StringFixed(3))
		}
		printSummary(fmt.Sprintf("\t %s balance", x), totals)
	}

	log.Println("ONLINE COINS SUMMARY:")
	for x, y := range result.OnlineSummary {
		log.Printf("\t%s:", x)
		totals = decimal.Zero
		for z, w := range y {
			value := priceMap[z].Mul(w.Balance)
			totals = totals.Add(value)
			log.Printf("\t %s Amount: %v Subtotal $%v Coin percentage: %v%%",
				z, w.Balance.String(), value.StringFixed(2), w.Percentage.StringFixed(2))
		}
		printSummary("\t Exchange balance", totals)
	}
}
