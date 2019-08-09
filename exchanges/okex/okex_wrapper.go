package okex

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
			asset.PerpetualSwap,
			asset.Index,
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
	o.Websocket = wshandler.New()
	o.APIVersion = okExAPIVersion
	o.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketMessageCorrelationSupported
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
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
		log.Debugf(log.ExchangeSys, "%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.API.Endpoints.WebsocketURL)
	}

	if !o.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := o.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", o.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKEX) FetchTradablePairs(i asset.Item) ([]string, error) {
	var pairs []string
	switch i {
	case asset.Spot:
		prods, err := o.GetSpotTokenPairDetails()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
		}
		return pairs, nil
	case asset.Futures:
		prods, err := o.GetFuturesContractInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, prods[x].UnderlyingIndex+prods[x].QuoteCurrency+"_"+prods[x].Delivery)
		}
		return pairs, nil

	case asset.PerpetualSwap:
		prods, err := o.GetSwapContractInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, prods[x].UnderlyingIndex+"_"+prods[x].QuoteCurrency+"_SWAP")
		}
		return pairs, nil
	case asset.Index:
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
