package huobi

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	huobiAPIURL       = "https://api.huobi.pro"
	huobiURL          = "https://api.hbdm.com"
	huobiFuturesURL   = huobiURL
	huobiAPIVersion   = "1"
	huobiAPIVersion2  = "2"
	tradeBaseURL      = "https://www.htx.com/"
	tradeSpot         = "trade/"
	tradeFutures      = "futures/linear_swap/exchange#contract_code="
	tradeCoinMargined = "futures/swap/exchange/#symbol="

	// Spot endpoints
	huobiMarketHistoryKline           = "/market/history/kline"
	huobiMarketDetail                 = "/market/detail"
	huobiMarketDetailMerged           = "/market/detail/merged"
	huobi24HrMarketSummary            = "/market/detail?"
	huobiMarketDepth                  = "/market/depth"
	huobiMarketTrade                  = "/market/trade"
	huobiMarketTickers                = "/market/tickers"
	huobiMarketTradeHistory           = "/market/history/trade"
	huobiSymbols                      = "/v1/common/symbols"
	huobiCurrencies                   = "/v1/common/currencys"
	huobiTimestamp                    = "/common/timestamp"
	huobiAccounts                     = "/account/accounts"
	huobiAccountBalance               = "/account/accounts/%s/balance"
	huobiAccountDepositAddress        = "/account/deposit/address"
	huobiAccountWithdrawQuota         = "/account/withdraw/quota"
	huobiAccountQueryWithdrawAddress  = "/account/withdraw/"
	huobiAggregatedBalance            = "/subuser/aggregate-balance"
	huobiOrderPlace                   = "/order/orders/place"
	huobiOrderCancel                  = "/order/orders/%s/submitcancel"
	huobiOrderCancelBatch             = "/order/orders/batchcancel"
	huobiBatchCancelOpenOrders        = "/order/orders/batchCancelOpenOrders"
	huobiGetOrder                     = "/order/orders/getClientOrder"
	huobiGetOrderMatch                = "/order/orders/%s/matchresults"
	huobiGetOrders                    = "/order/orders"
	huobiGetOpenOrders                = "/order/openOrders"
	huobiGetOrdersMatch               = "/orders/matchresults"
	huobiMarginTransferIn             = "/dw/transfer-in/margin"
	huobiMarginTransferOut            = "/dw/transfer-out/margin"
	huobiMarginOrders                 = "/margin/orders"
	huobiMarginRepay                  = "/margin/orders/%s/repay"
	huobiMarginLoanOrders             = "/margin/loan-orders"
	huobiMarginAccountBalance         = "/margin/accounts/balance"
	huobiWithdrawCreate               = "/dw/withdraw/api/create"
	huobiWithdrawCancel               = "/dw/withdraw-virtual/%s/cancel"
	huobiStatusError                  = "error"
	huobiMarginRates                  = "/margin/loan-info"
	huobiCurrenciesReference          = "/v2/reference/currencies"
	huobiWithdrawHistory              = "/query/deposit-withdraw"
	huobiBatchCoinMarginSwapContracts = "/v2/swap-ex/market/detail/batch_merged"
	huobiBatchLinearSwapContracts     = "/v2/linear-swap-ex/market/detail/batch_merged"
	huobiBatchContracts               = "/v2/market/detail/batch_merged"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Huobi
type Exchange struct {
	exchange.Base
	AccountID                string
	futureContractCodesMutex sync.RWMutex
	futureContractCodes      map[string]currency.Code
}

// GetMarginRates gets margin rates
func (e *Exchange) GetMarginRates(ctx context.Context, symbol currency.Pair) (MarginRatesData, error) {
	var resp MarginRatesData
	vals := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		vals.Set("symbol", symbolValue)
	}
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiMarginRates, vals, nil, &resp, false)
}

// GetSpotKline returns kline data
// KlinesRequestParams contains symbol currency.Pair, period and size
func (e *Exchange) GetSpotKline(ctx context.Context, arg KlinesRequestParams) ([]KlineItem, error) {
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

	type response struct {
		Response
		Data []KlineItem `json:"data"`
	}

	var result response

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketHistoryKline, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// Get24HrMarketSummary returns 24hr market summary for a given market symbol
func (e *Exchange) Get24HrMarketSummary(ctx context.Context, symbol currency.Pair) (MarketSummary24Hr, error) {
	var result MarketSummary24Hr
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return result, err
	}
	params.Set("symbol", symbolValue)
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, huobi24HrMarketSummary+params.Encode(), &result)
}

// GetBatchCoinMarginSwapContracts returns the tickers for coin margined swap contracts
func (e *Exchange) GetBatchCoinMarginSwapContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	var result struct {
		Data []FuturesBatchTicker `json:"ticks"`
	}
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, huobiBatchCoinMarginSwapContracts, &result)
	return result.Data, err
}

// GetBatchLinearSwapContracts  returns the tickers for linear swap contracts
func (e *Exchange) GetBatchLinearSwapContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	var result struct {
		Data []FuturesBatchTicker `json:"ticks"`
	}
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, huobiBatchLinearSwapContracts, &result)
	return result.Data, err
}

// GetBatchFuturesContracts returns the tickers for futures contracts
func (e *Exchange) GetBatchFuturesContracts(ctx context.Context) ([]FuturesBatchTicker, error) {
	var result struct {
		Data []FuturesBatchTicker `json:"ticks"`
	}
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, huobiBatchContracts, &result)
	return result.Data, err
}

// GetTickers returns the ticker for the specified symbol
func (e *Exchange) GetTickers(ctx context.Context) (Tickers, error) {
	var result Tickers
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, huobiMarketTickers, &result)
}

// GetMarketDetailMerged returns the ticker for the specified symbol
func (e *Exchange) GetMarketDetailMerged(ctx context.Context, symbol currency.Pair) (DetailMerged, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return DetailMerged{}, err
	}
	vals.Set("symbol", symbolValue)

	type response struct {
		Response
		Tick DetailMerged `json:"tick"`
	}

	var result response

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketDetailMerged, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (e *Exchange) GetDepth(ctx context.Context, obd *OrderBookDataRequestParams) (*Orderbook, error) {
	symbolValue, err := e.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals := url.Values{}
	vals.Set("symbol", symbolValue)

	if obd.Type != OrderBookDataRequestParamsTypeNone {
		vals.Set("type", string(obd.Type))
	}

	type response struct {
		Response
		Depth Orderbook `json:"tick"`
	}

	var result response
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketDepth, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return &result.Depth, err
}

// GetTrades returns the trades for the specified symbol
func (e *Exchange) GetTrades(ctx context.Context, symbol currency.Pair) ([]Trade, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	type response struct {
		Response
		Tick struct {
			Data []Trade `json:"data"`
		} `json:"tick"`
	}

	var result response

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketTrade, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Tick.Data, err
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
		return 0, errors.New("the length of the list is 0")
	}

	return list[0].Trades[0].Price, nil
}

// GetTradeHistory returns the trades for the specified symbol
func (e *Exchange) GetTradeHistory(ctx context.Context, symbol currency.Pair, size int64) ([]TradeHistory, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)

	if size > 0 {
		vals.Set("size", strconv.FormatInt(size, 10))
	}

	type response struct {
		Response
		TradeHistory []TradeHistory `json:"data"`
	}

	var result response

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketTradeHistory, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.TradeHistory, err
}

// GetMarketDetail returns the ticker for the specified symbol
func (e *Exchange) GetMarketDetail(ctx context.Context, symbol currency.Pair) (Detail, error) {
	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return Detail{}, err
	}
	vals.Set("symbol", symbolValue)

	type response struct {
		Response
		Tick Detail `json:"tick"`
	}

	var result response

	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(huobiMarketDetail, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetSymbols returns an array of symbols supported by Huobi
func (e *Exchange) GetSymbols(ctx context.Context) ([]Symbol, error) {
	type response struct {
		Response
		Symbols []Symbol `json:"data"`
	}

	var result response

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, huobiSymbols, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Symbols, err
}

// GetCurrencies returns a list of currencies supported by Huobi
func (e *Exchange) GetCurrencies(ctx context.Context) ([]string, error) {
	type response struct {
		Response
		Currencies []string `json:"data"`
	}

	var result response

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, huobiCurrencies, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Currencies, err
}

// GetCurrenciesIncludingChains returns currency and chain data
func (e *Exchange) GetCurrenciesIncludingChains(ctx context.Context, curr currency.Code) ([]CurrenciesChainData, error) {
	resp := struct {
		Data []CurrenciesChainData `json:"data"`
	}{}

	vals := url.Values{}
	if !curr.IsEmpty() {
		vals.Set("currency", curr.Lower().String())
	}
	path := common.EncodeURLValues(huobiCurrenciesReference, vals)
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, path, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetCurrentServerTime returns the Huobi server time
func (e *Exchange) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	var result struct {
		Response
		Timestamp types.Time `json:"data"`
	}
	err := e.SendHTTPRequest(ctx, exchange.RestSpot, "/v"+huobiAPIVersion+"/"+huobiTimestamp, &result)
	if result.ErrorMessage != "" {
		return time.Time{}, errors.New(result.ErrorMessage)
	}
	return result.Timestamp.Time(), err
}

// GetAccounts returns the Huobi user accounts
func (e *Exchange) GetAccounts(ctx context.Context) ([]Account, error) {
	result := struct {
		Accounts []Account `json:"data"`
	}{}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiAccounts, url.Values{}, nil, &result, false)
	return result.Accounts, err
}

// GetAccountBalance returns the users Huobi account balance
func (e *Exchange) GetAccountBalance(ctx context.Context, accountID string) ([]AccountBalanceDetail, error) {
	result := struct {
		AccountBalanceData AccountBalance `json:"data"`
	}{}
	endpoint := fmt.Sprintf(huobiAccountBalance, accountID)
	v := url.Values{}
	v.Set("account-id", accountID)
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, v, nil, &result, false)
	return result.AccountBalanceData.AccountBalanceDetails, err
}

// GetAggregatedBalance returns the balances of all the sub-account aggregated.
func (e *Exchange) GetAggregatedBalance(ctx context.Context) ([]AggregatedBalance, error) {
	result := struct {
		AggregatedBalances []AggregatedBalance `json:"data"`
	}{}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet,
		huobiAggregatedBalance,
		nil,
		nil,
		&result,
		false,
	)
	return result.AggregatedBalances, err
}

// SpotNewOrder submits an order to Huobi
func (e *Exchange) SpotNewOrder(ctx context.Context, arg *SpotNewOrderRequestParams) (int64, error) {
	symbolValue, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return 0, err
	}

	data := struct {
		AccountID int    `json:"account-id,string"`
		Amount    string `json:"amount"`
		Price     string `json:"price"`
		Source    string `json:"source"`
		Symbol    string `json:"symbol"`
		Type      string `json:"type"`
	}{
		AccountID: arg.AccountID,
		Amount:    strconv.FormatFloat(arg.Amount, 'f', -1, 64),
		Symbol:    symbolValue,
		Type:      string(arg.Type),
	}

	// Only set price if order type is not equal to buy-market or sell-market
	if arg.Type != SpotNewOrderRequestTypeBuyMarket && arg.Type != SpotNewOrderRequestTypeSellMarket {
		data.Price = strconv.FormatFloat(arg.Price, 'f', -1, 64)
	}

	if arg.Source != "" {
		data.Source = arg.Source
	}

	result := struct {
		OrderID int64 `json:"data,string"`
	}{}
	err = e.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost,
		huobiOrderPlace,
		nil,
		data,
		&result,
		false,
	)
	return result.OrderID, err
}

// CancelExistingOrder cancels an order on Huobi
func (e *Exchange) CancelExistingOrder(ctx context.Context, orderID int64) (int64, error) {
	resp := struct {
		OrderID int64 `json:"data,string"`
	}{}
	endpoint := fmt.Sprintf(huobiOrderCancel, strconv.FormatInt(orderID, 10))
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, url.Values{}, nil, &resp, false)
	return resp.OrderID, err
}

// CancelOrderBatch cancels a batch of orders
func (e *Exchange) CancelOrderBatch(ctx context.Context, orderIDs, clientOrderIDs []string) (*CancelOrderBatch, error) {
	resp := struct {
		Response
		Data *CancelOrderBatch `json:"data"`
	}{}
	data := struct {
		ClientOrderIDs []string `json:"client-order-ids"`
		OrderIDs       []string `json:"order-ids"`
	}{
		ClientOrderIDs: clientOrderIDs,
		OrderIDs:       orderIDs,
	}
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, huobiOrderCancelBatch, nil, data, &resp, false)
}

// CancelOpenOrdersBatch cancels a batch of orders -- to-do
func (e *Exchange) CancelOpenOrdersBatch(ctx context.Context, accountID string, symbol currency.Pair) (CancelOpenOrdersBatch, error) {
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return CancelOpenOrdersBatch{}, err
	}
	params.Set("account-id", accountID)
	var result CancelOpenOrdersBatch

	data := struct {
		AccountID string `json:"account-id"`
		Symbol    string `json:"symbol"`
	}{
		AccountID: accountID,
		Symbol:    symbolValue,
	}

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, huobiBatchCancelOpenOrders, url.Values{}, data, &result, false)
	if result.Data.FailedCount > 0 {
		return result, fmt.Errorf("there were %v failed order cancellations", result.Data.FailedCount)
	}

	return result, err
}

// GetOrder returns order information for the specified order
func (e *Exchange) GetOrder(ctx context.Context, orderID int64) (OrderInfo, error) {
	resp := struct {
		Order OrderInfo `json:"data"`
	}{}
	urlVal := url.Values{}
	urlVal.Set("clientOrderId", strconv.FormatInt(orderID, 10))
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet,
		huobiGetOrder,
		urlVal,
		nil,
		&resp,
		false)
	return resp.Order, err
}

// GetOrderMatchResults returns matched order info for the specified order
func (e *Exchange) GetOrderMatchResults(ctx context.Context, orderID int64) ([]OrderMatchInfo, error) {
	resp := struct {
		Orders []OrderMatchInfo `json:"data"`
	}{}
	endpoint := fmt.Sprintf(huobiGetOrderMatch, strconv.FormatInt(orderID, 10))
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, endpoint, url.Values{}, nil, &resp, false)
	return resp.Orders, err
}

// GetOrders returns a list of orders
func (e *Exchange) GetOrders(ctx context.Context, symbol currency.Pair, orderTypes, start, end, states, from, direct, size string) ([]OrderInfo, error) {
	resp := struct {
		Orders []OrderInfo `json:"data"`
	}{}

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

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiGetOrders, vals, nil, &resp, false)
	return resp.Orders, err
}

// GetOpenOrders returns a list of orders
func (e *Exchange) GetOpenOrders(ctx context.Context, symbol currency.Pair, accountID, side string, size int64) ([]OrderInfo, error) {
	resp := struct {
		Orders []OrderInfo `json:"data"`
	}{}

	vals := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	vals.Set("symbol", symbolValue)
	vals.Set("accountID", accountID)
	if side != "" {
		vals.Set("side", side)
	}
	vals.Set("size", strconv.FormatInt(size, 10))

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiGetOpenOrders, vals, nil, &resp, false)
	return resp.Orders, err
}

// GetOrdersMatch returns a list of matched orders
func (e *Exchange) GetOrdersMatch(ctx context.Context, symbol currency.Pair, orderTypes, start, end, from, direct, size string) ([]OrderMatchInfo, error) {
	resp := struct {
		Orders []OrderMatchInfo `json:"data"`
	}{}

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

	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiGetOrdersMatch, vals, nil, &resp, false)
	return resp.Orders, err
}

// MarginTransfer transfers assets into or out of the margin account
func (e *Exchange) MarginTransfer(ctx context.Context, symbol currency.Pair, ccy string, amount float64, in bool) (int64, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return 0, err
	}
	data := struct {
		Symbol   string `json:"symbol"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	}{
		Symbol:   symbolValue,
		Currency: ccy,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	path := huobiMarginTransferIn
	if !in {
		path = huobiMarginTransferOut
	}

	resp := struct {
		TransferID int64 `json:"data"`
	}{}
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, data, &resp, false)
	return resp.TransferID, err
}

// MarginOrder submits a margin order application
func (e *Exchange) MarginOrder(ctx context.Context, symbol currency.Pair, ccy string, amount float64) (int64, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return 0, err
	}
	data := struct {
		Symbol   string `json:"symbol"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	}{
		Symbol:   symbolValue,
		Currency: ccy,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := struct {
		MarginOrderID int64 `json:"data"`
	}{}
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, huobiMarginOrders, nil, data, &resp, false)
	return resp.MarginOrderID, err
}

// MarginRepayment repays a margin amount for a margin ID
func (e *Exchange) MarginRepayment(ctx context.Context, orderID int64, amount float64) (int64, error) {
	data := struct {
		Amount string `json:"amount"`
	}{
		Amount: strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := struct {
		MarginOrderID int64 `json:"data"`
	}{}

	endpoint := fmt.Sprintf(huobiMarginRepay, strconv.FormatInt(orderID, 10))
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, nil, data, &resp, false)
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

	resp := struct {
		MarginLoanOrders []MarginOrder `json:"data"`
	}{}
	err = e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiMarginLoanOrders, vals, nil, &resp, false)
	return resp.MarginLoanOrders, err
}

// GetMarginAccountBalance returns the margin account balances
func (e *Exchange) GetMarginAccountBalance(ctx context.Context, symbol currency.Pair) ([]MarginAccountBalance, error) {
	resp := struct {
		Balances []MarginAccountBalance `json:"data"`
	}{}
	vals := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp.Balances, err
		}
		vals.Set("symbol", symbolValue)
	}
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiMarginAccountBalance, vals, nil, &resp, false)
	return resp.Balances, err
}

// Withdraw withdraws the desired amount and currency
func (e *Exchange) Withdraw(ctx context.Context, c currency.Code, address, addrTag, chain string, amount, fee float64) (int64, error) {
	if c.IsEmpty() || address == "" || amount <= 0 {
		return 0, errors.New("currency, address and amount must be set")
	}

	resp := struct {
		WithdrawID int64 `json:"data"`
	}{}

	data := struct {
		Address  string `json:"address"`
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
		Fee      string `json:"fee,omitempty"`
		Chain    string `json:"chain,omitempty"`
		AddrTag  string `json:"addr-tag,omitempty"`
	}{
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

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, huobiWithdrawCreate, nil, data, &resp, false)
	return resp.WithdrawID, err
}

// CancelWithdraw cancels a withdraw request
func (e *Exchange) CancelWithdraw(ctx context.Context, withdrawID int64) (int64, error) {
	resp := struct {
		WithdrawID int64 `json:"data"`
	}{}
	vals := url.Values{}
	vals.Set("withdraw-id", strconv.FormatInt(withdrawID, 10))

	endpoint := fmt.Sprintf(huobiWithdrawCancel, strconv.FormatInt(withdrawID, 10))
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, endpoint, vals, nil, &resp, false)
	return resp.WithdrawID, err
}

// QueryDepositAddress returns the deposit address for a specified currency
func (e *Exchange) QueryDepositAddress(ctx context.Context, cryptocurrency currency.Code) ([]DepositAddress, error) {
	resp := struct {
		DepositAddress []DepositAddress `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("currency", cryptocurrency.Lower().String())

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiAccountDepositAddress, vals, nil, &resp, true)
	if err != nil {
		return nil, err
	}
	if len(resp.DepositAddress) == 0 {
		return nil, errors.New("deposit address data isn't populated")
	}
	return resp.DepositAddress, nil
}

// QueryWithdrawQuotas returns the users cryptocurrency withdraw quotas
func (e *Exchange) QueryWithdrawQuotas(ctx context.Context, cryptocurrency string) (WithdrawQuota, error) {
	resp := struct {
		WithdrawQuota WithdrawQuota `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("currency", cryptocurrency)

	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiAccountWithdrawQuota, vals, nil, &resp, true)
	if err != nil {
		return WithdrawQuota{}, err
	}
	return resp.WithdrawQuota, nil
}

// SearchForExistedWithdrawsAndDeposits returns withdrawal and deposit data
func (e *Exchange) SearchForExistedWithdrawsAndDeposits(ctx context.Context, c currency.Code, transferType, direction string, fromID, limit int64) (WithdrawalHistory, error) {
	var resp WithdrawalHistory
	vals := url.Values{}
	vals.Set("type", transferType)
	if !c.IsEmpty() {
		vals.Set("currency", c.Lower().String())
	}
	if direction != "" {
		vals.Set("direction", direction)
	}
	if fromID > 0 {
		vals.Set("from", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		vals.Set("size", strconv.FormatInt(limit, 10))
	}
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, huobiWithdrawHistory, vals, nil, &resp, false)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var tempResp json.RawMessage

	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpoint + path,
		Result:                 &tempResp,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}

	var errCap errorCapture
	if err := json.Unmarshal(tempResp, &errCap); err == nil {
		if errCap.ErrMsgType1 != "" {
			return fmt.Errorf("error code: %v error message: %s", errCap.CodeType1,
				errors.New(errCap.ErrMsgType1))
		}
		if errCap.ErrMsgType2 != "" {
			return fmt.Errorf("error code: %v error message: %s", errCap.CodeType2,
				errors.New(errCap.ErrMsgType2))
		}
	}
	return json.Unmarshal(tempResp, result)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
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
	if values == nil {
		values = url.Values{}
	}

	interim := json.RawMessage{}
	newRequest := func() (*request.Item, error) {
		values.Set("AccessKeyId", creds.Key)
		values.Set("SignatureMethod", "HmacSHA256")
		values.Set("SignatureVersion", "2")
		values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))

		if isVersion2API {
			endpoint = "/v" + huobiAPIVersion2 + endpoint
		} else {
			endpoint = "/v" + huobiAPIVersion + endpoint
		}

		payload := fmt.Sprintf("%s\napi.huobi.pro\n%s\n%s",
			method, endpoint, values.Encode())

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
			Path:                   ePoint + common.EncodeURLValues(endpoint, values),
			Headers:                headers,
			Body:                   bytes.NewReader(body),
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}

	err = e.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	if isVersion2API {
		var errCap ResponseV2
		if err = json.Unmarshal(interim, &errCap); err == nil {
			if errCap.Code != 200 && errCap.Message != "" {
				return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.Code, errCap.Message)
			}
		}
	} else {
		var errCap Response
		if err = json.Unmarshal(interim, &errCap); err == nil {
			if errCap.Status == huobiStatusError && errCap.ErrorMessage != "" {
				return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.ErrorCode, errCap.ErrorMessage)
			}
		}
	}
	err = json.Unmarshal(interim, result)
	if err != nil {
		return common.AppendError(err, request.ErrAuthRequestFailed)
	}
	return nil
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
