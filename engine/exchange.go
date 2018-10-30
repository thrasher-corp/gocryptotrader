package engine

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/anx"
	"github.com/thrasher-/gocryptotrader/exchanges/binance"
	"github.com/thrasher-/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-/gocryptotrader/exchanges/bitflyer"
	"github.com/thrasher-/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-/gocryptotrader/exchanges/bitmex"
	"github.com/thrasher-/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-/gocryptotrader/exchanges/btcc"
	"github.com/thrasher-/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-/gocryptotrader/exchanges/coinbasepro"
	"github.com/thrasher-/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-/gocryptotrader/exchanges/exmo"
	"github.com/thrasher-/gocryptotrader/exchanges/gateio"
	"github.com/thrasher-/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-/gocryptotrader/exchanges/hitbtc"
	"github.com/thrasher-/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-/gocryptotrader/exchanges/huobihadax"
	"github.com/thrasher-/gocryptotrader/exchanges/itbit"
	"github.com/thrasher-/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-/gocryptotrader/exchanges/lakebtc"
	"github.com/thrasher-/gocryptotrader/exchanges/liqui"
	"github.com/thrasher-/gocryptotrader/exchanges/localbitcoins"
	"github.com/thrasher-/gocryptotrader/exchanges/okcoin"
	"github.com/thrasher-/gocryptotrader/exchanges/okex"
	"github.com/thrasher-/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-/gocryptotrader/exchanges/wex"
	"github.com/thrasher-/gocryptotrader/exchanges/yobit"
	"github.com/thrasher-/gocryptotrader/exchanges/zb"
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
	for x := range Bot.Exchanges {
		if common.StringToLower(Bot.Exchanges[x].GetName()) == common.StringToLower(exchName) {
			return true
		}
	}
	return false
}

// GetExchangeByName returns an exchange given an exchange name
func GetExchangeByName(exchName string) exchange.IBotExchange {
	for x := range Bot.Exchanges {
		if common.StringToLower(Bot.Exchanges[x].GetName()) == common.StringToLower(exchName) {
			return Bot.Exchanges[x]
		}
	}
	return nil
}

// ReloadExchange loads an exchange config by name
func ReloadExchange(name string) error {
	nameLower := common.StringToLower(name)

	if len(Bot.Exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(nameLower) {
		return ErrExchangeNotFound
	}

	exchCfg, err := Bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	e := GetExchangeByName(nameLower)
	e.Setup(exchCfg)
	log.Printf("%s exchange reloaded successfully.\n", name)
	return nil
}

// UnloadExchange unloads an exchange by name
func UnloadExchange(name string) error {
	nameLower := common.StringToLower(name)

	if len(Bot.Exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(nameLower) {
		return ErrExchangeNotFound
	}

	exchCfg, err := Bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	err = Bot.Config.UpdateExchangeConfig(exchCfg)
	if err != nil {
		return err
	}

	for x := range Bot.Exchanges {
		if Bot.Exchanges[x].GetName() == name {
			Bot.Exchanges[x].SetEnabled(false)
			Bot.Exchanges = append(Bot.Exchanges[:x], Bot.Exchanges[x+1:]...)
			return nil
		}
	}

	return ErrExchangeNotFound
}

// LoadExchange loads an exchange by name
func LoadExchange(name string, useWG bool, wg *sync.WaitGroup) error {
	nameLower := common.StringToLower(name)
	var exch exchange.IBotExchange

	if len(Bot.Exchanges) > 0 {
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
	case "bitmex":
		exch = new(bitmex.Bitmex)
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
	case "coinbasepro":
		exch = new(coinbasepro.CoinbasePro)
	case "gateio":
		exch = new(gateio.Gateio)
	case "gemini":
		exch = new(gemini.Gemini)
	case "hitbtc":
		exch = new(hitbtc.HitBTC)
	case "huobi":
		exch = new(huobi.HUOBI)
	case "huobihadax":
		exch = new(huobihadax.HUOBIHADAX)
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
	case "zb":
		exch = new(zb.ZB)
	default:
		return ErrExchangeNotFound
	}

	if exch == nil {
		return ErrExchangeFailedToLoad
	}

	exch.SetDefaults()
	Bot.Exchanges = append(Bot.Exchanges, exch)
	exchCfg, err := Bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if Bot.Settings.EnableAllPairs {
		exchCfg.EnabledPairs = exchCfg.AvailablePairs
	}

	if Bot.Settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}

	if Bot.Settings.ExchangeHTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = Bot.Settings.ExchangeHTTPUserAgent
	}

	if Bot.Settings.ExchangeHTTPProxy != "" {
		exchCfg.ProxyAddress = Bot.Settings.ExchangeHTTPProxy
	}

	if Bot.Settings.ExchangeHTTPTimeout != time.Duration(time.Second*15) {
		exchCfg.HTTPTimeout = Bot.Settings.ExchangeHTTPTimeout
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

// SetupExchanges sets up the exchanges used by the Bot
func SetupExchanges() {
	var wg sync.WaitGroup
	for _, exch := range Bot.Config.Exchanges {
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
		if !exch.Enabled && !Bot.Settings.EnableAllExchanges {
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
