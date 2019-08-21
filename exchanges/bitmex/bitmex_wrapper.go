package bitmex

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

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
func (b *Bitmex) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for Bitmex
func (b *Bitmex) SetDefaults() {
	b.Name = "Bitmex"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.PerpetualContract,
			asset.Futures,
			asset.DownsideProfitContract,
			asset.UpsideProfitContract,
		},
		UseGlobalFormat: false,
	}

	// Same format used for perpetual contracts and futures
	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}
	b.CurrencyPairs.Store(asset.PerpetualContract, fmt1)
	b.CurrencyPairs.Store(asset.Futures, fmt1)

	// Upside and Downside profit contracts use the same format
	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}
	b.CurrencyPairs.Store(asset.DownsideProfitContract, fmt2)
	b.CurrencyPairs.Store(asset.UpsideProfitContract, fmt2)

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawCryptoWithEmail |
				exchange.WithdrawCryptoWith2FA |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, bitmexAuthRate),
		request.NewRateLimit(time.Second, bitmexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = bitmexAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.API.Endpoints.WebsocketURL = bitmexWSURL
	b.Websocket = wshandler.New()
	b.Websocket.Functionality = wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketAccountDataSupported |
		wshandler.WebsocketDeadMansSwitchSupported
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitmex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(b.WsConnector,
		b.Subscribe,
		b.Unsubscribe,
		exch.Name,
		exch.Features.Enabled.Websocket,
		exch.Verbose,
		bitmexWSURL,
		exch.API.Endpoints.WebsocketURL,
		exch.API.AuthenticatedWebsocketSupport)
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
		true,
		false,
		false,
		true,
		exch.Name)
	return nil
}

// Start starts the Bitmex go routine
func (b *Bitmex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitmex wrapper
func (b *Bitmex) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.API.Endpoints.WebsocketURL)
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitmex) FetchTradablePairs(asset asset.Item) ([]currency.Pair, error) {
	marketInfo, err := b.GetActiveInstruments(&GenericRequestParams{})
	if err != nil {
		return nil, err
	}

	var products []currency.Pair
	for x := range marketInfo {
		products = append(products, marketInfo[x].Symbol)
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitmex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	var assetPairs []string
	for x := range b.CurrencyPairs.AssetTypes {
		switch b.CurrencyPairs.AssetTypes[x] {
		case asset.PerpetualContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "USD") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case asset.Futures:
			for y := range pairs {
				if strings.Contains(pairs[y], "19") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case asset.DownsideProfitContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "_D") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case asset.UpsideProfitContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "_U") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		}

		err = b.UpdatePairs(currency.NewPairsFromStrings(assetPairs), b.CurrencyPairs.AssetTypes[x], false, false)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s failed to update available pairs. Err: %v", b.Name, err)
		}
		assetPairs = nil
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitmex) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	curr := b.FormatExchangeCurrency(p, assetType)

	tick, err := b.GetActiveInstruments(&GenericRequestParams{
		Symbol:  curr.String(),
		Reverse: true,
		Count:   1})
	if err != nil {
		return tickerPrice, err
	}

	if len(tick) == 0 {
		return tickerPrice, fmt.Errorf("%s REST error: no ticker return", b.Name)
	}

	tickerPrice = ticker.Price{
		Last:        tick[0].LastPrice,
		High:        tick[0].HighPrice,
		Low:         tick[0].LowPrice,
		Bid:         tick[0].BidPrice,
		Ask:         tick[0].AskPrice,
		Volume:      tick[0].Volume24h,
		Close:       tick[0].PrevClosePrice,
		Pair:        tick[0].Symbol,
		LastUpdated: tick[0].Timestamp,
	}
	return tickerPrice, ticker.ProcessTicker(b.Name, &tickerPrice, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitmex) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Bitmex) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitmex) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base

	orderbookNew, err := b.GetOrderbook(OrderBookGetL2Params{
		Symbol: b.FormatExchangeCurrency(p, assetType).String(),
		Depth:  500})
	if err != nil {
		return orderBook, err
	}

	for _, ob := range orderbookNew {
		if strings.EqualFold(ob.Side, exchange.SellOrderSide.ToString()) {
			orderBook.Asks = append(orderBook.Asks,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
		if strings.EqualFold(ob.Side, exchange.BuyOrderSide.ToString()) {
			orderBook.Bids = append(orderBook.Bids,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
	}

	orderBook.Pair = p
	orderBook.ExchangeName = b.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Bitmex exchange
func (b *Bitmex) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	bal, err := b.GetAllUserMargin()
	if err != nil {
		return info, err
	}

	// Need to update to add Margin/Liquidity availibilty
	var balances []exchange.AccountCurrencyInfo
	for i := range bal {
		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(bal[i].Currency),
			TotalValue:   float64(bal[i].WalletBalance),
		})
	}

	info.Exchange = b.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitmex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitmex) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitmex) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	if math.Mod(order.Amount, 1) != 0 {
		return submitOrderResponse,
			errors.New("order contract amount can not have decimals")
	}

	var orderNewParams = OrderNewParams{
		OrdType:  order.OrderSide.ToString(),
		Symbol:   order.Pair.String(),
		OrderQty: order.Amount,
		Side:     order.OrderSide.ToString(),
	}

	if order.OrderType == exchange.LimitOrderType {
		orderNewParams.Price = order.Price
	}

	response, err := b.CreateOrder(&orderNewParams)
	if response.OrderID != "" {
		submitOrderResponse.OrderID = response.OrderID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitmex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	var params OrderAmendParams

	if math.Mod(action.Amount, 1) != 0 {
		return "", errors.New("contract amount can not have decimals")
	}

	params.OrderID = action.OrderID
	params.OrderQty = int32(action.Amount)
	params.Price = action.Price

	order, err := b.AmendOrder(&params)
	if err != nil {
		return "", err
	}

	return order.OrderID, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitmex) CancelOrder(order *exchange.OrderCancellation) error {
	var params = OrderCancelParams{
		OrderID: order.OrderID,
	}
	_, err := b.CancelOrders(&params)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitmex) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var emptyParams OrderCancelAllParams
	orders, err := b.CancelAllExistingOrders(emptyParams)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range orders {
		if orders[i].OrdRejReason != "" {
			cancelAllOrdersResponse.OrderStatus[orders[i].OrderID] = orders[i].OrdRejReason
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bitmex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitmex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetCryptoDepositAddress(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	var request = UserRequestWithdrawalParams{
		Address:  withdrawRequest.Address,
		Amount:   withdrawRequest.Amount,
		Currency: withdrawRequest.Currency.String(),
		OtpToken: withdrawRequest.OneTimePassword,
	}
	if withdrawRequest.FeeAmount > 0 {
		request.Fee = withdrawRequest.FeeAmount
	}

	resp, err := b.UserRequestWithdrawal(request)
	if err != nil {
		return "", err
	}

	return resp.TransactID, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitmex) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitmex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	params := OrdersRequest{}
	params.Filter = "{\"open\":true}"

	resp, err := b.GetOrders(&params)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		orderType := orderTypeMap[resp[i].OrdType]
		if orderType == "" {
			orderType = exchange.UnknownOrderType
		}

		orderDetail := exchange.OrderDetail{
			Price:     resp[i].Price,
			Amount:    float64(resp[i].OrderQty),
			Exchange:  b.Name,
			ID:        resp[i].OrderID,
			OrderSide: orderSide,
			OrderType: orderType,
			Status:    resp[i].OrdStatus,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				b.GetPairFormat(asset.PerpetualContract, false).Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	params := OrdersRequest{}
	resp, err := b.GetOrders(&params)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		orderType := orderTypeMap[resp[i].OrdType]
		if orderType == "" {
			orderType = exchange.UnknownOrderType
		}

		orderDetail := exchange.OrderDetail{
			Price:     resp[i].Price,
			Amount:    float64(resp[i].OrderQty),
			Exchange:  b.Name,
			ID:        resp[i].OrderID,
			OrderSide: orderSide,
			OrderType: orderType,
			Status:    resp[i].OrdStatus,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				b.GetPairFormat(asset.PerpetualContract, false).Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Bitmex) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Bitmex) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Bitmex) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitmex) AuthenticateWebsocket() error {
	return b.websocketSendAuth()
}
