package bithumb

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the bithumb go routine
func (b *Bithumb) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the bithumb wrapper
func (b *Bithumb) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	exchangeProducts, err := b.GetTradingPairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		err = b.UpdateCurrencies(exchangeProducts, false, false)
		if err != nil {
			log.Printf("%s Failed to update available symbols.\n", b.GetName())
		}
	}
}

// GetTradingPairs gets the available trading currencies
func (b *Bithumb) GetTradingPairs() ([]string, error) {
	currencies, err := b.GetTradablePairs()
	if err != nil {
		return nil, err
	}

	for x := range currencies {
		currencies[x] = currencies[x] + "KRW"
	}

	return currencies, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bithumb) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	tickers, err := b.GetAllTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledCurrencies() {
		currency := x.FirstCurrency.String()
		var tp ticker.Price
		tp.Pair = x
		tp.Ask = tickers[currency].SellPrice
		tp.Bid = tickers[currency].BuyPrice
		tp.Low = tickers[currency].MinPrice
		tp.Last = tickers[currency].ClosingPrice
		tp.Volume = tickers[currency].Volume1Day
		tp.High = tickers[currency].MaxPrice
		ticker.ProcessTicker(b.Name, x, tp, assetType)
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bithumb) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *Bithumb) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bithumb) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := p.FirstCurrency.String()

	orderbookNew, err := b.GetOrderBook(currency)
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Bithumb) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bithumb) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bithumb) GetExchangeHistory(p pair.CurrencyPair, assetType string, timestampStart time.Time, tradeID int64) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory
	trans, err := b.GetTransactionHistory(p.GetFirstCurrency().String(), tradeID)
	if err != nil {
		return resp, err
	}

	for _, data := range trans.Data {
		resp = append(resp, exchange.TradeHistory{
			Timestamp: ConvertToRFC3339(data.TransactionDate),
			TID:       data.ContNumber,
			Price:     data.Price,
			Amount:    data.UnitsTraded,
			Exchange:  b.GetName(),
			Type:      data.Type,
		})
	}

	return resp, nil
}

// ConvertToRFC3339 converts string from bithumb to a RFC3339 format
func ConvertToRFC3339(t string) time.Time {
	split := common.SplitStrings(t, " ")
	join := common.JoinStrings(split, "T")
	join += "Z"
	newTime, _ := time.Parse(time.RFC3339, join)
	return newTime
}

// SubmitOrder submits a new order
func (b *Bithumb) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var err error
	var orderID string
	if side == exchange.Buy {
		var result MarketBuy
		result, err = b.MarketBuyOrder(p.FirstCurrency.String(), amount)
		orderID = result.OrderID
	} else if side == exchange.Sell {
		var result MarketSell
		result, err = b.MarketSellOrder(p.FirstCurrency.String(), amount)
		orderID = result.OrderID
	}

	if orderID != "" {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bithumb) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bithumb) CancelOrder(order exchange.OrderCancellation) error {
	_, err := b.CancelTrade(order.Side.ToString(), order.OrderID, order.CurrencyPair.FirstCurrency.String())
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bithumb) CancelAllOrders() error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *Bithumb) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bithumb) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bithumb) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bithumb) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bithumb) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bithumb) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bithumb) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Bithumb) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}
