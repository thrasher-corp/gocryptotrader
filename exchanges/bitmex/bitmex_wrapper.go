package bitmex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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

// SetDefaults sets the basic defaults for Bitmex
func (e *Exchange) SetDefaults() {
	e.Name = "Bitmex"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract, asset.Futures, asset.Index} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Spot {
			ps.RequestFormat.Delimiter = currency.UnderscoreDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	if err := e.DisableAssetWebsocketSupport(asset.Index); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, asset.Index, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				ModifyOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
				FundingRateFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				DeadMansSwitch:         true,
				GetOrders:              true,
				GetOrder:               true,
				FundingRateFetching:    false, // supported but not implemented // TODO when multi-websocket support added
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.PerpetualContract: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
				},
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawCryptoWithEmail |
				exchange.WithdrawCryptoWith2FA |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitmexAPIURL,
		exchange.WebsocketSpot: bitmexWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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

	wsEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            bitmexWSURL,
		RunningURL:            wsEndpoint,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  wsEndpoint,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	marketInfo, err := e.GetActiveAndIndexInstruments(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(marketInfo))
	for x := range marketInfo {
		if marketInfo[x].State != "Open" && a != asset.Index {
			continue
		}

		var pair currency.Pair
		switch a {
		case asset.Spot:
			if marketInfo[x].Typ == spotID {
				pair, err = currency.NewPairFromString(marketInfo[x].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.PerpetualContract:
			if marketInfo[x].Typ == perpetualContractID {
				var settleTrail string
				if strings.Contains(marketInfo[x].Symbol, currency.UnderscoreDelimiter) {
					// Example: ETHUSD_ETH quoted in USD, paid out in ETH.
					settlement := strings.Split(marketInfo[x].Symbol, currency.UnderscoreDelimiter)
					if len(settlement) != 2 {
						log.Warnf(log.ExchangeSys, "%s currency %s %s cannot be added to tradable pairs",
							e.Name,
							marketInfo[x].Symbol,
							a)
						break
					}
					settleTrail = currency.UnderscoreDelimiter + settlement[1]
				}
				pair, err = currency.NewPairFromStrings(marketInfo[x].Underlying,
					marketInfo[x].QuoteCurrency+settleTrail)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.Futures:
			if marketInfo[x].Typ == futuresID {
				isolate := strings.Split(marketInfo[x].Symbol, currency.UnderscoreDelimiter)
				if len(isolate[0]) < 3 {
					log.Warnf(log.ExchangeSys, "%s currency %s %s be cannot added to tradable pairs",
						e.Name,
						marketInfo[x].Symbol,
						a)
					break
				}
				var settleTrail string
				if len(isolate) == 2 {
					// Example: ETHUSDU22_ETH quoted in USD, paid out in ETH.
					settleTrail = currency.UnderscoreDelimiter + isolate[1]
				}

				root := isolate[0][:len(isolate[0])-3]
				contract := isolate[0][len(isolate[0])-3:]

				pair, err = currency.NewPairFromStrings(root, contract+settleTrail)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.Index:
			// TODO: This can be expanded into individual assets later.
			if marketInfo[x].Typ == bitMEXBasketIndexID ||
				marketInfo[x].Typ == bitMEXPriceIndexID ||
				marketInfo[x].Typ == bitMEXLendingPremiumIndexID ||
				marketInfo[x].Typ == bitMEXVolatilityIndexID {
				pair, err = currency.NewPairFromString(marketInfo[x].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		default:
			return nil, errors.New("unhandled asset type")
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for x := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[x], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w for [%v]", asset.ErrNotSupported, a)
	}

	tick, err := e.GetActiveAndIndexInstruments(ctx)
	if err != nil {
		return err
	}

	var enabled bool
instruments:
	for j := range tick {
		var pair currency.Pair
		switch a {
		case asset.Futures:
			if tick[j].Typ != futuresID {
				continue instruments
			}
			pair, enabled, err = e.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.Index:
			switch tick[j].Typ {
			case bitMEXBasketIndexID,
				bitMEXPriceIndexID,
				bitMEXLendingPremiumIndexID,
				bitMEXVolatilityIndexID:
			default:
				continue instruments
			}
			// NOTE: Filtering is done below to remove the underscore in a
			// limited amount of index asset strings while the rest do not
			// contain an underscore. Calling DeriveFrom will then error and
			// the instruments will be missed.
			tick[j].Symbol = strings.Replace(tick[j].Symbol, currency.UnderscoreDelimiter, "", 1)
			pair, enabled, err = e.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.PerpetualContract:
			if tick[j].Typ != perpetualContractID {
				continue instruments
			}
			pair, enabled, err = e.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.Spot:
			if tick[j].Typ != spotID {
				continue instruments
			}
			tick[j].Symbol = strings.Replace(tick[j].Symbol, currency.UnderscoreDelimiter, "", 1)
			pair, enabled, err = e.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		}

		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return err
		}
		if !enabled {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick[j].LastPrice,
			High:         tick[j].HighPrice,
			Low:          tick[j].LowPrice,
			Bid:          tick[j].BidPrice,
			Ask:          tick[j].AskPrice,
			Volume:       tick[j].Volume24h,
			Close:        tick[j].PrevClosePrice,
			Pair:         pair,
			LastUpdated:  tick[j].Timestamp,
			ExchangeName: e.Name,
			OpenInterest: tick[j].OpenInterest,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(e.Name, fPair, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}

	if assetType == asset.Index {
		return book, common.ErrFunctionNotSupported
	}

	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := e.GetOrderbook(ctx,
		OrderBookGetL2Params{
			Symbol: fPair.String(),
			Depth:  500,
		})
	if err != nil {
		return book, err
	}

	book.Asks = make(orderbook.Levels, 0, len(orderbookNew))
	book.Bids = make(orderbook.Levels, 0, len(orderbookNew))
	for i := range orderbookNew {
		switch {
		case strings.EqualFold(orderbookNew[i].Side, order.Sell.String()):
			book.Asks = append(book.Asks, orderbook.Level{
				Amount: float64(orderbookNew[i].Size),
				Price:  orderbookNew[i].Price,
			})
		case strings.EqualFold(orderbookNew[i].Side, order.Buy.String()):
			book.Bids = append(book.Bids, orderbook.Level{
				Amount: float64(orderbookNew[i].Size),
				Price:  orderbookNew[i].Price,
			})
		default:
			return book,
				fmt.Errorf("could not process orderbook, order side [%s] could not be matched",
					orderbookNew[i].Side)
		}
	}
	book.Asks.Reverse() // Reverse order of asks to ascending

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	userMargins, err := e.GetAllUserMargin(ctx)
	if err != nil {
		return nil, err
	}
	var subAccts accounts.SubAccounts
	// Need to update to add Margin/Liquidity availability
	for i := range userMargins {
		wallet, err := e.GetWalletInfo(ctx, userMargins[i].Currency)
		if err != nil {
			continue
		}
		a := accounts.NewSubAccount(assetType, strconv.FormatInt(userMargins[i].Account, 10))
		a.Balances.Set(wallet.Currency, accounts.Balance{Total: wallet.Amount})
		subAccts = subAccts.Merge(a)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	history, err := e.GetWalletHistory(ctx, "all")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(history))
	for i := range history {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          history[i].TransactStatus,
			Timestamp:       history[i].Timestamp,
			Currency:        history[i].Currency,
			Amount:          history[i].Amount,
			Fee:             history[i].Fee,
			TransferType:    history[i].TransactType,
			CryptoToAddress: history[i].Address,
			CryptoTxID:      history[i].TransactID,
			CryptoChain:     history[i].Network,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	history, err := e.GetWalletHistory(ctx, c.String())
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(history))
	for i := range history {
		resp[i] = exchange.WithdrawalHistory{
			Status:          history[i].TransactStatus,
			Timestamp:       history[i].Timestamp,
			Currency:        history[i].Currency,
			Amount:          history[i].Amount,
			Fee:             history[i].Fee,
			TransferType:    history[i].TransactType,
			CryptoToAddress: history[i].Address,
			CryptoTxID:      history[i].TransactID,
			CryptoChain:     history[i].Network,
		}
	}
	return resp, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return e.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if assetType == asset.Index {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	req := &GenericRequestParams{
		Symbol:  p.String(),
		Count:   countLimit,
		EndTime: timestampEnd.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	ts := timestampStart
	var resp []trade.Data
allTrades:
	for {
		req.StartTime = ts.UTC().Format("2006-01-02T15:04:05.000Z")
		var tradeData []Trade
		tradeData, err = e.GetTrade(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			if tradeData[i].Timestamp.Before(timestampStart) || tradeData[i].Timestamp.After(timestampEnd) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			if tradeData[i].Price == 0 {
				// Please note that indices (symbols starting with .) post trades at intervals to the trade feed.
				// These have a size of 0 and are used only to indicate a changing price.
				continue
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       float64(tradeData[i].Size),
				Timestamp:    tradeData[i].Timestamp,
				TID:          tradeData[i].TrdMatchID,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeData[i].Timestamp) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeData[i].Timestamp
			}
		}
		if len(tradeData) != int(countLimit) {
			break allTrades
		}
	}
	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if math.Trunc(s.Amount) != s.Amount {
		return nil,
			errors.New("order contract amount can not have decimals")
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	orderNewParams := OrderNewParams{
		OrderType:     s.Type.Title(),
		Symbol:        fPair.String(),
		OrderQuantity: s.Amount,
		Side:          s.Side.Title(),
	}

	if s.Type == order.Limit {
		orderNewParams.Price = s.Price
	}

	response, err := e.CreateOrder(ctx, &orderNewParams)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(response.OrderID)
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}

	if math.Trunc(action.Amount) != action.Amount {
		return nil, errors.New("contract amount can not have decimals")
	}

	o, err := e.AmendOrder(ctx, &OrderAmendParams{
		OrderID:  action.OrderID,
		OrderQty: int32(action.Amount),
		Price:    action.Price,
	})
	if err != nil {
		return nil, err
	}

	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}

	resp.OrderID = o.OrderID
	resp.RemainingAmount = o.OrderQty
	resp.LastUpdated = o.TransactTime
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	_, err := e.CancelOrders(ctx, &OrderCancelParams{
		OrderID: o.OrderID,
	})
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	var orderIDs, clientIDs []string
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			clientIDs = append(clientIDs, o[i].ClientID)
		case o[i].OrderID != "":
			orderIDs = append(orderIDs, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	joinedOrderIDs := strings.Join(orderIDs, ",")
	joinedClientIDs := strings.Join(clientIDs, ",")
	params := &OrderCancelParams{
		OrderID:       joinedOrderIDs,
		ClientOrderID: joinedClientIDs,
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	cancelResponse, err := e.CancelOrders(ctx, params)
	if err != nil {
		return nil, err
	}
	for i := range cancelResponse {
		resp.Status[cancelResponse[i].OrderID] = cancelResponse[i].OrdStatus
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	var emptyParams OrderCancelAllParams
	orders, err := e.CancelAllExistingOrders(ctx, emptyParams)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range orders {
		if orders[i].OrdRejReason != "" {
			cancelAllOrdersResponse.Status[orders[i].OrderID] = orders[i].OrdRejReason
		}
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

	resp, err := e.GetOrders(ctx, &OrdersRequest{
		Filter: `{"orderID":"` + orderID + `"}`,
	})
	if err != nil {
		return nil, err
	}
	for i := range resp {
		if resp[i].OrderID != orderID {
			continue
		}
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			return nil, err
		}
		var oType order.Type
		oType, err = e.getOrderType(resp[i].OrdType)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Date:            resp[i].Timestamp,
			Price:           resp[i].Price,
			Amount:          resp[i].OrderQty,
			ExecutedAmount:  resp[i].CumQty,
			RemainingAmount: resp[i].LeavesQty,
			Exchange:        e.Name,
			OrderID:         resp[i].OrderID,
			Side:            orderSideMap[resp[i].Side],
			Status:          orderStatus,
			Type:            oType,
			Pair:            pair,
			AssetType:       assetType,
		}, nil
	}
	return nil, fmt.Errorf("%w %v", order.ErrOrderNotFound, orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	resp, err := e.GetCryptoDepositAddress(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: resp,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	r := UserRequestWithdrawalParams{
		Address:  withdrawRequest.Crypto.Address,
		Amount:   withdrawRequest.Amount,
		Currency: withdrawRequest.Currency.String(),
		OtpToken: withdrawRequest.OneTimePassword,
	}
	if withdrawRequest.Crypto.FeeAmount > 0 {
		r.Fee = withdrawRequest.Crypto.FeeAmount
	}

	resp, err := e.UserRequestWithdrawal(ctx, r)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		Status: resp.Text,
		ID:     resp.Tx,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	params := OrdersRequest{
		Filter: "{\"open\":true}",
	}
	resp, err := e.GetOrders(ctx, &params)
	if err != nil {
		return nil, err
	}

	format, err := e.GetPairFormat(asset.PerpetualContract, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		var oType order.Type
		oType, err = e.getOrderType(resp[i].OrdType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		orderDetail := order.Detail{
			Date:            resp[i].Timestamp,
			Price:           resp[i].Price,
			Amount:          resp[i].OrderQty,
			ExecutedAmount:  resp[i].CumQty,
			RemainingAmount: resp[i].LeavesQty,
			Exchange:        e.Name,
			OrderID:         resp[i].OrderID,
			Side:            orderSideMap[resp[i].Side],
			Status:          orderStatus,
			Type:            oType,
			Pair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				format.Delimiter),
		}

		orders[i] = orderDetail
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	params := OrdersRequest{}
	resp, err := e.GetOrders(ctx, &params)
	if err != nil {
		return nil, err
	}

	format, err := e.GetPairFormat(asset.PerpetualContract, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}

		pair := currency.NewPairWithDelimiter(resp[i].Symbol, resp[i].SettlCurrency, format.Delimiter)

		var oType order.Type
		oType, err = e.getOrderType(resp[i].OrdType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}

		orderDetail := order.Detail{
			Price:                resp[i].Price,
			AverageExecutedPrice: resp[i].AvgPx,
			Amount:               resp[i].OrderQty,
			ExecutedAmount:       resp[i].CumQty,
			RemainingAmount:      resp[i].LeavesQty,
			Date:                 resp[i].TransactTime,
			CloseTime:            resp[i].Timestamp,
			Exchange:             e.Name,
			OrderID:              resp[i].OrderID,
			Side:                 orderSide,
			Status:               orderStatus,
			Type:                 oType,
			Pair:                 pair,
		}
		orderDetail.InferCostsAndTimes()

		orders[i] = orderDetail
	}
	return req.Filter(e.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	return e.websocketSendAuth(ctx)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// getOrderType derives an order type from bitmex int representation
func (e *Exchange) getOrderType(id int64) (order.Type, error) {
	o, ok := orderTypeMap[id]
	if !ok {
		return order.UnknownType, fmt.Errorf("unhandled order type for '%d': %w", id, order.ErrTypeIsInvalid)
	}
	return o, nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) || item == asset.Index {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	var resp []futures.Contract
	switch item {
	case asset.PerpetualContract:
		marketInfo, err := e.GetInstruments(ctx, &GenericRequestParams{
			Count:  countLimit,
			Filter: `{"typ": "` + perpetualContractID + `"}`,
		})
		if err != nil {
			return nil, err
		}
		for x := range marketInfo {
			cp, err := currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].QuoteCurrency)
			if err != nil {
				return nil, err
			}
			var s time.Time
			if marketInfo[x].Front != "" {
				s, err = time.Parse(time.RFC3339, marketInfo[x].Front)
				if err != nil {
					return nil, err
				}
			}
			var contractSettlementType futures.ContractSettlementType
			switch {
			case cp.Quote.Equal(currency.USDT):
				contractSettlementType = futures.Linear
			case cp.Quote.Equal(currency.USD):
				contractSettlementType = futures.Quanto
			default:
				contractSettlementType = futures.Inverse
			}
			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         currency.NewPair(cp.Base, marketInfo[x].SettlementCurrency),
				Asset:              item,
				StartDate:          s,
				IsActive:           marketInfo[x].State == "Open",
				Status:             marketInfo[x].State,
				Type:               futures.Perpetual,
				SettlementType:     contractSettlementType,
				SettlementCurrency: marketInfo[x].SettlementCurrency,
				Multiplier:         marketInfo[x].Multiplier,
				LatestRate: fundingrate.Rate{
					Time: marketInfo[x].FundingTimestamp,
					Rate: decimal.NewFromFloat(marketInfo[x].FundingRate),
				},
			})
		}
	case asset.Futures:
		marketInfo, err := e.GetInstruments(ctx, &GenericRequestParams{
			Count:  countLimit,
			Filter: `{"typ": "` + futuresID + `"}`,
		})
		if err != nil {
			return nil, err
		}

		for x := range marketInfo {
			cp, err := currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].Symbol[len(marketInfo[x].RootSymbol):])
			if err != nil {
				return nil, err
			}
			var startTime, endTime time.Time
			if marketInfo[x].Front != "" {
				startTime, err = time.Parse(time.RFC3339, marketInfo[x].Front)
				if err != nil {
					return nil, err
				}
			}
			if marketInfo[x].Expiry != "" {
				endTime, err = time.Parse(time.RFC3339, marketInfo[x].Expiry)
				if err != nil {
					return nil, err
				}
			}
			var ct futures.ContractType
			contractDuration := endTime.Sub(startTime)
			switch {
			case contractDuration <= kline.OneWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Weekly
			case contractDuration <= kline.TwoWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Fortnightly
			case contractDuration <= kline.OneMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Monthly
			case contractDuration <= kline.ThreeMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Quarterly
			case contractDuration <= kline.SixMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.HalfYearly
			case contractDuration <= kline.NineMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.NineMonthly
			case contractDuration <= kline.OneYear.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Yearly
			}
			contractSettlementType := futures.Inverse
			switch {
			case strings.Contains(cp.Quote.String(), "USDT"):
				contractSettlementType = futures.Linear
			case strings.Contains(cp.Quote.String(), "USD"):
				contractSettlementType = futures.Quanto
			}
			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp,
				Underlying:         currency.NewPair(cp.Base, marketInfo[x].SettlementCurrency),
				Asset:              item,
				StartDate:          startTime,
				EndDate:            endTime,
				IsActive:           marketInfo[x].State == "Open",
				Status:             marketInfo[x].State,
				Type:               ct,
				SettlementCurrency: marketInfo[x].SettlementCurrency,
				Multiplier:         marketInfo[x].Multiplier,
				SettlementType:     contractSettlementType,
			})
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}

	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}

	count := "1"
	if r.Pair.IsEmpty() {
		count = "500"
	} else {
		isPerp, err := e.IsPerpetualFutureCurrency(r.Asset, r.Pair)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
	}

	format, err := e.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := format.Format(r.Pair)
	rates, err := e.GetFullFundingHistory(ctx, fPair, count, "", "", "", true, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	// Bitmex returns historical rates from this endpoint, we only want the latest
	latestRateSymbol := make(map[string]bool)
	for i := range rates {
		if _, ok := latestRateSymbol[rates[i].Symbol]; ok {
			continue
		}
		latestRateSymbol[rates[i].Symbol] = true
		var nr time.Time
		nr, err = time.Parse(time.RFC3339, rates[i].FundingInterval)
		if err != nil {
			return nil, err
		}
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = e.MatchSymbolCheckEnabled(rates[i].Symbol, r.Asset, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		resp = append(resp, fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: rates[i].Timestamp,
				Rate: decimal.NewFromFloat(rates[i].FundingRate),
			},
			TimeOfNextRate: rates[i].Timestamp.Add(time.Duration(nr.Hour()) * time.Hour),
			TimeChecked:    time.Now(),
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualContract, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset == asset.Spot || k[i].Asset == asset.Index {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) != 1 {
		activeInstruments, err := e.GetActiveAndIndexInstruments(ctx)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.OpenInterest, 0, len(activeInstruments))
		for i := range activeInstruments {
			for _, a := range e.CurrencyPairs.GetAssetTypes(true) {
				var symbol currency.Pair
				var enabled bool
				symbol, enabled, err = e.MatchSymbolCheckEnabled(activeInstruments[i].Symbol, a, false)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !enabled {
					continue
				}
				var appendData bool
				for j := range k {
					if k[j].Pair().Equal(symbol) && k[j].Asset == a {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				resp = append(resp, futures.OpenInterest{
					Key:          key.NewExchangeAssetPair(e.Name, a, symbol),
					OpenInterest: activeInstruments[i].OpenInterest,
				})
			}
		}
		return resp, nil
	}
	_, isEnabled, err := e.MatchSymbolCheckEnabled(k[0].Pair().String(), k[0].Asset, false)
	if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
		return nil, err
	}
	if !isEnabled {
		return nil, fmt.Errorf("%w %v %v", currency.ErrPairNotEnabled, k[0].Asset, k[0].Pair())
	}
	symbolStr, err := e.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	instrument, err := e.GetInstrument(ctx, &GenericRequestParams{Symbol: symbolStr})
	if err != nil {
		return nil, err
	}
	if len(instrument) != 1 {
		return nil, fmt.Errorf("%w %v", currency.ErrPairNotFound, k[0].Pair())
	}
	resp := make([]futures.OpenInterest, 1)
	resp[0] = futures.OpenInterest{
		Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, k[0].Pair()),
		OpenInterest: instrument[0].OpenInterest,
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}
