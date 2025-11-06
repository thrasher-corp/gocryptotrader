package btcmarkets

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
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

// SetDefaults sets basic defaults
func (e *Exchange) SetDefaults() {
	e.Name = "BTC Markets"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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
				CancelOrder:         true,
				SubmitOrder:         true,
				UserTradeHistory:    true,
				CryptoWithdrawal:    true,
				FiatWithdraw:        true,
				TradeFee:            true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
				ModifyOrder:         true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
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
					kline.IntervalCapacity{Interval: kline.ThreeHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      btcMarketsAPIURL,
		exchange.WebsocketSpot: btcMarketsWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in an exchange configuration and sets all parameters
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

	wsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            btcMarketsWSURL,
		RunningURL:            wsURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
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
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	markets, err := e.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, len(markets))
	for x := range markets {
		pairs[x] = markets[x].MarketID
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
	allPairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	tickers, err := e.GetTickers(ctx, allPairs)
	if err != nil {
		return err
	}

	if len(allPairs) != len(tickers) {
		return errors.New("enabled pairs differ from returned tickers")
	}

	for x := range tickers {
		if err := ticker.ProcessTicker(&ticker.Price{
			Pair:         tickers[x].MarketID,
			Last:         tickers[x].LastPrice,
			High:         tickers[x].High24h,
			Low:          tickers[x].Low24h,
			Bid:          tickers[x].BestBID,
			Ask:          tickers[x].BestAsk,
			Volume:       tickers[x].Volume,
			LastUpdated:  time.Now(),
			ExchangeName: e.Name,
			AssetType:    a,
		}); err != nil {
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
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	// Retrieve level one book which is the top 50 ask and bids, this is not
	// cached.
	resp, err := e.GetOrderbook(ctx, fPair.String(), 1)
	if err != nil {
		return nil, err
	}

	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		PriceDuplication:  true,
		ValidateOrderbook: e.ValidateOrderbook,
		Asks:              resp.Asks,
		Bids:              resp.Bids,
	}
	if err := ob.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	resp, err := e.GetAccountBalance(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp {
		subAccts[0].Balances.Set(resp[i].AssetName, accounts.Balance{
			Total: resp[i].Balance,
			Hold:  resp[i].Locked,
			Free:  resp[i].Available,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.ListWithdrawals(ctx, -1, -1, -1)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for i := range withdrawals {
		if c.IsEmpty() || c.Equal(withdrawals[i].AssetName) {
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          withdrawals[i].Status,
				TransferID:      withdrawals[i].ID,
				Description:     withdrawals[i].Description,
				Timestamp:       withdrawals[i].CreationTime,
				Currency:        withdrawals[i].AssetName.String(),
				Amount:          withdrawals[i].Amount,
				Fee:             withdrawals[i].Fee,
				TransferType:    withdrawals[i].RequestType,
				CryptoToAddress: withdrawals[i].PaymentDetails.Address,
			})
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	var tradeData []Trade
	tradeData, err = e.GetTrades(ctx, p.String(), 0, 0, 200)
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		var side order.Side
		if tradeData[i].Side != "" {
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          tradeData[i].TradeID,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Timestamp,
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if s.Side.IsLong() {
		s.Side = order.Bid
	}
	if s.Side.IsShort() {
		s.Side = order.Ask
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	fOrderType, err := e.formatOrderType(s.Type)
	if err != nil {
		return nil, err
	}

	fOrderSide, err := e.formatOrderSide(s.Side)
	if err != nil {
		return nil, err
	}

	tempResp, err := e.NewOrder(ctx,
		s.Price,
		s.Amount,
		s.TriggerPrice,
		s.QuoteAmount,
		fPair.String(),
		fOrderType,
		fOrderSide,
		e.getTimeInForce(s),
		"",
		s.ClientID,
		s.TimeInForce.Is(order.PostOnly))
	if err != nil {
		return nil, err
	}

	submitResp, err := s.DeriveSubmitResponse(tempResp.OrderID)
	if err != nil {
		return nil, err
	}

	if tempResp.Amount != 0 {
		err = submitResp.AdjustBaseAmount(tempResp.Amount)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Exchange %s: OrderID: %s base amount conversion error: %s\n", e.Name, submitResp.OrderID, err)
		}
	}

	if tempResp.TargetAmount != 0 {
		err = submitResp.AdjustQuoteAmount(tempResp.TargetAmount)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Exchange %s: OrderID: %s quote amount conversion error: %s\n", e.Name, submitResp.OrderID, err)
		}
	}
	// With market orders the price is optional, so we can set it to the
	// actual price that was filled.
	submitResp.Price = tempResp.Price
	return submitResp, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.ReplaceOrder(ctx, action.OrderID, action.ClientOrderID, action.Price, action.Amount)
	if err != nil {
		return nil, err
	}
	mod, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	mod.Pair = resp.MarketID
	mod.Side, err = order.StringToOrderSide(resp.Side)
	if err != nil {
		return nil, err
	}
	mod.Type, err = order.StringToOrderType(resp.Type)
	if err != nil {
		return nil, err
	}
	mod.Status, err = order.StringToOrderStatus(resp.Status)
	if err != nil {
		return nil, err
	}
	mod.OrderID = resp.OrderID
	mod.LastUpdated = resp.CreationTime
	mod.Price = resp.Price
	mod.Amount = resp.Amount
	mod.RemainingAmount = resp.OpenAmount
	return mod, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	err := o.Validate(o.StandardCancel())
	if err != nil {
		return err
	}
	_, err = e.RemoveOrder(ctx, o.OrderID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	ids := make([]string, len(o))
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			return nil, order.ErrClientOrderIDNotSupported
		case o[i].OrderID != "":
			ids[i] = o[i].OrderID
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	batchResp, err := e.CancelBatch(ctx, ids)
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for i := range batchResp.CancelOrders {
		resp.Status[batchResp.CancelOrders[i].OrderID] = "success"
	}
	for i := range batchResp.UnprocessedRequests {
		resp.Status[batchResp.UnprocessedRequests[i].RequestID] = batchResp.UnprocessedRequests[i].Code + " - " + batchResp.UnprocessedRequests[i].Message
	}

	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	resp := order.CancelAllResponse{Status: map[string]string{}}
	orders, err := e.GetOrders(ctx, "", -1, -1, -1, true)
	if err != nil {
		return resp, err
	}

	orderIDs := make([]string, len(orders))
	for x := range orders {
		orderIDs[x] = orders[x].OrderID
	}
	for _, batch := range common.Batch(orderIDs, 20) {
		cancelResp, err := e.CancelBatch(ctx, batch)
		if err != nil {
			return resp, err
		}
		for _, r := range cancelResp.CancelOrders {
			resp.Status[r.OrderID] = "Success"
		}
		for _, r := range cancelResp.UnprocessedRequests {
			resp.Status[r.RequestID] = "Cancellation Failed"
		}
	}
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	var resp order.Detail
	o, err := e.FetchOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	resp.Exchange = e.Name
	resp.OrderID = orderID
	resp.Pair = o.MarketID
	resp.Price = o.Price
	resp.Date = o.CreationTime
	resp.ExecutedAmount = o.Amount - o.OpenAmount
	resp.Side = order.Bid
	if o.Side == ask {
		resp.Side = order.Ask
	}
	switch o.Type {
	case limit:
		resp.Type = order.Limit
	case market:
		resp.Type = order.Market
	case stopLimit:
		resp.Type = order.Stop
	case stop:
		resp.Type = order.Stop
	case takeProfit:
		resp.Type = order.TakeProfit
	default:
		resp.Type = order.UnknownType
	}
	resp.RemainingAmount = o.OpenAmount
	switch o.Status {
	case orderAccepted:
		resp.Status = order.Active
	case orderPlaced:
		resp.Status = order.Active
	case orderPartiallyMatched:
		resp.Status = order.PartiallyFilled
	case orderFullyMatched:
		resp.Status = order.Filled
	case orderCancelled:
		resp.Status = order.Cancelled
	case orderPartiallyCancelled:
		resp.Status = order.PartiallyCancelled
	case orderFailed:
		resp.Status = order.Rejected
	default:
		resp.Status = order.UnknownStatus
	}
	return &resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	depositAddr, err := e.FetchDepositAddress(ctx, cryptocurrency, -1, -1, -1)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: depositAddr.Address,
		Tag:     depositAddr.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	a, err := e.RequestWithdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Amount,
		withdrawRequest.Crypto.Address,
		"",
		"",
		"",
		"")
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     a.ID,
		Status: a.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.Currency != currency.AUD {
		return nil, errors.New("only aud is supported for withdrawals")
	}
	a, err := e.RequestWithdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Amount,
		"",
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.AccountNumber,
		withdrawRequest.Fiat.Bank.BSBNumber,
		withdrawRequest.Fiat.Bank.BankName)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     a.ID,
		Status: a.Status,
	}, nil
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
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
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
	if len(req.Pairs) == 0 {
		allPairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
		req.Pairs = append(req.Pairs, allPairs...)
	}

	var resp []order.Detail
	for x := range req.Pairs {
		fPair, err := e.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		tempData, err := e.GetOrders(ctx, fPair.String(), -1, -1, -1, true)
		if err != nil {
			return resp, err
		}
		for y := range tempData {
			var tempResp order.Detail
			tempResp.Exchange = e.Name
			tempResp.Pair = req.Pairs[x]
			tempResp.OrderID = tempData[y].OrderID
			tempResp.Side = order.Bid
			if tempData[y].Side == ask {
				tempResp.Side = order.Ask
			}
			tempResp.Date = tempData[y].CreationTime

			switch tempData[y].Type {
			case limit:
				tempResp.Type = order.Limit
			case market:
				tempResp.Type = order.Market
			default:
				log.Errorf(log.ExchangeSys,
					"%s unknown order type %s getting order",
					e.Name,
					tempData[y].Type)
				tempResp.Type = order.UnknownType
			}
			switch tempData[y].Status {
			case orderAccepted:
				tempResp.Status = order.Active
			case orderPlaced:
				tempResp.Status = order.Active
			case orderPartiallyMatched:
				tempResp.Status = order.PartiallyFilled
			default:
				log.Errorf(log.ExchangeSys,
					"%s unexpected status %s on order %v",
					e.Name,
					tempData[y].Status,
					tempData[y].OrderID)
				tempResp.Status = order.UnknownStatus
			}
			tempResp.Price = tempData[y].Price
			tempResp.Amount = tempData[y].Amount
			tempResp.ExecutedAmount = tempData[y].Amount - tempData[y].OpenAmount
			tempResp.RemainingAmount = tempData[y].OpenAmount
			resp = append(resp, tempResp)
		}
	}
	return req.Filter(e.Name, resp), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	var tempResp order.Detail
	var tempArray []string
	if len(req.Pairs) == 0 {
		orders, err := e.GetOrders(ctx, "", -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for x := range orders {
			tempArray = append(tempArray, orders[x].OrderID)
		}
	}
	for y := range req.Pairs {
		fPair, err := e.FormatExchangeCurrency(req.Pairs[y], asset.Spot)
		if err != nil {
			return nil, err
		}

		orders, err := e.GetOrders(ctx, fPair.String(), -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for z := range orders {
			tempArray = append(tempArray, orders[z].OrderID)
		}
	}
	for _, batch := range common.Batch(tempArray, 50) {
		tempData, err := e.GetBatchTrades(ctx, batch)
		if err != nil {
			return resp, err
		}
		for c := range tempData.Orders {
			switch tempData.Orders[c].Status {
			case orderFailed:
				tempResp.Status = order.Rejected
			case orderPartiallyCancelled:
				tempResp.Status = order.PartiallyCancelled
			case orderCancelled:
				tempResp.Status = order.Cancelled
			case orderFullyMatched:
				tempResp.Status = order.Filled
			case orderPartiallyMatched:
				continue
			case orderPlaced:
				continue
			case orderAccepted:
				continue
			}

			tempResp.Exchange = e.Name
			tempResp.Pair = tempData.Orders[c].MarketID
			tempResp.Side = order.Bid
			if tempData.Orders[c].Side == ask {
				tempResp.Side = order.Ask
			}
			tempResp.OrderID = tempData.Orders[c].OrderID
			tempResp.Date = tempData.Orders[c].CreationTime
			tempResp.Price = tempData.Orders[c].Price
			tempResp.Amount = tempData.Orders[c].Amount
			tempResp.ExecutedAmount = tempData.Orders[c].Amount - tempData.Orders[c].OpenAmount
			tempResp.RemainingAmount = tempData.Orders[c].OpenAmount
			tempResp.InferCostsAndTimes()
			resp = append(resp, tempResp)
		}
	}
	return req.Filter(e.Name, resp), nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	if err != nil {
		if e.CheckTransientError(err) == nil {
			return nil
		}
		// Check for specific auth errors; all other errors can be disregarded
		// as this does not affect authenticated requests.
		if strings.Contains(err.Error(), "InvalidAPIKey") ||
			strings.Contains(err.Error(), "InvalidAuthTimestamp") ||
			strings.Contains(err.Error(), "InvalidAuthSignature") ||
			strings.Contains(err.Error(), "InsufficientAPIPermission") {
			return err
		}
	}

	return nil
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin:
		return "1m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1h"
	case kline.SixHour:
		return "6h"
	case kline.OneDay:
		return "1d"
	case kline.OneWeek:
		return "1w"
	case kline.OneMonth:
		return "1mo"
	}
	return in.Short()
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	candles, err := e.GetMarketCandles(ctx,
		req.RequestFormatted.String(),
		e.FormatExchangeKlineInterval(req.ExchangeInterval),
		req.Start,
		req.End,
		-1,
		-1,
		-1)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].Timestamp,
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
		var candles []CandleResponse
		candles, err = e.GetMarketCandles(ctx,
			req.RequestFormatted.String(),
			e.FormatExchangeKlineInterval(req.ExchangeInterval),
			req.RangeHolder.Ranges[x].Start.Time,
			req.RangeHolder.Ranges[x].End.Time,
			-1,
			-1,
			-1)
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].Timestamp,
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

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetCurrentServerTime(ctx)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	markets, err := e.GetMarkets(ctx)
	if err != nil {
		return err
	}

	l := make([]limits.MinMaxLevel, len(markets))
	for x := range markets {
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(markets[x].BaseAsset, markets[x].QuoteAsset)
		if err != nil {
			return err
		}

		l[x] = limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.Spot, pair),
			MinimumBaseAmount:       markets[x].MinOrderAmount,
			MaximumBaseAmount:       markets[x].MaxOrderAmount,
			AmountStepIncrementSize: math.Pow(10, -markets[x].AmountDecimals),
			PriceStepIncrementSize:  math.Pow(10, -markets[x].PriceDecimals),
		}
	}
	return limits.Load(l)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Base.Upper().String(), nil
}
