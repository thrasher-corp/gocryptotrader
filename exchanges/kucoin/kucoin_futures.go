package kucoin

import (
	"context"
	"fmt"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

const (
	kucoinFuturesAPIURL = "https://api-futures.kucoin.com"

	kucoinFuturesOpenContracts     = "/api/v2/contracts/active"
	kucoinFuturesContract          = "/api/v2/contracts/%s"
	kucoinFuturesContractRiskLimit = "/api/v2/contracts/risk-limit/%s"
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

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesContract, symbol), publicSpotRate, &resp)
}

// GetFuturesContractRiskLimit get contract risk limit list
func (k *Kucoin) GetFuturesContractRiskLimit(ctx context.Context, symbol string) (Contract, error) {
	resp := struct {
		Data Contract `json:"data"`
		Error
	}{}

	return resp.Data, k.SendHTTPRequest(ctx, exchange.RestFutures, fmt.Sprintf(kucoinFuturesContractRiskLimit, symbol), publicSpotRate, &resp)
}
