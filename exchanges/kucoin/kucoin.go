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

	// Authenticated endpoints
	kucoinGetOrderbook       = "/api/v3/market/orderbook/level2"
	kucoinGetMarginAccount   = "/api/v1/margin/account"
	kucoinGetMarginRiskLimit = "/api/v1/risk/limit/strategy"
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
