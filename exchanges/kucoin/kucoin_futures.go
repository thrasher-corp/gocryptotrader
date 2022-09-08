package kucoin

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com"

	kucoinFuturesOpenContracts         = "/api/v2/contracts/active"
	kucoinFuturesContract              = "/api/v2/contracts/%s"
	kucoinFuturesRiskLimit             = "/api/v2/contracts/risk-limit/%s"
	kucoinFuturesKline                 = "/api/v2/kline/query"
	kucoinFuturesGetFundingRate        = "/api/v2/contract/%s/funding-rates"
	kucoinFuturesGetCurrentFundingRate = "/api/v2/funding-rate/%s/current"
	kucoinFuturesGetContractMarkPrice  = "/api/v2/mark-price/%s/current"
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
	if !common.StringDataContains(validGranularity, granularity) {
		return nil, errors.New("invalid granularity")
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
		tStr, ok := resp.Data[i][0].(string)
		if !ok {
			return nil, common.GetAssertError("string", resp.Data[i][0])
		}
		tInMilliSec, err := strconv.ParseInt(tStr, 10, 64)
		if err != nil {
			return nil, err
		}
		kline[i].StartTime = time.UnixMilli(tInMilliSec)

		openPrice, ok := resp.Data[i][1].(float64)
		if !ok {
			return nil, common.GetAssertError("float64", resp.Data[i][1])
		}
		kline[i].Open = openPrice

		maxPrice, ok := resp.Data[i][2].(float64)
		if !ok {
			return nil, common.GetAssertError("float64", resp.Data[i][2])
		}
		kline[i].High = maxPrice

		minPrice, ok := resp.Data[i][3].(float64)
		if !ok {
			return nil, common.GetAssertError("float64", resp.Data[i][3])
		}
		kline[i].Low = minPrice

		closePrice, ok := resp.Data[i][4].(float64)
		if !ok {
			return nil, common.GetAssertError("float64", resp.Data[i][4])
		}
		kline[i].Close = closePrice

		volume, ok := resp.Data[i][5].(float64)
		if !ok {
			return nil, common.GetAssertError("float64", resp.Data[i][5])
		}
		kline[i].Volume = volume
	}
	return kline, nil
}

// GetFuturesFundingRate get funding rate list
func (k *Kucoin) GetFuturesFundingRate(ctx context.Context, symbol string, startAt, endAt, offSet time.Time, limit int64) ([]FuturesFundingRate, error) {
	resp := struct {
		Data struct {
			List    []FuturesFundingRate `json:"dataList"`
			HasMore bool                 `json:"hasMore"`
		} `json:"data"`
		Error
	}{}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.List, errors.New("symbol can't be empty")
	}
	params.Set("symbol", symbol)
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if !offSet.IsZero() {
		params.Set("offset", strconv.FormatInt(offSet.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data.List, k.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues(fmt.Sprintf(kucoinFuturesGetFundingRate, symbol), params), publicSpotRate, &resp)
}

// GetFuturesCurrentFundingRate get current funding rate
func (k *Kucoin) GetFuturesCurrentFundingRate(ctx context.Context, symbol string) (FuturesFundingRate, error) {
	resp := struct {
		Data FuturesFundingRate `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesGetCurrentFundingRate, symbol), publicSpotRate, &resp)
}

// GetFuturesCurrentFundingRate get current funding rate
func (k *Kucoin) GetFuturesContractMarkPrice(ctx context.Context, symbol string) (FuturesMarkPrice, error) {
	resp := struct {
		Data FuturesMarkPrice `json:"data"`
		Error
	}{}

	if symbol == "" {
		return resp.Data, errors.New("symbol can't be empty")
	}
	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesGetContractMarkPrice, symbol), publicSpotRate, &resp)
}
