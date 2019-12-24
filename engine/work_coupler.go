package engine

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
	"golang.org/x/time/rate"
)

const (
	fetchTicker = iota
	updateTicker
	fetchOrderbook
	updateOrderbook
	getAccountInfo
	getExchangeHistory
	getFeeByType
	getFundingHistory
	submitOrder
	modifyOrder
	cancelOrder
	cancelAllOrders
	getOrderDetail
	getDepositAddress
	getOrderHistory
	withdrawal
)

var (
	errFunctionalityNotSupported = errors.New("function not supported")
	errFunctionalityNotFound     = errors.New("function not found")
)

// TODO:
// ClientPermissions: Determine if calling system can execute, set API KEYS
// DATABASE: Insert intention into audit table
// TRADE Heuristics: trade/account security
// DATABASE: Insert event in audit table

// Execute implements the command interface for the exchange coupler
func (f *FetchTicker) Execute() {
	f.Price, f.Error = f.FetchTicker(f.Pair, f.Asset)
}

// GetReservation implements the command interface and returns a reservation
func (f *FetchTicker) GetReservation() *rate.Reservation {
	return f.Reservation
}

// // IsCancelled checks if update is cancelled
// func (f *FetchTicker) IsCancelled() bool {
// 	return atomic.LoadInt32(f.CancelMe) == 1
// }

// FetchTicker initiates a call to an exchange through the priority job queue
func (i Exchange) FetchTicker(p currency.Pair, a asset.Item, withCancel chan int) (*ticker.Price, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, fetchTicker)
	if err != nil {
		return nil, err
	}

	t := &FetchTicker{
		Pair:         p,
		Asset:        a,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().UnauthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(t, low, withCancel)
	if err != nil {
		return t.Price, err
	}

	return t.Price, t.Error
}

// Execute implements the command interface for the exchange coupler
func (f *UpdateTicker) Execute() {
	f.Price, f.Error = f.UpdateTicker(f.Pair, f.Asset)
}

// GetReservation implements the command interface and returns a reservation
func (f *UpdateTicker) GetReservation() *rate.Reservation {
	return f.Reservation
}

// // IsCancelled checks if update is cancelled
// func (f *UpdateTicker) IsCancelled() bool {
// 	b := atomic.LoadInt32(f.CancelMe) == 1
// 	fmt.Println("checking ticker is cancelled", b)
// 	return b
// }

// UpdateTicker initiates a call to an exchange through the priority job queue
func (i Exchange) UpdateTicker(p currency.Pair, a asset.Item, withCancel chan int) (*ticker.Price, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, updateTicker)
	if err != nil {
		return nil, err
	}

	t := &UpdateTicker{
		Pair:         p,
		Asset:        a,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().UnauthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(t, low, withCancel)
	if err != nil {
		return t.Price, err
	}

	return t.Price, t.Error
}

// Execute implements the command interface for the exchange coupler
func (o *FetchOrderbook) Execute() {
	o.Orderbook, o.Error = o.FetchOrderbook(o.Pair, o.Asset)
}

// GetReservation implements the command interface and returns a reservation
func (o *FetchOrderbook) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *FetchOrderbook) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// FetchOrderbook initiates a call to an exchange through the priority job queue
func (i Exchange) FetchOrderbook(p currency.Pair, a asset.Item, withCancel chan int) (*orderbook.Base, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, fetchOrderbook)
	if err != nil {
		return nil, err
	}

	o := &FetchOrderbook{Pair: p,
		Asset:        a,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().UnauthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, medium, withCancel)
	if err != nil {
		return o.Orderbook, err
	}

	return o.Orderbook, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *UpdateOrderbook) Execute() {
	o.Orderbook, o.Error = o.UpdateOrderbook(o.Pair, o.Asset)
}

// GetReservation implements the command interface and returns a reservation
func (o *UpdateOrderbook) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *UpdateOrderbook) IsCancelled() bool {
// 	b := atomic.LoadInt32(o.CancelMe) == 1
// 	fmt.Println("checking orderbook is cancelled", b)
// 	return b
// }

// UpdateOrderbook initiates a call to an exchange through the priority job
// queue
func (i Exchange) UpdateOrderbook(p currency.Pair, a asset.Item, withCancel chan int) (*orderbook.Base, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, updateOrderbook)
	if err != nil {
		return nil, err
	}

	o := &UpdateOrderbook{
		Pair:         p,
		Asset:        a,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().UnauthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, medium, withCancel)
	if err != nil {
		return o.Orderbook, err
	}

	return o.Orderbook, o.Error
}

// Execute implements the command interface for the exchange coupler
func (g *GetAccountInfo) Execute() {
	g.AccountInfo, g.Error = g.GetAccountInfo()
}

// GetReservation implements the command interface and returns a reservation
func (g *GetAccountInfo) GetReservation() *rate.Reservation {
	return g.Reservation
}

// // IsCancelled checks if update is cancelled
// func (g *GetAccountInfo) IsCancelled() bool {
// 	return atomic.LoadInt32(g.CancelMe) == 1
// }

// GetAccountInfo initiates a call to an exchange through the priority job queue
func (i Exchange) GetAccountInfo(withCancel chan int) (exchange.AccountInfo, error) {
	if i.e == nil {
		return exchange.AccountInfo{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getAccountInfo)
	if err != nil {
		return exchange.AccountInfo{}, err
	}

	acc := &GetAccountInfo{
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(acc, high, withCancel)
	if err != nil {
		return acc.AccountInfo, err
	}

	return acc.AccountInfo, acc.Error
}

// Execute implements the command interface for the exchange coupler
func (e *GetExchangeHistory) Execute() {
	e.Response, e.Error = e.GetExchangeHistory(e.Request.Pair, e.Request.Asset)
}

// GetReservation implements the command interface and returns a reservation
func (e *GetExchangeHistory) GetReservation() *rate.Reservation {
	return e.Reservation
}

// // IsCancelled checks if update is cancelled
// func (e *GetExchangeHistory) IsCancelled() bool {
// 	return atomic.LoadInt32(e.CancelMe) == 1
// }

// GetExchangeHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetExchangeHistory(r *exchange.TradeHistoryRequest, withCancel chan int) ([]exchange.TradeHistory, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getExchangeHistory)
	if err != nil {
		return nil, err
	}

	h := &GetExchangeHistory{
		Request:      r,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().UnauthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(h, low, withCancel)
	if err != nil {
		return nil, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (f *GetFeeByType) Execute() {
	f.Response, f.Error = f.GetFeeByType(f.Request)
}

// GetReservation implements the command interface and returns a reservation
func (f *GetFeeByType) GetReservation() *rate.Reservation {
	return f.Reservation
}

// // IsCancelled checks if update is cancelled
// func (f *GetFeeByType) IsCancelled() bool {
// 	return atomic.LoadInt32(f.CancelMe) == 1
// }

// GetFeeByType initiates a call to an exchange through the priority job queue
func (i Exchange) GetFeeByType(feeBuilder *exchange.FeeBuilder, withCancel chan int) (float64, error) {
	if i.e == nil {
		return 0, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getFeeByType)
	if err != nil {
		return 0, err
	}

	f := &GetFeeByType{
		Request:      feeBuilder,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(f, high, withCancel)
	if err != nil {
		return 0, err
	}

	return f.Response, f.Error
}

// Execute implements the command interface for the exchange coupler
func (f *GetFundingHistory) Execute() {
	f.Response, f.Error = f.GetFundingHistory()
}

// GetReservation implements the command interface and returns a reservation
func (f *GetFundingHistory) GetReservation() *rate.Reservation {
	return f.Reservation
}

// // IsCancelled checks if update is cancelled
// func (f *GetFundingHistory) IsCancelled() bool {
// 	return atomic.LoadInt32(f.CancelMe) == 1
// }

// GetFundingHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetFundingHistory(withCancel chan int) ([]exchange.FundHistory, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getFundingHistory)
	if err != nil {
		return nil, err
	}

	f := &GetFundingHistory{IBotExchange: i.e}

	err = i.wm.ExecuteJob(f, medium, withCancel)
	if err != nil {
		return nil, err
	}

	return f.Response, f.Error
}

// Execute implements the command interface for the exchange coupler
func (o *SubmitOrder) Execute() {
	o.Response, o.Error = o.SubmitOrder(o.Request)
}

// GetReservation implements the command interface and returns a reservation
func (o *SubmitOrder) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *SubmitOrder) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// SubmitOrder initiates a call to an exchange through the priority job queue
func (i Exchange) SubmitOrder(s *order.Submit, withCancel chan int) (order.SubmitResponse, error) {
	if i.e == nil {
		return order.SubmitResponse{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, submitOrder)
	if err != nil {
		return order.SubmitResponse{}, err
	}

	o := &SubmitOrder{
		Request:      s,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, extreme, withCancel)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *ModifyOrder) Execute() {
	o.Response, o.Error = o.ModifyOrder(o.Request)
}

// GetReservation implements the command interface and returns a reservation
func (o *ModifyOrder) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *ModifyOrder) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// ModifyOrder initiates a call to an exchange through the priority job queue
func (i Exchange) ModifyOrder(action *order.Modify, withCancel chan int) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, modifyOrder)
	if err != nil {
		return "", err
	}

	o := &ModifyOrder{
		Request:      action,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, extreme, withCancel)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *CancelOrder) Execute() {
	o.Error = o.CancelOrder(o.Request)
}

// GetReservation implements the command interface and returns a reservation
func (o *CancelOrder) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *CancelOrder) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// CancelOrder initiates a call to an exchange through the priority job queue
func (i Exchange) CancelOrder(cancel *order.Cancel, withCancel chan int) error {
	if i.e == nil {
		return errExchangNotFound
	}

	err := i.checkFunctionality(i.e, cancelOrder)
	if err != nil {
		return err
	}

	o := &CancelOrder{
		Request:      cancel,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, extreme, withCancel)
	if err != nil {
		return err
	}

	return o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *CancelAllOrders) Execute() {
	o.Response, o.Error = o.CancelAllOrders(o.Request)
}

// GetReservation implements the command interface and returns a reservation
func (o *CancelAllOrders) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *CancelAllOrders) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// CancelAllOrders initiates a call to an exchange through the priority job
// queue
func (i Exchange) CancelAllOrders(cancel *order.Cancel, withCancel chan int) (order.CancelAllResponse, error) {
	if i.e == nil {
		return order.CancelAllResponse{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, cancelAllOrders)
	if err != nil {
		return order.CancelAllResponse{}, err
	}

	o := &CancelAllOrders{
		Request:      cancel,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, extreme, withCancel)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (o *GetOrderInfo) Execute() {
	o.Response, o.Error = o.GetOrderInfo(o.Request)
}

// GetReservation implements the command interface and returns a reservation
func (o *GetOrderInfo) GetReservation() *rate.Reservation {
	return o.Reservation
}

// // IsCancelled checks if update is cancelled
// func (o *GetOrderInfo) IsCancelled() bool {
// 	return atomic.LoadInt32(o.CancelMe) == 1
// }

// GetOrderInfo initiates a call to an exchange through the priority job queue
func (i Exchange) GetOrderInfo(orderID string, withCancel chan int) (order.Detail, error) {
	if i.e == nil {
		return order.Detail{}, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return order.Detail{}, err
	}

	o := &GetOrderInfo{
		Request:      orderID,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(o, high, withCancel)
	if err != nil {
		return o.Response, err
	}

	return o.Response, o.Error
}

// Execute implements the command interface for the exchange coupler
func (a *GetDepositAddress) Execute() {
	a.Response, a.Error = a.GetDepositAddress(a.Crypto, a.AccountID)
}

// GetReservation implements the command interface and returns a reservation
func (a *GetDepositAddress) GetReservation() *rate.Reservation {
	return a.Reservation
}

// // IsCancelled checks if update is cancelled
// func (a *GetDepositAddress) IsCancelled() bool {
// 	return atomic.LoadInt32(a.CancelMe) == 1
// }

// GetDepositAddress initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetDepositAddress(crypto currency.Code, accountID string, withCancel chan int) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getDepositAddress)
	if err != nil {
		return "", err
	}

	a := &GetDepositAddress{
		Crypto:       crypto,
		AccountID:    accountID,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(a, medium, withCancel)
	if err != nil {
		return a.Response, err
	}

	return a.Response, a.Error
}

// Execute implements the command interface for the exchange coupler
func (h *GetOrderHistory) Execute() {
	h.Response, h.Error = h.GetOrderHistory(h.Request)
}

// GetReservation implements the command interface and returns a reservation
func (h *GetOrderHistory) GetReservation() *rate.Reservation {
	return h.Reservation
}

// // IsCancelled checks if update is cancelled
// func (h *GetOrderHistory) IsCancelled() bool {
// 	return atomic.LoadInt32(h.CancelMe) == 1
// }

// GetOrderHistory initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetOrderHistory(req *order.GetOrdersRequest, withCancel chan int) ([]order.Detail, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return nil, err
	}

	h := &GetOrderHistory{
		Request:      req,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(h, medium, withCancel)
	if err != nil {
		return h.Response, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (h *GetActiveOrders) Execute() {
	h.Response, h.Error = h.GetActiveOrders(h.Request)
}

// GetReservation implements the command interface and returns a reservation
func (h *GetActiveOrders) GetReservation() *rate.Reservation {
	return h.Reservation
}

// // IsCancelled checks if update is cancelled
// func (h *GetActiveOrders) IsCancelled() bool {
// 	return atomic.LoadInt32(h.CancelMe) == 1
// }

// GetActiveOrders initiates a call to an exchange through the priority job
// queue
func (i Exchange) GetActiveOrders(req *order.GetOrdersRequest, withCancel chan int) ([]order.Detail, error) {
	if i.e == nil {
		return nil, errExchangNotFound
	}

	err := i.checkFunctionality(i.e, getOrderDetail)
	if err != nil {
		return nil, err
	}

	h := &GetActiveOrders{
		Request:      req,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(h, medium, withCancel)
	if err != nil {
		return h.Response, err
	}

	return h.Response, h.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawCryptocurrencyFunds) Execute() {
	w.Response, w.Error = w.WithdrawCryptocurrencyFunds(w.Request)
}

// GetReservation implements the command interface and returns a reservation
func (w *WithdrawCryptocurrencyFunds) GetReservation() *rate.Reservation {
	return w.Reservation
}

// // IsCancelled checks if update is cancelled
// func (w *WithdrawCryptocurrencyFunds) IsCancelled() bool {
// 	return atomic.LoadInt32(w.CancelMe) == 1
// }

// WithdrawCryptocurrencyFunds initiates a call to an exchange through the
// priority job queue
func (i Exchange) WithdrawCryptocurrencyFunds(req *withdraw.CryptoRequest, withCancel chan int) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdrawal)
	if err != nil {
		return "", err
	}

	w := &WithdrawCryptocurrencyFunds{
		Request:      req,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(w, high, withCancel)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawFiatFunds) Execute() {
	w.Response, w.Error = w.WithdrawFiatFunds(w.Request)
}

// GetReservation implements the command interface and returns a reservation
func (w *WithdrawFiatFunds) GetReservation() *rate.Reservation {
	return w.Reservation
}

// // IsCancelled checks if update is cancelled
// func (w *WithdrawFiatFunds) IsCancelled() bool {
// 	return atomic.LoadInt32(w.CancelMe) == 1
// }

// WithdrawFiatFunds initiates a call to an exchange through the priority job
// queue
func (i Exchange) WithdrawFiatFunds(req *withdraw.FiatRequest, withCancel chan int) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdrawal)
	if err != nil {
		return "", err
	}

	w := &WithdrawFiatFunds{
		Request:      req,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(w, high, withCancel)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

// Execute implements the command interface for the exchange coupler
func (w *WithdrawFiatFundsToInternationalBank) Execute() {
	w.Response, w.Error = w.WithdrawFiatFundsToInternationalBank(w.Request)
}

// GetReservation implements the command interface and returns a reservation
func (w *WithdrawFiatFundsToInternationalBank) GetReservation() *rate.Reservation {
	return w.Reservation
}

// // IsCancelled checks if update is cancelled
// func (w *WithdrawFiatFundsToInternationalBank) IsCancelled() bool {
// 	return atomic.LoadInt32(w.CancelMe) == 1
// }

// WithdrawFiatFundsToInternationalBank initiates a call to an exchange through
// the priority job queue
func (i Exchange) WithdrawFiatFundsToInternationalBank(req *withdraw.FiatRequest, withCancel chan int) (string, error) {
	if i.e == nil {
		return "", errExchangNotFound
	}

	err := i.checkFunctionality(i.e, withdrawal)
	if err != nil {
		return "", err
	}

	w := &WithdrawFiatFundsToInternationalBank{
		Request:      req,
		IBotExchange: i.e,
		Reservation:  i.e.GetBase().AuthLimit.Reserve(),
		CancelMe:     withCancel,
	}

	err = i.wm.ExecuteJob(w, high, withCancel)
	if err != nil {
		return w.Response, err
	}

	return w.Response, w.Error
}

func (i Exchange) checkFunctionality(e exchange.IBotExchange, function int) error {
	// b := e.GetBase()
	// fmt.Println(b.API)

	switch function {
	case fetchTicker, updateTicker:
		// if !b.Features.REST.TickerFetching.IsEnabled() {
		// 	return errFunctionalityNotSupported
		// }
	case fetchOrderbook, updateOrderbook:
		// if !b.Features.REST.OrderbookFetching.IsEnabled() {
		// 	return errFunctionalityNotSupported
		// }
		// case getAccountInfo:
		// 	if !b.Features.REST.AccountInfo.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case getExchangeHistory:
		// 	if !b.Features.REST.ExchangeTradeHistory.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case getFeeByType:
		// 	// need to fix this
		// 	if !b.Features.REST.TradeFee.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case getFundingHistory:
		// 	// fix this
		// 	return errFunctionalityNotSupported

		// case submitOrder:
		// 	if !b.Features.REST.SubmitOrder.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case modifyOrder:
		// 	if !b.Features.REST.ModifyOrder.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case cancelOrder:
		// 	if !b.Features.REST.CancelOrder.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case cancelAllOrders:
		// 	if !b.Features.REST.CancelOrders.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case getOrderDetail:
		// 	if !b.Features.REST.GetOrder.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case getOrderHistory:
		// 	if !b.Features.REST.GetOrders.IsEnabled() {
		// 		return errFunctionalityNotSupported
		// 	}
		// case withdraw:
		// 	if *b.Features.REST.Withdraw == 0 {
		// 		return errFunctionalityNotSupported
		// 	}
		// default:
		// 	return errFunctionalityNotFound
	}
	return nil
}
