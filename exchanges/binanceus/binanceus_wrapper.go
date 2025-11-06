package binanceus

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Binanceus
func (e *Exchange) SetDefaults() {
	e.Name = "Binanceus"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	fmt1 := currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	if err := e.SetAssetPairStore(asset.Spot, fmt1); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing `spot` default asset formats: %s", e.Name, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				SubmitOrders:          true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				TradeFee:              true,
				CryptoDepositFee:      true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				TradeFetching:          true,
				KlineFetching:          true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	if err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   binanceusAPIURL,
		exchange.RestSpotSupplementary:      binanceusAPIURL,
		exchange.WebsocketSpot:              binanceusDefaultWebsocketURL,
		exchange.WebsocketSpotSupplementary: binanceusDefaultWebsocketURL,
	}); err != nil {
		log.Errorf(log.ExchangeSys,
			"%s setting default endpoints error %v",
			e.Name, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}

	ePoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            binanceusDefaultWebsocketURL,
		RunningURL:            ePoint,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.GenerateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed: e.Features.Enabled.TradeFeed,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(300 * time.Millisecond),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	info, err := e.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(info.Symbols))
	for x := range info.Symbols {
		if info.Symbols[x].Status != "TRADING" ||
			!info.Symbols[x].IsSpotTradingAllowed {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(info.Symbols[x].BaseAsset,
			info.Symbols[x].QuoteAsset)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	if err := e.UpdatePairs(pairs, asset.Spot, false); err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if a != asset.Spot {
		return nil, fmt.Errorf("%w '%v'", asset.ErrNotSupported, a)
	}
	tick, err := e.GetPriceChangeStats(ctx, p)
	if err != nil {
		return nil, err
	}
	err = ticker.ProcessTicker(&ticker.Price{
		Last:         tick.LastPrice,
		High:         tick.HighPrice,
		Low:          tick.LowPrice,
		Bid:          tick.BidPrice,
		Ask:          tick.AskPrice,
		Volume:       tick.Volume,
		QuoteVolume:  tick.QuoteVolume,
		Open:         tick.OpenPrice,
		Close:        tick.PrevClosePrice,
		Pair:         p,
		ExchangeName: e.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	tick, err := e.GetTickers(ctx)
	if err != nil {
		return err
	}

	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range pairs {
		for y := range tick {
			pairFmt, err := e.FormatExchangeCurrency(pairs[i], a)
			if err != nil {
				return err
			}
			if tick[y].Symbol != pairFmt.String() {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice,
				High:         tick[y].HighPrice,
				Low:          tick[y].LowPrice,
				Bid:          tick[y].BidPrice,
				Ask:          tick[y].AskPrice,
				Volume:       tick[y].Volume,
				QuoteVolume:  tick[y].QuoteVolume,
				Open:         tick[y].OpenPrice,
				Close:        tick[y].PrevClosePrice,
				Pair:         pairFmt,
				ExchangeName: e.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	orderbookNew, err := e.GetOrderBookDepth(ctx, pair, 1000)
	if err != nil {
		return nil, err
	}
	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbookNew.Bids,
		Asks:              orderbookNew.Asks,
	}
	if err := ob.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	resp, err := e.GetAccount(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp.Balances {
		freeBalance := resp.Balances[i].Free.InexactFloat64()
		locked := resp.Balances[i].Locked.InexactFloat64()
		subAccts[0].Balances.Set(resp.Balances[i].Asset, accounts.Balance{
			Total: freeBalance + locked,
			Hold:  locked,
			Free:  freeBalance,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	withdrawals, err := e.WithdrawalHistory(ctx, c, "", time.Time{}, time.Time{}, 0, 10000)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		resp[i] = exchange.WithdrawalHistory{
			Status:          strconv.FormatInt(withdrawals[i].Status, 10),
			TransferID:      withdrawals[i].ID,
			Currency:        withdrawals[i].Coin,
			Amount:          withdrawals[i].Amount,
			Fee:             withdrawals[i].TransactionFee,
			CryptoToAddress: withdrawals[i].Address,
			CryptoTxID:      withdrawals[i].ID,
			CryptoChain:     withdrawals[i].Network,
			Timestamp:       withdrawals[i].ApplyTime.Time(),
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	const limit = 1000
	tradeData, err := e.GetMostRecentTrades(ctx, RecentTradeRequestParams{p, limit})
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		resp[i] = trade.Data{
			TID:          strconv.FormatInt(tradeData[i].ID, 10),
			Exchange:     e.Name,
			AssetType:    assetType,
			CurrencyPair: p,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Quantity,
			Timestamp:    tradeData[i].Time.Time(),
		}
	}

	if e.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	req := AggregatedTradeRequestParams{
		Symbol:    p,
		StartTime: timestampStart,
		EndTime:   timestampEnd,
	}
	trades, err := e.GetAggregateTrades(ctx, &req)
	if err != nil {
		return nil, err
	}
	result := make([]trade.Data, len(trades))
	exName := e.Name
	for i := range trades {
		t := trades[i].toTradeData(p, exName, assetType)
		result[i] = *t
	}
	return result, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	var timeInForce string
	var sideType string
	err := s.Validate(e.GetTradingRequirements())
	if err != nil {
		return nil, err
	}
	if s.AssetType != asset.Spot {
		return nil, fmt.Errorf("%s %w", s.AssetType, asset.ErrNotSupported)
	}
	if s.Side.IsLong() {
		sideType = order.Buy.String()
	} else {
		sideType = order.Sell.String()
	}
	var requestParamOrderType RequestParamsOrderType
	switch s.Type {
	case order.Market:
		requestParamOrderType = BinanceRequestParamsOrderMarket
	case order.Limit:
		timeInForce = order.GoodTillCancel.String()
		requestParamOrderType = BinanceRequestParamsOrderLimit
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
	}
	var response NewOrderResponse
	response, err = e.NewOrder(ctx, &NewOrderRequest{
		Symbol:           s.Pair,
		Side:             sideType,
		Price:            s.Price,
		Quantity:         s.Amount,
		TradeType:        requestParamOrderType,
		TimeInForce:      timeInForce,
		NewClientOrderID: s.ClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	if response.OrderID > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
	}
	if response.ExecutedQty == response.OrigQty {
		submitOrderResponse.Status = order.Filled
	}
	for i := range response.Fills {
		submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
			Price:    response.Fills[i].Price,
			Amount:   response.Fills[i].Qty,
			Fee:      response.Fills[i].Commission,
			FeeAsset: response.Fills[i].CommissionAsset,
			Exchange: e.Name,
		})
	}

	return &submitOrderResponse, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	if o.AssetType != asset.Spot {
		return fmt.Errorf("%w '%v'", asset.ErrNotSupported, o.AssetType)
	}
	_, err := e.CancelExistingOrder(ctx,
		&CancelOrderRequestParams{
			Symbol:                o.Pair,
			OrderID:               o.OrderID,
			ClientSuppliedOrderID: o.ClientOrderID,
		})
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	if orderCancellation.AssetType == asset.Spot {
		symbolValue, err := e.FormatSymbol(orderCancellation.Pair, asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		openOrders, err := e.GetAllOpenOrders(ctx, symbolValue)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for ind := range openOrders {
			pair, err := currency.NewPairFromString(openOrders[ind].Symbol)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			_, err = e.CancelExistingOrder(ctx, &CancelOrderRequestParams{
				Symbol:                pair,
				OrderID:               strconv.FormatUint(openOrders[ind].OrderID, 10),
				ClientSuppliedOrderID: openOrders[ind].ClientOrderID,
			})
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	} else {
		return cancelAllOrdersResponse, fmt.Errorf("%w '%v'", asset.ErrNotSupported, orderCancellation.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	orderIDInt, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid orderID %w", err)
	}
	symbolValue, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	var orderType order.Type
	resp, err := e.GetOrder(ctx, &OrderRequestParams{
		Symbol:  symbolValue,
		OrderID: orderIDInt,
	})
	if err != nil {
		return nil, err
	}
	orderSide, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
	}
	status, err := order.StringToOrderStatus(resp.Status)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
	}
	orderType, err = order.StringToOrderType(resp.Type)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
	}

	return &order.Detail{
		Amount:         resp.OrigQty,
		Exchange:       e.Name,
		OrderID:        strconv.FormatUint(resp.OrderID, 10),
		ClientOrderID:  resp.ClientOrderID,
		Side:           orderSide,
		Type:           orderType,
		Pair:           pair,
		Cost:           resp.CummulativeQuoteQty,
		AssetType:      assetType,
		Status:         status,
		Price:          resp.Price,
		ExecutedAmount: resp.ExecutedQty,
		Date:           resp.Time.Time(),
		LastUpdated:    resp.UpdateTime.Time(),
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _ /*accountID*/, chain string) (*deposit.Address, error) {
	address, err := e.GetDepositAddressForCurrency(ctx, c.String(), chain)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: address.Address,
		Tag:     address.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawID, err := e.WithdrawCrypto(ctx, withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted. But, GCT has no concept of withdrawal via SEN
// the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
// So, this method is not implemented.
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// But, GCT has no concept of withdrawal via SEN the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	var symbol string
	var pair currency.Pair
	var selectedOrders []Order
	if getOrdersRequest.AssetType != asset.Spot {
		return nil, fmt.Errorf("%s %w", getOrdersRequest.AssetType, asset.ErrNotSupported)
	}
	if len(getOrdersRequest.Pairs) != 1 {
		symbol = ""
	} else {
		symbol, err = e.FormatSymbol(getOrdersRequest.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
	}
	resp, err := e.GetAllOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}
	for s := range resp {
		ord := resp[s]
		pair, err = currency.NewPairFromString(ord.Symbol)
		if err != nil {
			continue
		}
		for p := range getOrdersRequest.Pairs {
			if getOrdersRequest.Pairs[p].Equal(pair) {
				selectedOrders = append(selectedOrders, ord)
			}
		}
	}
	orders := make([]order.Detail, len(selectedOrders))
	for x := range selectedOrders {
		var orderSide order.Side
		var orderType order.Type
		var orderStatus order.Status
		orderSide, err = order.StringToOrderSide(strings.ToUpper(resp[x].Side))
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		orderType, err = order.StringToOrderType(strings.ToUpper(resp[x].Type))
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		orderStatus, err = order.StringToOrderStatus(resp[x].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		orders[x] = order.Detail{
			Amount:        resp[x].OrigQty,
			Date:          resp[x].Time.Time(),
			Exchange:      e.Name,
			OrderID:       strconv.FormatUint(resp[x].OrderID, 10),
			ClientOrderID: resp[x].ClientOrderID,
			Side:          orderSide,
			Type:          orderType,
			Price:         resp[x].Price,
			Status:        orderStatus,
			Pair:          getOrdersRequest.Pairs[0],
			AssetType:     getOrdersRequest.AssetType,
			LastUpdated:   resp[x].UpdateTime.Time(),
		}
	}
	return getOrdersRequest.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// An endpoint like /api/v3/allOrders does not exist in the binance us
	// so This end point is left unimplemented
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	candles, err := e.GetSpotKline(ctx, &KlinesRequestParams{
		Interval:  e.GetIntervalEnum(req.ExchangeInterval),
		Symbol:    req.Pair,
		StartTime: req.Start,
		EndTime:   req.End,
		Limit:     req.RequestLimit,
	})
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].OpenTime.Time(),
			Open:   candles[x].Open.Float64(),
			High:   candles[x].High.Float64(),
			Low:    candles[x].Low.Float64(),
			Close:  candles[x].Close.Float64(),
			Volume: candles[x].Volume.Float64(),
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []CandleStick
		candles, err = e.GetSpotKline(ctx, &KlinesRequestParams{
			Interval:  e.GetIntervalEnum(req.ExchangeInterval),
			Symbol:    req.Pair,
			StartTime: req.RangeHolder.Ranges[x].Start.Time,
			EndTime:   req.RangeHolder.Ranges[x].End.Time,
			Limit:     req.RequestLimit,
		})
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coinInfo, err := e.GetAssetFeesAndWalletStatus(ctx)
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range coinInfo {
		if strings.EqualFold(coinInfo[x].Coin, cryptocurrency.String()) {
			for y := range coinInfo[x].NetworkList {
				if coinInfo[x].NetworkList[y].DepositEnable {
					availableChains = append(availableChains, coinInfo[x].NetworkList[y].Network)
				}
			}
		}
	}
	return availableChains, nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	symbol, err := e.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	return tradeBaseURL + symbol, nil
}
