package localbitcoins

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/symbol"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the LocalBitcoins go routine
func (l *LocalBitcoins) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LocalBitcoins wrapper
func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Debugf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	currencies, err := l.GetTradableCurrencies()
	if err != nil {
		log.Errorf("%s failed to obtain available tradable currencies. Err: %s", l.Name, err)
		return
	}

	var pairs []string
	for x := range currencies {
		pairs = append(pairs, "BTC"+currencies[x])
	}

	err = l.UpdateCurrencies(pairs, false, false)
	if err != nil {
		log.Errorf("%s failed to update available currencies. Err %s", l.Name, err)
	}

}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LocalBitcoins) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := l.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := x.SecondCurrency.String()
		var tp ticker.Price
		tp.Pair = x
		tp.Last = tick[currency].Avg24h
		tp.Volume = tick[currency].VolumeBTC
		ticker.ProcessTicker(l.GetName(), x, tp, assetType)
	}

	return ticker.GetTicker(l.GetName(), p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LocalBitcoins) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LocalBitcoins) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LocalBitcoins) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderbook(p.SecondCurrency.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount / data.Price, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount / data.Price, Price: data.Price})
	}

	orderbook.ProcessOrderbook(l.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// LocalBitcoins exchange
func (l *LocalBitcoins) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = l.GetName()
	accountBalance, err := l.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency exchange.AccountCurrencyInfo
	exchangeCurrency.CurrencyName = "BTC"
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: []exchange.AccountCurrencyInfo{exchangeCurrency},
	})
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LocalBitcoins) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LocalBitcoins) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LocalBitcoins) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	// These are placeholder details
	// TODO store a user's localbitcoin details to use here
	var params = AdCreate{
		PriceEquation:              "USD_in_AUD",
		Latitude:                   1,
		Longitude:                  1,
		City:                       "City",
		Location:                   "Location",
		CountryCode:                "US",
		Currency:                   p.SecondCurrency.String(),
		AccountInfo:                "-",
		BankName:                   "Bank",
		MSG:                        side.ToString(),
		SMSVerficationRequired:     true,
		TrackMaxAmount:             true,
		RequireTrustedByAdvertiser: true,
		RequireIdentification:      true,
		OnlineProvider:             "",
		TradeType:                  "",
		MinAmount:                  int(math.Round(amount)),
	}

	// Does not return any orderID, so create the add, then get the order
	err := l.CreateAd(params)
	if err != nil {
		return submitOrderResponse, err
	}
	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	// Now to figure out what ad we just submitted
	// The only details we have are the params above
	var adID string
	ads, err := l.Getads()
	for _, i := range ads.AdList {
		if i.Data.PriceEquation == params.PriceEquation &&
			i.Data.Lat == float64(params.Latitude) &&
			i.Data.Lon == float64(params.Longitude) &&
			i.Data.City == params.City &&
			i.Data.Location == params.Location &&
			i.Data.CountryCode == params.CountryCode &&
			i.Data.Currency == params.Currency &&
			i.Data.AccountInfo == params.AccountInfo &&
			i.Data.BankName == params.BankName &&
			i.Data.SMSVerficationRequired == params.SMSVerficationRequired &&
			i.Data.TrackMaxAmount == params.TrackMaxAmount &&
			i.Data.RequireTrustedByAdvertiser == params.RequireTrustedByAdvertiser &&
			i.Data.OnlineProvider == params.OnlineProvider &&
			i.Data.TradeType == params.TradeType &&
			i.Data.MinAmount == fmt.Sprintf("%v", params.MinAmount) {
			adID = fmt.Sprintf("%v", i.Data.AdID)
		}
	}

	if adID != "" {
		submitOrderResponse.OrderID = adID
	} else {
		return submitOrderResponse, errors.New("Ad placed, but not found via API")
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LocalBitcoins) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LocalBitcoins) CancelOrder(order exchange.OrderCancellation) error {
	return l.DeleteAd(order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LocalBitcoins) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	ads, err := l.Getads()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for _, ad := range ads.AdList {
		adIDString := strconv.FormatInt(ad.Data.AdID, 10)
		err = l.DeleteAd(adIDString)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(ad.Data.AdID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (l *LocalBitcoins) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LocalBitcoins) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	if !strings.EqualFold(symbol.BTC, cryptocurrency.String()) {
		return "", fmt.Errorf("Localbitcoins do not have support for currency %s just bitcoin",
			cryptocurrency.String())
	}

	return l.GetWalletAddress()
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LocalBitcoins) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	_, err := l.WalletSend(withdrawRequest.Address, withdrawRequest.Amount, withdrawRequest.PIN)
	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LocalBitcoins) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LocalBitcoins) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return l.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (l *LocalBitcoins) GetWithdrawCapabilities() uint32 {
	return l.GetWithdrawPermissions()
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LocalBitcoins) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetDashboardInfo()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, trade := range resp {
		t, err := time.Parse(time.RFC3339, trade.Data.CreatedAt)
		if err != nil {
			log.Errorf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name, "GetActiveOrders", trade.Data.Advertisement.ID, trade.Data.CreatedAt)
		}

		side := ""
		if trade.Data.IsBuying {
			side = string(exchange.BuyOrderSide)
		} else if trade.Data.IsSelling {
			side = string(exchange.SellOrderSide)
		}

		orders = append(orders, exchange.OrderDetail{
			Amount:        trade.Data.AmountBTC,
			Price:         trade.Data.Amount,
			ID:            fmt.Sprintf("%v", trade.Data.Advertisement.ID),
			OrderDate:     t.Unix(),
			Fee:           trade.Data.FeeBTC,
			OrderSide:     side,
			BaseCurrency:  symbol.BTC,
			QuoteCurrency: trade.Data.Currency,
		})
	}

	l.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	l.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LocalBitcoins) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var allTrades []DashBoardInfo
	resp, err := l.GetDashboardCancelledTrades()
	if err != nil {
		return nil, err
	}
	for _, trade := range resp {
		allTrades = append(allTrades, trade)
	}

	resp, err = l.GetDashboardClosedTrades()
	if err != nil {
		return nil, err
	}
	for _, trade := range resp {
		allTrades = append(allTrades, trade)
	}

	resp, err = l.GetDashboardReleasedTrades()
	if err != nil {
		return nil, err
	}
	for _, trade := range resp {
		allTrades = append(allTrades, trade)
	}

	var orders []exchange.OrderDetail
	for _, trade := range resp {
		t, err := time.Parse(time.RFC3339, trade.Data.CreatedAt)
		if err != nil {
			log.Errorf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name, "GetActiveOrders", trade.Data.Advertisement.ID, trade.Data.CreatedAt)
		}

		side := ""
		if trade.Data.IsBuying {
			side = string(exchange.BuyOrderSide)
		} else if trade.Data.IsSelling {
			side = string(exchange.SellOrderSide)
		}

		status := ""
		if trade.Data.ReleasedAt != "" && trade.Data.ReleasedAt != "null" {
			status = "Released"
		} else if trade.Data.CanceledAt != "" && trade.Data.CanceledAt != "null" {
			status = "Cancelled"
		} else if trade.Data.ClosedAt != "" && trade.Data.ClosedAt != "null" {
			status = "Closed"
		}

		orders = append(orders, exchange.OrderDetail{
			Amount:        trade.Data.AmountBTC,
			Price:         trade.Data.Amount,
			ID:            fmt.Sprintf("%v", trade.Data.Advertisement.ID),
			OrderDate:     t.Unix(),
			Fee:           trade.Data.FeeBTC,
			OrderSide:     side,
			Status:        status,
			BaseCurrency:  symbol.BTC,
			QuoteCurrency: trade.Data.Currency,
		})
	}

	l.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	l.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}
