package mexc

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Mexc
func (me *MEXC) SetDefaults() {
	me.Name = "MEXC"
	me.Enabled = true
	me.Verbose = true
	me.API.CredentialsValidator.RequiresKey = true
	me.API.CredentialsValidator.RequiresSecret = true

	err := me.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: ""},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = me.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	me.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	me.Requester, err = request.New(me.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	me.API.Endpoints = me.NewEndpoints()
	me.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      spotAPIURL,
		exchange.WebsocketSpot: spotWSAPIURL,
		exchange.RestFutures:   contractAPIURL,
	})
	me.Websocket = stream.NewWebsocket()
	me.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	me.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	me.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (me *MEXC) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		me.SetEnabled(false)
		return nil
	}
	err = me.SetupDefaults(exch)
	if err != nil {
		return err
	}

	/*
		wsRunningEndpoint, err := me.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = me.Websocket.Setup(
			&stream.WebsocketSetup{
				ExchangeConfig:  exch,
				DefaultURL:      mexcWSAPIURL,
				RunningURL:      wsRunningEndpoint,
				Connector:       me.WsConnect,
				Subscriber:      me.Subscribe,
				UnSubscriber:    me.Unsubscribe,
				Features:        &me.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		me.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         me.Name,
			URL:                  me.Websocket.GetWebsocketURL(),
			ProxyURL:             me.Websocket.GetProxyAddress(),
			Verbose:              me.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
	*/
	return nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (me *MEXC) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		result, err := me.GetSymbols(ctx, nil)
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, len(result.Symbols))
		for i := range result.Symbols {
			currencyPairs[i], err = currency.NewPairFromString(result.Symbols[i].Symbol)
			if err != nil {
				return nil, err
			}
		}
		return currencyPairs, nil
	case asset.Futures:
		result, err := me.GetContractsDetail(ctx, "")
		if err != nil {
			return nil, err
		}
		currencyPairs := make(currency.Pairs, len(result.Data))
		for i := range result.Data {
			currencyPairs[i], err = currency.NewPairFromString(result.Data[i].Symbol)
			if err != nil {
				return nil, err
			}
		}
		return currencyPairs, nil
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (me *MEXC) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := me.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := me.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		err = me.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (me *MEXC) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pFormat, err := me.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		pairString := pFormat.Format(p)
		tickers, err := me.Get24HourTickerPriceChangeStatistics(ctx, []string{pairString})
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers {
			if tickers[t].Symbol != pairString {
				continue
			}
			found = true
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         p,
				ExchangeName: me.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				Volume:       tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			})
			if err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	case asset.Futures:
		pairString := pFormat.Format(p)
		tickers, err := me.GetContractTickers(ctx, pairString)
		if err != nil {
			return nil, err
		}
		var found bool
		for t := range tickers.Data {
			if tickers.Data[t].Symbol != pairString {
				continue
			}
			found = true
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         p,
				ExchangeName: me.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			})
			if err != nil {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("%w for currency pair: %s", ticker.ErrTickerNotFound, p)
		}
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return ticker.GetTicker(me.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (me *MEXC) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		tickers, err := me.Get24HourTickerPriceChangeStatistics(ctx, []string{})
		if err != nil {
			return err
		}
		for t := range tickers {
			pair, err := currency.NewPairFromString(tickers[t].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				ExchangeName: me.Name,
				AssetType:    assetType,
				Last:         tickers[t].LastPrice.Float64(),
				High:         tickers[t].HighPrice.Float64(),
				Low:          tickers[t].LowPrice.Float64(),
				Bid:          tickers[t].BidPrice.Float64(),
				BidSize:      tickers[t].BidQty.Float64(),
				Ask:          tickers[t].AskPrice.Float64(),
				AskSize:      tickers[t].AskQty.Float64(),
				Volume:       tickers[t].Volume.Float64(),
				QuoteVolume:  tickers[t].QuoteVolume.Float64(),
				Open:         tickers[t].OpenPrice.Float64(),
				LastUpdated:  tickers[t].CloseTime.Time(),
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		tickers, err := me.GetContractTickers(ctx, "")
		if err != nil {
			return err
		}
		for t := range tickers.Data {
			pair, err := currency.NewPairFromString(tickers.Data[t].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers.Data[t].LastPrice,
				High:         tickers.Data[t].High24Price,
				Low:          tickers.Data[t].Lower24Price,
				Bid:          tickers.Data[t].MaxBidPrice,
				AskSize:      tickers.Data[t].MinAskPrice,
				Volume:       tickers.Data[t].Volume24,
				MarkPrice:    tickers.Data[t].FairPrice,
				IndexPrice:   tickers.Data[t].IndexPrice,
				Pair:         pair,
				ExchangeName: me.Name,
				AssetType:    asset.Futures,
				LastUpdated:  tickers.Data[t].Timestamp.Time(),
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (me *MEXC) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(me.Name, p, assetType)
	if err != nil {
		return me.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (me *MEXC) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(me.Name, pair, assetType)
	if err != nil {
		return me.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (me *MEXC) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	book := &orderbook.Base{
		Exchange:        me.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: me.CanVerifyOrderbook,
	}
	pFormat, err := me.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		result, err := me.GetOrderbook(ctx, pFormat.Format(pair), 1000)
		if err != nil {
			return book, err
		}

		book.Bids = make([]orderbook.Tranche, len(result.Bids))
		for x := range result.Bids {
			book.Bids[x] = orderbook.Tranche{
				Price:  result.Bids[x][0].Float64(),
				Amount: result.Bids[x][1].Float64(),
			}
		}
		book.Asks = make([]orderbook.Tranche, len(result.Asks))
		for x := range result.Asks {
			book.Asks[x] = orderbook.Tranche{
				Price:  result.Asks[x][0].Float64(),
				Amount: result.Asks[x][1].Float64(),
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
		return orderbook.Get(me.Name, pair, assetType)
	case asset.Futures:
		result, err := me.GetContractDepthInformation(ctx, pFormat.Format(pair), 1000)
		if err != nil {
			return nil, err
		}
		book.Bids = make([]orderbook.Tranche, len(result.Bids))
		for x := range result.Bids {
			book.Bids[x] = orderbook.Tranche{
				Price:  result.Bids[x].Price,
				Amount: result.Bids[x].Amount,
			}
		}
		book.Asks = make([]orderbook.Tranche, len(result.Asks))
		for x := range result.Asks {
			book.Asks[x] = orderbook.Tranche{
				Price:  result.Asks[x].Price,
				Amount: result.Asks[x].Amount,
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
		return orderbook.Get(me.Name, pair, assetType)
	default:
		return nil, fmt.Errorf("%w: asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (me *MEXC) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// If fetching requires more than one asset type please set
	// HasAssetTypeAccountSegregation to true in RESTCapabilities above.
	// var info account.Holdings
	// accAssets,err := me.GetSubAccountAsset(ctx, )
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (me *MEXC) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// Example implementation below:
	// 	creds, err := me.GetCredentials(ctx)
	// 	if err != nil {
	// 		return account.Holdings{}, err
	// 	}
	// 	acc, err := account.GetHoldings(me.Name, creds, assetType)
	// 	if err != nil {
	// 		return me.UpdateAccountInfo(ctx, assetType)
	// 	}
	// 	return acc, nil
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (me *MEXC) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (me *MEXC) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (me *MEXC) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (me *MEXC) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetServerTime returns the current exchange server time.
func (me *MEXC) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (me *MEXC) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(me.GetTradingRequirements()); err != nil {
		return nil, err
	}
	// When an order has been submitted you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.SubmitResponse if you think they are applicable.
	// resp, err := s.DeriveSubmitResponse( /*newOrderID*/)
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.Date = exampleTime // e.g. If this is supplied by the exchanges API.
	// return resp, nil
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (me *MEXC) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// When an order has been modified you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.ModifyResponse if you think they are applicable.
	// resp, err := action.DeriveModifyResponse()
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.OrderID = maybeANewOrderID // e.g. If this is supplied by the exchanges API.
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (me *MEXC) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (me *MEXC) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (me *MEXC) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (me *MEXC) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (me *MEXC) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (me *MEXC) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (me *MEXC) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (me *MEXC) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (me *MEXC) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateAPICredentials validates current credentials used for wrapper
func (me *MEXC) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := me.UpdateAccountInfo(ctx, assetType)
	return me.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (me *MEXC) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	intervalString, err := intervalToString(kline.FiveMin)
	if err != nil {
		return nil, err
	}
	pFormat, err := me.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	req, err := me.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		result, err := me.GetCandlestick(ctx, pFormat.Format(pair), intervalString, start, end, 0)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(result))
		for c := range result {
			timeSeries[c] = kline.Candle{
				Time:   result[c].CloseTime.Time(),
				Open:   result[c].OpenPrice.Float64(),
				High:   result[c].HighPrice.Float64(),
				Low:    result[c].LowPrice.Float64(),
				Close:  result[c].ClosePrice.Float64(),
				Volume: result[c].Volume.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		result, err := me.GetContractsCandlestickData(ctx, pFormat.Format(pair), req.ExchangeInterval, start, end)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(result.Data.ClosePrice))
		for i := range result.Data.ClosePrice {
			timeSeries[i] = kline.Candle{
				Open:   result.Data.ClosePrice[i],
				Time:   result.Data.Time[i].Time(),
				High:   result.Data.HighPrice[i],
				Low:    result.Data.LowPrice[i],
				Close:  result.Data.ClosePrice[i],
				Volume: result.Data.Volume[i],
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (me *MEXC) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (me *MEXC) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrNotYetImplemented
}

// GetLatestFundingRates returns the latest funding rates data
func (me *MEXC) GetLatestFundingRates(_ context.Context, _ *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// UpdateOrderExecutionLimits updates order execution limits
func (me *MEXC) UpdateOrderExecutionLimits(ctx context.Context, assetType asset.Item) error {
	return common.ErrNotYetImplemented
}
