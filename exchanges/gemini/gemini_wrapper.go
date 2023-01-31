package gemini

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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

// GetDefaultConfig returns a default exchange config
func (g *Gemini) GetDefaultConfig() (*config.Exchange, error) {
	g.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = g.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = g.BaseCurrencies

	err := g.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if g.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := g.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

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
	}

	g.Requester, err = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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
	g.Websocket = stream.New()
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
		err = g.API.Endpoints.SetRunning(exchange.RestSpot.String(), geminiSandboxAPIURL)
		if err != nil {
			log.Error(log.ExchangeSys, err)
		}
	}

	wsRunningURL, err := g.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = g.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             geminiWebsocketEndpoint,
		RunningURL:             wsRunningURL,
		Connector:              g.WsConnect,
		Subscriber:             g.Subscribe,
		Unsubscriber:           g.Unsubscribe,
		GenerateSubscriptions:  g.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &g.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  geminiWebsocketEndpoint + "/v2/" + geminiWsMarketData,
	})
	if err != nil {
		return err
	}

	return g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  geminiWebsocketEndpoint + "/v1/" + geminiWsOrderEvents,
		Authenticated:        true,
	})
}

// Start starts the Gemini go routine
func (g *Gemini) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		g.PrintEnabledPairs()
	}

	forceUpdate := false
	if !g.BypassConfigFormatUpgrades {
		format, err := g.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
				g.Name,
				err)
			return
		}

		enabled, err := g.CurrencyPairs.GetPairs(asset.Spot, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
				g.Name,
				err)
			return
		}

		avail, err := g.CurrencyPairs.GetPairs(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get available currencies. Err %s\n",
				g.Name,
				err)
			return
		}

		if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
			!common.StringDataContains(avail.Strings(), format.Delimiter) {
			var enabledPairs currency.Pairs
			enabledPairs, err = currency.NewPairsFromStrings([]string{
				currency.BTC.String() + format.Delimiter + currency.USD.String()})
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err %s\n",
					g.Name,
					err)
			} else {
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, g.Name, asset.Spot, enabledPairs)
				forceUpdate = true

				err = g.UpdatePairs(enabledPairs, asset.Spot, true, true)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies. Err: %s\n",
						g.Name,
						err)
				}
			}
		}
	}

	if !g.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}
	err := g.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			g.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gemini) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !g.SupportsAsset(a) {
		return nil, asset.ErrNotSupported
	}

	symbols, err := g.GetSymbols(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(symbols))
	for x := range symbols {
		var pair currency.Pair
		switch len(symbols[x]) {
		case 8:
			pair, err = currency.NewPairFromStrings(symbols[x][0:5], symbols[x][5:])
		case 7:
			pair, err = currency.NewPairFromStrings(symbols[x][0:4], symbols[x][4:])
		default:
			pair, err = currency.NewPairFromStrings(symbols[x][0:3], symbols[x][3:])
		}
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
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
	return g.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
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

// FetchAccountInfo retrieves balances for all enabled currencies
func (g *Gemini) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(g.Name, creds, assetType)
	if err != nil {
		return g.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (g *Gemini) UpdateTickers(ctx context.Context, a asset.Item) error {
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
		AssetType:    a})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(g.Name, fPair, a)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gemini) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateTicker(ctx, fPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gemini) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        g.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := g.GetOrderbook(ctx, fPair.String(), url.Values{})
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
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

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gemini) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
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
		tradeData, err = g.GetTrades(ctx,
			p.String(),
			ts.Unix(),
			int64(limit),
			false)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTS := time.Unix(tradeData[i].Timestamp, 0)
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
				if ts == tradeTS {
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
	if err := s.Validate(); err != nil {
		return nil, err
	}

	if s.Type != order.Limit {
		return nil, errors.New("only limit orders are enabled through this exchange")
	}

	fpair, err := g.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	response, err := g.NewOrder(ctx,
		fpair.String(),
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
func (g *Gemini) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
func (g *Gemini) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
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
func (g *Gemini) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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
		if resp[i].Type == "exchange limit" {
			orderType = order.Limit
		} else if resp[i].Type == "market buy" || resp[i].Type == "market sell" {
			orderType = order.Market
		}
		var side order.Side
		side, err = order.StringToOrderSide(resp[i].Type)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(resp[i].Timestamp, 0)

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
			Date:            orderDate,
		}
	}
	return req.Filter(g.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gemini) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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
		orderDate := time.Unix(trades[i].Timestamp, 0)

		detail := order.Detail{
			OrderID:              strconv.FormatInt(trades[i].OrderID, 10),
			Amount:               trades[i].Amount,
			ExecutedAmount:       trades[i].Amount,
			Exchange:             g.Name,
			Date:                 orderDate,
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

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (g *Gemini) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
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
