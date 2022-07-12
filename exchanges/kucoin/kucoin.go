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
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Kucoin is the overarching type across this package
type Kucoin struct {
	exchange.Base
}

const (
	kucoinAPIURL        = "https://api.kucoin.com"
	kucoinAPIVersion    = "1"
	kucoinAPIKeyVersion = "2"

	// Public endpoints
	kucoinGetSymbols             = "/api/v1/symbols"
	kucoinGetTicker              = "/api/v1/market/orderbook/level1"
	kucoinGetAllTickers          = "/api/v1/market/allTickers"
	kucoinGet24hrStats           = "/api/v1/market/stats"
	kucoinGetMarketList          = "/api/v1/markets"
	kucoinGetPartOrderbook20     = "/api/v1/market/orderbook/level2_20"
	kucoinGetPartOrderbook100    = "/api/v1/market/orderbook/level2_100"
	kucoinGetTradeHistory        = "/api/v1/market/histories"
	kucoinGetKlines              = "/api/v1/market/candles"
	kucoinGetCurrencies          = "/api/v1/currencies"
	kucoinGetCurrency            = "/api/v2/currencies/"
	kucoinGetFiatPrice           = "/api/v1/prices"
	kucoinGetMarkPrice           = "/api/v1/mark-price/%s/current"
	kucoinGetMarginConfiguration = "/api/v1/margin/config"
	kucoinGetServerTime          = "/api/v1/timestamp"
	kucoinGetServiceStatus       = "/api/v1/status"

	// Authenticated endpoints
	kucoinGetOrderbook         = "/api/v3/market/orderbook/level2"
	kucoinGetMarginAccount     = "/api/v1/margin/account"
	kucoinGetMarginRiskLimit   = "/api/v1/risk/limit/strategy"
	kucoinBorrowOrder          = "/api/v1/margin/borrow"
	kucoinGetOutstandingRecord = "/api/v1/margin/borrow/outstanding"
	kucoinGetRepaidRecord      = "/api/v1/margin/borrow/repaid"
	kucoinOneClickRepayment    = "/api/v1/margin/repay/all"
	kucoinRepaySingleOrder     = "/api/v1/margin/repay/single"
	kucoinPostLendOrder        = "/api/v1/margin/lend"
	kucoinCancelLendOrder      = "/api/v1/margin/lend/%s"
	kucoinSetAutoLend          = "/api/v1/margin/toggle-auto-lend"
	kucoinGetActiveOrder       = "/api/v1/margin/lend/active"
	kucoinGetLendHistory       = "/api/v1/margin/lend/done"
	kucoinGetUnsettleLendOrder = "/api/v1/margin/lend/trade/unsettled"
	kucoinGetSettleLendOrder   = "/api/v1/margin/lend/trade/settled"
	kucoinGetAccountLendRecord = "/api/v1/margin/lend/assets"
	kucoinGetLendingMarketData = "/api/v1/margin/market"
	kucoinGetMarginTradeData   = "/api/v1/margin/trade/last"
)

// GetSymbols gets pairs details on the exchange
func (k *Kucoin) GetSymbols(ctx context.Context, currency string) ([]SymbolInfo, error) {
	resp := struct {
		Data []SymbolInfo `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("market", currency)
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetSymbols, params), publicSpotRate, &resp)
}

// GetTicker gets pair ticker information
func (k *Kucoin) GetTicker(ctx context.Context, pair string) (Ticker, error) {
	resp := struct {
		Data Ticker `json:"data"`
		Error
	}{}

	params := url.Values{}
	if pair == "" {
		return Ticker{}, errors.New("pair can't be empty") // TODO: error as constant
	}
	params.Set("symbol", pair)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetTicker, params), publicSpotRate, &resp)
}

// GetAllTickers gets all trading pair ticker information including 24h volume
func (k *Kucoin) GetAllTickers(ctx context.Context) ([]TickerInfo, error) {
	resp := struct {
		Data struct {
			Time    uint64       `json:"time"` // TODO: find a way to convert it to time.Time
			Tickers []TickerInfo `json:"ticker"`
		} `json:"data"`
		Error
	}{}

	return resp.Data.Tickers, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetAllTickers, publicSpotRate, &resp)
}

// Get24hrStats get the statistics of the specified pair in the last 24 hours
func (k *Kucoin) Get24hrStats(ctx context.Context, pair string) (Stats24hrs, error) {
	resp := struct {
		Data Stats24hrs `json:"data"`
		Error
	}{}

	params := url.Values{}
	if pair == "" {
		return Stats24hrs{}, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGet24hrStats, params), publicSpotRate, &resp)
}

// GetMarketList get the transaction currency for the entire trading market
func (k *Kucoin) GetMarketList(ctx context.Context) ([]string, error) {
	resp := struct {
		Data []string `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetMarketList, publicSpotRate, &resp)
}

func processOB(ob [][2]string) ([]orderbook.Item, error) {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		amount, err := strconv.ParseFloat(ob[x][1], 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(ob[x][0], 64)
		if err != nil {
			return nil, err
		}
		o[x] = orderbook.Item{
			Price:  price,
			Amount: amount,
		}
	}
	return o, nil
}

func constructOrderbook(o *orderbookResponse) (s Orderbook, err error) {
	s.Bids, err = processOB(o.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(o.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = o.Data.Time.Time()
	return
}

// GetPartOrderbook20 gets orderbook for a specified pair with depth 20
func (k *Kucoin) GetPartOrderbook20(ctx context.Context, pair string) (Orderbook, error) {
	var o orderbookResponse
	params := url.Values{}
	if pair == "" {
		return Orderbook{}, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	err := k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetPartOrderbook20, params), publicSpotRate, &o)
	if err != nil {
		return Orderbook{}, err
	}
	return constructOrderbook(&o)
}

// GetPartOrderbook100 gets orderbook for a specified pair with depth 100
func (k *Kucoin) GetPartOrderbook100(ctx context.Context, pair string) (Orderbook, error) {
	var o orderbookResponse
	params := url.Values{}
	if pair == "" {
		return Orderbook{}, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	err := k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetPartOrderbook100, params), publicSpotRate, &o)
	if err != nil {
		return Orderbook{}, err
	}
	return constructOrderbook(&o)
}

// GetOrderbook gets full orderbook for a specified pair
func (k *Kucoin) GetOrderbook(ctx context.Context, pair string) (Orderbook, error) {
	var o orderbookResponse
	params := url.Values{}
	if pair == "" {
		return Orderbook{}, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	err := k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetOrderbook, params), nil, publicSpotRate, &o)
	if err != nil {
		return Orderbook{}, err
	}
	return constructOrderbook(&o)
}

// GetTradeHistory gets trade history of the specified pair
func (k *Kucoin) GetTradeHistory(ctx context.Context, pair string) ([]Trade, error) {
	resp := struct {
		Data []Trade `json:"data"`
		Error
	}{}

	params := url.Values{}
	if pair == "" {
		return nil, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetTradeHistory, params), publicSpotRate, &resp)
}

// GetKlines gets kline of the specified pair
func (k *Kucoin) GetKlines(ctx context.Context, pair, period string, start, end time.Time) ([]Kline, error) {
	resp := struct {
		Data [][7]string `json:"data"`
		Error
	}{}

	params := url.Values{}
	if pair == "" {
		return nil, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)

	if period == "" {
		return nil, errors.New("period can't be empty")
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
	err := k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetKlines, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	klines := make([]Kline, len(resp.Data))
	for i := range resp.Data {
		t, err := strconv.ParseInt(resp.Data[i][0], 10, 64)
		if err != nil {
			return nil, err
		}

		open, err := strconv.ParseFloat(resp.Data[i][1], 64)
		if err != nil {
			return nil, err
		}

		close, err := strconv.ParseFloat(resp.Data[i][2], 64)
		if err != nil {
			return nil, err
		}

		high, err := strconv.ParseFloat(resp.Data[i][3], 64)
		if err != nil {
			return nil, err
		}

		low, err := strconv.ParseFloat(resp.Data[i][4], 64)
		if err != nil {
			return nil, err
		}

		volume, err := strconv.ParseFloat(resp.Data[i][5], 64)
		if err != nil {
			return nil, err
		}

		amount, err := strconv.ParseFloat(resp.Data[i][6], 64)
		if err != nil {
			return nil, err
		}

		klines[i] = Kline{
			StartTime: time.Unix(t, 0),
			Open:      open,
			Close:     close,
			High:      high,
			Low:       low,
			Volume:    volume,
			Amount:    amount,
		}
	}
	return klines, nil
}

// GetCurrencies gets list of currencies
func (k *Kucoin) GetCurrencies(ctx context.Context) ([]Currency, error) {
	resp := struct {
		Data []Currency `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetCurrencies, publicSpotRate, &resp)
}

// GetCurrencies gets list of currencies
func (k *Kucoin) GetCurrency(ctx context.Context, currency, chain string) (CurrencyDetail, error) {
	resp := struct {
		Data CurrencyDetail `json:"data"`
		Error
	}{}

	if currency == "" {
		return CurrencyDetail{}, errors.New("currency can't be empty")
	}
	params := url.Values{}
	if chain != "" {
		params.Set("chain", chain)
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetCurrency+strings.ToUpper(currency), params), publicSpotRate, &resp)
}

// GetFiatPrice gets fiat prices of currencies, default base currency is USD
func (k *Kucoin) GetFiatPrice(ctx context.Context, base, currencies string) (map[string]string, error) {
	resp := struct {
		Data map[string]string `json:"data"`
		Error
	}{}

	params := url.Values{}
	if base != "" {
		params.Set("base", base)
	}
	if currencies != "" {
		params.Set("currencies", currencies)
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(kucoinGetFiatPrice, params), publicSpotRate, &resp)
}

// GetMarkPrice gets index price of the specified pair
func (k *Kucoin) GetMarkPrice(ctx context.Context, pair string) (MarkPrice, error) {
	resp := struct {
		Data MarkPrice `json:"data"`
		Error
	}{}

	if pair == "" {
		return MarkPrice{}, errors.New("pair can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, fmt.Sprintf(kucoinGetMarkPrice, pair), publicSpotRate, &resp)
}

// GetMarginConfiguration gets configure info of the margin
func (k *Kucoin) GetMarginConfiguration(ctx context.Context) (MarginConfiguration, error) {
	resp := struct {
		Data MarginConfiguration `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetMarginConfiguration, publicSpotRate, &resp)
}

// GetMarginAccount gets configure info of the margin
func (k *Kucoin) GetMarginAccount(ctx context.Context) (MarginAccounts, error) {
	resp := struct {
		Data MarginAccounts `json:"data"`
		Error
	}{}

	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, kucoinGetMarginAccount, nil, publicSpotRate, &resp)
}

// GetMarginRiskLimit gets cross/isolated margin risk limit, default model is cross margin
func (k *Kucoin) GetMarginRiskLimit(ctx context.Context, marginModel string) ([]MarginRiskLimit, error) {
	resp := struct {
		Data []MarginRiskLimit `json:"data"`
		Error
	}{}

	params := url.Values{}
	if marginModel != "" {
		params.Set("marginModel", marginModel)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetMarginRiskLimit, params), nil, publicSpotRate, &resp)
}

// PostBorrowOrder used to post borrow order
func (k *Kucoin) PostBorrowOrder(ctx context.Context, currency, orderType, term string, size, maxRate float64) (PostBorrowOrderResp, error) {
	resp := struct {
		Data PostBorrowOrderResp `json:"data"`
		Error
	}{}

	params := make(map[string]interface{})
	if currency == "" {
		return PostBorrowOrderResp{}, errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if orderType == "" {
		return PostBorrowOrderResp{}, errors.New("orderType can't be empty")
	}
	params["type"] = orderType
	if size == 0 {
		return PostBorrowOrderResp{}, errors.New("size can't be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)

	if maxRate != 0 {
		params["maxRate"] = strconv.FormatFloat(maxRate, 'f', -1, 64)
	}
	if term != "" {
		params["term"] = term
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, kucoinBorrowOrder, params, publicSpotRate, &resp)
}

// GetBorrowOrder gets borrow order information
func (k *Kucoin) GetBorrowOrder(ctx context.Context, orderID string) (BorrowOrder, error) {
	resp := struct {
		Data BorrowOrder `json:"data"`
		Error
	}{}

	params := url.Values{}

	if orderID == "" {
		return resp.Data, errors.New("empty orderID")
	}
	params.Set("orderId", orderID)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinBorrowOrder, params), nil, publicSpotRate, &resp)
}

// GetOutstandingRecord gets outstanding record information
func (k *Kucoin) GetOutstandingRecord(ctx context.Context, currency string) ([]OutstandingRecord, error) {
	resp := struct {
		Data []OutstandingRecord `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetOutstandingRecord, params), nil, publicSpotRate, &resp)
}

// GetRepaidRecord gets repaid record information
func (k *Kucoin) GetRepaidRecord(ctx context.Context, currency string) ([]RepaidRecord, error) {
	resp := struct {
		Data []RepaidRecord `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetRepaidRecord, params), nil, publicSpotRate, &resp)
}

// OneClickRepayment used to compplete repayment in single go
func (k *Kucoin) OneClickRepayment(ctx context.Context, currency, sequence string, size float64) error {
	resp := struct {
		Error
	}{}

	params := make(map[string]interface{})
	if currency == "" {
		return errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if sequence == "" {
		return errors.New("sequence can't be empty")
	}
	params["sequence"] = sequence
	if size == 0 {
		return errors.New("size can't be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, kucoinOneClickRepayment, params, publicSpotRate, &resp)
}

// SingleOrderRepayment used to repay single order
func (k *Kucoin) SingleOrderRepayment(ctx context.Context, currency, tradeID string, size float64) error {
	resp := struct {
		Error
	}{}

	params := make(map[string]interface{})
	if currency == "" {
		return errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if tradeID == "" {
		return errors.New("tradeId can't be empty")
	}
	params["tradeId"] = tradeID
	if size == 0 {
		return errors.New("size can't be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	return k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, kucoinRepaySingleOrder, params, publicSpotRate, &resp)
}

// PostLendOrder used to create lend order
func (k *Kucoin) PostLendOrder(ctx context.Context, currency string, dailyIntRate, size float64, term int64) (string, error) {
	resp := struct {
		OrderID string `json:"orderId"`
		Error
	}{}

	params := make(map[string]interface{})
	if currency == "" {
		return "", errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if dailyIntRate == 0 {
		return "", errors.New("dailyIntRate can't be zero")
	}
	params["dailyIntRate"] = strconv.FormatFloat(dailyIntRate, 'f', -1, 64)
	if size == 0 {
		return "", errors.New("size can't be zero")
	}
	params["size"] = strconv.FormatFloat(size, 'f', -1, 64)
	if term == 0 {
		return "", errors.New("term can't be zero")
	}
	params["term"] = strconv.FormatInt(term, 10)
	return resp.OrderID, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, kucoinPostLendOrder, params, publicSpotRate, &resp)
}

// CancelLendOrder used to cancel lend order
func (k *Kucoin) CancelLendOrder(ctx context.Context, orderID string) error {
	resp := struct {
		Error
	}{}

	return k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, fmt.Sprintf(kucoinCancelLendOrder, orderID), nil, publicSpotRate, &resp)
}

// SetAutoLend used to set up the automatic lending for a specified currency
func (k *Kucoin) SetAutoLend(ctx context.Context, currency string, dailyIntRate, retainSize float64, term int64, isEnable bool) error {
	resp := struct {
		Error
	}{}

	params := make(map[string]interface{})
	if currency == "" {
		return errors.New("currency can't be empty")
	}
	params["currency"] = currency
	if dailyIntRate == 0 {
		return errors.New("dailyIntRate can't be zero")
	}
	params["dailyIntRate"] = strconv.FormatFloat(dailyIntRate, 'f', -1, 64)
	if retainSize == 0 {
		return errors.New("retainSize can't be zero")
	}
	params["retainSize"] = strconv.FormatFloat(retainSize, 'f', -1, 64)
	if term == 0 {
		return errors.New("term can't be zero")
	}
	params["term"] = strconv.FormatInt(term, 10)
	params["isEnable"] = isEnable
	return k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, kucoinSetAutoLend, params, publicSpotRate, &resp)
}

// GetActiveOrder gets active lend orders
func (k *Kucoin) GetActiveOrder(ctx context.Context, currency string) ([]LendOrder, error) {
	resp := struct {
		Data []LendOrder `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetActiveOrder, params), nil, publicSpotRate, &resp)
}

// GetLendHistory gets lend orders
func (k *Kucoin) GetLendHistory(ctx context.Context, currency string) ([]LendOrderHistory, error) {
	resp := struct {
		Data []LendOrderHistory `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetLendHistory, params), nil, publicSpotRate, &resp)
}

// GetUnsettleLendOrder gets outstanding lend order list
func (k *Kucoin) GetUnsettleLendOrder(ctx context.Context, currency string) ([]UnsettleLendOrder, error) {
	resp := struct {
		Data []UnsettleLendOrder `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetUnsettleLendOrder, params), nil, publicSpotRate, &resp)
}

// GetSettleLendOrder gets settle lend orders
func (k *Kucoin) GetSettleLendOrder(ctx context.Context, currency string) ([]SettleLendOrder, error) {
	resp := struct {
		Data []SettleLendOrder `json:"items"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetSettleLendOrder, params), nil, publicSpotRate, &resp)
}

// GetAccountLendRecord get the lending history of the main account
func (k *Kucoin) GetAccountLendRecord(ctx context.Context, currency string) ([]LendRecord, error) {
	resp := struct {
		Data []LendRecord `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetAccountLendRecord, params), nil, publicSpotRate, &resp)
}

// GetLendingMarketData get the lending market data
func (k *Kucoin) GetLendingMarketData(ctx context.Context, currency string, term int64) ([]LendMarketData, error) {
	resp := struct {
		Data []LendMarketData `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency == "" {
		return resp.Data, errors.New("currency can't be empty")
	}
	params.Set("currency", currency)
	if term != 0 {
		params.Set("term", strconv.FormatInt(term, 10))
	}
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetLendingMarketData, params), nil, publicSpotRate, &resp)
}

// GetMarginTradeData get the last 300 fills in the lending and borrowing market
func (k *Kucoin) GetMarginTradeData(ctx context.Context, currency string) ([]MarginTradeData, error) {
	resp := struct {
		Data []MarginTradeData `json:"data"`
		Error
	}{}

	params := url.Values{}
	if currency == "" {
		return resp.Data, errors.New("currency can't be empty")
	}
	params.Set("currency", currency)
	return resp.Data, k.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues(kucoinGetMarginTradeData, params), nil, publicSpotRate, &resp)
}

// GetCurrentServerTime gets the server time
func (k *Kucoin) GetCurrentServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Timestamp kucoinTimeMilliSec `json:"data"`
		Error
	}{}
	return resp.Timestamp.Time(), k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetServerTime, publicSpotRate, &resp)
}

// GetServiceStatus gets the service status
func (k *Kucoin) GetServiceStatus(ctx context.Context) (string, string, error) {
	resp := struct {
		Data struct {
			Status  string `json:"status"`
			Message string `json:"msg"`
		} `json:"data"`
		Error
	}{}
	return resp.Data.Status, resp.Data.Message, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetServiceStatus, publicSpotRate, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (k *Kucoin) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result UnmarshalTo) error {
	endpointPath, err := k.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	err = k.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + path,
			Result:        result,
			Verbose:       k.Verbose,
			HTTPDebugging: k.HTTPDebugging,
			HTTPRecording: k.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	return result.GetError()
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Request parameters are added to path variable for GET and DELETE request and for other requests its passed in params variable
func (k *Kucoin) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params map[string]interface{}, f request.EndpointLimit, result UnmarshalTo) error {
	creds, err := k.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpointPath, err := k.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	if result == nil {
		result = &Error{}
	}

	err = k.SendPayload(ctx, f, func() (*request.Item, error) {
		var (
			body    io.Reader
			payload []byte
			err     error
		)
		if params != nil && len(params) != 0 {
			payload, err = json.Marshal(params)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		timeStamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		var signHash, passPhraseHash []byte
		signHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(timeStamp+method+path+string(payload)), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		passPhraseHash, err = crypto.GetHMAC(crypto.HashSHA256, []byte(creds.OneTimePassword), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        crypto.Base64Encode(signHash),
			"KC-API-TIMESTAMP":   timeStamp,
			"KC-API-PASSPHRASE":  crypto.Base64Encode(passPhraseHash), // TODO: need pass phrase here!,
			"KC-API-KEY-VERSION": kucoinAPIKeyVersion,
			"Content-Type":       "application/json",
		}

		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          body,
			Result:        &result,
			AuthRequest:   true,
			Verbose:       k.Verbose,
			HTTPDebugging: k.HTTPDebugging,
			HTTPRecording: k.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	return result.GetError()
}
