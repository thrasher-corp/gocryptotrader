package gemini

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the Gemini go routine
func (g *Gemini) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
}

// Run implements the Gemini wrapper
func (g *Gemini) Run() {
	if g.Verbose {
		log.Debugf("%s polling delay: %ds.\n", g.GetName(), g.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", g.GetName(), len(g.EnabledPairs), g.EnabledPairs)
	}

	exchangeProducts, err := g.GetSymbols()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", g.GetName())
	} else {
		var newExchangeProducts currency.Pairs
		for _, p := range exchangeProducts {
			newExchangeProducts = append(newExchangeProducts,
				currency.NewPairFromString(p))
		}

		err = g.UpdateCurrencies(newExchangeProducts, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", g.GetName())
		}
	}
}

// GetAccountInfo Retrieves balances for all enabled currencies for the
// Gemini exchange
func (g *Gemini) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = g.GetName()
	accountBalance, err := g.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Amount
		exchangeCurrency.Hold = accountBalance[i].Available
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gemini) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := g.GetTicker(p.String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume.USD

	err = ticker.ProcessTicker(g.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(g.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (g *Gemini) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (g *Gemini) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(g.GetName(), p, assetType)
	if err != nil {
		return g.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gemini) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := g.GetOrderbook(p.String(), url.Values{})
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
	orderBook.ExchangeName = g.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(g.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gemini) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (g *Gemini) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (g *Gemini) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	p = exchange.FormatExchangeCurrency(g.Name, p)

	if orderType != exchange.LimitOrderType {
		return submitOrderResponse, errors.New("only limit orders are enabled through this API")
	}

	response, err := g.NewOrder(p.String(),
		amount,
		price,
		side.ToString(),
		"exchange limit")

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
func (g *Gemini) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gemini) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = g.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gemini) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	resp, err := g.CancelExistingOrders(false)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, order := range resp.Details.CancelRejects {
		cancelAllOrdersResponse.OrderStatus[order] = "Could not cancel order"
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (g *Gemini) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gemini) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addr, err := g.GetCryptoDepositAddress("", cryptocurrency.String())
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gemini) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	resp, err := g.WithdrawCrypto(withdrawRequest.Address, withdrawRequest.Currency.String(), withdrawRequest.Amount)
	if err != nil {
		return "", err
	}
	if resp.Result == "error" {
		return "", errors.New(resp.Message)
	}

	return resp.TXHash, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gemini) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (g *Gemini) GetWebsocket() (*wshandler.Websocket, error) {
	return g.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gemini) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (g.APIKey == "" || g.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gemini) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := g.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp {
		symbol := currency.NewPairDelimiter(resp[i].Symbol,
			g.ConfigCurrencyPairFormat.Delimiter)
		var orderType exchange.OrderType
		if resp[i].Type == "exchange limit" {
			orderType = exchange.LimitOrderType
		} else if resp[i].Type == "market buy" || resp[i].Type == "market sell" {
			orderType = exchange.MarketOrderType
		}

		side := exchange.OrderSide(strings.ToUpper(resp[i].Type))
		orderDate := time.Unix(resp[i].Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			Amount:          resp[i].OriginalAmount,
			RemainingAmount: resp[i].RemainingAmount,
			ID:              fmt.Sprintf("%v", resp[i].OrderID),
			ExecutedAmount:  resp[i].ExecutedAmount,
			Exchange:        g.Name,
			OrderType:       orderType,
			OrderSide:       side,
			Price:           resp[i].Price,
			CurrencyPair:    symbol,
			OrderDate:       orderDate,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gemini) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var trades []TradeHistory
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := g.GetTradeHistory(exchange.FormatExchangeCurrency(g.Name, currency).String(),
			getOrdersRequest.StartTicks.Unix())
		if err != nil {
			return nil, err
		}

		for i := range resp {
			resp[i].BaseCurrency = currency.Base.String()
			resp[i].QuoteCurrency = currency.Quote.String()
			trades = append(trades, resp[i])
		}
	}

	var orders []exchange.OrderDetail
	for i := range trades {
		side := exchange.OrderSide(strings.ToUpper(trades[i].Type))
		orderDate := time.Unix(trades[i].Timestamp, 0)

		orders = append(orders, exchange.OrderDetail{
			Amount:    trades[i].Amount,
			ID:        fmt.Sprintf("%v", trades[i].OrderID),
			Exchange:  g.Name,
			OrderDate: orderDate,
			OrderSide: side,
			Fee:       trades[i].FeeAmount,
			Price:     trades[i].Price,
			CurrencyPair: currency.NewPairWithDelimiter(trades[i].BaseCurrency,
				trades[i].QuoteCurrency,
				g.ConfigCurrencyPairFormat.Delimiter),
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (g *Gemini) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (g *Gemini) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (g *Gemini) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (g *Gemini) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
