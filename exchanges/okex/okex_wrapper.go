package okex

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// GetDefaultConfig returns a default exchange config
func (o *OKEX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	o.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	o.SetupDefaults(exchCfg)

	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := o.UpdateTradablePairs(true)
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

	// Same format used for perpetual swap and futures
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Dash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Underscore,
		},
	}

	swap := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Dash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Underscore,
		},
	}

	err := o.StoreAssetPairFormat(asset.PerpetualSwap, swap)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = o.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	index := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Dash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Dash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.Dash,
		},
	}

	err = o.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = o.StoreAssetPairFormat(asset.Index, index)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
				AccountBalance:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	o.Requester = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// TODO: Specify each individual endpoint rate limits as per docs
		request.WithLimiter(request.NewBasicRateLimit(okExRateInterval, okExRequestRate)),
	)

	o.API.Endpoints.URLDefault = okExAPIURL
	o.API.Endpoints.URL = okExAPIURL
	o.API.Endpoints.WebsocketURL = OkExWebsocketURL
	o.Websocket = stream.New()
	o.APIVersion = okExAPIVersion
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
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()),
			o.API.Endpoints.WebsocketURL)
	}

	format, err := o.GetPairFormat(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
		return
	}

	forceUpdate := false
	enabled, err := o.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
		return
	}

	avail, err := o.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
		return
	}

	if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
		!common.StringDataContains(avail.Strings(), format.Delimiter) {
		forceUpdate = true
		var p currency.Pairs
		p, err = currency.NewPairsFromStrings([]string{currency.BTC.String() +
			format.Delimiter +
			currency.USDT.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies.\n",
				o.Name)
		} else {
			log.Warnf(log.ExchangeSys,
				"Enabled pairs for %v reset due to config upgrade, please enable the ones you would like again.",
				o.Name)

			err = o.UpdatePairs(p, asset.Spot, true, forceUpdate)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies.\n",
					o.Name)
				return
			}
		}
	}

	if !o.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = o.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKEX) FetchTradablePairs(i asset.Item) ([]string, error) {
	var pairs []string

	format, err := o.GetPairFormat(i, false)
	if err != nil {
		return nil, err
	}

	switch i {
	case asset.Spot:
		prods, err := o.GetSpotTokenPairDetails()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs,
				currency.NewPairWithDelimiter(prods[x].BaseCurrency,
					prods[x].QuoteCurrency,
					format.Delimiter).String())
		}
		return pairs, nil
	case asset.Futures:
		prods, err := o.GetFuturesContractInformation()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			p := strings.Split(prods[x].InstrumentID, currency.Dash)
			pairs = append(pairs, p[0]+currency.Dash+p[1]+format.Delimiter+p[2])
		}
		return pairs, nil

	case asset.PerpetualSwap:
		prods, err := o.GetSwapContractInformation()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs,
				prods[x].UnderlyingIndex+
					currency.Dash+
					prods[x].QuoteCurrency+
					format.Delimiter+
					"SWAP")
		}
		return pairs, nil
	case asset.Index:
		// This is updated in futures index
		return nil, errors.New("index updated in futures")
	}

	return nil, fmt.Errorf("%s invalid asset type", o.Name)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKEX) UpdateTradablePairs(forceUpdate bool) error {
	assets := o.CurrencyPairs.GetAssetTypes()
	for x := range assets {
		if assets[x] == asset.Index {
			// Update from futures
			continue
		}

		pairs, err := o.FetchTradablePairs(assets[x])
		if err != nil {
			return err
		}

		if assets[x] == asset.Futures {
			var indexPairs []string
			var futuresContracts []string
			for i := range pairs {
				item := strings.Split(pairs[i], currency.Underscore)[0]
				futuresContracts = append(futuresContracts, pairs[i])
				if common.StringDataContains(indexPairs, item) {
					continue
				}
				indexPairs = append(indexPairs, item)
			}
			var indexPair currency.Pairs
			indexPair, err = currency.NewPairsFromStrings(indexPairs)
			if err != nil {
				return err
			}

			err = o.UpdatePairs(indexPair, asset.Index, false, forceUpdate)
			if err != nil {
				return err
			}

			var futurePairs currency.Pairs
			for i := range futuresContracts {
				c := currency.NewPairDelimiter(futuresContracts[i], currency.Underscore)
				futurePairs = append(futurePairs, c)
			}

			err = o.UpdatePairs(futurePairs, asset.Futures, false, forceUpdate)
			if err != nil {
				return err
			}
			continue
		}
		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}

		err = o.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKEX) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	switch assetType {
	case asset.Spot:
		resp, err := o.GetSpotAllTokenPairsInformation()
		if err != nil {
			return tickerPrice, err
		}

		enabled, err := o.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}

		for j := range resp {
			if !enabled.Contains(resp[j].InstrumentID, true) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         resp[j].Last,
				High:         resp[j].High24h,
				Low:          resp[j].Low24h,
				Bid:          resp[j].BestBid,
				Ask:          resp[j].BestAsk,
				Volume:       resp[j].BaseVolume24h,
				QuoteVolume:  resp[j].QuoteVolume24h,
				Open:         resp[j].Open24h,
				Pair:         resp[j].InstrumentID,
				LastUpdated:  resp[j].Timestamp,
				ExchangeName: o.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}

	case asset.PerpetualSwap:
		resp, err := o.GetAllSwapTokensInformation()
		if err != nil {
			return nil, err
		}

		enabled, err := o.GetEnabledPairs(asset.PerpetualSwap)
		if err != nil {
			return nil, err
		}

		for j := range resp {
			p := strings.Split(resp[j].InstrumentID, currency.Dash)
			nC := currency.NewPairWithDelimiter(p[0]+currency.Dash+p[1],
				p[2],
				currency.Underscore)
			if !enabled.Contains(nC, true) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         resp[j].Last,
				High:         resp[j].High24H,
				Low:          resp[j].Low24H,
				Bid:          resp[j].BestBid,
				Ask:          resp[j].BestAsk,
				Volume:       resp[j].Volume24H,
				Pair:         nC,
				LastUpdated:  resp[j].Timestamp,
				ExchangeName: o.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}

	case asset.Futures:
		resp, err := o.GetAllFuturesTokenInfo()
		if err != nil {
			return nil, err
		}

		enabled, err := o.GetEnabledPairs(asset.Futures)
		if err != nil {
			return nil, err
		}

		for j := range resp {
			p := strings.Split(resp[j].InstrumentID, currency.Dash)
			nC := currency.NewPairWithDelimiter(p[0]+currency.Dash+p[1],
				p[2],
				currency.Underscore)
			if !enabled.Contains(nC, true) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         resp[j].Last,
				High:         resp[j].High24h,
				Low:          resp[j].Low24h,
				Bid:          resp[j].BestBid,
				Ask:          resp[j].BestAsk,
				Volume:       resp[j].Volume24h,
				Pair:         nC,
				LastUpdated:  resp[j].Timestamp,
				ExchangeName: o.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}
	}

	return ticker.GetTicker(o.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKEX) FetchTicker(p currency.Pair, assetType asset.Item) (tickerData *ticker.Price, err error) {
	if assetType == asset.Index {
		return tickerData, errors.New("ticker fetching not supported for index")
	}
	tickerData, err = ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (o *OKEX) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
