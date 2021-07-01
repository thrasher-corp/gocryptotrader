package gemini

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
func (g *Gemini) GetDefaultConfig() (*config.ExchangeConfig, error) {
	g.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = g.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = g.BaseCurrencies

	err := g.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if g.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := g.UpdateTradablePairs(true)
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

	g.Requester = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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
func (g *Gemini) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}

	err := g.SetupDefaults(exch)
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
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       geminiWebsocketEndpoint,
		ExchangeName:                     exch.Name,
		RunningURL:                       wsRunningURL,
		Connector:                        g.WsConnect,
		Subscriber:                       g.Subscribe,
		UnSubscriber:                     g.Unsubscribe,
		GenerateSubscriptions:            g.GenerateDefaultSubscriptions,
		Features:                         &g.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
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
func (g *Gemini) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		g.PrintEnabledPairs()
	}

	forceUpdate := false
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
			log.Warn(log.ExchangeSys,
				"Available pairs for Gemini reset due to config upgrade, please enable the ones you would like to use again")
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

	if !g.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}
	err = g.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			g.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gemini) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := g.GetSymbols()
	if err != nil {
		return nil, err
	}

	var tradablePairs []string
	for x := range pairs {
		switch len(pairs[x]) {
		case 8:
			tradablePairs = append(tradablePairs, pairs[x][0:5]+currency.DashDelimiter+pairs[x][5:])
		case 7:
			tradablePairs = append(tradablePairs, pairs[x][0:4]+currency.DashDelimiter+pairs[x][4:])
		default:
			tradablePairs = append(tradablePairs, pairs[x][0:3]+currency.DashDelimiter+pairs[x][3:])
		}
	}
	return tradablePairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gemini) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := g.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return g.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = g.Name
	accountBalance, err := g.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: currencies,
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (g *Gemini) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(g.Name, assetType)
	if err != nil {
		return g.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tick, err := g.GetTicker(fPair.String())
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
		AssetType:    assetType})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(g.Name, fPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gemini) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateTicker(fPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gemini) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateOrderbook(fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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

	orderbookNew, err := g.GetOrderbook(fPair.String(), url.Values{})
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(g.Name, fPair, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gemini) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (g *Gemini) GetRecentTrades(currencyPair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return g.GetHistoricTrades(currencyPair, assetType, time.Time{}, time.Time{})
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (g *Gemini) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
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
		tradeData, err = g.GetTrades(p.String(), ts.Unix(), int64(limit), false)
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
func (g *Gemini) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	if s.Type != order.Limit {
		return submitOrderResponse,
			errors.New("only limit orders are enabled through this exchange")
	}

	fpair, err := g.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return submitOrderResponse, err
	}

	response, err := g.NewOrder(fpair.String(),
		s.Amount,
		s.Price,
		s.Side.String(),
		"exchange limit")
	if err != nil {
		return submitOrderResponse, err
	}
	if response > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
	}

	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (g *Gemini) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gemini) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	_, err = g.CancelExistingOrder(orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (g *Gemini) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gemini) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	resp, err := g.CancelExistingOrders(false)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp.Details.CancelRejects {
		cancelAllOrdersResponse.Status[resp.Details.CancelRejects[i]] = "Could not cancel order"
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (g *Gemini) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gemini) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addr, err := g.GetCryptoDepositAddress("", cryptocurrency.String())
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gemini) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := g.WithdrawCrypto(withdrawRequest.Crypto.Address, withdrawRequest.Currency.String(), withdrawRequest.Amount)
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
func (g *Gemini) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gemini) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!g.AllowAuthenticatedRequest() || g.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gemini) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := g.GetOrders()
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

	var orders []order.Detail
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

		side := order.Side(strings.ToUpper(resp[i].Type))
		orderDate := time.Unix(resp[i].Timestamp, 0)

		orders = append(orders, order.Detail{
			Amount:          resp[i].OriginalAmount,
			RemainingAmount: resp[i].RemainingAmount,
			ID:              strconv.FormatInt(resp[i].OrderID, 10),
			ExecutedAmount:  resp[i].ExecutedAmount,
			Exchange:        g.Name,
			Type:            orderType,
			Side:            side,
			Price:           resp[i].Price,
			Pair:            symbol,
			Date:            orderDate,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gemini) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var trades []TradeHistory
	for j := range req.Pairs {
		fpair, err := g.FormatExchangeCurrency(req.Pairs[j], asset.Spot)
		if err != nil {
			return nil, err
		}

		resp, err := g.GetTradeHistory(fpair.String(), req.StartTime.Unix())
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

	var orders []order.Detail
	for i := range trades {
		side := order.Side(strings.ToUpper(trades[i].Type))
		orderDate := time.Unix(trades[i].Timestamp, 0)

		orders = append(orders, order.Detail{
			Amount:   trades[i].Amount,
			ID:       strconv.FormatInt(trades[i].OrderID, 10),
			Exchange: g.Name,
			Date:     orderDate,
			Side:     side,
			Fee:      trades[i].FeeAmount,
			Price:    trades[i].Price,
			Pair: currency.NewPairWithDelimiter(trades[i].BaseCurrency,
				trades[i].QuoteCurrency,
				format.Delimiter),
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (g *Gemini) ValidateCredentials(assetType asset.Item) error {
	_, err := g.UpdateAccountInfo(assetType)
	return g.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (g *Gemini) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (g *Gemini) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
