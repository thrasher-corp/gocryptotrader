package htx

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	htxAPIURL         = "https://api.huobi.pro"
	htxURL            = "https://api.hbdm.com"
	htxFuturesURL     = htxURL
	htxAPIVersion     = "1"
	htxAPIVersion2    = "2"
	tradeBaseURL      = "https://www.htx.com/"
	tradeSpot         = "trade/"
	tradeFutures      = "futures/linear_swap/exchange#contract_code="
	tradeCoinMargined = "futures/swap/exchange/#symbol="

	htxStatusSuccess = "success"
	htxStatusError   = "error"
)

// Exchange implements exchange.IBotExchange and contains additional API methods for interacting with HTX.
type Exchange struct {
	exchange.Base
	AccountID                string
	futureContractCodesMutex sync.RWMutex
	futureContractCodes      map[string]currency.Code
}

func getSignatureHost(endpoint string) (string, error) {
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errInvalidEndpoint, err)
	}
	if parsedEndpoint.Host == "" {
		return "", fmt.Errorf("%w: missing host", errInvalidEndpoint)
	}
	return parsedEndpoint.Host, nil
}

// GetMarginRates gets margin rates
func (e *Exchange) GetMarginRates(ctx context.Context, symbol currency.Pair) (*MarginRatesResponse, error) {
	resp := new(MarginRatesResponse)
	vals := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		vals.Set("symbol", symbolValue)
	}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/margin/loan-info", vals, nil, resp, false)
	return resp, err
}

// GetSpotKline returns spot candles for the requested symbol, interval and optional size.
func (e *Exchange) GetSpotKline(ctx context.Context, arg *KlinesRequest) ([]KlineItem, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)
	vals.Set("period", arg.Period)

	if arg.Size != 0 {
		vals.Set("size", strconv.FormatUint(arg.Size, 10))
	}

	result := new(SpotKlineResponse)

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/history/kline", vals), result)
	return result.Candles, err
}

// Get24HourMarketSummary returns the rolling 24-hour summary for a spot symbol
func (e *Exchange) Get24HourMarketSummary(ctx context.Context, symbol currency.Pair) (*MarketSummary24HourResponse, error) {
	result := new(MarketSummary24HourResponse)
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return result, err
	}
	params.Set("symbol", symbolValue)
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/detail", params), result)
	return result, err
}

// GetBatchCoinMarginSwapContracts returns the tickers for coin margined swap contracts
func (e *Exchange) GetBatchCoinMarginSwapContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	result := new(FuturesBatchTickersResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/v2/swap-ex/market/detail/batch_merged", result)
	return result.Tickers, err
}

// GetBatchLinearSwapContracts  returns the tickers for linear swap contracts
func (e *Exchange) GetBatchLinearSwapContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	result := new(FuturesBatchTickersResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/v2/linear-swap-ex/market/detail/batch_merged", result)
	return result.Tickers, err
}

// GetBatchFuturesContracts returns the tickers for futures contracts
func (e *Exchange) GetBatchFuturesContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	result := new(FuturesBatchTickersResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/v2/market/detail/batch_merged", result)
	return result.Tickers, err
}

// GetTickers returns the ticker for the specified symbol
func (e *Exchange) GetTickers(ctx context.Context) (*TickersResponse, error) {
	result := new(TickersResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, "/market/tickers", result)
	return result, err
}

// GetMarketDetailMerged returns the ticker for the specified symbol
func (e *Exchange) GetMarketDetailMerged(ctx context.Context, symbol currency.Pair) (*DetailMerged, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	result := new(MarketDetailMergedResponse)

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/detail/merged", vals), result)
	// the tick carries no time of its own on this endpoint, only the envelope does
	result.Tick.Timestamp = result.Timestamp
	return &result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (e *Exchange) GetDepth(ctx context.Context, obd *OrderBookRequest) (*Orderbook, error) {
	symbolValue, err := e.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals := url.Values{}
	vals.Set("symbol", symbolValue)

	if obd.Type != OrderBookDepthTypeNone {
		vals.Set("type", string(obd.Type))
	}

	result := new(OrderBookResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/depth", vals), result); err != nil {
		return nil, err
	}
	return &result.Depth, nil
}

// GetTrades returns the trades for the specified symbol
func (e *Exchange) GetTrades(ctx context.Context, symbol currency.Pair) ([]Trade, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	result := new(TradesResponse)

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/trade", vals), result)
	return result.Tick.Trades, err
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (e *Exchange) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair) (float64, error) {
	list, err := e.GetTradeHistory(ctx, symbol, 1)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, errEmptyResult
	}

	return list[0].Trades[0].Price, nil
}

// GetTradeHistory returns the trades for the specified symbol
func (e *Exchange) GetTradeHistory(ctx context.Context, symbol currency.Pair, size uint64) ([]TradeHistory, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	if size > 0 {
		vals.Set("size", strconv.FormatUint(size, 10))
	}

	result := new(TradeHistoryResponse)

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/history/trade", vals), result)
	return result.TradeHistory, err
}

// GetMarketDetail returns the ticker for the specified symbol
func (e *Exchange) GetMarketDetail(ctx context.Context, symbol currency.Pair) (*Detail, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	result := new(MarketDetailResponse)

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/market/detail", vals), result)
	return &result.Tick, err
}

// GetSymbols returns an array of symbols supported by HTX
func (e *Exchange) GetSymbols(ctx context.Context) ([]Symbol, error) {
	result := new(SymbolsResponse)

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, "/v1/common/symbols", result)
	return result.Symbols, err
}

// GetCurrencies returns a list of currencies supported by HTX
func (e *Exchange) GetCurrencies(ctx context.Context) ([]currency.Code, error) {
	result := new(CurrenciesResponse)

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, "/v1/common/currencys", result)
	return result.Currencies, err
}

// GetCurrenciesIncludingChains returns currency and chain data
func (e *Exchange) GetCurrenciesIncludingChains(ctx context.Context, curr currency.Code) ([]CurrencyChains, error) {
	resp := new(CurrenciesIncludingChainsResponse)

	vals := url.Values{}
	if !curr.IsEmpty() {
		vals.Set("currency", curr.Lower().String())
	}
	path := common.EncodeURLValues("/v2/reference/currencies", vals)
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, resp); err != nil {
		return nil, err
	}
	return resp.Currencies, nil
}

// GetCurrentServerTime returns the HTX server time
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	result := new(CurrentServerTimeResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, "/v"+htxAPIVersion+"//common/timestamp", result)
	return result.Timestamp.Time(), err
}

// GetAccounts returns the HTX user accounts
func (e *Exchange) GetAccounts(ctx context.Context) ([]Account, error) {
	result := new(AccountsResponse)
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/account/accounts", url.Values{}, nil, result, false)
	return result.Accounts, err
}

// GetAccountBalance returns the users HTX account balance
func (e *Exchange) GetAccountBalance(ctx context.Context, accountID string) ([]AccountBalanceDetail, error) {
	result := new(AccountBalanceResponse)
	endpoint := "/account/accounts/" + accountID + "/balance"
	v := url.Values{}
	v.Set("account-id", accountID)
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, v, nil, result, false)
	return result.Balance.CurrencyBalances, err
}

// GetAggregatedBalance returns the balances of all the sub-account aggregated.
func (e *Exchange) GetAggregatedBalance(ctx context.Context) ([]AggregatedBalance, error) {
	result := new(AggregatedBalanceResponse)
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet,
		"/subuser/aggregate-balance",
		nil,
		nil,
		result,
		false,
	)
	return result.AggregatedBalances, err
}

// SpotNewOrder submits an order to HTX
func (e *Exchange) SpotNewOrder(ctx context.Context, arg *SpotOrderRequest) (uint64, error) {
	symbolValue, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return 0, err
	}

	data := &spotOrderSubmissionRequest{
		AccountID:     arg.AccountID,
		ClientOrderID: arg.ClientOrderID,
		Amount:        strconv.FormatFloat(arg.Amount, 'f', -1, 64),
		Symbol:        symbolValue,
		Type:          string(arg.Type),
	}

	// Only set price if order type is not equal to buy-market or sell-market
	if arg.Type != SpotOrderTypeBuyMarket && arg.Type != SpotOrderTypeSellMarket {
		data.Price = strconv.FormatFloat(arg.Price, 'f', -1, 64)
	}

	if arg.Source != "" {
		data.Source = arg.Source
	}

	result := new(SpotOrderIDResponse)
	err = e.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		"/order/orders/place",
		nil,
		data,
		result,
		false,
	)
	return result.OrderID, err
}

// CancelExistingOrder cancels an order on HTX
func (e *Exchange) CancelExistingOrder(ctx context.Context, orderID uint64) (uint64, error) {
	resp := new(SpotOrderIDResponse)
	endpoint := "/order/orders/" + strconv.FormatUint(orderID, 10) + "/submitcancel"
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, url.Values{}, nil, resp, false)
	return resp.OrderID, err
}

// CancelOrderBatch cancels a batch of orders
func (e *Exchange) CancelOrderBatch(ctx context.Context, orderIDs, clientOrderIDs []string) (*CancelOrderBatch, error) {
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(orderIDs) != 0 && len(clientOrderIDs) != 0 {
		return nil, errMixedOrderIdentifiers
	}
	if len(orderIDs) > 50 || len(clientOrderIDs) > 50 {
		return nil, fmt.Errorf("%w: maximum 50 orders", errCancelOrderLimitExceeded)
	}
	resp := new(CancelOrderBatchResponse)
	data := &cancelOrderBatchRequest{OrderIDs: orderIDs, ClientOrderIDs: clientOrderIDs}
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/order/orders/batchcancel", nil, data, resp, false); err != nil {
		return nil, err
	}
	if resp.Cancellations == nil {
		return nil, errEmptyResult
	}
	return resp.Cancellations, nil
}

// CancelOpenOrdersBatch cancels open spot orders for an account and symbol in one request.
func (e *Exchange) CancelOpenOrdersBatch(ctx context.Context, accountID string, symbol currency.Pair) (*CancelOpenOrdersBatchResponse, error) {
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("account-id", accountID)
	result := new(CancelOpenOrdersBatchResponse)

	data := &cancelOpenOrdersBatchRequest{
		AccountID: accountID,
		Symbol:    symbolValue,
	}

	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/order/orders/batchCancelOpenOrders", url.Values{}, data, result, false); err != nil {
		return result, err
	}
	return result, validateCancelOpenOrdersBatchResponse(result)
}

func validateCancelOpenOrdersBatchResponse(result *CancelOpenOrdersBatchResponse) error {
	if result == nil {
		return common.ErrNilPointer
	}
	if result.Cancellations.FailedCount > 0 {
		return fmt.Errorf("%w: %d orders failed to cancel", errOrderCancellationFailed, result.Cancellations.FailedCount)
	}
	if result.Cancellations.NextID != -1 {
		return fmt.Errorf("%w: cancellation incomplete; next open order %d", errOrderCancellationFailed, result.Cancellations.NextID)
	}
	return nil
}

// GetOrder returns order information for the specified order
func (e *Exchange) GetOrder(ctx context.Context, orderID uint64) (*OrderInfo, error) {
	resp := new(OrderResponse)
	endpoint := "/order/orders/" + strconv.FormatUint(orderID, 10)
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		endpoint,
		nil,
		nil,
		resp,
		false)
	return &resp.Order, err
}

// GetOrderMatchResults returns matched order info for the specified order
func (e *Exchange) GetOrderMatchResults(ctx context.Context, orderID uint64) ([]OrderMatchInfo, error) {
	resp := new(OrderMatchResultsResponse)
	endpoint := "/order/orders/" + strconv.FormatUint(orderID, 10) + "/matchresults"
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, url.Values{}, nil, resp, false)
	return resp.Orders, err
}

// GetOrders returns a list of orders
func (e *Exchange) GetOrders(ctx context.Context, symbol currency.Pair, orderTypes, start, end, states, from, direct, size string) ([]OrderInfo, error) {
	resp := new(OrdersResponse)

	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)
	vals.Set("states", states)

	if orderTypes != "" {
		vals.Set("types", orderTypes)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/order/orders", vals, nil, resp, false)
	return resp.Orders, err
}

// GetOpenOrders returns a list of orders
func (e *Exchange) GetOpenOrders(ctx context.Context, symbol currency.Pair, accountID, side string, size uint64) ([]OrderInfo, error) {
	resp := new(OrdersResponse)

	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)
	vals.Set("account-id", accountID)
	if side != "" {
		vals.Set("side", side)
	}
	vals.Set("size", strconv.FormatUint(size, 10))

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/order/openOrders", vals, nil, resp, false)
	return resp.Orders, err
}

// GetOrdersMatch returns a list of matched orders
func (e *Exchange) GetOrdersMatch(ctx context.Context, symbol currency.Pair, orderTypes, start, end, from, direct, size string) ([]OrderMatchInfo, error) {
	resp := new(OrderMatchResultsResponse)

	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	if orderTypes != "" {
		vals.Set("types", orderTypes)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/order/matchresults", vals, nil, resp, false)
	return resp.Orders, err
}

// MarginTransfer transfers assets into or out of the margin account
func (e *Exchange) MarginTransfer(ctx context.Context, symbol currency.Pair, ccy string, amount float64, in bool) (uint64, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return 0, err
	}
	data := &marginAmountRequest{
		Symbol:   symbolValue,
		Currency: ccy,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	path := "/dw/transfer-in/margin"
	if !in {
		path = "/dw/transfer-out/margin"
	}

	resp := new(MarginTransferResponse)
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, data, resp, false)
	return resp.TransferID, err
}

// MarginOrder submits a margin order application
func (e *Exchange) MarginOrder(ctx context.Context, symbol currency.Pair, ccy string, amount float64) (uint64, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return 0, err
	}
	data := &marginAmountRequest{
		Symbol:   symbolValue,
		Currency: ccy,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := new(MarginOrderResponse)
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/margin/orders", nil, data, resp, false)
	return resp.MarginOrderID, err
}

// MarginRepayment repays a margin amount for a margin ID
func (e *Exchange) MarginRepayment(ctx context.Context, orderID uint64, amount float64) (uint64, error) {
	data := &marginRepaymentRequest{
		Amount: strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := new(MarginOrderResponse)

	endpoint := "/margin/orders/" + strconv.FormatUint(orderID, 10) + "/repay"
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, nil, data, resp, false)
	return resp.MarginOrderID, err
}

// GetMarginLoanOrders returns the margin loan orders
func (e *Exchange) GetMarginLoanOrders(ctx context.Context, symbol currency.Pair, ccy, start, end, states, from, direct, size string) ([]MarginOrder, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)
	vals.Set("currency", ccy)

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if states != "" {
		vals.Set("states", states)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	resp := new(MarginLoanOrdersResponse)
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/margin/loan-orders", vals, nil, resp, false)
	return resp.MarginLoanOrders, err
}

// GetMarginAccountBalance returns the margin account balances
func (e *Exchange) GetMarginAccountBalance(ctx context.Context, symbol currency.Pair) ([]MarginAccountBalance, error) {
	resp := new(MarginAccountBalanceResponse)
	vals := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp.Balances, err
		}
		vals.Set("symbol", symbolValue)
	}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/margin/accounts/balance", vals, nil, resp, false)
	return resp.Balances, err
}

// Withdraw withdraws the desired amount and currency
func (e *Exchange) Withdraw(ctx context.Context, c currency.Code, address, addrTag, chain string, amount, fee float64) (uint64, error) {
	if c.IsEmpty() || address == "" || amount <= 0 {
		return 0, errWithdrawDetailsUnset
	}

	resp := new(WithdrawResponse)

	data := &withdrawalRequest{
		Address:  address,
		Currency: c.Lower().String(),
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	if fee > 0 {
		data.Fee = strconv.FormatFloat(fee, 'f', -1, 64)
	}

	if addrTag != "" {
		data.AddrTag = addrTag
	}

	if chain != "" {
		data.Chain = strings.ToLower(chain)
	}

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/dw/withdraw/api/create", nil, data, resp, false)
	return resp.WithdrawID, err
}

// CancelWithdraw cancels a withdraw request
func (e *Exchange) CancelWithdraw(ctx context.Context, withdrawID uint64) (uint64, error) {
	resp := new(WithdrawResponse)
	vals := url.Values{}
	vals.Set("withdraw-id", strconv.FormatUint(withdrawID, 10))

	endpoint := "/dw/withdraw-virtual/" + strconv.FormatUint(withdrawID, 10) + "/cancel"
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, vals, nil, resp, false)
	return resp.WithdrawID, err
}

// QueryDepositAddress returns the deposit address for a specified currency
func (e *Exchange) QueryDepositAddress(ctx context.Context, cryptocurrency currency.Code) ([]DepositAddress, error) {
	resp := new(DepositAddressResponse)

	vals := url.Values{}
	vals.Set("currency", cryptocurrency.Lower().String())

	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/account/deposit/address", vals, nil, resp, true); err != nil {
		return nil, err
	}
	if len(resp.DepositAddress) == 0 {
		return nil, errDepositAddressMissing
	}
	return resp.DepositAddress, nil
}

// QueryWithdrawQuotas returns the users cryptocurrency withdraw quotas
func (e *Exchange) QueryWithdrawQuotas(ctx context.Context, cryptocurrency string) (*WithdrawQuota, error) {
	resp := new(WithdrawQuotasResponse)

	vals := url.Values{}
	vals.Set("currency", cryptocurrency)

	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/account/withdraw/quota", vals, nil, resp, true); err != nil {
		return nil, err
	}
	return &resp.WithdrawQuota, nil
}

// SearchForExistedWithdrawsAndDeposits returns withdrawal and deposit data
func (e *Exchange) SearchForExistedWithdrawsAndDeposits(ctx context.Context, c currency.Code, transferType, direction string, fromID, limit uint64) (*WithdrawalHistoryResponse, error) {
	resp := new(WithdrawalHistoryResponse)
	vals := url.Values{}
	vals.Set("type", transferType)
	if !c.IsEmpty() {
		vals.Set("currency", c.Lower().String())
	}
	if direction != "" {
		vals.Set("direct", direction)
	}
	if fromID > 0 {
		vals.Set("from", strconv.FormatUint(fromID, 10))
	}
	if limit > 0 {
		vals.Set("size", strconv.FormatUint(limit, 10))
	}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/query/deposit-withdraw", vals, nil, resp, false)
	return resp, err
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	response := &httpResponse{
		Result:   result,
		Version5: strings.HasPrefix(path, "/v5/") || strings.Contains(path, "/v3/"),
	}

	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 response,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, getRateLimitID(ep, path, http.MethodGet, false), func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}

	if !response.Received && result != nil {
		return errExpectedResponseBody
	}
	return nil
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the HTX API
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, data, result any, isVersion2API bool) error {
	var err error
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	signatureHost, err := getSignatureHost(ePoint)
	if err != nil {
		return err
	}
	if values == nil {
		values = url.Values{}
	}

	response := &httpResponse{Result: result, Authenticated: true, Version2: isVersion2API}
	newRequest := func() (*request.Item, error) {
		values.Del("Signature")
		values.Set("AccessKeyId", creds.Key)
		values.Set("SignatureMethod", "HmacSHA256")
		values.Set("SignatureVersion", "2")
		values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))

		signatureEndpoint := endpoint
		if isVersion2API {
			signatureEndpoint = "/v" + htxAPIVersion2 + signatureEndpoint
		} else {
			signatureEndpoint = "/v" + htxAPIVersion + signatureEndpoint
		}

		payload := fmt.Sprintf("%s\n%s\n%s\n%s",
			method, signatureHost, signatureEndpoint, values.Encode())

		headers := make(map[string]string)

		if method == http.MethodGet {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		} else {
			headers["Content-Type"] = "application/json"
		}

		hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		values.Set("Signature", base64.StdEncoding.EncodeToString(hmac))

		var body []byte
		if data != nil {
			body, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}

		return &request.Item{
			Method:                 method,
			Path:                   ePoint + common.EncodeURLValues(signatureEndpoint, values),
			Headers:                headers,
			Body:                   bytes.NewReader(body),
			Result:                 response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}

	if err := e.SendPayload(ctx, getRateLimitID(ep, endpoint, method, true), newRequest, request.AuthenticatedRequest); err != nil {
		return err
	}

	if !response.Received && result != nil {
		return common.AppendError(errExpectedResponseBody, request.ErrAuthRequestFailed)
	}
	return nil
}

func unmarshalResponse(response []byte, result any) error {
	trimmed := bytes.TrimSpace(response)
	if len(trimmed) == 0 {
		if result == nil {
			return nil
		}
		return errExpectedResponseBody
	}
	if result == nil {
		return fmt.Errorf("%w: received %d bytes", errUnexpectedResponseBody, len(trimmed))
	}
	return json.Unmarshal(trimmed, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	if feeBuilder.FeeType == exchange.OfflineTradeFee || feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		fee = calculateTradingFee(feeBuilder.Pair, feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(c currency.Pair, price, amount float64) float64 {
	if c.IsCryptoFiatPair() {
		return 0.001 * price * amount
	}
	return 0.002 * price * amount
}
