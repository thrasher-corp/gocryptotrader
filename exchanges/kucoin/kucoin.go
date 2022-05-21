package kucoin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	kucoinGetSymbols    = "/api/v1/symbols"
	kucoinGetTicker     = "/api/v1/market/orderbook/level1"
	kucoinGetAllTickers = "/api/v1/market/allTickers"
	kucoinGet24hrStats  = "/api/v1/market/stats"
	kucoinGetMarketList = "/api/v1/markets"

	// Authenticated endpoints
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
	path := common.EncodeURLValues(kucoinGetSymbols, params)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
}

// GetTicker gets pair ticker information
func (k *Kucoin) GetTicker(ctx context.Context, pair string) (Ticker, error) {
	resp := struct {
		Data Ticker `json:"data"`
		Error
	}{}

	params := url.Values{}
	if pair != "" {
		return Ticker{}, errors.New("pair can't be empty") // TODO: error as constant
	}
	params.Set("symbol", pair)
	path := common.EncodeURLValues(kucoinGetTicker, params)
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
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
	if pair != "" {
		return Stats24hrs{}, errors.New("pair can't be empty")
	}
	params.Set("symbol", pair)
	path := common.EncodeURLValues(kucoinGet24hrStats, params)

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
}

// GetMarketList get the transaction currency for the entire trading market
func (k *Kucoin) GetMarketList(ctx context.Context) ([]string, error) {
	resp := struct {
		Data []string `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestSpot, kucoinGetMarketList, publicSpotRate, &resp)
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
		var body io.Reader
		var payload []byte
		if len(params) != 0 {
			payload, err = json.Marshal(params)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		signature := strconv.FormatInt(time.Now().UnixMilli(), 10) + method + path + string(payload)
		var hmacSigned []byte
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        crypto.HexEncodeToString(hmacSigned),
			"KC-API-TIMESTAMP":   strconv.FormatInt(time.Now().UnixMilli(), 10),
			"KC-API-PASSPHRASE":  creds.OneTimePassword, // TODO: need pass phrase here!,
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
