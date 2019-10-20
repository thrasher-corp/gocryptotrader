package lakebtc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (l *LakeBTC) GetDefaultConfig() (*config.ExchangeConfig, error) {
	l.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = l.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = l.BaseCurrencies

	err := l.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if l.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = l.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets LakeBTC defaults
func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = true
	l.Verbose = true
	l.API.CredentialsValidator.RequiresKey = true
	l.API.CredentialsValidator.RequiresSecret = true

	l.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:    true,
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				UserTradeHistory:  true,
				CryptoWithdrawal:  true,
				TradeFee:          true,
				CryptoDepositFee:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:     true,
				OrderbookFetching: true,
				Subscribe:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	l.Requester = request.New(l.Name,
		request.NewRateLimit(time.Second, lakeBTCAuthRate),
		request.NewRateLimit(time.Second, lakeBTCUnauth),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	l.API.Endpoints.URLDefault = lakeBTCAPIURL
	l.API.Endpoints.URL = l.API.Endpoints.URLDefault
	l.Websocket = wshandler.New()
	l.API.Endpoints.WebsocketURL = lakeBTCWSURL
	l.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	l.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	l.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets exchange configuration profile
func (l *LakeBTC) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}

	err := l.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = l.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       lakeBTCWSURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        l.WsConnect,
			Subscriber:                       l.Subscribe,
			Features:                         &l.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	l.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		false,
		false,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the LakeBTC go routine
func (l *LakeBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LakeBTC wrapper
func (l *LakeBTC) Run() {
	if l.Verbose {
		l.PrintEnabledPairs()
	}

	if !l.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := l.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", l.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *LakeBTC) FetchTradablePairs(asset asset.Item) ([]string, error) {
	result, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result {
		currencies = append(currencies, strings.ToUpper(x))
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *LakeBTC) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return l.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LakeBTC) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	ticks, err := l.GetTicker()
	if err != nil {
		return ticker.Price{}, err
	}

	pairs := l.GetEnabledPairs(assetType)
	for i := range pairs {
		currency := l.FormatExchangeCurrency(pairs[i], assetType).String()
		if _, ok := ticks[currency]; !ok {
			continue
		}
		var tickerPrice ticker.Price
		tickerPrice.Pair = pairs[i]
		tickerPrice.Ask = ticks[currency].Ask
		tickerPrice.Bid = ticks[currency].Bid
		tickerPrice.Volume = ticks[currency].Volume
		tickerPrice.High = ticks[currency].High
		tickerPrice.Low = ticks[currency].Low
		tickerPrice.Last = ticks[currency].Last

		err = ticker.ProcessTicker(l.GetName(), &tickerPrice, assetType)
		if err != nil {
			log.Error(log.Ticker, err)
		}
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (l *LakeBTC) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (l *LakeBTC) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LakeBTC) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderBook(p.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = l.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// LakeBTC exchange
func (l *LakeBTC) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = l.GetName()
	accountInfo, err := l.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z != x {
				continue
			}
			var exchangeCurrency exchange.AccountCurrencyInfo
			exchangeCurrency.CurrencyName = currency.NewCode(x)
			exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
			exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
			currencies = append(currencies, exchangeCurrency)
		}
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LakeBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LakeBTC) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LakeBTC) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	isBuyOrder := order.OrderSide == exchange.BuyOrderSide
	response, err := l.Trade(isBuyOrder, order.Amount, order.Price,
		order.Pair.Lower().String())
	if response.ID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.ID)
	}
	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LakeBTC) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LakeBTC) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	return l.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LakeBTC) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var cancelAllOrdersResponse exchange.CancelAllOrdersResponse
	openOrders, err := l.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var ordersToCancel []string
	for _, order := range openOrders {
		ordersToCancel = append(ordersToCancel, strconv.FormatInt(order.ID, 10))
	}

	return cancelAllOrdersResponse, l.CancelExistingOrders(ordersToCancel)

}

// GetOrderInfo returns information on a current open order
func (l *LakeBTC) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LakeBTC) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	if !strings.EqualFold(cryptocurrency.String(), currency.BTC.String()) {
		return "", fmt.Errorf("unsupported currency %s deposit address can only be BTC, manual deposit is required for other currencies",
			cryptocurrency.String())
	}

	info, err := l.GetAccountInformation()
	if err != nil {
		return "", err
	}

	return info.Profile.BTCDepositAddress, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LakeBTC) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	if withdrawRequest.Currency != currency.BTC {
		return "", errors.New("only BTC supported for withdrawals")
	}

	resp, err := l.CreateWithdraw(withdrawRequest.Amount, withdrawRequest.Description)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", resp.ID), nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LakeBTC) GetWebsocket() (*wshandler.Websocket, error) {
	return l.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LakeBTC) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !l.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return l.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LakeBTC) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		symbol := currency.NewPairDelimiter(order.Symbol,
			l.GetPairFormat(asset.Spot, false).Delimiter)
		orderDate := time.Unix(order.At, 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:       order.Amount,
			ID:           fmt.Sprintf("%v", order.ID),
			Price:        order.Price,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LakeBTC) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetOrders([]int64{})
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		if order.State == "active" {
			continue
		}

		symbol := currency.NewPairDelimiter(order.Symbol,
			l.GetPairFormat(asset.Spot, false).Delimiter)
		orderDate := time.Unix(order.At, 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:       order.Amount,
			ID:           fmt.Sprintf("%v", order.ID),
			Price:        order.Price,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (l *LakeBTC) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (l *LakeBTC) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (l *LakeBTC) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (l *LakeBTC) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
