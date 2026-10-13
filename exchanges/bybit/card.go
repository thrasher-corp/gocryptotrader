package bybit

import (
	"context"
	"errors"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Bybit Card errors
var (
	errCardQueryTypeRequired           = errors.New("query type required without a transaction ID or order number")
	errCardBusinessTransactionIDNotSet = errors.New("business transaction ID not set")
)

// GetCardAssetRecords returns a page of Bybit Card transactions (Query Asset Records)
func (e *Exchange) GetCardAssetRecords(ctx context.Context, req *CardAssetRecordsRequest) (*CardAssetRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" && req.TransactionID == "" && req.OrderNumber == "" {
		return nil, errCardQueryTypeRequired
	}
	if !req.CreateBeginTime.IsZero() && !req.CreateEndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.CreateBeginTime, req.CreateEndTime); err != nil {
			return nil, err
		}
	}
	body := map[string]any{}
	if req.StatusCode != "" {
		body["statusCode"] = req.StatusCode
	}
	if req.Limit != 0 {
		body["limit"] = req.Limit
	}
	if req.Page != 0 {
		body["page"] = req.Page
	}
	if req.PAN4 != "" {
		body["pan4"] = req.PAN4
	}
	if !req.CreateBeginTime.IsZero() {
		body["createBeginTime"] = req.CreateBeginTime.UnixMilli()
	}
	if !req.CreateEndTime.IsZero() {
		body["createEndTime"] = req.CreateEndTime.UnixMilli()
	}
	if req.MerchantName != "" {
		body["merchName"] = req.MerchantName
	}
	if req.Type != "" {
		body["type"] = req.Type
	}
	if req.TransactionID != "" {
		body["txnId"] = req.TransactionID
	}
	if req.CardToken != "" {
		body["cardToken"] = req.CardToken
	}
	if req.OrderNumber != "" {
		body["orderNo"] = req.OrderNumber
	}
	var resp *CardAssetRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/transaction/query-asset-records", nil, body, &resp, defaultEPL)
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
	body := map[string]any{}
	if req.Type != "" {
		body["type"] = req.Type
	}
	if req.PageSize != 0 {
		body["pageSize"] = req.PageSize
	}
	if req.PageNumber != 0 {
		body["pageNo"] = req.PageNumber
	}
	if !req.StartTime.IsZero() {
		body["startTime"] = req.StartTime.UnixMilli()
	}
	if !req.EndTime.IsZero() {
		body["endTime"] = req.EndTime.UnixMilli()
	}
	if req.OutOrderID != "" {
		body["outOrderId"] = req.OutOrderID
	}
	if req.BusinessID != "" {
		body["bizId"] = req.BusinessID
	}
	if req.BusinessTransactionID != "" {
		body["bizTxnId"] = req.BusinessTransactionID
	}
	if req.Side != "" {
		body["side"] = req.Side
	}
	var resp *CardPointRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/points/records", nil, body, &resp, defaultEPL)
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
	body := map[string]any{}
	if req.PageNumber != 0 {
		body["pageNo"] = req.PageNumber
	}
	if req.PageSize != 0 {
		body["pageSize"] = req.PageSize
	}
	if req.ItemType != 0 {
		body["itemType"] = req.ItemType
	}
	if req.ItemBusinessType != 0 {
		body["itemBizType"] = req.ItemBusinessType
	}
	if req.OrderBy != 0 {
		body["orderBy"] = req.OrderBy
	}
	if req.Ascending != nil {
		body["asc"] = *req.Ascending
	}
	if req.Source != 0 {
		body["source"] = req.Source
	}
	var resp *CardMallItemListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/mall/item/list", nil, body, &resp, defaultEPL)
}

// GetCardCashbackDetail returns the cashback of a Bybit Card reward order (Query Cashback Detail)
func (e *Exchange) GetCardCashbackDetail(ctx context.Context, businessTransactionID string) (*CardCashbackDetailResponse, error) {
	if businessTransactionID == "" {
		return nil, errCardBusinessTransactionIDNotSet
	}
	var resp *CardCashbackDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/card/reward/point/cashback/detail", nil, map[string]any{"bizTxnId": businessTransactionID}, &resp, defaultEPL)
}
