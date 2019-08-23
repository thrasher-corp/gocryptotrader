package huobi

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
func (h *HUOBI) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets default values for the exchange
func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
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
			Uppercase: false,
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
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	h.Requester = request.New(h.Name,
		request.NewRateLimit(time.Second*10, huobiAuthRate),
		request.NewRateLimit(time.Second*10, huobiUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	h.API.Endpoints.URLDefault = huobiAPIURL
	h.API.Endpoints.URL = h.API.Endpoints.URLDefault
	h.API.Endpoints.WebsocketURL = wsMarketURL
	h.Websocket = wshandler.New()
	h.Websocket.Functionality = wshandler.WebsocketKlineSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported |
		wshandler.WebsocketAccountDataSupported |
		wshandler.WebsocketMessageCorrelationSupported
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}

	err := h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	h.API.PEMKeySupport = exch.API.PEMKeySupport
	h.API.Credentials.PEMKey = exch.API.Credentials.PEMKey

	err = h.Websocket.Setup(h.WsConnect,
		h.Subscribe,
		h.Unsubscribe,
		exch.Name,
		exch.Features.Enabled.Websocket,
		exch.Verbose,
		wsMarketURL,
		exch.API.Endpoints.WebsocketURL,
		exch.API.AuthenticatedWebsocketSupport)
	if err != nil {
		return err
	}

	h.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         h.Name,
		URL:                  wsMarketURL,
		ProxyURL:             h.Websocket.GetProxyAddress(),
		Verbose:              h.Verbose,
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}
	h.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         h.Name,
		URL:                  wsAccountsOrdersURL,
		ProxyURL:             h.Websocket.GetProxyAddress(),
		Verbose:              h.Verbose,
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	h.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		false,
		false,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the HUOBI go routine
func (h *HUOBI) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HUOBI wrapper
func (h *HUOBI) Run() {
	if h.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), wsMarketURL)
		h.PrintEnabledPairs()
	}

	var forceUpdate bool
	if common.StringDataContains(h.GetEnabledPairs(asset.Spot).Strings(), "CNY") ||
		common.StringDataContains(h.GetAvailablePairs(asset.Spot).Strings(), "CNY") {
		forceUpdate = true
	}

	if common.StringDataContains(h.BaseCurrencies.Strings(), "CNY") {
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(h.Name)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get exchange config. %s\n", h.Name, err)
			return
		}
		exchCfg.BaseCurrencies = currency.Currencies{currency.USD}
		h.BaseCurrencies = currency.Currencies{currency.USD}
	}

	if forceUpdate {
		enabledPairs := currency.Pairs{currency.Pair{
			Base:      currency.BTC.Lower(),
			Quote:     currency.USDT.Lower(),
			Delimiter: "-",
		},
		}
		log.Warn(log.ExchangeSys, "Available and enabled pairs for Huobi reset due to config upgrade, please enable the ones you would like again")

		err := h.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s Failed to update enabled currencies.\n", h.GetName())
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
func (h *HUOBI) FetchTradablePairs(asset asset.Item) ([]string, error) {
	symbols, err := h.GetSymbols()
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
func (h *HUOBI) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := h.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return h.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBI) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tickers, err := h.GetTickers()
	if err != nil {
		return tickerPrice, err
	}
	pairs := h.GetEnabledPairs(assetType)
	for i := range pairs {
		for j := range tickers.Data {
			if !pairs[i].Equal(tickers.Data[j].Symbol) {
				continue
			}
			tickerPrice := ticker.Price{
				High:   tickers.Data[j].High,
				Low:    tickers.Data[j].Low,
				Volume: tickers.Data[j].Vol,
				Open:   tickers.Data[j].Open,
				Close:  tickers.Data[j].Close,
				Pair:   tickers.Data[j].Symbol,
			}
			err = ticker.ProcessTicker(h.GetName(), &tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}

	return ticker.GetTicker(h.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HUOBI) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HUOBI) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBI) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := h.GetDepth(OrderBookDataRequestParams{
		Symbol: h.FormatExchangeCurrency(p, assetType).String(),
		Type:   OrderBookDataRequestParamsTypeStep1,
	})
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = h.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(h.Name, p, assetType)
}

// GetAccountID returns the account ID for trades
func (h *HUOBI) GetAccountID() ([]Account, error) {
	acc, err := h.GetAccounts()
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errors.New("no account returned")
	}

	return acc, nil
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// HUOBI exchange - to-do
func (h *HUOBI) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	info.Exchange = h.GetName()

	accounts, err := h.GetAccountID()
	if err != nil {
		return info, err
	}

	for _, account := range accounts {
		var acc exchange.Account

		acc.ID = strconv.FormatInt(account.ID, 10)

		balances, err := h.GetAccountBalance(acc.ID)
		if err != nil {
			return info, err
		}

		var currencyDetails []exchange.AccountCurrencyInfo
		for _, balance := range balances {
			var frozen bool
			if balance.Type == "frozen" {
				frozen = true
			}

			var updated bool
			for i := range currencyDetails {
				if currencyDetails[i].CurrencyName == currency.NewCode(balance.Currency) {
					if frozen {
						currencyDetails[i].Hold = balance.Balance
					} else {
						currencyDetails[i].TotalValue = balance.Balance
					}
					updated = true
				}
			}

			if updated {
				continue
			}

			if frozen {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: currency.NewCode(balance.Currency),
						Hold:         balance.Balance,
					})
			} else {
				currencyDetails = append(currencyDetails,
					exchange.AccountCurrencyInfo{
						CurrencyName: currency.NewCode(balance.Currency),
						TotalValue:   balance.Balance,
					})
			}
		}

		acc.Currencies = currencyDetails
		info.Accounts = append(info.Accounts, acc)
	}

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HUOBI) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HUOBI) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HUOBI) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	accountID, err := strconv.ParseInt(order.ClientID, 10, 64)
	if err != nil {
		return submitOrderResponse, err
	}

	var formattedType SpotNewOrderRequestParamsType
	var params = SpotNewOrderRequestParams{
		Amount:    order.Amount,
		Source:    "api",
		Symbol:    strings.ToLower(order.Pair.String()),
		AccountID: int(accountID),
	}

	switch {
	case order.OrderSide == exchange.BuyOrderSide && order.OrderType == exchange.MarketOrderType:
		formattedType = SpotNewOrderRequestTypeBuyMarket
	case order.OrderSide == exchange.SellOrderSide && order.OrderType == exchange.MarketOrderType:
		formattedType = SpotNewOrderRequestTypeSellMarket
	case order.OrderSide == exchange.BuyOrderSide && order.OrderType == exchange.LimitOrderType:
		formattedType = SpotNewOrderRequestTypeBuyLimit
		params.Price = order.Price
	case order.OrderSide == exchange.SellOrderSide && order.OrderType == exchange.LimitOrderType:
		formattedType = SpotNewOrderRequestTypeSellLimit
		params.Price = order.Price
	}

	params.Type = formattedType
	response, err := h.SpotNewOrder(params)
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
func (h *HUOBI) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HUOBI) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HUOBI) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var cancelAllOrdersResponse exchange.CancelAllOrdersResponse
	for _, currency := range h.GetEnabledPairs(asset.Spot) {
		resp, err := h.CancelOpenOrdersBatch(orderCancellation.AccountID,
			h.FormatExchangeCurrency(currency, asset.Spot).String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		if resp.Data.FailedCount > 0 {
			return cancelAllOrdersResponse, fmt.Errorf("%v orders failed to cancel", resp.Data.FailedCount)
		}

		if resp.Status == "error" {
			return cancelAllOrdersResponse, errors.New(resp.ErrorMessage)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HUOBI) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HUOBI) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HUOBI) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	resp, err := h.Withdraw(withdrawRequest.Currency, withdrawRequest.Address, withdrawRequest.AddressTag, withdrawRequest.Amount, withdrawRequest.FeeAmount)
	return fmt.Sprintf("%v", resp), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HUOBI) GetWebsocket() (*wshandler.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBI) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !h.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HUOBI) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	side := ""
	if getOrdersRequest.OrderSide == exchange.AnyOrderSide || getOrdersRequest.OrderSide == "" {
		side = ""
	} else if getOrdersRequest.OrderSide == exchange.SellOrderSide {
		side = strings.ToLower(string(getOrdersRequest.OrderSide))
	}

	var orders []exchange.OrderDetail

	for _, c := range getOrdersRequest.Currencies {
		resp, err := h.GetOpenOrders(h.API.Credentials.ClientID,
			c.Lower().String(), side, 500)
		if err != nil {
			return nil, err
		}

		for i := range resp {
			orderDetail := exchange.OrderDetail{
				ID:             fmt.Sprintf("%v", resp[i].ID),
				Price:          resp[i].Price,
				Amount:         resp[i].Amount,
				CurrencyPair:   c,
				Exchange:       h.Name,
				ExecutedAmount: resp[i].FilledAmount,
				OrderDate:      time.Unix(0, resp[i].CreatedAt*int64(time.Millisecond)),
				Status:         resp[i].State,
				AccountID:      strconv.FormatFloat(resp[i].AccountID, 'f', -1, 64),
				Fee:            resp[i].FilledFees,
			}

			setOrderSideAndType(resp[i].Type, &orderDetail)

			orders = append(orders, orderDetail)
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HUOBI) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	states := "partial-canceled,filled,canceled"
	var orders []exchange.OrderDetail
	for _, c := range getOrdersRequest.Currencies {
		resp, err := h.GetOrders(c.Lower().String(),
			"",
			"",
			"",
			states,
			"",
			"",
			"")
		if err != nil {
			return nil, err
		}

		for i := range resp {
			orderDetail := exchange.OrderDetail{
				ID:             fmt.Sprintf("%v", resp[i].ID),
				Price:          resp[i].Price,
				Amount:         resp[i].Amount,
				CurrencyPair:   c,
				Exchange:       h.Name,
				ExecutedAmount: resp[i].FilledAmount,
				OrderDate:      time.Unix(0, resp[i].CreatedAt*int64(time.Millisecond)),
				Status:         resp[i].State,
				AccountID:      strconv.FormatFloat(resp[i].AccountID, 'f', -1, 64),
				Fee:            resp[i].FilledFees,
			}

			setOrderSideAndType(resp[i].Type, &orderDetail)

			orders = append(orders, orderDetail)
		}
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	return orders, nil
}

func setOrderSideAndType(requestType string, orderDetail *exchange.OrderDetail) {
	switch SpotNewOrderRequestParamsType(requestType) {
	case SpotNewOrderRequestTypeBuyMarket:
		orderDetail.OrderSide = exchange.BuyOrderSide
		orderDetail.OrderType = exchange.MarketOrderType
	case SpotNewOrderRequestTypeSellMarket:
		orderDetail.OrderSide = exchange.SellOrderSide
		orderDetail.OrderType = exchange.MarketOrderType
	case SpotNewOrderRequestTypeBuyLimit:
		orderDetail.OrderSide = exchange.BuyOrderSide
		orderDetail.OrderType = exchange.LimitOrderType
	case SpotNewOrderRequestTypeSellLimit:
		orderDetail.OrderSide = exchange.SellOrderSide
		orderDetail.OrderType = exchange.LimitOrderType
	}
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (h *HUOBI) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (h *HUOBI) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (h *HUOBI) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return h.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (h *HUOBI) AuthenticateWebsocket() error {
	return h.wsLogin()
}
