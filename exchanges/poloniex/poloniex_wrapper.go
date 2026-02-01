package poloniex

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

// SetDefaults sets default settings for poloniex
func (e *Exchange) SetDefaults() {
	e.Name = "Poloniex"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 500,
			},
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              poloniexAPIURL,
		exchange.RestSpotSupplementary: poloniexAltAPIUrl,
		exchange.WebsocketSpot:         poloniexWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
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

	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            poloniexWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.GenerateDefaultSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	// TODO: Upgrade to new API version for fetching operational pairs.
	resp, err := e.GetTicker(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(resp))
	for key, info := range resp {
		// Poloniex returns 0 for highest bid and lowest ask if support has been
		// dropped from the front end. We don't want to add these pairs.
		if info.HighestBid == 0 || info.LowestAsk == 0 {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairFromString(key)
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

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := e.GetTicker(ctx)
	if err != nil {
		return err
	}

	enabledPairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range enabledPairs {
		fPair, err := e.FormatExchangeCurrency(enabledPairs[i], a)
		if err != nil {
			return err
		}
		curr := fPair.String()
		if _, ok := tick[curr]; !ok {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[i],
			Ask:          tick[curr].LowestAsk,
			Bid:          tick[curr].HighestBid,
			High:         tick[curr].High24Hr,
			Last:         tick[curr].Last,
			Low:          tick[curr].Low24Hr,
			Volume:       tick[curr].BaseVolume,
			QuoteVolume:  tick[curr].QuoteVolume,
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, currencyPair currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, currencyPair, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	callingBook := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	orderbookNew, err := e.GetOrderbook(ctx, "", poloniexMaxOrderbookDepth)
	if err != nil {
		return callingBook, err
	}

	enabledPairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return callingBook, err
	}
	for i := range enabledPairs {
		pFmt, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		fP := enabledPairs[i].Format(pFmt)
		data, ok := orderbookNew.Data[fP.Base.String()+fP.Delimiter+fP.Quote.String()]
		if !ok {
			data, ok = orderbookNew.Data[fP.Quote.String()+fP.Delimiter+fP.Base.String()]
			if !ok {
				continue
			}
		}
		book := &orderbook.Book{
			Exchange:          e.Name,
			Pair:              enabledPairs[i],
			Asset:             assetType,
			ValidateOrderbook: e.ValidateOrderbook,
		}

		book.Bids = make(orderbook.Levels, len(data.Bids))
		for y := range data.Bids {
			book.Bids[y] = orderbook.Level{
				Amount: data.Bids[y].Amount,
				Price:  data.Bids[y].Price,
			}
		}

		book.Asks = make(orderbook.Levels, len(data.Asks))
		for y := range data.Asks {
			book.Asks[y] = orderbook.Level{
				Amount: data.Asks[y].Amount,
				Price:  data.Asks[y].Price,
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	resp, err := e.GetBalances(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for curr, bal := range resp {
		subAccts[0].Balances.Set(curr, accounts.Balance{Total: bal})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	end := time.Now()
	walletActivity, err := e.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(walletActivity.Deposits))
	for i := range walletActivity.Deposits {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          walletActivity.Deposits[i].Status,
			Timestamp:       walletActivity.Deposits[i].Timestamp.Time(),
			Currency:        walletActivity.Deposits[i].Currency.String(),
			Amount:          walletActivity.Deposits[i].Amount,
			CryptoToAddress: walletActivity.Deposits[i].Address,
			CryptoTxID:      walletActivity.Deposits[i].TransactionID,
		}
	}
	for i := range walletActivity.Withdrawals {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          walletActivity.Withdrawals[i].Status,
			Timestamp:       walletActivity.Withdrawals[i].Timestamp.Time(),
			Currency:        walletActivity.Withdrawals[i].Currency.String(),
			Amount:          walletActivity.Withdrawals[i].Amount,
			Fee:             walletActivity.Withdrawals[i].Fee,
			CryptoToAddress: walletActivity.Withdrawals[i].Address,
			CryptoTxID:      walletActivity.Withdrawals[i].TransactionID,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	end := time.Now()
	withdrawals, err := e.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "withdrawals")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals.Withdrawals))
	for i := range withdrawals.Withdrawals {
		if !withdrawals.Withdrawals[i].Currency.Equal(c) {
			continue
		}
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals.Withdrawals[i].Status,
			Timestamp:       withdrawals.Withdrawals[i].Timestamp.Time(),
			Currency:        withdrawals.Withdrawals[i].Currency.String(),
			Amount:          withdrawals.Withdrawals[i].Amount,
			Fee:             withdrawals.Withdrawals[i].Fee,
			CryptoToAddress: withdrawals.Withdrawals[i].Address,
			CryptoTxID:      withdrawals.Withdrawals[i].TransactionID,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return e.GetHistoricTrades(ctx, pair, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	pair, err = e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	ts := timestampStart
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = e.GetTradeHistory(ctx,
			pair.String(),
			ts.Unix(),
			timestampEnd.Unix())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var tt time.Time
			tt, err = time.Parse(time.DateTime, tradeData[i].Date)
			if err != nil {
				return nil, err
			}
			if (tt.Before(timestampStart) && !timestampStart.IsZero()) || (tt.After(timestampEnd) && !timestampEnd.IsZero()) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Type)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				TID:          tradeData[i].TradeID,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Rate,
				Amount:       tradeData[i].Amount,
				Timestamp:    tt,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tt) {
					// reached end of trades to crawl
					break allTrades
				}
				if timestampStart.IsZero() {
					break allTrades
				}
				ts = tt
			}
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	response, err := e.PlaceOrder(ctx,
		fPair.String(),
		s.Price,
		s.Amount,
		false,
		s.Type == order.Market,
		s.Side.IsLong())
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatInt(response.OrderNumber, 10))
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}

	oID, err := strconv.ParseInt(action.OrderID, 10, 64)
	if err != nil {
		return nil, err
	}

	resp, err := e.MoveOrder(ctx,
		oID,
		action.Price,
		action.Amount,
		action.TimeInForce.Is(order.PostOnly),
		action.TimeInForce.Is(order.ImmediateOrCancel))
	if err != nil {
		return nil, err
	}

	modResp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	modResp.OrderID = strconv.FormatInt(resp.OrderNumber, 10)
	return modResp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	return e.CancelExistingOrder(ctx, orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	orderIDs := make([]string, 0, len(o))
	clientOrderIDs := make([]string, 0, len(o))
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			clientOrderIDs = append(clientOrderIDs, o[i].ClientOrderID)
		case o[i].OrderID != "":
			orderIDs = append(orderIDs, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	cancelledOrders, err := e.CancelMultipleOrdersByIDs(ctx, orderIDs, clientOrderIDs)
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for i := range cancelledOrders {
		if cancelledOrders[i].ClientOrderID != "" {
			resp.Status[cancelledOrders[i].ClientOrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
			continue
		}
		resp.Status[cancelledOrders[i].OrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := e.GetOpenOrdersForAllCurrencies(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for key := range openOrders.Data {
		for i := range openOrders.Data[key] {
			err = e.CancelExistingOrder(ctx, openOrders.Data[key][i].OrderNumber)
			if err != nil {
				id := strconv.FormatInt(openOrders.Data[key][i].OrderNumber, 10)
				cancelAllOrdersResponse.Status[id] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, _ asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	orderInfo := order.Detail{
		Exchange: e.Name,
		Pair:     pair,
	}

	trades, err := e.GetAuthenticatedOrderTrades(ctx, orderID)
	if err != nil && !strings.Contains(err.Error(), "Order not found") {
		return nil, err
	}

	for i := range trades {
		var tradeHistory order.TradeHistory
		tradeHistory.Exchange = e.Name
		tradeHistory.Side, err = order.StringToOrderSide(trades[i].Type)
		if err != nil {
			return nil, err
		}
		tradeHistory.TID = trades[i].GlobalTradeID
		tradeHistory.Timestamp, err = time.Parse(time.DateTime, trades[i].Date)
		if err != nil {
			return nil, err
		}
		tradeHistory.Price = trades[i].Rate
		tradeHistory.Amount = trades[i].Amount
		tradeHistory.Total = trades[i].Total
		tradeHistory.Fee = trades[i].Fee
		orderInfo.Trades = append(orderInfo.Trades, tradeHistory)
	}

	resp, err := e.GetAuthenticatedOrderStatus(ctx, orderID)
	if err != nil {
		if len(orderInfo.Trades) > 0 { // on closed orders return trades only
			if strings.Contains(err.Error(), "Order not found") {
				orderInfo.Status = order.Closed
			}
			return &orderInfo, nil
		}
		return nil, err
	}

	if orderInfo.Status, err = order.StringToOrderStatus(resp.Status); err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
	}
	orderInfo.Price = resp.Rate
	orderInfo.Amount = resp.Amount
	orderInfo.Cost = resp.Total
	orderInfo.Fee = resp.Fee
	orderInfo.QuoteAmount = resp.StartingAmount

	orderInfo.Side, err = order.StringToOrderSide(resp.Type)
	if err != nil {
		return nil, err
	}

	orderInfo.Date, err = time.Parse(time.DateTime, resp.Date)
	if err != nil {
		return nil, err
	}

	return &orderInfo, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	depositAddrs, err := e.GetDepositAddresses(ctx)
	if err != nil {
		return nil, err
	}

	// Some coins use a main address, so we must use this in conjunction with the returned
	// deposit address to produce the full deposit address and tag
	currencies, err := e.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	coinParams, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
	}

	// Handle coins with payment ID's like XRP
	var address, tag string
	if coinParams.CurrencyType == "address-payment-id" && coinParams.DepositAddress != "" {
		address = coinParams.DepositAddress
		tag, ok = depositAddrs.Addresses[cryptocurrency.Upper().String()]
		if !ok {
			newAddr, err := e.GenerateNewAddress(ctx, cryptocurrency.Upper().String())
			if err != nil {
				return nil, err
			}
			tag = newAddr
		}
		return &deposit.Address{
			Address: address,
			Tag:     tag,
		}, nil
	}

	// Handle coins like BTC or multichain coins
	targetCurrency := cryptocurrency.String()
	if chain != "" && !strings.EqualFold(chain, cryptocurrency.String()) {
		targetCurrency = chain
	}

	address, ok = depositAddrs.Addresses[strings.ToUpper(targetCurrency)]
	if !ok {
		if len(coinParams.ChildChains) > 1 && chain != "" && !slices.Contains(coinParams.ChildChains, targetCurrency) {
			// rather than assume, return an error
			return nil, fmt.Errorf("currency %s has %v chains available, one of these must be specified",
				cryptocurrency,
				coinParams.ChildChains)
		}

		coinParams, ok = currencies[strings.ToUpper(targetCurrency)]
		if !ok {
			return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
		}
		if coinParams.WithdrawalDepositDisabled == 1 {
			return nil, fmt.Errorf("deposits and withdrawals for %v are currently disabled", targetCurrency)
		}

		newAddr, err := e.GenerateNewAddress(ctx, targetCurrency)
		if err != nil {
			return nil, err
		}
		address = newAddr
	}
	return &deposit.Address{Address: address}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	targetCurrency := withdrawRequest.Currency.String()
	if withdrawRequest.Crypto.Chain != "" {
		targetCurrency = withdrawRequest.Crypto.Chain
	}
	v, err := e.Withdraw(ctx, targetCurrency, withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v.Response,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := e.GetOpenOrdersForAllCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}
		for i := range resp.Data[key] {
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(resp.Data[key][i].Type)
			if err != nil {
				return nil, err
			}
			var orderDate time.Time
			orderDate, err = time.Parse(time.DateTime, resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					e.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			orders = append(orders, order.Detail{
				OrderID:  strconv.FormatInt(resp.Data[key][i].OrderNumber, 10),
				Side:     orderSide,
				Amount:   resp.Data[key][i].Amount,
				Date:     orderDate,
				Price:    resp.Data[key][i].Rate,
				Pair:     symbol,
				Exchange: e.Name,
			})
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := e.GetAuthenticatedTradeHistory(ctx,
		req.StartTime.Unix(),
		req.EndTime.Unix(),
		10000)
	if err != nil {
		return nil, err
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data[key] {
			orderSide, err := order.StringToOrderSide(resp.Data[key][i].Type)
			if err != nil {
				return nil, err
			}
			orderDate, err := time.Parse(time.DateTime,
				resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					e.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			detail := order.Detail{
				OrderID:              resp.Data[key][i].GlobalTradeID,
				Side:                 orderSide,
				Amount:               resp.Data[key][i].Amount,
				ExecutedAmount:       resp.Data[key][i].Amount,
				Date:                 orderDate,
				Price:                resp.Data[key][i].Rate,
				AverageExecutedPrice: resp.Data[key][i].Rate,
				Pair:                 pair,
				Status:               order.Filled,
				Exchange:             e.Name,
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
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

	resp, err := e.GetChartData(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End,
		e.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(resp))
	for x := range resp {
		timeSeries[x] = kline.Candle{
			Time:   resp[x].Date.Time(),
			Open:   resp[x].Open.Float64(),
			High:   resp[x].High.Float64(),
			Low:    resp[x].Low.Float64(),
			Close:  resp[x].Close.Float64(),
			Volume: resp[x].Volume.Float64(),
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
	for i := range req.RangeHolder.Ranges {
		resp, err := e.GetChartData(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[i].Start.Time,
			req.RangeHolder.Ranges[i].End.Time,
			e.FormatExchangeKlineInterval(req.ExchangeInterval))
		if err != nil {
			return nil, err
		}
		for x := range resp {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   resp[x].Date.Time(),
				Open:   resp[x].Open.Float64(),
				High:   resp[x].High.Float64(),
				Low:    resp[x].Low.Float64(),
				Close:  resp[x].Close.Float64(),
				Volume: resp[x].Volume.Float64(),
			})
		}
	}

	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencies, err := e.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	curr, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, errors.New("unable to locate currency in map")
	}

	return curr.ChildChains, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetTimestamp(ctx)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	// TODO: implement with API upgrade
	return nil, common.ErrNotYetImplemented
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	// TODO: implement with API upgrade
	return nil, common.ErrNotYetImplemented
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
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return poloniexAPIURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		cp.Delimiter = ""
		return poloniexAPIURL + tradeFutures + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
