package okex

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (o *OKEX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	o.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	err := o.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = o.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults method assignes the default values for OKEX
func (o *OKEX) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = okExExchangeName
	o.Enabled = true
	o.Verbose = true
	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true

	o.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
			assets.AssetTypeFutures,
			assets.AssetTypePerpetualSwap,
			assets.AssetTypeIndex,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: false,
			Delimiter: "_",
		},

		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
		},
	}

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	o.Requester = request.New(o.Name,
		request.NewRateLimit(time.Second, okExAuthRate),
		request.NewRateLimit(time.Second, okExUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
	)

	o.API.Endpoints.URLDefault = okExAPIURL
	o.API.Endpoints.URL = okExAPIURL
	o.API.Endpoints.WebsocketURL = OkExWebsocketURL
	o.APIVersion = okExAPIVersion
	o.WebsocketInit()
	o.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketKlineSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketSubscribeSupported |
		exchange.WebsocketUnsubscribeSupported
}

// Start starts the OKGroup go routine
func (o *OKEX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKEX) Run() {
	if o.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
	}

	if !o.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := o.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", o.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKEX) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	var pairs []string
	switch asset {
	case assets.AssetTypeSpot:
		prods, err := o.GetSpotTokenPairDetails()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
		}
		return pairs, nil
	case assets.AssetTypeFutures:
		prods, err := o.GetFuturesContractInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, prods[x].UnderlyingIndex+prods[x].QuoteCurrency+"_"+prods[x].Delivery)
		}
		return pairs, nil

	case assets.AssetTypePerpetualSwap:
		prods, err := o.GetAllSwapTokensInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, prods[x].InstrumentID)
		}
		return pairs, nil
	case assets.AssetTypeIndex:
		return []string{"BTC_USD"}, nil
	}

	return nil, fmt.Errorf("%s invalid asset type", o.Name)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKEX) UpdateTradablePairs(forceUpdate bool) error {
	for x := range o.CurrencyPairs.AssetTypes {
		a := o.CurrencyPairs.AssetTypes[x]
		pairs, err := o.FetchTradablePairs(a)
		if err != nil {
			return err
		}

		err = o.UpdatePairs(currency.NewPairsFromStrings(pairs), a, false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}
