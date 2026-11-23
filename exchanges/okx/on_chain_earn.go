package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var errMissingRequiredParameter = errors.New("missing required parameter")

// GetOnChainEarnOffers returns the on-chain earn products available to purchase
func (e *Exchange) GetOnChainEarnOffers(ctx context.Context, arg *OnChainEarnOffersRequest) ([]OnChainEarnOfferResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.ProductID != "" {
		params.Set("productId", arg.ProductID)
	}
	if arg.ProtocolType != "" {
		params.Set("protocolType", arg.ProtocolType)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	var resp []OnChainEarnOfferResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getOfferEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/offers", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// PurchaseOnChainEarn invests funding account assets in an on-chain earn product
func (e *Exchange) PurchaseOnChainEarn(ctx context.Context, arg *OnChainEarnPurchaseRequest) (*OnChainEarnOrderIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ProductID == "" {
		return nil, fmt.Errorf("%w: productId", errMissingRequiredParameter)
	}
	if len(arg.InvestData) == 0 {
		return nil, fmt.Errorf("%w: investData", errMissingRequiredParameter)
	}
	for i := range arg.InvestData {
		if arg.InvestData[i].Currency.IsEmpty() {
			return nil, fmt.Errorf("%w: investData[%d]", currency.ErrCurrencyCodeEmpty, i)
		}
		if arg.InvestData[i].Amount <= 0 {
			return nil, fmt.Errorf("%w: investData[%d]", limits.ErrAmountBelowMin, i)
		}
	}
	var resp *OnChainEarnOrderIDResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, purchaseEPL, http.MethodPost, "finance/staking-defi/purchase", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// RedeemOnChainEarn redeems an on-chain earn investment
func (e *Exchange) RedeemOnChainEarn(ctx context.Context, arg *OnChainEarnRedeemRequest) (*OnChainEarnOrderIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.ProtocolType == "" {
		return nil, fmt.Errorf("%w: protocolType", errMissingRequiredParameter)
	}
	var resp *OnChainEarnOrderIDResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, redeemEPL, http.MethodPost, "finance/staking-defi/redeem", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelOnChainEarnPurchaseOrRedemption cancels a pending on-chain earn purchase or redemption; the returned funds go to
// the funding account
func (e *Exchange) CancelOnChainEarnPurchaseOrRedemption(ctx context.Context, arg *OnChainEarnCancelRequest) (*OnChainEarnOrderIDResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.ProtocolType == "" {
		return nil, fmt.Errorf("%w: protocolType", errMissingRequiredParameter)
	}
	var resp *OnChainEarnOrderIDResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, cancelPurchaseOrRedemptionEPL, http.MethodPost, "finance/staking-defi/cancel", arg, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOnChainEarnActiveOrders returns the account's active on-chain earn orders
func (e *Exchange) GetOnChainEarnActiveOrders(ctx context.Context, arg *OnChainEarnActiveOrdersRequest) ([]OnChainEarnActiveOrderResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.ProductID != "" {
		params.Set("productId", arg.ProductID)
	}
	if arg.ProtocolType != "" {
		params.Set("protocolType", arg.ProtocolType)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.State != "" {
		params.Set("state", arg.State)
	}
	var resp []OnChainEarnActiveOrderResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getEarnActiveOrdersEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-active", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOnChainEarnOrderHistory returns the account's completed on-chain earn orders
func (e *Exchange) GetOnChainEarnOrderHistory(ctx context.Context, arg *OnChainEarnOrderHistoryRequest) ([]OnChainEarnOrderHistoryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if arg.ProductID != "" {
		params.Set("productId", arg.ProductID)
	}
	if arg.ProtocolType != "" {
		params.Set("protocolType", arg.ProtocolType)
	}
	if !arg.Currency.IsEmpty() {
		params.Set("ccy", arg.Currency.String())
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []OnChainEarnOrderHistoryResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, getFundingOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("finance/staking-defi/orders-history", params), nil, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	return resp, nil
}
