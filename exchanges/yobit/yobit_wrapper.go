package yobit

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (y *Yobit) GetDefaultConfig() (*config.ExchangeConfig, error) {
	y.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = y.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = y.BaseCurrencies

	err := y.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if y.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = y.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default value for Yobit
func (y *Yobit) SetDefaults() {
	y.Name = "Yobit"
	y.Enabled = true
	y.Verbose = true
	y.API.CredentialsValidator.RequiresKey = true
	y.API.CredentialsValidator.RequiresSecret = true

	y.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: false,
			Separator: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	y.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
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
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	y.Requester = request.New(y.Name,
		request.NewRateLimit(time.Second, yobitAuthRate),
		request.NewRateLimit(time.Second, yobitUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	y.API.Endpoints.URLDefault = apiPublicURL
	y.API.Endpoints.URL = y.API.Endpoints.URLDefault
	y.API.Endpoints.URLSecondaryDefault = apiPrivateURL
	y.API.Endpoints.URLSecondary = y.API.Endpoints.URLSecondaryDefault
}

// Setup sets exchange configuration parameters for Yobit
func (y *Yobit) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		y.SetEnabled(false)
		return nil
	}

	return y.SetupDefaults(exch)
}

// Start starts the WEX go routine
func (y *Yobit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		y.Run()
		wg.Done()
	}()
}

// Run implements the Yobit wrapper
func (y *Yobit) Run() {
	if y.Verbose {
		y.PrintEnabledPairs()
	}

	if !y.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := y.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", y.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (y *Yobit) FetchTradablePairs(asset asset.Item) ([]string, error) {
	info, err := y.GetInfo()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range info.Pairs {
		currencies = append(currencies, strings.ToUpper(x))
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (y *Yobit) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := y.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return y.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (y *Yobit) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairsCollated, err := y.FormatExchangeCurrencies(y.GetEnabledPairs(assetType), assetType)
	if err != nil {
		return tickerPrice, err
	}

	result, err := y.GetTicker(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range y.GetEnabledPairs(assetType) {
		curr := y.FormatExchangeCurrency(x, assetType).Lower().String()
		if _, ok := result[curr]; !ok {
			continue
		}
		var tickerPrice ticker.Price
		tickerPrice.Pair = x
		tickerPrice.Last = result[curr].Last
		tickerPrice.Ask = result[curr].Sell
		tickerPrice.Bid = result[curr].Buy
		tickerPrice.Last = result[curr].Last
		tickerPrice.Low = result[curr].Low
		tickerPrice.QuoteVolume = result[curr].VolumeCurrent
		tickerPrice.Volume = result[curr].Vol

		err = ticker.ProcessTicker(y.Name, &tickerPrice, assetType)
		if err != nil {
			log.Error(log.Ticker, err)
		}
	}
	return ticker.GetTicker(y.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (y *Yobit) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tick, err := ticker.GetTicker(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (y *Yobit) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(y.GetName(), p, assetType)
	if err != nil {
		return y.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (y *Yobit) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := y.GetDepth(y.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: data[0], Amount: data[1]})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = y.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(y.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Yobit exchange
func (y *Yobit) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = y.GetName()
	accountBalance, err := y.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for x, y := range accountBalance.FundsInclOrders {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		for z, w := range accountBalance.Funds {
			if z == x {
				exchangeCurrency.Hold = y - w
			}
		}

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (y *Yobit) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (y *Yobit) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// Yobit only supports limit orders
func (y *Yobit) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	if order.OrderType != exchange.LimitOrderType {
		return submitOrderResponse, errors.New("only limit orders are allowed")
	}

	response, err := y.Trade(order.Pair.String(), order.OrderSide.ToString(),
		order.Amount, order.Price)
	if response > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}
	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (y *Yobit) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (y *Yobit) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = y.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (y *Yobit) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allActiveOrders []map[string]ActiveOrders

	for _, pair := range y.GetEnabledPairs(asset.Spot) {
		activeOrdersForPair, err := y.GetOpenOrders(y.FormatExchangeCurrency(pair,
			asset.Spot).String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		allActiveOrders = append(allActiveOrders, activeOrdersForPair)
	}

	for _, activeOrders := range allActiveOrders {
		for key := range activeOrders {
			orderIDInt, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[key] = err.Error()
				continue
			}

			_, err = y.CancelExistingOrder(orderIDInt)
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[key] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (y *Yobit) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (y *Yobit) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	a, err := y.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return a.Return.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (y *Yobit) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	resp, err := y.WithdrawCoinsToAddress(withdrawRequest.Currency.String(), withdrawRequest.Amount, withdrawRequest.Address)
	if err != nil {
		return "", err
	}
	if len(resp.Error) > 0 {
		return "", errors.New(resp.Error)
	}
	return "success", nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (y *Yobit) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (y *Yobit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !y.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return y.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (y *Yobit) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	for _, c := range getOrdersRequest.Currencies {
		resp, err := y.GetOpenOrders(y.FormatExchangeCurrency(c,
			asset.Spot).String())
		if err != nil {
			return nil, err
		}

		for ID, order := range resp {
			symbol := currency.NewPairDelimiter(order.Pair,
				y.GetPairFormat(asset.Spot, false).Delimiter)
			orderDate := time.Unix(int64(order.TimestampCreated), 0)
			side := exchange.OrderSide(strings.ToUpper(order.Type))
			orders = append(orders, exchange.OrderDetail{
				ID:           ID,
				Amount:       order.Amount,
				Price:        order.Rate,
				OrderSide:    side,
				OrderDate:    orderDate,
				CurrencyPair: symbol,
				Exchange:     y.Name,
			})
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (y *Yobit) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var allOrders []TradeHistory
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := y.GetTradeHistory(0,
			10000,
			math.MaxInt64,
			getOrdersRequest.StartTicks.Unix(),
			getOrdersRequest.EndTicks.Unix(),
			"DESC",
			y.FormatExchangeCurrency(currency, asset.Spot).String())
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			allOrders = append(allOrders, order)
		}
	}

	var orders []exchange.OrderDetail
	for _, order := range allOrders {
		symbol := currency.NewPairDelimiter(order.Pair,
			y.GetPairFormat(asset.Spot, false).Delimiter)
		orderDate := time.Unix(int64(order.Timestamp), 0)
		side := exchange.OrderSide(strings.ToUpper(order.Type))
		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.OrderID),
			Amount:       order.Amount,
			Price:        order.Rate,
			OrderSide:    side,
			OrderDate:    orderDate,
			CurrencyPair: symbol,
			Exchange:     y.Name,
		})
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (y *Yobit) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (y *Yobit) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (y *Yobit) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (y *Yobit) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
