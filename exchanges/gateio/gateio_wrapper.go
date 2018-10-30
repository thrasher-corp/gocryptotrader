package gateio

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
func (g *Gateio) GetDefaultConfig() (*config.ExchangeConfig, error) {
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
		err = g.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = true
	g.Verbose = true
	g.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.NoFiatWithdrawals
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	g.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	g.Features = exchange.Features{
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

	g.Requester = request.New(g.Name,
		request.NewRateLimit(time.Second*10, gateioAuthRate),
		request.NewRateLimit(time.Second*10, gateioUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	g.API.Endpoints.URLDefault = gateioTradeURL
	g.API.Endpoints.URL = g.API.Endpoints.URLDefault
	g.API.Endpoints.URLSecondaryDefault = gateioMarketURL
	g.API.Endpoints.URLSecondary = g.API.Endpoints.URLSecondaryDefault
	g.WebsocketInit()
	g.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketOrderbookSupported |
		exchange.WebsocketKlineSupported
}

// Setup sets user configuration
func (g *Gateio) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}

	err := g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return g.WebsocketSetup(g.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		gateioWebsocketEndpoint,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the GateIO go routine
func (g *Gateio) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the GateIO wrapper
func (g *Gateio) Run() {
	if g.Verbose {
		g.PrintEnabledPairs()
	}

	if !g.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := g.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", g.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gateio) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	return g.GetSymbols()
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gateio) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := g.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return g.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	result, err := g.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range g.GetEnabledPairs(assetType) {
		currency := g.FormatExchangeCurrency(x, assetType).String()
		var tp ticker.Price
		tp.Pair = x
		tp.High = result[currency].High
		tp.Last = result[currency].Last
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].Volume

		err = ticker.ProcessTicker(g.Name, &tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}

	return ticker.GetTicker(g.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gateio) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gateio) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := g.FormatExchangeCurrency(p, assetType).String()

	orderbookNew, err := g.GetOrderbook(currency)
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

	orderBook.Pair = p
	orderBook.ExchangeName = g.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(g.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (g *Gateio) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	balance, err := g.GetBalances()
	if err != nil {
		return info, err
	}

	if len(balance.Available) == 0 && len(balance.Locked) == 0 {
		return info, nil
	}

	var balances []exchange.AccountCurrencyInfo

	for key, amountStr := range balance.Locked {

		lockedF, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return info, err
		}

		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(key),
			Hold:         lockedF,
		})
	}

	for key, amountStr := range balance.Available {
		availAmount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return info, err
		}

		var updated bool
		for i := range balances {
			if balances[i].CurrencyName == currency.NewCode(key) {
				balances[i].TotalValue = balances[i].Hold + availAmount
				updated = true
				break
			}
		}

		if !updated {
			balances = append(balances, exchange.AccountCurrencyInfo{
				CurrencyName: currency.NewCode(key),
				TotalValue:   availAmount,
			})
		}
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	info.Exchange = g.GetName()

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gateio) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (g *Gateio) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var orderTypeFormat string

	if side == exchange.BuyOrderSide {
		orderTypeFormat = exchange.BuyOrderSide.ToLower().ToString()
	} else {
		orderTypeFormat = exchange.SellOrderSide.ToLower().ToString()
	}

	var spotNewOrderRequestParams = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: p.String(),
		Type:   orderTypeFormat,
	}

	response, err := g.SpotNewOrder(spotNewOrderRequestParams)

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
func (g *Gateio) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}
	_, err = g.CancelExistingOrder(orderIDInt, g.FormatExchangeCurrency(order.CurrencyPair,
		order.AssetType).String())

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := g.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	uniqueSymbols := make(map[string]string)
	for i := range openOrders.Orders {
		uniqueSymbols[openOrders.Orders[i].CurrencyPair] = openOrders.Orders[i].CurrencyPair
	}

	for unique := range uniqueSymbols {
		err = g.CancelAllExistingOrders(-1, uniqueSymbols[unique])
		if err != nil {
			return cancelAllOrdersResponse, err
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (g *Gateio) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gateio) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addr, err := g.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	// Waits for new generated address if not created yet, its variable per
	// currency
	if addr == gateioGenerateAddress {
		time.Sleep(10 * time.Second)
		addr, err = g.GetCryptoDepositAddress(cryptocurrency.String())
		if err != nil {
			return "", err
		}
		if addr == gateioGenerateAddress {
			return "", errors.New("new deposit address is being generated, please retry again shortly")
		}
		return addr, nil
	}

	return addr, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gateio) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return g.WithdrawCrypto(withdrawRequest.Currency.String(), withdrawRequest.Address, withdrawRequest.Amount)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gateio) GetWebsocket() (*exchange.Websocket, error) {
	return g.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !g.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gateio) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := g.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Orders {
		if resp.Orders[i].Status != "open" {
			continue
		}

		symbol := currency.NewPairDelimiter(resp.Orders[i].CurrencyPair,
			g.CurrencyPairs.ConfigFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(resp.Orders[i].Type))
		orderDate := time.Unix(resp.Orders[i].Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			ID:              resp.Orders[i].OrderNumber,
			Amount:          resp.Orders[i].Amount,
			Price:           resp.Orders[i].Rate,
			RemainingAmount: resp.Orders[i].FilledAmount,
			OrderDate:       orderDate,
			OrderSide:       side,
			Exchange:        g.Name,
			CurrencyPair:    symbol,
			Status:          resp.Orders[i].Status,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var trades []TradesResponse
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := g.GetTradeHistory(currency.String())
		if err != nil {
			return nil, err
		}
		trades = append(trades, resp.Trades...)
	}

	var orders []exchange.OrderDetail
	for _, trade := range trades {
		symbol := currency.NewPairDelimiter(trade.Pair,
			g.CurrencyPairs.ConfigFormat.Delimiter)
		side := exchange.OrderSide(strings.ToUpper(trade.Type))
		orderDate := time.Unix(trade.TimeUnix, 0)
		orders = append(orders, exchange.OrderDetail{
			ID:           strconv.FormatInt(trade.OrderID, 10),
			Amount:       trade.Amount,
			Price:        trade.Rate,
			OrderDate:    orderDate,
			OrderSide:    side,
			Exchange:     g.Name,
			CurrencyPair: symbol,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}
