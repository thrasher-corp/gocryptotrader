package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Bybit Card errors
var (
	errCardQueryTypeRequired           = errors.New("query type required without a transaction ID or order number")
	errCardBusinessTransactionIDNotSet = errors.New("business transaction ID not set")
	errCardTransactionIDAndOrderNotSet = errors.New("query type needs both a transaction ID and an order number")
)

// GetCardAssetRecords returns a page of Bybit Card transactions (Query Asset Records). The Bybit Card endpoints are
// POST requests that take their parameters in the query string, as every example on their pages sends them
func (e *Exchange) GetCardAssetRecords(ctx context.Context, req *CardAssetRecordsRequest) (*CardAssetRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" && req.TransactionID == "" && req.OrderNumber == "" {
		return nil, errCardQueryTypeRequired
	}
	if (req.Type == "SIDE_QUERY_FINANCIAL" || req.Type == "SIDE_QUERY_REFUND") && (req.TransactionID == "" || req.OrderNumber == "") {
		return nil, fmt.Errorf("%w: %s", errCardTransactionIDAndOrderNotSet, req.Type)
	}
	if !req.CreateBeginTime.IsZero() && !req.CreateEndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.CreateBeginTime, req.CreateEndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.StatusCode != "" {
		params.Set("statusCode", req.StatusCode)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PAN4 != "" {
		params.Set("pan4", req.PAN4)
	}
	if !req.CreateBeginTime.IsZero() {
		params.Set("createBeginTime", strconv.FormatInt(req.CreateBeginTime.UnixMilli(), 10))
	}
	if !req.CreateEndTime.IsZero() {
		params.Set("createEndTime", strconv.FormatInt(req.CreateEndTime.UnixMilli(), 10))
	}
	if req.MerchantName != "" {
		params.Set("merchName", req.MerchantName)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.TransactionID != "" {
		params.Set("txnId", req.TransactionID)
	}
	if req.CardToken != "" {
		params.Set("cardToken", req.CardToken)
	}
	if req.OrderNumber != "" {
		params.Set("orderNo", req.OrderNumber)
	}
	var resp *CardAssetRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/transaction/query-asset-records", params, nil, &resp, defaultEPL)
}

// GetCardPointBalance returns the Bybit Card reward point balance (Query Point Balance)
func (e *Exchange) GetCardPointBalance(ctx context.Context) (*CardPointBalanceResponse, error) {
	var resp *CardPointBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/points/balance", nil, nil, &resp, defaultEPL)
}

// GetCardPointRecords returns a page of Bybit Card reward point records (Query Point Records)
func (e *Exchange) GetCardPointRecords(ctx context.Context, req *CardPointRecordsRequest) (*CardPointRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.PageNumber != 0 {
		params.Set("pageNo", strconv.FormatUint(req.PageNumber, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.OutOrderID != "" {
		params.Set("outOrderId", req.OutOrderID)
	}
	if req.BusinessID != "" {
		params.Set("bizId", req.BusinessID)
	}
	if req.BusinessTransactionID != "" {
		params.Set("bizTxnId", req.BusinessTransactionID)
	}
	if req.Side != "" {
		params.Set("side", req.Side)
	}
	var resp *CardPointRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/points/records", params, nil, &resp, defaultEPL)
}

// GetCardTierInfo returns the Bybit Card reward tier and spending limit (Query Tier Info)
func (e *Exchange) GetCardTierInfo(ctx context.Context) (*CardTierInfoResponse, error) {
	var resp *CardTierInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/points/tier", nil, nil, &resp, defaultEPL)
}

// GetCardMallItemList returns a page of the items in the Bybit Card reward mall (Query Mall Item List)
func (e *Exchange) GetCardMallItemList(ctx context.Context, req *CardMallItemListRequest) (*CardMallItemListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.PageNumber != 0 {
		params.Set("pageNo", strconv.FormatUint(req.PageNumber, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	if req.ItemType != 0 {
		params.Set("itemType", strconv.FormatUint(req.ItemType, 10))
	}
	if req.ItemBusinessType != 0 {
		params.Set("itemBizType", strconv.FormatUint(req.ItemBusinessType, 10))
	}
	if req.OrderBy != 0 {
		params.Set("orderBy", strconv.FormatUint(req.OrderBy, 10))
	}
	if req.Ascending != nil {
		params.Set("asc", strconv.FormatBool(*req.Ascending))
	}
	if req.Source != 0 {
		params.Set("source", strconv.FormatUint(req.Source, 10))
	}
	var resp *CardMallItemListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/mall/item/list", params, nil, &resp, defaultEPL)
}

// GetCardCashbackDetail returns the cashback of a Bybit Card reward order (Query Cashback Detail)
func (e *Exchange) GetCardCashbackDetail(ctx context.Context, businessTransactionID string) (*CardCashbackDetailResponse, error) {
	if businessTransactionID == "" {
		return nil, errCardBusinessTransactionIDNotSet
	}
	params := url.Values{}
	params.Set("bizTxnId", businessTransactionID)
	var resp *CardCashbackDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/point/cashback/detail", params, nil, &resp, defaultEPL)
}
