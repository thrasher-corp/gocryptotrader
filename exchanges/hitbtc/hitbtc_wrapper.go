package hitbtc

import (
	"errors"
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
	h.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	h.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
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
	h.WebsocketInit()
	h.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported
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

	return h.WebsocketSetup(h.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		hitbtcWebsocketAddress,
		exch.API.Endpoints.WebsocketURL)
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
		log.Debugf("%s Websocket: %s (url: %s).\n", h.GetName(), common.IsEnabled(h.Websocket.IsEnabled()), hitbtcWebsocketAddress)
		h.PrintEnabledPairs()
	}

	forceUpdate := false
	if !common.StringDataContains(h.GetEnabledPairs(assets.AssetTypeSpot).Strings(), "-") ||
		!common.StringDataContains(h.GetAvailablePairs(assets.AssetTypeSpot).Strings(), "-") {
		enabledPairs := []string{"BTC-USD"}
		log.Warn("WARNING: Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")
		forceUpdate = true

		err := h.UpdatePairs(currency.NewPairsFromStrings(enabledPairs), assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update enabled currencies.\n", h.GetName())
		}
	}

	if !h.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := h.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", h.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HitBTC) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
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
	pairs, err := h.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return h.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(currencyPair currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := h.GetTicker("")
	if err != nil {
		return ticker.Price{}, err
	}

	for _, x := range h.GetEnabledPairs(assetType) {
		var tp ticker.Price
		curr := h.FormatExchangeCurrency(x, assetType).String()
		tp.Pair = x
		tp.Ask = tick[curr].Ask
		tp.Bid = tick[curr].Bid
		tp.High = tick[curr].High
		tp.Last = tick[curr].Last
		tp.Low = tick[curr].Low
		tp.Volume = tick[curr].Volume

		err = ticker.ProcessTicker(h.GetName(), &tp, assetType)
		if err != nil {
			return ticker.Price{}, err
		}
	}
	return ticker.GetTicker(h.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HitBTC) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HitBTC) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(h.GetName(), p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(currencyPair currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
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
func (h *HitBTC) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := h.PlaceOrder(p.String(),
		price,
		amount,
		common.StringToLower(orderType.ToString()),
		common.StringToLower(side.ToString()))

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
		cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(resp[i].ID, 10)] = fmt.Sprintf("Could not cancel order %v. Status: %v", resp[i].ID, resp[i].Status)
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
func (h *HitBTC) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	_, err := h.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)

	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HitBTC) GetWebsocket() (*exchange.Websocket, error) {
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
			h.CurrencyPairs.ConfigFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[i].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[i].CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				h.Name, "GetActiveOrders", allOrders[i].ID, allOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:           allOrders[i].ID,
			Amount:       allOrders[i].Quantity,
			Exchange:     h.Name,
			Price:        allOrders[i].Price,
			OrderDate:    orderDate,
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
			h.CurrencyPairs.ConfigFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(allOrders[i].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[i].CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				h.Name, "GetOrderHistory", allOrders[i].ID, allOrders[i].CreatedAt)
		}

		orders = append(orders, exchange.OrderDetail{
			ID:           allOrders[i].ID,
			Amount:       allOrders[i].Quantity,
			Exchange:     h.Name,
			Price:        allOrders[i].Price,
			OrderDate:    orderDate,
			OrderSide:    side,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}
