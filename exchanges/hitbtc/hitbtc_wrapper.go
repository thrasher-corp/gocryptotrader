package hitbtc

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (h *HitBTC) GetDefaultConfig() (*config.ExchangeConfig, error) {
	h.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = h.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = h.BaseCurrencies

	err := h.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if h.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = h.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	h.CurrencyPairs = currency.PairsManager{
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

	h.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	h.Requester = request.New(h.Name,
		request.NewRateLimit(time.Second, hitbtcAuthRate),
		request.NewRateLimit(time.Second, hitbtcUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	h.API.Endpoints.URLDefault = apiURL
	h.API.Endpoints.URL = h.API.Endpoints.URLDefault
	h.API.Endpoints.WebsocketURL = hitbtcWebsocketAddress
	h.Websocket = wshandler.New()
	h.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketSubmitOrderSupported |
		wshandler.WebsocketCancelOrderSupported |
		wshandler.WebsocketMessageCorrelationSupported
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}

	err := h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(h.WsConnect,
		h.Subscribe,
		h.Unsubscribe,
		exch.Name,
		exch.Features.Enabled.Websocket,
		exch.Verbose,
		hitbtcWebsocketAddress,
		exch.API.Endpoints.WebsocketURL,
		exch.API.AuthenticatedWebsocketSupport)
	if err != nil {
		return err
	}

	h.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         h.Name,
		URL:                  h.Websocket.GetWebsocketURL(),
		ProxyURL:             h.Websocket.GetProxyAddress(),
		Verbose:              h.Verbose,
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	h.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		true,
		true,
		false,
		exch.Name)
	return nil
}

// Start starts the HitBTC go routine
func (h *HitBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HitBTC wrapper
func (h *HitBTC) Run() {
	if h.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), hitbtcWebsocketAddress)
		h.PrintEnabledPairs()
	}

	forceUpdate := false
	if !common.StringDataContains(h.GetEnabledPairs(asset.Spot).Strings(), "-") ||
		!common.StringDataContains(h.GetAvailablePairs(asset.Spot).Strings(), "-") {
		enabledPairs := []string{"BTC-USD"}
		log.Warn(log.ExchangeSys, "Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")
		forceUpdate = true

		err := h.UpdatePairs(currency.NewPairsFromStrings(enabledPairs), asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update enabled currencies.\n", h.GetName())
		}
	}

	if !h.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := h.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", h.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HitBTC) FetchTradablePairs(asset asset.Item) ([]string, error) {
	symbols, err := h.GetSymbolsDetailed()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range symbols {
		pairs = append(pairs, symbols[x].BaseCurrency+"-"+symbols[x].QuoteCurrency)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HitBTC) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := h.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return h.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(currencyPair currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	if !h.Features.Supports.RESTCapabilities.TickerBatching {
		return tickerPrice, common.ErrFunctionNotSupported
	}

	tick, err := h.GetTicker("")
	if err != nil {
		return tickerPrice, err
	}
	pairs := h.GetEnabledPairs(assetType)
	for i := range pairs {
		for j := range tick {
			if tick[j].Symbol.Equal(pairs[i]) {
				tickerPrice := ticker.Price{
					Last:        tick[j].Last,
					High:        tick[j].High,
					Low:         tick[j].Low,
					Bid:         tick[j].Bid,
					Ask:         tick[j].Ask,
					Volume:      tick[j].Volume,
					QuoteVolume: tick[j].VolumeQuote,
					Open:        tick[j].Open,
					Pair:        pairs[j],
					LastUpdated: tick[j].Timestamp,
				}
				err = ticker.ProcessTicker(h.GetName(), &tickerPrice, assetType)
				if err != nil {
					return tickerPrice, err
				}
			}
		}
	}
	return ticker.GetTicker(h.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HitBTC) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HitBTC) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(currencyPair currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetOrderbook(h.FormatExchangeCurrency(currencyPair, assetType).String(), 1000)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	orderBook.Pair = currencyPair
	orderBook.ExchangeName = h.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(h.Name, currencyPair, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = h.GetName()
	accountBalance, err := h.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for _, item := range accountBalance {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(item.Currency)
		exchangeCurrency.TotalValue = item.Available
		exchangeCurrency.Hold = item.Reserved
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HitBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HitBTC) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	response, err := h.PlaceOrder(order.Pair.String(),
		order.Price,
		order.Amount,
		strings.ToLower(order.OrderType.ToString()),
		strings.ToLower(order.OrderSide.ToString()))
	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.OrderNumber)
	}
	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HitBTC) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HitBTC) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HitBTC) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}

	resp, err := h.CancelAllExistingOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp {
		if resp[i].Status != "canceled" {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(resp[i].ID, 10)] =
				fmt.Sprintf("Could not cancel order %v. Status: %v",
					resp[i].ID,
					resp[i].Status)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HitBTC) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HitBTC) GetDepositAddress(currency currency.Code, _ string) (string, error) {
	resp, err := h.GetDepositAddresses(currency.String())
	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HitBTC) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	_, err := h.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)

	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HitBTC) GetWebsocket() (*wshandler.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !h.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HitBTC) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := h.GetOpenOrders(currency.String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[i].Symbol,
			h.GetPairFormat(asset.Spot, false).Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[i].Side))
		orders = append(orders, exchange.OrderDetail{
			ID:           allOrders[i].ID,
			Amount:       allOrders[i].Quantity,
			Exchange:     h.Name,
			Price:        allOrders[i].Price,
			OrderDate:    allOrders[i].CreatedAt,
			OrderSide:    side,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HitBTC) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := h.GetOrders(currency.String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[i].Symbol,
			h.GetPairFormat(asset.Spot, false).Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[i].Side))
		orders = append(orders, exchange.OrderDetail{
			ID:           allOrders[i].ID,
			Amount:       allOrders[i].Quantity,
			Exchange:     h.Name,
			Price:        allOrders[i].Price,
			OrderDate:    allOrders[i].CreatedAt,
			OrderSide:    side,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (h *HitBTC) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (h *HitBTC) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (h *HitBTC) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return h.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (h *HitBTC) AuthenticateWebsocket() error {
	return h.wsLogin()
}
