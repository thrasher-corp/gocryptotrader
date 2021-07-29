package okex

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
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

	// Same format used for perpetual swap and futures
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}

	swap := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
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
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
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
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
				},
				ResultLimit: 1440,
			},
		},
	}

	o.Requester = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// TODO: Specify each individual endpoint rate limits as per docs
		request.WithLimiter(request.NewBasicRateLimit(okExRateInterval, okExRequestRate)),
	)
	o.API.Endpoints = o.NewEndpoints()
	err = o.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      okExAPIURL,
		exchange.WebsocketSpot: OkExWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
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
		wsEndpoint, err := o.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			log.Error(log.ExchangeSys, err)
		}
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()),
			wsEndpoint)
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
			p := strings.Split(prods[x].InstrumentID, currency.DashDelimiter)
			pairs = append(pairs, p[0]+currency.DashDelimiter+p[1]+format.Delimiter+p[2])
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
					currency.DashDelimiter+
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
	assets := o.CurrencyPairs.GetAssetTypes(false)
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
				item := strings.Split(pairs[i], currency.UnderscoreDelimiter)[0]
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
				var c currency.Pair
				c, err = currency.NewPairDelimiter(futuresContracts[i], currency.UnderscoreDelimiter)
				if err != nil {
					return err
				}
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
			p := strings.Split(resp[j].InstrumentID, currency.DashDelimiter)
			nC := currency.NewPairWithDelimiter(p[0]+currency.DashDelimiter+p[1],
				p[2],
				currency.UnderscoreDelimiter)
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
			p := strings.Split(resp[j].InstrumentID, currency.DashDelimiter)
			nC := currency.NewPairWithDelimiter(p[0]+currency.DashDelimiter+p[1],
				p[2],
				currency.UnderscoreDelimiter)
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
	fPair, err := o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerData, err = ticker.GetTicker(o.Name, fPair, assetType)
	if err != nil {
		return o.UpdateTicker(fPair, assetType)
	}
	return
}

// GetRecentTrades returns recent trade data
func (o *OKEX) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	var side order.Side
	switch assetType {
	case asset.Spot:
		var tradeData []okgroup.GetSpotFilledOrdersInformationResponse
		tradeData, err = o.GetSpotFilledOrdersInformation(okgroup.GetSpotFilledOrdersInformationRequest{
			InstrumentID: p.String(),
		})
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     o.Name,
				TID:          tradeData[i].TradeID,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Timestamp,
			})
		}
	case asset.Futures:
		var tradeData []okgroup.GetFuturesFilledOrdersResponse
		tradeData, err = o.GetFuturesFilledOrder(okgroup.GetFuturesFilledOrderRequest{
			InstrumentID: p.String(),
		})
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     o.Name,
				TID:          tradeData[i].TradeID,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Timestamp,
			})
		}
	case asset.PerpetualSwap:
		var tradeData []okgroup.GetSwapFilledOrdersDataResponse
		tradeData, err = o.GetSwapFilledOrdersData(&okgroup.GetSwapFilledOrdersDataRequest{
			InstrumentID: p.String(),
		})
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     o.Name,
				TID:          tradeData[i].TradeID,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Timestamp,
			})
		}
	default:
		return nil, fmt.Errorf("%s asset type %v unsupported", o.Name, assetType)
	}

	err = o.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (o *OKEX) CancelBatchOrders(_ []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}
