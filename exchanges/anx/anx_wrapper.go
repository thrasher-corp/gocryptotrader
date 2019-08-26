package anx

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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

// GetDefaultConfig returns a default exchange config for Alphapoint
func (a *ANX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	a.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = a.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = a.BaseCurrencies

	err := a.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if a.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = a.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default settings
func (a *ANX) SetDefaults() {
	a.Name = "ANX"
	a.Enabled = true
	a.Verbose = true
	a.BaseCurrencies = currency.Currencies{
		currency.USD,
		currency.HKD,
		currency.EUR,
		currency.CAD,
		currency.AUD,
		currency.SGD,
		currency.JPY,
		currency.GBP,
		currency.NZD,
	}
	a.API.CredentialsValidator.RequiresKey = true
	a.API.CredentialsValidator.RequiresSecret = true
	a.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	a.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	a.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
			WithdrawPermissions: exchange.WithdrawCryptoWithEmail |
				exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: false,
		},
	}

	a.Requester = request.New(a.Name,
		request.NewRateLimit(time.Second, anxAuthRate),
		request.NewRateLimit(time.Second, anxUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	a.API.Endpoints.URLDefault = anxAPIURL
	a.API.Endpoints.URL = a.API.Endpoints.URLDefault
}

// Setup is run on startup to setup exchange with config values
func (a *ANX) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		a.SetEnabled(false)
		return nil
	}

	return a.SetupDefaults(exch)
}

// Start starts the ANX go routine
func (a *ANX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		a.Run()
		wg.Done()
	}()
}

// Run implements the ANX wrapper
func (a *ANX) Run() {
	if a.Verbose {
		a.PrintEnabledPairs()
	}

	forceUpdate := false
	if !common.StringDataContains(a.GetEnabledPairs(asset.Spot).Strings(), "_") ||
		!common.StringDataContains(a.GetAvailablePairs(asset.Spot).Strings(), "_") {
		enabledPairs := currency.NewPairsFromStrings([]string{"BTC_USD,BTC_HKD,BTC_EUR,BTC_CAD,BTC_AUD,BTC_SGD,BTC_JPY,BTC_GBP,BTC_NZD,LTC_BTC,DOG_EBTC,STR_BTC,XRP_BTC"})
		log.Warn(log.ExchangeSys,
			"Enabled pairs for ANX reset due to config upgrade, please enable the ones you would like again.")

		forceUpdate = true
		err := a.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies.\n", a.GetName())
			return
		}
	}

	if !a.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := a.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", a.GetName(), err)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *ANX) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := a.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return a.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *ANX) FetchTradablePairs(asset asset.Item) ([]string, error) {
	result, err := a.GetCurrencies()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result.CurrencyPairs {
		currencies = append(currencies, result.CurrencyPairs[x].TradedCcy+"_"+result.CurrencyPairs[x].SettlementCcy)
	}

	return currencies, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *ANX) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(a.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return tickerPrice, err
	}
	last, _ := convert.FloatFromString(tick.Data.Last.Value)
	high, _ := convert.FloatFromString(tick.Data.High.Value)
	low, _ := convert.FloatFromString(tick.Data.Low.Value)
	bid, _ := convert.FloatFromString(tick.Data.Buy.Value)
	ask, _ := convert.FloatFromString(tick.Data.Sell.Value)
	volume, _ := convert.FloatFromString(tick.Data.Vol.Value)

	tickerPrice = ticker.Price{
		Last:        last,
		High:        high,
		Low:         low,
		Bid:         bid,
		Ask:         ask,
		Volume:      volume,
		Pair:        p,
		LastUpdated: time.Unix(0, tick.Data.UpdateTime),
	}

	err = ticker.ProcessTicker(a.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(a.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (a *ANX) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (a *ANX) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetDepth(a.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Price:  orderbookNew.Data.Asks[x].Price,
				Amount: orderbookNew.Data.Asks[x].Amount})
	}

	for x := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Price:  orderbookNew.Data.Bids[x].Price,
				Amount: orderbookNew.Data.Bids[x].Amount})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = a.GetName()
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(a.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies on the
// exchange
func (a *ANX) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	raw, err := a.GetAccountInformation()
	if err != nil {
		return info, err
	}

	var balance []exchange.AccountCurrencyInfo
	for c := range raw.Wallets {
		balance = append(balance, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(c),
			TotalValue:   raw.Wallets[c].AvailableBalance.Value,
			Hold:         raw.Wallets[c].Balance.Value,
		})
	}

	info.Exchange = a.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *ANX) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *ANX) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (a *ANX) SubmitOrder(order *exchange.OrderSubmission) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	if order == nil {
		return submitOrderResponse, exchange.ErrOrderSubmissionIsNil
	}

	if err := order.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var isBuying bool
	var limitPriceInSettlementCurrency float64

	if order.OrderSide == exchange.BuyOrderSide {
		isBuying = true
	}

	if order.OrderType == exchange.LimitOrderType {
		limitPriceInSettlementCurrency = order.Price
	}

	response, err := a.NewOrder(order.OrderType.ToString(),
		isBuying,
		order.Pair.Base.String(),
		order.Amount,
		order.Pair.Quote.String(),
		order.Amount,
		limitPriceInSettlementCurrency,
		false,
		"",
		false)

	if response != "" {
		submitOrderResponse.OrderID = response
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (a *ANX) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (a *ANX) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDs := []string{order.OrderID}
	_, err := a.CancelOrderByIDs(orderIDs)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (a *ANX) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	placedOrders, err := a.GetOrderList(true)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderIDs []string
	for i := range placedOrders {
		orderIDs = append(orderIDs, placedOrders[i].OrderID)
	}

	resp, err := a.CancelOrderByIDs(orderIDs)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range resp.OrderCancellationResponses {
		if order.Error != CancelRequestSubmitted {
			cancelAllOrdersResponse.OrderStatus[order.UUID] = order.Error
		}
	}

	return cancelAllOrdersResponse, err
}

// GetOrderInfo returns information on a current open order
func (a *ANX) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *ANX) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return a.GetDepositAddressByCurrency(cryptocurrency.String(), "", false)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	return a.Send(withdrawRequest.Currency.String(), withdrawRequest.Address, "", fmt.Sprintf("%v", withdrawRequest.Amount))
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *ANX) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *ANX) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!a.AllowAuthenticatedRequest() || a.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return a.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (a *ANX) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrderList(true)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp {
		orderDate := time.Unix(resp[i].Timestamp, 0)
		orderType := exchange.OrderType(strings.ToUpper(resp[i].OrderType))

		orderDetail := exchange.OrderDetail{
			Amount: resp[i].TradedCurrencyAmount,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].TradedCurrency,
				resp[i].SettlementCurrency,
				a.GetPairFormat(asset.Spot, false).Delimiter),
			OrderDate: orderDate,
			Exchange:  a.Name,
			ID:        resp[i].OrderID,
			OrderType: orderType,
			Price:     resp[i].SettlementCurrencyAmount,
			Status:    resp[i].OrderStatus,
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (a *ANX) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrderList(false)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp {
		orderDate := time.Unix(resp[i].Timestamp, 0)
		orderType := exchange.OrderType(strings.ToUpper(resp[i].OrderType))

		orderDetail := exchange.OrderDetail{
			Amount:    resp[i].TradedCurrencyAmount,
			OrderDate: orderDate,
			Exchange:  a.Name,
			ID:        resp[i].OrderID,
			OrderType: orderType,
			Price:     resp[i].SettlementCurrencyAmount,
			Status:    resp[i].OrderStatus,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].TradedCurrency,
				resp[i].SettlementCurrency,
				a.GetPairFormat(asset.Spot, false).Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (a *ANX) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (a *ANX) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (a *ANX) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (a *ANX) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
