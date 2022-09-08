package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com"

	kucoinFuturesOpenContracts = "/api/v2/contracts/active"
	kucoinFuturesContract      = "/api/v2/contracts/%s"
	kucoinFuturesRiskLimit     = "/api/v2/contracts/risk-limit/%s"
	kucoinFuturesKline         = "/api/v2/kline/query"
)

// GetFuturesOpenContracts gets all open futures contract with its details
func (k *Kucoin) GetFuturesOpenContracts(ctx context.Context) ([]Contract, error) {
	resp := struct {
		Data []Contract `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, kucoinFuturesOpenContracts, publicSpotRate, &resp)
}

// GetFuturesContract get contract details
func (k *Kucoin) GetFuturesContract(ctx context.Context, symbol string) (Contract, error) {
	resp := struct {
		Data Contract `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesContract, symbol), publicSpotRate, &resp)
}

// GetFuturesRiskLimit get contract risk limit list
func (k *Kucoin) GetFuturesRiskLimit(ctx context.Context, symbol string) ([]RiskLimitInfo, error) {
	resp := struct {
		Data []RiskLimitInfo `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesRiskLimit, symbol), publicSpotRate, &resp)
}

// GetFuturesKline get contract's kline data
func (k *Kucoin) GetFuturesKline(ctx context.Context, granularity, symbol string, from, to time.Time) ([]FuturesKline, error) {
	resp := struct {
		Data [][]interface{} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if granularity == "" {
		return nil, errors.New("granularity can't be empty")
	}
	params.Set("granularity", granularity)
	if symbol == "" {
		return nil, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(from.UnixMilli(), 10))
	}

	err := k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(kucoinFuturesKline, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	kline := make([]FuturesKline, len(resp.Data))
	for i := range resp.Data {

		t, ok := string.(resp.Data[i][0])
		if !ok {
			return nil, fmt.Errorf("%s: GetFuturesKline failed while type casting %v to string", k.Name, resp.Data[i][0])
		}
		
		t, err := strconv.ParseInt(resp.Data[i][0], 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return kline, 
}
