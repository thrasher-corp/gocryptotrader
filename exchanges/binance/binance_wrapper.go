package binance

import (
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (b *Binance) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Binance
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.SetValues()

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				TradeFetching:       true,
				UserTradeHistory:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrder:               true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

	b.API.Endpoints.URLDefault = apiURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.Websocket = wshandler.New()
	b.API.Endpoints.WebsocketURL = binanceDefaultWebsocketURL
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       binanceDefaultWebsocketURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        b.WsConnect,
			Features:                         &b.Features.Supports.WebsocketCapabilities,
		})

	if err != nil {
		return err
	}

	b.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         b.Name,
		URL:                  b.Websocket.GetWebsocketURL(),
		ProxyURL:             b.Websocket.GetProxyAddress(),
		Verbose:              b.Verbose,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	b.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		false,
		true,
		true,
		false,
		exch.Name)
	return nil
}

// Start starts the Binance go routine
func (b *Binance) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Binance wrapper
func (b *Binance) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()),
			b.Websocket.GetWebsocketURL())
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := b.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(b.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(b.GetAvailablePairs(asset.Spot).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.BTC.String() + delim + currency.USDT.String()},
		)
		log.Warn(log.ExchangeSys,
			"Available pairs for Binance reset due to config upgrade, please enable the ones you would like to use again")
		forceUpdate = true

		err := b.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				b.Name,
				err)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Binance) FetchTradablePairs(asset asset.Item) ([]string, error) {
	var validCurrencyPairs []string

	info, err := b.GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	for x := range info.Symbols {
		if info.Symbols[x].Status == "TRADING" {
			validCurrencyPairs = append(validCurrencyPairs, info.Symbols[x].BaseAsset+
				b.GetPairFormat(asset, false).Delimiter+
				info.Symbols[x].QuoteAsset)
		}
	}
	return validCurrencyPairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot,
		false,
		forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := b.GetTickers()
	if err != nil {
		return nil, err
	}
	pairs := b.GetEnabledPairs(assetType)
	for i := range pairs {
		for y := range tick {
			pairFmt := b.FormatExchangeCurrency(pairs[i], assetType).String()
			if tick[y].Symbol != pairFmt {
				continue
			}
			tickerPrice := &ticker.Price{
				Last:        tick[y].LastPrice,
				High:        tick[y].HighPrice,
				Low:         tick[y].LowPrice,
				Bid:         tick[y].BidPrice,
				Ask:         tick[y].AskPrice,
				Volume:      tick[y].Volume,
				QuoteVolume: tick[y].QuoteVolume,
				Open:        tick[y].OpenPrice,
				Close:       tick[y].PrevClosePrice,
				Pair:        pairs[i],
			}
			err = ticker.ProcessTicker(b.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Binance) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := b.GetOrderBook(OrderBookDataRequestParams{Symbol: b.FormatExchangeCurrency(p,
		assetType).String(), Limit: 1000})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Bids[x].Quantity,
				Price:  orderbookNew.Bids[x].Price,
			})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Asks[x].Quantity,
				Price:  orderbookNew.Asks[x].Price,
			})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = b.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Binance) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	raw, err := b.GetAccount()
	if err != nil {
		return info, err
	}

	var currencyBalance []account.Balance
	for i := range raw.Balances {
		freeCurrency, parseErr := strconv.ParseFloat(raw.Balances[i].Free, 64)
		if parseErr != nil {
			return info, parseErr
		}

		lockedCurrency, parseErr := strconv.ParseFloat(raw.Balances[i].Locked, 64)
		if parseErr != nil {
			return info, parseErr
		}

		currencyBalance = append(currencyBalance, account.Balance{
			CurrencyName: currency.NewCode(raw.Balances[i].Asset),
			TotalValue:   freeCurrency + lockedCurrency,
			Hold:         freeCurrency,
		})
	}

	info.Exchange = b.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: currencyBalance,
	})

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Binance) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Binance) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var sideType string
	if s.Side == order.Buy {
		sideType = order.Buy.String()
	} else {
		sideType = order.Sell.String()
	}

	var requestParamsOrderType RequestParamsOrderType
	switch s.Type {
	case order.Market:
		requestParamsOrderType = BinanceRequestParamsOrderMarket
	case order.Limit:
		requestParamsOrderType = BinanceRequestParamsOrderLimit
	default:
		submitOrderResponse.IsOrderPlaced = false
		return submitOrderResponse, errors.New("unsupported order type")
	}

	var orderRequest = NewOrderRequest{
		Symbol:      s.Pair.Base.String() + s.Pair.Quote.String(),
		Side:        sideType,
		Price:       s.Price,
		Quantity:    s.Amount,
		TradeType:   requestParamsOrderType,
		TimeInForce: BinanceRequestParamsTimeGTC,
	}

	response, err := b.NewOrder(&orderRequest)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.OrderID > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
	}
	if response.ExecutedQty == response.OrigQty {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(b.FormatExchangeCurrency(order.Pair,
		order.AssetType).String(),
		orderIDInt,
		order.AccountID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := b.OpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range openOrders {
		_, err = b.CancelExistingOrder(openOrders[i].Symbol,
			openOrders[i].OrderID,
			"")
		if err != nil {
			cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Binance) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetDepositAddressForCurrency(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	v, err := b.WithdrawCrypto(withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description, amountStr)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Binance) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Binance) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!b.AllowAuthenticatedRequest() || b.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Binance) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []order.Detail
	for x := range req.Pairs {
		resp, err := b.OpenOrders(b.FormatExchangeCurrency(req.Pairs[x],
			asset.Spot).String())
		if err != nil {
			return nil, err
		}

		for i := range resp {
			orderSide := order.Side(strings.ToUpper(resp[i].Side))
			orderType := order.Type(strings.ToUpper(resp[i].Type))
			orderDate := time.Unix(0, int64(resp[i].Time)*int64(time.Millisecond))

			orders = append(orders, order.Detail{
				Amount:   resp[i].OrigQty,
				Date:     orderDate,
				Exchange: b.Name,
				ID:       strconv.FormatInt(resp[i].OrderID, 10),
				Side:     orderSide,
				Type:     orderType,
				Price:    resp[i].Price,
				Status:   order.Status(resp[i].Status),
				Pair:     currency.NewPairFromString(resp[i].Symbol),
			})
		}
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []order.Detail
	for x := range req.Pairs {
		resp, err := b.AllOrders(b.FormatExchangeCurrency(req.Pairs[x],
			asset.Spot).String(),
			"",
			"1000")
		if err != nil {
			return nil, err
		}

		for i := range resp {
			orderSide := order.Side(strings.ToUpper(resp[i].Side))
			orderType := order.Type(strings.ToUpper(resp[i].Type))
			orderDate := time.Unix(0, int64(resp[i].Time)*int64(time.Millisecond))
			// New orders are covered in GetOpenOrders
			if resp[i].Status == "NEW" {
				continue
			}

			orders = append(orders, order.Detail{
				Amount:   resp[i].OrigQty,
				Date:     orderDate,
				Exchange: b.Name,
				ID:       strconv.FormatInt(resp[i].OrderID, 10),
				Side:     orderSide,
				Type:     orderType,
				Price:    resp[i].Price,
				Pair:     currency.NewPairFromString(resp[i].Symbol),
				Status:   order.Status(resp[i].Status),
			})
		}
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Binance) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Binance) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Binance) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Binance) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Binance) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Binance) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	intervalToString, err := parseInterval(interval)
	if err != nil {
		return kline.Item{}, err
	}
	klineParams := KlinesRequestParams{
		Interval:  intervalToString,
		Symbol:    b.FormatExchangeCurrency(pair, a).String(),
		StartTime: start.Unix() * 1000,
		EndTime:   end.Unix() * 1000,
	}

	candles, err := b.GetSpotKline(klineParams)
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].Close,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}
	return ret, nil
}
