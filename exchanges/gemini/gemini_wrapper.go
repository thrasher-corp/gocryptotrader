package gemini

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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

// SetDefaults sets package defaults for gemini exchange
func (g *Gemini) SetDefaults() {
	g.Name = "Gemini"
	g.Enabled = true
	g.Verbose = true
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Uppercase: true,
		Separator: ",",
	}
	configFmt := &currency.PairFormat{
		Uppercase: true,
		Delimiter: currency.DashDelimiter,
	}
	err := g.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	g.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching:      true,
				TradeFetching:          true,
				AuthenticatedEndpoints: true,
				MessageSequenceNumbers: true,
				KlineFetching:          true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	g.Requester, err = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.API.Endpoints = g.NewEndpoints()
	err = g.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      geminiAPIURL,
		exchange.WebsocketSpot: geminiWebsocketEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.Websocket = websocket.NewManager()
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets exchange configuration parameters
func (g *Gemini) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}
	err = g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if exch.UseSandbox {
		err = g.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), geminiSandboxAPIURL)
		if err != nil {
			log.Errorln(log.ExchangeSys, err)
		}
	}

	wsRunningURL, err := g.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = g.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            geminiWebsocketEndpoint,
		RunningURL:            wsRunningURL,
		Connector:             g.WsConnect,
		Subscriber:            g.Subscribe,
		Unsubscriber:          g.Unsubscribe,
		GenerateSubscriptions: g.generateSubscriptions,
		Features:              &g.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  geminiWebsocketEndpoint + "/v2/" + geminiWsMarketData,
	})
	if err != nil {
		return err
	}

	return g.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  geminiWebsocketEndpoint + "/v1/" + geminiWsOrderEvents,
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gemini) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !g.SupportsAsset(a) {
		return nil, asset.ErrNotSupported
	}

	details, err := g.GetSymbolDetails(ctx, "all")
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(details))
	for i := range details {
		status := strings.ToLower(details[i].Status)
		if status != "open" && status != "limit_only" {
			continue
		}
		if !strings.EqualFold(details[i].ContractType, "vanilla") {
			// TODO: add support for futures
			continue
		}

		cp, err := currency.NewPairFromStrings(details[i].BaseCurrency, details[i].Symbol[len(details[i].BaseCurrency):])
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, cp)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gemini) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := g.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = g.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return g.EnsureOnePairEnabled()
}

// UpdateAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = g.Name
	accountBalance, err := g.GetBalances(ctx)
	if err != nil {
		return response, err
	}

	currencies := make([]account.Balance, len(accountBalance))
	for i := range accountBalance {
		currencies[i] = account.Balance{
			Currency: currency.NewCode(accountBalance[i].Currency),
			Total:    accountBalance[i].Amount,
			Hold:     accountBalance[i].Amount - accountBalance[i].Available,
			Free:     accountBalance[i].Available,
		}
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: currencies,
	})

	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (g *Gemini) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := g.GetTicker(ctx, fPair.String())
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		High:         tick.High,
		Low:          tick.Low,
		Bid:          tick.Bid,
		Ask:          tick.Ask,
		Open:         tick.Open,
		Close:        tick.Close,
		Pair:         fPair,
		ExchangeName: g.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(g.Name, fPair, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := g.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          g.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: g.ValidateOrderbook,
	}
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := g.GetOrderbook(ctx, fPair.String(), url.Values{})
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(g.Name, fPair, assetType)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	transfers, err := g.Transfers(ctx, currency.EMPTYCODE, time.Time{}, 50, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(transfers))
	for i := range transfers {
		resp[i] = exchange.FundingHistory{
			Status:          transfers[i].Status,
			TransferID:      transfers[i].WithdrawalID,
			Timestamp:       transfers[i].Timestamp.Time(),
			Currency:        transfers[i].Currency.String(),
			Amount:          transfers[i].Amount,
			Fee:             transfers[i].FeeAmount,
			TransferType:    transfers[i].Type,
			CryptoToAddress: transfers[i].Destination,
			CryptoTxID:      transfers[i].TxHash,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gemini) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if err := g.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	transfers, err := g.Transfers(ctx, c, time.Time{}, 50, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(transfers))
	for i := range transfers {
		if transfers[i].Type != "Withdrawal" {
			continue
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          transfers[i].Status,
			TransferID:      transfers[i].WithdrawalID,
			Timestamp:       transfers[i].Timestamp.Time(),
			Currency:        transfers[i].Currency.String(),
			Amount:          transfers[i].Amount,
			Fee:             transfers[i].FeeAmount,
			TransferType:    transfers[i].Type,
			CryptoToAddress: transfers[i].Destination,
			CryptoTxID:      transfers[i].TxHash,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (g *Gemini) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return g.GetHistoricTrades(ctx, pair, assetType, time.Time{}, time.Time{})
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (g *Gemini) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil && !errors.Is(err, common.ErrDateUnset) {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	ts := timestampStart
	limit := 500
allTrades:
	for {
		var tradeData []Trade
		tradeData, err = g.GetTrades(ctx, p.String(), ts.Unix(), int64(limit), false)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTS := tradeData[i].Timestamp.Time()
			if tradeTS.After(timestampEnd) && !timestampEnd.IsZero() {
				break allTrades
			}

			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Type)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     g.Name,
				TID:          strconv.FormatInt(tradeData[i].TID, 10),
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Timestamp:    tradeTS,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeTS) {
					break allTrades
				}
				ts = tradeTS
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}

	err = g.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (g *Gemini) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(g.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if s.Type != order.Limit {
		return nil, errors.New("only limit orders are enabled through this exchange")
	}

	fPair, err := g.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	response, err := g.NewOrder(ctx,
		fPair.String(),
		s.Amount,
		s.Price,
		s.Side.String(),
		"exchange limit")
	if err != nil {
		return nil, err
	}

	return s.DeriveSubmitResponse(strconv.FormatInt(response, 10))
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (g *Gemini) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gemini) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = g.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (g *Gemini) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (g *Gemini) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gemini) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	resp, err := g.CancelExistingOrders(ctx, false)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp.Details.CancelRejects {
		cancelAllOrdersResponse.Status[resp.Details.CancelRejects[i]] = "Could not cancel order"
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (g *Gemini) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	iOID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	resp, err := g.GetOrderStatus(ctx, iOID)
	if err != nil {
		return nil, err
	}

	cp, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return nil, err
	}

	var orderType order.Type
	switch resp.Type {
	case "exchange limit":
		orderType = order.Limit
	case "market buy", "market sell":
		orderType = order.Market
	default:
		return nil, fmt.Errorf("unknown order type: %q", resp.Type)
	}

	var side order.Side
	side, err = order.StringToOrderSide(resp.Side)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		OrderID:         strconv.FormatInt(resp.OrderID, 10),
		Amount:          resp.OriginalAmount,
		RemainingAmount: resp.RemainingAmount,
		Pair:            cp,
		Date:            resp.TimestampMS.Time(),
		Price:           resp.Price,
		HiddenOrder:     resp.IsHidden,
		ClientOrderID:   resp.ClientOrderID,
		Type:            orderType,
		Side:            side,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gemini) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addr, err := g.GetCryptoDepositAddress(ctx, "", cryptocurrency.String())
	if err != nil {
		return nil, err
	}
	return &deposit.Address{Address: addr.Address}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gemini) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := g.WithdrawCrypto(ctx,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Currency.String(),
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	if resp.Result == "error" {
		return nil, errors.New(resp.Message)
	}

	return &withdraw.ExchangeResponse{
		ID: resp.TXHash,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gemini) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!g.AreCredentialsValid(ctx) || g.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gemini) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := g.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	availPairs, err := g.GetAvailablePairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	format, err := g.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		var symbol currency.Pair
		symbol, err = currency.NewPairFromFormattedPairs(resp[i].Symbol, availPairs, format)
		if err != nil {
			return nil, err
		}

		var orderType order.Type
		switch resp[i].Type {
		case "exchange limit":
			orderType = order.Limit
		case "market buy", "market sell":
			orderType = order.Market
		default:
			return nil, fmt.Errorf("unknown order type: %q", resp[i].Type)
		}

		var side order.Side
		side, err = order.StringToOrderSide(resp[i].Side)
		if err != nil {
			return nil, err
		}

		orders[i] = order.Detail{
			Amount:          resp[i].OriginalAmount,
			RemainingAmount: resp[i].RemainingAmount,
			OrderID:         strconv.FormatInt(resp[i].OrderID, 10),
			ExecutedAmount:  resp[i].ExecutedAmount,
			Exchange:        g.Name,
			Type:            orderType,
			Side:            side,
			Price:           resp[i].Price,
			Pair:            symbol,
			Date:            resp[i].Timestamp.Time(),
		}
	}
	return req.Filter(g.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gemini) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var trades []TradeHistory
	for j := range req.Pairs {
		var fPair currency.Pair
		fPair, err = g.FormatExchangeCurrency(req.Pairs[j], asset.Spot)
		if err != nil {
			return nil, err
		}

		var resp []TradeHistory
		resp, err = g.GetTradeHistory(ctx, fPair.String(), req.StartTime.Unix())
		if err != nil {
			return nil, err
		}

		for i := range resp {
			resp[i].BaseCurrency = req.Pairs[j].Base.String()
			resp[i].QuoteCurrency = req.Pairs[j].Quote.String()
			trades = append(trades, resp[i])
		}
	}

	format, err := g.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(trades))
	for i := range trades {
		var side order.Side
		side, err = order.StringToOrderSide(trades[i].Type)
		if err != nil {
			return nil, err
		}
		detail := order.Detail{
			OrderID:              strconv.FormatInt(trades[i].OrderID, 10),
			Amount:               trades[i].Amount,
			ExecutedAmount:       trades[i].Amount,
			Exchange:             g.Name,
			Date:                 trades[i].Timestamp.Time(),
			Side:                 side,
			Fee:                  trades[i].FeeAmount,
			Price:                trades[i].Price,
			AverageExecutedPrice: trades[i].Price,
			Pair: currency.NewPairWithDelimiter(
				trades[i].BaseCurrency,
				trades[i].QuoteCurrency,
				format.Delimiter,
			),
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(g.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (g *Gemini) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := g.UpdateAccountInfo(ctx, assetType)
	return g.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (g *Gemini) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (g *Gemini) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (g *Gemini) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (g *Gemini) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	details, err := g.GetSymbolDetails(ctx, "all")
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	resp := make([]order.MinMaxLevel, 0, len(details))
	for i := range details {
		status := strings.ToLower(details[i].Status)
		if status != "open" && status != "limit_only" {
			continue
		}
		cp, err := currency.NewPairFromStrings(details[i].BaseCurrency, details[i].QuoteCurrency)
		if err != nil {
			return err
		}
		resp = append(resp, order.MinMaxLevel{
			Pair:                    cp,
			Asset:                   a,
			AmountStepIncrementSize: details[i].TickSize,
			MinimumBaseAmount:       details[i].MinOrderSize.Float64(),
			QuoteStepIncrementSize:  details[i].QuoteIncrement,
		})
	}
	return g.LoadLimits(resp)
}

// GetLatestFundingRates returns the latest funding rates data
func (g *Gemini) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (g *Gemini) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := g.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = ""
	return tradeBaseURL + cp.Upper().String(), nil
}
