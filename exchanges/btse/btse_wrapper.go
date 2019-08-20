package btse

import (
	"errors"
	"fmt"
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

// Start starts the BTSE go routine
func (b *BTSE) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTSE wrapper
func (b *BTSE) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.Websocket.GetWebsocketURL())
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	markets, err := b.GetMarkets()
	if err != nil {
		log.Errorf("%s failed to get trading pairs. Err: %s", b.Name, err)
	} else {
		var currencies []string
		for _, m := range *markets {
			if m.Status != "active" {
				continue
			}
			currencies = append(currencies, m.Symbol)
		}
		err = b.UpdateCurrencies(currency.NewPairsFromStrings(currencies),
			false,
			false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies. Error: %s\n",
				b.Name, err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTSE) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	t, err := b.GetTicker(exchange.FormatExchangeCurrency(b.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}

	s, err := b.GetMarketStatistics(exchange.FormatExchangeCurrency(b.Name, p).String())
	if err != nil {
		return tickerPrice, err

	}

	tickerPrice.Pair = p
	tickerPrice.Ask = t.Ask
	tickerPrice.Bid = t.Bid
	tickerPrice.Low = s.Low
	tickerPrice.Last = t.Price
	tickerPrice.Volume = s.Volume
	tickerPrice.High = s.High

	err = ticker.ProcessTicker(b.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTSE) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *BTSE) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTSE) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTSE exchange
func (b *BTSE) GetAccountInfo() (exchange.AccountInfo, error) {
	var a exchange.AccountInfo
	balance, err := b.GetAccountBalance()
	if err != nil {
		return a, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for _, b := range *balance {
		currencies = append(currencies,
			exchange.AccountCurrencyInfo{
				CurrencyName: currency.NewCode(b.Currency),
				TotalValue:   b.Total,
				Hold:         b.Available,
			},
		)
	}
	a.Exchange = b.Name
	a.Accounts = []exchange.Account{
		{
			Currencies: currencies,
		},
	}
	return a, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTSE) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTSE) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTSE) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	r, err := b.CreateOrder(amount, price, side.ToString(),
		orderType.ToString(), exchange.FormatExchangeCurrency(b.Name, p).String(), "GTC", clientID)
	if err != nil {
		return resp, err
	}

	if *r != "" {
		resp.IsOrderPlaced = true
		resp.OrderID = *r
	}

	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTSE) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTSE) CancelOrder(order *exchange.OrderCancellation) error {
	r, err := b.CancelExistingOrder(order.OrderID,
		exchange.FormatExchangeCurrency(b.Name, order.CurrencyPair).String())
	if err != nil {
		return err
	}

	switch r.Code {
	case -1:
		return errors.New("order cancellation unsuccessful")
	case 4:
		return errors.New("order cancellation timeout")
	}

	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
// If product ID is sent, all orders of that specified market will be cancelled
// If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var resp exchange.CancelAllOrdersResponse
	r, err := b.CancelOrders(exchange.FormatExchangeCurrency(b.Name,
		orderCancellation.CurrencyPair).String())
	if err != nil {
		return resp, err
	}

	switch r.Code {
	case -1:
		return resp, errors.New("order cancellation unsuccessful")
	case 4:
		return resp, errors.New("order cancellation timeout")
	}

	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTSE) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	o, err := b.GetOrders("")
	if err != nil {
		return exchange.OrderDetail{}, err
	}

	var od exchange.OrderDetail
	if len(*o) == 0 {
		return od, errors.New("no orders found")
	}

	for i := range *o {
		o := (*o)[i]
		if o.ID != orderID {
			continue
		}

		var side = exchange.BuyOrderSide
		if strings.EqualFold(o.Side, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		}

		od.CurrencyPair = currency.NewPairDelimiter(o.ProductID,
			b.ConfigCurrencyPairFormat.Delimiter)
		od.Exchange = b.Name
		od.Amount = o.Amount
		od.ID = o.ID
		od.OrderDate = parseOrderTime(o.CreatedAt)
		od.OrderSide = side
		od.OrderType = exchange.OrderType(strings.ToUpper(o.Type))
		od.Price = o.Price
		od.Status = o.Status

		fills, err := b.GetFills(orderID, "", "", "", "")
		if err != nil {
			return od, fmt.Errorf("unable to get order fills for orderID %s", orderID)
		}

		for i := range *fills {
			f := (*fills)[i]
			createdAt, _ := time.Parse(time.RFC3339, f.CreatedAt)
			od.Trades = append(od.Trades, exchange.TradeHistory{
				Timestamp: createdAt,
				TID:       f.ID,
				Price:     f.Price,
				Amount:    f.Amount,
				Exchange:  b.Name,
				Type:      exchange.OrderSide(f.Side).ToString(),
				Fee:       f.Fee,
			})
		}
	}
	return od, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTSE) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTSE) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTSE) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := b.GetOrders("")
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range *resp {
		order := (*resp)[i]
		var side = exchange.BuyOrderSide
		if strings.EqualFold(order.Side, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		}

		openOrder := exchange.OrderDetail{
			CurrencyPair: currency.NewPairDelimiter(order.ProductID,
				b.ConfigCurrencyPairFormat.Delimiter),
			Exchange:  b.Name,
			Amount:    order.Amount,
			ID:        order.ID,
			OrderDate: parseOrderTime(order.CreatedAt),
			OrderSide: side,
			OrderType: exchange.OrderType(strings.ToUpper(order.Type)),
			Price:     order.Price,
			Status:    order.Status,
		}

		fills, err := b.GetFills(order.ID, "", "", "", "")
		if err != nil {
			log.Errorf("unable to get order fills for orderID %s", order.ID)
			continue
		}

		for i := range *fills {
			f := (*fills)[i]
			createdAt, _ := time.Parse(time.RFC3339, f.CreatedAt)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Timestamp: createdAt,
				TID:       f.ID,
				Price:     f.Price,
				Amount:    f.Amount,
				Exchange:  b.Name,
				Type:      exchange.OrderSide(f.Side).ToString(),
				Fee:       f.Fee,
			})
		}
		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTSE) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTSE) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (b.APIKey == "" || b.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *BTSE) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *BTSE) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (b *BTSE) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *BTSE) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
