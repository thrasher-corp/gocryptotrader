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
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets basic defaults
func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := b.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
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

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      btcMarketsAPIURL,
		exchange.WebsocketSpot: btcMarketsWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Websocket = websocket.NewManager()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in an exchange configuration and sets all parameters
func (b *BTCMarkets) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	err = b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsURL, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            btcMarketsWSURL,
		RunningURL:            wsURL,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.generateSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTCMarkets) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	markets, err := b.GetMarkets(ctx)
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
func (b *BTCMarkets) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = b.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return b.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *BTCMarkets) UpdateTickers(ctx context.Context, a asset.Item) error {
	allPairs, err := b.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	tickers, err := b.GetTickers(ctx, allPairs)
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
			ExchangeName: b.Name,
			AssetType:    a,
		}); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := b.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(b.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	book := &orderbook.Book{
		Exchange:          b.Name,
		Pair:              p,
		Asset:             assetType,
		PriceDuplication:  true,
		ValidateOrderbook: b.ValidateOrderbook,
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}
	// Retrieve level one book which is the top 50 ask and bids, this is not
	// cached.
	tempResp, err := b.GetOrderbook(ctx, fPair.String(), 1)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(tempResp.Bids))
	for x := range tempResp.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: tempResp.Bids[x].Volume,
			Price:  tempResp.Bids[x].Price,
		}
	}

	book.Asks = make(orderbook.Levels, len(tempResp.Asks))
	for y := range tempResp.Asks {
		book.Asks[y] = orderbook.Level{
			Amount: tempResp.Asks[y].Volume,
			Price:  tempResp.Asks[y].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (b *BTCMarkets) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var resp account.Holdings
	data, err := b.GetAccountBalance(ctx)
	if err != nil {
		return resp, err
	}
	var acc account.SubAccount
	acc.AssetType = assetType
	for x := range data {
		acc.Currencies = append(acc.Currencies, account.Balance{
			Currency: currency.NewCode(data[x].AssetName),
			Total:    data[x].Balance,
			Hold:     data[x].Locked,
			Free:     data[x].Available,
		})
	}
	resp.Accounts = append(resp.Accounts, acc)
	resp.Exchange = b.Name

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&resp, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return resp, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *BTCMarkets) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := b.ListWithdrawals(ctx, -1, -1, -1)
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
func (b *BTCMarkets) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	var tradeData []Trade
	tradeData, err = b.GetTrades(ctx, p.String(), 0, 0, 200)
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
			Exchange:     b.Name,
			TID:          tradeData[i].TradeID,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Timestamp,
		}
	}

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *BTCMarkets) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *BTCMarkets) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(b.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if s.Side.IsLong() {
		s.Side = order.Bid
	}
	if s.Side.IsShort() {
		s.Side = order.Ask
	}

	fPair, err := b.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	fOrderType, err := b.formatOrderType(s.Type)
	if err != nil {
		return nil, err
	}

	fOrderSide, err := b.formatOrderSide(s.Side)
	if err != nil {
		return nil, err
	}

	tempResp, err := b.NewOrder(ctx,
		s.Price,
		s.Amount,
		s.TriggerPrice,
		s.QuoteAmount,
		fPair.String(),
		fOrderType,
		fOrderSide,
		b.getTimeInForce(s),
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
			log.Errorf(log.ExchangeSys, "Exchange %s: OrderID: %s base amount conversion error: %s\n", b.Name, submitResp.OrderID, err)
		}
	}

	if tempResp.TargetAmount != 0 {
		err = submitResp.AdjustQuoteAmount(tempResp.TargetAmount)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Exchange %s: OrderID: %s quote amount conversion error: %s\n", b.Name, submitResp.OrderID, err)
		}
	}
	// With market orders the price is optional, so we can set it to the
	// actual price that was filled.
	submitResp.Price = tempResp.Price
	return submitResp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCMarkets) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	resp, err := b.ReplaceOrder(ctx, action.OrderID, action.ClientOrderID, action.Price, action.Amount)
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
func (b *BTCMarkets) CancelOrder(ctx context.Context, o *order.Cancel) error {
	err := o.Validate(o.StandardCancel())
	if err != nil {
		return err
	}
	_, err = b.RemoveOrder(ctx, o.OrderID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *BTCMarkets) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
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
	batchResp, err := b.CancelBatch(ctx, ids)
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
func (b *BTCMarkets) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	resp := order.CancelAllResponse{Status: map[string]string{}}
	orders, err := b.GetOrders(ctx, "", -1, -1, -1, true)
	if err != nil {
		return resp, err
	}

	orderIDs := make([]string, len(orders))
	for x := range orders {
		orderIDs[x] = orders[x].OrderID
	}
	for _, batch := range common.Batch(orderIDs, 20) {
		cancelResp, err := b.CancelBatch(ctx, batch)
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
func (b *BTCMarkets) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	var resp order.Detail
	o, err := b.FetchOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	resp.Exchange = b.Name
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
func (b *BTCMarkets) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	depositAddr, err := b.FetchDepositAddress(ctx, cryptocurrency, -1, -1, -1)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: depositAddr.Address,
		Tag:     depositAddr.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	a, err := b.RequestWithdraw(ctx,
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
func (b *BTCMarkets) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.Currency != currency.AUD {
		return nil, errors.New("only aud is supported for withdrawals")
	}
	a, err := b.RequestWithdraw(ctx,
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
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCMarkets) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		allPairs, err := b.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
		req.Pairs = append(req.Pairs, allPairs...)
	}

	var resp []order.Detail
	for x := range req.Pairs {
		fPair, err := b.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		tempData, err := b.GetOrders(ctx, fPair.String(), -1, -1, -1, true)
		if err != nil {
			return resp, err
		}
		for y := range tempData {
			var tempResp order.Detail
			tempResp.Exchange = b.Name
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
					b.Name,
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
					b.Name,
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
	return req.Filter(b.Name, resp), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	var tempResp order.Detail
	var tempArray []string
	if len(req.Pairs) == 0 {
		orders, err := b.GetOrders(ctx, "", -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for x := range orders {
			tempArray = append(tempArray, orders[x].OrderID)
		}
	}
	for y := range req.Pairs {
		fPair, err := b.FormatExchangeCurrency(req.Pairs[y], asset.Spot)
		if err != nil {
			return nil, err
		}

		orders, err := b.GetOrders(ctx, fPair.String(), -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for z := range orders {
			tempArray = append(tempArray, orders[z].OrderID)
		}
	}
	for _, batch := range common.Batch(tempArray, 50) {
		tempData, err := b.GetBatchTrades(ctx, batch)
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

			tempResp.Exchange = b.Name
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
	return req.Filter(b.Name, resp), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *BTCMarkets) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	if err != nil {
		if b.CheckTransientError(err) == nil {
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
func (b *BTCMarkets) FormatExchangeKlineInterval(in kline.Interval) string {
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
func (b *BTCMarkets) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	candles, err := b.GetMarketCandles(ctx,
		req.RequestFormatted.String(),
		b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (b *BTCMarkets) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []CandleResponse
		candles, err = b.GetMarketCandles(ctx,
			req.RequestFormatted.String(),
			b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (b *BTCMarkets) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return b.GetCurrentServerTime(ctx)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (b *BTCMarkets) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	markets, err := b.GetMarkets(ctx)
	if err != nil {
		return err
	}

	limits := make([]order.MinMaxLevel, len(markets))
	for x := range markets {
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(markets[x].BaseAsset, markets[x].QuoteAsset)
		if err != nil {
			return err
		}

		limits[x] = order.MinMaxLevel{
			Pair:                    pair,
			Asset:                   asset.Spot,
			MinimumBaseAmount:       markets[x].MinOrderAmount,
			MaximumBaseAmount:       markets[x].MaxOrderAmount,
			AmountStepIncrementSize: math.Pow(10, -markets[x].AmountDecimals),
			PriceStepIncrementSize:  math.Pow(10, -markets[x].PriceDecimals),
		}
	}
	return b.LoadLimits(limits)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (b *BTCMarkets) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (b *BTCMarkets) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *BTCMarkets) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Base.Upper().String(), nil
}
