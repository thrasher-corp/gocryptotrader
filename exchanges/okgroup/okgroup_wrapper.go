package okgroup

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Note: GoCryptoTrader wrapper funcs currently only support SPOT trades.
// Therefore this OKGroup_Wrapper can be shared between OKEX and OKCoin.
// When circumstances change, wrapper funcs can be split appropriately

// Start starts the OKGroup go routine
func (o *OKGroup) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKGroup) Run() {
	if o.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	prods, err := o.GetSpotTokenPairDetails()
	if err != nil {
		log.Errorf("%v failed to obtain available spot instruments. Err: %d", o.Name, err)
		return
	}

	var pairs []string
	for x := range prods {
		pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
	}

	err = o.UpdateCurrencies(pairs, false, false)
	if err != nil {
		log.Errorf("%v failed to update available currencies. Err: %s", o.Name, err)
		return
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKGroup) UpdateTicker(p pair.CurrencyPair, assetType string) (tickerData ticker.Price, err error) {
	resp, err := o.GetSpotAllTokenPairsInformationForCurrency(exchange.FormatExchangeCurrency(o.Name, p).String())
	if err != nil {
		return
	}
	respTime, err := time.Parse(time.RFC3339Nano, resp.Timestamp)
	if err != nil {
		log.Warnf("Exchange %v Func %v Currency %v Could not parse date to unix with value of %v",
			o.Name, "UpdateTicker", exchange.FormatExchangeCurrency(o.Name, p).String(), resp.Timestamp)
	}
	tickerData = ticker.Price{
		Ask:          resp.BestAsk,
		Bid:          resp.BestBid,
		CurrencyPair: exchange.FormatExchangeCurrency(o.Name, p).String(),
		High:         resp.High24h,
		Last:         resp.Last,
		LastUpdated:  respTime,
		Low:          resp.Low24h,
		Pair:         p,
		Volume:       resp.BaseVolume24h,
	}

	ticker.ProcessTicker(o.Name, p, tickerData, assetType)

	return
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKGroup) GetTickerPrice(p pair.CurrencyPair, assetType string) (tickerData ticker.Price, err error) {
	tickerData, err = ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKGroup) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (resp orderbook.Base, err error) {
	_, err = o.GetSpotOrderBook(GetSpotOrderBookRequest{
		InstrumentID: exchange.FormatExchangeCurrency(o.Name, currency).String(),
	})
	if err != nil {
		return
	}
	var asks []orderbook.Item
	var bids []orderbook.Item
	/*for _, respAsk := range getSpotOrderBookResponse.Asks {
		asks = append(asks, orderbook.Item{
			Amount: respAsk[0],
			Price:  respAsk[2],
		})
	}

	for _, respBid := range resp.Bids {
		bids = append(asks, orderbook.Item{
			Amount: respAsk[3],
			Price:  respAsk[0],
		})
	}
	*/
	resp = orderbook.Base{
		Asks:         asks,
		Bids:         bids,
		Pair:         currency,
		CurrencyPair: exchange.FormatExchangeCurrency(o.Name, currency).String(),
		LastUpdated:  time.Now(),
		AssetType:    assetType,
	}
	return
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKGroup) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrNotYetImplemented
}

// GetAccountInfo retrieves balances for all enabled currencies
func (o *OKGroup) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKGroup) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKGroup) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *OKGroup) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	return exchange.SubmitOrderResponse{}, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKGroup) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKGroup) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *OKGroup) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (o *OKGroup) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	return exchange.OrderDetail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKGroup) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKGroup) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (o *OKGroup) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (o *OKGroup) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKGroup) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKGroup) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
