package main

import (
	"errors"
	"log"
	"sync"

	"gocryptotrader/exchanges/anx"
	"gocryptotrader/exchanges/binance"
	"gocryptotrader/exchanges/bitfinex"
	"gocryptotrader/exchanges/bitflyer"
	"gocryptotrader/exchanges/bithumb"
	"gocryptotrader/exchanges/bitstamp"
	"gocryptotrader/exchanges/bittrex"
	"gocryptotrader/exchanges/btcc"
	"gocryptotrader/exchanges/btcmarkets"
	"gocryptotrader/exchanges/coinut"
	"gocryptotrader/exchanges/exmo"
	"gocryptotrader/exchanges/gdax"
	"gocryptotrader/exchanges/gemini"
	"gocryptotrader/exchanges/hitbtc"
	"gocryptotrader/exchanges/huobi"
	"gocryptotrader/exchanges/itbit"
	"gocryptotrader/exchanges/kraken"
	"gocryptotrader/exchanges/lakebtc"
	"gocryptotrader/exchanges/liqui"
	"gocryptotrader/exchanges/localbitcoins"
	"gocryptotrader/exchanges/okcoin"
	"gocryptotrader/exchanges/okex"
	"gocryptotrader/exchanges/poloniex"
	"gocryptotrader/exchanges/wex"
	"gocryptotrader/exchanges/yobit"

	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")
)

// CheckExchangeExists returns true whether or not an exchange has already
// been loaded
func CheckExchangeExists(exchName string) bool {
	for x := range bot.exchanges {
		if common.StringToLower(bot.exchanges[x].GetName()) == common.StringToLower(exchName) {
			return true
		}
	}
	return false
}

// GetExchangeByName returns an exchange given an exchange name
func GetExchangeByName(exchName string) exchange.IBotExchange {
	for x := range bot.exchanges {
		if common.StringToLower(bot.exchanges[x].GetName()) == common.StringToLower(exchName) {
			return bot.exchanges[x]
		}
	}
	return nil
}

// ReloadExchange loads an exchange config by name
func ReloadExchange(name string) error {
	nameLower := common.StringToLower(name)

	if len(bot.exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(nameLower) {
		return ErrExchangeNotFound
	}

	exchCfg, err := bot.config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	e := GetExchangeByName(nameLower)
	e.Setup(exchCfg)
	log.Printf("%s exchange reloaded successfully.\n", name)
	return nil
}

// UnloadExchange unloads an exchange by
func UnloadExchange(name string) error {
	nameLower := common.StringToLower(name)

	if len(bot.exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(nameLower) {
		return ErrExchangeNotFound
	}

	exchCfg, err := bot.config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	err = bot.config.UpdateExchangeConfig(exchCfg)
	if err != nil {
		return err
	}

	for x := range bot.exchanges {
		if bot.exchanges[x].GetName() == name {
			bot.exchanges[x].SetEnabled(false)
			bot.exchanges = append(bot.exchanges[:x], bot.exchanges[x+1:]...)
			return nil
		}
	}

	return ErrExchangeNotFound
}

// LoadExchange loads an exchange by name
func LoadExchange(name string, useWG bool, wg *sync.WaitGroup) error {
	nameLower := common.StringToLower(name)
	var exch exchange.IBotExchange

	if len(bot.exchanges) > 0 {
		if CheckExchangeExists(nameLower) {
			return ErrExchangeAlreadyLoaded
		}
	}

	switch nameLower {
	case "anx":
		exch = new(anx.ANX)
	case "binance":
		exch = new(binance.Binance)
	case "bitfinex":
		exch = new(bitfinex.Bitfinex)
	case "bitflyer":
		exch = new(bitflyer.Bitflyer)
	case "bithumb":
		exch = new(bithumb.Bithumb)
	case "bitstamp":
		exch = new(bitstamp.Bitstamp)
	case "bittrex":
		exch = new(bittrex.Bittrex)
	case "btcc":
		exch = new(btcc.BTCC)
	case "btc markets":
		exch = new(btcmarkets.BTCMarkets)
	case "coinut":
		exch = new(coinut.COINUT)
	case "exmo":
		exch = new(exmo.EXMO)
	case "gdax":
		exch = new(gdax.GDAX)
	case "gemini":
		exch = new(gemini.Gemini)
	case "hitbtc":
		exch = new(hitbtc.HitBTC)
	case "huobi":
		exch = new(huobi.HUOBI)
	case "itbit":
		exch = new(itbit.ItBit)
	case "kraken":
		exch = new(kraken.Kraken)
	case "lakebtc":
		exch = new(lakebtc.LakeBTC)
	case "liqui":
		exch = new(liqui.Liqui)
	case "localbitcoins":
		exch = new(localbitcoins.LocalBitcoins)
	case "okcoin china":
		exch = new(okcoin.OKCoin)
	case "okcoin international":
		exch = new(okcoin.OKCoin)
	case "okex":
		exch = new(okex.OKEX)
	case "poloniex":
		exch = new(poloniex.Poloniex)
	case "wex":
		exch = new(wex.WEX)
	case "yobit":
		exch = new(yobit.Yobit)
	default:
		return ErrExchangeNotFound
	}

	if exch == nil {
		return ErrExchangeFailedToLoad
	}

	exch.SetDefaults()
	bot.exchanges = append(bot.exchanges, exch)
	exchCfg, err := bot.config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	exchCfg.Enabled = true
	exch.Setup(exchCfg)

	if useWG {
		exch.Start(wg)
	} else {
		wg := sync.WaitGroup{}
		exch.Start(&wg)
		wg.Wait()
	}
	return nil
}

// SetupExchanges sets up the exchanges used by the bot
func SetupExchanges() {
	var wg sync.WaitGroup
	for _, exch := range bot.config.Exchanges {
		if CheckExchangeExists(exch.Name) {
			e := GetExchangeByName(exch.Name)
			if e == nil {
				log.Println(ErrExchangeNotFound)
				continue
			}

			err := ReloadExchange(exch.Name)
			if err != nil {
				log.Printf("ReloadExchange %s failed: %s", exch.Name, err)
				continue
			}

			if !e.IsEnabled() {
				UnloadExchange(exch.Name)
				continue
			}
			return

		}
		if !exch.Enabled {
			log.Printf("%s: Exchange support: Disabled", exch.Name)
			continue
		} else {
			err := LoadExchange(exch.Name, true, &wg)
			if err != nil {
				log.Printf("LoadExchange %s failed: %s", exch.Name, err)
				continue
			}
		}
		log.Printf(
			"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
			exch.Name,
			common.IsEnabled(exch.AuthenticatedAPISupport),
			common.IsEnabled(exch.Verbose),
		)
	}
	wg.Wait()
}
