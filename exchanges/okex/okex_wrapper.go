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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
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
		UseGlobalFormat: false,
	}
	// Same format used for perpetual swap and futures
	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
		},
	}
	o.CurrencyPairs.Store(asset.PerpetualSwap, fmt1)
	o.CurrencyPairs.Store(asset.Futures, fmt1)

	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}
	o.CurrencyPairs.Store(asset.Spot, fmt2)
	o.CurrencyPairs.Store(asset.Index, fmt2)

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  true,
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
	o.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
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

	if o.Config.CurrencyPairs.Pairs[asset.Spot].ConfigFormat == nil || o.Config.CurrencyPairs.Pairs[asset.Spot].RequestFormat == nil ||
		o.Config.CurrencyPairs.Pairs[asset.Index].ConfigFormat == nil || o.Config.CurrencyPairs.Pairs[asset.Index].RequestFormat == nil {
		fmt := currency.PairStore{
			RequestFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
		}
		o.CurrencyPairs.Store(asset.Spot, fmt)
		o.Config.CurrencyPairs.Store(asset.Spot, fmt)
		o.CurrencyPairs.Store(asset.Index, fmt)
		o.Config.CurrencyPairs.Store(asset.Index, fmt)
	}

	if o.Config.CurrencyPairs.Pairs[asset.Futures].ConfigFormat == nil || o.Config.CurrencyPairs.Pairs[asset.Futures].RequestFormat == nil ||
		o.Config.CurrencyPairs.Pairs[asset.PerpetualSwap].ConfigFormat == nil || o.Config.CurrencyPairs.Pairs[asset.PerpetualSwap].RequestFormat == nil {
		fmt := currency.PairStore{
			RequestFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "_",
			},
		}
		o.CurrencyPairs.Store(asset.Futures, fmt)
		o.Config.CurrencyPairs.Store(asset.Futures, fmt)
		o.CurrencyPairs.Store(asset.PerpetualSwap, fmt)
		o.Config.CurrencyPairs.Store(asset.PerpetualSwap, fmt)
	}

	if !common.StringDataContains(o.Config.CurrencyPairs.Pairs[asset.Spot].Enabled.Strings(), o.CurrencyPairs.Pairs[asset.Spot].RequestFormat.Delimiter) {
		enabledPairs := currency.NewPairsFromStrings([]string{"EOS-USDT"})
		log.Warnf(log.ExchangeSys,
			"Enabled pairs for %v reset due to config upgrade, please enable the ones you would like again.", o.Name)

		err := o.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies.\n", o.GetName())
			return
		}
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
			pairs = append(pairs, fmt.Sprintf("%v%v%v", prods[x].BaseCurrency, o.GetPairFormat(i, false).Delimiter, prods[x].QuoteCurrency))
		}
		return pairs, nil
	case asset.Futures:
		prods, err := o.GetFuturesContractInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, fmt.Sprintf("%v%v%v", prods[x].UnderlyingIndex+prods[x].QuoteCurrency, o.GetPairFormat(i, false).Delimiter, prods[x].Delivery))
		}
		return pairs, nil

	case asset.PerpetualSwap:
		prods, err := o.GetSwapContractInformation()
		if err != nil {
			return nil, err
		}

		var pairs []string
		for x := range prods {
			pairs = append(pairs, fmt.Sprintf("%v%v%v%vSWAP", prods[x].UnderlyingIndex, o.GetPairFormat(i, false).Delimiter, prods[x].QuoteCurrency, o.GetPairFormat(i, false).Delimiter))
		}
		return pairs, nil
	case asset.Index:
		return []string{fmt.Sprintf("BTC%vUSD", o.GetPairFormat(i, false).Delimiter)}, nil
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

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKEX) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerData ticker.Price
	switch assetType {
	case asset.Spot:
		resp, err := o.GetSpotAllTokenPairsInformation()
		if err != nil {
			return tickerData, err
		}
		pairs := o.GetEnabledPairs(assetType)
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}
				tickerData = ticker.Price{
					Last:        resp[j].Last,
					High:        resp[j].High24h,
					Low:         resp[j].Low24h,
					Bid:         resp[j].BestBid,
					Ask:         resp[j].BestAsk,
					Volume:      resp[j].BaseVolume24h,
					QuoteVolume: resp[j].QuoteVolume24h,
					Open:        resp[j].Open24h,
					Pair:        pairs[i],
					LastUpdated: resp[j].Timestamp,
				}
				err = ticker.ProcessTicker(o.Name, &tickerData, assetType)
				if err != nil {
					log.Error(log.Ticker, err)
				}
			}
		}
	case asset.PerpetualSwap:
		resp, err := o.GetAllSwapTokensInformation()
		if err != nil {
			return tickerData, err
		}
		pairs := o.GetEnabledPairs(assetType)
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}
				tickerData = ticker.Price{
					Last:        resp[j].Last,
					High:        resp[j].High24H,
					Low:         resp[j].Low24H,
					Bid:         resp[j].BestBid,
					Ask:         resp[j].BestAsk,
					Volume:      resp[j].Volume24H,
					Pair:        resp[j].InstrumentID,
					LastUpdated: resp[j].Timestamp,
				}
				err = ticker.ProcessTicker(o.Name, &tickerData, assetType)
				if err != nil {
					log.Error(log.Ticker, err)
				}
			}
		}
	case asset.Futures:
		resp, err := o.GetAllFuturesTokenInfo()
		if err != nil {
			return tickerData, err
		}
		pairs := o.GetEnabledPairs(assetType)
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}
				tickerData = ticker.Price{
					Last:        resp[j].Last,
					High:        resp[j].High24h,
					Low:         resp[j].Low24h,
					Bid:         resp[j].BestBid,
					Ask:         resp[j].BestAsk,
					Volume:      resp[j].Volume24h,
					Pair:        resp[j].InstrumentID,
					LastUpdated: resp[j].Timestamp,
				}
				err = ticker.ProcessTicker(o.Name, &tickerData, assetType)
				if err != nil {
					log.Error(log.Ticker, err)
				}
			}
		}
	}

	return ticker.GetTicker(o.GetName(), p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKEX) FetchTicker(p currency.Pair, assetType asset.Item) (tickerData ticker.Price, err error) {
	tickerData, err = ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}
