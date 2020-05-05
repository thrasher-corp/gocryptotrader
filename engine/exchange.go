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
	"github.com/thrasher-corp/gocryptotrader/log"
)

// vars related to exchange functions
var (
	ErrNoExchangesLoaded     = errors.New("no exchanges have been loaded")
	ErrExchangeNotFound      = errors.New("exchange not found")
	ErrExchangeAlreadyLoaded = errors.New("exchange already loaded")
	ErrExchangeFailedToLoad  = errors.New("exchange failed to load")
)

type exchangeManager struct {
	m         sync.Mutex
	exchanges map[string]exchange.IBotExchange
}

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

func (e *exchangeManager) add(exch exchange.IBotExchange) {
	e.m.Lock()
	if e.exchanges == nil {
		e.exchanges = make(map[string]exchange.IBotExchange)
	}
	e.exchanges[strings.ToLower(exch.GetName())] = exch
	e.m.Unlock()
}

func (e *exchangeManager) getExchanges() []exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}

	e.m.Lock()
	defer e.m.Unlock()
	var exchs []exchange.IBotExchange
	for x := range e.exchanges {
		exchs = append(exchs, e.exchanges[x])
	}
	return exchs
}

func (e *exchangeManager) removeExchange(exchName string) error {
	if e.Len() == 0 {
		return ErrNoExchangesLoaded
	}
	exch := e.getExchangeByName(exchName)
	if exch == nil {
		return ErrExchangeNotFound
	}
	e.m.Lock()
	defer e.m.Unlock()
	delete(e.exchanges, strings.ToLower(exchName))
	log.Infof(log.ExchangeSys, "%s exchange unloaded successfully.\n", exchName)
	return nil
}

func (e *exchangeManager) getExchangeByName(exchangeName string) exchange.IBotExchange {
	if e.Len() == 0 {
		return nil
	}
	e.m.Lock()
	defer e.m.Unlock()
	exch, ok := e.exchanges[strings.ToLower(exchangeName)]
	if !ok {
		return nil
	}
	return exch
}

func (e *exchangeManager) Len() int {
	e.m.Lock()
	defer e.m.Unlock()
	return len(e.exchanges)
}

func (e *exchangeManager) unloadExchange(exchangeName string) error {
	exchCfg, err := Bot.Config.GetExchangeConfig(exchangeName)
	if err != nil {
		return err
	}

	err = e.removeExchange(exchangeName)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	return nil
}

// GetExchangeByName returns an exchange given an exchange name
func GetExchangeByName(exchName string) exchange.IBotExchange {
	return Bot.exchangeManager.getExchangeByName(exchName)
}

// UnloadExchange unloads an exchange by name
func UnloadExchange(exchName string) error {
	return Bot.exchangeManager.unloadExchange(exchName)
}

// GetExchanges retrieves the loaded exchanges
func GetExchanges() []exchange.IBotExchange {
	return Bot.exchangeManager.getExchanges()
}

// LoadExchange loads an exchange by name
func LoadExchange(name string, useWG bool, wg *sync.WaitGroup) error {
	nameLower := strings.ToLower(name)
	var exch exchange.IBotExchange

	if Bot.exchangeManager.getExchangeByName(nameLower) != nil {
		return ErrExchangeAlreadyLoaded
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

	if Bot.Settings.HTTPUserAgent != "" {
		dryrunParamInteraction("httpuseragent")
		exchCfg.HTTPUserAgent = Bot.Settings.HTTPUserAgent
	}

	if Bot.Settings.HTTPProxy != "" {
		dryrunParamInteraction("httpproxy")
		exchCfg.ProxyAddress = Bot.Settings.HTTPProxy
	}

	if Bot.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		dryrunParamInteraction("httptimeout")
		exchCfg.HTTPTimeout = Bot.Settings.HTTPTimeout
	}

	if Bot.Settings.EnableExchangeHTTPDebugging {
		dryrunParamInteraction("exchangehttpdebugging")
		exchCfg.HTTPDebugging = Bot.Settings.EnableExchangeHTTPDebugging
	}

	if Bot.Settings.EnableAllExchanges {
		dryrunParamInteraction("enableallexchanges")
	}

	if !Bot.Settings.EnableExchangeHTTPRateLimiter {
		log.Warnf(log.ExchangeSys,
			"Loaded exchange %s rate limiting has been turned off.\n",
			exch.GetName(),
		)
		err = exch.DisableRateLimiter()
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Loaded exchange %s rate limiting cannot be turned off: %s.\n",
				exch.GetName(),
				err,
			)
		}
	}

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		exchCfg.Enabled = false
		return err
	}

	Bot.exchangeManager.add(exch)

	base := exch.GetBase()
	if base.API.AuthenticatedSupport ||
		base.API.AuthenticatedWebsocketSupport {
		err = exch.ValidateCredentials()
		if err != nil {
			log.Warnf(log.ExchangeSys,
				"%s: Cannot validate credentials, authenticated support has been disabled, Error: %s\n",
				base.Name,
				err)
			base.API.AuthenticatedSupport = false
			base.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.AuthenticatedSupport = false
			exchCfg.API.AuthenticatedWebsocketSupport = false
		}
	}

	if useWG {
		exch.Start(wg)
	} else {
		tempWG := sync.WaitGroup{}
		exch.Start(&tempWG)
		tempWG.Wait()
	}

	return nil
}

// SetupExchanges sets up the exchanges used by the Bot
func SetupExchanges() {
	var wg sync.WaitGroup
	configs := Bot.Config.GetAllExchangeConfigs()
	for x := range configs {
		if !configs[x].Enabled && !Bot.Settings.EnableAllExchanges {
			log.Debugf(log.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}
		err := LoadExchange(configs[x].Name, true, &wg)
		if err != nil {
			log.Errorf(log.ExchangeSys, "LoadExchange %s failed: %s\n", configs[x].Name, err)
			continue
		}
		log.Debugf(log.ExchangeSys,
			"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
			configs[x].Name,
			common.IsEnabled(configs[x].API.AuthenticatedSupport),
			common.IsEnabled(configs[x].Verbose),
		)
	}
	wg.Wait()
}
