package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// The NFT endpoints are no longer in Binance's API reference, but no changelog retires them and the routes still
// answer; they follow the last published reference, binance-api-swagger's spot_api.yaml

// GetNFTTransactionHistory calls Get NFT Transaction History for a window of at most 90 days; without a window the
// API returns the last 7 days
func (e *Exchange) GetNFTTransactionHistory(ctx context.Context, arg *NFTTransactionHistoryRequest) (*NFTTransactionHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderType > 4 {
		return nil, fmt.Errorf("%w: %d, want 0 purchase, 1 sell, 2 royalty income, 3 primary market or 4 mint fee", order.ErrUnsupportedOrderType, arg.OrderType)
	}
	params, err := nftHistoryParams(arg.StartTime, arg.EndTime, arg.Limit, arg.Page)
	if err != nil {
		return nil, err
	}
	params.Set("orderType", strconv.FormatUint(arg.OrderType, 10))
	var resp *NFTTransactionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/nft/history/transactions", params, nftRate, &resp)
}

// GetNFTDepositHistory calls Get NFT Deposit History for a window of at most 90 days; without a window the API
// returns the last 7 days
func (e *Exchange) GetNFTDepositHistory(ctx context.Context, arg *NFTHistoryRequest) (*NFTDepositHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := nftHistoryParams(arg.StartTime, arg.EndTime, arg.Limit, arg.Page)
	if err != nil {
		return nil, err
	}
	var resp *NFTDepositHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/nft/history/deposit", params, nftRate, &resp)
}

// GetNFTWithdrawalHistory calls Get NFT Withdraw History for a window of at most 90 days; without a window the API
// returns the last 7 days
func (e *Exchange) GetNFTWithdrawalHistory(ctx context.Context, arg *NFTHistoryRequest) (*NFTWithdrawalHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := nftHistoryParams(arg.StartTime, arg.EndTime, arg.Limit, arg.Page)
	if err != nil {
		return nil, err
	}
	var resp *NFTWithdrawalHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/nft/history/withdraw", params, nftRate, &resp)
}

// GetNFTAsset calls Get NFT Asset, returning a page of the account's NFTs; limit is at most 50, the default
func (e *Exchange) GetNFTAsset(ctx context.Context, limit, page uint64) (*NFTAssetsResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var resp *NFTAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/nft/user/getAsset", params, nftRate, &resp)
}

// nftHistoryParams builds the window and paging parameters the NFT history queries share
func nftHistoryParams(startTime, endTime time.Time, limit, page uint64) (url.Values, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	return params, nil
}
