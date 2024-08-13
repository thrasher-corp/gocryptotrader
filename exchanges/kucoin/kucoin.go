package kucoin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Kucoin is the overarching type across this package
type Kucoin struct {
	exchange.Base
	obm *orderbookManager
}

var locker sync.Mutex

const (
	kucoinAPIURL        = "https://api.kucoin.com/api"
	kucoinAPIKeyVersion = "2"
	tradeBaseURL        = "https://www.kucoin.com/"
	tradeSpot           = "trade/"
	tradeMargin         = "margin/"
	tradeFutures        = "futures/"

	// Public endpoints
	kucoinGetSymbols             = "/v2/symbols"
	kucoinGetTicker              = "/v1/market/orderbook/level1"
	kucoinGetAllTickers          = "/v1/market/allTickers"
	kucoinGet24hrStats           = "/v1/market/stats"
	kucoinGetMarketList          = "/v1/markets"
	kucoinGetPartOrderbook20     = "/v1/market/orderbook/level2_20"
	kucoinGetPartOrderbook100    = "/v1/market/orderbook/level2_100"
	kucoinGetTradeHistory        = "/v1/market/histories"
	kucoinGetKlines              = "/v1/market/candles"
	kucoinGetCurrencies          = "/v1/currencies"
	kucoinGetCurrency            = "/v2/currencies/"
	kucoinGetFiatPrice           = "/v1/prices"
	kucoinGetMarkPrice           = "/v1/mark-price/%s/current"
	kucoinGetMarginConfiguration = "/v1/margin/config"
	kucoinGetServerTime          = "/v1/timestamp"
	kucoinGetServiceStatus       = "/v1/status"

	// Authenticated endpoints
	kucoinGetOrderbook         = "/v3/market/orderbook/level2"
	kucoinGetMarginAccount     = "/v1/margin/account"
	kucoinGetMarginRiskLimit   = "/v1/risk/limit/strategy"
	kucoinBorrowOrder          = "/v1/margin/borrow"
	kucoinGetOutstandingRecord = "/v1/margin/borrow/outstanding"
	kucoinGetRepaidRecord      = "/v1/margin/borrow/repaid"
	kucoinOneClickRepayment    = "/v1/margin/repay/all"
	kucoinRepaySingleOrder     = "/v1/margin/repay/single"
	kucoinLendOrder            = "/v1/margin/lend"
	kucoinSetAutoLend          = "/v1/margin/toggle-auto-lend"
	kucoinGetActiveOrder       = "/v1/margin/lend/active"
	kucoinGetLendHistory       = "/v1/margin/lend/done"
	kucoinGetUnsettleLendOrder = "/v1/margin/lend/trade/unsettled"
	kucoinGetSettleLendOrder   = "/v1/margin/lend/trade/settled"
	kucoinGetAccountLendRecord = "/v1/margin/lend/assets"
	kucoinGetLendingMarketData = "/v1/margin/market"
	kucoinGetMarginTradeData   = "/v1/margin/trade/last"

	kucoinGetIsolatedMarginPairConfig            = "/v1/isolated/symbols"
	kucoinGetIsolatedMarginAccountInfo           = "/v1/isolated/accounts"
	kucoinGetSingleIsolatedMarginAccountInfo     = "/v1/isolated/account/"
	kucoinInitiateIsolatedMarginBorrowing        = "/v1/isolated/borrow"
	kucoinGetIsolatedOutstandingRepaymentRecords = "/v1/isolated/borrow/outstanding"
	kucoinGetIsolatedMarginRepaymentRecords      = "/v1/isolated/borrow/repaid"
	kucoinInitiateIsolatedMarginQuickRepayment   = "/v1/isolated/repay/all"
	kucoinInitiateIsolatedMarginSingleRepayment  = "/v1/isolated/repay/single"

	kucoinPostOrder        = "/v1/orders"
	kucoinPostMarginOrder  = "/v1/margin/order"
	kucoinPostBulkOrder    = "/v1/orders/multi"
	kucoinOrderByID        = "/v1/orders/"             // used by CancelSingleOrder and GetOrderByID
	kucoinOrderByClientOID = "/v1/order/client-order/" // used by CancelOrderByClientOID and GetOrderByClientOID
	kucoinOrders           = "/v1/orders"              // used by CancelAllOpenOrders and GetOrders
	kucoinGetRecentOrders  = "/v1/limit/orders"

	kucoinGetFills       = "/v1/fills"
	kucoinGetRecentFills = "/v1/limit/fills"

	kucoinStopOrder                 = "/v1/stop-order"
	kucoinStopOrderByID             = "/v1/stop-order/"
	kucoinCancelAllStopOrder        = "/v1/stop-order/cancel"
	kucoinGetStopOrderByClientID    = "/v1/stop-order/queryOrderByClientOid"
	kucoinCancelStopOrderByClientID = "/v1/stop-order/cancelOrderByClientOid"

	// user info endpoints
	kucoinSubUserCreated = "/v2/sub/user/created"
	kucoinSubUser        = "/v2/sub/user"

	kucoinSubAccountSpotAPIs             = "/v1/sub/api-key"
	kucoinUpdateModifySubAccountSpotAPIs = "/v1/sub/api-key/update"

	// account
	kucoinAccount                        = "/v1/accounts"
	kucoinGetAccount                     = "/v1/accounts/"
	kucoinGetAccountLedgers              = "/v1/accounts/ledgers"
	kucoinUserInfo                       = "/v2/user-info"
	kucoinGetSubAccountBalance           = "/v1/sub-accounts/"
	kucoinGetAggregatedSubAccountBalance = "/v1/sub-accounts"
	kucoinGetTransferableBalance         = "/v1/accounts/transferable"
	kucoinTransferMainToSubAccount       = "/v2/accounts/sub-transfer"
	kucoinInnerTransfer                  = "/v2/accounts/inner-transfer"

	// deposit
	kucoinGetDepositAddressesV2    = "/v2/deposit-addresses"
	kucoinGetDepositAddressV1      = "/v1/deposit-addresses"
	kucoinGetDepositList           = "/v1/deposits"
	kucoinGetHistoricalDepositList = "/v1/hist-deposits"

	// withdrawal
	kucoinWithdrawal                  = "/v1/withdrawals"
	kucoinGetHistoricalWithdrawalList = "/v1/hist-withdrawals"
	kucoinGetWithdrawalQuotas         = "/v1/withdrawals/quotas"
	kucoinCancelWithdrawal            = "/v1/withdrawals/"

	kucoinBasicFee   = "/v1/base-fee"
	kucoinTradingFee = "/v1/trade-fees"
)

// GetSymbols gets pairs details on the exchange
// For market details see endpoint: https://www.kucoin.com/docs/rest/spot-trading/market-data/get-market-list
func (ku *Kucoin) GetSymbols(ctx context.Context, market string) ([]SymbolInfo, error) {
	params := url.Values{}
	if market != "" {
		params.Set("market", market)
	}
	var resp []SymbolInfo
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetSymbols, params), &resp)
}

// GetTicker gets pair ticker information
func (ku *Kucoin) GetTicker(ctx context.Context, pair string) (*Ticker, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp *Ticker
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetTicker, params), &resp)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTickers gets all trading pair ticker information including 24h volume
func (ku *Kucoin) GetTickers(ctx context.Context) (*TickersResponse, error) {
	var resp *TickersResponse
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetAllTickers, &resp)
}

// Get24hrStats get the statistics of the specified pair in the last 24 hours
func (ku *Kucoin) Get24hrStats(ctx context.Context, pair string) (*Stats24hrs, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp *Stats24hrs
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGet24hrStats, params), &resp)
}

// GetMarketList get the transaction currency for the entire trading market
func (ku *Kucoin) GetMarketList(ctx context.Context) ([]string, error) {
	var resp []string
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetMarketList, &resp)
}

func processOB(ob [][2]string) ([]orderbook.Tranche, error) {
	o := make([]orderbook.Tranche, len(ob))
	for x := range ob {
		amount, err := strconv.ParseFloat(ob[x][1], 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(ob[x][0], 64)
		if err != nil {
			return nil, err
		}
		o[x] = orderbook.Tranche{
			Price:  price,
			Amount: amount,
		}
	}
	return o, nil
}

func constructOrderbook(o *orderbookResponse) (*Orderbook, error) {
	var (
		s   Orderbook
		err error
	)
	s.Bids, err = processOB(o.Bids)
	if err != nil {
		return nil, err
	}
	s.Asks, err = processOB(o.Asks)
	if err != nil {
		return nil, err
	}
	s.Time = o.Time.Time()
	if o.Sequence != "" {
		s.Sequence, err = strconv.ParseInt(o.Sequence, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return &s, err
}

// GetPartOrderbook20 gets orderbook for a specified pair with depth 20
func (ku *Kucoin) GetPartOrderbook20(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetPartOrderbook20, params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetPartOrderbook100 gets orderbook for a specified pair with depth 100
func (ku *Kucoin) GetPartOrderbook100(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetPartOrderbook100, params), &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetOrderbook gets full orderbook for a specified pair
func (ku *Kucoin) GetOrderbook(ctx context.Context, pair string) (*Orderbook, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var o *orderbookResponse
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveFullOrderbookEPL, http.MethodGet, common.EncodeURLValues(kucoinGetOrderbook, params), nil, &o)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(o)
}

// GetTradeHistory gets trade history of the specified pair
func (ku *Kucoin) GetTradeHistory(ctx context.Context, pair string) ([]Trade, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	var resp []Trade
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetTradeHistory, params), &resp)
}

// GetKlines gets kline of the specified pair
func (ku *Kucoin) GetKlines(ctx context.Context, pair, period string, start, end time.Time) ([]Kline, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair)
	if period == "" {
		return nil, errors.New("period can not be empty")
	}
	if !common.StringDataContains(validPeriods, period) {
		return nil, errors.New("invalid period")
	}
	params.Set("type", period)
	if !start.IsZero() {
		params.Set("startAt", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		params.Set("endAt", strconv.FormatInt(end.Unix(), 10))
	}
	var resp [][7]string
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetKlines, params), &resp)
	if err != nil {
		return nil, err
	}
	klines := make([]Kline, len(resp))
	for i := range resp {
		t, err := strconv.ParseInt(resp[i][0], 10, 64)
		if err != nil {
			return nil, err
		}
		klines[i].StartTime = time.Unix(t, 0)
		klines[i].Open, err = strconv.ParseFloat(resp[i][1], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Close, err = strconv.ParseFloat(resp[i][2], 64)
		if err != nil {
			return nil, err
		}
		klines[i].High, err = strconv.ParseFloat(resp[i][3], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Low, err = strconv.ParseFloat(resp[i][4], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Volume, err = strconv.ParseFloat(resp[i][5], 64)
		if err != nil {
			return nil, err
		}
		klines[i].Amount, err = strconv.ParseFloat(resp[i][6], 64)
		if err != nil {
			return nil, err
		}
	}
	return klines, nil
}

// GetCurrencies gets list of currencies
func (ku *Kucoin) GetCurrencies(ctx context.Context) ([]Currency, error) {
	var resp []Currency
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetCurrencies, &resp)
}

// GetCurrencyDetail gets currency detail using currency code and chain information.
func (ku *Kucoin) GetCurrencyDetail(ctx context.Context, ccy, chain string) (*CurrencyDetail, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *CurrencyDetail
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetCurrency+strings.ToUpper(ccy), params), &resp)
}

// GetFiatPrice gets fiat prices of currencies, default base currency is USD
func (ku *Kucoin) GetFiatPrice(ctx context.Context, base, currencies string) (map[string]string, error) {
	params := url.Values{}
	if base != "" {
		params.Set("base", base)
	}
	if currencies != "" {
		params.Set("currencies", currencies)
	}
	var resp map[string]string
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, common.EncodeURLValues(kucoinGetFiatPrice, params), &resp)
}

// GetMarkPrice gets index price of the specified pair
func (ku *Kucoin) GetMarkPrice(ctx context.Context, pair string) (*MarkPrice, error) {
	if pair == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *MarkPrice
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, fmt.Sprintf(kucoinGetMarkPrice, pair), &resp)
}

// GetMarginConfiguration gets configure info of the margin
func (ku *Kucoin) GetMarginConfiguration(ctx context.Context) (*MarginConfiguration, error) {
	var resp *MarginConfiguration
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetMarginConfiguration, &resp)
}

// GetMarginAccount gets configure info of the margin
func (ku *Kucoin) GetMarginAccount(ctx context.Context) (*MarginAccounts, error) {
	var resp *MarginAccounts
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetMarginAccount, nil, &resp)
}

// GetMarginRiskLimit gets cross/isolated margin risk limit, default model is cross margin
func (ku *Kucoin) GetMarginRiskLimit(ctx context.Context, marginModel string) ([]MarginRiskLimit, error) {
	params := url.Values{}
	if marginModel != "" {
		params.Set("marginModel", marginModel)
	}
	var resp []MarginRiskLimit
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveMarginAccountEPL, http.MethodGet, common.EncodeURLValues(kucoinGetMarginRiskLimit, params), nil, &resp)
}

// PostBorrowOrder used to post borrow order
func (ku *Kucoin) PostBorrowOrder(ctx context.Context, ccy, orderType, term string, size, maxRate float64) (*PostBorrowOrderResp, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if orderType == "" {
		return nil, errors.New("orderType can not be empty")
	}
	if size == 0 {
		return nil, errors.New("size can not be zero")
	}
	params := make(map[string]interface{})
	params["currency"] = strings.ToUpper(ccy)
	params["type"] = orderType
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	if maxRate != 0 {
		params["maxRate"] = strconv.FormatFloat(maxRate, 'f', -1, 64)
	}
	if term != "" {
		params["term"] = term
	}
	var resp *PostBorrowOrderResp
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinBorrowOrder, params, &resp)
}

// GetBorrowOrder gets borrow order information
func (ku *Kucoin) GetBorrowOrder(ctx context.Context, orderID string) (*BorrowOrder, error) {
	if orderID == "" {
		return nil, errors.New("empty orderID")
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *BorrowOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinBorrowOrder, params), nil, &resp)
}

// GetOutstandingRecord gets outstanding record information
func (ku *Kucoin) GetOutstandingRecord(ctx context.Context, ccy string) (*OutstandingRecordResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *OutstandingRecordResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetOutstandingRecord, params), nil, &resp)
}

// GetRepaidRecord gets repaid record information
func (ku *Kucoin) GetRepaidRecord(ctx context.Context, ccy string) (*RepaidRecordsResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *RepaidRecordsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetRepaidRecord, params), nil, &resp)
}

// OneClickRepayment used to complete repayment in single go
func (ku *Kucoin) OneClickRepayment(ctx context.Context, ccy, sequence string, size float64) error {
	if ccy == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if sequence == "" {
		return errors.New("sequence can not be empty")
	}
	params["sequence"] = sequence
	if size == 0 {
		return errors.New("size can not be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinOneClickRepayment, params, &struct{}{})
}

// SingleOrderRepayment used to repay single order
func (ku *Kucoin) SingleOrderRepayment(ctx context.Context, ccy, tradeID string, size float64) error {
	if ccy == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if tradeID == "" {
		return errors.New("tradeId can not be empty")
	}
	params["tradeId"] = tradeID
	if size == 0 {
		return errors.New("size can not be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinRepaySingleOrder, params, &struct{}{})
}

// PostLendOrder used to create lend order
func (ku *Kucoin) PostLendOrder(ctx context.Context, ccy string, dailyInterestRate, size float64, term int64) (string, error) {
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if dailyInterestRate == 0 {
		return "", errors.New("dailyIntRate can not be zero")
	}
	params["dailyIntRate"] = strconv.FormatFloat(dailyInterestRate, 'f', -1, 64)
	if size == 0 {
		return "", errors.New("size can not be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	if term == 0 {
		return "", errors.New("term can not be zero")
	}
	params["term"] = strconv.FormatInt(term, 10)
	resp := struct {
		OrderID string `json:"orderId"`
		Error
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinLendOrder, params, &resp)
}

// CancelLendOrder used to cancel lend order
func (ku *Kucoin) CancelLendOrder(ctx context.Context, orderID string) error {
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, kucoinLendOrder+"/"+orderID, nil, &resp)
}

// SetAutoLend used to set up the automatic lending for a specified currency
func (ku *Kucoin) SetAutoLend(ctx context.Context, ccy string, dailyInterestRate, retainSize float64, term int64, isEnable bool) error {
	if ccy == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if dailyInterestRate == 0 {
		return errors.New("dailyIntRate can not be zero")
	}
	params["dailyIntRate"] = strconv.FormatFloat(dailyInterestRate, 'f', -1, 64)
	if retainSize == 0 {
		return errors.New("retainSize can not be zero")
	}
	params["retainSize"] = strconv.FormatFloat(retainSize, 'f', -1, 64)
	if term == 0 {
		return errors.New("term can not be zero")
	}
	params["term"] = strconv.FormatInt(term, 10)
	params["isEnable"] = isEnable
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinSetAutoLend, params, &resp)
}

// GetActiveOrder gets active lend orders
func (ku *Kucoin) GetActiveOrder(ctx context.Context, ccy string) ([]LendOrder, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	resp := struct {
		Data []LendOrder `json:"items"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetActiveOrder, params), nil, &resp)
}

// GetLendHistory gets lend orders
func (ku *Kucoin) GetLendHistory(ctx context.Context, ccy string) ([]LendOrderHistory, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	resp := struct {
		Data []LendOrderHistory `json:"items"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetLendHistory, params), nil, &resp)
}

// GetUnsettledLendOrder gets outstanding lend order list
func (ku *Kucoin) GetUnsettledLendOrder(ctx context.Context, ccy string) ([]UnsettleLendOrder, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	resp := struct {
		Data []UnsettleLendOrder `json:"items"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetUnsettleLendOrder, params), nil, &resp)
}

// GetSettledLendOrder gets settle lend orders
func (ku *Kucoin) GetSettledLendOrder(ctx context.Context, ccy string) ([]SettleLendOrder, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	resp := struct {
		Data []SettleLendOrder `json:"items"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetSettleLendOrder, params), nil, &resp)
}

// GetAccountLendRecord get the lending history of the main account
func (ku *Kucoin) GetAccountLendRecord(ctx context.Context, ccy string) ([]LendRecord, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp []LendRecord
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetAccountLendRecord, params), nil, &resp)
}

// GetLendingMarketData get the lending market data
func (ku *Kucoin) GetLendingMarketData(ctx context.Context, ccy string, term int64) ([]LendMarketData, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if term != 0 {
		params.Set("term", strconv.FormatInt(term, 10))
	}
	var resp []LendMarketData
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetLendingMarketData, params), nil, &resp)
}

// GetMarginTradeData get the last 300 fills in the lending and borrowing market
func (ku *Kucoin) GetMarginTradeData(ctx context.Context, ccy string) ([]MarginTradeData, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	var resp []MarginTradeData
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetMarginTradeData, params), nil, &resp)
}

// GetIsolatedMarginPairConfig get the current isolated margin trading pair configuration
func (ku *Kucoin) GetIsolatedMarginPairConfig(ctx context.Context) ([]IsolatedMarginPairConfig, error) {
	var resp []IsolatedMarginPairConfig
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetIsolatedMarginPairConfig, nil, &resp)
}

// GetIsolatedMarginAccountInfo get all isolated margin accounts of the current user
func (ku *Kucoin) GetIsolatedMarginAccountInfo(ctx context.Context, balanceCurrency string) (*IsolatedMarginAccountInfo, error) {
	params := url.Values{}
	if balanceCurrency != "" {
		params.Set("balanceCurrency", balanceCurrency)
	}
	var resp *IsolatedMarginAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetIsolatedMarginAccountInfo, params), nil, &resp)
}

// GetSingleIsolatedMarginAccountInfo get single isolated margin accounts of the current user
func (ku *Kucoin) GetSingleIsolatedMarginAccountInfo(ctx context.Context, symbol string) (*AssetInfo, error) {
	if symbol == "" {
		return nil, errors.New("symbol can not be empty")
	}
	var resp *AssetInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetSingleIsolatedMarginAccountInfo+symbol, nil, &resp)
}

// InitiateIsolatedMarginBorrowing initiates isolated margin borrowing
func (ku *Kucoin) InitiateIsolatedMarginBorrowing(ctx context.Context, symbol, ccy, borrowStrategy, period string, size, maxRate int64) (*IsolatedMarginBorrowing, error) {
	if symbol == "" {
		return nil, errors.New("symbol can not be empty")
	}
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["currency"] = ccy
	if borrowStrategy == "" {
		return nil, errors.New("borrowStrategy can not be empty")
	}
	params["borrowStrategy"] = borrowStrategy
	if size == 0 {
		return nil, errors.New("size can not be zero")
	}
	params["size"] = strconv.FormatInt(size, 10)

	if period != "" {
		params["period"] = period
	}
	if maxRate == 0 {
		params["maxRate"] = strconv.FormatInt(maxRate, 10)
	}
	var resp *IsolatedMarginBorrowing
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinInitiateIsolatedMarginBorrowing, params, &resp)
}

// GetIsolatedOutstandingRepaymentRecords get the outstanding repayment records of isolated margin positions
func (ku *Kucoin) GetIsolatedOutstandingRepaymentRecords(ctx context.Context, symbol, ccy string, pageSize, currentPage int64) (*OutstandingRepaymentRecordsResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	var resp *OutstandingRepaymentRecordsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetIsolatedOutstandingRepaymentRecords, params), nil, &resp)
}

// GetIsolatedMarginRepaymentRecords get the repayment records of isolated margin positions
func (ku *Kucoin) GetIsolatedMarginRepaymentRecords(ctx context.Context, symbol, ccy string, pageSize, currentPage int64) (*CompletedRepaymentRecordsResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	var resp *CompletedRepaymentRecordsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetIsolatedMarginRepaymentRecords, params), nil, &resp)
}

// InitiateIsolatedMarginQuickRepayment is used to initiate quick repayment for isolated margin accounts
func (ku *Kucoin) InitiateIsolatedMarginQuickRepayment(ctx context.Context, symbol, ccy, seqStrategy string, size int64) error {
	if symbol == "" {
		return currency.ErrCurrencyPairEmpty
	}
	if size == 0 {
		return errors.New("size can not be zero")
	}
	if seqStrategy == "" {
		return errors.New("seqStrategy can not be empty")
	}
	if ccy == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["currency"] = ccy
	params["seqStrategy"] = seqStrategy
	params["size"] = strconv.FormatInt(size, 10)
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinInitiateIsolatedMarginQuickRepayment, params, &resp)
}

// InitiateIsolatedMarginSingleRepayment is used to initiate quick repayment for single margin accounts
func (ku *Kucoin) InitiateIsolatedMarginSingleRepayment(ctx context.Context, symbol, ccy, loanID string, size int64) error {
	if symbol == "" {
		return currency.ErrCurrencyPairEmpty
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	if ccy == "" {
		return currency.ErrCurrencyCodeEmpty
	}
	params["currency"] = ccy
	if loanID == "" {
		return errors.New("loanId can not be empty")
	}
	params["loanId"] = loanID
	if size == 0 {
		return errors.New("size can not be zero")
	}
	params["size"] = strconv.FormatInt(size, 10)
	resp := struct {
		Error
	}{}
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinInitiateIsolatedMarginSingleRepayment, params, &resp)
}

// GetCurrentServerTime gets the server time
func (ku *Kucoin) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Timestamp convert.ExchangeTime `json:"data"`
		Error
	}{}
	err := ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetServerTime, &resp)
	if err != nil {
		return time.Time{}, err
	}
	return resp.Timestamp.Time(), nil
}

// GetServiceStatus gets the service status
func (ku *Kucoin) GetServiceStatus(ctx context.Context) (*ServiceStatus, error) {
	var resp *ServiceStatus
	return resp, ku.SendHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, kucoinGetServiceStatus, &resp)
}

// PostOrder used to place two types of orders: limit and market
// Note: use this only for SPOT trades
func (ku *Kucoin) PostOrder(ctx context.Context, arg *SpotOrderParam) (string, error) {
	if arg.ClientOrderID == "" {
		// NOTE: 128 bit max length character string. UUID recommended.
		return "", errInvalidClientOrderID
	}
	if arg.Side == "" {
		return "", order.ErrSideIsInvalid
	}
	if arg.Symbol.IsEmpty() {
		return "", fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return "", fmt.Errorf("%w, price =%.3f", errInvalidPrice, arg.Price)
		}
		if arg.Size <= 0 {
			return "", errInvalidSize
		}
		if arg.VisibleSize < 0 {
			return "", fmt.Errorf("%w, visible size must be non-zero positive value", errInvalidSize)
		}
	case "market":
		if arg.Size == 0 && arg.Funds == 0 {
			return "", errSizeOrFundIsRequired
		}
	default:
		return "", fmt.Errorf("%w %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	var resp struct {
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
		Error
	}
	return resp.Data.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, kucoinPostOrder, &arg, &resp)
}

// PostMarginOrder used to place two types of margin orders: limit and market
func (ku *Kucoin) PostMarginOrder(ctx context.Context, arg *MarginOrderParam) (*PostMarginOrderResp, error) {
	if arg.ClientOrderID == "" {
		return nil, errInvalidClientOrderID
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Symbol.IsEmpty() {
		return nil, fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	arg.OrderType = strings.ToLower(arg.OrderType)
	switch arg.OrderType {
	case "limit", "":
		if arg.Price <= 0 {
			return nil, fmt.Errorf("%w, price=%.3f", errInvalidPrice, arg.Price)
		}
		if arg.Size <= 0 {
			return nil, errInvalidSize
		}
		if arg.VisibleSize < 0 {
			return nil, fmt.Errorf("%w, visible size must be non-zero positive value", errInvalidSize)
		}
	case "market":
		sum := arg.Size + arg.Funds
		if sum <= 0 || (sum != arg.Size && sum != arg.Funds) {
			return nil, fmt.Errorf("%w, either 'size' or 'funds' has to be set, but not both", errSizeOrFundIsRequired)
		}
	default:
		return nil, fmt.Errorf("%w %s", order.ErrTypeIsInvalid, arg.OrderType)
	}
	resp := struct {
		PostMarginOrderResp
		Error
	}{}
	return &resp.PostMarginOrderResp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeMarginOrdersEPL, http.MethodPost, kucoinPostMarginOrder, &arg, &resp)
}

// PostBulkOrder used to place 5 orders at the same time. The order type must be a limit order of the same symbol
// Note: it supports only SPOT trades
// Note: To check if order was posted successfully, check status field in response
func (ku *Kucoin) PostBulkOrder(ctx context.Context, symbol string, orderList []OrderRequest) ([]PostBulkOrderResp, error) {
	if symbol == "" {
		return nil, errors.New("symbol can not be empty")
	}
	for i := range orderList {
		if orderList[i].ClientOID == "" {
			return nil, errors.New("clientOid can not be empty")
		}
		if orderList[i].Side == "" {
			return nil, errors.New("side can not be empty")
		}
		if orderList[i].Price <= 0 {
			return nil, errors.New("price must be positive")
		}
		if orderList[i].Size <= 0 {
			return nil, errors.New("size must be positive")
		}
	}
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["orderList"] = orderList
	resp := &struct {
		Data []PostBulkOrderResp `json:"data"`
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeBulkOrdersEPL, http.MethodPost, kucoinPostBulkOrder, params, &resp)
}

// CancelSingleOrder used to cancel single order previously placed
func (ku *Kucoin) CancelSingleOrder(ctx context.Context, orderID string) ([]string, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelOrderEPL, http.MethodDelete, kucoinOrderByID+orderID, nil, &resp)
}

// CancelOrderByClientOID used to cancel order via the clientOid
func (ku *Kucoin) CancelOrderByClientOID(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, kucoinOrderByClientOID+orderID, nil, &resp)
}

// CancelAllOpenOrders used to cancel all order based upon the parameters passed
func (ku *Kucoin) CancelAllOpenOrders(ctx context.Context, symbol, tradeType string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, cancelAllOrdersEPL, http.MethodDelete, common.EncodeURLValues(kucoinOrders, params), nil, &resp)
}

// ListOrders gets the user order list
func (ku *Kucoin) ListOrders(ctx context.Context, status, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*OrdersListResponse, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if status != "" {
		params.Set("status", status)
	}
	var resp *OrdersListResponse
	err := ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listOrdersEPL, http.MethodGet, common.EncodeURLValues(kucoinOrders, params), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func fillParams(symbol, side, orderType, tradeType string, startAt, endAt time.Time) url.Values {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	return params
}

// GetRecentOrders get orders in the last 24 hours.
func (ku *Kucoin) GetRecentOrders(ctx context.Context) ([]OrderDetail, error) {
	var resp []OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetRecentOrders, nil, &resp)
}

// GetOrderByID get a single order info by order ID
func (ku *Kucoin) GetOrderByID(ctx context.Context, orderID string) (*OrderDetail, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinOrderByID+orderID, nil, &resp)
}

// GetOrderByClientSuppliedOrderID get a single order info by client order ID
func (ku *Kucoin) GetOrderByClientSuppliedOrderID(ctx context.Context, clientOID string) (*OrderDetail, error) {
	if clientOID == "" {
		return nil, errors.New("client order ID can not be empty")
	}
	var resp *OrderDetail
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinOrderByClientOID+clientOID, nil, &resp)
}

// GetFills get fills
func (ku *Kucoin) GetFills(ctx context.Context, orderID, symbol, side, orderType, tradeType string, startAt, endAt time.Time) (*ListFills, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	var resp *ListFills
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, listFillsEPL, http.MethodGet, common.EncodeURLValues(kucoinGetFills, params), nil, &resp)
}

// GetRecentFills get a list of 1000 fills in last 24 hours
func (ku *Kucoin) GetRecentFills(ctx context.Context) ([]Fill, error) {
	var resp []Fill
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetRecentFills, nil, &resp)
}

// PostStopOrder used to place two types of stop orders: limit and market
func (ku *Kucoin) PostStopOrder(ctx context.Context, clientOID, side, symbol, orderType, remark, stop, stp, tradeType, timeInForce string, size, price, stopPrice, cancelAfter, visibleSize, funds float64, postOnly, hidden, iceberg bool) (string, error) {
	params := make(map[string]interface{})
	if clientOID == "" {
		return "", errors.New("clientOid can not be empty")
	}
	params["clientOid"] = clientOID
	if side == "" {
		return "", errors.New("side can not be empty")
	}
	params["side"] = side
	if symbol == "" {
		return "", fmt.Errorf("%w, empty symbol", currency.ErrCurrencyPairEmpty)
	}
	params["symbol"] = symbol
	if remark != "" {
		params["remark"] = remark
	}
	if stop != "" {
		params["stop"] = stop
		if stopPrice <= 0 {
			return "", errors.New("stopPrice is required")
		}
		params["stopPrice"] = strconv.FormatFloat(stopPrice, 'f', -1, 64)
	}
	if stp != "" {
		params["stp"] = stp
	}
	if tradeType != "" {
		params["tradeType"] = tradeType
	}
	orderType = strings.ToLower(orderType)
	switch orderType {
	case "limit", "":
		if price <= 0 {
			return "", errors.New("price is required")
		}
		params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
		if size <= 0 {
			return "", errors.New("size can not be zero or negative")
		}
		params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		if timeInForce != "" {
			params["timeInForce"] = timeInForce
		}
		if cancelAfter > 0 && timeInForce == "GTT" {
			params["cancelAfter"] = strconv.FormatFloat(cancelAfter, 'f', -1, 64)
		}
		params["postOnly"] = postOnly
		params["hidden"] = hidden
		params["iceberg"] = iceberg
		if visibleSize > 0 {
			params["visibleSize"] = strconv.FormatFloat(visibleSize, 'f', -1, 64)
		}
	case "market":
		switch {
		case size > 0:
			params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
		case funds > 0:
			params["funds"] = strconv.FormatFloat(funds, 'f', -1, 64)
		default:
			return "", errSizeOrFundIsRequired
		}
	default:
		return "", fmt.Errorf("%w, order type: %s", order.ErrTypeIsInvalid, orderType)
	}
	if orderType != "" {
		params["type"] = orderType
	}
	resp := struct {
		OrderID string `json:"orderId"`
		Error
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, placeOrderEPL, http.MethodPost, kucoinStopOrder, params, &resp)
}

// CancelStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) CancelStopOrder(ctx context.Context, orderID string) ([]string, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		Data []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.Data, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, kucoinStopOrderByID+orderID, nil, &resp)
}

// CancelStopOrders used to cancel all order based upon the parameters passed
func (ku *Kucoin) CancelStopOrders(ctx context.Context, symbol, tradeType, orderIDs string) ([]string, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if tradeType != "" {
		params.Set("tradeType", tradeType)
	}
	if orderIDs != "" {
		params.Set("orderIds", orderIDs)
	}
	resp := struct {
		CancelledOrderIDs []string `json:"cancelledOrderIds"`
		Error
	}{}
	return resp.CancelledOrderIDs, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues(kucoinCancelAllStopOrder, params), nil, &resp)
}

// GetStopOrder used to cancel single stop order previously placed
func (ku *Kucoin) GetStopOrder(ctx context.Context, orderID string) (*StopOrder, error) {
	if orderID == "" {
		return nil, errors.New("orderID can not be empty")
	}
	resp := struct {
		StopOrder
		Error
	}{}
	return &resp.StopOrder, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinStopOrderByID+orderID, nil, &resp)
}

// ListStopOrders get all current untriggered stop orders
func (ku *Kucoin) ListStopOrders(ctx context.Context, symbol, side, orderType, tradeType, orderIDs string, startAt, endAt time.Time, currentPage, pageSize int64) (*StopOrderListResponse, error) {
	params := fillParams(symbol, side, orderType, tradeType, startAt, endAt)
	if orderIDs != "" {
		params.Set("orderIds", orderIDs)
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *StopOrderListResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinStopOrder, params), nil, &resp)
}

// GetStopOrderByClientID get a stop order information via the clientOID
func (ku *Kucoin) GetStopOrderByClientID(ctx context.Context, symbol, clientOID string) ([]StopOrder, error) {
	if clientOID == "" {
		return nil, errors.New("clientOID can not be empty")
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []StopOrder
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetStopOrderByClientID, params), nil, &resp)
}

// CancelStopOrderByClientID used to cancel a stop order via the clientOID.
func (ku *Kucoin) CancelStopOrderByClientID(ctx context.Context, symbol, clientOID string) (*CancelOrderResponse, error) {
	if clientOID == "" {
		return nil, errors.New("clientOID can not be empty")
	}
	params := url.Values{}
	params.Set("clientOid", clientOID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *CancelOrderResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues(kucoinCancelStopOrderByClientID, params), nil, &resp)
}

// CreateSubUser creates a new sub-user for the account.
func (ku *Kucoin) CreateSubUser(ctx context.Context, subAccountName, password, remarks, access string) (*SubAccount, error) {
	params := make(map[string]interface{})
	if regexp.MustCompile("^[a-zA-Z0-9]{7-32}$").MatchString(subAccountName) {
		return nil, errors.New("invalid sub-account name")
	}
	if regexp.MustCompile("^[a-zA-Z0-9]{7-24}$").MatchString(password) {
		return nil, errInvalidPassPhraseInstance
	}
	params["subName"] = subAccountName
	params["password"] = password
	if remarks != "" {
		params["remarks"] = remarks
	}
	if access != "" {
		params["access"] = access
	}
	var resp *SubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinSubUserCreated, params, &resp)
}

// GetSubAccountSpotAPIList used to obtain a list of Spot APIs pertaining to a sub-account.
func (ku *Kucoin) GetSubAccountSpotAPIList(ctx context.Context, subAccountName, apiKeys string) (*SubAccountResponse, error) {
	params := url.Values{}
	if subAccountRegExp.MatchString(subAccountName) {
		return nil, errInvalidSubAccountName
	}
	params.Set("subName", subAccountName)
	if apiKeys != "" {
		params.Set("apiKey", apiKeys)
	}
	var resp SubAccountResponse
	return &resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinSubAccountSpotAPIs, params), nil, &resp)
}

// CreateSpotAPIsForSubAccount can be used to create Spot APIs for sub-accounts.
func (ku *Kucoin) CreateSpotAPIsForSubAccount(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if subAccountRegExp.MatchString(arg.SubAccountName) {
		return nil, errInvalidSubAccountName
	}
	if subAccountPassphraseRegExp.MatchString(arg.Passphrase) {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	if arg.Remark == "" {
		return nil, errors.New("remark is required")
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinSubAccountSpotAPIs, &arg, &resp)
}

// ModifySubAccountSpotAPIs modifies sub-account Spot APIs.
func (ku *Kucoin) ModifySubAccountSpotAPIs(ctx context.Context, arg *SpotAPISubAccountParams) (*SpotAPISubAccount, error) {
	if subAccountRegExp.MatchString(arg.SubAccountName) {
		return nil, errInvalidSubAccountName
	}
	if subAccountPassphraseRegExp.MatchString(arg.Passphrase) {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	if arg.Remark == "" {
		return nil, errors.New("remark is required")
	}
	var resp *SpotAPISubAccount
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPut, kucoinUpdateModifySubAccountSpotAPIs, &arg, &resp)
}

// DeleteSubAccountSpotAPI delete sub-account Spot APIs.
func (ku *Kucoin) DeleteSubAccountSpotAPI(ctx context.Context, apiKey, passphrase, subAccountName string) (*DeleteSubAccountResponse, error) {
	if subAccountRegExp.MatchString(subAccountName) {
		return nil, errInvalidSubAccountName
	}
	if subAccountPassphraseRegExp.MatchString(passphrase) {
		return nil, fmt.Errorf("%w, must contain 7-32 characters. cannot contain any spaces", errInvalidPassPhraseInstance)
	}
	if apiKey == "" {
		return nil, errors.New("apiKey is required")
	}
	params := url.Values{}
	params.Set("apiKey", apiKey)
	params.Set("passphrase", passphrase)
	params.Set("subName", subAccountName)
	var resp *DeleteSubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, common.EncodeURLValues(kucoinSubAccountSpotAPIs, params), nil, &resp)
}

// GetUserInfoOfAllSubAccounts get the user info of all sub-users via this interface.
func (ku *Kucoin) GetUserInfoOfAllSubAccounts(ctx context.Context) (*SubAccountResponse, error) {
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinSubUser, nil, &resp)
}

// GetPaginatedListOfSubAccounts to retrieve a paginated list of sub-accounts. Pagination is required.
func (ku *Kucoin) GetPaginatedListOfSubAccounts(ctx context.Context, currentPage, pageSize int64) (*SubAccountResponse, error) {
	params := url.Values{}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if currentPage > 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	var resp *SubAccountResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinSubUser, params), nil, &resp)
}

// GetAllAccounts get all accounts
func (ku *Kucoin) GetAllAccounts(ctx context.Context, ccy, accountType string) ([]AccountInfo, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if accountType != "" {
		params.Set("type", accountType)
	}
	var resp []AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinAccount, params), nil, &resp)
}

// GetAccount get information of single account
func (ku *Kucoin) GetAccount(ctx context.Context, accountID string) (*AccountInfo, error) {
	var resp *AccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetAccount+accountID, nil, &resp)
}

// GetAccountLedgers get the history of deposit/withdrawal of all accounts, supporting inquiry of various currencies
func (ku *Kucoin) GetAccountLedgers(ctx context.Context, ccy, direction, bizType string, startAt, endAt time.Time) (*AccountLedgerResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if bizType != "" {
		params.Set("bizType", bizType)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *AccountLedgerResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveAccountLedgerEPL, http.MethodGet, common.EncodeURLValues(kucoinGetAccountLedgers, params), nil, &resp)
}

// GetAccountSummaryInformation this can be used to obtain account summary information.
func (ku *Kucoin) GetAccountSummaryInformation(ctx context.Context) (*AccountSummaryInformation, error) {
	var resp *AccountSummaryInformation
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinUserInfo, nil, &resp)
}

// GetSubAccountBalance get account info of a sub-user specified by the subUserID
func (ku *Kucoin) GetSubAccountBalance(ctx context.Context, subUserID string, includeBaseAmount bool) (*SubAccountInfo, error) {
	params := url.Values{}
	if includeBaseAmount {
		params.Set("includeBaseAmount", "true")
	}
	var resp *SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetSubAccountBalance+subUserID, params), nil, &resp)
}

// GetAggregatedSubAccountBalance get the account info of all sub-users
func (ku *Kucoin) GetAggregatedSubAccountBalance(ctx context.Context) ([]SubAccountInfo, error) {
	var resp []SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, kucoinGetAggregatedSubAccountBalance, nil, &resp)
}

// GetPaginatedSubAccountInformation this endpoint can be used to get paginated sub-account information. Pagination is required.
func (ku *Kucoin) GetPaginatedSubAccountInformation(ctx context.Context, currentPage, pageSize int64) ([]SubAccountInfo, error) {
	params := url.Values{}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp []SubAccountInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetAggregatedSubAccountBalance, params), nil, &resp)
}

// GetTransferableBalance get the transferable balance of a specified account
func (ku *Kucoin) GetTransferableBalance(ctx context.Context, ccy, accountType, tag string) (*TransferableBalanceInfo, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if accountType == "" {
		return nil, errors.New("accountType can not be empty")
	}
	params.Set("type", accountType)
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *TransferableBalanceInfo
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetTransferableBalance, params), nil, &resp)
}

// TransferMainToSubAccount used to transfer funds from main account to sub-account
func (ku *Kucoin) TransferMainToSubAccount(ctx context.Context, clientOID, ccy, amount, direction, accountType, subAccountType, subUserID string) (string, error) {
	if clientOID == "" {
		return "", errors.New("clientOID can not be empty")
	}
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	if amount == "" {
		return "", errors.New("amount can not be empty")
	}
	if direction == "" {
		return "", errors.New("direction can not be empty")
	}
	if subUserID == "" {
		return "", errors.New("subUserID can not be empty")
	}
	params := make(map[string]interface{})
	params["clientOid"] = clientOID
	params["currency"] = ccy
	params["amount"] = amount
	params["direction"] = direction
	if accountType != "" {
		params["accountType"] = accountType
	}
	if subAccountType != "" {
		params["subAccountType"] = subAccountType
	}
	params["subUserId"] = subUserID
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, masterSubUserTransferEPL, http.MethodPost, kucoinTransferMainToSubAccount, params, &resp)
}

// MakeInnerTransfer used to transfer funds between accounts internally
func (ku *Kucoin) MakeInnerTransfer(ctx context.Context, clientOID, ccy, from, to, amount, fromTag, toTag string) (string, error) {
	if clientOID == "" {
		return "", errors.New("clientOID can not be empty")
	}
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	if amount == "" {
		return "", errors.New("amount can not be empty")
	}
	if from == "" {
		return "", errors.New("from can not be empty")
	}
	if to == "" {
		return "", errors.New("to can not be empty")
	}
	params := make(map[string]interface{})
	params["clientOid"] = clientOID
	params["currency"] = ccy
	params["amount"] = amount
	params["from"] = from
	params["to"] = to
	if fromTag != "" {
		params["fromTag"] = fromTag
	}
	if toTag != "" {
		params["toTag"] = toTag
	}
	resp := struct {
		OrderID string `json:"orderId"`
	}{}
	return resp.OrderID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinInnerTransfer, params, &resp)
}

// CreateDepositAddress create a deposit address for a currency you intend to deposit
func (ku *Kucoin) CreateDepositAddress(ctx context.Context, ccy, chain string) (*DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if chain != "" {
		params["chain"] = chain
	}
	var resp *DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinGetDepositAddressV1, params, &resp)
}

// GetDepositAddressesV2 get all deposit addresses for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressesV2(ctx context.Context, ccy string) ([]DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	var resp []DepositAddress
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetDepositAddressesV2, params), nil, &resp)
}

// GetDepositAddressV1 get a deposit address for the currency you intend to deposit
func (ku *Kucoin) GetDepositAddressV1(ctx context.Context, ccy, chain string) (*DepositAddress, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp DepositAddress
	return &resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetDepositAddressV1, params), nil, &resp)
}

// GetDepositList get deposit list items and sorted to show the latest first
func (ku *Kucoin) GetDepositList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*DepositResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *DepositResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveDepositListEPL, http.MethodGet, common.EncodeURLValues(kucoinGetDepositList, params), nil, &resp)
}

// GetHistoricalDepositList get historical deposit list items
func (ku *Kucoin) GetHistoricalDepositList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *HistoricalDepositWithdrawalResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveV1HistoricalDepositListEPL, http.MethodGet, common.EncodeURLValues(kucoinGetHistoricalDepositList, params), nil, &resp)
}

// GetWithdrawalList get withdrawal list items
func (ku *Kucoin) GetWithdrawalList(ctx context.Context, ccy, status string, startAt, endAt time.Time) (*WithdrawalsResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *WithdrawalsResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveWithdrawalListEPL, http.MethodGet, common.EncodeURLValues(kucoinWithdrawal, params), nil, &resp)
}

// GetHistoricalWithdrawalList get historical withdrawal list items
func (ku *Kucoin) GetHistoricalWithdrawalList(ctx context.Context, ccy, status string, startAt, endAt time.Time, currentPage, pageSize int64) (*HistoricalDepositWithdrawalResponse, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if currentPage != 0 {
		params.Set("currentPage", strconv.FormatInt(currentPage, 10))
	}
	if pageSize != 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *HistoricalDepositWithdrawalResponse
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, retrieveV1HistoricalWithdrawalListEPL, http.MethodGet, common.EncodeURLValues(kucoinGetHistoricalWithdrawalList, params), nil, &resp)
}

// GetWithdrawalQuotas get withdrawal quota details
func (ku *Kucoin) GetWithdrawalQuotas(ctx context.Context, ccy, chain string) (*WithdrawalQuota, error) {
	if ccy == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy)
	if chain != "" {
		params.Set("chain", chain)
	}
	var resp *WithdrawalQuota
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinGetWithdrawalQuotas, params), nil, &resp)
}

// ApplyWithdrawal create a withdrawal request
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first, and then withdraw from the MAIN account
func (ku *Kucoin) ApplyWithdrawal(ctx context.Context, ccy, address, memo, remark, chain, feeDeductType string, isInner bool, amount float64) (string, error) {
	if ccy == "" {
		return "", currency.ErrCurrencyPairEmpty
	}
	params := make(map[string]interface{})
	params["currency"] = ccy
	if address == "" {
		return "", errors.New("address can not be empty")
	}
	params["address"] = address
	if amount == 0 {
		return "", errors.New("amount can not be empty")
	}
	params["amount"] = amount
	if memo != "" {
		params["memo"] = memo
	}
	params["isInner"] = isInner
	if remark != "" {
		params["remark"] = remark
	}
	if chain != "" {
		params["chain"] = chain
	}
	if feeDeductType != "" {
		params["feeDeductType"] = feeDeductType
	}
	resp := struct {
		WithdrawalID string `json:"withdrawalId"`
		Error
	}{}
	return resp.WithdrawalID, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodPost, kucoinWithdrawal, params, &resp)
}

// CancelWithdrawal used to cancel a withdrawal request
func (ku *Kucoin) CancelWithdrawal(ctx context.Context, withdrawalID string) error {
	return ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodDelete, kucoinCancelWithdrawal+withdrawalID, nil, &struct{}{})
}

// GetBasicFee get basic fee rate of users
func (ku *Kucoin) GetBasicFee(ctx context.Context, currencyType string) (*Fees, error) {
	params := url.Values{}
	if currencyType != "" {
		params.Set("currencyType", currencyType)
	}
	var resp *Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, common.EncodeURLValues(kucoinBasicFee, params), nil, &resp)
}

// GetTradingFee get fee rate of trading pairs
// WARNING: There is a limit of 10 currency pairs allowed to be requested per call.
func (ku *Kucoin) GetTradingFee(ctx context.Context, pairs currency.Pairs) ([]Fees, error) {
	if len(pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	path := kucoinTradingFee + "?symbols=" + pairs.Upper().Join()
	var resp []Fees
	return resp, ku.SendAuthHTTPRequest(ctx, exchange.RestSpot, defaultSpotEPL, http.MethodGet, path, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (ku *Kucoin) SendHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, result interface{}) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := ku.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	err = ku.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        resp,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	if result == nil {
		return errNoValidResponseFromServer
	}
	return resp.GetError()
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Request parameters are added to path variable for GET and DELETE request and for other requests its passed in params variable
func (ku *Kucoin) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, method, path string, arg, result interface{}) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	creds, err := ku.GetCredentials(ctx)
	if err != nil {
		return err
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := ku.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	if value.IsNil() || value.Kind() != reflect.Pointer {
		return fmt.Errorf("%w receiver has to be non-nil pointer", errInvalidResponseReceiver)
	}
	err = ku.SendPayload(ctx, epl, func() (*request.Item, error) {
		var (
			body    io.Reader
			payload []byte
		)
		if arg != nil {
			payload, err = json.Marshal(arg)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		timeStamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		var signHash, passPhraseHash []byte
		signHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(timeStamp+method+"/api"+path+string(payload)), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		passPhraseHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(creds.ClientID), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        crypto.Base64Encode(signHash),
			"KC-API-TIMESTAMP":   timeStamp,
			"KC-API-PASSPHRASE":  crypto.Base64Encode(passPhraseHash),
			"KC-API-KEY-VERSION": kucoinAPIKeyVersion,
			"Content-Type":       "application/json",
		}
		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          body,
			Result:        &resp,
			Verbose:       ku.Verbose,
			HTTPDebugging: ku.HTTPDebugging,
			HTTPRecording: ku.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	if result == nil {
		return errNoValidResponseFromServer
	}
	return resp.GetError()
}

func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1min", nil
	case kline.ThreeMin:
		return "3min", nil
	case kline.FiveMin:
		return "5min", nil
	case kline.FifteenMin:
		return "15min", nil
	case kline.ThirtyMin:
		return "30min", nil
	case kline.OneHour:
		return "1hour", nil
	case kline.TwoHour:
		return "2hour", nil
	case kline.FourHour:
		return "4hour", nil
	case kline.SixHour:
		return "6hour", nil
	case kline.EightHour:
		return "8hour", nil
	case kline.TwelveHour:
		return "12hour", nil
	case kline.OneDay:
		return "1day", nil
	case kline.OneWeek:
		return "1week", nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

func (ku *Kucoin) stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "match":
		return order.Filled, nil
	case "open":
		return order.Open, nil
	case "done":
		return order.Closed, nil
	default:
		return order.StringToOrderStatus(status)
	}
}

func (ku *Kucoin) accountTypeToString(a asset.Item) string {
	switch a {
	case asset.Spot:
		return "trade"
	case asset.Margin:
		return "margin"
	case asset.Empty:
		return ""
	default:
		return "main"
	}
}

func (ku *Kucoin) accountToTradeTypeString(a asset.Item, marginMode string) string {
	switch a {
	case asset.Spot:
		return "TRADE"
	case asset.Margin:
		if strings.EqualFold(marginMode, "isolated") {
			return "MARGIN_ISOLATED_TRADE"
		}
		return "MARGIN_TRADE"
	default:
		return ""
	}
}

func (ku *Kucoin) orderTypeToString(orderType order.Type) (string, error) {
	switch orderType {
	case order.AnyType, order.UnknownType:
		return "", nil
	case order.Market, order.Limit:
		return orderType.Lower(), nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

func (ku *Kucoin) orderSideString(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return order.Buy.Lower(), nil
	case side.IsShort():
		return order.Sell.Lower(), nil
	case side == order.AnySide:
		return "", nil
	default:
		return "", fmt.Errorf("%w, side:%s", order.ErrSideIsInvalid, side.String())
	}
}
