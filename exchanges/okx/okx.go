package okx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Okx is the overarching type across this package
type Okx struct {
	okgroup.OKGroup
}

const (
	okxRateInterval = time.Second
	okxAPIURL       = "https://aws.okx.com/" + okxAPIPath
	okxAPIVersion   = "/v5/"

	okxStandardRequestRate = 6
	okxAPIPath             = "api" + okxAPIVersion
	okxExchangeName        = "OKCOIN International"
	okxWebsocketURL        = "wss://ws.okx.com:8443/ws" + okxAPIVersion

	// publicWsURL  = okxWebsocketURL + "public"
	// privateWsURL = okxWebsocketURL + "private"

	// tradeEndpoints
	tradeOrder                = "trade/order"
	placeMultipleOrderUrl     = "trade/batch-orders"
	cancelTradeOrder          = "trade/cancel-order"
	cancelBatchTradeOrders    = "trade/cancel-batch-orders"
	amendOrder                = "trade/amend-order"
	amendBatchOrders          = "trade/amend-batch-orders"
	closePositionPath         = "trade/close-position"
	pandingTradeOrders        = "trade/orders-pending"
	tradeHistory              = "trade/orders-history"
	orderHistoryArchive       = "trade/orders-history-archive"
	tradeFills                = "trade/fills"
	tradeFillsHistory         = "trade/fills-history"
	assetbills                = "asset/bills"
	lightningDeposit          = "asset/deposit-lightning"
	assetDeposits             = "asset/deposit-address"
	pathToAssetDepositHistory = "asset/deposit-history"
	//
	assetWithdrawal = "asset/withdrawal"

	// algo order routes
	cancelAlgoOrder        = "trade/cancel-algos"
	algoTradeOrder         = "trade/order-algo"
	canceAdvancedAlgoOrder = "trade/cancel-advance-algos"
	getAlgoOrders          = "trade/orders-algo-pending"
	algoOrderHistory       = "trade/orders-algo-history"
	getAlgoOrdersHistory   = "trade/orders-algo-history"

	// funding orders routes
	assetCurrencies    = "asset/currencies"
	assetBalance       = "asset/balances"
	assetValuation     = "asset/asset-valuation"
	assetTransfer      = "asset/transfer"
	assetTransferState = "asset/transfer-state"

	// Market Data
	marketTickers                = "market/tickers"
	indexTickers                 = "market/index-tickers"
	marketBooks                  = "market/books"
	marketCandles                = "market/candles"
	marketCandlesHistory         = "market/history-candles"
	marketCandlesIndex           = "market/index-candles"
	marketPriceCandles           = "market/mark-price-candles"
	marketTrades                 = "market/trades"
	marketPlatformVolumeIn24Hour = "market/platform-24-volume"
	marketOpenOracles            = "market/open-oracle"
	marketExchangeRate           = "market/exchange-rate"
	marketIndexComponents        = "market/index-components"

	// Public endpoints
	publicInstruments                 = "public/instruments"
	publicDeliveryExerciseHistory     = "public/delivery-exercise-history"
	publicOpenInterestValues          = "public/open-interest"
	publicFundingRate                 = "public/funding-rate"
	publicFundingRateHistory          = "public/funding-rate-history"
	publicLimitPath                   = "public/price-limit"
	publicOptionalData                = "public/opt-summary"
	publicEstimatedPrice              = "public/estimated-price"
	publicDiscountRate                = "public/discount-rate-interest-free-quota"
	publicTime                        = "public/time"
	publicLiquidationOrders           = "public/liquidation-orders"
	publicMarkPrice                   = "public/mark-price"
	publicPositionTiers               = "public/position-tiers"
	publicInterestRateAndLoanQuota    = "public/interest-rate-loan-quota"
	publicVIPInterestRateAndLoanQuota = "public/vip-interest-rate-loan-quota"
	publicUnderlyings                 = "public/underlying"
	publicInsuranceFunds              = "public/insurance-fund"

	// Trading Endpoints
	tradingDataSupportedCoins      = "rubik/stat/trading-data/support-coin"
	tradingTakerVolume             = "rubik/stat/taker-volume"
	tradingMarginLoanRatio         = "rubik/stat/margin/loan-ratio"
	longShortAccountRatio          = "rubik/stat/contracts/long-short-account-ratio"
	contractOpenInterestVolume     = "rubik/stat/contracts/open-interest-volume"
	optionOpenInterestVolume       = "rubik/stat/option/open-interest-volume"
	optionOpenInterestVolumeRatio  = "rubik/stat/option/open-interest-volume-ratio"
	optionOpenInterestVolumeExpiry = "rubik/stat/option/open-interest-volume-expiry"
	optionOpenInterestVolumeStrike = "rubik/stat/option/open-interest-volume-strike"
	takerBlockVolume               = "rubik/stat/option/taker-block-volume"

	// Authenticated endpoints
)

var (
	errEmptyPairValues                         = errors.New("empty pair values")
	errDataNotFound                            = errors.New("data not found ")
	errUnableToTypeAssertResponseData          = errors.New("unable to type assert responseData")
	errUnableToTypeAssertKlineData             = errors.New("unable to type assert kline data")
	errUnexpectedKlineDataLength               = errors.New("unexpected kline data length")
	errLimitExceedsMaximumResultPerRequest     = errors.New("maximum result per request exeeds the limit")
	errNo24HrTradeVolumeFound                  = errors.New("no trade record found in the 24 trade volume ")
	errOracleInformationNotFound               = errors.New("oracle informations not found")
	errExchangeInfoNotFound                    = errors.New("exchange information not found")
	errIndexComponentNotFound                  = errors.New("unable to fetch index components")
	errMissingRequiredArgInstType              = errors.New("invalid required argument instrument type")
	errLimitValueExceedsMaxof100               = fmt.Errorf("limit value exceeds the maximum value %d", 100)
	errParameterUnderlyingCanNotBeEmpty        = errors.New("parameter uly can not be empty")
	errMissingInstrumentID                     = errors.New("missing instrument id")
	errFundingRateHistoryNotFound              = errors.New("funding rate history not found")
	errMissingRequiredUnderlying               = errors.New("error missing required parameter underlying")
	errMissingRequiredParamInstID              = errors.New("missing required parameter instrument id")
	errLiquidationOrderResponseNotFound        = errors.New("liquidation order not found")
	errEitherInstIDOrCcyIsRequired             = errors.New("either parameter instId or ccy is required")
	errIncorrectRequiredParameterTradeMode     = errors.New("unacceptable required argument, trade mode")
	errInterestRateAndLoanQuotaNotFound        = errors.New("interest rate and loan quota not found")
	errUnderlyingsForSpecifiedInstTypeNofFound = errors.New("underlyings for the specified instrument id is not found")
	errInsuranceFundInformationNotFound        = errors.New("insurance fund information not found")
	errMissingExpiryTimeParameter              = errors.New("missing expiry date parameter")
	errParsingResponseError                    = errors.New("error while parsing the response")
	errInvalidTradeModeValue                   = errors.New("invalid trade mode value")
	errMissingOrderSide                        = errors.New("missing order side")
	errInvalidOrderType                        = errors.New("invalid order type")
	errInvalidQuantityToButOrSell              = errors.New("unacceptable quantity to buy or sell")
	errMissingClientOrderIDOrOrderID           = errors.New("client supplier order id or order id is missing")
	errMissingNewSizeOrPriceInformation        = errors.New("missing the new size or price information")
	errMissingNewSize                          = errors.New("missing the order size information")
	errMissingMarginMode                       = errors.New("missing required param margin mode \"mgnMode\"")
	errMissingTradeMode                        = errors.New("missing trade mode")
	errInvalidTriggerPrice                     = errors.New("invalid trigger price value")
	errMssingAlgoOrderID                       = errors.New("missing algo orders id")
	errInvalidAverageAmountParameterValue      = errors.New("invalid average amount parameter value")
	errInvalidPriceLimit                       = errors.New("invalid price limit value")
	errMissingIntervalValue                    = errors.New("missing interval value")
	errMissingEitherAlgoIDOrState              = errors.New("either algo id or order state is required")
	errInvalidFundingAmount                    = errors.New("invalid funding amount")
	errInvalidCurrencyValue                    = errors.New("invalid currency value")
	errInvalidDepositAmount                    = errors.New("invalid deposit amount")
)

/************************************ MarketData Endpoints *************************************************/

// PlaceOrder place an order only if you have sufficient funds.
func (ok *Okx) PlaceOrder(ctx context.Context, arg PlaceOrderRequestParam) (*PlaceOrderResponse, error) {
	if arg.InstrumentID == "" {
		return nil, errMissingInstrumentID
	}
	arg.TradeMode = strings.Trim(arg.TradeMode, " ")
	if !(strings.EqualFold("cross", arg.TradeMode) || strings.EqualFold("isolated", arg.TradeMode) || strings.EqualFold("cash", arg.TradeMode)) {
		return nil, errInvalidTradeModeValue
	}
	if !(strings.EqualFold(arg.Side, "buy") || strings.EqualFold(arg.Side, "sell")) {
		return nil, errMissingOrderSide
	}
	if !(strings.EqualFold(arg.OrderType, "market") || strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc") || strings.EqualFold(arg.OrderType, "optimal_limit_ioc")) {
		return nil, errInvalidOrderType
	}
	if arg.QuantityToBuyOrSell <= 0 {
		return nil, errInvalidQuantityToButOrSell
	}
	if arg.OrderPrice <= 0 && (strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc")) {
		return nil, fmt.Errorf("invalid order price for %s order types", arg.OrderType)
	}
	if !(strings.EqualFold(arg.QuantityType, "base_ccy") || strings.EqualFold(arg.QuantityType, "quote_ccy")) {
		arg.QuantityType = ""
	}
	type response struct {
		Msg  string                `json:"msg"`
		Code string                `json:"code"`
		Data []*PlaceOrderResponse `json:"data"`
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, tradeOrder, &arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("error parsing the response")
	}
	return resp.Data[0], nil
}

// PlaceMultipleOrders  to place orders in batches. Maximum 20 orders can be placed at a time. Request parameters should be passed in the form of an array.
func (ok *Okx) PlaceMultipleOrders(ctx context.Context, args []PlaceOrderRequestParam) ([]*PlaceOrderResponse, error) {
	for x := range args {
		arg := args[x]
		if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
		arg.TradeMode = strings.Trim(arg.TradeMode, " ")
		if !(strings.EqualFold("cross", arg.TradeMode) || strings.EqualFold("isolated", arg.TradeMode) || strings.EqualFold("cash", arg.TradeMode)) {
			return nil, errInvalidTradeModeValue
		}
		if !(strings.EqualFold(arg.Side, "buy") || strings.EqualFold(arg.Side, "sell")) {
			return nil, errMissingOrderSide
		}
		if !(strings.EqualFold(arg.OrderType, "market") || strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
			strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc") || strings.EqualFold(arg.OrderType, "optimal_limit_ioc")) {
			return nil, errInvalidOrderType
		}
		if arg.QuantityToBuyOrSell <= 0 {
			return nil, errInvalidQuantityToButOrSell
		}
		if arg.OrderPrice <= 0 && (strings.EqualFold(arg.OrderType, "limit") || strings.EqualFold(arg.OrderType, "post_only") ||
			strings.EqualFold(arg.OrderType, "fok") || strings.EqualFold(arg.OrderType, "ioc")) {
			return nil, fmt.Errorf("invalid order price for %s order types", arg.OrderType)
		}
		if !(strings.EqualFold(arg.QuantityType, "base_ccy") || strings.EqualFold(arg.QuantityType, "quote_ccy")) {
			arg.QuantityType = ""
		}
	}
	type response struct {
		Msg  string                `json:"msg"`
		Code string                `json:"code"`
		Data []*PlaceOrderResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, placeMultipleOrderUrl, &args, &resp, true)
}

// CancelOrder cancel an incomplete order.
func (ok *Okx) CancelOrder(ctx context.Context, arg CancelOrderRequestParam) (*CancelOrderResponse, error) {
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
		return nil, fmt.Errorf("either order id or client supplier id is required")
	}
	type response struct {
		Msg  string                 `json:"msg"`
		Code string                 `json:"code"`
		Data []*CancelOrderResponse `json:"data"`
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, cancelTradeOrder, &arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("error parsing the response data")
	}
	return resp.Data[0], nil
}

// CancelMultipleOrders cancel incomplete orders in batches. Maximum 20 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelMultipleOrders(ctx context.Context, args []CancelOrderRequestParam) ([]*CancelOrderResponse, error) {
	for x := range args {
		arg := args[x]
		if strings.Trim(arg.InstrumentID, " ") == "" {
			return nil, errMissingInstrumentID
		}
		if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
			return nil, fmt.Errorf("either order id or client supplier id is required")
		}
	}
	type response struct {
		Msg  string                 `json:"msg"`
		Code string                 `json:"code"`
		Data []*CancelOrderResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot,
		http.MethodPost, cancelBatchTradeOrders, args, &resp, true)
}

// AmendOrder an incomplete order.
func (ok *Okx) AmendOrder(ctx context.Context, arg *AmendOrderRequestParams) (*AmendOrderResponse, error) {
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	if arg.ClientSuppliedOrderID == "" && arg.OrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	}
	if arg.NewQuantity <= 0 && arg.NewPrice <= 0 {
		return nil, errMissingNewSizeOrPriceInformation
	}
	type response struct {
		Code string                `json:"code"`
		Msg  string                `json:"msg"`
		Data []*AmendOrderResponse `json:"data"`
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, amendOrder, arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) == 0 {
		return nil, errParsingResponseError
	}
	return resp.Data[0], nil
}

// AmendMultipleOrders amend incomplete orders in batches. Maximum 20 orders can be amended at a time. Request parameters should be passed in the form of an array.
func (ok *Okx) AmendMultipleOrders(ctx context.Context, args []AmendOrderRequestParams) ([]*AmendOrderResponse, error) {
	for x := range args {
		if strings.Trim(args[x].InstrumentID, " ") == "" {
			return nil, errMissingInstrumentID
		}
		if args[x].ClientSuppliedOrderID == "" && args[x].OrderID == "" {
			return nil, errMissingClientOrderIDOrOrderID
		}
		if args[x].NewQuantity <= 0 && args[x].NewPrice <= 0 {
			return nil, errMissingNewSizeOrPriceInformation
		}
	}
	type response struct {
		Code string                `json:"code"`
		Msg  string                `json:"msg"`
		Data []*AmendOrderResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, amendBatchOrders, &args, &resp, true)
}

// ClosePositions Close all positions of an instrument via a market order.
func (ok *Okx) ClosePositions(ctx context.Context, arg *ClosePositionsRequestParams) (*ClosePositionResponse, error) {
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	if !(arg.MarginMode != "" && (strings.EqualFold(arg.MarginMode, "cross") || strings.EqualFold(arg.MarginMode, "isolated"))) {
		return nil, errMissingMarginMode
	}
	// if arg.MarginMode != "" && strings.EqualFold(arg.MarginMode, "cross") && (arg.Currency == "") {
	// 	return nil, errMissingRequiredParamCurrency
	// }
	type response struct {
		Msg  string                   `json:"msg"`
		Code string                   `json:"code"`
		Data []*ClosePositionResponse `json:"data"`
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, closePositionPath, arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) == 0 {
		return nil, errors.New(resp.Msg)
	}
	return resp.Data[0], nil
}

// GetOrderDetails retrieve order details.
func (ok *Okx) GetOrderDetail(ctx context.Context, arg *OrderDetailRequestParam) (*OrderDetail, error) {
	params := url.Values{}
	if strings.Trim(arg.InstrumentID, " ") == "" {
		return nil, errMissingInstrumentID
	}
	params.Set("instId", arg.InstrumentID)
	if arg.OrderID == "" && arg.ClientSupplierOrderID == "" {
		return nil, errMissingClientOrderIDOrOrderID
	} else if arg.ClientSupplierOrderID == "" {
		params.Set("ordId", arg.OrderID)
	} else {
		params.Set("clOrdId", arg.ClientSupplierOrderID)
	}
	type response struct {
		Code string         `json:"code"`
		Msg  string         `json:"msg"`
		Data []*OrderDetail `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(tradeOrder, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) == 0 {
		return nil, errors.New(resp.Msg)
	}
	return resp.Data[0], nil
}

// GetOrderList retrieve all incomplete orders under the current account.
func (ok *Okx) GetOrderList(ctx context.Context, arg *OrderListRequestParams) ([]*PendingOrderItem, error) {
	params := url.Values{}
	if strings.EqualFold(arg.InstrumentType, "SPOT") ||
		strings.EqualFold(arg.InstrumentType, "MARGIN") ||
		strings.EqualFold(arg.InstrumentType, "SWAP") ||
		strings.EqualFold(arg.InstrumentType, "FUTURES") ||
		strings.EqualFold(arg.InstrumentType, "OPTION") {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if strings.EqualFold(arg.OrderType, "market") ||
		strings.EqualFold(arg.OrderType, "limit") ||
		strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") ||
		strings.EqualFold(arg.OrderType, "ioc") ||
		strings.EqualFold(arg.OrderType, "optimal_limit_ioc") {
		params.Set("orderType", arg.OrderType)
	}
	if strings.EqualFold(arg.State, "canceled") ||
		strings.EqualFold(arg.State, "filled") {
		params.Set("state", arg.State)
	}
	if !(arg.Before.IsZero()) {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !(arg.After.IsZero()) {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	path := common.EncodeURLValues(pandingTradeOrders, params)
	type response struct {
		Code string              `json:"code"`
		Msg  string              `json:"msg"`
		Data []*PendingOrderItem `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// Get7DayOrderHistory retrieve the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours.
func (ok *Okx) Get7DayOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]*PendingOrderItem, error) {
	return ok.getOrderHistory(ctx, arg, tradeHistory)
}

// Get3MonthOrderHistory retrieve the completed order data for the last 7 days, and the incomplete orders that have been cancelled are only reserved for 2 hours.
func (ok *Okx) Get3MonthOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams) ([]*PendingOrderItem, error) {
	return ok.getOrderHistory(ctx, arg, orderHistoryArchive)
}

// getOrderHistory retrives the order history of the past limited times
func (ok *Okx) getOrderHistory(ctx context.Context, arg *OrderHistoryRequestParams, route string) ([]*PendingOrderItem, error) {
	params := url.Values{}
	if strings.EqualFold(arg.InstrumentType, "SPOT") ||
		strings.EqualFold(arg.InstrumentType, "MARGIN") ||
		strings.EqualFold(arg.InstrumentType, "SWAP") ||
		strings.EqualFold(arg.InstrumentType, "FUTURES") ||
		strings.EqualFold(arg.InstrumentType, "OPTION") {
		params.Set("instType", arg.InstrumentType)
	} else {
		return nil, errMissingRequiredArgInstType
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if strings.EqualFold(arg.OrderType, "market") ||
		strings.EqualFold(arg.OrderType, "limit") ||
		strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") ||
		strings.EqualFold(arg.OrderType, "ioc") ||
		strings.EqualFold(arg.OrderType, "optimal_limit_ioc") {
		params.Set("orderType", arg.OrderType)
	}
	if strings.EqualFold(arg.State, "canceled") ||
		strings.EqualFold(arg.State, "filled") {
		params.Set("state", arg.State)
	}
	if !(arg.Before.IsZero()) {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !(arg.After.IsZero()) {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if strings.EqualFold("twap", arg.Category) || strings.EqualFold("adl", arg.Category) || strings.EqualFold("full_liquidation", arg.Category) || strings.EqualFold("partial_liquidation", arg.Category) || strings.EqualFold("delivery", arg.Category) || strings.EqualFold("ddh", arg.Category) {
		params.Set("category", strings.ToLower(arg.Category))
	}
	path := common.EncodeURLValues(tradeHistory, params)
	type response struct {
		Code string              `json:"code"`
		Msg  string              `json:"msg"`
		Data []*PendingOrderItem `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// GetTransactionDetailsLast3Days retrieve recently-filled transaction details in the last 3 day.
func (ok *Okx) GetTransactionDetailsLast3Days(ctx context.Context, arg *TransactionDetailRequestParams) ([]*TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, tradeFills)
}

// GetTransactionDetailsLast3Months Retrieve recently-filled transaction details in the last 3 months.
func (ok *Okx) GetTransactionDetailsLast3Months(ctx context.Context, arg *TransactionDetailRequestParams) ([]*TransactionDetail, error) {
	return ok.getTransactionDetails(ctx, arg, tradeFillsHistory)
}

// GetTransactionDetails retrieve recently-filled transaction details.
func (ok *Okx) getTransactionDetails(ctx context.Context, arg *TransactionDetailRequestParams, route string) ([]*TransactionDetail, error) {
	params := url.Values{}
	if strings.EqualFold(arg.InstrumentType, "SPOT") ||
		strings.EqualFold(arg.InstrumentType, "MARGIN") ||
		strings.EqualFold(arg.InstrumentType, "SWAP") ||
		strings.EqualFold(arg.InstrumentType, "FUTURES") ||
		strings.EqualFold(arg.InstrumentType, "OPTION") {
		params.Set("instType", arg.InstrumentType)
	} else {
		return nil, errMissingRequiredArgInstType
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if strings.EqualFold(arg.OrderType, "market") ||
		strings.EqualFold(arg.OrderType, "limit") ||
		strings.EqualFold(arg.OrderType, "post_only") ||
		strings.EqualFold(arg.OrderType, "fok") ||
		strings.EqualFold(arg.OrderType, "ioc") ||
		strings.EqualFold(arg.OrderType, "optimal_limit_ioc") {
		params.Set("orderType", arg.OrderType)
	}
	if !(arg.Begin.IsZero()) {
		params.Set("begin", strconv.FormatInt(arg.Begin.UnixMilli(), 10))
	}
	if !(arg.End.IsZero()) {
		params.Set("end", strconv.FormatInt(arg.End.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	if arg.After != "" {
		params.Set("after", arg.After)
	}
	if arg.Before != "" {
		params.Set("before", arg.Before)
	}
	path := common.EncodeURLValues(route, params)
	type response struct {
		Code string               `json:"code"`
		Msg  string               `json:"msg"`
		Data []*TransactionDetail `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// PlaceAlgoOrder order includes trigger order, oco order, conditional order,iceberg order, twap order and trailing order.
func (ok *Okx) PlaceAlgoOrder(ctx context.Context, arg *AlgoOrderParams) (*AlgoOrder, error) {
	if !(arg.InstrumentID != "") {
		return nil, errMissingInstrumentID
	}
	if !(arg.TradeMode != "") {
		return nil, errMissingTradeMode
	}
	if arg.Side == "" {
		return nil, errMissingOrderSide
	}
	if arg.OrderType == "" {
		return nil, errInvalidOrderType
	}
	if arg.Size <= 0 {
		return nil, errMissingNewSize
	}
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, errors.New(resp.Msg)
}

// StopOrderParams
func (ok *Okx) StopOrderParams(ctx context.Context, arg *StopOrderParams) ([]*AlgoOrder, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
}

// PlaceTrailingStopOrder
func (ok *Okx) PlaceTrailingStopOrder(ctx context.Context, arg *TrailingStopOrderRequestParam) ([]*AlgoOrder, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
}

// IceburgOrder
func (ok *Okx) PlaceIceburgOrder(ctx context.Context, arg *IceburgOrder) ([]*AlgoOrder, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
}

// PlaceTWAPOrder
func (ok *Okx) PlaceTWAPOrder(ctx context.Context, arg *TWAPOrderRequestParams) ([]*AlgoOrder, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	if arg.PriceRatio == "" || arg.PriceVariance == "" {
		return nil, fmt.Errorf("missing price ratio or price variance parameter")
	} else if arg.AverageAmount <= 0 {
		return nil, errInvalidAverageAmountParameterValue
	} else if arg.PriceLimit <= 0 {
		return nil, errInvalidPriceLimit
	} else if ok.GetIntervalEnum(arg.Timeinterval) == "" {
		return nil, errMissingIntervalValue
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
}

// TriggerAlogOrder  fetches algo trigger orders for SWAP market types.
func (ok *Okx) TriggerAlogOrder(ctx context.Context, arg *TriggerAlogOrderParams) (*AlgoOrder, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	if arg.TriggerPrice <= 0 {
		return nil, errInvalidTriggerPrice
	}
	if !(strings.EqualFold(arg.TriggerPriceType, "last") ||
		strings.EqualFold(arg.TriggerPriceType, "index") ||
		strings.EqualFold(arg.TriggerPriceType, "mark")) {
		arg.TriggerPriceType = ""
	}
	var resp response
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, algoTradeOrder, arg, &resp, true)
	if er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, errors.New(resp.Msg)
}

// CancelAlgoOrder to cancel unfilled algo orders (not including Iceberg order, TWAP order, Trailing Stop order).
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) ([]*AlgoOrder, error) {
	return ok.cancelAlgoOrder(ctx, args, cancelAlgoOrder)
}

// cancelAlgoOrder to cancel unfilled algo orders.
func (ok *Okx) cancelAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams, route string) ([]*AlgoOrder, error) {
	for x := range args {
		arg := args[x]
		if arg.AlgoOrderID == "" {
			return nil, errMssingAlgoOrderID
		} else if arg.InstrumentID == "" {
			return nil, errMissingInstrumentID
		}
	}
	if len(args) == 0 {
		return nil, errors.New("missing important arguments")
	}
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []*AlgoOrder `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, route, args, &resp, true)
}

// CancelAdvanceAlgoOrder Cancel unfilled algo orders
// A maximum of 10 orders can be canceled at a time.
// Request parameters should be passed in the form of an array.
func (ok *Okx) CancelAdvanceAlgoOrder(ctx context.Context, args []AlgoOrderCancelParams) ([]*AlgoOrder, error) {
	return ok.cancelAlgoOrder(ctx, args, canceAdvancedAlgoOrder)
}

// GetAlgoOrderList retrieve a list of untriggered Algo orders under the current account.
func (ok *Okx) GetAlgoOrderList(ctx context.Context, orderType, algoOrderID, instrumentType, instrumentID string, after, before time.Time, limit uint) ([]*AlgoOrderResponse, error) {
	params := url.Values{}
	if !(strings.EqualFold(orderType, "conditional") || strings.EqualFold(orderType, "oco") || strings.EqualFold(orderType, "trigger") || strings.EqualFold(orderType, "move_order_stop") ||
		strings.EqualFold(orderType, "iceberg") || strings.EqualFold(orderType, "twap")) {
		return nil, fmt.Errorf("invalid order type value %s,%s,%s,%s,%s,and %s", "conditional", "oco", "trigger", "move_order_stop", "iceberg", "twap")
	} else {
		params.Set("ordType", orderType)
	}
	type response struct {
		Code string               `json:"code"`
		Msg  string               `json:"msg"`
		Data []*AlgoOrderResponse `json:"data"`
	}
	var resp response
	if algoOrderID != "" {
		params.Set("algoId", algoOrderID)
	}
	if strings.EqualFold(instrumentType, "spot") || strings.EqualFold(instrumentType, "swap") ||
		strings.EqualFold(instrumentType, "futures") || strings.EqualFold(instrumentType, "option") {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.After(time.Now()) {
		params.
			Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	path := common.EncodeURLValues(getAlgoOrders, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// GetAlgoOrderHistory load a list of all algo orders under the current account in the last 3 months.
func (ok *Okx) GetAlgoOrderHistory(ctx context.Context, orderType, state, algoOrderID, instrumentType, instrumentID string, after, before time.Time, limit uint) ([]*AlgoOrderResponse, error) {
	params := url.Values{}
	if !(strings.EqualFold(orderType, "conditional") || strings.EqualFold(orderType, "oco") || strings.EqualFold(orderType, "trigger") || strings.EqualFold(orderType, "move_order_stop") ||
		strings.EqualFold(orderType, "iceberg") || strings.EqualFold(orderType, "twap")) {
		return nil, fmt.Errorf("invalid order type value %s,%s,%s,%s,%s,and %s", "conditional", "oco", "trigger", "move_order_stop", "iceberg", "twap")
	} else {
		params.Set("ordType", orderType)
	}
	type response struct {
		Code string               `json:"code"`
		Msg  string               `json:"msg"`
		Data []*AlgoOrderResponse `json:"data"`
	}
	var resp response
	if algoOrderID == "" &&
		!(strings.EqualFold(state, "effective") ||
			strings.EqualFold(state, "order_failed") ||
			strings.EqualFold(state, "canceled")) {
		return nil, errMissingEitherAlgoIDOrState
	} else {
		if algoOrderID != "" {
			params.Set("algoId", algoOrderID)
		} else {
			params.Set("state", state)
		}
	}
	if strings.EqualFold(instrumentType, "spot") || strings.EqualFold(instrumentType, "swap") ||
		strings.EqualFold(instrumentType, "futures") || strings.EqualFold(instrumentType, "option") {
		params.Set("instType", instrumentType)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	if !before.IsZero() && before.Before(time.Now()) {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if !after.IsZero() && after.After(time.Now()) {
		params.
			Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	path := common.EncodeURLValues(getAlgoOrdersHistory, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

/*************************************** Block trading ********************************/
/*************************************** Funding Tradings ********************************/

// GetCurrencies Retrieve a list of all currencies.
func (ok *Okx) GetCurrencies(ctx context.Context) ([]*CurrencyResponse, error) {
	type response struct {
		Code string              `json:"code"`
		Msg  string              `json:"msg"`
		Data []*CurrencyResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, assetCurrencies, nil, &resp, true)
}

// GetBalance retrieve the balances of all the assets and the amount that is available or on hold.
func (ok *Okx) GetBalance(ctx context.Context, currency string) ([]*AssetBalance, error) {
	type response struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data []*AssetBalance `json:"data"`
	}
	var resp response
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	path := common.EncodeURLValues(assetBalance, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// GetAccountAssetValuation view account asset valuation
func (ok *Okx) GetAccountAssetValuation(ctx context.Context, currency string) ([]*AccountAssetValuation, error) {
	params := url.Values{}
	if strings.EqualFold(currency, "BTC") || strings.EqualFold(currency, "USDT") ||
		strings.EqualFold(currency, "USD") || strings.EqualFold(currency, "CNY") ||
		strings.EqualFold(currency, "JPY") || strings.EqualFold(currency, "KRW") ||
		strings.EqualFold(currency, "RUB") || strings.EqualFold(currency, "EUR") ||
		strings.EqualFold(currency, "VND") || strings.EqualFold(currency, "IDR") ||
		strings.EqualFold(currency, "INR") || strings.EqualFold(currency, "PHP") ||
		strings.EqualFold(currency, "THB") || strings.EqualFold(currency, "TRY") ||
		strings.EqualFold(currency, "AUD") || strings.EqualFold(currency, "SGD") ||
		strings.EqualFold(currency, "ARS") || strings.EqualFold(currency, "SAR") ||
		strings.EqualFold(currency, "AED") || strings.EqualFold(currency, "IQD") {
		params.Set("ccy", currency)
	}
	path := common.EncodeURLValues(assetValuation, params)
	type response struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []*AccountAssetValuation `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// FundingTransfer transfer of funds between your funding account and trading account,
// and from the master account to sub-accounts.
func (ok *Okx) FundingTransfer(ctx context.Context, arg *FundingTransferRequestInput) ([]*FundingTransferResponse, error) {
	type response struct {
		Msg  string                     `json:"msg"`
		Code string                     `json:"code"`
		Data []*FundingTransferResponse `json:"data"`
	}
	var resp response
	if arg == nil {
		return nil, errors.New("argument can not be null")
	}
	if arg.Amount <= 0 {
		return nil, errors.New("invalid funding amount")
	}
	if arg.Currency == "" {
		return nil, errors.New("invalid currency value")
	}
	if !(strings.EqualFold(arg.From, "6") || strings.EqualFold(arg.From, "18")) {
		return nil, errors.New("missing funding source field \"From\"")
	}
	if arg.To == "" {
		return nil, errors.New("missing funding destination field \"To\"")
	}
	if arg.Type >= 0 && arg.Type <= 4 {
		if arg.Type == 1 || arg.Type == 2 {
			if arg.SubAccount == "" {
				return nil, errors.New("subaccount name required")
			}
		}
	} else {
		return nil, errors.New("invalid reqest type")
	}
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, assetTransfer, arg, &resp, true)
}

// GetFundsTransferState get funding rate response.
func (ok *Okx) GetFundsTransferState(ctx context.Context, transferID, clientID string, transfer_type int) ([]*TransferFundRateResponse, error) {
	params := url.Values{}
	if transferID == "" && clientID == "" {
		return nil, errors.New("either 'transfer id' or 'client id' is required")
	} else if transferID != "" {
		params.Set("transId", transferID)
	} else if clientID == "" {
		params.Set("clientId", clientID)
	}
	if transfer_type > 0 && transfer_type <= 2 {
		params.Set("type", strconv.Itoa(transfer_type))
	}
	path := common.EncodeURLValues(assetTransferState, params)
	type response struct {
		Code string                      `json:"code"`
		Msg  string                      `json:"msg"`
		Data []*TransferFundRateResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

//  AssetBillsDeta Query the billing record, you can get the latest 1 month historical data
func (ok *Okx) GetAssetBillsDetails(ctx context.Context, currency string, bill_type int, clientID string, clientSecret string, after time.Time, before time.Time, limit int) ([]*AssetBillDetail, error) {
	params := url.Values{}
	billTypeIDS := []int{1, 2, 13, 20, 21, 28, 41, 42, 47, 48, 49, 50, 51, 52, 53, 54, 59, 60, 61, 68, 69, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 150, 151, 198, 199}
	for x := range billTypeIDS {
		if billTypeIDS[x] == bill_type {
			params.Set("type", strconv.Itoa(bill_type))
		}
	}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	path := common.EncodeURLValues(assetbills, params)
	type response struct {
		Code string             `json:"code"`
		Msg  string             `json:"msg"`
		Data []*AssetBillDetail `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, path, nil, &resp, true)
}

// GetLightningDeposits users can create up to 10 thousand different invoices within 24 hours.
// this method fetches list of lightning deposits filtered by a currency and amount.
func (ok *Okx) GetLightningDeposits(ctx context.Context, currency string, amount float64, to int) ([]*LightningDepositItem, error) {
	params := url.Values{}
	if currency == "" {
		return nil, errInvalidCurrencyValue
	} else {
		params.Set("ccy", currency)
	}
	if amount <= 0 {
		return nil, errInvalidDepositAmount
	} else {
		params.Set("amt", strconv.FormatFloat(amount, 'f', 0, 64))
	}
	if to == 6 || to == 18 {
		params.Set("to", strconv.Itoa(to))
	}
	path := common.EncodeURLValues(lightningDeposit, params)
	type response struct {
		Code string                  `json:"code"`
		Data []*LightningDepositItem `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// GetCurrencyDepositAddress returns the deposit address and related informations for the provided currency information.
func (ok *Okx) GetCurrencyDepositAddress(ctx context.Context, currency string) ([]*CurrencyDepositResponseItem, error) {
	params := url.Values{}
	if currency == "" {
		return nil, errInvalidCurrencyValue
	} else {
		params.Set("ccy", currency)
	}
	path := common.EncodeURLValues(assetDeposits, params)
	type response struct {
		Code string                         `json:"code"`
		Msg  string                         `json:"msg"`
		Data []*CurrencyDepositResponseItem `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// GetCurrencyDepositHistory retrives deposit records and withdrawal status information depending on the currency, timestamp, and chronogical order.
func (ok *Okx) GetCurrencyDepositHistory(ctx context.Context, currency string, depositID string, transactionID string, state int, after, before time.Time, limit uint) ([]*DepositHistoryResponseItem, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if depositID != "" {
		params.Set("depId", depositID)
	}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if state == 0 ||
		state == 1 ||
		state == 2 {
		params.Set("state", strconv.Itoa(state))
	}
	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}
	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	path := common.EncodeURLValues(pathToAssetDepositHistory, params)
	type response struct {
		Code string                        `json:"code"`
		Msg  string                        `json:"msg"`
		Data []*DepositHistoryResponseItem `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, true)
}

// Withdrawal to perform a withdrawal action. Sub-account does not support withdrawal.
func (ok *Okx) Withdrawal(ctx context.Context, input WithdrawalInput) (*WithdrawalResponse, error) {
	type response struct {
		Code string                `json:"code"`
		Msg  string                `json:"msg"`
		Data []*WithdrawalResponse `json:"data"`
	}
	var resp response
	if input.Currency == "" {
		return nil, errInvalidCurrencyValue
	} else if input.Amount <= 0 {
		return nil, errors.New("invalid withdrawal amount")
	} else if input.WithdrawalDestination == "" {
		return nil, errors.New("withdrawal destination")
	} else if input.ToAddress == "" {
		return nil, errors.New("missing verified digital currency address \"toAddr\" information")
	}
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, assetWithdrawal, &input, &resp, true)
	if er != nil {
		return nil, er
	}
	return resp.Data[0], nil
}

/***************************** Funding Endpoints Ends here ***************************/

// GetTickers retrives the latest price snopshots best bid/ ask price, and tranding volume in the last 34 hours.
func (ok *Okx) GetTickers(ctx context.Context, instType, uly, instId string) ([]MarketDataResponse, error) {
	params := url.Values{}
	if strings.EqualFold(instType, "spot") || strings.EqualFold(instType, "swap") || strings.EqualFold(instType, "futures") || strings.EqualFold(instType, "option") {
		params.Set("instType", instType)
		if (strings.EqualFold(instType, "swap") || strings.EqualFold(instType, "futures") || strings.EqualFold(instType, "option")) && uly != "" {
			params.Set("uly", uly)
		}
	} else if instId != "" {
		params.Set("instId", instId)
	} else {
		return nil, errors.New("missing required variable instType(instruction type) or insId( Instrument ID )")
	}
	path := marketTickers
	if len(params) > 0 {
		path = common.EncodeURLValues(path, params)
	}
	var response OkxMarketDataResponse
	return response.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
}

// GetIndexTickers Retrieves index tickers.
func (ok *Okx) GetIndexTickers(ctx context.Context, quoteCurrency, instId string) ([]OKXIndexTickerResponse, error) {
	response := &struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []OKXIndexTickerResponse `json:"data"`
	}{}
	if instId == "" && quoteCurrency == "" {
		return nil, errors.New("missing required variable! param quoteCcy or instId has to be set")
	}
	params := url.Values{}

	if quoteCurrency != "" {
		params.Set("quoteCcy", quoteCurrency)
	} else if instId != "" {
		params.Set("instId", instId)
	}
	path := indexTickers + "?" + params.Encode()
	return response.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, response, false)
}

// GetOrderBook returns the recent order asks and bids before specified timestamp.
func (ok *Okx) GetOrderBookDepth(ctx context.Context, instrumentID currency.Pair, depth uint) (*OrderBookResponse, error) {
	instId, er := ok.FormatSymbol(instrumentID, asset.Spot)
	if er != nil || instrumentID.IsEmpty() {
		if instrumentID.IsEmpty() {
			return nil, errEmptyPairValues
		}
		return nil, er
	}
	params := url.Values{}
	params.Set("instId", instId)
	if depth > 0 {
		params.Set("sz", strconv.Itoa(int(depth)))
	}
	type response struct {
		Code int                  `json:"code,string"`
		Msg  string               `json:"msg"`
		Data []*OrderBookResponse `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(marketBooks, params)
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if err != nil {
		return nil, err
	} else if len(resp.Data) == 0 {
		return nil, errDataNotFound
	}
	return resp.Data[0], nil
}

// GetIntervalEnum allowed interval params by Okx Exchange
func (ok *Okx) GetIntervalEnum(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.TwoHour:
		return "2H"
	case kline.FourHour:
		return "4H"
	case kline.SixHour:
		return "6H"
	case kline.EightHour:
		return "8H"
	case kline.TwelveHour:
		return "12H"
	case kline.OneDay:
		return "1D"
	case kline.ThreeDay:
		return "3D"
	case kline.FifteenDay:
		return "15D"
	case kline.OneWeek:
		return "1W"
	case kline.TwoWeek:
		return "2W"
	case kline.OneMonth:
		return "1M"
	case kline.ThreeMonth:
		return "3M"
	case kline.SixMonth:
		return "6M"
	case kline.OneYear:
		return "1"
	default:
		return ""
	}
}

// GetCandlesticks Retrieve the candlestick charts. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (ok *Okx) GetCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before time.Time, after time.Time, limit uint64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, marketCandles)
}

// GetCandlesticksHistory Retrieve history candlestick charts from recent years.
func (ok *Okx) GetCandlesticksHistory(ctx context.Context, instrumentID string, interval kline.Interval, before time.Time, after time.Time, limit uint64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, marketCandlesHistory)
}

// GetIndexCandlesticks Retrieve the candlestick charts of the index. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
// the respos is a lis of Candlestick data.
func (ok *Okx) GetIndexCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before time.Time, after time.Time, limit uint64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, marketCandlesIndex)
}

// GetMarkPriceCandlesticks Retrieve the candlestick charts of mark price. This endpoint can retrieve the latest 1,440 data entries. Charts are returned in groups based on the requested bar.
func (ok *Okx) GetMarkPriceCandlesticks(ctx context.Context, instrumentID string, interval kline.Interval, before time.Time, after time.Time, limit uint64) ([]CandleStick, error) {
	return ok.GetCandlestickData(ctx, instrumentID, interval, before, after, limit, marketPriceCandles)
}

// GetCandlestickData handles fetching the data for both the default GetCandlesticks, GetCandlesticksHistory, and GetIndexCandlesticks() methods.
func (ok *Okx) GetCandlestickData(ctx context.Context, instrumentID string, interval kline.Interval, before time.Time, after time.Time, limit uint64, route string) ([]CandleStick, error) {
	params := url.Values{}
	if instrumentID == "" {
		return nil, errMissingInstrumentID
	} else {
		params.Set("instId", instrumentID)
	}
	type response struct {
		Msg  string      `json:"msg"`
		Code int         `json:"code,string"`
		Data interface{} `json:"data"`
	}
	var resp response
	if limit > 0 && limit <= 100 {
		params.Set("limit", strconv.Itoa(int(limit)))
	} else {
		return nil, errLimitExceedsMaximumResultPerRequest
	}
	if !before.IsZero() {
		params.Set("before", strconv.Itoa(int(before.UnixMilli())))
	}
	if !after.IsZero() {
		params.Set("after", strconv.Itoa(int(after.UnixMilli())))
	}
	bar := ok.GetIntervalEnum(interval)
	if bar != "" {
		params.Set("bar", bar)
	}
	path := common.EncodeURLValues(marketCandles, params)
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if err != nil {
		return nil, err
	}
	responseData, okk := (resp.Data).([]interface{})
	if !okk {
		return nil, errUnableToTypeAssertResponseData
	}
	klineData := make([]CandleStick, len(responseData))
	for x := range responseData {
		individualData, ok := responseData[x].([]interface{})
		if !ok {
			return nil, errUnableToTypeAssertKlineData
		}
		if len(individualData) != 7 {
			return nil, errUnexpectedKlineDataLength
		}
		var candle CandleStick
		var er error
		timestamp, er := strconv.Atoi(individualData[0].(string))
		if er != nil {
			return nil, er
		}
		candle.OpenTime = time.UnixMilli(int64(timestamp))
		if candle.OpenPrice, er = convert.FloatFromString(individualData[1]); er != nil {
			return nil, er
		}
		if candle.HighestPrice, er = convert.FloatFromString(individualData[2]); er != nil {
			return nil, er
		}
		if candle.LowestPrice, er = convert.FloatFromString(individualData[3]); er != nil {
			return nil, er
		}
		if candle.ClosePrice, er = convert.FloatFromString(individualData[4]); er != nil {
			return nil, er
		}
		if candle.Volume, er = convert.FloatFromString(individualData[5]); er != nil {
			return nil, er
		}
		if candle.QuoteAssetVolume, er = convert.FloatFromString(individualData[6]); er != nil {
			return nil, er
		}
		klineData[x] = candle
	}
	return klineData, nil
}

// GetTrades Retrieve the recent transactions of an instrument.
func (ok *Okx) GetTrades(ctx context.Context, instrumentId string, limit uint) ([]TradeResponse, error) {
	type response struct {
		Msg  string          `json:"msg"`
		Code int             `json:"code,string"`
		Data []TradeResponse `json:"data"`
	}
	var resp response
	params := url.Values{}
	if instrumentId == "" {
		return nil, errMissingInstrumentID
	} else {
		params.Set("instId", instrumentId)
	}
	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.Itoa(int(limit)))
	}
	path := common.EncodeURLValues(marketTrades, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// Get24HTotalVolume The 24-hour trading volume is calculated on a rolling basis, using USD as the pricing unit.
func (ok *Okx) Get24HTotalVolume(ctx context.Context) (*TradingVolumdIn24HR, error) {
	type response struct {
		Msg  string                 `json:"msg"`
		Code int                    `json:"code,string"`
		Data []*TradingVolumdIn24HR `json:"data"`
	}
	var resp response
	era := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marketPlatformVolumeIn24Hour, nil, &resp, false)
	if era != nil {
		return nil, era
	}
	if len(resp.Data) == 0 {
		return nil, errNo24HrTradeVolumeFound
	}
	return resp.Data[0], nil
}

// GetOracle Get the crypto price of signing using Open Oracle smart contract.
func (ok *Okx) GetOracle(ctx context.Context) (*OracleSmartContractResponse, error) {
	type response struct {
		Msg  string                         `json:"msg"`
		Code int                            `json:"code,string"`
		Data []*OracleSmartContractResponse `json:"data"`
	}
	var resp response
	era := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marketOpenOracles, nil, &resp, false)
	if era != nil {
		return nil, era
	}
	if len(resp.Data) == 0 {
		return nil, errOracleInformationNotFound
	}
	return resp.Data[0], nil
}

// GetExchangeRate this interface provides the average exchange rate data for 2 weeks
// from USD to CNY
func (ok *Okx) GetExchangeRate(ctx context.Context) (*UsdCnyExchangeRate, error) {
	type response struct {
		Msg  string                `json:"msg"`
		Code int                   `json:"code,string"`
		Data []*UsdCnyExchangeRate `json:"data"`
	}
	var resp response
	era := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, marketExchangeRate, nil, &resp, false)
	if era != nil {
		return nil, era
	}
	if len(resp.Data) == 0 {
		return nil, errExchangeInfoNotFound
	}
	return resp.Data[0], nil
}

// GetIndexComponents returns the index component information data on the market
func (ok *Okx) GetIndexComponents(ctx context.Context, index currency.Pair) (*IndexComponent, error) {
	symbolString, err := ok.FormatSymbol(index, asset.Spot)
	if err != nil {
		return nil, err
	}
	type response struct {
		Msg  string          `json:"msg"`
		Code int             `json:"code,string"`
		Data *IndexComponent `json:"data"`
	}
	params := url.Values{}
	params.Set("index", symbolString)
	var resp response
	path := common.EncodeURLValues(marketIndexComponents, params)
	era := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if era != nil {
		return nil, era
	}
	if resp.Data == nil {
		return nil, errIndexComponentNotFound
	}
	return resp.Data, nil
}

/************************************ Public Data Endpoinst *************************************************/

// GetInstruments Retrieve a list of instruments with open contracts.
func (ok *Okx) GetInstruments(ctx context.Context, arg *InstrumentsFetchParams) ([]*Instrument, error) {
	params := url.Values{}
	if !(strings.EqualFold(arg.InstrumentType, "SPOT") || strings.EqualFold(arg.InstrumentType, "MARGIN") || strings.EqualFold(arg.InstrumentType, "SWAP") || strings.EqualFold(arg.InstrumentType, "FUTURES") || strings.EqualFold(arg.InstrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", arg.InstrumentType)
	}
	if arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	}
	type response struct {
		Code string        `json:"code"`
		Msg  string        `json:"msg"`
		Data []*Instrument `json:"data"`
	}

	var resp response
	path := common.EncodeURLValues(publicInstruments, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// GetDeliveryHistory retrieve the estimated delivery price of the last 3 months, which will only have a return value one hour before the delivery/exercise.
func (ok *Okx) GetDeliveryHistory(ctx context.Context, instrumentType, underlying string, after, before time.Time, limit int) ([]*DeliveryHistory, error) {
	params := url.Values{}
	if instrumentType != "" && !(strings.EqualFold(instrumentType, "FUTURES") || strings.EqualFold(instrumentType, "OPTION")) {
		return nil, fmt.Errorf("unacceptable instrument Type! Only %s and %s are allowed", "FUTURE", "OPTION")
	} else if instrumentType == "" {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", instrumentType)
	}
	if underlying != "" {
		params.Set("Underlying", underlying)
		params.Set("uly", underlying)
	} else {
		return nil, errParameterUnderlyingCanNotBeEmpty
	}
	if !(after.IsZero()) {
		params.Set("after", strconv.Itoa(int(after.UnixMilli())))
	}
	if !(before.IsZero()) {
		params.Set("before", strconv.Itoa(int(before.UnixMilli())))
	}
	if limit > 0 && limit <= 100 {
		params.Set("limit", strconv.Itoa(limit))
	} else {
		return nil, errLimitValueExceedsMaxof100
	}
	type response struct {
		Code int                        `json:"code,string"`
		Msg  string                     `json:"msg"`
		Data []*DeliveryHistoryResponse `json:"data"`
	}
	var resp DeliveryHistoryResponse
	path := common.EncodeURLValues(publicDeliveryExerciseHistory, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// GetOpenInterest retrieve the total open interest for contracts on OKX
func (ok *Okx) GetOpenInterest(ctx context.Context, instType, uly, instId string) ([]*OpenInterestResponse, error) {
	params := url.Values{}
	if !(strings.EqualFold(instType, "SPOT") || strings.EqualFold(instType, "FUTURES") || strings.EqualFold(instType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", instType)
	}
	if uly != "" {
		params.Set("uly", uly)
	}
	if instId != "" {
		params.Set("instId", instId)
	}
	type response struct {
		Code int                     `json:"code,string"`
		Msg  string                  `json:"msg"`
		Data []*OpenInterestResponse `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(publicOpenInterestValues, params)
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// GetFundingRate  Retrieve funding rate.
func (ok *Okx) GetFundingRate(ctx context.Context, instrumentID string) (*FundingRateResponse, error) {
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	} else {
		return nil, errMissingInstrumentID
	}
	path := common.EncodeURLValues(publicFundingRate, params)
	type response struct {
		Code string                 `json:"code"`
		Data []*FundingRateResponse `json:"data"`
		Msg  string                 `json:"msg"`
	}
	var resp response
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, err
}

// GetFundingRateHistory retrieve funding rate history. This endpoint can retrieve data from the last 3 months.
func (ok *Okx) GetFundingRateHistory(ctx context.Context, instrumentID string, before time.Time, after time.Time, limit uint) ([]*FundingRateResponse, error) {
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	} else {
		return nil, errMissingInstrumentID
	}
	if !before.IsZero() {
		params.Set("before", strconv.Itoa(int(before.UnixMilli())))
	}
	if !after.IsZero() {
		params.Set("after", strconv.Itoa(int(after.UnixMilli())))
	}
	if limit > 0 && limit < 100 {
		params.Set("limit", strconv.Itoa(int(limit)))
	} else if limit > 0 {
		return nil, errLimitValueExceedsMaxof100
	}
	type response struct {
		Code string                 `json:"code"`
		Msg  string                 `json:"msg"`
		Data []*FundingRateResponse `json:"data"`
	}
	path := common.EncodeURLValues(publicFundingRateHistory, params)
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// GetLimitPrice retrieve the highest buy limit and lowest sell limit of the instrument.
func (ok *Okx) GetLimitPrice(ctx context.Context, instrumentID string) (*LimitPriceResponse, error) {
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	} else {
		return nil, errMissingInstrumentID
	}
	path := common.EncodeURLValues(publicLimitPath, params)
	type response struct {
		Code string                `json:"code"`
		Msg  string                `json:"msg"`
		Data []*LimitPriceResponse `json:"data"`
	}
	var resp response
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false); er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, errFundingRateHistoryNotFound
}

// GetOptionMarketData retrieve option market data.
func (ok *Okx) GetOptionMarketData(ctx context.Context, underlying string, expTime time.Time) ([]*OptionMarketDataResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("uly", underlying)
	} else {
		return nil, errMissingRequiredUnderlying
	}
	if !expTime.IsZero() {
		params.Set("expTime", fmt.Sprintf("%d%d%d", expTime.Year(), expTime.Month(), expTime.Day()))
	}
	path := common.EncodeURLValues(publicOptionalData, params)
	type response struct {
		Code string                      `json:"code"`
		Msg  string                      `json:"msg"`
		Data []*OptionMarketDataResponse `json:"data"`
	}
	var resp response
	return resp.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
}

// GetEstimatedDeliveryPrice retrieve the estimated delivery price which will only have a return value one hour before the delivery/exercise.
func (ok *Okx) GetEstimatedDeliveryPrice(ctx context.Context, instrumentID string) (*DeliveryEstimatedPrice, error) {
	var resp DeliveryEstimatedPriceResponse
	params := url.Values{}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	} else {
		return nil, errMissingRequiredParamInstID
	}
	path := common.EncodeURLValues(publicEstimatedPrice, params)
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false); er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, errors.New(resp.Msg)
}

// GetDiscountRateAndInterestFreeQuota retrieve discount rate level and interest-free quota.
func (ok *Okx) GetDiscountRateAndInterestFreeQuota(ctx context.Context, currency string, discountLevel int8) (*DiscountRate, error) {
	var response DiscountRateResponse
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if discountLevel > 0 && discountLevel < 5 {
		params.Set("discountLv", strconv.Itoa(int(discountLevel)))
	}
	path := common.EncodeURLValues(publicDiscountRate, params)
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false); er != nil {
		return nil, er
	}
	if len(response.Data) > 0 {
		return response.Data[0], nil
	}
	return nil, errors.New(response.Msg)
}

// GetSystemTime Retrieve API server time.
func (ok *Okx) GetSystemTime(ctx context.Context) (*time.Time, error) {
	type response struct {
		Code string       `json:"code"`
		Msg  string       `json:"msg"`
		Data []ServerTime `json:"data"`
	}
	var resp response
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, publicTime, nil, &resp, false); er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return &(resp.Data[0].Timestamp), nil
	}
	return nil, errors.New(resp.Msg)
}

// GetLiquidationOrders retrieve information on liquidation orders in the last day.
func (ok *Okx) GetLiquidationOrders(ctx context.Context, arg *LiquidationOrderRequestParams) (*LiquidationOrder, error) {
	params := url.Values{}
	if !(strings.EqualFold(arg.InstrumentType, "MARGIN") || strings.EqualFold(arg.InstrumentType, "FUTURES") || strings.EqualFold(arg.InstrumentType, "SWAP") || strings.EqualFold(arg.InstrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", arg.InstrumentType)
	}
	if strings.EqualFold(arg.MarginMode, "isolated") || strings.EqualFold(arg.MarginMode, "cross") {
		params.Set("mgnMode", arg.MarginMode)
	}
	if strings.EqualFold(arg.InstrumentType, "MARGIN") && arg.InstrumentID != "" {
		params.Set("instId", arg.InstrumentID)
	} else if strings.EqualFold("MARGIN", arg.InstrumentType) && arg.Currency.String() != "" {
		params.Set("ccy", arg.Currency.String())
	} else {
		return nil, errEitherInstIDOrCcyIsRequired
	}
	if (strings.EqualFold(arg.InstrumentType, "FUTURES") || strings.EqualFold(arg.InstrumentType, "SWAP") || strings.EqualFold(arg.InstrumentType, "OPTION")) && arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	}
	if strings.EqualFold(arg.InstrumentType, "FUTURES") && (strings.EqualFold(arg.Alias, "this_week") || strings.EqualFold(arg.Alias, "next_week ") || strings.EqualFold(arg.Alias, "quarter") || strings.EqualFold(arg.Alias, "next_quarter")) {
		params.Set("alias", arg.Alias)
	}
	if ((strings.EqualFold(arg.InstrumentType, "FUTURES") || strings.EqualFold(arg.InstrumentType, "SWAP")) &&
		strings.EqualFold(arg.Alias, "unfilled")) || strings.EqualFold(arg.Alias, "filled ") {
		params.Set("alias", arg.Underlying)
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 && arg.Limit < 100 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	path := common.EncodeURLValues(publicLiquidationOrders, params)
	var response LiquidationOrderResponse
	println("No ERROR MESSAGE UNTIL NOW")
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	if len(response.Data) > 0 {
		return response.Data[0], nil
	}
	return nil, errLiquidationOrderResponseNotFound
}

// GetMarkPrice  Retrieve mark price.
func (ok *Okx) GetMarkPrice(ctx context.Context, instrumentType, underlying, instrumentID string) ([]*MarkPrice, error) {
	params := url.Values{}
	if !(strings.EqualFold(instrumentType, "MARGIN") ||
		strings.EqualFold(instrumentType, "FUTURES") ||
		strings.EqualFold(instrumentType, "SWAP") ||
		strings.EqualFold(instrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", instrumentType)
	}
	if underlying != "" {
		params.Set("uly", underlying)
	}
	if instrumentID != "" {
		params.Set("instId", instrumentID)
	}
	var response MarkPriceResponse
	path := common.EncodeURLValues(publicMarkPrice, params)
	return response.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
}

// GetPositionTiers retrieve position tiers information，maximum leverage depends on your borrowings and margin ratio.
func (ok *Okx) GetPositionTiers(ctx context.Context, instrumentType, tradeMode, underlying, instrumentID, tiers string) ([]*PositionTiers, error) {
	params := url.Values{}
	if !(strings.EqualFold(instrumentType, "MARGIN") ||
		strings.EqualFold(instrumentType, "FUTURES") ||
		strings.EqualFold(instrumentType, "SWAP") ||
		strings.EqualFold(instrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", instrumentType)
	}
	if !(strings.EqualFold(tradeMode, "cross") || strings.EqualFold(tradeMode, "isolated")) {
		return nil, errIncorrectRequiredParameterTradeMode
	} else {
		params.Set("tdMode", tradeMode)
	}
	if (!strings.EqualFold(instrumentType, "MARGIN")) && underlying != "" {
		params.Set("uly", underlying)
	}
	if strings.EqualFold(instrumentType, "MARGIN") && instrumentID != "" {
		params.Set("instId", instrumentID)
	} else if strings.EqualFold(instrumentType, "MARGIN") {
		return nil, errMissingInstrumentID
	}
	if tiers != "" {
		params.Set("tiers", tiers)
	}
	var response PositionTiersResponse
	path := common.EncodeURLValues(publicPositionTiers, params)
	return response.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
}

// GetInterestRate retrives an interest rate and loan quota information for various currencies.
func (ok *Okx) GetInterestRateAndLoanQuota(ctx context.Context) (map[string][]*InterestRateLoanQuotaItem, error) {
	var response InterestRateLoanQuotaResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, publicInterestRateAndLoanQuota, nil, &response, false)
	if err != nil {
		return nil, err
	} else if len(response.Data) == 0 {
		return nil, errInterestRateAndLoanQuotaNotFound
	}
	return response.Data[0], nil
}

// GetInterestRate retrives an interest rate and loan quota information for VIP users of various currencies.
func (ok *Okx) GetInterestRateAndLoanQuotaForVIPLoans(ctx context.Context) (*VIPInterestRateAndLoanQuotaInformation, error) {
	var response VIPInterestRateAndLoanQuotaInformationResponse
	err := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, publicVIPInterestRateAndLoanQuota, nil, &response, false)
	if err != nil {
		return nil, err
	} else if len(response.Data) == 0 {
		return nil, errInterestRateAndLoanQuotaNotFound
	}
	return response.Data[0], nil
}

// GetPublicUnderlyings returns list of underlyings for various instrument types.
func (ok *Okx) GetPublicUnderlyings(ctx context.Context, instrumentType string) ([]string, error) {
	params := url.Values{}
	if !(strings.EqualFold(instrumentType, "FUTURES") ||
		strings.EqualFold(instrumentType, "SWAP") ||
		strings.EqualFold(instrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	path := common.EncodeURLValues(publicUnderlyings, params)
	type response struct {
		Code string     `json:"code"`
		Msg  string     `json:"msg"`
		Data [][]string `json:"data"`
	}
	var resp response
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false); er != nil {
		return nil, er
	}
	if len(resp.Data) > 0 {
		return resp.Data[0], nil
	}
	return nil, errUnderlyingsForSpecifiedInstTypeNofFound
}

// Trading data Endpoints

// GetSupportCoins retrieve the currencies supported by the trading data endpoints
func (ok *Okx) GetSupportCoins(ctx context.Context) (*SupportedCoinsData, error) {
	var response SupportedCoinsResponse
	return response.Data, ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, tradingDataSupportedCoins, nil, &response, false)
}

// GetTakerVolume retrieve the taker volume for both buyers and sellers.
func (ok *Okx) GetTakerVolume(ctx context.Context, currency, instrumentType string, begin, end time.Time, period kline.Interval) ([]*TakerVolume, error) {
	params := url.Values{}
	if !(strings.EqualFold(instrumentType, "CONTRACTS") ||
		strings.EqualFold(instrumentType, "SPOT")) {
		return nil, errMissingRequiredArgInstType
	} else if strings.EqualFold(instrumentType, "FUTURES") ||
		strings.EqualFold(instrumentType, "MARGIN") ||
		strings.EqualFold(instrumentType, "SWAP") ||
		strings.EqualFold(instrumentType, "OPTION") {
		return nil, fmt.Errorf("instrument type %s is not allowed for this query", instrumentType)
	} else {
		params.Set("instType", strings.ToUpper(instrumentType))
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(end.UnixMilli(), 10))
	}
	path := common.EncodeURLValues(tradingTakerVolume, params)
	var response TakerVolumeResponse
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	takerVolumes := []*TakerVolume{}
	for x := range response.Data {
		if len(response.Data[x]) != 3 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil {
			continue
		}
		sellVolume, er := strconv.ParseFloat(response.Data[x][1], 64)
		if er != nil {
			continue
		}
		buyVolume, er := strconv.ParseFloat(response.Data[x][2], 64)
		if er != nil {
			continue
		}
		takerVolume := &TakerVolume{
			Timestamp:  time.UnixMilli(int64(timestamp)),
			SellVolume: sellVolume,
			BuyVolume:  buyVolume,
		}
		takerVolumes = append(takerVolumes, takerVolume)
	}
	return takerVolumes, nil
}

// GetInsuranceFund returns insurance fund balance informations.
func (ok *Okx) GetInsuranceFundInformations(ctx context.Context, arg InsuranceFundInformationRequestParams) (*InsuranceFundInformation, error) {
	params := url.Values{}
	if !(strings.EqualFold(arg.InstrumentType, "FUTURES") ||
		strings.EqualFold(arg.InstrumentType, "MARGIN") ||
		strings.EqualFold(arg.InstrumentType, "SWAP") ||
		strings.EqualFold(arg.InstrumentType, "OPTION")) {
		return nil, errMissingRequiredArgInstType
	} else {
		params.Set("instType", strings.ToUpper(arg.InstrumentType))
	}
	if strings.EqualFold(arg.Type, "liquidation_balance_deposit") ||
		strings.EqualFold(arg.Type, "bankruptcy_loss") ||
		strings.EqualFold(arg.Type, "platform_revenue ") {
		params.Set("type", arg.Type)
	}
	if (!strings.EqualFold(arg.InstrumentType, "MARGIN")) && arg.Underlying != "" {
		params.Set("uly", arg.Underlying)
	} else if !strings.EqualFold(arg.InstrumentType, "MARGIN") {
		return nil, errParameterUnderlyingCanNotBeEmpty
	}
	if (strings.EqualFold(arg.InstrumentType, "MARGIN")) && arg.Currency != "" {
		params.Set("ccy", arg.Currency)
	}
	if !arg.Before.IsZero() {
		params.Set("before", strconv.FormatInt(arg.Before.UnixMilli(), 10))
	}
	if !arg.After.IsZero() {
		params.Set("after", strconv.FormatInt(arg.After.UnixMilli(), 10))
	}
	if arg.Limit > 0 && arg.Limit < 100 {
		params.Set("limit", strconv.Itoa(int(arg.Limit)))
	}
	var response InsuranceFundInformationResponse
	path := common.EncodeURLValues(publicInsuranceFunds, params)
	if er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false); er != nil {
		return nil, er
	}
	if len(response.Data) > 0 {
		return response.Data[0], nil
	}
	return nil, errInsuranceFundInformationNotFound
}

// GetMarginLendingRatio retrieve the ratio of cumulative amount between currency margin quote currency and base currency.
func (ok *Okx) GetMarginLendingRatio(ctx context.Context, currency string, begin, end time.Time, period kline.Interval) ([]*MarginLendRatioItem, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	var response MarginLendRatioResponse
	path := common.EncodeURLValues(tradingMarginLoanRatio, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	lendingRatios := []*MarginLendRatioItem{}
	for x := range response.Data {
		if len(response.Data[x]) != 2 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil || timestamp <= 0 {
			continue
		}
		ratio, er := strconv.ParseFloat(response.Data[x][0], 64)
		if er != nil || ratio <= 0 {
			continue
		}
		lendRatio := &MarginLendRatioItem{
			Timestamp:       time.UnixMilli(int64(timestamp)),
			MarginLendRatio: ratio,
		}
		lendingRatios = append(lendingRatios, lendRatio)
	}
	return lendingRatios, nil
}

// GetLongShortRatio retrieve the ratio of users with net long vs net short positions for futures and perpetual swaps.
func (ok *Okx) GetLongShortRatio(ctx context.Context, currency string, begin, end time.Time, period kline.Interval) ([]*LongShortRatio, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	var response LongShortRatioResponse
	path := common.EncodeURLValues(longShortAccountRatio, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	ratios := []*LongShortRatio{}
	for x := range response.Data {
		if len(response.Data[x]) != 2 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil || timestamp <= 0 {
			continue
		}
		ratio, er := strconv.ParseFloat(response.Data[x][0], 64)
		if er != nil || ratio <= 0 {
			continue
		}
		dratio := &LongShortRatio{
			Timestamp:       time.UnixMilli(int64(timestamp)),
			MarginLendRatio: ratio,
		}
		ratios = append(ratios, dratio)
	}
	return ratios, nil
}

// GetContractsOpenInterestAndVolume retrieve the open interest and trading volume for futures and perpetual swaps.
func (ok *Okx) GetContractsOpenInterestAndVolume(
	ctx context.Context, currency string,
	begin, end time.Time, period kline.Interval) ([]*OpenInterestVolume, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	if !begin.IsZero() {
		params.Set("begin", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("end", strconv.FormatInt(begin.UnixMilli(), 10))
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	openInterestVolumes := []*OpenInterestVolume{}
	var response OpenInterestVolumeResponse
	path := common.EncodeURLValues(contractOpenInterestVolume, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	for x := range response.Data {
		if len(response.Data[x]) != 3 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil || timestamp <= 0 {
			continue
		}
		openInterest, er := strconv.Atoi(response.Data[x][1])
		if er != nil || openInterest <= 0 {
			continue
		}
		volumen, err := strconv.Atoi(response.Data[x][2])
		if err != nil {
			continue
		}
		openInterestVolume := &OpenInterestVolume{
			Timestamp:    time.UnixMilli(int64(timestamp)),
			Volume:       float64(volumen),
			OpenInterest: float64(openInterest),
		}
		openInterestVolumes = append(openInterestVolumes, openInterestVolume)
	}
	return openInterestVolumes, nil
}

// GetOptionsOpenInterestAndVolume retrieve the open interest and trading volume for options.
func (ok *Okx) GetOptionsOpenInterestAndVolume(ctx context.Context, currency string,
	period kline.Interval) ([]*OpenInterestVolume, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	openInterestVolumes := []*OpenInterestVolume{}
	var response OpenInterestVolumeResponse
	path := common.EncodeURLValues(optionOpenInterestVolume, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	for x := range response.Data {
		if len(response.Data[x]) != 3 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil || timestamp <= 0 {
			continue
		}
		openInterest, er := strconv.Atoi(response.Data[x][1])
		if er != nil || openInterest <= 0 {
			continue
		}
		volumen, err := strconv.Atoi(response.Data[x][2])
		if err != nil {
			continue
		}
		openInterestVolume := &OpenInterestVolume{
			Timestamp:    time.UnixMilli(int64(timestamp)),
			Volume:       float64(volumen),
			OpenInterest: float64(openInterest),
		}
		openInterestVolumes = append(openInterestVolumes, openInterestVolume)
	}
	return openInterestVolumes, nil
}

// GetPutCallRatio retrieve the open interest ration and trading volume ratio of calls vs puts.
func (ok *Okx) GetPutCallRatio(ctx context.Context, currency string,
	period kline.Interval) ([]*OpenInterestVolumeRatio, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	openInterestVolumeRatios := []*OpenInterestVolumeRatio{}
	var response OpenInterestVolumeResponse
	path := common.EncodeURLValues(optionOpenInterestVolumeRatio, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &response, false)
	if er != nil {
		return nil, er
	}
	for x := range response.Data {
		if len(response.Data[x]) != 3 {
			continue
		}
		timestamp, er := strconv.Atoi(response.Data[x][0])
		if er != nil || timestamp <= 0 {
			continue
		}
		openInterest, er := strconv.Atoi(response.Data[x][1])
		if er != nil || openInterest <= 0 {
			continue
		}
		volumen, err := strconv.Atoi(response.Data[x][2])
		if err != nil {
			continue
		}
		openInterestVolume := &OpenInterestVolumeRatio{
			Timestamp:         time.UnixMilli(int64(timestamp)),
			VolumeRatio:       float64(volumen),
			OpenInterestRatio: float64(openInterest),
		}
		openInterestVolumeRatios = append(openInterestVolumeRatios, openInterestVolume)
	}
	return openInterestVolumeRatios, nil
}

// GetOpenInterestAndVolume retrieve the open interest and trading volume of calls and puts for each upcoming expiration.
func (ok *Okx) GetOpenInterestAndVolumeExpiry(ctx context.Context, currency string, period kline.Interval) ([]*ExpiryOpenInterestAndVolume, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	type response struct {
		Code string      `json:"code"`
		Msg  string      `json:"msg"`
		Data [][6]string `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(optionOpenInterestVolumeExpiry, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if er != nil {
		return nil, er
	}
	var volumes []*ExpiryOpenInterestAndVolume
	for x := range resp.Data {
		if len(resp.Data[x]) != 6 {
			continue
		}
		timestamp, er := strconv.Atoi(resp.Data[x][0])
		if er != nil {
			continue
		}
		var expiryTime time.Time
		expTime := resp.Data[x][1]
		if expTime != "" && len(expTime) == 8 {
			year, er := strconv.Atoi(expTime[0:4])
			if er != nil {
				continue
			}
			month, er := strconv.Atoi(expTime[4:6])
			var months string
			var days string
			if month <= 9 {
				months = fmt.Sprintf("0%d", month)
			} else {
				months = strconv.Itoa(month)
			}
			if er != nil {
				continue
			}
			day, er := strconv.Atoi(expTime[6:])
			if day <= 9 {
				days = fmt.Sprintf("0%d", day)
			} else {
				days = strconv.Itoa(day)
			}
			if er != nil {
				continue
			}
			expiryTime, er = time.Parse("2006-01-02", fmt.Sprintf("%d-%s-%s", year, months, days))
			if er != nil {
				continue
			}
		}
		calloi, er := strconv.ParseFloat(resp.Data[x][2], 64)
		if er != nil {
			continue
		}
		putoi, er := strconv.ParseFloat(resp.Data[x][3], 64)
		if er != nil {
			continue
		}
		callvol, er := strconv.ParseFloat(resp.Data[x][4], 64)
		if er != nil {
			continue
		}
		putvol, er := strconv.ParseFloat(resp.Data[x][5], 64)
		if er != nil {
			continue
		}
		volume := &ExpiryOpenInterestAndVolume{
			Timestamp:        time.UnixMilli(int64(timestamp)),
			ExpiryTime:       expiryTime,
			CallOpenInterest: calloi,
			PutOpenInterest:  putoi,
			CallVolume:       callvol,
			PutVolume:        putvol,
		}
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// GetOpenInterestAndVolumeStrike retrieve the taker volume for both buyers and sellers of calls and puts.
func (ok *Okx) GetOpenInterestAndVolumeStrike(ctx context.Context, currency string,
	expTime time.Time, period kline.Interval) ([]*StrikeOpenInterestAndVolume, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	if !expTime.IsZero() {
		var months string
		var days string
		if expTime.Month() <= 9 {
			months = fmt.Sprintf("0%d", expTime.Month())
		} else {
			months = strconv.Itoa(int(expTime.Month()))
		}
		if expTime.Day() <= 9 {
			days = fmt.Sprintf("0%d", expTime.Day())
		} else {
			days = strconv.Itoa(int(expTime.Day()))
		}
		params.Set("expTime", fmt.Sprintf("%d%s%s", expTime.Year(), months, days))
	} else {
		return nil, errMissingExpiryTimeParameter
	}
	type response struct {
		Code string      `json:"code"`
		Msg  string      `json:"msg"`
		Data [][6]string `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(optionOpenInterestVolumeStrike, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if er != nil {
		return nil, er
	}
	var volumes []*StrikeOpenInterestAndVolume
	for x := range resp.Data {
		if len(resp.Data[x]) != 6 {
			continue
		}
		timestamp, er := strconv.Atoi(resp.Data[x][0])
		if er != nil {
			continue
		}
		strike, er := strconv.ParseInt(resp.Data[x][1], 10, 64)
		if er != nil {
			continue
		}
		calloi, er := strconv.ParseFloat(resp.Data[x][2], 64)
		if er != nil {
			continue
		}
		putoi, er := strconv.ParseFloat(resp.Data[x][3], 64)
		if er != nil {
			continue
		}
		callvol, er := strconv.ParseFloat(resp.Data[x][4], 64)
		if er != nil {
			continue
		}
		putvol, er := strconv.ParseFloat(resp.Data[x][5], 64)
		if er != nil {
			continue
		}
		volume := &StrikeOpenInterestAndVolume{
			Timestamp:        time.UnixMilli(int64(timestamp)),
			Strike:           strike,
			CallOpenInterest: calloi,
			PutOpenInterest:  putoi,
			CallVolume:       callvol,
			PutVolume:        putvol,
		}
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

// GetTakerFlow shows the relative buy/sell volume for calls and puts.
// It shows whether traders are bullish or bearish on price and volatility
func (ok *Okx) GetTakerFlow(ctx context.Context, currency string, period kline.Interval) (*CurrencyTakerFlow, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("ccy", currency)
	}
	interval := ok.GetIntervalEnum(period)
	if interval != "" {
		params.Set("period", interval)
	}
	type response struct {
		Msg  string    `json:"msg"`
		Code string    `json:"code"`
		Data [7]string `json:"data"`
	}
	var resp response
	path := common.EncodeURLValues(takerBlockVolume, params)
	er := ok.SendHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, nil, &resp, false)
	if er != nil {
		return nil, er
	}
	timestamp, era := strconv.ParseInt(resp.Data[0], 10, 64)
	callbuyvol, erb := strconv.ParseFloat(resp.Data[1], 64)
	callselvol, erc := strconv.ParseFloat(resp.Data[2], 64)
	putbutvol, erd := strconv.ParseFloat(resp.Data[3], 64)
	putsellvol, ere := strconv.ParseFloat(resp.Data[4], 64)
	callblockvol, erf := strconv.ParseFloat(resp.Data[5], 64)
	putblockvol, erg := strconv.ParseFloat(resp.Data[6], 64)
	if era != nil || erb != nil || erc != nil || erd != nil || ere != nil || erf != nil || erg != nil {
		return nil, errParsingResponseError
	}
	return &CurrencyTakerFlow{
		Timestamp:       time.UnixMilli(timestamp),
		CallBuyVolume:   callbuyvol,
		CallSellVolume:  callselvol,
		PutBuyVolume:    putbutvol,
		PutSellVolume:   putsellvol,
		CallBlockVolume: callblockvol,
		PutBlockVolume:  putblockvol,
	}, nil
}

// Stauts Endpoints
// https://www.okx.com/docs-v5/en/#rest-api-status
// The response out of this endpoint is an XML file. So, for the timebeing, it is not implemented.

// SendHTTPRequest sends an authenticated http request to a desired
// path with a JSON payload (of present)
// URL arguments must be in the request path and not as url.URL values
func (ok *Okx) SendHTTPRequest(ctx context.Context, ep exchange.URL, httpMethod, requestPath string, data, result interface{}, authenticated bool) (err error) {
	endpoint, err := ok.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var intermediary json.RawMessage
	newRequest := func() (*request.Item, error) {
		utcTime := time.Now().UTC().Format(time.RFC3339)
		payload := []byte("")

		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		path := endpoint + requestPath
		headers := make(map[string]string)
		headers["Content-Type"] = "application/json"
		if authenticated {
			var creds *exchange.Credentials
			creds, err = ok.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			signPath := fmt.Sprintf("/%v%v", okxAPIPath, requestPath)
			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256,
				[]byte(utcTime+httpMethod+signPath+string(payload)),
				[]byte(creds.Secret))
			if err != nil {
				return nil, err
			}
			headers["OK-ACCESS-KEY"] = creds.Key
			headers["OK-ACCESS-SIGN"] = crypto.Base64Encode(hmac)
			headers["OK-ACCESS-TIMESTAMP"] = utcTime
			headers["OK-ACCESS-PASSPHRASE"] = creds.ClientID
			headers["x-simulated-trading"] = "1"
		}
		return &request.Item{
			Method:        strings.ToUpper(httpMethod),
			Path:          path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &intermediary,
			AuthRequest:   authenticated,
			Verbose:       ok.Verbose,
			HTTPDebugging: ok.HTTPDebugging,
			HTTPRecording: ok.HTTPRecording,
		}, nil
	}
	err = ok.SendPayload(ctx, request.Unset, newRequest)
	if err != nil {
		return err
	}
	type errCapFormat struct {
		Error        int64  `json:"error_code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
		Result       bool   `json:"result,string,omitempty"`
	}
	errCap := errCapFormat{Result: true}
	err = json.Unmarshal(intermediary, &errCap)
	if err == nil {
		if errCap.ErrorMessage != "" {
			return fmt.Errorf("error: %v", errCap.ErrorMessage)
		}
		if errCap.Error > 0 {
			return fmt.Errorf("sendHTTPRequest error - %s",
				ok.ErrorCodes[strconv.FormatInt(errCap.Error, 10)])
		}
		if !errCap.Result {
			return errors.New("unspecified error occurred")
		}
	}
	return json.Unmarshal(intermediary, result)
}
