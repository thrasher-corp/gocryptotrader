package anx

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
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

	a.APIWithdrawPermissions = exchange.WithdrawCryptoWithEmail |
		exchange.AutoWithdrawCryptoWithSetup |
		exchange.WithdrawCryptoWith2FA |
		exchange.WithdrawFiatViaWebsiteOnly

	a.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
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
	if !common.StringDataContains(a.GetEnabledPairs(assets.AssetTypeSpot).Strings(), "_") ||
		!common.StringDataContains(a.GetAvailablePairs(assets.AssetTypeSpot).Strings(), "_") {
		enabledPairs := currency.NewPairsFromStrings([]string{"BTC_USD,BTC_HKD,BTC_EUR,BTC_CAD,BTC_AUD,BTC_SGD,BTC_JPY,BTC_GBP,BTC_NZD,LTC_BTC,DOG_EBTC,STR_BTC,XRP_BTC"})
		log.Warn("WARNING: Enabled pairs for ANX reset due to config upgrade, please enable the ones you would like again.")

		forceUpdate = true
		err := a.UpdatePairs(enabledPairs, assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update currencies.\n", a.GetName())
			return
		}
	}

	if !a.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := a.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", a.GetName(), err)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *ANX) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := a.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return a.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *ANX) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
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
func (a *ANX) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(a.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p

	if tick.Data.Sell.Value != "" {
		tickerPrice.Ask, err = strconv.ParseFloat(tick.Data.Sell.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Ask = 0
	}

	if tick.Data.Buy.Value != "" {
		tickerPrice.Bid, err = strconv.ParseFloat(tick.Data.Buy.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Bid = 0
	}

	if tick.Data.Low.Value != "" {
		tickerPrice.Low, err = strconv.ParseFloat(tick.Data.Low.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Low = 0
	}

	if tick.Data.Last.Value != "" {
		tickerPrice.Last, err = strconv.ParseFloat(tick.Data.Last.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Last = 0
	}

	if tick.Data.Vol.Value != "" {
		tickerPrice.Volume, err = strconv.ParseFloat(tick.Data.Vol.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.Volume = 0
	}

	if tick.Data.High.Value != "" {
		tickerPrice.High, err = strconv.ParseFloat(tick.Data.High.Value, 64)
		if err != nil {
			return tickerPrice, err
		}
	} else {
		tickerPrice.High = 0
	}

	err = ticker.ProcessTicker(a.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(a.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (a *ANX) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (a *ANX) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
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
func (a *ANX) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (a *ANX) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	var isBuying bool
	var limitPriceInSettlementCurrency float64

	if side == exchange.BuyOrderSide {
		isBuying = true
	}

	if orderType == exchange.LimitOrderType {
		limitPriceInSettlementCurrency = price
	}

	response, err := a.NewOrder(orderType.ToString(),
		isBuying,
		p.Base.String(),
		amount,
		p.Quote.String(),
		amount,
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
func (a *ANX) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return a.Send(withdrawRequest.Currency.String(), withdrawRequest.Address, "", fmt.Sprintf("%v", withdrawRequest.Amount))
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *ANX) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *ANX) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !a.AllowAuthenticatedRequest() && // Todo check connection status
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
				resp[i].SettlementCurrency, a.CurrencyPairs.ConfigFormat.Delimiter),
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
				a.CurrencyPairs.ConfigFormat.Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}
