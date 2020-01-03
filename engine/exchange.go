package engine

import (
	"errors"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitflyer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitmex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bittrex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbasepro"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbene"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinut"
	"github.com/thrasher-corp/gocryptotrader/exchanges/exmo"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gateio"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-corp/gocryptotrader/exchanges/hitbtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-corp/gocryptotrader/exchanges/itbit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kraken"
	"github.com/thrasher-corp/gocryptotrader/exchanges/lakebtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/lbank"
	"github.com/thrasher-corp/gocryptotrader/exchanges/localbitcoins"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okcoin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/poloniex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/yobit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/zb"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")
)

func dryrunParamInteraction(param string) {
	if !Bot.Settings.CheckParamInteraction {
		return
	}

	if !Bot.Settings.EnableDryRun && !flagSet["dryrun"] {
		log.Warnf(log.Global,
			"Command line argument '-%s' induces dry run mode."+
				" Set -dryrun=false if you wish to override this.",
			param)
		Bot.Settings.EnableDryRun = true
	}
}

// CheckExchangeExists returns true whether or not an exchange has already
// been loaded
func CheckExchangeExists(exchName string) bool {
	for x := range Bot.Exchanges {
		if strings.EqualFold(Bot.Exchanges[x].GetName(), exchName) {
			return true
		}
	}
	return false
}

// GetExchangeByName returns an exchange given an exchange name
func GetExchangeByName(exchName string) exchange.IBotExchange {
	for x := range Bot.Exchanges {
		if strings.EqualFold(Bot.Exchanges[x].GetName(), exchName) {
			return Bot.Exchanges[x]
		}
	}
	return nil
}

// ReloadExchange loads an exchange config by name
func ReloadExchange(name string) error {
	if len(Bot.Exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(name) {
		return ErrExchangeNotFound
	}

	exchCfg, err := Bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	e := GetExchangeByName(name)
	e.Setup(exchCfg)
	log.Debugf(log.ExchangeSys, "%s exchange reloaded successfully.\n", name)
	return nil
}

// UnloadExchange unloads an exchange by name
func UnloadExchange(name string) error {
	if len(Bot.Exchanges) == 0 {
		return ErrNoExchangesLoaded
	}

	if !CheckExchangeExists(name) {
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
		if strings.EqualFold(Bot.Exchanges[x].GetName(), name) {
			Bot.Exchanges[x].SetEnabled(false)
			Bot.Exchanges = append(Bot.Exchanges[:x], Bot.Exchanges[x+1:]...)
			return nil
		}
	}

	return ErrExchangeNotFound
}

// LoadExchange loads an exchange by name
func LoadExchange(name string, useWG bool, wg *sync.WaitGroup) error {
	nameLower := strings.ToLower(name)
	var exch exchange.IBotExchange

	if len(Bot.Exchanges) > 0 {
		if CheckExchangeExists(name) {
			return ErrExchangeAlreadyLoaded
		}
	}

	switch nameLower {
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
	case "btc markets":
		exch = new(btcmarkets.BTCMarkets)
	case "btse":
		exch = new(btse.BTSE)
	case "coinbene":
		exch = new(coinbene.Coinbene)
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
	case "itbit":
		exch = new(itbit.ItBit)
	case "kraken":
		exch = new(kraken.Kraken)
	case "lakebtc":
		exch = new(lakebtc.LakeBTC)
	case "lbank":
		exch = new(lbank.Lbank)
	case "localbitcoins":
		exch = new(localbitcoins.LocalBitcoins)
	case "okcoin international":
		exch = new(okcoin.OKCoin)
	case "okex":
		exch = new(okex.OKEX)
	case "poloniex":
		exch = new(poloniex.Poloniex)
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
	exchCfg, err := Bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if Bot.Settings.EnableAllPairs {
		if exchCfg.CurrencyPairs != nil {
			dryrunParamInteraction("enableallpairs")
			assets := exchCfg.CurrencyPairs.GetAssetTypes()
			for x := range assets {
				pairs := exchCfg.CurrencyPairs.GetPairs(assets[x], false)
				exchCfg.CurrencyPairs.StorePairs(assets[x], pairs, true)
			}
		}
	}

	if Bot.Settings.EnableExchangeVerbose {
		dryrunParamInteraction("exchangeverbose")
		exchCfg.Verbose = true
	}

	if Bot.Settings.EnableExchangeWebsocketSupport {
		dryrunParamInteraction("exchangewebsocketsupport")
		if exchCfg.Features != nil {
			if exchCfg.Features.Supports.Websocket {
				exchCfg.Features.Enabled.Websocket = true
			}
		}
	}

	if Bot.Settings.EnableExchangeAutoPairUpdates {
		dryrunParamInteraction("exchangeautopairupdates")
		if exchCfg.Features != nil {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = true
			}
		}
	}

	if Bot.Settings.DisableExchangeAutoPairUpdates {
		dryrunParamInteraction("exchangedisableautopairupdates")
		if exchCfg.Features != nil {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = false
			}
		}
	}

	if Bot.Settings.ExchangeHTTPUserAgent != "" {
		dryrunParamInteraction("exchangehttpuseragent")
		exchCfg.HTTPUserAgent = Bot.Settings.ExchangeHTTPUserAgent
	}

	if Bot.Settings.ExchangeHTTPProxy != "" {
		dryrunParamInteraction("exchangehttpproxy")
		exchCfg.ProxyAddress = Bot.Settings.ExchangeHTTPProxy
	}

	if Bot.Settings.ExchangeHTTPTimeout != exchange.DefaultHTTPTimeout {
		dryrunParamInteraction("exchangehttptimeout")
		exchCfg.HTTPTimeout = Bot.Settings.ExchangeHTTPTimeout
	}

	if Bot.Settings.EnableExchangeHTTPDebugging {
		dryrunParamInteraction("exchangehttpdebugging")
		exchCfg.HTTPDebugging = Bot.Settings.EnableExchangeHTTPDebugging
	}

	if Bot.Settings.EnableAllExchanges {
		dryrunParamInteraction("enableallexchanges")
	}

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		return err
	}

	Bot.Exchanges = append(Bot.Exchanges, exch)

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
	exchanges := Bot.Config.GetAllExchangeConfigs()
	for x := range exchanges {
		exch := exchanges[x]
		if CheckExchangeExists(exch.Name) {
			e := GetExchangeByName(exch.Name)
			if e == nil {
				log.Errorln(log.ExchangeSys, ErrExchangeNotFound)
				continue
			}

			err := ReloadExchange(exch.Name)
			if err != nil {
				log.Errorf(log.ExchangeSys, "ReloadExchange %s failed: %s\n", exch.Name, err)
				continue
			}

			if !e.IsEnabled() {
				UnloadExchange(exch.Name)
				continue
			}
			return
		}
		if !exch.Enabled && !Bot.Settings.EnableAllExchanges {
			log.Debugf(log.ExchangeSys, "%s: Exchange support: Disabled\n", exch.Name)
			continue
		}
		err := LoadExchange(exch.Name, true, &wg)
		if err != nil {
			log.Errorf(log.ExchangeSys, "LoadExchange %s failed: %s\n", exch.Name, err)
			continue
		}
		log.Debugf(log.ExchangeSys,
			"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
			exch.Name,
			common.IsEnabled(exch.API.AuthenticatedSupport),
			common.IsEnabled(exch.Verbose),
		)
	}
	wg.Wait()
}
