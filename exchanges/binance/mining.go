package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetAccountList calls Account List, returning the hourly and daily hashrate of a mining account
func (e *Exchange) GetAccountList(ctx context.Context, algorithm, userName string) ([]MiningAccountHashrate, error) {
	params, err := miningAccountParams(algorithm, userName)
	if err != nil {
		return nil, err
	}
	var resp *MiningResponse[[]MiningAccountHashrate]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/statistics/user/list", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// AcquiringAlgorithm calls Acquiring Algorithm, returning the mining algorithms
func (e *Exchange) AcquiringAlgorithm(ctx context.Context) ([]MiningAlgorithm, error) {
	var resp *MiningResponse[[]MiningAlgorithm]
	if err := e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/pub/algoList", sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetCoinNames calls Acquiring CoinName, returning the mineable coins and their algorithms
func (e *Exchange) GetCoinNames(ctx context.Context) ([]MiningCoin, error) {
	var resp *MiningResponse[[]MiningCoin]
	if err := e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/pub/coinList", sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CancelHashrateResaleConfiguration calls Cancel hashrate resale configuration, returning whether it was cancelled
func (e *Exchange) CancelHashrateResaleConfiguration(ctx context.Context, configID uint64, userName string) (bool, error) {
	if configID == 0 {
		return false, errConfigIDRequired
	}
	if userName == "" {
		return false, errUsernameRequired
	}
	params := url.Values{}
	params.Set("configId", strconv.FormatUint(configID, 10))
	params.Set("userName", userName)
	var resp *MiningResponse[bool]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/mining/hash-transfer/config/cancel", params, sapiDefaultRate, &resp); err != nil {
		return false, err
	}
	return resp.Data, nil
}

// GetEarningList calls Earnings List
func (e *Exchange) GetEarningList(ctx context.Context, arg *MiningPaymentRequest) (*MiningEarningsResponse, error) {
	params, err := miningPaymentParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *MiningResponse[*MiningEarningsResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/payment/list", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetExtraBonusList calls Extra Bonus List
func (e *Exchange) GetExtraBonusList(ctx context.Context, arg *MiningPaymentRequest) (*MiningExtraBonusResponse, error) {
	params, err := miningPaymentParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *MiningResponse[*MiningExtraBonusResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/payment/other", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// miningPaymentParams builds the parameters Earnings List and Extra Bonus List share
func miningPaymentParams(arg *MiningPaymentRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := miningAccountParams(arg.Algorithm, arg.UserName)
	if err != nil {
		return nil, err
	}
	if !arg.StartDate.IsZero() && !arg.EndDate.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartDate, arg.EndDate); err != nil {
			return nil, err
		}
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.Upper().String())
	}
	if !arg.StartDate.IsZero() {
		params.Set("startDate", strconv.FormatInt(arg.StartDate.UnixMilli(), 10))
	}
	if !arg.EndDate.IsZero() {
		params.Set("endDate", strconv.FormatInt(arg.EndDate.UnixMilli(), 10))
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	return params, nil
}

// GetHashrateResaleDetail calls Hashrate Resale Detail
func (e *Exchange) GetHashrateResaleDetail(ctx context.Context, configID, pageIndex, pageSize uint64) (*HashrateResaleDetailResponse, error) {
	if configID == 0 {
		return nil, errConfigIDRequired
	}
	params := url.Values{}
	params.Set("configId", strconv.FormatUint(configID, 10))
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *MiningResponse[*HashrateResaleDetailResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/hash-transfer/profit/details", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetHashrateResaleList calls Hashrate Resale List
func (e *Exchange) GetHashrateResaleList(ctx context.Context, pageIndex, pageSize uint64) (*HashrateResaleListResponse, error) {
	params := url.Values{}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(pageSize, 10))
	}
	var resp *MiningResponse[*HashrateResaleListResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/hash-transfer/config/details/list", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// RequestHashrateResale calls Hashrate Resale Request, returning the resale configuration ID
func (e *Exchange) RequestHashrateResale(ctx context.Context, arg *HashrateResaleRequest) (uint64, error) {
	if err := common.NilGuard(arg); err != nil {
		return 0, err
	}
	if arg.UserName == "" {
		return 0, errUsernameRequired
	}
	if arg.Algorithm == "" {
		return 0, errTransferAlgorithmRequired
	}
	if err := common.StartEndTimeCheck(arg.StartDate, arg.EndDate); err != nil {
		return 0, err
	}
	if arg.ToPoolUser == "" {
		return 0, fmt.Errorf("%w: receiving mining account is required", errAccountRequired)
	}
	if arg.HashRate == 0 {
		return 0, errHashRateRequired
	}
	params := url.Values{}
	params.Set("userName", arg.UserName)
	params.Set("algo", arg.Algorithm)
	params.Set("endDate", strconv.FormatInt(arg.EndDate.UnixMilli(), 10))
	params.Set("startDate", strconv.FormatInt(arg.StartDate.UnixMilli(), 10))
	params.Set("toPoolUser", arg.ToPoolUser)
	params.Set("hashRate", strconv.FormatUint(arg.HashRate, 10))
	var resp *MiningResponse[uint64]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/mining/hash-transfer/config", params, sapiDefaultRate, &resp); err != nil {
		return 0, err
	}
	return resp.Data, nil
}

// GetMiningAccountEarning calls Mining Account Earning
func (e *Exchange) GetMiningAccountEarning(ctx context.Context, arg *MiningAccountEarningRequest) (*MiningAccountEarningResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if !arg.StartDate.IsZero() && !arg.EndDate.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartDate, arg.EndDate); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("algo", arg.Algorithm)
	if !arg.StartDate.IsZero() {
		params.Set("startDate", strconv.FormatInt(arg.StartDate.UnixMilli(), 10))
	}
	if !arg.EndDate.IsZero() {
		params.Set("endDate", strconv.FormatInt(arg.EndDate.UnixMilli(), 10))
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize > 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *MiningResponse[*MiningAccountEarningResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/payment/uid", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetDetailMinerList calls Request for Detail Miner List, returning a miner's hashrate history
func (e *Exchange) GetDetailMinerList(ctx context.Context, algorithm, userName, workerName string) ([]MinerDetail, error) {
	params, err := miningAccountParams(algorithm, userName)
	if err != nil {
		return nil, err
	}
	if workerName == "" {
		return nil, fmt.Errorf("%w: miner name is required", errNameRequired)
	}
	params.Set("workerName", workerName)
	var resp *MiningResponse[[]MinerDetail]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/worker/detail", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetMinersList calls Request for Miner List
func (e *Exchange) GetMinersList(ctx context.Context, arg *MinerListRequest) (*MinerListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := miningAccountParams(arg.Algorithm, arg.UserName)
	if err != nil {
		return nil, err
	}
	if arg.PageIndex > 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.Descending {
		params.Set("sort", "1")
	}
	if arg.SortColumn > 0 {
		params.Set("sortColumn", strconv.FormatUint(arg.SortColumn, 10))
	}
	if arg.WorkerStatus > 0 {
		params.Set("workerStatus", strconv.FormatUint(arg.WorkerStatus, 10))
	}
	var resp *MiningResponse[*MinerListResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/worker/list", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// StatisticsList calls Statistic List, returning a mining account's hashrate and earnings summary
func (e *Exchange) StatisticsList(ctx context.Context, algorithm, userName string) (*MiningStatisticsResponse, error) {
	params, err := miningAccountParams(algorithm, userName)
	if err != nil {
		return nil, err
	}
	var resp *MiningResponse[*MiningStatisticsResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/mining/statistics/user/status", params, sapiDefaultRate, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// miningAccountParams builds the algorithm and mining account parameters most mining endpoints require
func miningAccountParams(algorithm, userName string) (url.Values, error) {
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("algo", algorithm)
	params.Set("userName", userName)
	return params, nil
}
