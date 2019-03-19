package bitflyer

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the Bitflyer go routine
func (b *Bitflyer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitflyer wrapper
func (b *Bitflyer) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	/*
		marketInfo, err := b.GetMarkets()
		if err != nil {
			log.Printf("%s Failed to get available symbols.\n", b.GetName())
		} else {
			var exchangeProducts []string

			for _, info := range marketInfo {
				exchangeProducts = append(exchangeProducts, info.ProductCode)
			}

			err = b.UpdateAvailableCurrencies(exchangeProducts, false)
			if err != nil {
				log.Printf("%s Failed to get config.\n", b.GetName())
			}
		}
	*/
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitflyer) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	p = b.CheckFXString(p)

	tickerNew, err := b.GetTicker(p.String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tickerNew.BestAsk
	tickerPrice.Bid = tickerNew.BestBid
	// tickerPrice.Low
	tickerPrice.Last = tickerNew.Last
	tickerPrice.Volume = tickerNew.Volume
	// tickerPrice.High
	err = ticker.ProcessTicker(b.GetName(), tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitflyer) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, ticker.Spot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// CheckFXString upgrades currency pair if needed
func (b *Bitflyer) CheckFXString(p currency.Pair) currency.Pair {
	if common.StringContains(p.Base.String(), "FX") {
		p.Base = currency.FX_BTC
		return p
	}
	return p
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bitflyer) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitflyer) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base

	p = b.CheckFXString(p)

	orderbookNew, err := b.GetOrderBook(p.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: orderbookNew.Asks[x].Price, Amount: orderbookNew.Asks[x].Size})
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: orderbookNew.Bids[x].Price, Amount: orderbookNew.Bids[x].Size})
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

// GetAccountInfo retrieves balances for all enabled currencies on the
// Bitflyer exchange
func (b *Bitflyer) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()
	// accountBalance, err := b.GetAccountBalance()
	// if err != nil {
	// 	return response, err
	// }
	if !b.Enabled {
		return response, errors.New("exchange not enabled")
	}

	// implement once authenticated requests are introduced

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitflyer) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetPlatformHistory returns historic platform trade data since exchange
// intial operations
func (b *Bitflyer) GetPlatformHistory(p currency.Pair, assetType string, timestampStart time.Time, tradeID string) ([]exchange.PlatformTrade, error) {
	var resp []exchange.PlatformTrade
	ID, err := strconv.ParseInt(tradeID, 10, 64)
	if err != nil {
		return nil, err
	}

	trades, err := b.GetExecutionHistory(p.String(), ID)
	if err != nil {
		return resp, err
	}

	for i := range trades {
		t, err := time.Parse(time.RFC3339, trades[i].ExecDate+"Z")
		if err != nil {
			return resp, err
		}

		orderID := strconv.FormatInt(trades[i].ID, 10)

		resp = append(resp, exchange.PlatformTrade{
			Timestamp: t,
			TID:       orderID,
			Price:     trades[i].Price,
			Amount:    trades[i].Size,
			Exchange:  b.GetName(),
			Type:      trades[i].Side,
		})
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (b *Bitflyer) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitflyer) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitflyer) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitflyer) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// TODO, implement BitFlyer API
	b.CancelAllExistingOrders()
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *Bitflyer) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitflyer) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitflyer) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitflyer) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitflyer) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitflyer) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
