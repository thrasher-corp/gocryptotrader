package bitstamp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	bitstampAPIURL = "https://www.bitstamp.net/api"
	tradeBaseURL   = "https://www.bitstamp.net/trade/"

	bitstampRateInterval = time.Minute * 10
	bitstampRequestRate  = 10000

	maxWithdrawalRequestsTimeDelta = 50000000 * time.Second
	maxResultLimit                 = 1000
	maxResultOffset                = 200000
	maxFundingRateHistoryLimit     = 100
	maxClientOrderIDLength         = 180
	maxTrailingDelta               = 2000
	minLeverage                    = 1
	maxLeverage                    = 25

	formContentType = "application/x-www-form-urlencoded"
	jsonContentType = "application/json"
	trueValue       = "True"
	falseValue      = "False"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitstamp
type Exchange struct {
	exchange.Base

	// websocketUserID is the user ID of the last websocket token, which names the private channels
	websocketUserID atomic.Uint64
	// orderEventIDs holds the ID of the last private order event processed for each market and order source
	orderEventIDs map[string]string
	orderEventMu  sync.Mutex
}

var (
	errAPIResponse              = errors.New("API returned an error")
	errMalformedOrderbookLevel  = errors.New("malformed orderbook level")
	errMalformedTransaction     = errors.New("malformed transaction")
	errNullNumber               = errors.New("number is null")
	errInvalidOrderbookGrouping = errors.New("invalid orderbook grouping")
	errInvalidTransactionPeriod = errors.New("invalid transaction time period")
	errInvalidInterval          = errors.New("invalid interval")
	errInvalidLimit             = errors.New("invalid limit")
	errInvalidOffset            = errors.New("invalid offset")
	errInvalidTimeDelta         = errors.New("time delta must be between 1 and 50000000 seconds")
	errInvalidDateTime          = errors.New("invalid date time")
	errNoResponse               = errors.New("no response data")
	errOrderSourceRequired      = errors.New("order source is required")
	errNetworkRequired          = errors.New("network is required")
	errWithdrawalIDRequired     = errors.New("withdrawal ID is required")
	errSubAccountRequired       = errors.New("sub account is required")
	errDepositIDRequired        = errors.New("deposit ID is required")
	errContactIDRequired        = errors.New("contact ID is required")
	errSatoshiTestIDRequired    = errors.New("satoshi test ID is required")
	errRegistrationIDRequired   = errors.New("registration ID is required")
	errPublicKeyRequired        = errors.New("extended public key is required")
	errContactInfoRequired      = errors.New("retail or corporate contact information is required")
	errEarnTypeRequired         = errors.New("earn type is required")
	errEarnTermRequired         = errors.New("earn term is required")
	errEarnSettingRequired      = errors.New("earn setting is required")
	errMarginModeRequired       = errors.New("margin mode is required")
	errLeverageRequired         = errors.New("leverage is required")
	errOrderTypeRequired        = errors.New("order type is required")
	errPositionIDRequired       = errors.New("position ID is required")
	errPairOrOrderIDRequired    = errors.New("currency pair or order ID is required")
	errCollateralRequired       = errors.New("exactly one of target collateral or collateral deltas is required")
	errBankDetailsRequired      = errors.New("bank account details are required")
	errContactNameRequired      = errors.New("retail contact first and last names are required")
	errCompanyNameRequired      = errors.New("corporate contact company name is required")
	errIdentificationRequired   = errors.New("contact identification is required")
	errBeneficiaryRequired      = errors.New("beneficiary ID or details are required for a third party withdrawal")
	errOriginatorRequired       = errors.New("originator ID or details are required for a third party deposit")
	errContactUUIDRequired      = errors.New("contact UUID is required for a third party address")
	errOrderFlagsConflict       = errors.New("only one of the daily, immediate or cancel, fill or kill, maker or cancel and good till date flags may be set")
	errLimitPriceConflict       = errors.New("limit price cannot be combined with an order flag")
	errInvalidPrice             = errors.New("price must not be negative")
	errInvalidOrderSubtype      = errors.New("invalid order subtype")
	errInvalidTrailingDelta     = errors.New("trailing delta must be between 1 and 2000 basis points")
	errTrailingDeltaRequired    = errors.New("trailing delta is required with an activation price")
	errClientOrderIDTooLong     = errors.New("client order ID exceeds 180 characters")
	errReduceOnlyUnsupported    = errors.New("reduce only is not supported with these order parameters")
	errInvalidLeverage          = errors.New("leverage must be between 1 and 25")
	errInvalidCollateral        = errors.New("collateral amounts must not be negative")
)

// Order subtypes accepted by the limit and market order endpoints
var (
	limitOrderSubtypes  = []string{OrderSubtypeLimit, OrderSubtypeStopLossLimit, OrderSubtypeTakeProfitLimit, OrderSubtypeTrailingStopLossLimit, OrderSubtypeTrailingTakeProfitLimit}
	marketOrderSubtypes = []string{OrderSubtypeMarket, OrderSubtypeStopLoss, OrderSubtypeTakeProfit, OrderSubtypeTrailingStopLoss, OrderSubtypeTrailingTakeProfit}
)

// ohlcSteps are the candle intervals supported by the OHLC endpoint
var ohlcSteps = []kline.Interval{kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.FourHour, kline.SixHour, kline.TwelveHour, kline.OneDay, kline.ThreeDay}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		tradingFee, err := e.getTradingFee(ctx, feeBuilder)
		if err != nil {
			return 0, fmt.Errorf("error getting trading fee: %w", err)
		}
		fee = tradingFee
	case exchange.CryptocurrencyWithdrawalFee:
		withdrawalFee, err := e.GetWithdrawalFee(ctx, feeBuilder.Pair.Base, "")
		if err != nil {
			return 0, fmt.Errorf("error getting withdrawal fee: %w", err)
		}
		fee = withdrawalFee.Fee.Float64()
	case exchange.CryptocurrencyDepositFee:
		fee = 0
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getTradingFee returns a trading fee based on a currency
func (e *Exchange) getTradingFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	tradingFee, err := e.GetTradingFee(ctx, feeBuilder.Pair)
	if err != nil {
		return 0, err
	}
	fee := tradingFee.Fees.Taker.Float64()
	if feeBuilder.IsMaker {
		fee = tradingFee.Fees.Maker.Float64()
	}
	return fee / 100 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.0009

	if fee < 15 {
		return 15
	}
	return fee
}

// getInternationalBankDepositFee returns international deposit fee
func getInternationalBankDepositFee(amount float64) float64 {
	fee := amount * 0.0005

	if fee < 7.5 {
		return 7.5
	}
	if fee > 300 {
		return 300
	}
	return fee
}

// GetCurrencies returns all listed currencies with their deposit and withdrawal networks
func (e *Exchange) GetCurrencies(ctx context.Context) ([]CurrencyResponse, error) {
	var resp []CurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/currencies/", nil, &resp)
}

// GetTickers returns ticker data for all markets
func (e *Exchange) GetTickers(ctx context.Context) ([]TickerResponse, error) {
	var resp []TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker/", nil, &resp)
}

// GetTicker returns ticker data for a market
func (e *Exchange) GetTicker(ctx context.Context, pair currency.Pair) (*TickerResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetHourlyTicker returns ticker data for a market over the last hour
// The open, high, low, volume and volume weighted average price cover the last hour, and the 24 hour fields are unset
func (e *Exchange) GetHourlyTicker(ctx context.Context, pair currency.Pair) (*TickerResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ticker_hour/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetOrderbook returns the orderbook for a market
func (e *Exchange) GetOrderbook(ctx context.Context, pair currency.Pair, grouping OrderbookGrouping) (*OrderbookResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if grouping > OrderbookUngroupedWithOrderIDs {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderbookGrouping, grouping)
	}
	params := url.Values{}
	params.Set("group", strconv.FormatUint(uint64(grouping), 10))
	var resp *OrderbookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/order_book/"+formatMarketSymbol(pair)+"/", params, &resp)
}

// GetTransactions returns public trades for a market
// timePeriod may be TransactionPeriodMinute, TransactionPeriodHour or TransactionPeriodDay, and defaults to an hour when empty
func (e *Exchange) GetTransactions(ctx context.Context, pair currency.Pair, timePeriod string) ([]TransactionResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	switch timePeriod {
	case "":
	case TransactionPeriodMinute, TransactionPeriodHour, TransactionPeriodDay:
		params.Set("time", timePeriod)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTransactionPeriod, timePeriod)
	}
	var resp []TransactionResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/transactions/"+formatMarketSymbol(pair)+"/", params, &resp)
}

// GetMarkets returns all available markets
// entity optionally restricts the markets to those available to a broker entity, e.g. EntityEuropeSA
func (e *Exchange) GetMarkets(ctx context.Context, entity string) ([]MarketResponse, error) {
	params := url.Values{}
	if entity != "" {
		params.Set("entity", entity)
	}
	var resp []MarketResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/markets/", params, &resp)
}

// GetOHLC returns candle data for a market
func (e *Exchange) GetOHLC(ctx context.Context, req *OHLCRequest) (*OHLCResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(ohlcSteps, req.Step) {
		return nil, fmt.Errorf("%w: %s", errInvalidInterval, req.Step)
	}
	if req.Limit == 0 || req.Limit > maxResultLimit {
		return nil, fmt.Errorf("%w: %d must be between 1 and %d", errInvalidLimit, req.Limit, maxResultLimit)
	}
	if !req.Start.IsZero() && !req.End.IsZero() && req.Start.After(req.End) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	params.Set("step", strconv.FormatInt(int64(req.Step.Duration()/time.Second), 10))
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	if !req.Start.IsZero() {
		params.Set("start", strconv.FormatInt(req.Start.Unix(), 10))
	}
	if !req.End.IsZero() {
		params.Set("end", strconv.FormatInt(req.End.Unix(), 10))
	}
	if req.ExcludeCurrentCandle {
		params.Set("exclude_current_candle", "true")
	}
	var resp *OHLCResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/ohlc/"+formatMarketSymbol(req.Pair)+"/", params, &resp)
}

// GetEURUSDConversionRate returns the EUR/USD conversion rate
func (e *Exchange) GetEURUSDConversionRate(ctx context.Context) (*EURUSDConversionRateResponse, error) {
	var resp *EURUSDConversionRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/eur_usd/", nil, &resp)
}

// GetFundingRate returns the current funding rate for a perpetual market
func (e *Exchange) GetFundingRate(ctx context.Context, pair currency.Pair) (*FundingRateResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *FundingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/funding_rate/"+formatMarketSymbol(pair)+"/", nil, &resp)
}

// GetFundingRateHistory returns up to 100 funding rates for a perpetual market, oldest first, from up to 30 days ago
// Limit keeps the most recent rates within the time range, so earlier rates are fetched by moving Until back
func (e *Exchange) GetFundingRateHistory(ctx context.Context, req *FundingRateHistoryRequest) (*FundingRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Limit > maxFundingRateHistoryLimit {
		return nil, fmt.Errorf("%w: %d must not exceed %d", errInvalidLimit, req.Limit, maxFundingRateHistoryLimit)
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	var resp *FundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/funding_rate_history/"+formatMarketSymbol(req.Pair)+"/", params, &resp)
}

// GetVASPs returns the Virtual Asset Service Providers used for Travel Rule compliance
func (e *Exchange) GetVASPs(ctx context.Context, req *PaginationRequest) (*VASPListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := paginationParams(req)
	var resp *VASPListResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/v2/travel_rule/vasps/", params, &resp)
}

// GetAccountBalances returns the balances of all currencies
func (e *Exchange) GetAccountBalances(ctx context.Context) ([]AccountBalanceResponse, error) {
	var resp []AccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_balances/", nil, nil, &resp)
}

// GetAccountBalance returns the balance of a currency
func (e *Exchange) GetAccountBalance(ctx context.Context, c currency.Code) (*AccountBalanceResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AccountBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_balances/"+c.Lower().String()+"/", nil, nil, &resp)
}

// GetTradingFees returns the trading fees for all markets
func (e *Exchange) GetTradingFees(ctx context.Context) ([]TradingFeeResponse, error) {
	var resp []TradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/trading/", nil, nil, &resp)
}

// GetTradingFee returns the trading fees for a market
func (e *Exchange) GetTradingFee(ctx context.Context, pair currency.Pair) (*TradingFeeResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/trading/"+formatMarketSymbol(pair)+"/", nil, nil, &resp)
}

// GetWithdrawalFees returns the withdrawal fees for all currencies
func (e *Exchange) GetWithdrawalFees(ctx context.Context) ([]WithdrawalFeeResponse, error) {
	var resp []WithdrawalFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/withdrawal/", nil, nil, &resp)
}

// GetWithdrawalFee returns the withdrawal fee for a currency
// network is optional and selects the currency network the fee applies to
func (e *Exchange) GetWithdrawalFee(ctx context.Context, c currency.Code, network string) (*WithdrawalFeeResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if network != "" {
		params.Set("network", network)
	}
	var resp *WithdrawalFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/withdrawal/"+c.Lower().String()+"/", nil, params, &resp)
}

// GetOrderStatus returns the status of an order
// Closed orders are only available for 30 days
func (e *Exchange) GetOrderStatus(ctx context.Context, req *OrderStatusRequest) (*OrderStatusResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	if req.OmitTransactions {
		params.Set("omit_transactions", trueValue)
	}
	var resp *OrderStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/order_status/", nil, params, &resp)
}

// GetAccountOrderData returns up to 1000 of the account's order events for a market, oldest first, to recover events
// missed on the private orders websocket channel; events are retained for 30 days
func (e *Exchange) GetAccountOrderData(ctx context.Context, req *OrderEventsRequest) ([]OrderEventResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderSource == "" {
		return nil, errOrderSourceRequired
	}
	params, err := orderEventParams(req)
	if err != nil {
		return nil, err
	}
	params.Set("order_source", req.OrderSource)
	var resp []OrderEventResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/account_order_data/", nil, params, &resp)
}

// GetOrderData returns up to 1000 public order events for a market, oldest first, to recover events missed on the
// live orders websocket channel; events are retained for 30 days
func (e *Exchange) GetOrderData(ctx context.Context, req *OrderEventsRequest) ([]OrderEventResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := orderEventParams(req)
	if err != nil {
		return nil, err
	}
	var resp []OrderEventResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/order_data/", nil, params, &resp)
}

func orderEventParams(req *OrderEventsRequest) (url.Values, error) {
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("market", formatMarketName(req.Pair))
	if req.SinceID != "" {
		params.Set("since_id", req.SinceID)
	}
	if req.UntilID != "" {
		params.Set("until_id", req.UntilID)
	}
	return params, nil
}

// GetOpenOrders returns the account's open orders
// pair is optional and restricts the results to a single market
func (e *Exchange) GetOpenOrders(ctx context.Context, pair currency.Pair) ([]OpenOrderResponse, error) {
	path := "/v2/open_orders/"
	if !pair.IsEmpty() {
		path += formatMarketSymbol(pair) + "/"
	}
	var resp []OpenOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, &resp)
}

// PlaceLimitOrder places a buy or sell limit order
func (e *Exchange) PlaceLimitOrder(ctx context.Context, req *LimitOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	if req.Amount < 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.LimitPrice < 0 {
		return nil, fmt.Errorf("%w: limit price %v", errInvalidPrice, req.LimitPrice)
	}
	var flags int
	for _, set := range []bool{req.DailyOrder, req.ImmediateOrCancel, req.FillOrKill, req.MakerOrCancel, req.GoodTillDate} {
		if set {
			flags++
		}
	}
	if flags > 1 {
		return nil, errOrderFlagsConflict
	}
	if flags > 0 && req.LimitPrice != 0 {
		return nil, errLimitPriceConflict
	}
	if req.GoodTillDate && req.ExpireTime.IsZero() {
		return nil, fmt.Errorf("%w: expire time is required for good till date orders", common.ErrDateUnset)
	}
	orderParams := &derivativesOrderParams{
		ClientOrderID:   req.ClientOrderID,
		Subtype:         req.Subtype,
		MarginMode:      req.MarginMode,
		Leverage:        req.Leverage,
		StopPrice:       req.StopPrice,
		Trigger:         req.Trigger,
		ActivationPrice: req.ActivationPrice,
		TrailingDelta:   req.TrailingDelta,
		ReduceOnly:      req.ReduceOnly,
	}
	if err := validateDerivativesOrderParams(req.Pair, orderParams, limitOrderSubtypes); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	setFloat(params, "price", req.Price)
	setFloat(params, "limit_price", req.LimitPrice)
	setTrue(params, "daily_order", req.DailyOrder)
	setTrue(params, "ioc_order", req.ImmediateOrCancel)
	setTrue(params, "fok_order", req.FillOrKill)
	setTrue(params, "moc_order", req.MakerOrCancel)
	setTrue(params, "gtd_order", req.GoodTillDate)
	if req.GoodTillDate {
		params.Set("expire_time", strconv.FormatInt(req.ExpireTime.UnixMilli(), 10))
	}
	setDerivativesOrderParams(params, orderParams)
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// PlaceInstantOrder places a buy or sell instant order
// Buy amounts are in the counter currency and sell amounts are in the base currency unless AmountInCounter is set
func (e *Exchange) PlaceInstantOrder(ctx context.Context, req *InstantOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	if req.Leverage < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidLeverage, req.Leverage)
	}
	// Only sell orders whose amount is in the base currency accept the reduce only flag
	if req.ReduceOnly && (req.Side.IsLong() || req.AmountInCounter) {
		return nil, errReduceOnlyUnsupported
	}
	if isPerpetualPair(req.Pair) && req.Side.IsLong() {
		if err := validateMarginParams(req.MarginMode, req.Leverage); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	if req.AmountInCounter && req.Side.IsShort() {
		params.Set("amount_in_counter", trueValue)
	}
	setDerivativesOrderParams(params, &derivativesOrderParams{
		ClientOrderID: req.ClientOrderID,
		MarginMode:    req.MarginMode,
		Leverage:      req.Leverage,
		ReduceOnly:    req.ReduceOnly,
	})
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/instant/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// PlaceMarketOrder places a buy or sell market order
func (e *Exchange) PlaceMarketOrder(ctx context.Context, req *MarketOrderRequest) (*OrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Amount < 0 {
		return nil, order.ErrAmountIsInvalid
	}
	orderParams := &derivativesOrderParams{
		ClientOrderID:   req.ClientOrderID,
		Subtype:         req.Subtype,
		MarginMode:      req.MarginMode,
		Leverage:        req.Leverage,
		StopPrice:       req.StopPrice,
		Trigger:         req.Trigger,
		ActivationPrice: req.ActivationPrice,
		TrailingDelta:   req.TrailingDelta,
		ReduceOnly:      req.ReduceOnly,
	}
	if err := validateDerivativesOrderParams(req.Pair, orderParams, marketOrderSubtypes); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	setDerivativesOrderParams(params, orderParams)
	var resp *OrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+side+"/market/"+formatMarketSymbol(req.Pair)+"/", nil, params, &resp)
}

// validateDerivativesOrderParams checks the parameters shared by limit and market orders against the documented limits
// Margin mode and leverage are required for derivatives markets
func validateDerivativesOrderParams(pair currency.Pair, p *derivativesOrderParams, subtypes []string) error {
	if err := validateClientOrderID(p.ClientOrderID); err != nil {
		return err
	}
	if p.Subtype != "" && !slices.Contains(subtypes, p.Subtype) {
		return fmt.Errorf("%w: %q", errInvalidOrderSubtype, p.Subtype)
	}
	if p.Leverage < 0 {
		return fmt.Errorf("%w: %v", errInvalidLeverage, p.Leverage)
	}
	if isPerpetualPair(pair) {
		if err := validateMarginParams(p.MarginMode, p.Leverage); err != nil {
			return err
		}
	}
	if p.StopPrice < 0 {
		return fmt.Errorf("%w: stop price %v", errInvalidPrice, p.StopPrice)
	}
	if p.ActivationPrice < 0 {
		return fmt.Errorf("%w: activation price %v", errInvalidPrice, p.ActivationPrice)
	}
	if p.TrailingDelta > maxTrailingDelta {
		return fmt.Errorf("%w: %d", errInvalidTrailingDelta, p.TrailingDelta)
	}
	if p.ActivationPrice > 0 && p.TrailingDelta == 0 {
		return errTrailingDeltaRequired
	}
	return nil
}

// validateMarginParams checks the margin mode and leverage required by derivatives orders
func validateMarginParams(marginMode string, leverage float64) error {
	if marginMode == "" {
		return errMarginModeRequired
	}
	if leverage <= 0 {
		return errLeverageRequired
	}
	return nil
}

// validateClientOrderID checks a client order ID against the documented maximum length
func validateClientOrderID(clientOrderID string) error {
	if len(clientOrderID) > maxClientOrderIDLength {
		return fmt.Errorf("%w: %d characters", errClientOrderIDTooLong, len(clientOrderID))
	}
	return nil
}

func setDerivativesOrderParams(params url.Values, p *derivativesOrderParams) {
	if p.ClientOrderID != "" {
		params.Set("client_order_id", p.ClientOrderID)
	}
	if p.Subtype != "" {
		params.Set("subtype", p.Subtype)
	}
	if p.MarginMode != "" {
		params.Set("margin_mode", p.MarginMode)
	}
	setFloat(params, "leverage", p.Leverage)
	setFloat(params, "stop_price", p.StopPrice)
	if p.Trigger != "" {
		params.Set("trigger", p.Trigger)
	}
	setFloat(params, "activation_price", p.ActivationPrice)
	if p.TrailingDelta != 0 {
		params.Set("trailing_delta", strconv.FormatUint(p.TrailingDelta, 10))
	}
	setTrue(params, "reduce_only", p.ReduceOnly)
}

// CancelExistingOrder cancels an order
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := validateClientOrderID(req.ClientOrderID); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	var resp *CancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/cancel_order/", nil, params, &resp)
}

// CancelAllExistingOrders cancels all open orders
// pair is optional and restricts the cancellation to a single market
func (e *Exchange) CancelAllExistingOrders(ctx context.Context, pair currency.Pair) (*CancelAllOrdersResponse, error) {
	path := "/v2/cancel_all_orders/"
	if !pair.IsEmpty() {
		path += formatMarketSymbol(pair) + "/"
	}
	var resp *CancelAllOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, nil, &resp)
}

// ReplaceOrder atomically cancels an order and places a new order with the supplied amount and price
func (e *Exchange) ReplaceOrder(ctx context.Context, req *ReplaceOrderRequest) (*ReplaceOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Price <= 0 {
		return nil, order.ErrPriceMustBeSetIfLimitOrder
	}
	for _, id := range []string{req.OriginalClientOrderID, req.ClientOrderID} {
		if err := validateClientOrderID(id); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.OrderID != 0 {
		params.Set("id", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("orig_client_order_id", req.OriginalClientOrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("client_order_id", req.ClientOrderID)
	}
	setFloat(params, "amount", req.Amount)
	setFloat(params, "price", req.Price)
	var resp *ReplaceOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/replace_order/", nil, params, &resp)
}

// GetTradingMarkets returns the markets which can be traded by the account
func (e *Exchange) GetTradingMarkets(ctx context.Context) ([]TradingMarketResponse, error) {
	var resp []TradingMarketResponse
	// The API documents this endpoint as a GET, however it only accepts POST requests
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/my_markets/", nil, nil, &resp)
}

// GetMaxOrderAmount returns the maximum amount and value of an order on a derivatives market
func (e *Exchange) GetMaxOrderAmount(ctx context.Context, req *MaxOrderAmountRequest) (*MaxOrderAmountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.MarginMode == "" {
		return nil, errMarginModeRequired
	}
	if err := validateLeverage(req.Leverage); err != nil {
		return nil, err
	}
	if req.OrderType == "" {
		return nil, errOrderTypeRequired
	}
	side, err := formatOrderSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Price < 0 || req.StopPrice < 0 || req.ActivationPrice < 0 {
		return nil, errInvalidPrice
	}
	if req.TrailingDelta > maxTrailingDelta {
		return nil, fmt.Errorf("%w: %d", errInvalidTrailingDelta, req.TrailingDelta)
	}
	if err := validateCollateral(req.AdditionalCollateral); err != nil {
		return nil, err
	}
	body := &maxOrderAmountBody{
		Market:               formatMarketName(req.Pair),
		MarginMode:           req.MarginMode,
		Leverage:             req.Leverage,
		OrderType:            req.OrderType,
		Side:                 strings.ToUpper(side),
		Price:                req.Price,
		StopPrice:            req.StopPrice,
		ActivationPrice:      req.ActivationPrice,
		TrailingDelta:        req.TrailingDelta,
		AdditionalCollateral: formatCollateral(req.AdditionalCollateral),
	}
	var resp *MaxOrderAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/get_max_order_amount/", nil, body, &resp)
}

// validateLeverage checks a derivatives leverage against the documented range
func validateLeverage(leverage float64) error {
	if leverage == 0 {
		return errLeverageRequired
	}
	if leverage < minLeverage || leverage > maxLeverage {
		return fmt.Errorf("%w: %v", errInvalidLeverage, leverage)
	}
	return nil
}

// validateCollateral checks that collateral amounts are not negative
func validateCollateral(collateral map[currency.Code]float64) error {
	for c, amount := range collateral {
		if amount < 0 {
			return fmt.Errorf("%w: %s %v", errInvalidCollateral, c, amount)
		}
	}
	return nil
}

// GetWithdrawalRequests returns the account's withdrawal requests
func (e *Exchange) GetWithdrawalRequests(ctx context.Context, req *WithdrawalRequestsRequest) ([]WithdrawalRequestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TimeDelta != 0 && (req.TimeDelta < time.Second || req.TimeDelta > maxWithdrawalRequestsTimeDelta) {
		return nil, fmt.Errorf("%w: %s", errInvalidTimeDelta, req.TimeDelta)
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.ID != 0 {
		params.Set("id", strconv.FormatUint(req.ID, 10))
	}
	if req.TimeDelta != 0 {
		params.Set("timedelta", strconv.FormatInt(int64(req.TimeDelta/time.Second), 10))
	}
	// The limit and offset parameters must be sent together
	if req.Limit != 0 || req.Offset != 0 {
		limit := req.Limit
		if limit == 0 {
			limit = maxResultLimit
		}
		params.Set("limit", strconv.FormatUint(limit, 10))
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	var resp []WithdrawalRequestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal-requests/", nil, params, &resp)
}

// OpenBankWithdrawal opens a SEPA or international bank withdrawal request from the main account
func (e *Exchange) OpenBankWithdrawal(ctx context.Context, req *BankWithdrawalRequest) (*BankWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.AccountCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Name == "" || req.IBAN == "" || req.BIC == "" || req.Address == "" || req.PostalCode == "" || req.City == "" || req.Country == "" || req.Type == "" {
		return nil, errBankDetailsRequired
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("account_currency", req.AccountCurrency.Upper().String())
	params.Set("name", req.Name)
	params.Set("iban", req.IBAN)
	params.Set("bic", req.BIC)
	params.Set("address", req.Address)
	params.Set("postal_code", req.PostalCode)
	params.Set("city", req.City)
	params.Set("country", req.Country)
	params.Set("type", req.Type)
	for k, v := range map[string]string{
		"comment":                     req.Comment,
		"bank_name":                   req.BankName,
		"bank_address":                req.BankAddress,
		"bank_postal_code":            req.BankPostalCode,
		"bank_city":                   req.BankCity,
		"bank_country":                req.BankCountry,
		"intermed_routing_num_or_bic": req.IntermediaryRoutingNumberOrBIC,
	} {
		if v != "" {
			params.Set(k, v)
		}
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	var resp *BankWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/open/", nil, params, &resp)
}

// CancelWithdrawal cancels a bank or cryptocurrency withdrawal request from the main account
func (e *Exchange) CancelWithdrawal(ctx context.Context, withdrawalID uint64) (*CancelWithdrawalResponse, error) {
	if withdrawalID == 0 {
		return nil, errWithdrawalIDRequired
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(withdrawalID, 10))
	var resp *CancelWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/cancel/", nil, params, &resp)
}

// GetFiatWithdrawalStatus returns the status of a bank withdrawal request from the main account
func (e *Exchange) GetFiatWithdrawalStatus(ctx context.Context, withdrawalID uint64) (*WithdrawalStatusResponse, error) {
	if withdrawalID == 0 {
		return nil, errWithdrawalIDRequired
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(withdrawalID, 10))
	var resp *WithdrawalStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/withdrawal/status/", nil, params, &resp)
}

// CryptoWithdrawal requests a cryptocurrency withdrawal
func (e *Exchange) CryptoWithdrawal(ctx context.Context, req *CryptoWithdrawalRequest) (*CryptoWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	travelRule := &travelRuleParams{
		OriginatorInfo:        req.OriginatorInfo,
		BeneficiaryInfo:       req.BeneficiaryInfo,
		BeneficiaryThirdParty: req.BeneficiaryThirdParty,
		BeneficiaryID:         req.BeneficiaryID,
		VASPUUID:              req.VASPUUID,
	}
	if err := validateTravelRuleParams(travelRule); err != nil {
		return nil, err
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("address", req.Address)
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if req.MemoID != "" {
		params.Set("memo_id", req.MemoID)
	}
	if req.DestinationTag != "" {
		params.Set("destination_tag", req.DestinationTag)
	}
	if req.TransferID != 0 {
		params.Set("transfer_id", strconv.FormatUint(req.TransferID, 10))
	}
	if err := setTravelRuleParams(params, travelRule); err != nil {
		return nil, err
	}
	var resp *CryptoWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+req.Currency.Lower().String()+"_withdrawal/", nil, params, &resp)
}

// RippleIOUWithdrawal requests a USD, BTC, EUR or ETH IOU withdrawal on the XRP Ledger
func (e *Exchange) RippleIOUWithdrawal(ctx context.Context, req *RippleIOUWithdrawalRequest) (*CryptoWithdrawalResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	travelRule := &travelRuleParams{
		OriginatorInfo:        req.OriginatorInfo,
		BeneficiaryInfo:       req.BeneficiaryInfo,
		BeneficiaryThirdParty: req.BeneficiaryThirdParty,
		BeneficiaryID:         req.BeneficiaryID,
		VASPUUID:              req.VASPUUID,
	}
	if err := validateTravelRuleParams(travelRule); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currency", req.Currency.Upper().String())
	setFloat(params, "amount", req.Amount)
	params.Set("address", req.Address)
	if err := setTravelRuleParams(params, travelRule); err != nil {
		return nil, err
	}
	var resp *CryptoWithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/ripple_withdrawal/", nil, params, &resp)
}

// validateTravelRuleParams checks the Travel Rule details of a cryptocurrency withdrawal
// The originator details describe the sender and must identify them, and a third party beneficiary must be identified by
// a contact ID or details
func validateTravelRuleParams(p *travelRuleParams) error {
	if p.OriginatorInfo != nil {
		if err := validateIdentifiedCustomer(p.OriginatorInfo); err != nil {
			return fmt.Errorf("originator: %w", err)
		}
	}
	if p.BeneficiaryInfo != nil {
		if err := validateContact(p.BeneficiaryInfo.RetailInfo, p.BeneficiaryInfo.CorporateInfo); err != nil {
			return fmt.Errorf("beneficiary: %w", err)
		}
	}
	if p.BeneficiaryThirdParty != nil && *p.BeneficiaryThirdParty && p.BeneficiaryID == "" && p.BeneficiaryInfo == nil {
		return errBeneficiaryRequired
	}
	return nil
}

// validateContact checks that Travel Rule contact details hold the names the API requires
func validateContact(retail *RetailInfo, corporate *CorporateInfo) error {
	if retail == nil && corporate == nil {
		return errContactInfoRequired
	}
	if retail != nil && (retail.FirstName == "" || retail.LastName == "") {
		return errContactNameRequired
	}
	if corporate != nil && corporate.CompanyName == "" {
		return errCompanyNameRequired
	}
	return nil
}

// validateIdentifiedCustomer checks Travel Rule details which must also identify the customer: a retail customer by
// date and place of birth or by identity document, and a corporate customer by LEI or company ID
func validateIdentifiedCustomer(info *CustomerInfo) error {
	if err := validateContact(info.RetailInfo, info.CorporateInfo); err != nil {
		return err
	}
	if r := info.RetailInfo; r != nil && (r.DateOfBirth.IsZero() || r.PlaceOfBirth == "") && (r.IDType == "" || r.IDNumber == "") {
		return fmt.Errorf("%w: date and place of birth or identity document type and number", errIdentificationRequired)
	}
	if c := info.CorporateInfo; c != nil && c.LEI == "" && c.CompanyID == "" {
		return fmt.Errorf("%w: LEI or company ID", errIdentificationRequired)
	}
	return nil
}

func setTravelRuleParams(params url.Values, p *travelRuleParams) error {
	// Contact details are submitted as JSON encoded form values
	for k, v := range map[string]*CustomerInfo{"originator_info": p.OriginatorInfo, "beneficiary_info": p.BeneficiaryInfo} {
		if v == nil {
			continue
		}
		info, err := json.Marshal(v)
		if err != nil {
			return err
		}
		params.Set(k, string(info))
	}
	if p.BeneficiaryThirdParty != nil {
		params.Set("beneficiary_thirdparty", formatBool(*p.BeneficiaryThirdParty))
	}
	if p.BeneficiaryID != "" {
		params.Set("beneficiary_id", p.BeneficiaryID)
	}
	if p.VASPUUID != "" {
		params.Set("vasp_uuid", p.VASPUUID)
	}
	return nil
}

// GetCryptoDepositAddress returns a deposit address for a cryptocurrency
// network is optional and selects the currency network of the address
func (e *Exchange) GetCryptoDepositAddress(ctx context.Context, c currency.Code, network string) (*DepositAddressResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if network != "" {
		params.Set("network", network)
	}
	var resp *DepositAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/"+c.Lower().String()+"_address/", nil, params, &resp)
}

// GetUnconfirmedBitcoinDeposits returns the account's unconfirmed bitcoin deposits
func (e *Exchange) GetUnconfirmedBitcoinDeposits(ctx context.Context) ([]UnconfirmedDepositResponse, error) {
	var resp unconfirmedDeposits
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/btc_unconfirmed/", nil, nil, &resp)
}

// GetRippleIOUDepositAddress returns the account's Ripple IOU deposit address
func (e *Exchange) GetRippleIOUDepositAddress(ctx context.Context) (*RippleIOUDepositAddressResponse, error) {
	var resp *RippleIOUDepositAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/ripple_address/", nil, nil, &resp)
}

// TransferToMainAccount transfers a balance from a sub account to the main account
// SubAccount must be supplied when called with main account credentials
func (e *Exchange) TransferToMainAccount(ctx context.Context, req *TransferRequest) error {
	params, err := transferParams(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/transfer-to-main/", nil, params, nil)
}

// TransferFromMainAccount transfers a balance from the main account to a sub account
func (e *Exchange) TransferFromMainAccount(ctx context.Context, req *TransferRequest) error {
	params, err := transferParams(req)
	if err != nil {
		return err
	}
	if req.SubAccount == 0 {
		return errSubAccountRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/transfer-from-main/", nil, params, nil)
}

func transferParams(req *TransferRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("currency", req.Currency.Upper().String())
	if req.SubAccount != 0 {
		params.Set("subAccount", strconv.FormatUint(req.SubAccount, 10))
	}
	return params, nil
}

// CreateInstantConvertAddress creates an address which automatically sells deposited bitcoin for the liquidation currency
// addressFormat is optional and may be P2SHP2WSH or BECH32
func (e *Exchange) CreateInstantConvertAddress(ctx context.Context, liquidationCurrency currency.Code, addressFormat string) (*InstantConvertAddressResponse, error) {
	if liquidationCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("liquidation_currency", liquidationCurrency.Upper().String())
	if addressFormat != "" {
		params.Set("address_format", addressFormat)
	}
	var resp *InstantConvertAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/instant_convert_address/new/", nil, params, &resp)
}

// GetInstantConvertAddressInfo returns the transactions of the account's instant convert addresses
// address is optional and restricts the results to a single address
func (e *Exchange) GetInstantConvertAddressInfo(ctx context.Context, address string) ([]InstantConvertAddressInfoResponse, error) {
	params := url.Values{}
	if address != "" {
		params.Set("address", address)
	}
	var resp []InstantConvertAddressInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/instant_convert_address/info/", nil, params, &resp)
}

// GetWebsocketToken returns a token and user ID for subscribing to private websocket channels
// The token is only valid for a short period and must be used immediately
func (e *Exchange) GetWebsocketToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	var resp *WebsocketTokenResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/websockets_token/", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("error fetching websocket token: %w", err)
	}
	return resp, nil
}

// GetUserTransactions returns the account's transactions
// Since and Until may only be up to 30 days ago, while older transactions can be paged through with SinceID
func (e *Exchange) GetUserTransactions(ctx context.Context, req *UserTransactionsRequest) ([]UserTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Sort != "" {
		params.Set("sort", req.Sort)
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	if req.SinceID != 0 {
		params.Set("since_id", strconv.FormatUint(req.SinceID, 10))
	}
	path := "/v2/user_transactions/"
	if !req.Pair.IsEmpty() {
		path += formatMarketSymbol(req.Pair) + "/"
	}
	var resp []UserTransactionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, nil, params, &resp)
}

// GetCryptoTransactions returns the main account's cryptocurrency deposits and withdrawals
// Since and Until may only be up to 30 days ago
func (e *Exchange) GetCryptoTransactions(ctx context.Context, req *CryptoTransactionsRequest) (*CryptoTransactionsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	setTrue(params, "include_ious", req.IncludeIOUs)
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	var resp *CryptoTransactionsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/", nil, params, &resp)
}

// GetCryptoDeposits returns the account's cryptocurrency deposits with their review status
func (e *Exchange) GetCryptoDeposits(ctx context.Context, req *CryptoDepositsRequest) ([]CryptoDepositResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	if !req.Since.IsZero() && !req.Until.IsZero() && req.Since.After(req.Until) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.Since.IsZero() {
		params.Set("since_timestamp", strconv.FormatInt(req.Since.Unix(), 10))
	}
	if !req.Until.IsZero() {
		params.Set("until_timestamp", strconv.FormatInt(req.Until.Unix(), 10))
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	var resp []CryptoDepositResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/crypto-transactions/deposits/", params, nil, &resp)
}

// UpdateCryptoDepositOriginator submits Travel Rule originator details for a pending cryptocurrency deposit
func (e *Exchange) UpdateCryptoDepositOriginator(ctx context.Context, depositID uint64, req *DepositOriginatorRequest) error {
	if depositID == 0 {
		return errDepositIDRequired
	}
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.OriginatorThirdParty != nil && *req.OriginatorThirdParty && req.OriginatorID == "" && req.OriginatorInfo == nil {
		return errOriginatorRequired
	}
	if req.OriginatorInfo != nil {
		if err := validateContact(req.OriginatorInfo.RetailInfo, req.OriginatorInfo.CorporateInfo); err != nil {
			return fmt.Errorf("originator: %w", err)
		}
	}
	// The beneficiary details describe the account holder receiving the deposit, so they must identify them
	if req.BeneficiaryInfo != nil {
		if err := validateIdentifiedCustomer(req.BeneficiaryInfo); err != nil {
			return fmt.Errorf("beneficiary: %w", err)
		}
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/deposits/"+strconv.FormatUint(depositID, 10)+"/", nil, req, nil)
}

// RejectCryptoDeposit rejects a pending cryptocurrency deposit
// Rejected funds are not automatically returned to the originator address
func (e *Exchange) RejectCryptoDeposit(ctx context.Context, depositID uint64) error {
	if depositID == 0 {
		return errDepositIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/crypto-transactions/deposits/"+strconv.FormatUint(depositID, 10)+"/reject/", nil, nil, nil)
}

// GetContact returns a Travel Rule contact
func (e *Exchange) GetContact(ctx context.Context, contactID string) (*ContactResponse, error) {
	if contactID == "" {
		return nil, errContactIDRequired
	}
	var resp *ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/contacts/"+url.PathEscape(contactID)+"/", nil, nil, &resp)
}

// GetContacts returns the account's Travel Rule contacts
func (e *Exchange) GetContacts(ctx context.Context, req *PaginationRequest) ([]ContactResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := paginationParams(req)
	var resp []ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/contacts/", params, nil, &resp)
}

// CreateContact creates a Travel Rule contact for use as a withdrawal beneficiary or deposit originator
func (e *Exchange) CreateContact(ctx context.Context, req *ContactRequest) (*ContactResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validateContact(req.RetailInfo, req.CorporateInfo); err != nil {
		return nil, err
	}
	var resp *ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/contacts/", nil, req, &resp)
}

// SubmitCounterpartyInfo registers Travel Rule counterparty details for a cryptocurrency address
func (e *Exchange) SubmitCounterpartyInfo(ctx context.Context, req *CounterpartyAddressRequest) (*CounterpartyAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.ContactThirdParty && req.ContactUUID == "" {
		return nil, errContactUUIDRequired
	}
	var resp *CounterpartyAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/addresses/", nil, req, &resp)
}

// GetSatoshiTests returns the account's Satoshi tests for verifying ownership of external addresses
func (e *Exchange) GetSatoshiTests(ctx context.Context, req *SatoshiTestsRequest) ([]SatoshiTestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	for k, v := range map[string]string{"network": req.Network, "address": req.Address, "status": req.Status} {
		if v != "" {
			params.Set(k, v)
		}
	}
	var resp []SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/satoshi_test/", params, nil, &resp)
}

// CreateSatoshiTest starts a Satoshi test to verify ownership of an external address
func (e *Exchange) CreateSatoshiTest(ctx context.Context, req *SatoshiTestRequest) (*SatoshiTestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := &satoshiTestBody{
		Address:  req.Address,
		Network:  req.Network,
		Currency: req.Currency.Upper().String(),
	}
	var resp *SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/satoshi_test/", nil, body, &resp)
}

// GetSatoshiTest returns a Satoshi test
func (e *Exchange) GetSatoshiTest(ctx context.Context, satoshiTestID string) (*SatoshiTestResponse, error) {
	if satoshiTestID == "" {
		return nil, errSatoshiTestIDRequired
	}
	var resp *SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/satoshi_test/"+url.PathEscape(satoshiTestID)+"/", nil, nil, &resp)
}

// GetXpubRegistrations returns the account's extended public key registrations
func (e *Exchange) GetXpubRegistrations(ctx context.Context, req *XpubRegistrationsRequest) ([]XpubRegistrationResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/utxo/xpub_registrations/", params, nil, &resp)
}

// CreateXpubRegistration registers an extended public key for UTXO address derivation
// The registration remains pending until ownership is proven with a Satoshi test
func (e *Exchange) CreateXpubRegistration(ctx context.Context, req *XpubRegistrationRequest) (*XpubRegistrationResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.ExtendedPublicKey == "" {
		return nil, errPublicKeyRequired
	}
	var resp *XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/utxo/xpub_registrations/", nil, req, &resp)
}

// RevokeXpubRegistration revokes an extended public key registration
func (e *Exchange) RevokeXpubRegistration(ctx context.Context, registrationID string) (*XpubRevocationResponse, error) {
	if registrationID == "" {
		return nil, errRegistrationIDRequired
	}
	var resp *XpubRevocationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/utxo/xpub_registrations/"+url.PathEscape(registrationID)+"/revoke/", nil, nil, &resp)
}

// GetXpubRegistration returns an extended public key registration
func (e *Exchange) GetXpubRegistration(ctx context.Context, registrationID string) (*XpubRegistrationResponse, error) {
	if registrationID == "" {
		return nil, errRegistrationIDRequired
	}
	var resp *XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/utxo/xpub_registrations/"+url.PathEscape(registrationID)+"/", nil, nil, &resp)
}

// GetAddressVerification returns whether ownership of a cryptocurrency address has been verified for the account
func (e *Exchange) GetAddressVerification(ctx context.Context, network, address string) (*AddressVerificationResponse, error) {
	if network == "" {
		return nil, errNetworkRequired
	}
	if address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	params := url.Values{}
	params.Set("network", network)
	params.Set("address", address)
	var resp *AddressVerificationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/address_verification/", params, nil, &resp)
}

// EarnSubscribe subscribes an amount to a staking or lending product
func (e *Exchange) EarnSubscribe(ctx context.Context, req *EarnSubscriptionRequest) error {
	body, err := earnSubscription(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/subscribe/", nil, body, nil)
}

// EarnUnsubscribe unsubscribes an amount from a staking or lending product
func (e *Exchange) EarnUnsubscribe(ctx context.Context, req *EarnSubscriptionRequest) error {
	body, err := earnSubscription(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/unsubscribe/", nil, body, nil)
}

func earnSubscription(req *EarnSubscriptionRequest) (*earnSubscriptionBody, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.EarnType == "" {
		return nil, errEarnTypeRequired
	}
	if req.EarnTerm == "" {
		return nil, errEarnTermRequired
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	return &earnSubscriptionBody{
		Currency: req.Currency.Upper().String(),
		EarnType: req.EarnType,
		EarnTerm: req.EarnTerm,
		Amount:   req.Amount,
	}, nil
}

// SetEarnSubscriptionSetting opts in to or out of an Earn product; only staking is currently supported
func (e *Exchange) SetEarnSubscriptionSetting(ctx context.Context, req *EarnSubscriptionSettingRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Setting == "" {
		return errEarnSettingRequired
	}
	if req.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if req.EarnType == "" {
		return errEarnTypeRequired
	}
	body := &earnSubscriptionSettingBody{
		Setting:  req.Setting,
		Currency: req.Currency.Upper().String(),
		EarnType: req.EarnType,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/earn/subscriptions/setting/", nil, body, nil)
}

// GetEarnTransactions returns the account's Earn transaction history
func (e *Exchange) GetEarnTransactions(ctx context.Context, req *EarnTransactionsRequest) ([]EarnTransactionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if !req.QuoteCurrency.IsEmpty() {
		params.Set("quote_currency", req.QuoteCurrency.Upper().String())
	}
	var resp []EarnTransactionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/earn/transactions/", params, nil, &resp)
}

// GetEarnSubscriptions returns the account's Earn subscriptions
func (e *Exchange) GetEarnSubscriptions(ctx context.Context) ([]EarnSubscriptionResponse, error) {
	var resp []EarnSubscriptionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/earn/subscriptions/", nil, nil, &resp)
}

// RevokeAllAPIKeys revokes every API key across all of the user's accounts, including the key making the request
func (e *Exchange) RevokeAllAPIKeys(ctx context.Context) (*RevokeAPIKeysResponse, error) {
	var resp *RevokeAPIKeysResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/revoke_all_api_keys/", nil, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated HTTP GET request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, params url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var interim json.RawMessage
	var responseHeaders http.Header
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		responseHeaders = http.Header{}
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   common.EncodeURLValues(endpoint+path, params),
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
			HeaderResponse:         &responseHeaders,
		}, nil
	}, request.UnauthenticatedRequest)
	return processResponse(interim, err, len(responseHeaders) > 0, result, false)
}

// SendAuthenticatedHTTPRequest sends an HTTP request signed with Bitstamp's v2 authentication scheme
// params are sent in the query string. body is form encoded when it is url.Values and JSON encoded otherwise; a nil or
// empty body is sent without a Content-Type header, as the API rejects requests with an empty body and Content-Type
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, body, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	target, err := url.Parse(common.EncodeURLValues(endpoint+path, params))
	if err != nil {
		return err
	}

	var payload []byte
	var contentType string
	switch b := body.(type) {
	case nil:
	case url.Values:
		if len(b) > 0 {
			payload = []byte(b.Encode())
			contentType = formContentType
		}
	default:
		if payload, err = json.Marshal(b); err != nil {
			return err
		}
		contentType = jsonContentType
	}

	// The query component of the signature includes its leading question mark, as the URL is built by concatenating it
	var query string
	if target.RawQuery != "" {
		query = "?" + target.RawQuery
	}

	var interim json.RawMessage
	var responseHeaders http.Header
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		// Headers are only filled when a response arrives, so each attempt starts with none
		responseHeaders = http.Header{}
		nonce := uuid.NewV4().String()
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		authHeader := "BITSTAMP " + creds.Key
		message := authHeader + method + target.Host + target.Path + query + contentType + nonce + timestamp + "v2" + string(payload)
		signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"X-Auth":           authHeader,
			"X-Auth-Signature": hex.EncodeToString(signature),
			"X-Auth-Nonce":     nonce,
			"X-Auth-Timestamp": timestamp,
			"X-Auth-Version":   "v2",
		}
		if contentType != "" {
			headers["Content-Type"] = contentType
		}
		if creds.SubAccount != "" {
			headers["X-Auth-Subaccount-Id"] = creds.SubAccount
		}
		var reader io.Reader
		if len(payload) > 0 {
			reader = bytes.NewReader(payload)
		}
		return &request.Item{
			Method:                 method,
			Path:                   target.String(),
			Headers:                headers,
			Body:                   reader,
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
			HeaderResponse:         &responseHeaders,
		}, nil
	}, request.AuthenticatedRequest)
	return processResponse(interim, err, len(responseHeaders) > 0, result, true)
}

// processResponse returns any error reported in a response body before decoding the body into result
// responded reports whether a response was received, which distinguishes an empty successful body that cannot be
// decoded from a request which failed
func processResponse(data json.RawMessage, requestErr error, responded bool, result any, authenticated bool) error {
	if apiErr := checkResponseError(data, requestErr != nil); apiErr != nil {
		if authenticated {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, apiErr)
		}
		return apiErr
	}
	data = bytes.TrimSpace(data)
	empty := len(data) == 0 || bytes.Equal(data, []byte("null"))
	if requestErr != nil {
		// A successful response without a body fails to decode, which only matters when a result is expected
		if result == nil && empty && responded && !errors.Is(requestErr, request.ErrBadStatus) {
			return nil
		}
		return requestErr
	}
	if result == nil {
		return nil
	}
	if empty {
		// A missing body would otherwise leave a pointer result nil for callers to dereference
		if v := reflect.ValueOf(result); v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Pointer {
			return errNoResponse
		}
		return nil
	}
	return json.Unmarshal(data, result)
}

// checkResponseError returns an error describing an error response body
// Successful responses may contain status, code and message fields, so they are only treated as errors when they match
// a documented error shape, whereas any recognised field is reported for responses with an unsuccessful HTTP status
func checkResponseError(data json.RawMessage, unsuccessfulStatus bool) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil
	}
	var resp errorResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil //nolint:nilerr // Malformed bodies are reported by the caller's decode or by the request error
	}
	code := formatErrorDetail(resp.Code)
	message := formatErrorDetail(resp.Message)
	errorDetail := formatErrorDetail(resp.Error)
	if !unsuccessfulStatus && formatErrorDetail(resp.Status) != "error" && errorDetail == "" && (code == "" || message == "") && formatErrorDetail(resp.Errors) == "" {
		return nil
	}

	var codes, details []string
	for _, c := range []string{code, formatErrorDetail(resp.ResponseCode)} {
		if c != "" {
			codes = append(codes, c)
		}
	}
	for _, d := range []string{formatErrorDetail(resp.Reason), errorDetail, message, formatErrorDetail(resp.ResponseExplanation), formatErrorDetail(resp.Errors)} {
		if d != "" {
			details = append(details, d)
		}
	}
	if field := formatErrorDetail(resp.Field); field != "" {
		details = append(details, "field: "+field)
	}
	switch {
	case len(codes) == 0 && len(details) == 0:
		if unsuccessfulStatus {
			return nil
		}
		return fmt.Errorf("%w: %s", errAPIResponse, data)
	case len(codes) == 0:
		return fmt.Errorf("%w: %s", errAPIResponse, strings.Join(details, "; "))
	case len(details) == 0:
		return fmt.Errorf("%w: %s", errAPIResponse, strings.Join(codes, " "))
	default:
		return fmt.Errorf("%w: %s %s", errAPIResponse, strings.Join(codes, " "), strings.Join(details, "; "))
	}
}

// formatErrorDetail flattens an error detail which may be a string, a number, a list or an object of field errors
func formatErrorDetail(data json.RawMessage) string {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return ""
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return s
	}
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err == nil {
		details := make([]string, 0, len(list))
		for _, v := range list {
			if d := formatErrorDetail(v); d != "" {
				details = append(details, d)
			}
		}
		return strings.Join(details, ", ")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err == nil {
		details := make([]string, 0, len(fields))
		for _, k := range slices.Sorted(maps.Keys(fields)) {
			d := formatErrorDetail(fields[k])
			if d == "" {
				continue
			}
			if k == "__all__" {
				details = append(details, d)
				continue
			}
			details = append(details, k+": "+d)
		}
		return strings.Join(details, "; ")
	}
	return string(data)
}

// isPerpetualPair reports whether a pair is a perpetual contract market, e.g. BTC/USD-PERP
func isPerpetualPair(pair currency.Pair) bool {
	return strings.HasSuffix(formatMarketName(pair), perpetualMarketSuffix)
}

// paginationParams returns the query parameters of a page request
func paginationParams(req *PaginationRequest) url.Values {
	params := url.Values{}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PerPage != 0 {
		params.Set("per_page", strconv.FormatUint(req.PerPage, 10))
	}
	return params
}

// validatePaging checks a limit and offset against the documented maximums
func validatePaging(limit, offset uint64) error {
	if limit > maxResultLimit {
		return fmt.Errorf("%w: %d must not exceed %d", errInvalidLimit, limit, maxResultLimit)
	}
	if offset > maxResultOffset {
		return fmt.Errorf("%w: %d must not exceed %d", errInvalidOffset, offset, maxResultOffset)
	}
	return nil
}

// formatMarketSymbol returns the market symbol used in endpoint paths, e.g. btcusd or btcusd-perp
func formatMarketSymbol(pair currency.Pair) string {
	return currency.EMPTYFORMAT.Format(pair)
}

// formatMarketName returns the market name used in request parameters, e.g. BTC/USD or BTC/USD-PERP
func formatMarketName(pair currency.Pair) string {
	return pair.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.ForwardSlashDelimiter}).String()
}

// formatOrderSide returns the side used in order endpoint paths
func formatOrderSide(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return "buy", nil
	case side.IsShort():
		return "sell", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// formatCollateral formats collateral amounts keyed by currency for JSON request bodies
func formatCollateral(collateral map[currency.Code]float64) map[string]string {
	if len(collateral) == 0 {
		return nil
	}
	formatted := make(map[string]string, len(collateral))
	for c, amount := range collateral {
		formatted[c.Upper().String()] = strconv.FormatFloat(amount, 'f', -1, 64)
	}
	return formatted
}

func setFloat(params url.Values, key string, value float64) {
	if value != 0 {
		params.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
	}
}

func setTrue(params url.Values, key string, value bool) {
	if value {
		params.Set(key, trueValue)
	}
}

// formatBool returns a boolean in the form the API expects in form bodies
func formatBool(value bool) string {
	if value {
		return trueValue
	}
	return falseValue
}
