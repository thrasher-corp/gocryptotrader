package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Advanced Earn and Liquidity Mining errors
var (
	errUnsupportedAdvancedEarnCategory  = errors.New("unsupported Advanced Earn category")
	errAdvancedEarnProductIDNotSet      = errors.New("product ID not set")
	errAdvancedEarnOrderTypeNotSet      = errors.New("order type not set")
	errAdvancedEarnDirectionNotSet      = errors.New("order direction not set")
	errAdvancedEarnPriceNotSet          = errors.New("price not set")
	errAdvancedEarnAPYNotSet            = errors.New("APY not set")
	errAdvancedEarnLeverageNotSet       = errors.New("leverage not set")
	errAdvancedEarnPositionIDNotSet     = errors.New("position ID not set")
	errAdvancedEarnRedeemAmountNotSet   = errors.New("estimated redeem amount not set")
	errAdvancedEarnKnockoutCouponNotSet = errors.New("knockout coupon not set")
	errAdvancedEarnSettleTypeNotSet     = errors.New("settle type not set")
	errAdvancedEarnInstitutionUIDNotSet = errors.New("institution UID not set")
	errAdvancedEarnAwardIDNotSet        = errors.New("interest card award ID not set")
	errAdvancedEarnSpecCodeNotSet       = errors.New("interest card spec code not set")
	errLiquidityMiningAmountNotSet      = errors.New("quote amount or base amount must be set")
	errLiquidityMiningOrderIDsExclusive = errors.New("order ID and order link ID are mutually exclusive")
)

// GetAdvancedEarnProductInfo returns the products of an Advanced Earn category (Get Product Info)
func (e *Exchange) GetAdvancedEarnProductInfo(ctx context.Context, req *AdvancedEarnProductInfoRequest) (*AdvancedEarnProductInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.Duration != "" {
		params.Set("duration", req.Duration)
	}
	var resp *AdvancedEarnProductInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/advance/product", params), getAdvancedEarnProductInfoEPL, &resp)
}

// GetAdvancedEarnProductQuote returns the current quote of an Advanced Earn product, or the quotes of every
// DiscountBuy product when productID is empty (Get Product Quote; Get Fixed Product Quote for DoubleWin)
func (e *Exchange) GetAdvancedEarnProductQuote(ctx context.Context, category, productID string) (*AdvancedEarnProductQuoteResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *AdvancedEarnProductQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/advance/product-extra-info", params), getAdvancedEarnProductQuoteEPL, &resp)
}

// PlaceAdvancedEarnOrder places an Advanced Earn stake or early redemption order. Bybit accepts the order and processes
// it asynchronously, so its outcome comes from GetAdvancedEarnOrderInfo (Place Order)
func (e *Exchange) PlaceAdvancedEarnOrder(ctx context.Context, req *AdvancedEarnPlaceOrderRequest) (*AdvancedEarnOrderResponse, error) {
	body, err := req.Payload()
	if err != nil {
		return nil, err
	}
	var resp *AdvancedEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/advance/place-order", nil, body, &resp, placeAdvancedEarnOrderEPL)
}

// Payload validates the request and returns the Place Order body
func (r *AdvancedEarnPlaceOrderRequest) Payload() (map[string]any, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	switch {
	case r.Category == "":
		return nil, errCategoryNotSet
	case r.ProductID == "":
		return nil, errAdvancedEarnProductIDNotSet
	case r.OrderType == "":
		return nil, errAdvancedEarnOrderTypeNotSet
	case r.OrderLinkID == "":
		return nil, errOrderLinkIDMissing
	}
	if r.OrderType == AdvancedEarnOrderTypeStake {
		switch {
		case r.Amount <= 0:
			return nil, order.ErrAmountIsInvalid
		case r.Coin.IsEmpty():
			return nil, currency.ErrCurrencyCodeEmpty
		case r.AccountType == "":
			return nil, errMissingAccountType
		}
	}
	body := map[string]any{
		"category":    r.Category,
		"productId":   r.ProductID,
		"orderType":   r.OrderType,
		"orderLinkId": r.OrderLinkID,
	}
	if r.Amount != 0 {
		body["amount"] = strconv.FormatFloat(r.Amount, 'f', -1, 64)
	}
	if r.AccountType != "" {
		body["accountType"] = r.AccountType
	}
	if !r.Coin.IsEmpty() {
		body["coin"] = r.Coin.Upper().String()
	}
	if r.DualAssetsExtra != nil {
		extra, err := r.DualAssetsExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("dualAssetsExtra: %w", err)
		}
		body["dualAssetsExtra"] = extra
	}
	if r.SmartLeverageStakeExtra != nil {
		extra, err := r.SmartLeverageStakeExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("smartLeverageStakeExtra: %w", err)
		}
		body["smartLeverageStakeExtra"] = extra
	}
	if r.SmartLeverageRedeemExtra != nil {
		extra, err := r.SmartLeverageRedeemExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("smartLeverageRedeemExtra: %w", err)
		}
		body["smartLeverageRedeemExtra"] = extra
	}
	if r.DoubleWinStakeExtra != nil {
		extra, err := r.DoubleWinStakeExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("doubleWinStakeExtra: %w", err)
		}
		body["doubleWinStakeExtra"] = extra
	}
	if r.DoubleWinRedeemExtra != nil {
		extra, err := r.DoubleWinRedeemExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("doubleWinRedeemExtra: %w", err)
		}
		body["doubleWinRedeemExtra"] = extra
	}
	if r.DiscountBuyExtra != nil {
		extra, err := r.DiscountBuyExtra.payload()
		if err != nil {
			return nil, fmt.Errorf("discountBuyExtra: %w", err)
		}
		body["discountBuyExtra"] = extra
	}
	if r.InterestCard != nil {
		extra, err := r.InterestCard.payload()
		if err != nil {
			return nil, fmt.Errorf("interestCard: %w", err)
		}
		body["interestCard"] = extra
	}
	return body, nil
}

func (x *DualAssetsOrderExtra) payload() (map[string]any, error) {
	switch {
	case x.OrderDirection == "":
		return nil, errAdvancedEarnDirectionNotSet
	case x.SelectPrice <= 0:
		return nil, fmt.Errorf("%w: selectPrice", errAdvancedEarnPriceNotSet)
	case x.APYE8 == 0:
		return nil, errAdvancedEarnAPYNotSet
	}
	return map[string]any{
		"orderDirection": x.OrderDirection,
		"selectPrice":    strconv.FormatFloat(x.SelectPrice, 'f', -1, 64),
		"apyE8":          x.APYE8,
	}, nil
}

func (x *SmartLeverageStakeExtra) payload() (map[string]any, error) {
	switch {
	case x.InitialPrice <= 0:
		return nil, fmt.Errorf("%w: initialPrice", errAdvancedEarnPriceNotSet)
	case x.BreakevenPrice <= 0:
		return nil, fmt.Errorf("%w: breakevenPrice", errAdvancedEarnPriceNotSet)
	}
	return map[string]any{
		"initialPrice":   strconv.FormatFloat(x.InitialPrice, 'f', -1, 64),
		"breakevenPrice": strconv.FormatFloat(x.BreakevenPrice, 'f', -1, 64),
	}, nil
}

func (x *AdvancedEarnRedeemExtra) payload() (map[string]any, error) {
	switch {
	case x.PositionID == "":
		return nil, errAdvancedEarnPositionIDNotSet
	case x.EstimatedRedeemAmount <= 0:
		return nil, errAdvancedEarnRedeemAmountNotSet
	}
	extra := map[string]any{
		"positionId":      x.PositionID,
		"estRedeemAmount": strconv.FormatFloat(x.EstimatedRedeemAmount, 'f', -1, 64),
	}
	if x.IsSlippageProtected {
		extra["isSlippageProtected"] = true
	}
	return extra, nil
}

func (x *DoubleWinStakeExtra) payload() (map[string]any, error) {
	switch {
	case x.Leverage <= 0:
		return nil, errAdvancedEarnLeverageNotSet
	case x.InitialPrice <= 0:
		return nil, fmt.Errorf("%w: initialPrice", errAdvancedEarnPriceNotSet)
	}
	extra := map[string]any{
		"leverage":     strconv.FormatFloat(x.Leverage, 'f', -1, 64),
		"initialPrice": strconv.FormatFloat(x.InitialPrice, 'f', -1, 64),
	}
	if x.LowerPrice != 0 {
		extra["lowerPrice"] = strconv.FormatFloat(x.LowerPrice, 'f', -1, 64)
	}
	if x.UpperPrice != 0 {
		extra["upperPrice"] = strconv.FormatFloat(x.UpperPrice, 'f', -1, 64)
	}
	return extra, nil
}

func (x *DiscountBuyOrderExtra) payload() (map[string]any, error) {
	switch {
	case x.InitialPrice <= 0:
		return nil, fmt.Errorf("%w: initialPrice", errAdvancedEarnPriceNotSet)
	case x.PurchasePrice <= 0:
		return nil, fmt.Errorf("%w: purchasePrice", errAdvancedEarnPriceNotSet)
	case x.KnockoutPrice <= 0:
		return nil, fmt.Errorf("%w: knockoutPrice", errAdvancedEarnPriceNotSet)
	case x.KnockoutCouponE8 == 0:
		return nil, errAdvancedEarnKnockoutCouponNotSet
	case x.SettleType == "":
		return nil, errAdvancedEarnSettleTypeNotSet
	case x.InstitutionUID == "":
		return nil, errAdvancedEarnInstitutionUIDNotSet
	}
	return map[string]any{
		"initialPrice":     strconv.FormatFloat(x.InitialPrice, 'f', -1, 64),
		"purchasePrice":    strconv.FormatFloat(x.PurchasePrice, 'f', -1, 64),
		"knockoutPrice":    strconv.FormatFloat(x.KnockoutPrice, 'f', -1, 64),
		"knockoutCouponE8": strconv.FormatUint(x.KnockoutCouponE8, 10),
		"settleType":       x.SettleType,
		"instUid":          x.InstitutionUID,
	}, nil
}

func (x *AdvancedEarnInterestCard) payload() (map[string]any, error) {
	switch {
	case x.AwardID == 0:
		return nil, errAdvancedEarnAwardIDNotSet
	case x.SpecCode == "":
		return nil, errAdvancedEarnSpecCodeNotSet
	}
	return map[string]any{
		"awardId":  x.AwardID,
		"specCode": x.SpecCode,
	}, nil
}

// GetAdvancedEarnPositionInfo returns a page of active Advanced Earn positions of a category (Get Position Info)
func (e *Exchange) GetAdvancedEarnPositionInfo(ctx context.Context, req *AdvancedEarnPositionInfoRequest) (*AdvancedEarnPositionInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if !req.Coin.IsEmpty() {
		params.Set("coin", req.Coin.Upper().String())
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *AdvancedEarnPositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/advance/position", params, nil, &resp, getAdvancedEarnPositionInfoEPL)
}

// GetAdvancedEarnOrderInfo returns a page of Advanced Earn orders of a category (Get Order Info)
func (e *Exchange) GetAdvancedEarnOrderInfo(ctx context.Context, req *AdvancedEarnOrderInfoRequest) (*AdvancedEarnOrderInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if err := setAdvancedEarnTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *AdvancedEarnOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/advance/order", params, nil, &resp, getAdvancedEarnOrderInfoEPL)
}

// setAdvancedEarnTimeWindow sets the startTime and endTime parameters in milliseconds, checking the window when both
// bounds are given
func setAdvancedEarnTimeWindow(params url.Values, start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return err
		}
	}
	if !start.IsZero() {
		params.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}

// GetAdvancedEarnRedeemEstimatedAmount returns the estimated early redemption of up to 5 SmartLeverage or DoubleWin
// positions. A redeem order needs a current estimate, which Bybit caches for 10 minutes (Get Redeem Estimated Amount)
func (e *Exchange) GetAdvancedEarnRedeemEstimatedAmount(ctx context.Context, category string, positionIDs []string) (*AdvancedEarnRedeemEstimateResponse, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}
	if len(positionIDs) == 0 {
		return nil, errAdvancedEarnPositionIDNotSet
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("positionIds", strings.Join(positionIDs, ","))
	var resp *AdvancedEarnRedeemEstimateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/advance/get-redeem-est-amount-list", params, nil, &resp, getAdvancedEarnRedeemEstimatedAmountEPL)
}

// GetDoubleWinCustomProductQuote returns the leverage quoted for a custom price range of an RFQ Double Win product
// (Get Custom Product Quote)
func (e *Exchange) GetDoubleWinCustomProductQuote(ctx context.Context, req *DoubleWinCustomQuoteRequest) (*DoubleWinCustomQuoteResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch {
	case req.ProductID == "":
		return nil, errAdvancedEarnProductIDNotSet
	case req.InitialPrice <= 0:
		return nil, fmt.Errorf("%w: initialPrice", errAdvancedEarnPriceNotSet)
	case req.LowerPrice <= 0:
		return nil, fmt.Errorf("%w: lowerPrice", errAdvancedEarnPriceNotSet)
	case req.UpperPrice <= 0:
		return nil, fmt.Errorf("%w: upperPrice", errAdvancedEarnPriceNotSet)
	}
	params := url.Values{}
	params.Set("productId", req.ProductID)
	params.Set("initialPrice", strconv.FormatFloat(req.InitialPrice, 'f', -1, 64))
	params.Set("lowerPrice", strconv.FormatFloat(req.LowerPrice, 'f', -1, 64))
	params.Set("upperPrice", strconv.FormatFloat(req.UpperPrice, 'f', -1, 64))
	var resp *DoubleWinCustomQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/advance/double-win-leverage", params, nil, &resp, defaultEPL)
}

// GetLiquidityMiningProductInfo returns the Liquidity Mining products, optionally filtered by coin (Get Product Info)
func (e *Exchange) GetLiquidityMiningProductInfo(ctx context.Context, baseCoin, quoteCoin currency.Code) (*LiquidityMiningProductInfoResponse, error) {
	params := url.Values{}
	if !baseCoin.IsEmpty() {
		params.Set("baseCoin", baseCoin.Upper().String())
	}
	if !quoteCoin.IsEmpty() {
		params.Set("quoteCoin", quoteCoin.Upper().String())
	}
	var resp *LiquidityMiningProductInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/v5/earn/liquidity-mining/product", params), getLiquidityMiningProductInfoEPL, &resp)
}

// AddLiquidity adds liquidity to a Liquidity Mining pool. Bybit accepts the order and processes it asynchronously, so
// its outcome comes from GetLiquidityMiningOrderInfo (Add Liquidity)
func (e *Exchange) AddLiquidity(ctx context.Context, req *AddLiquidityRequest) (*AdvancedEarnOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch {
	case req.ProductID == "":
		return nil, errAdvancedEarnProductIDNotSet
	case req.OrderLinkID == "":
		return nil, errOrderLinkIDMissing
	case req.QuoteAmount <= 0 && req.BaseAmount <= 0:
		return nil, errLiquidityMiningAmountNotSet
	case req.QuoteAmount > 0 && req.QuoteAccountType == "":
		return nil, fmt.Errorf("%w: quoteAccountType", errMissingAccountType)
	case req.BaseAmount > 0 && req.BaseAccountType == "":
		return nil, fmt.Errorf("%w: baseAccountType", errMissingAccountType)
	}
	body := map[string]any{
		"productId":   req.ProductID,
		"orderLinkId": req.OrderLinkID,
	}
	if req.QuoteAmount > 0 {
		body["quoteAmount"] = strconv.FormatFloat(req.QuoteAmount, 'f', -1, 64)
		body["quoteAccountType"] = req.QuoteAccountType
	}
	if req.BaseAmount > 0 {
		body["baseAmount"] = strconv.FormatFloat(req.BaseAmount, 'f', -1, 64)
		body["baseAccountType"] = req.BaseAccountType
	}
	if req.Leverage != 0 {
		body["leverage"] = strconv.FormatFloat(req.Leverage, 'f', -1, 64)
	}
	var resp *AdvancedEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/liquidity-mining/add-liquidity", nil, body, &resp, addLiquidityEPL)
}

// RemoveLiquidity withdraws liquidity from a Liquidity Mining position. Bybit accepts the order and processes it
// asynchronously, allowing one pending order per position (Remove Liquidity)
func (e *Exchange) RemoveLiquidity(ctx context.Context, req *RemoveLiquidityRequest) (*AdvancedEarnOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := liquidityMiningPositionOrderBody(req.ProductID, req.OrderLinkID, req.PositionID)
	if err != nil {
		return nil, err
	}
	if req.RemoveRate != 0 {
		body["removeRate"] = req.RemoveRate
	}
	if req.RemoveType != "" {
		body["removeType"] = req.RemoveType
	}
	var resp *AdvancedEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/liquidity-mining/remove-liquidity", nil, body, &resp, removeLiquidityEPL)
}

// ReinvestLiquidityMining reinvests a Liquidity Mining position's claimable yield into its pool. Bybit accepts the
// order and processes it asynchronously, allowing one pending order per position (Reinvest)
func (e *Exchange) ReinvestLiquidityMining(ctx context.Context, req *ReinvestLiquidityMiningRequest) (*AdvancedEarnOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := liquidityMiningPositionOrderBody(req.ProductID, req.OrderLinkID, req.PositionID)
	if err != nil {
		return nil, err
	}
	if req.Leverage != 0 {
		body["leverage"] = strconv.FormatUint(req.Leverage, 10)
	}
	var resp *AdvancedEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/liquidity-mining/reinvest", nil, body, &resp, reinvestLiquidityMiningEPL)
}

// AddLiquidityMiningMargin adds quote coin margin to a leveraged Liquidity Mining position to reduce its liquidation
// risk. Bybit accepts the order and processes it asynchronously, allowing one pending order per position (Add Margin)
func (e *Exchange) AddLiquidityMiningMargin(ctx context.Context, req *AddLiquidityMiningMarginRequest) (*AdvancedEarnOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := liquidityMiningPositionOrderBody(req.ProductID, req.OrderLinkID, req.PositionID)
	if err != nil {
		return nil, err
	}
	switch {
	case req.Amount <= 0:
		return nil, order.ErrAmountIsInvalid
	case req.QuoteAccountType == "":
		return nil, errMissingAccountType
	}
	body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	body["quoteAccountType"] = req.QuoteAccountType
	var resp *AdvancedEarnOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/liquidity-mining/add-margin", nil, body, &resp, addLiquidityMiningMarginEPL)
}

// liquidityMiningPositionOrderBody validates the parameters every order on a Liquidity Mining position needs and
// returns them as the start of its body
func liquidityMiningPositionOrderBody(productID, orderLinkID, positionID string) (map[string]any, error) {
	switch {
	case productID == "":
		return nil, errAdvancedEarnProductIDNotSet
	case orderLinkID == "":
		return nil, errOrderLinkIDMissing
	case positionID == "":
		return nil, errAdvancedEarnPositionIDNotSet
	}
	return map[string]any{
		"productId":   productID,
		"orderLinkId": orderLinkID,
		"positionId":  positionID,
	}, nil
}

// ClaimLiquidityMiningInterest claims the yield of a Liquidity Mining product, or of every position when productID is
// "-1", into the default account (Claim Interest)
func (e *Exchange) ClaimLiquidityMiningInterest(ctx context.Context, productID string) error {
	if productID == "" {
		return errAdvancedEarnProductIDNotSet
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/earn/liquidity-mining/claim-interest", nil, map[string]any{"productId": productID}, nil, claimLiquidityMiningInterestEPL)
}

// GetLiquidityMiningPositionInfo returns the active Liquidity Mining positions, optionally filtered by product and
// base coin (Get Position Info)
func (e *Exchange) GetLiquidityMiningPositionInfo(ctx context.Context, productID string, baseCoin currency.Code) (*LiquidityMiningPositionInfoResponse, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if !baseCoin.IsEmpty() {
		params.Set("baseCoin", baseCoin.Upper().String())
	}
	var resp *LiquidityMiningPositionInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/liquidity-mining/position", params, nil, &resp, getLiquidityMiningPositionInfoEPL)
}

// GetLiquidityMiningOrderInfo returns one Liquidity Mining order, or a page of them (Get Order Info)
func (e *Exchange) GetLiquidityMiningOrderInfo(ctx context.Context, req *LiquidityMiningOrderInfoRequest) (*LiquidityMiningOrderInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID != "" && req.OrderLinkID != "" {
		return nil, errLiquidityMiningOrderIDsExclusive
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.OrderLinkID != "" {
		params.Set("orderLinkId", req.OrderLinkID)
	}
	if req.ProductID != "" {
		params.Set("productId", req.ProductID)
	}
	if req.OrderType != "" {
		params.Set("orderType", req.OrderType)
	}
	if err := setAdvancedEarnTimeWindow(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *LiquidityMiningOrderInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/liquidity-mining/order", params, nil, &resp, getLiquidityMiningOrderInfoEPL)
}

// GetLiquidityMiningYieldRecords returns a page of Liquidity Mining yield claims, both manual and settled on removing
// liquidity (Get Yield Records)
func (e *Exchange) GetLiquidityMiningYieldRecords(ctx context.Context, req *LiquidityMiningRecordsRequest) (*LiquidityMiningYieldRecordsResponse, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *LiquidityMiningYieldRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/liquidity-mining/yield-records", params, nil, &resp, getLiquidityMiningYieldRecordsEPL)
}

// GetLiquidityMiningLiquidationRecords returns a page of liquidations of leveraged Liquidity Mining positions (Get
// Liquidation Records)
func (e *Exchange) GetLiquidityMiningLiquidationRecords(ctx context.Context, req *LiquidityMiningRecordsRequest) (*LiquidityMiningLiquidationRecordsResponse, error) {
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *LiquidityMiningLiquidationRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/earn/liquidity-mining/liquidation-records", params, nil, &resp, getLiquidityMiningLiquidationRecordsEPL)
}

// params returns the query parameters of the Liquidity Mining record endpoints, which share them
func (r *LiquidityMiningRecordsRequest) params() (url.Values, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !r.BaseCoin.IsEmpty() {
		params.Set("baseCoin", r.BaseCoin.Upper().String())
	}
	if !r.QuoteCoin.IsEmpty() {
		params.Set("quoteCoin", r.QuoteCoin.Upper().String())
	}
	if err := setAdvancedEarnTimeWindow(params, r.StartTime, r.EndTime); err != nil {
		return nil, err
	}
	if r.Limit != 0 {
		params.Set("limit", strconv.FormatUint(r.Limit, 10))
	}
	if r.Cursor != "" {
		params.Set("cursor", r.Cursor)
	}
	return params, nil
}
