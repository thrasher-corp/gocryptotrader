package bitget

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange is the overarching type across this package
type Exchange struct {
	exchange.Base
}

const (
	bitgetAPIURL = "https://api.bitget.com/api/v2/"

	// Public endpoints
	bitgetPublic                   = "public/"
	bitgetAnnouncements            = "annoucements" // Misspelling of announcements
	bitgetTime                     = "time"
	bitgetMarket                   = "market/"
	bitgetWhaleNetFlow             = "whale-net-flow"
	bitgetTakerBuySell             = "taker-buy-sell"
	bitgetPositionLongShort        = "position-long-short"
	bitgetLongShortRatio           = "long-short-ratio"
	bitgetLoanGrowth               = "loan-growth"
	bitgetIsolatedBorrowRate       = "isolated-borrow-rate"
	bitgetLongShort                = "long-short"
	bitgetFundFlow                 = "fund-flow"
	bitgetSupportSymbols           = "support-symbols"
	bitgetFundNetFlow              = "fund-net-flow"
	bitgetAccountLongShort         = "account-long-short"
	bitgetCoins                    = "coins"
	bitgetSymbols                  = "symbols"
	bitgetVIPFeeRate               = "vip-fee-rate"
	bitgetUnionInterestRateHistory = "union-interest-rate-history"
	bitgetExchangeRate             = "exchange-rate"
	bitgetDiscountRate             = "discount-rate"
	bitgetTickers                  = "tickers"
	bitgetMergeDepth               = "merge-depth"
	bitgetOrderbook                = "orderbook"
	bitgetCandles                  = "candles"
	bitgetHistoryCandles           = "history-candles"
	bitgetFillsHistory             = "fills-history"
	bitgetTicker                   = "ticker"
	bitgetHistoryIndexCandles      = "history-index-candles"
	bitgetHistoryMarkCandles       = "history-mark-candles"
	bitgetOpenInterest             = "open-interest"
	bitgetFundingTime              = "funding-time"
	bitgetSymbolPrice              = "symbol-price"
	bitgetHistoryFundRate          = "history-fund-rate"
	bitgetCurrentFundRate          = "current-fund-rate"
	bitgetContracts                = "contracts"
	bitgetQueryPositionLever       = "query-position-lever"
	bitgetCoinInfos                = "coinInfos"
	bitgetHourInterest             = "hour-interest"

	// Mixed endpoints
	bitgetSpot   = "spot/"
	bitgetMix    = "mix/"
	bitgetFills  = "fills"
	bitgetMargin = "margin/"
	bitgetEarn   = "earn/"
	bitgetLoan   = "loan"

	// Authenticated endpoints
	bitgetCommon                   = "common/"
	bitgetTradeRate                = "trade-rate"
	bitgetTax                      = "tax/"
	bitgetSpotRecord               = "spot-record"
	bitgetFutureRecord             = "future-record"
	bitgetMarginRecord             = "margin-record"
	bitgetP2PRecord                = "p2p-record"
	bitgetP2P                      = "p2p/"
	bitgetMerchantList             = "merchantList"
	bitgetMerchantInfo             = "merchantInfo"
	bitgetOrderList                = "orderList"
	bitgetAdvList                  = "advList"
	bitgetUser                     = "user/"
	bitgetCreate                   = "create-"
	bitgetVirtualSubaccount        = "virtual-subaccount"
	bitgetModify                   = "modify-"
	bitgetBatchCreateSubAccAPI     = "batch-create-subaccount-and-apikey"
	bitgetList                     = "list"
	bitgetAPIKey                   = "apikey"
	bitgetFundingAssets            = "/funding-assets"
	bitgetBotAssets                = "/bot-assets"
	bitgetAllAccountBalance        = "/all-account-balance"
	bitgetConvert                  = "convert/"
	bitgetCurrencies               = "currencies"
	bitgetQuotedPrice              = "quoted-price"
	bitgetTrade                    = "trade"
	bitgetConvertRecord            = "convert-record"
	bitgetBGBConvert               = "bgb-convert"
	bitgetConvertCoinList          = "bgb-convert-coin-list"
	bitgetBGBConvertRecords        = "bgb-convert-records"
	bitgetPlaceOrder               = "/place-order"
	bitgetCancelReplaceOrder       = "/cancel-replace-order"
	bitgetBatchCancelReplaceOrder  = "/batch-cancel-replace-order"
	bitgetCancelOrder              = "/cancel-order"
	bitgetBatchOrders              = "/batch-orders"
	bitgetBatchCancel              = "/batch-cancel-order"
	bitgetCancelSymbolOrder        = "/cancel-symbol-order"
	bitgetOrderInfo                = "/orderInfo"
	bitgetUnfilledOrders           = "/unfilled-orders"
	bitgetHistoryOrders            = "/history-orders"
	bitgetPlacePlanOrder           = "/place-plan-order"
	bitgetModifyPlanOrder          = "/modify-plan-order"
	bitgetCancelPlanOrder          = "/cancel-plan-order"
	bitgetCurrentPlanOrder         = "/current-plan-order"
	bitgetPlanSubOrder             = "/plan-sub-order"
	bitgetPlanOrderHistory         = "/history-plan-order"
	bitgetBatchCancelPlanOrder     = "/batch-cancel-plan-order"
	bitgetAccount                  = "account"
	bitgetInfo                     = "/info"
	bitgetAssets                   = "/assets"
	bitgetSubaccountAssets         = "/subaccount-assets"
	bitgetWallet                   = "wallet/"
	bitgetModifyDepositAccount     = "modify-deposit-account"
	bitgetBills                    = "/bills"
	bitgetTransfer                 = "transfer"
	bitgetTransferCoinInfo         = "transfer-coin-info"
	bitgetSubaccountTransfer       = "subaccount-transfer"
	bitgetWithdrawal               = "withdrawal"
	bitgetSubaccountTransferRecord = "/sub-main-trans-record"
	bitgetTransferRecord           = "/transferRecords"
	bitgetSwitchDeduct             = "/switch-deduct"
	bitgetDepositAddress           = "deposit-address"
	bitgetSubaccountDepositAddress = "subaccount-deposit-address"
	bitgetDeductInfo               = "/deduct-info"
	bitgetCancelWithdrawal         = "cancel-withdrawal"
	bitgetSubaccountDepositRecord  = "subaccount-deposit-records"
	bitgetWithdrawalRecord         = "withdrawal-records"
	bitgetDepositRecord            = "deposit-records"
	bitgetAccounts                 = "/accounts"
	bitgetSubaccountAssets2        = "/sub-account-assets"
	bitgetOpenCount                = "/open-count"
	bitgetSetLeverage              = "/set-leverage"
	bitgetSetAutoMargin            = "/set-auto-margin"
	bitgetSetMargin                = "/set-margin"
	bitgetSetAssetMode             = "/set-asset-mode"
	bitgetSetMarginMode            = "/set-margin-mode"
	bitgetSetPositionMode          = "/set-position-mode"
	bitgetBill                     = "/bill"
	bitgetPosition                 = "position/"
	bitgetSinglePosition           = "single-position"
	bitgetAllPositions             = "all-position" // Misspelling of all-positions
	bitgetHistoryPosition          = "history-position"
	bitgetOrder                    = "order"
	bitgetClickBackhand            = "/click-backhand"
	bitgetBatchPlaceOrder          = "/batch-place-order"
	bitgetModifyOrder              = "/modify-order"
	bitgetBatchCancelOrders        = "/batch-cancel-orders"
	bitgetClosePositions           = "/close-positions"
	bitgetDetail                   = "/detail"
	bitgetOrdersPending            = "/orders-pending"
	bitgetFillHistory              = "/fill-history"
	bitgetOrdersHistory            = "/orders-history"
	bitgetCancelAllOrders          = "/cancel-all-orders"
	bitgetPlaceTPSLOrder           = "/place-tpsl-order"
	bitgetPlacePOSTPSL             = "/place-pos-tpsl"
	bitgetModifyTPSLOrder          = "/modify-tpsl-order"
	bitgetOrdersPlanPending        = "/orders-plan-pending"
	bitgetOrdersPlanHistory        = "/orders-plan-history"
	bitgetCrossed                  = "crossed"
	bitgetBorrowHistory            = "/borrow-history"
	bitgetRepayHistory             = "/repay-history"
	bitgetInterestHistory          = "/interest-history"
	bitgetLiquidationHistory       = "/liquidation-history"
	bitgetFinancialRecords         = "/financial-records"
	bitgetBorrow                   = "/borrow"
	bitgetRepay                    = "/repay"
	bitgetRiskRate                 = "/risk-rate"
	bitgetMaxBorrowableAmount      = "/max-borrowable-amount"
	bitgetMaxTransferOutAmount     = "/max-transfer-out-amount"
	bitgetInterestRateAndLimit     = "/interest-rate-and-limit"
	bitgetTierData                 = "/tier-data"
	bitgetFlashRepay               = "/flash-repay"
	bitgetQueryFlashRepayStatus    = "/query-flash-repay-status"
	bitgetBatchCancelOrder         = "/batch-cancel-order"
	bitgetOpenOrders               = "/open-orders"
	bitgetLiquidationOrder         = "/liquidation-order"
	bitgetIsolated                 = "isolated"
	bitgetSavings                  = "savings"
	bitgetProduct                  = "/product"
	bitgetRecords                  = "/records"
	bitgetSubscribeInfo            = "/subscribe-info"
	bitgetSubscribe                = "/subscribe"
	bitgetSubscribeResult          = "/subscribe-result"
	bitgetRedeem                   = "/redeem"
	bitgetRedeemResult             = "/redeem-result"
	bitgetSharkFin                 = "sharkfin"
	bitgetOngoingOrders            = "/ongoing-orders"
	bitgetRevisePledge             = "/revise-pledge"
	bitgetReviseHistory            = "/revise-history"
	bitgetDebts                    = "/debts"
	bitgetReduces                  = "/reduces"
	bitgetInsLoan                  = "ins-loan/"
	bitgetProductInfos             = "product-infos"
	bitgetEnsureCoinsConvert       = "ensure-coins-convert"
	bitgetLTVConvert               = "ltv-convert"
	bitgetTransferred              = "transfered" //nolint:misspell // Bitget spelling mistake
	bitgetRiskUnit                 = "risk-unit"
	bitgetBindUID                  = "bind-uid"
	bitgetLoanOrder                = "loan-order"
	bitgetRepaidHistory            = "repaid-history"

	// Websocket endpoints
	// Unauthenticated
	bitgetCandleDailyChannel = "candle1D" // There's one of these for each time period, but we'll ignore those for now
	bitgetBookFullChannel    = "books"    // There's more of these for varying orderbook depths, ignored for now
	bitgetIndexPriceChannel  = "index-price"

	// Authenticated
	bitgetFillChannel             = "fill"
	bitgetOrdersChannel           = "orders"
	bitgetOrdersAlgoChannel       = "orders-algo"
	bitgetPositionsChannel        = "positions"
	bitgetPositionsHistoryChannel = "positions-history"
	bitgetAccountCrossedChannel   = "account-crossed"
	bitgetOrdersCrossedChannel    = "orders-crossed"
	bitgetAccountIsolatedChannel  = "account-isolated"
	bitgetOrdersIsolatedChannel   = "orders-isolated"

	// Error strings
	errIntervalNotSupported = "interval not supported"
	errWebsocketGeneric     = "%v - Websocket error, code: %v message: %v"
	errWebsocketLoginFailed = "%v - Websocket login failed: %v"
)

var (
	errBusinessTypeEmpty              = errors.New("businessType cannot be empty")
	errCurrencyEmpty                  = errors.New("currency cannot be empty")
	errProductTypeEmpty               = errors.New("productType cannot be empty")
	errSubaccountEmpty                = errors.New("subaccounts cannot be empty")
	errNewStatusEmpty                 = errors.New("newStatus cannot be empty")
	errNewPermsEmpty                  = errors.New("newPerms cannot be empty")
	errPassphraseEmpty                = errors.New("passphrase cannot be empty")
	errLabelEmpty                     = errors.New("label cannot be empty")
	errAPIKeyEmpty                    = errors.New("apiKey cannot be empty")
	errFromToMutex                    = errors.New("exactly one of fromAmount and toAmount must be set")
	errTraceIDEmpty                   = errors.New("traceID cannot be empty")
	errGranEmpty                      = errors.New("granularity cannot be empty")
	errEndTimeEmpty                   = errors.New("endTime cannot be empty")
	errSideEmpty                      = fmt.Errorf("%w, empty order side", order.ErrSideIsInvalid)
	errOrderTypeEmpty                 = fmt.Errorf("%w empty order type", order.ErrTypeIsInvalid)
	errStrategyEmpty                  = errors.New("strategy cannot be empty")
	errLimitPriceEmpty                = fmt.Errorf("%w: Price below minimum for limit orders", order.ErrPriceBelowMin)
	errOrdersEmpty                    = errors.New("orders cannot be empty")
	errTriggerPriceEmpty              = fmt.Errorf("%w: TriggerPrice below minimum", order.ErrPriceBelowMin)
	errTriggerTypeEmpty               = errors.New("triggerType cannot be empty")
	errAccountTypeEmpty               = errors.New("accountType cannot be empty")
	errFromTypeEmpty                  = errors.New("fromType cannot be empty")
	errToTypeEmpty                    = errors.New("toType cannot be empty")
	errCurrencyAndPairEmpty           = errors.New("currency and pair cannot both be empty")
	errFromIDEmpty                    = errors.New("fromID cannot be empty")
	errToIDEmpty                      = errors.New("toID cannot be empty")
	errTransferTypeEmpty              = errors.New("transferType cannot be empty")
	errAddressEmpty                   = errors.New("address cannot be empty")
	errMarginCoinEmpty                = errors.New("marginCoin cannot be empty")
	errAmountEmpty                    = errors.New("amount cannot be empty")
	errOpenAmountEmpty                = fmt.Errorf("%w: OpenAmount below minimum", order.ErrAmountBelowMin)
	errOpenPriceEmpty                 = fmt.Errorf("%w: OpenPrice below minimum", order.ErrPriceBelowMin)
	errLeverageEmpty                  = errors.New("leverage cannot be empty")
	errMarginModeEmpty                = fmt.Errorf("%w margin mode can not be empty", margin.ErrInvalidMarginType)
	errPositionModeEmpty              = errors.New("positionMode cannot be empty")
	errNewClientOrderIDEmpty          = errors.New("newClientOrderID cannot be empty")
	errPlanTypeEmpty                  = errors.New("planType cannot be empty")
	errPlanOrderIDEmpty               = errors.New("planOrderID cannot be empty")
	errHoldSideEmpty                  = errors.New("holdSide cannot be empty")
	errExecutePriceEmpty              = fmt.Errorf("%w: ExecutePrice below minimum", order.ErrPriceBelowMin)
	errIDListEmpty                    = errors.New("idList cannot be empty")
	errLoanTypeEmpty                  = errors.New("loanType cannot be empty")
	errProductIDEmpty                 = errors.New("productID cannot be empty")
	errPeriodTypeEmpty                = errors.New("periodType cannot be empty")
	errLoanCoinEmpty                  = fmt.Errorf("%w: LoanCoin is required", currency.ErrCurrencyCodeEmpty)
	errCollateralCoinEmpty            = fmt.Errorf("%w: CollateralCoin is required", currency.ErrCurrencyCodeEmpty)
	errTermEmpty                      = errors.New("term cannot be empty")
	errCollateralAmountEmpty          = fmt.Errorf("%w: CollateralAmount below minimum", order.ErrAmountBelowMin)
	errCollateralLoanMutex            = errors.New("exactly one of collateralAmount and loanAmount must be set")
	errReviseTypeEmpty                = errors.New("reviseType cannot be empty")
	errUnknownPairQuote               = errors.New("unknown pair quote; pair can't be split due to lack of delimiter and unclear base length")
	errStrategyMutex                  = errors.New("only one of immediate or cancel, fill or kill, and post only can be set to true")
	errReturnEmpty                    = errors.New("returned data unexpectedly empty")
	errAuthenticatedWebsocketDisabled = errors.New("authenticatedWebsocketAPISupport not enabled")
	errAssetModeEmpty                 = errors.New("assetMode cannot be empty")
	errTakeProfitTriggerPriceEmpty    = fmt.Errorf("%w: TakeProfitTriggerPrice below minimum", order.ErrPriceBelowMin)
	errStopLossTriggerPriceEmpty      = fmt.Errorf("%w: StopLossTriggerPrice below minimum", order.ErrPriceBelowMin)
	errOrderIDMutex                   = errors.New("exactly one of orderID and clientOrderID must be set")
	errProductTypeAndPairEmpty        = errors.New("productType and pair cannot both be empty")

	prodTypes = []string{"USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES"}
	planTypes = []string{"normal_plan", "track_plan", "profit_loss"}
)

// QueryAnnouncements returns announcements from the exchange, filtered by type and time
func (e *Exchange) QueryAnnouncements(ctx context.Context, annType string, startTime, endTime time.Time) ([]AnnResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("annType", annType)
	params.Values.Set("language", "en_US")
	var resp []AnnResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetAnnouncements, params.Values, &resp)
}

// GetTime returns the server's time
func (e *Exchange) GetTime(ctx context.Context) (*TimeResp, error) {
	var resp TimeResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetTime, nil, &resp)
}

// GetTradeRate returns the fees the user would face for trading a given symbol
func (e *Exchange) GetTradeRate(ctx context.Context, pair currency.Pair, businessType string) (*TradeRateResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if businessType == "" {
		return nil, errBusinessTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("businessType", businessType)
	var resp TradeRateResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetCommon+bitgetTradeRate, vals, nil, &resp)
}

// GetSpotTransactionRecords returns the user's spot transaction records
func (e *Exchange) GetSpotTransactionRecords(ctx context.Context, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) ([]SpotTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp []SpotTrResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetSpotRecord, params.Values, nil, &resp)
}

// GetFuturesTransactionRecords returns the user's futures transaction records
func (e *Exchange) GetFuturesTransactionRecords(ctx context.Context, productType string, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) ([]FutureTrResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	params.Values.Set("marginCoin", currency.String())
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp []FutureTrResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetFutureRecord, params.Values, nil, &resp)
}

// GetMarginTransactionRecords returns the user's margin transaction records
func (e *Exchange) GetMarginTransactionRecords(ctx context.Context, marginType string, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) ([]MarginTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if marginType != "" {
		params.Values.Set("marginType", marginType)
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp []MarginTrResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetMarginRecord, params.Values, nil, &resp)
}

// GetP2PTransactionRecords returns the user's P2P transaction records
func (e *Exchange) GetP2PTransactionRecords(ctx context.Context, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) ([]P2PTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp []P2PTrResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetP2PRecord, params.Values, nil, &resp)
}

// GetP2PMerchantList returns detailed information on merchants
func (e *Exchange) GetP2PMerchantList(ctx context.Context, online string, limit, pagination int64) (*P2PMerListResp, error) {
	vals := url.Values{}
	vals.Set("online", online)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp P2PMerListResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetMerchantList, vals, nil, &resp)
}

// GetMerchantInfo returns detailed information on the user as a merchant
func (e *Exchange) GetMerchantInfo(ctx context.Context) (*P2PMerInfoResp, error) {
	var resp P2PMerInfoResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetMerchantInfo, nil, nil, &resp)
}

// GetMerchantP2POrders returns information on the user's P2P orders
func (e *Exchange) GetMerchantP2POrders(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, ordNum int64, status, side string, cryptoCurrency, fiatCurrency currency.Code) (*P2POrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("advNo", strconv.FormatInt(adNum, 10))
	params.Values.Set("orderNo", strconv.FormatInt(ordNum, 10))
	params.Values.Set("status", status)
	params.Values.Set("side", side)
	if !cryptoCurrency.IsEmpty() {
		params.Values.Set("coin", cryptoCurrency.String())
	}
	params.Values.Set("fiat", fiatCurrency.String())
	var resp P2POrdersResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetOrderList, params.Values, nil, &resp)
}

// GetMerchantAdvertisementList returns information on a variety of merchant advertisements
func (e *Exchange) GetMerchantAdvertisementList(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, payMethodID int64, status, side, orderBy, sourceType string, cryptoCurrency, fiatCurrency currency.Code) (*P2PAdListResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("advNo", strconv.FormatInt(adNum, 10))
	params.Values.Set("payMethodId", strconv.FormatInt(payMethodID, 10))
	params.Values.Set("status", status)
	params.Values.Set("side", side)
	if !cryptoCurrency.IsEmpty() {
		params.Values.Set("coin", cryptoCurrency.String())
	}
	params.Values.Set("fiat", fiatCurrency.String())
	params.Values.Set("orderBy", orderBy)
	params.Values.Set("sourceType", sourceType)
	var resp P2PAdListResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetAdvList, params.Values, nil, &resp)
}

// GetSpotWhaleNetFlow returns the amount whales have been trading in a specified pair recently
func (e *Exchange) GetSpotWhaleNetFlow(ctx context.Context, pair currency.Pair) ([]WhaleNetFlowResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetSpot + bitgetMarket + bitgetWhaleNetFlow
	var resp []WhaleNetFlowResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesActiveVolume returns the active volume of a specified pair
func (e *Exchange) GetFuturesActiveVolume(ctx context.Context, pair currency.Pair, period string) ([]ActiveVolumeResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetTakerBuySell
	var resp []ActiveVolumeResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesPositionRatios returns the ratio of long to short positions for a specified pair
func (e *Exchange) GetFuturesPositionRatios(ctx context.Context, pair currency.Pair, period string) ([]PosRatFutureResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetPositionLongShort
	var resp []PosRatFutureResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetMarginPositionRatios returns the ratio of long to short positions for a specified pair in margin accounts
func (e *Exchange) GetMarginPositionRatios(ctx context.Context, pair currency.Pair, period string, cur currency.Code) ([]PosRatMarginResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if period != "" {
		vals.Set("period", period)
	}
	vals.Set("coin", cur.String())
	path := bitgetMargin + bitgetMarket + bitgetLongShortRatio
	var resp []PosRatMarginResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetMarginLoanGrowth returns the growth rate of borrowed funds for a specified pair in margin accounts
func (e *Exchange) GetMarginLoanGrowth(ctx context.Context, pair currency.Pair, period string, cur currency.Code) ([]LoanGrowthResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if period != "" {
		vals.Set("period", period)
	}
	vals.Set("coin", cur.String())
	path := bitgetMargin + bitgetMarket + bitgetLoanGrowth
	var resp []LoanGrowthResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetIsolatedBorrowingRatio returns the ratio of borrowed funds between base and quote currencies, after converting to USDT, within isolated margin accounts
func (e *Exchange) GetIsolatedBorrowingRatio(ctx context.Context, pair currency.Pair, period string) ([]BorrowRatioResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if period != "" {
		vals.Set("period", period)
	}
	path := bitgetMargin + bitgetMarket + bitgetIsolatedBorrowRate
	var resp []BorrowRatioResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesRatios returns the ratio of long to short positions for a specified pair
func (e *Exchange) GetFuturesRatios(ctx context.Context, pair currency.Pair, period string) ([]RatioResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetLongShort
	var resp []RatioResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetSpotFundFlows returns information on volumes and buy/sell ratios for whales, dolphins, and fish for a particular pair
func (e *Exchange) GetSpotFundFlows(ctx context.Context, pair currency.Pair, period string) (*FundFlowResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if period != "" {
		vals.Set("period", period)
	}
	path := bitgetSpot + bitgetMarket + bitgetFundFlow
	var resp FundFlowResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetTradeSupportSymbols returns a list of supported symbols
func (e *Exchange) GetTradeSupportSymbols(ctx context.Context) (*SymbolsResp, error) {
	path := bitgetSpot + bitgetMarket + bitgetSupportSymbols
	var resp SymbolsResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, nil, &resp)
}

// GetSpotWhaleFundFlows returns the amount whales have been trading in a specified pair recently
func (e *Exchange) GetSpotWhaleFundFlows(ctx context.Context, pair currency.Pair) ([]WhaleFundFlowResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetSpot + bitgetMarket + bitgetFundNetFlow
	var resp []WhaleFundFlowResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesAccountRatios returns the ratio of long to short positions for a specified pair
func (e *Exchange) GetFuturesAccountRatios(ctx context.Context, pair currency.Pair, period string) ([]AccountRatioResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetAccountLongShort
	var resp []AccountRatioResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// CreateVirtualSubaccounts creates a batch of virtual subaccounts. These names must use English letters, no spaces, no numbers, and be exactly 8 characters long.
func (e *Exchange) CreateVirtualSubaccounts(ctx context.Context, subaccounts []string) (*CrVirSubResp, error) {
	if len(subaccounts) == 0 {
		return nil, errSubaccountEmpty
	}
	path := bitgetUser + bitgetCreate + bitgetVirtualSubaccount
	req := map[string]any{
		"subAccountList": subaccounts,
	}
	var resp CrVirSubResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ModifyVirtualSubaccount changes the permissions and/or status of a virtual subaccount
func (e *Exchange) ModifyVirtualSubaccount(ctx context.Context, subaccountID, newStatus string, newPerms []string) (*SuccessBool, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	if newStatus == "" {
		return nil, errNewStatusEmpty
	}
	if len(newPerms) == 0 {
		return nil, errNewPermsEmpty
	}
	path := bitgetUser + bitgetModify + bitgetVirtualSubaccount
	req := map[string]any{
		"subAccountUid": subaccountID,
		"status":        newStatus,
		"permList":      newPerms,
	}
	var resp SuccessBool
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &ResultWrapper{Result: &resp})
}

// CreateSubaccountAndAPIKey creates a subaccounts and an API key. Every account can have up to 20 sub-accounts, and every API key can have up to 10 API keys. The name of the sub-account must be exactly 8 English letters. The passphrase of the API key must be 8-32 letters and/or numbers. The label must be 20 or fewer characters. A maximum of 30 IPs can be a part of the whitelist.
func (e *Exchange) CreateSubaccountAndAPIKey(ctx context.Context, subaccountName, passphrase, label string, allowedIPList, permList []string) ([]CrSubAccAPIKeyResp, error) {
	if subaccountName == "" {
		return nil, errSubaccountEmpty
	}
	req := map[string]any{
		"subAccountName": subaccountName,
		"passphrase":     passphrase,
		"label":          label,
		"ipList":         allowedIPList,
		"permList":       permList,
	}
	var resp []CrSubAccAPIKeyResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, bitgetUser+bitgetBatchCreateSubAccAPI, nil, req, &resp)
}

// GetVirtualSubaccounts returns a list of the user's virtual sub-accounts
func (e *Exchange) GetVirtualSubaccounts(ctx context.Context, limit, pagination int64, status string) (*GetVirSubResp, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	vals.Set("status", status)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetList
	var resp GetVirSubResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CreateAPIKey creates an API key for the selected virtual sub-account
func (e *Exchange) CreateAPIKey(ctx context.Context, subaccountID, passphrase, label string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	if passphrase == "" {
		return nil, errPassphraseEmpty
	}
	if label == "" {
		return nil, errLabelEmpty
	}
	path := bitgetUser + bitgetCreate + bitgetVirtualSubaccount + "-" + bitgetAPIKey
	req := map[string]any{
		"subAccountUid": subaccountID,
		"passphrase":    passphrase,
		"label":         label,
		"ipList":        whiteList,
		"permList":      permList,
	}
	var resp AlterAPIKeyResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ModifyAPIKey modifies the label, IP whitelist, and/or permissions of the API key associated with the selected virtual sub-account
func (e *Exchange) ModifyAPIKey(ctx context.Context, subaccountID, passphrase, label, apiKey string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
	if apiKey == "" {
		return nil, errAPIKeyEmpty
	}
	if passphrase == "" {
		return nil, errPassphraseEmpty
	}
	if label == "" {
		return nil, errLabelEmpty
	}
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	path := bitgetUser + bitgetModify + bitgetVirtualSubaccount + "-" + bitgetAPIKey
	req := map[string]any{
		"subAccountUid":    subaccountID,
		"passphrase":       passphrase,
		"label":            label,
		"subAccountApiKey": apiKey,
		"ipList":           whiteList,
		"permList":         permList,
	}
	var resp AlterAPIKeyResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetAPIKeys lists the API keys associated with the selected virtual sub-account
func (e *Exchange) GetAPIKeys(ctx context.Context, subaccountID string) ([]GetAPIKeyResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	vals := url.Values{}
	vals.Set("subAccountUid", subaccountID)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetAPIKey + "-" + bitgetList
	var resp []GetAPIKeyResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetFundingAssets returns the user's assets
func (e *Exchange) GetFundingAssets(ctx context.Context, currency currency.Code) ([]FundingAssetsResp, error) {
	vals := url.Values{}
	if !currency.IsEmpty() {
		vals.Set("coin", currency.String())
	}
	var resp []FundingAssetsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetAccount+bitgetFundingAssets, vals, nil, &resp)
}

// GetBotAccountAssets returns the user's bot account assets
func (e *Exchange) GetBotAccountAssets(ctx context.Context, accountType string) ([]BotAccAssetsResp, error) {
	vals := url.Values{}
	if accountType != "" {
		vals.Set("accountType", accountType)
	}
	var resp []BotAccAssetsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetAccount+bitgetBotAssets, vals, nil, &resp)
}

// GetAssetOverview returns an overview of the user's assets across various account types
func (e *Exchange) GetAssetOverview(ctx context.Context) ([]AssetOverviewResp, error) {
	var resp []AssetOverviewResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetAccount+bitgetAllAccountBalance, nil, nil, &resp)
}

// GetConvertCoins returns a list of supported currencies, your balance in those currencies, and the maximum and minimum tradable amounts of those currencies
func (e *Exchange) GetConvertCoins(ctx context.Context) ([]ConvertCoinsResp, error) {
	var resp []ConvertCoinsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetCurrencies, nil, nil, &resp)
}

// GetQuotedPrice returns the price of a given amount of one currency in terms of another currency, and an ID for this quote, to be used in a subsequent conversion
func (e *Exchange) GetQuotedPrice(ctx context.Context, fromCurrency, toCurrency currency.Code, fromAmount, toAmount float64) (*QuotedPriceResp, error) {
	if fromCurrency.IsEmpty() || toCurrency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if (fromAmount <= 0 && toAmount <= 0) || (fromAmount != 0 && toAmount != 0) {
		return nil, errFromToMutex
	}
	vals := url.Values{}
	vals.Set("fromCoin", fromCurrency.String())
	vals.Set("toCoin", toCurrency.String())
	if fromAmount != 0 {
		vals.Set("fromCoinSize", strconv.FormatFloat(fromAmount, 'f', -1, 64))
	} else {
		vals.Set("toCoinSize", strconv.FormatFloat(toAmount, 'f', -1, 64))
	}
	var resp QuotedPriceResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetQuotedPrice, vals, nil, &resp)
}

// CommitConversion commits a conversion previously quoted by GetQuotedPrice. This quote has to have been issued within the last 8 seconds.
func (e *Exchange) CommitConversion(ctx context.Context, fromCurrency, toCurrency currency.Code, traceID string, fromAmount, toAmount, price float64) (*CommitConvResp, error) {
	if fromCurrency.IsEmpty() || toCurrency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if traceID == "" {
		return nil, errTraceIDEmpty
	}
	if fromAmount <= 0 || toAmount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	req := map[string]any{
		"fromCoin":     fromCurrency,
		"toCoin":       toCurrency,
		"traceId":      traceID,
		"fromCoinSize": strconv.FormatFloat(fromAmount, 'f', -1, 64),
		"toCoinSize":   strconv.FormatFloat(toAmount, 'f', -1, 64),
		"cnvtPrice":    strconv.FormatFloat(price, 'f', -1, 64),
	}
	var resp CommitConvResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, bitgetConvert+bitgetTrade, nil, req, &resp)
}

// GetConvertHistory returns a list of the user's previous conversions
func (e *Exchange) GetConvertHistory(ctx context.Context, startTime, endTime time.Time, limit, pagination int64) (*ConvHistResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp ConvHistResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetConvertRecord, params.Values, nil, &resp)
}

// GetBGBConvertCoins returns a list of available currencies, with information on converting them to BGB
func (e *Exchange) GetBGBConvertCoins(ctx context.Context) ([]BGBConvertCoinsResp, error) {
	var resp struct {
		CoinList []BGBConvertCoinsResp `json:"coinList"`
	}
	return resp.CoinList, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetConvertCoinList, nil, nil, &resp)
}

// ConvertBGB converts all funds in the listed currencies to BGB
func (e *Exchange) ConvertBGB(ctx context.Context, currencies []currency.Code) ([]ConvertBGBResp, error) {
	if len(currencies) == 0 {
		return nil, errCurrencyEmpty
	}
	req := map[string]any{
		"coinList": currencies,
	}
	var resp struct {
		OrderList []ConvertBGBResp `json:"orderList"`
	}
	return resp.OrderList, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, bitgetConvert+bitgetBGBConvert, nil, req, &resp)
}

// GetBGBConvertHistory returns a list of the user's previous BGB conversions
func (e *Exchange) GetBGBConvertHistory(ctx context.Context, orderID, limit, pagination int64, startTime, endTime time.Time) ([]BGBConvHistResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp []BGBConvHistResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetBGBConvertRecords, params.Values, nil, &resp)
}

// GetCoinInfo returns information on all supported spot currencies, or a single currency of the user's choice
func (e *Exchange) GetCoinInfo(ctx context.Context, currency currency.Code) ([]CoinInfoResp, error) {
	vals := url.Values{}
	if !currency.IsEmpty() {
		vals.Set("coin", currency.String())
	}
	path := bitgetSpot + bitgetPublic + bitgetCoins
	var resp []CoinInfoResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate3, path, vals, &resp)
}

// GetSymbolInfo returns information on all supported spot trading pairs, or a single pair of the user's choice
func (e *Exchange) GetSymbolInfo(ctx context.Context, pair currency.Pair) ([]SymbolInfoResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetSpot + bitgetPublic + bitgetSymbols
	var resp []SymbolInfoResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotVIPFeeRate returns the different levels of VIP fee rates for spot trading
func (e *Exchange) GetSpotVIPFeeRate(ctx context.Context) ([]VIPFeeRateResp, error) {
	path := bitgetSpot + bitgetMarket + bitgetVIPFeeRate
	var resp []VIPFeeRateResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetSpotTickerInformation returns the ticker information for all trading pairs, or a single pair of the user's choice
func (e *Exchange) GetSpotTickerInformation(ctx context.Context, pair currency.Pair) ([]TickerResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetSpot + bitgetMarket + bitgetTickers
	var resp []TickerResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotMergeDepth returns part of the orderbook, with options to merge orders of similar price levels together, and to change how many results are returned. Limit's a string instead of the typical int64 because the API will accept a value of "max"
func (e *Exchange) GetSpotMergeDepth(ctx context.Context, pair currency.Pair, precision, limit string) (*DepthResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("precision", precision)
	vals.Set("limit", limit)
	path := bitgetSpot + bitgetMarket + bitgetMergeDepth
	var resp DepthResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetOrderbookDepth returns the orderbook for a given trading pair, with options to merge orders of similar price levels together, and to change how many results are returned.
func (e *Exchange) GetOrderbookDepth(ctx context.Context, pair currency.Pair, step string, limit uint8) (*OrderbookResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if step != "" {
		vals.Set("type", step)
	}
	vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetOrderbook
	var resp OrderbookResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotCandlestickData returns candlestick data for a given trading pair
func (e *Exchange) GetSpotCandlestickData(ctx context.Context, pair currency.Pair, granularity string, startTime, endTime time.Time, limit uint16, historic bool) ([]OneSpotCandle, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if granularity == "" {
		return nil, errGranEmpty
	}
	var path string
	var params Params
	params.Values = make(url.Values)
	if historic {
		if endTime.IsZero() || endTime.Equal(time.Unix(0, 0)) {
			return nil, errEndTimeEmpty
		}
		path = bitgetSpot + bitgetMarket + bitgetHistoryCandles
		params.Values.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	} else {
		path = bitgetSpot + bitgetMarket + bitgetCandles
		err := params.prepareDateString(startTime, endTime, true, true)
		if err != nil {
			return nil, err
		}
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("granularity", granularity)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	var resp []OneSpotCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, params.Values, &resp)
}

// GetRecentSpotFills returns the most recent trades for a given pair
func (e *Exchange) GetRecentSpotFills(ctx context.Context, pair currency.Pair, limit uint16) ([]MarketFillsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetFills
	var resp []MarketFillsResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetSpotMarketTrades returns trades for a given pair within a particular time range, and/or before a certain ID
func (e *Exchange) GetSpotMarketTrades(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) ([]MarketFillsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetSpot + bitgetMarket + bitgetFillsHistory
	var resp []MarketFillsResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, params.Values, &resp)
}

// PlaceSpotOrder places a spot order on the exchange
func (e *Exchange) PlaceSpotOrder(ctx context.Context, pair currency.Pair, side, orderType, strategy, clientOrderID, stpMode string, price, amount, triggerPrice, presetTPPrice, executeTPPrice, presetSLPrice, executeSLPrice float64, isCopyTradeLeader bool, acceptableDelay time.Duration) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if strategy == "" {
		return nil, errStrategyEmpty
	}
	if orderType == "limit" && price <= 0 {
		return nil, errLimitPriceEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"symbol":      pair,
		"side":        side,
		"orderType":   orderType,
		"force":       strategy,
		"price":       strconv.FormatFloat(price, 'f', -1, 64),
		"size":        strconv.FormatFloat(amount, 'f', -1, 64),
		"stpMode":     stpMode,
		"requestTime": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	if triggerPrice != 0 {
		req["triggerPrice"] = strconv.FormatFloat(triggerPrice, 'f', -1, 64)
		req["tpslType"] = "tpsl"
	} else {
		req["clientOid"] = clientOrderID
	}
	if acceptableDelay != 0 {
		req["receiveWindow"] = acceptableDelay.Milliseconds()
	}
	if presetTPPrice != 0 {
		req["presetTakeProfitPrice"] = strconv.FormatFloat(presetTPPrice, 'f', -1, 64)
		req["executeTakeProfitPrice"] = strconv.FormatFloat(executeTPPrice, 'f', -1, 64)
	}
	if presetSLPrice != 0 {
		req["presetStopLossPrice"] = strconv.FormatFloat(presetSLPrice, 'f', -1, 64)
		req["executeStopLossPrice"] = strconv.FormatFloat(executeSLPrice, 'f', -1, 64)
	}
	path := bitgetSpot + bitgetTrade + bitgetPlaceOrder
	var resp OrderIDStruct
	// I suspect the two rate limits have to do with distinguishing ordinary traders, and traders who are also copy trade leaders. Since this isn't detectable, it'll be handled in the relevant functions through a bool
	rLim := Rate10
	if isCopyTradeLeader {
		rLim = Rate1
	}
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// CancelAndPlaceSpotOrder cancels an order and places a new one on the exchange
func (e *Exchange) CancelAndPlaceSpotOrder(ctx context.Context, pair currency.Pair, oldClientOrderID, newClientOrderID string, price, amount, presetTPPrice, executeTPPrice, presetSLPrice, executeSLPrice float64, orderID int64) (*CancelAndPlaceResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if oldClientOrderID == "" && orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	req := map[string]any{
		"symbol": pair,
		"price":  strconv.FormatFloat(price, 'f', -1, 64),
		"size":   strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if oldClientOrderID != "" {
		req["clientOid"] = oldClientOrderID
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	if newClientOrderID != "" {
		req["newClientOid"] = newClientOrderID
	}
	if presetTPPrice != 0 {
		req["presetTakeProfitPrice"] = strconv.FormatFloat(presetTPPrice, 'f', -1, 64)
		req["executeTakeProfitPrice"] = strconv.FormatFloat(executeTPPrice, 'f', -1, 64)
	}
	if presetSLPrice != 0 {
		req["presetStopLossPrice"] = strconv.FormatFloat(presetSLPrice, 'f', -1, 64)
		req["executeStopLossPrice"] = strconv.FormatFloat(executeSLPrice, 'f', -1, 64)
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelReplaceOrder
	var resp CancelAndPlaceResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelAndPlaceSpotOrders cancels and places up to fifty orders on the exchange
func (e *Exchange) BatchCancelAndPlaceSpotOrders(ctx context.Context, orders []ReplaceSpotOrderStruct) ([]CancelAndPlaceResp, error) {
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"orderList": orders,
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchCancelReplaceOrder
	var resp []CancelAndPlaceResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// CancelSpotOrderByID cancels an order on the exchange
func (e *Exchange) CancelSpotOrderByID(ctx context.Context, pair currency.Pair, tpslType, clientOrderID string, orderID int64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"symbol":   pair,
		"tpslType": tpslType,
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceSpotOrders places up to fifty orders on the exchange
func (e *Exchange) BatchPlaceSpotOrders(ctx context.Context, pair currency.Pair, multiCurrencyMode, isCopyTradeLeader bool, orders []PlaceSpotOrderStruct) (*BatchOrderResp, error) {
	if pair.IsEmpty() && !multiCurrencyMode {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	if multiCurrencyMode {
		req["batchMode"] = "multiple"
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchOrders
	var resp BatchOrderResp
	rLim := Rate5
	if isCopyTradeLeader {
		rLim = Rate1
	}
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelOrders cancels up to fifty orders on the exchange
func (e *Exchange) BatchCancelOrders(ctx context.Context, pair currency.Pair, multiCurrencyMode bool, orderIDs []CancelSpotOrderStruct) (*BatchOrderResp, error) {
	if pair.IsEmpty() && !multiCurrencyMode {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orderIDs,
	}
	if multiCurrencyMode {
		req["batchMode"] = "multiple"
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchCancel
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelOrdersBySymbol cancels orders for a given symbol. Doesn't return information on failures/successes
func (e *Exchange) CancelOrdersBySymbol(ctx context.Context, pair currency.Pair) (string, error) {
	if pair.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	req := map[string]any{
		"symbol": pair,
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelSymbolOrder
	var resp struct {
		Symbol string `json:"symbol"`
	}
	return resp.Symbol, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetSpotOrderDetails returns information on a single order
func (e *Exchange) GetSpotOrderDetails(ctx context.Context, orderID int64, clientOrderID string, acceptableDelay time.Duration) ([]SpotOrderDetailData, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	vals := url.Values{}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if clientOrderID != "" {
		vals.Set("clientOid", clientOrderID)
	}
	vals.Set("requestTime", strconv.FormatInt(time.Now().UnixMilli(), 10))
	vals.Set("receiveWindow", strconv.FormatInt(acceptableDelay.Milliseconds(), 10))
	path := bitgetSpot + bitgetTrade + bitgetOrderInfo
	return e.spotOrderHelper(ctx, path, vals)
}

// GetUnfilledOrders returns information on the user's unfilled orders
func (e *Exchange) GetUnfilledOrders(ctx context.Context, pair currency.Pair, tpslType string, startTime, endTime time.Time, limit, pagination, orderID int64, acceptableDelay time.Duration) ([]UnfilledOrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("tpslType", tpslType)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("requestTime", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Values.Set("receiveWindow", strconv.FormatInt(acceptableDelay.Milliseconds(), 10))
	path := bitgetSpot + bitgetTrade + bitgetUnfilledOrders
	var resp []UnfilledOrdersResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetHistoricalSpotOrders returns the user's spot order history, within the last 90 days
func (e *Exchange) GetHistoricalSpotOrders(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination, orderID int64, tpslType string, acceptableDelay time.Duration) ([]SpotOrderDetailData, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("tpslType", tpslType)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("requestTime", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Values.Set("receiveWindow", strconv.FormatInt(acceptableDelay.Milliseconds(), 10))
	path := bitgetSpot + bitgetTrade + bitgetHistoryOrders
	return e.spotOrderHelper(ctx, path, params.Values)
}

// GetSpotFills returns information on the user's fulfilled orders in a certain pair
func (e *Exchange) GetSpotFills(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination, orderID int64) ([]SpotFillsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetSpot + bitgetTrade + "/" + bitgetFills
	var resp []SpotFillsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// PlacePlanSpotOrder sets up an order to be placed after certain conditions are met
func (e *Exchange) PlacePlanSpotOrder(ctx context.Context, pair currency.Pair, side, orderType, planType, triggerType, clientOrderID, strategy, stpMode string, triggerPrice, executePrice, amount float64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if triggerPrice <= 0 {
		return nil, errTriggerPriceEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if orderType == "limit" && executePrice <= 0 {
		return nil, errLimitPriceEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if triggerType == "" {
		return nil, errTriggerTypeEmpty
	}
	req := map[string]any{
		"symbol":       pair,
		"side":         side,
		"triggerPrice": strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"orderType":    orderType,
		"executePrice": strconv.FormatFloat(executePrice, 'f', -1, 64),
		"planType":     planType,
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
		"triggerType":  triggerType,
		"force":        strategy,
		"stpMode":      stpMode,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetTrade + bitgetPlacePlanOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &resp)
}

// ModifyPlanSpotOrder alters the price, trigger price, amount, or order type of a plan order
func (e *Exchange) ModifyPlanSpotOrder(ctx context.Context, orderID int64, clientOrderID, orderType string, triggerPrice, executePrice, amount float64) (*OrderIDStruct, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if triggerPrice <= 0 {
		return nil, errTriggerPriceEmpty
	}
	if orderType == "limit" && executePrice <= 0 {
		return nil, errLimitPriceEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"orderType":    orderType,
		"triggerPrice": strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"executePrice": strconv.FormatFloat(executePrice, 'f', -1, 64),
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	path := bitgetSpot + bitgetTrade + bitgetModifyPlanOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &resp)
}

// CancelPlanSpotOrder cancels a plan order
func (e *Exchange) CancelPlanSpotOrder(ctx context.Context, orderID int64, clientOrderID string) (*SuccessBool, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	req := make(map[string]any)
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelPlanOrder
	var resp SuccessBool
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &ResultWrapper{Result: &resp})
}

// GetCurrentSpotPlanOrders returns the user's current plan orders
func (e *Exchange) GetCurrentSpotPlanOrders(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) (*PlanSpotOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetTrade + bitgetCurrentPlanOrder
	var resp PlanSpotOrderResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSpotPlanSubOrder returns the sub-orders of a triggered plan order
func (e *Exchange) GetSpotPlanSubOrder(ctx context.Context, orderID int64) (*SubOrderResp, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	vals := url.Values{}
	vals.Set("planOrderId", strconv.FormatInt(orderID, 10))
	path := bitgetSpot + bitgetTrade + bitgetPlanSubOrder
	var resp SubOrderResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, vals, nil, &resp)
}

// GetSpotPlanOrderHistory returns the user's plan order history
func (e *Exchange) GetSpotPlanOrderHistory(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) (*PlanSpotOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	// Despite this not being included in the documentation, it's verified as working
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetTrade + bitgetPlanOrderHistory
	var resp PlanSpotOrderResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// BatchCancelSpotPlanOrders cancels all plan orders, with the option to restrict to only those for particular pairs
func (e *Exchange) BatchCancelSpotPlanOrders(ctx context.Context, pairs currency.Pairs) (*BatchOrderResp, error) {
	req := make(map[string]any)
	if len(pairs) > 0 {
		req["symbolList"] = pairs.Strings()
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchCancelPlanOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetAccountInfo returns the user's account information
func (e *Exchange) GetAccountInfo(ctx context.Context) (*AccountInfoResp, error) {
	path := bitgetSpot + bitgetAccount + bitgetInfo
	var resp AccountInfoResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path, nil, nil, &resp)
}

// GetAccountAssets returns information on the user's assets
func (e *Exchange) GetAccountAssets(ctx context.Context, currency currency.Code, assetType string) ([]AssetData, error) {
	vals := url.Values{}
	if !currency.IsEmpty() {
		vals.Set("coin", currency.String())
	}
	if assetType != "" {
		vals.Set("assetType", assetType)
	}
	path := bitgetSpot + bitgetAccount + bitgetAssets
	var resp []AssetData
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSpotSubaccountAssets returns information on assets in the user's sub-accounts
func (e *Exchange) GetSpotSubaccountAssets(ctx context.Context) ([]SubaccountAssetsResp, error) {
	path := bitgetSpot + bitgetAccount + bitgetSubaccountAssets
	var resp []SubaccountAssetsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// ModifyDepositAccount changes which account is automatically used for deposits of a particular currency
func (e *Exchange) ModifyDepositAccount(ctx context.Context, accountType string, currency currency.Code) (*SuccessBool, error) {
	if accountType == "" {
		return nil, errAccountTypeEmpty
	}
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	req := map[string]any{
		"coin":        currency,
		"accountType": accountType,
	}
	path := bitgetSpot + bitgetWallet + bitgetModifyDepositAccount
	var resp SuccessBool
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSpotAccountBills returns a section of the user's billing history
func (e *Exchange) GetSpotAccountBills(ctx context.Context, currency currency.Code, groupType, businessType string, startTime, endTime time.Time, limit, pagination int64) ([]SpotAccBillResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("groupType", groupType)
	params.Values.Set("businessType", businessType)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetSpot + bitgetAccount + bitgetBills
	var resp []SpotAccBillResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// TransferAsset transfers a certain amount of a currency or pair between different productType accounts
func (e *Exchange) TransferAsset(ctx context.Context, fromType, toType, clientOrderID string, currency currency.Code, pair currency.Pair, amount float64) (*TransferResp, error) {
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	if toType == "" {
		return nil, errToTypeEmpty
	}
	if currency.IsEmpty() && pair.IsEmpty() {
		return nil, errCurrencyAndPairEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"fromType": fromType,
		"toType":   toType,
		"amount":   strconv.FormatFloat(amount, 'f', -1, 64),
		"coin":     currency,
		"symbol":   pair,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetWallet + bitgetTransfer
	var resp TransferResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetTransferableCoinList returns a list of coins that can be transferred between the provided accounts
func (e *Exchange) GetTransferableCoinList(ctx context.Context, fromType, toType string) ([]string, error) {
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	if toType == "" {
		return nil, errToTypeEmpty
	}
	vals := url.Values{}
	vals.Set("fromType", fromType)
	vals.Set("toType", toType)
	path := bitgetSpot + bitgetWallet + bitgetTransferCoinInfo
	var resp []string
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubaccountTransfer transfers assets between sub-accounts
func (e *Exchange) SubaccountTransfer(ctx context.Context, fromType, toType, clientOrderID, fromID, toID string, currency currency.Code, pair currency.Pair, amount float64) (*TransferResp, error) {
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	if toType == "" {
		return nil, errToTypeEmpty
	}
	if currency.IsEmpty() && pair.IsEmpty() {
		return nil, errCurrencyAndPairEmpty
	}
	if fromID == "" {
		return nil, errFromIDEmpty
	}
	if toID == "" {
		return nil, errToIDEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"fromType":   fromType,
		"toType":     toType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
		"coin":       currency,
		"symbol":     pair,
		"fromUserId": fromID,
		"toUserId":   toID,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetWallet + bitgetSubaccountTransfer
	var resp TransferResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// WithdrawFunds withdraws funds from the user's account
func (e *Exchange) WithdrawFunds(ctx context.Context, currency currency.Code, transferType, address, chain, innerAddressType, areaCode, tag, note, clientOrderID, memberCode, identityType, companyName, firstName, lastName string, amount float64) (*OrderIDStruct, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if transferType == "" {
		return nil, errTransferTypeEmpty
	}
	if address == "" {
		return nil, errAddressEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"coin":         currency,
		"transferType": transferType,
		"address":      address,
		"chain":        chain,
		"innerToType":  innerAddressType,
		"areaCode":     areaCode,
		"tag":          tag,
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
		"remark":       note,
		"memberCode":   memberCode,
		"identityType": identityType,
		"companyName":  companyName,
		"firstName":    firstName,
		"lastName":     lastName,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetWallet + bitgetWithdrawal
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetSubaccountTransferRecord returns the user's sub-account transfer history
func (e *Exchange) GetSubaccountTransferRecord(ctx context.Context, currency currency.Code, subaccountID, clientOrderID, role string, startTime, endTime time.Time, limit, pagination int64) ([]SubaccTfrRecResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("subUid", subaccountID)
	if role != "" {
		params.Values.Set("role", role)
	}
	if clientOrderID != "" {
		params.Values.Set("clientOid", clientOrderID)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetSpot + bitgetAccount + bitgetSubaccountTransferRecord
	var resp []SubaccTfrRecResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetTransferRecord returns the user's transfer history
func (e *Exchange) GetTransferRecord(ctx context.Context, currency currency.Code, fromType, clientOrderID string, startTime, endTime time.Time, limit, pagination, page int64) ([]TransferRecResp, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("coin", currency.String())
	params.Values.Set("fromType", fromType)
	if clientOrderID != "" {
		params.Values.Set("clientOid", clientOrderID)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if page != 0 {
		params.Values.Set("pageNum", strconv.FormatInt(page, 10))
	}
	path := bitgetSpot + bitgetAccount + bitgetTransferRecord
	var resp []TransferRecResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// SwitchBGBDeductionStatus switches the deduction of BGB for trading fees on and off
func (e *Exchange) SwitchBGBDeductionStatus(ctx context.Context, deduct bool) (bool, error) {
	req := make(map[string]any)
	if deduct {
		req["deduct"] = "on"
	} else {
		req["deduct"] = "off"
	}
	path := bitgetSpot + bitgetAccount + bitgetSwitchDeduct
	var resp bool
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, path, nil, req, &resp)
}

// GetDepositAddressForCurrency returns the user's deposit address for a particular currency
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, currency currency.Code, chain string, amount float64) (*DepositAddressResp, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	vals.Set("chain", chain)
	vals.Set("size", strconv.FormatFloat(amount, 'f', -1, 64))
	path := bitgetSpot + bitgetWallet + bitgetDepositAddress
	var resp DepositAddressResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSubaccountDepositAddress returns the deposit address for a particular currency and sub-account
func (e *Exchange) GetSubaccountDepositAddress(ctx context.Context, subaccountID, chain string, currency currency.Code, amount float64) (*DepositAddressResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("subUid", subaccountID)
	vals.Set("coin", currency.String())
	vals.Set("chain", chain)
	vals.Set("size", strconv.FormatFloat(amount, 'f', -1, 64))
	path := bitgetSpot + bitgetWallet + bitgetSubaccountDepositAddress
	var resp DepositAddressResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetBGBDeductionStatus returns the user's current BGB deduction status
func (e *Exchange) GetBGBDeductionStatus(ctx context.Context) (bool, error) {
	path := bitgetSpot + bitgetAccount + bitgetDeductInfo
	var resp struct {
		Deduct OnOffBool `json:"deduct"`
	}
	return bool(resp.Deduct), e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, nil, nil, &resp)
}

// CancelWithdrawal cancels a large withdrawal request that was placed in the last minute
func (e *Exchange) CancelWithdrawal(ctx context.Context, orderID int64) (*SuccessBool, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"orderId": strconv.FormatInt(orderID, 10),
	}
	path := bitgetSpot + bitgetWallet + bitgetCancelWithdrawal
	var resp *SuccessBool
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSubaccountDepositRecords returns the deposit history for a sub-account
func (e *Exchange) GetSubaccountDepositRecords(ctx context.Context, subaccountID string, currency currency.Code, pagination, limit int64, startTime, endTime time.Time) ([]SubaccDepRecResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("subUid", subaccountID)
	if currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetWallet + bitgetSubaccountDepositRecord
	var resp []SubaccDepRecResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetWithdrawalRecords returns the user's withdrawal history
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, currency currency.Code, clientOrderID string, startTime, endTime time.Time, pagination, orderID, limit int64) ([]WithdrawRecordsResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("clientOid", clientOrderID)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetWallet + bitgetWithdrawalRecord
	var resp []WithdrawRecordsResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetDepositRecords returns the user's cryptocurrency deposit history
func (e *Exchange) GetDepositRecords(ctx context.Context, crypto currency.Code, orderID, pagination, limit int64, startTime, endTime time.Time) ([]CryptoDepRecResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if !crypto.IsEmpty() {
		params.Values.Set("coin", crypto.String())
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetWallet + bitgetDepositRecord
	var resp []CryptoDepRecResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetFuturesVIPFeeRate returns the different levels of VIP fee rates for futures trading
func (e *Exchange) GetFuturesVIPFeeRate(ctx context.Context) ([]VIPFeeRateResp, error) {
	path := bitgetMix + bitgetMarket + bitgetVIPFeeRate
	var resp []VIPFeeRateResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetInterestRateHistory returns the historical interest rate for futures trading
func (e *Exchange) GetInterestRateHistory(ctx context.Context, currency currency.Code) (*InterestRateResp, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetMix + bitgetMarket + bitgetUnionInterestRateHistory
	var resp InterestRateResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate5, path, vals, &resp)
}

// GetInterestExchangeRate returns the interest exchange rate for all currencies
func (e *Exchange) GetInterestExchangeRate(ctx context.Context) ([]ExchangeRateResp, error) {
	path := bitgetMix + bitgetMarket + bitgetExchangeRate
	var resp []ExchangeRateResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate5, path, nil, &resp)
}

// GetDiscountRate returns the discount rate for all currencies
func (e *Exchange) GetDiscountRate(ctx context.Context) ([]DiscountRateResp, error) {
	path := bitgetMix + bitgetMarket + bitgetDiscountRate
	var resp []DiscountRateResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate5, path, nil, &resp)
}

// GetFuturesMergeDepth returns part of the orderbook, with options to merge orders of similar price levels together, and to change how many results are returned. Limit's a string instead of the typical int64 because the API will accept a value of "max"
func (e *Exchange) GetFuturesMergeDepth(ctx context.Context, pair currency.Pair, productType, precision, limit string) (*DepthResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	vals.Set("precision", precision)
	vals.Set("limit", limit)
	path := bitgetMix + bitgetMarket + bitgetMergeDepth
	var resp DepthResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesTicker returns the ticker information for a pair of the user's choice
func (e *Exchange) GetFuturesTicker(ctx context.Context, pair currency.Pair, productType string) ([]FutureTickerResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetTicker
	var resp []FutureTickerResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetAllFuturesTickers returns the ticker information for all pairs
func (e *Exchange) GetAllFuturesTickers(ctx context.Context, productType string) ([]FutureTickerResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetTickers
	var resp []FutureTickerResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetRecentFuturesFills returns the most recent trades for a given pair
func (e *Exchange) GetRecentFuturesFills(ctx context.Context, pair currency.Pair, productType string, limit int64) ([]MarketFillsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetFills
	var resp []MarketFillsResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesMarketTrades returns trades for a given pair within a particular time range, and/or before a certain ID
func (e *Exchange) GetFuturesMarketTrades(ctx context.Context, pair currency.Pair, productType string, limit, pagination int64, startTime, endTime time.Time) ([]MarketFillsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("productType", productType)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetFillsHistory
	var resp []MarketFillsResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, params.Values, &resp)
}

// GetFuturesCandlestickData returns candlestick data for a given pair within a particular time range
func (e *Exchange) GetFuturesCandlestickData(ctx context.Context, pair currency.Pair, productType, granularity, candleType string, startTime, endTime time.Time, limit uint16, mode CallMode) ([]OneFuturesCandle, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if granularity == "" {
		return nil, errGranEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	path := bitgetMix + bitgetMarket
	switch mode {
	case CallModeNormal:
		path += bitgetCandles
		params.Values.Set("kLineType", candleType)
	case CallModeHistory:
		path += bitgetHistoryCandles
	case CallModeIndex:
		path += bitgetHistoryIndexCandles
	case CallModeMark:
		path += bitgetHistoryMarkCandles
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("granularity", granularity)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	var resp []OneFuturesCandle
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, params.Values, &resp)
}

// GetOpenPositions returns the total positions of a particular pair
func (e *Exchange) GetOpenPositions(ctx context.Context, pair currency.Pair, productType string) (*OpenPositionsResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetOpenInterest
	var resp OpenPositionsResp
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetNextFundingTime returns the settlement time and period of a particular contract
func (e *Exchange) GetNextFundingTime(ctx context.Context, pair currency.Pair, productType string) ([]FundingTimeResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetFundingTime
	var resp []FundingTimeResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesPrices returns the current market, index, and mark prices for a given pair
func (e *Exchange) GetFuturesPrices(ctx context.Context, pair currency.Pair, productType string) ([]FuturesPriceResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetSymbolPrice
	var resp []FuturesPriceResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFundingHistorical returns the historical funding rates for a given pair
func (e *Exchange) GetFundingHistorical(ctx context.Context, pair currency.Pair, productType string, limit, pagination int64) ([]FundingHistoryResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	if limit != 0 {
		vals.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetHistoryFundRate
	var resp []FundingHistoryResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFundingCurrent returns the current funding rate for a given pair
func (e *Exchange) GetFundingCurrent(ctx context.Context, pair currency.Pair, productType string) ([]FundingCurrentResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetCurrentFundRate
	var resp []FundingCurrentResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetContractConfig returns details for a given contract
func (e *Exchange) GetContractConfig(ctx context.Context, pair currency.Pair, productType string) ([]ContractConfigResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetContracts
	var resp []ContractConfigResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetOneFuturesAccount returns details for the account associated with a given pair, margin coin, and product type
func (e *Exchange) GetOneFuturesAccount(ctx context.Context, pair currency.Pair, productType string, marginCoin currency.Code) (*OneAccResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin.String())
	path := bitgetMix + bitgetAccount + "/" + bitgetAccount
	var resp OneAccResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetAllFuturesAccounts returns details for all accounts
func (e *Exchange) GetAllFuturesAccounts(ctx context.Context, productType string) ([]FutureAccDetails, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetAccount + bitgetAccounts
	var resp []FutureAccDetails
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetFuturesSubaccountAssets returns details on the assets of all sub-accounts
func (e *Exchange) GetFuturesSubaccountAssets(ctx context.Context, productType string) ([]SubaccountFuturesResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetAccount + bitgetSubaccountAssets2
	var resp []SubaccountFuturesResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path, vals, nil, &resp)
}

// GetUSDTInterestHistory returns the interest history for USDT
func (e *Exchange) GetUSDTInterestHistory(ctx context.Context, currency currency.Code, productType string, idLessThan, limit int64, startTime, endTime time.Time) (*USDTInterestHistory, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("coin", currency.String())
	params.Values.Set("productType", productType)
	if idLessThan != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(idLessThan, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetAccount + bitgetInterestHistory
	var resp USDTInterestHistory
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, params.Values, nil, &resp)
}

// GetEstimatedOpenCount returns the estimated size of open orders for a given pair
func (e *Exchange) GetEstimatedOpenCount(ctx context.Context, pair currency.Pair, productType string, marginCoin currency.Code, openAmount, openPrice, leverage float64) (float64, error) {
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return 0, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return 0, errMarginCoinEmpty
	}
	if openAmount <= 0 {
		return 0, errOpenAmountEmpty
	}
	if openPrice <= 0 {
		return 0, errOpenPriceEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin.String())
	vals.Set("openAmount", strconv.FormatFloat(openAmount, 'f', -1, 64))
	vals.Set("openPrice", strconv.FormatFloat(openPrice, 'f', -1, 64))
	if leverage != 0 {
		vals.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	}
	path := bitgetMix + bitgetAccount + bitgetOpenCount
	var resp struct {
		Size float64 `json:"size,string"`
	}
	return resp.Size, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SetIsolatedAutoMargin sets the auto margin for a given pair
func (e *Exchange) SetIsolatedAutoMargin(ctx context.Context, pair currency.Pair, autoMargin OnOffBool, marginCoin currency.Code, holdSide string) (*SuccessBool, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if holdSide == "" {
		return nil, errHoldSideEmpty
	}
	req := map[string]any{
		"symbol":     pair,
		"autoMargin": autoMargin,
		"marginCoin": marginCoin,
		"holdSide":   holdSide,
	}
	path := bitgetMix + bitgetAccount + bitgetSetAutoMargin
	var resp SuccessBool
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ChangeLeverage changes the leverage for the given pair and product type
func (e *Exchange) ChangeLeverage(ctx context.Context, pair currency.Pair, productType, holdSide string, marginCoin currency.Code, leverage float64) (*ChangeLeverageResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if leverage == 0 {
		return nil, errLeverageEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginCoin":  marginCoin,
		"holdSide":    holdSide,
		"leverage":    strconv.FormatFloat(leverage, 'f', -1, 64),
	}
	path := bitgetMix + bitgetAccount + bitgetSetLeverage
	var resp ChangeLeverageResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// AdjustMargin adds or subtracts margin from a position
func (e *Exchange) AdjustMargin(ctx context.Context, pair currency.Pair, productType, holdSide string, marginCoin currency.Code, amount float64) error {
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return errMarginCoinEmpty
	}
	if amount == 0 {
		return errAmountEmpty
	}
	if holdSide == "" {
		return errHoldSideEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginCoin":  marginCoin,
		"amount":      strconv.FormatFloat(amount, 'f', -1, 64),
		"holdSide":    holdSide,
	}
	path := bitgetMix + bitgetAccount + bitgetSetMargin
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, nil)
}

// SetUSDTAssetMode sets the asset mode for USDT pairs
func (e *Exchange) SetUSDTAssetMode(ctx context.Context, productType, assetMode string) (*SuccessBool, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if assetMode == "" {
		return nil, errAssetModeEmpty
	}
	req := map[string]any{
		"productType": productType,
		"assetMode":   assetMode,
	}
	path := bitgetMix + bitgetAccount + bitgetSetAssetMode
	var resp SuccessBool
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate2, http.MethodPost, path, nil, req, &resp)
}

// ChangeMarginMode changes the margin mode for a given pair. Can only be done when there the user has no open positions or orders
func (e *Exchange) ChangeMarginMode(ctx context.Context, pair currency.Pair, productType, marginMode string, marginCoin currency.Code) (*ChangeMarginModeResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginCoin":  marginCoin,
		"marginMode":  marginMode,
	}
	path := bitgetMix + bitgetAccount + bitgetSetMarginMode
	var resp ChangeMarginModeResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ChangePositionMode changes the position mode for any pair. Having any positions or orders on any side of any pair may cause this to fail.
func (e *Exchange) ChangePositionMode(ctx context.Context, productType, positionMode string) (string, error) {
	if productType == "" {
		return "", errProductTypeEmpty
	}
	if positionMode == "" {
		return "", errPositionModeEmpty
	}
	req := map[string]any{
		"productType": productType,
		"posMode":     positionMode,
	}
	path := bitgetMix + bitgetAccount + bitgetSetPositionMode
	var resp struct {
		PosMode string `json:"posMode"`
	}
	return resp.PosMode, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesAccountBills returns a section of the user's billing history
func (e *Exchange) GetFuturesAccountBills(ctx context.Context, productType, businessType, onlyFunding string, currency currency.Code, pagination, limit int64, startTime, endTime time.Time) (*FutureAccBillResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("businessType", businessType)
	params.Values.Set("onlyFunding", onlyFunding)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetAccount + bitgetBill
	var resp FutureAccBillResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetPositionTier returns the position configuration for a given pair
func (e *Exchange) GetPositionTier(ctx context.Context, productType string, pair currency.Pair) ([]PositionTierResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("symbol", pair.String())
	path := bitgetMix + bitgetMarket + bitgetQueryPositionLever
	var resp []PositionTierResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetSinglePosition returns position details for a given productType, pair, and marginCoin. The exchange recommends using the websocket feed instead, as information from this endpoint may be delayed during settlement or market fluctuations
func (e *Exchange) GetSinglePosition(ctx context.Context, productType string, pair currency.Pair, marginCoin currency.Code) ([]SinglePositionResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("symbol", pair.String())
	vals.Set("marginCoin", marginCoin.String())
	path := bitgetMix + bitgetPosition + bitgetSinglePosition
	var resp []SinglePositionResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetAllPositions returns position details for a given productType and marginCoin. The exchange recommends using the websocket feed instead, as information from this endpoint may be delayed during settlement or market fluctuations
func (e *Exchange) GetAllPositions(ctx context.Context, productType string, marginCoin currency.Code) ([]AllPositionResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin.String())
	path := bitgetMix + bitgetPosition + bitgetAllPositions
	var resp []AllPositionResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetHistoricalPositions returns historical position details, up to a maximum of three months ago
func (e *Exchange) GetHistoricalPositions(ctx context.Context, pair currency.Pair, productType string, pagination, limit int64, startTime, endTime time.Time) (*HistPositionResp, error) {
	if pair.IsEmpty() && productType == "" {
		return nil, errProductTypeAndPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("productType", productType)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetPosition + bitgetHistoryPosition
	var resp HistPositionResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// PlaceFuturesOrder places a futures order on the exchange
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, pair currency.Pair, productType, marginMode, side, tradeSide, orderType, strategy, clientOID, stpMode string, marginCoin currency.Code, stopSurplusPrice, stopLossPrice, amount, price float64, reduceOnly, isCopyTradeLeader bool) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if orderType == "limit" && price <= 0 {
		return nil, errLimitPriceEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginMode":  marginMode,
		"marginCoin":  marginCoin,
		"side":        side,
		"tradeSide":   tradeSide,
		"orderType":   orderType,
		"force":       strategy,
		"size":        strconv.FormatFloat(amount, 'f', -1, 64),
		"price":       strconv.FormatFloat(price, 'f', -1, 64),
		"stpMode":     stpMode,
	}
	if clientOID != "" {
		req["clientOid"] = clientOID
	}
	if reduceOnly {
		req["reduceOnly"] = "YES"
	}
	if stopSurplusPrice != 0 {
		req["presetStopSurplusPrice"] = strconv.FormatFloat(stopSurplusPrice, 'f', -1, 64)
	}
	if stopLossPrice != 0 {
		req["presetStopLossPrice"] = strconv.FormatFloat(stopLossPrice, 'f', -1, 64)
	}
	path := bitgetMix + bitgetOrder + bitgetPlaceOrder
	rLim := Rate10
	if isCopyTradeLeader {
		rLim = Rate1
	}
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// PlaceReversal attempts to close a position, in part or in whole, and opens a position of corresponding size on the opposite side. This operation may only be done in part under certain margin levels, market conditions, or other unspecified factors. If a reversal is attempted for an amount greater than the current outstanding position, that position will be closed, and a new position will be opened for the amount of the closed position; not the amount specified in the request. The side specified in the parameter should correspond to the side of the position you're attempting to close; if the original is open_long, use close_long; if the original is open_short, use close_short; if the original is sell_single, use buy_single. If the position is sell_single or buy_single, the amount parameter will be ignored, and the entire position will be closed, with a corresponding amount opened on the opposite side.
func (e *Exchange) PlaceReversal(ctx context.Context, pair currency.Pair, marginCoin currency.Code, productType, side, tradeSide, clientOID string, amount float64, isCopyTradeLeader bool) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"symbol":      pair,
		"marginCoin":  marginCoin,
		"productType": productType,
		"side":        side,
		"tradeSide":   tradeSide,
		"size":        strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if clientOID != "" {
		req["clientOid"] = clientOID
	}
	path := bitgetMix + bitgetOrder + bitgetClickBackhand
	rLim := Rate10
	if isCopyTradeLeader {
		rLim = Rate1
	}
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceFuturesOrders places multiple orders at once. Can also be used to modify the take-profit and stop-loss of an open position.
func (e *Exchange) BatchPlaceFuturesOrders(ctx context.Context, pair currency.Pair, productType, marginMode string, marginCoin currency.Code, orders []PlaceFuturesOrderStruct, isCopyTradeLeader bool) (*BatchOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginCoin":  marginCoin,
		"marginMode":  marginMode,
		"orderList":   orders,
	}
	path := bitgetMix + bitgetOrder + bitgetBatchPlaceOrder
	rLim := Rate5
	if isCopyTradeLeader {
		rLim = Rate1
	}
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// ModifyFuturesOrder can change the size, price, take-profit, and stop-loss of an order. Size and price have to be modified at the same time, or the request will fail. If size and price are altered, the old order will be cancelled, and a new one will be created asynchronously. Due to the asynchronous creation of a new order, a new ClientOrderID must be supplied so it can be tracked.
func (e *Exchange) ModifyFuturesOrder(ctx context.Context, orderID int64, clientOrderID, productType, newClientOrderID string, pair currency.Pair, newAmount, newPrice, newTakeProfit, newStopLoss float64) (*OrderIDStruct, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if newClientOrderID == "" {
		return nil, errNewClientOrderIDEmpty
	}
	req := map[string]any{
		"symbol":                    pair,
		"productType":               productType,
		"newClientOid":              newClientOrderID,
		"newPresetStopSurplusPrice": strconv.FormatFloat(newTakeProfit, 'f', -1, 64),
		"newPresetStopLossPrice":    strconv.FormatFloat(newStopLoss, 'f', -1, 64),
	}
	if orderID != 0 {
		req["orderId"] = orderID
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if newAmount != 0 {
		req["newSize"] = strconv.FormatFloat(newAmount, 'f', -1, 64)
	}
	if newPrice != 0 {
		req["newPrice"] = strconv.FormatFloat(newPrice, 'f', -1, 64)
	}
	path := bitgetMix + bitgetOrder + bitgetModifyOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelFuturesOrder cancels an order on the exchange
func (e *Exchange) CancelFuturesOrder(ctx context.Context, pair currency.Pair, productType, clientOrderID string, marginCoin currency.Code, orderID int64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = orderID
	}
	if !marginCoin.IsEmpty() {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetCancelOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelFuturesOrders cancels multiple orders at once
func (e *Exchange) BatchCancelFuturesOrders(ctx context.Context, orderIDs []OrderIDStruct, pair currency.Pair, productType string, marginCoin currency.Code) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"orderList":   orderIDs,
	}
	if !marginCoin.IsEmpty() {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetBatchCancelOrders
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// FlashClosePosition attempts to close a position at the best available price
func (e *Exchange) FlashClosePosition(ctx context.Context, pair currency.Pair, holdSide, productType string) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"holdSide":    holdSide,
		"productType": productType,
	}
	path := bitgetMix + bitgetOrder + bitgetClosePositions
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesOrderDetails returns details on a given order
func (e *Exchange) GetFuturesOrderDetails(ctx context.Context, pair currency.Pair, productType, clientOrderID string, orderID int64) (*FuturesOrderDetailResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	vals.Set("productType", productType)
	if clientOrderID != "" {
		vals.Set("clientOid", clientOrderID)
	}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetDetail
	var resp *FuturesOrderDetailResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetFuturesFills returns fill details
func (e *Exchange) GetFuturesFills(ctx context.Context, orderID, pagination, limit int64, pair currency.Pair, productType string, startTime, endTime time.Time) (*FuturesFillsResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("symbol", pair.String())
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + "/" + bitgetFills
	var resp FuturesFillsResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetFuturesOrderFillHistory returns historical fill details
func (e *Exchange) GetFuturesOrderFillHistory(ctx context.Context, pair currency.Pair, productType string, orderID, pagination, limit int64, startTime, endTime time.Time) (*FuturesFillsResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("productType", productType)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetFillHistory
	var resp FuturesFillsResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetPendingFuturesOrders returns detailed information on pending futures orders
func (e *Exchange) GetPendingFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, productType, status string, pair currency.Pair, startTime, endTime time.Time) (*FuturesOrdResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("productType", productType)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	params.Values.Set("clientOid", clientOrderID)
	params.Values.Set("status", status)
	params.Values.Set("symbol", pair.String())
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetOrdersPending
	var resp FuturesOrdResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetHistoricalFuturesOrders returns information on futures orders that are no longer pending
func (e *Exchange) GetHistoricalFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, productType, orderSource string, pair currency.Pair, startTime, endTime time.Time) (*HistFuturesOrdResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("productType", productType)
	params.Values.Set("orderSource", orderSource)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	params.Values.Set("clientOid", clientOrderID)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetOrdersHistory
	var resp HistFuturesOrdResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// CancelAllFuturesOrders cancels all pending orders
func (e *Exchange) CancelAllFuturesOrders(ctx context.Context, pair currency.Pair, productType string, marginCoin currency.Code, acceptableDelay time.Duration) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"productType":   productType,
		"symbol":        pair,
		"requestTime":   time.Now().UnixMilli(),
		"receiveWindow": time.Unix(0, 0).Add(acceptableDelay).UnixMilli(),
	}
	if !marginCoin.IsEmpty() {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetCancelAllOrders
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesTriggerOrderByID returns information on a particular trigger order
func (e *Exchange) GetFuturesTriggerOrderByID(ctx context.Context, planType, productType string, planOrderID int64) ([]SubOrderResp, error) {
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if planOrderID == 0 {
		return nil, errPlanOrderIDEmpty
	}
	vals := url.Values{}
	vals.Set("planType", planType)
	vals.Set("planOrderId", strconv.FormatInt(planOrderID, 10))
	vals.Set("productType", productType)
	path := bitgetMix + bitgetOrder + bitgetPlanSubOrder
	var resp []SubOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// PlaceTPSLFuturesOrder places a take-profit or stop-loss futures order
func (e *Exchange) PlaceTPSLFuturesOrder(ctx context.Context, marginCoin currency.Code, productType, planType, triggerType, holdSide, rangeRate, clientOrderID, stpMode string, pair currency.Pair, triggerPrice, executePrice, amount float64) (*OrderIDStruct, error) {
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if holdSide == "" {
		return nil, errHoldSideEmpty
	}
	if triggerPrice <= 0 {
		return nil, errTriggerPriceEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"marginCoin":   marginCoin,
		"productType":  productType,
		"symbol":       pair,
		"planType":     planType,
		"triggerType":  triggerType,
		"holdSide":     holdSide,
		"rangeRate":    rangeRate,
		"stpMode":      stpMode,
		"triggerPrice": strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"executePrice": strconv.FormatFloat(executePrice, 'f', -1, 64),
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetMix + bitgetOrder + bitgetPlaceTPSLOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceTPAndSLFuturesOrder places a take-profit and stop-loss futures order
func (e *Exchange) PlaceTPAndSLFuturesOrder(ctx context.Context, marginCoin currency.Code, productType, takeProfitTriggerType, stopLossTriggerType, holdSide, stpMode string, pair currency.Pair, takeProfitTriggerPrice, takeProfitExecutePrice, stopLossTriggerPrice, stopLossExecutePrice float64) ([]int64, error) {
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if takeProfitTriggerPrice <= 0 {
		return nil, errTakeProfitTriggerPriceEmpty
	}
	if stopLossTriggerPrice <= 0 {
		return nil, errStopLossTriggerPriceEmpty
	}
	if holdSide == "" {
		return nil, errHoldSideEmpty
	}
	req := map[string]any{
		"marginCoin":              marginCoin,
		"productType":             productType,
		"symbol":                  pair,
		"stopSurplusTriggerPrice": takeProfitTriggerPrice,
		"stopSurplusTriggerType":  takeProfitTriggerType,
		"stopSurplusExecutePrice": takeProfitExecutePrice,
		"stopLossTriggerPrice":    stopLossTriggerPrice,
		"stopLossTriggerType":     stopLossTriggerType,
		"stopLossExecutePrice":    stopLossExecutePrice,
		"holdSide":                holdSide,
		"stpMode":                 stpMode,
	}
	path := bitgetMix + bitgetOrder + bitgetPlacePOSTPSL
	var resp struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	return resp.OrderIDs, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceTriggerFuturesOrder places a trigger futures order
func (e *Exchange) PlaceTriggerFuturesOrder(ctx context.Context, planType, productType, marginMode, triggerType, side, tradeSide, orderType, clientOrderID, takeProfitTriggerType, stopLossTriggerType, stpMode string, pair currency.Pair, marginCoin currency.Code, amount, executePrice, callbackRatio, triggerPrice, takeProfitTriggerPrice, takeProfitExecutePrice, stopLossTriggerPrice, stopLossExecutePrice float64, reduceOnly bool) (*OrderIDStruct, error) {
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if triggerType == "" {
		return nil, errTriggerTypeEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if executePrice <= 0 {
		return nil, errExecutePriceEmpty
	}
	if triggerPrice <= 0 {
		return nil, errTriggerPriceEmpty
	}
	req := map[string]any{
		"planType":      planType,
		"symbol":        pair,
		"productType":   productType,
		"marginMode":    marginMode,
		"marginCoin":    marginCoin,
		"triggerType":   triggerType,
		"side":          side,
		"tradeSide":     tradeSide,
		"orderType":     orderType,
		"stpMode":       stpMode,
		"size":          strconv.FormatFloat(amount, 'f', -1, 64),
		"price":         strconv.FormatFloat(executePrice, 'f', -1, 64),
		"triggerPrice":  strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"callbackRatio": strconv.FormatFloat(callbackRatio, 'f', -1, 64),
	}
	if reduceOnly {
		req["reduceOnly"] = "YES"
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if takeProfitTriggerPrice != 0 || takeProfitExecutePrice != 0 || takeProfitTriggerType != "" {
		req["stopSurplusTriggerPrice"] = strconv.FormatFloat(takeProfitTriggerPrice, 'f', -1, 64)
		req["stopSurplusExecutePrice"] = strconv.FormatFloat(takeProfitExecutePrice, 'f', -1, 64)
		req["stopSurplusTriggerType"] = takeProfitTriggerType
	}
	if stopLossTriggerPrice != 0 || stopLossExecutePrice != 0 || stopLossTriggerType != "" {
		req["stopLossTriggerPrice"] = strconv.FormatFloat(stopLossTriggerPrice, 'f', -1, 64)
		req["stopLossExecutePrice"] = strconv.FormatFloat(stopLossExecutePrice, 'f', -1, 64)
		req["stopLossTriggerType"] = stopLossTriggerType
	}
	path := bitgetMix + bitgetOrder + bitgetPlacePlanOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// ModifyTPSLFuturesOrder modifies a take-profit or stop-loss futures order
func (e *Exchange) ModifyTPSLFuturesOrder(ctx context.Context, orderID int64, clientOrderID, productType, triggerType string, marginCoin currency.Code, pair currency.Pair, triggerPrice, executePrice, amount, rangeRate float64) (*OrderIDStruct, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if marginCoin.IsEmpty() {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if triggerPrice <= 0 {
		return nil, errTriggerPriceEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"orderId":      orderID,
		"clientOid":    clientOrderID,
		"marginCoin":   marginCoin,
		"productType":  productType,
		"symbol":       pair,
		"triggerType":  triggerType,
		"triggerPrice": strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"executePrice": strconv.FormatFloat(executePrice, 'f', -1, 64),
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if rangeRate != 0 {
		req["rangeRate"] = strconv.FormatFloat(rangeRate, 'f', -1, 64)
	}
	path := bitgetMix + bitgetOrder + bitgetModifyTPSLOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// ModifyTriggerFuturesOrder modifies a trigger futures order
func (e *Exchange) ModifyTriggerFuturesOrder(ctx context.Context, orderID int64, clientOrderID, productType, triggerType, takeProfitTriggerType, stopLossTriggerType string, amount, executePrice, callbackRatio, triggerPrice, takeProfitTriggerPrice, takeProfitExecutePrice, stopLossTriggerPrice, stopLossExecutePrice float64) (*OrderIDStruct, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		// See whether planType is accepted
		// See whether symbol is accepted
		"orderId":                   orderID,
		"clientOid":                 clientOrderID,
		"productType":               productType,
		"newTriggerType":            triggerType,
		"newSize":                   strconv.FormatFloat(amount, 'f', -1, 64),
		"newPrice":                  strconv.FormatFloat(executePrice, 'f', -1, 64),
		"newCallbackRatio":          strconv.FormatFloat(callbackRatio, 'f', -1, 64),
		"newTriggerPrice":           strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"newStopSurplusTriggerType": takeProfitTriggerType,
		"newStopLossTriggerType":    stopLossTriggerType,
	}
	if takeProfitTriggerPrice >= 0 {
		req["newStopSurplusTriggerPrice"] = strconv.FormatFloat(takeProfitTriggerPrice, 'f', -1, 64)
	}
	if takeProfitExecutePrice >= 0 {
		req["newStopSurplusExecutePrice"] = strconv.FormatFloat(takeProfitExecutePrice, 'f', -1, 64)
	}
	if stopLossTriggerPrice >= 0 {
		req["newStopLossTriggerPrice"] = strconv.FormatFloat(stopLossTriggerPrice, 'f', -1, 64)
	}
	if stopLossExecutePrice >= 0 {
		req["newStopLossExecutePrice"] = strconv.FormatFloat(stopLossExecutePrice, 'f', -1, 64)
	}
	path := bitgetMix + bitgetOrder + bitgetModifyPlanOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetPendingTriggerFuturesOrders returns information on pending trigger orders
func (e *Exchange) GetPendingTriggerFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, planType, productType string, pair currency.Pair, startTime, endTime time.Time) (*PlanFuturesOrdResp, error) {
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("planType", planType)
	params.Values.Set("productType", productType)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	params.Values.Set("clientOid", clientOrderID)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetOrdersPlanPending
	var resp PlanFuturesOrdResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// CancelTriggerFuturesOrders cancels trigger futures orders
func (e *Exchange) CancelTriggerFuturesOrders(ctx context.Context, orderIDList []OrderIDStruct, pair currency.Pair, productType, planType string, marginCoin currency.Code) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"productType": productType,
		"planType":    planType,
		"symbol":      pair,
		"orderIdList": orderIDList,
		"marginCoin":  marginCoin,
	}
	path := bitgetMix + bitgetOrder + bitgetCancelPlanOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetHistoricalTriggerFuturesOrders returns information on historical trigger orders
func (e *Exchange) GetHistoricalTriggerFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, planType, planStatus, productType string, pair currency.Pair, startTime, endTime time.Time) (*HistTriggerFuturesOrdResp, error) {
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("planType", planType)
	params.Values.Set("productType", productType)
	params.Values.Set("planStatus", planStatus)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	params.Values.Set("clientOid", clientOrderID)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetOrdersPlanHistory
	var resp HistTriggerFuturesOrdResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSupportedCurrencies returns information on the currencies supported by the exchange
func (e *Exchange) GetSupportedCurrencies(ctx context.Context) ([]SupCurrencyResp, error) {
	path := bitgetMargin + bitgetCurrencies
	var resp []SupCurrencyResp
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetCrossBorrowHistory returns the borrowing history for cross margin
func (e *Exchange) GetCrossBorrowHistory(ctx context.Context, loanID, limit, pagination int64, currency currency.Code, startTime, endTime time.Time) (*BorrowHistCross, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if loanID != 0 {
		params.Values.Set("loanId", strconv.FormatInt(loanID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	path := bitgetMargin + bitgetCrossed + bitgetBorrowHistory
	var resp BorrowHistCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossRepayHistory returns the repayment history for cross margin
func (e *Exchange) GetCrossRepayHistory(ctx context.Context, repayID, limit, pagination int64, currency currency.Code, startTime, endTime time.Time) (*CrossRepayHistResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if repayID != 0 {
		params.Values.Set("repayId", strconv.FormatInt(repayID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	path := bitgetMargin + bitgetCrossed + bitgetRepayHistory
	var resp CrossRepayHistResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossInterestHistory returns the interest history for cross margin
func (e *Exchange) GetCrossInterestHistory(ctx context.Context, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) (*InterHistCross, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	path := bitgetMargin + bitgetCrossed + bitgetInterestHistory
	var resp InterHistCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossLiquidationHistory returns the liquidation history for cross margin
func (e *Exchange) GetCrossLiquidationHistory(ctx context.Context, startTime, endTime time.Time, limit, pagination int64) (*LiquidHistCross, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetCrossed + bitgetLiquidationHistory
	var resp LiquidHistCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossFinancialHistory returns the financial history for cross margin
func (e *Exchange) GetCrossFinancialHistory(ctx context.Context, marginType string, currency currency.Code, startTime, endTime time.Time, limit, pagination int64) (*FinHistCrossResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("marginType", marginType)
	if !currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	path := bitgetMargin + bitgetCrossed + bitgetFinancialRecords
	var resp FinHistCrossResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossAccountAssets returns the account assets for cross margin
func (e *Exchange) GetCrossAccountAssets(ctx context.Context, currency currency.Code) ([]CrossAssetResp, error) {
	vals := url.Values{}
	if !currency.IsEmpty() {
		vals.Set("coin", currency.String())
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetAssets
	var resp []CrossAssetResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CrossBorrow borrows funds for cross margin
func (e *Exchange) CrossBorrow(ctx context.Context, currency currency.Code, clientOrderID string, amount float64) (*BorrowCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"coin":         currency,
		"borrowAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid":    clientOrderID,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetBorrow
	var resp BorrowCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CrossRepay repays funds for cross margin
func (e *Exchange) CrossRepay(ctx context.Context, currency currency.Code, amount float64) (*RepayCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"coin":        currency,
		"repayAmount": strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetRepay
	var resp RepayCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossRiskRate returns the risk rate for cross margin
func (e *Exchange) GetCrossRiskRate(ctx context.Context) (float64, error) {
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetRiskRate
	var resp struct {
		RiskRateRatio float64 `json:"riskRateRatio,string"`
	}
	return resp.RiskRateRatio, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetCrossMaxBorrowable returns the maximum amount that can be borrowed for cross margin
func (e *Exchange) GetCrossMaxBorrowable(ctx context.Context, currency currency.Code) (*MaxBorrowCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetMaxBorrowableAmount
	var resp MaxBorrowCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossMaxTransferable returns the maximum amount that can be transferred out of cross margin
func (e *Exchange) GetCrossMaxTransferable(ctx context.Context, currency currency.Code) (*MaxTransferCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetMaxTransferOutAmount
	var resp MaxTransferCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossInterestRateAndMaxBorrowable returns the interest rate and maximum borrowable amount for cross margin
func (e *Exchange) GetCrossInterestRateAndMaxBorrowable(ctx context.Context, currency currency.Code) ([]IntRateMaxBorrowCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetMargin + bitgetCrossed + bitgetInterestRateAndLimit
	var resp []IntRateMaxBorrowCross
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossTierConfiguration returns tier information for the user's VIP level
func (e *Exchange) GetCrossTierConfiguration(ctx context.Context, currency currency.Code) ([]TierConfigCross, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetMargin + bitgetCrossed + bitgetTierData
	var resp []TierConfigCross
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CrossFlashRepay repays funds for cross margin, with the option to only repay for a particular currency
func (e *Exchange) CrossFlashRepay(ctx context.Context, currency currency.Code) (*FlashRepayCross, error) {
	req := map[string]any{
		"coin": currency,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetFlashRepay
	var resp FlashRepayCross
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossFlashRepayResult returns the result of the supplied flash repayments for cross margin
func (e *Exchange) GetCrossFlashRepayResult(ctx context.Context, idList []int64) ([]FlashRepayResult, error) {
	if len(idList) == 0 {
		return nil, errIDListEmpty
	}
	req := map[string]any{
		"repayIdList": idList,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetQueryFlashRepayStatus
	var resp []FlashRepayResult
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceCrossOrder places an order using cross margin
func (e *Exchange) PlaceCrossOrder(ctx context.Context, pair currency.Pair, orderType, loanType, strategy, clientOrderID, side, stpMode string, price, baseAmount, quoteAmount float64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if loanType == "" {
		return nil, errLoanTypeEmpty
	}
	if strategy == "" {
		return nil, errStrategyEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if baseAmount <= 0 && quoteAmount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"symbol":    pair,
		"orderType": orderType,
		"loanType":  loanType,
		"force":     strategy,
		"clientOid": clientOrderID,
		"side":      side,
		"stpMode":   stpMode,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
	}
	if baseAmount != 0 {
		req["baseSize"] = strconv.FormatFloat(baseAmount, 'f', -1, 64)
	}
	if quoteAmount != 0 {
		req["quoteSize"] = strconv.FormatFloat(quoteAmount, 'f', -1, 64)
	}
	path := bitgetMargin + bitgetCrossed + bitgetPlaceOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceCrossOrders places multiple orders using cross margin
func (e *Exchange) BatchPlaceCrossOrders(ctx context.Context, pair currency.Pair, orders []MarginOrderData) (*BatchOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetMargin + bitgetCrossed + bitgetBatchPlaceOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelCrossOrder cancels an order using cross margin
func (e *Exchange) CancelCrossOrder(ctx context.Context, pair currency.Pair, clientOrderID string, orderID int64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if (clientOrderID == "" && orderID == 0) || (clientOrderID != "" && orderID != 0) {
		return nil, errOrderIDMutex
	}
	req := map[string]any{
		"symbol": pair,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	path := bitgetMargin + bitgetCrossed + bitgetCancelOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelCrossOrders cancels multiple orders using cross margin
func (e *Exchange) BatchCancelCrossOrders(ctx context.Context, pair currency.Pair, orders []OrderIDStruct) (*BatchOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orders) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"symbol":      pair,
		"orderIdList": orders,
	}
	path := bitgetMargin + bitgetCrossed + bitgetBatchCancelOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossOpenOrders returns the open orders for cross margin
func (e *Exchange) GetCrossOpenOrders(ctx context.Context, pair currency.Pair, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOrders, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("clientOid", clientOrderID)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetCrossed + bitgetOpenOrders
	var resp MarginOrders
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossHistoricalOrders returns the historical orders for cross margin
func (e *Exchange) GetCrossHistoricalOrders(ctx context.Context, pair currency.Pair, enterPointSource, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOrders, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("enterPointSource", enterPointSource)
	params.Values.Set("clientOid", clientOrderID)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetCrossed + bitgetHistoryOrders
	var resp MarginOrders
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossOrderFills returns the fills for cross margin orders
func (e *Exchange) GetCrossOrderFills(ctx context.Context, pair currency.Pair, orderID, pagination, limit int64, startTime, endTime time.Time) (*MarginOrderFills, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetFills
	var resp MarginOrderFills
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossLiquidationOrders returns the liquidation orders for cross margin
func (e *Exchange) GetCrossLiquidationOrders(ctx context.Context, orderType, fromCoin, toCoin string, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) (*LiquidationResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Values.Set("type", orderType)
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("fromCoin", fromCoin)
	params.Values.Set("toCoin", toCoin)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetCrossed + bitgetLiquidationOrder
	var resp LiquidationResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedRepayHistory returns the repayment history for isolated margin
func (e *Exchange) GetIsolatedRepayHistory(ctx context.Context, pair currency.Pair, cur currency.Code, repayID, limit, pagination int64, startTime, endTime time.Time) (*IsoRepayHistResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if repayID != 0 {
		params.Values.Set("repayId", strconv.FormatInt(repayID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("symbol", pair.String())
	if !cur.IsEmpty() {
		params.Values.Set("coin", cur.String())
	}
	path := bitgetMargin + bitgetIsolated + bitgetRepayHistory
	var resp IsoRepayHistResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedBorrowHistory returns the borrowing history for isolated margin
func (e *Exchange) GetIsolatedBorrowHistory(ctx context.Context, pair currency.Pair, cur currency.Code, loanID, limit, pagination int64, startTime, endTime time.Time) (*BorrowHistIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("loanId", strconv.FormatInt(loanID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("coin", cur.String())
	path := bitgetMargin + bitgetIsolated + bitgetBorrowHistory
	var resp BorrowHistIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedInterestHistory returns the interest history for isolated margin
func (e *Exchange) GetIsolatedInterestHistory(ctx context.Context, pair currency.Pair, cur currency.Code, startTime, endTime time.Time, limit, pagination int64) (*InterHistIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("symbol", pair.String())
	if !cur.IsEmpty() {
		params.Values.Set("coin", cur.String())
	}
	path := bitgetMargin + bitgetIsolated + bitgetInterestHistory
	var resp InterHistIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedLiquidationHistory returns the liquidation history for isolated margin
func (e *Exchange) GetIsolatedLiquidationHistory(ctx context.Context, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) (*LiquidHistIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + bitgetLiquidationHistory
	var resp LiquidHistIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedFinancialHistory returns the financial history for isolated margin
func (e *Exchange) GetIsolatedFinancialHistory(ctx context.Context, pair currency.Pair, marginType string, cur currency.Code, startTime, endTime time.Time, limit, pagination int64) (*FinHistIsoResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("marginType", marginType)
	if !cur.IsEmpty() {
		params.Values.Set("coin", cur.String())
	}
	path := bitgetMargin + bitgetIsolated + bitgetFinancialRecords
	var resp FinHistIsoResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedAccountAssets returns the account assets for isolated margin
func (e *Exchange) GetIsolatedAccountAssets(ctx context.Context, pair currency.Pair) ([]IsoAssetResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetAssets
	var resp []IsoAssetResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// IsolatedBorrow borrows funds for isolated margin
func (e *Exchange) IsolatedBorrow(ctx context.Context, pair currency.Pair, cur currency.Code, clientOrderID string, amount float64) (*BorrowIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if cur.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"symbol":       pair,
		"coin":         cur,
		"borrowAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid":    clientOrderID,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetBorrow
	var resp BorrowIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// IsolatedRepay repays funds for isolated margin
func (e *Exchange) IsolatedRepay(ctx context.Context, amount float64, cur currency.Code, pair currency.Pair, clientOrderID string) (*RepayIso, error) {
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if cur.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	req := map[string]any{
		"coin":        cur,
		"repayAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"symbol":      pair,
		"clientOid":   clientOrderID,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetRepay
	var resp RepayIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedRiskRate returns the risk rate for isolated margin
func (e *Exchange) GetIsolatedRiskRate(ctx context.Context, pair currency.Pair, pagination, limit int64) ([]RiskRateIso, error) {
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	if limit != 0 {
		vals.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("pageNum", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetRiskRate
	var resp []RiskRateIso
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedInterestRateAndMaxBorrowable returns the interest rate and maximum borrowable amount for isolated margin
func (e *Exchange) GetIsolatedInterestRateAndMaxBorrowable(ctx context.Context, pair currency.Pair) ([]IntRateMaxBorrowIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + bitgetInterestRateAndLimit
	var resp []IntRateMaxBorrowIso
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedTierConfiguration returns tier information for the user's VIP level
func (e *Exchange) GetIsolatedTierConfiguration(ctx context.Context, pair currency.Pair) ([]TierConfigIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + bitgetTierData
	var resp []TierConfigIso
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedMaxBorrowable returns the maximum amount that can be borrowed for isolated margin
func (e *Exchange) GetIsolatedMaxBorrowable(ctx context.Context, pair currency.Pair) (*MaxBorrowIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetMaxBorrowableAmount
	var resp MaxBorrowIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedMaxTransferable returns the maximum amount that can be transferred out of isolated margin
func (e *Exchange) GetIsolatedMaxTransferable(ctx context.Context, pair currency.Pair) (*MaxTransferIso, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair.String())
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetMaxTransferOutAmount
	var resp MaxTransferIso
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// IsolatedFlashRepay repays funds for isolated margin, with the option to only repay for a set of up to 100 pairs
func (e *Exchange) IsolatedFlashRepay(ctx context.Context, pairs currency.Pairs) ([]FlashRepayIso, error) {
	req := map[string]any{
		"symbolList": pairs.Strings(),
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetFlashRepay
	var resp []FlashRepayIso
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedFlashRepayResult returns the result of the supplied flash repayments for isolated margin
func (e *Exchange) GetIsolatedFlashRepayResult(ctx context.Context, idList []int64) ([]FlashRepayResult, error) {
	if len(idList) == 0 {
		return nil, errIDListEmpty
	}
	req := map[string]any{
		"repayIdList": idList,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetQueryFlashRepayStatus
	var resp []FlashRepayResult
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceIsolatedOrder places an order using isolated margin
func (e *Exchange) PlaceIsolatedOrder(ctx context.Context, pair currency.Pair, orderType, loanType, strategy, clientOrderID, side, stpMode string, price, baseAmount, quoteAmount float64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if loanType == "" {
		return nil, errLoanTypeEmpty
	}
	if strategy == "" {
		return nil, errStrategyEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if baseAmount <= 0 && quoteAmount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"symbol":    pair,
		"orderType": orderType,
		"loanType":  loanType,
		"force":     strategy,
		"clientOid": clientOrderID,
		"side":      side,
		"stpMode":   stpMode,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
	}
	if baseAmount != 0 {
		req["baseSize"] = strconv.FormatFloat(baseAmount, 'f', -1, 64)
	}
	if quoteAmount != 0 {
		req["quoteSize"] = strconv.FormatFloat(quoteAmount, 'f', -1, 64)
	}
	path := bitgetMargin + bitgetIsolated + bitgetPlaceOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceIsolatedOrders places multiple orders using isolated margin
func (e *Exchange) BatchPlaceIsolatedOrders(ctx context.Context, pair currency.Pair, orders []MarginOrderData) (*BatchOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orders) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetMargin + bitgetIsolated + bitgetBatchPlaceOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelIsolatedOrder cancels an order using isolated margin
func (e *Exchange) CancelIsolatedOrder(ctx context.Context, pair currency.Pair, clientOrderID string, orderID int64) (*OrderIDStruct, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if (clientOrderID == "" && orderID == 0) || (clientOrderID != "" && orderID != 0) {
		return nil, errOrderIDMutex
	}
	req := map[string]any{
		"symbol": pair,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = orderID
	}
	path := bitgetMargin + bitgetIsolated + bitgetCancelOrder
	var resp OrderIDStruct
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelIsolatedOrders cancels multiple orders using isolated margin
func (e *Exchange) BatchCancelIsolatedOrders(ctx context.Context, pair currency.Pair, orders []OrderIDStruct) (*BatchOrderResp, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"orderIdList": orders,
	}
	path := bitgetMargin + bitgetIsolated + bitgetBatchCancelOrder
	var resp *BatchOrderResp
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedOpenOrders returns the open orders for isolated margin
func (e *Exchange) GetIsolatedOpenOrders(ctx context.Context, pair currency.Pair, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOrders, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("clientOid", clientOrderID)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + bitgetOpenOrders
	var resp MarginOrders
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedHistoricalOrders returns the historical orders for isolated margin
func (e *Exchange) GetIsolatedHistoricalOrders(ctx context.Context, pair currency.Pair, enterPointSource, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOrders, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("enterPointSource", enterPointSource)
	params.Values.Set("clientOid", clientOrderID)
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + bitgetHistoryOrders
	var resp MarginOrders
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedOrderFills returns the fills for isolated margin orders
func (e *Exchange) GetIsolatedOrderFills(ctx context.Context, pair currency.Pair, orderID, pagination, limit int64, startTime, endTime time.Time) (*MarginOrderFills, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair.String())
	if orderID != 0 {
		params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetFills
	var resp MarginOrderFills
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedLiquidationOrders returns the liquidation orders for isolated margin
func (e *Exchange) GetIsolatedLiquidationOrders(ctx context.Context, orderType, fromCoin, toCoin string, pair currency.Pair, startTime, endTime time.Time, limit, pagination int64) (*LiquidationResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Values.Set("type", orderType)
	}
	params.Values.Set("symbol", pair.String())
	params.Values.Set("fromCoin", fromCoin)
	params.Values.Set("toCoin", toCoin)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + bitgetLiquidationOrder
	var resp LiquidationResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsProductList returns the list of savings products for a particular currency
func (e *Exchange) GetSavingsProductList(ctx context.Context, currency currency.Code, filter string) ([]SavingsProductList, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	vals.Set("filter", filter)
	path := bitgetEarn + bitgetSavings + bitgetProduct
	var resp []SavingsProductList
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSavingsBalance returns the savings balance and amount earned in BTC and USDT
func (e *Exchange) GetSavingsBalance(ctx context.Context) (*SavingsBalance, error) {
	path := bitgetEarn + bitgetSavings + "/" + bitgetAccount
	var resp SavingsBalance
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetSavingsAssets returns information on assets held over the last three months
func (e *Exchange) GetSavingsAssets(ctx context.Context, periodType string, startTime, endTime time.Time, limit, pagination int64) (*SavingsAssetsResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetAssets
	var resp SavingsAssetsResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsRecords returns information on transactions performed over the last three months
func (e *Exchange) GetSavingsRecords(ctx context.Context, currency currency.Code, periodType, orderType string, startTime, endTime time.Time, limit, pagination int64) (*SavingsRecords, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("periodType", periodType)
	params.Values.Set("orderType", orderType)
	path := bitgetEarn + bitgetSavings + bitgetRecords
	var resp SavingsRecords
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsSubscriptionDetail returns detailed information on subscribing, for a single product
func (e *Exchange) GetSavingsSubscriptionDetail(ctx context.Context, productID int64, periodType string) (*SavingsSubDetail, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productId", strconv.FormatInt(productID, 10))
	vals.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetSubscribeInfo
	var resp SavingsSubDetail
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubscribeSavings applies funds to a savings product
func (e *Exchange) SubscribeSavings(ctx context.Context, productID int64, periodType string, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"productId":  productID,
		"periodType": periodType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSavings + bitgetSubscribe
	var resp SaveResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSavingsSubscriptionResult returns the result of a subscription attempt
func (e *Exchange) GetSavingsSubscriptionResult(ctx context.Context, productID int64, periodType string) (*SaveResult, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productId", strconv.FormatInt(productID, 10))
	vals.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetSubscribeResult
	var resp SaveResult
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// RedeemSavings redeems funds from a savings product
func (e *Exchange) RedeemSavings(ctx context.Context, productID, orderID int64, periodType string, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"productId":  productID,
		"orderId":    orderID,
		"periodType": periodType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSavings + bitgetRedeem
	var resp SaveResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSavingsRedemptionResult returns the result of a redemption attempt
func (e *Exchange) GetSavingsRedemptionResult(ctx context.Context, orderID int64, periodType string) (*SaveResult, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	vals := url.Values{}
	vals.Set("orderId", strconv.FormatInt(orderID, 10))
	vals.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetRedeemResult
	var resp SaveResult
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetEarnAccountAssets returns the assets in the earn account
func (e *Exchange) GetEarnAccountAssets(ctx context.Context, currency currency.Code) ([]EarnAssets, error) {
	vals := url.Values{}
	if currency.IsEmpty() {
		vals.Set("coin", currency.String())
	}
	path := bitgetEarn + bitgetAccount + bitgetAssets
	var resp []EarnAssets
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSharkFinProducts returns information on Shark Fin products
func (e *Exchange) GetSharkFinProducts(ctx context.Context, currency currency.Code, limit, pagination int64) (*SharkFinProductResp, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetEarn + bitgetSharkFin + bitgetProduct
	var resp SharkFinProductResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSharkFinBalance returns the balance and amount earned in BTC and USDT for Shark Fin products
func (e *Exchange) GetSharkFinBalance(ctx context.Context) (*SharkFinBalance, error) {
	path := bitgetEarn + bitgetSharkFin + "/" + bitgetAccount
	var resp SharkFinBalance
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetSharkFinAssets returns information on assets held over the last three months for Shark Fin products
func (e *Exchange) GetSharkFinAssets(ctx context.Context, status string, startTime, endTime time.Time, limit, pagination int64) (*SharkFinAssetsResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("status", status)
	path := bitgetEarn + bitgetSharkFin + bitgetAssets
	var resp SharkFinAssetsResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSharkFinRecords returns information on transactions performed over the last three months for Shark Fin products
func (e *Exchange) GetSharkFinRecords(ctx context.Context, currency currency.Code, transactionType string, startTime, endTime time.Time, limit, pagination int64) ([]SharkFinRecords, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if currency.IsEmpty() {
		params.Values.Set("coin", currency.String())
	}
	params.Values.Set("type", transactionType)
	path := bitgetEarn + bitgetSharkFin + bitgetRecords
	var resp struct {
		ResultList []SharkFinRecords `json:"resultList"`
	}
	return resp.ResultList, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSharkFinSubscriptionDetail returns detailed information on subscribing, for a single product
func (e *Exchange) GetSharkFinSubscriptionDetail(ctx context.Context, productID int64) (*SharkFinSubDetail, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("productId", strconv.FormatInt(productID, 10))
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribeInfo
	var resp SharkFinSubDetail
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubscribeSharkFin applies funds to a Shark Fin product
func (e *Exchange) SubscribeSharkFin(ctx context.Context, productID int64, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"productId": productID,
		"amount":    strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribe
	var resp SaveResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSharkFinSubscriptionResult returns the result of a subscription attempt
func (e *Exchange) GetSharkFinSubscriptionResult(ctx context.Context, orderID int64) (*SaveResult, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	vals := url.Values{}
	vals.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribeResult
	var resp SaveResult
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetLoanCurrencyList returns the list of currencies available for loan
func (e *Exchange) GetLoanCurrencyList(ctx context.Context, currency currency.Code) (*LoanCurList, error) {
	if currency.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency.String())
	path := bitgetEarn + bitgetLoan + "/" + bitgetPublic + bitgetCoinInfos
	var resp LoanCurList
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetEstimatedInterestAndBorrowable returns the estimated interest and borrowable amount for a currency
func (e *Exchange) GetEstimatedInterestAndBorrowable(ctx context.Context, loanCoin, collateralCoin currency.Code, term string, collateralAmount float64) (*EstimateInterest, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinEmpty
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinEmpty
	}
	if term == "" {
		return nil, errTermEmpty
	}
	if collateralAmount <= 0 {
		return nil, errCollateralAmountEmpty
	}
	vals := url.Values{}
	vals.Set("loanCoin", loanCoin.String())
	vals.Set("pledgeCoin", collateralCoin.String())
	vals.Set("daily", term)
	vals.Set("pledgeAmount", strconv.FormatFloat(collateralAmount, 'f', -1, 64))
	path := bitgetEarn + bitgetLoan + "/" + bitgetPublic + bitgetHourInterest
	var resp EstimateInterest
	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// BorrowFunds borrows funds for a currency, supplying a certain amount of currency as collateral
func (e *Exchange) BorrowFunds(ctx context.Context, loanCoin, collateralCoin currency.Code, term string, collateralAmount, loanAmount float64) (int64, error) {
	if loanCoin.IsEmpty() {
		return 0, errLoanCoinEmpty
	}
	if collateralCoin.IsEmpty() {
		return 0, errCollateralCoinEmpty
	}
	if term == "" {
		return 0, errTermEmpty
	}
	if (collateralAmount <= 0 && loanAmount <= 0) || (collateralAmount != 0 && loanAmount != 0) {
		return 0, errCollateralLoanMutex
	}
	req := map[string]any{
		"loanCoin":     loanCoin,
		"pledgeCoin":   collateralCoin,
		"daily":        term,
		"pledgeAmount": strconv.FormatFloat(collateralAmount, 'f', -1, 64),
		"loanAmount":   strconv.FormatFloat(loanAmount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetLoan + bitgetBorrow
	var resp struct {
		OrderID int64 `json:"orderId"`
	}
	return resp.OrderID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetOngoingLoans returns the ongoing loans, optionally filtered by currency
func (e *Exchange) GetOngoingLoans(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code) ([]OngoingLoans, error) {
	vals := url.Values{}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	vals.Set("loanCoin", loanCoin.String())
	vals.Set("pledgeCoin", collateralCoin.String())
	path := bitgetEarn + bitgetLoan + bitgetOngoingOrders
	var resp []OngoingLoans
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// RepayLoan repays a loan
func (e *Exchange) RepayLoan(ctx context.Context, orderID int64, amount float64, repayUnlock, repayAll bool) (*RepayResp, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 && !repayAll {
		return nil, order.ErrAmountBelowMin
	}
	req := map[string]any{
		"orderId": orderID,
		"amount":  strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if repayUnlock {
		req["repayUnlock"] = "yes"
	} else {
		req["repayUnlock"] = "no"
	}
	if repayAll {
		req["repayAll"] = "yes"
	} else {
		req["repayAll"] = "no"
	}
	path := bitgetEarn + bitgetLoan + bitgetRepay
	var resp RepayResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetLoanRepayHistory returns the repayment records for a loan
func (e *Exchange) GetLoanRepayHistory(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin currency.Code, startTime, endTime time.Time) ([]RepayRecords, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin.String())
	params.Values.Set("pledgeCoin", pledgeCoin.String())
	if pagination != 0 {
		params.Values.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetRepayHistory
	var resp []RepayRecords
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// ModifyPledgeRate modifies the amount of collateral pledged for a loan
func (e *Exchange) ModifyPledgeRate(ctx context.Context, orderID int64, amount float64, pledgeCoin currency.Code, reviseType string) (*ModPledgeResp, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if pledgeCoin.IsEmpty() {
		return nil, errCollateralCoinEmpty
	}
	if reviseType == "" {
		return nil, errReviseTypeEmpty
	}
	req := map[string]any{
		"orderId":    orderID,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
		"pledgeCoin": pledgeCoin,
		"reviseType": reviseType,
	}
	path := bitgetEarn + bitgetLoan + bitgetRevisePledge
	var resp ModPledgeResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetPledgeRateHistory returns the history of pledged rates for loans
func (e *Exchange) GetPledgeRateHistory(ctx context.Context, orderID, pagination, limit int64, reviseSide string, pledgeCoin currency.Code, startTime, endTime time.Time) ([]PledgeRateHist, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("reviseSide", reviseSide)
	params.Values.Set("pledgeCoin", pledgeCoin.String())
	if pagination != 0 {
		params.Values.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetReviseHistory
	var resp []PledgeRateHist
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetLoanHistory returns the loan history
func (e *Exchange) GetLoanHistory(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin currency.Code, status string, startTime, endTime time.Time) ([]LoanHistory, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin.String())
	params.Values.Set("pledgeCoin", pledgeCoin.String())
	params.Values.Set("status", status)
	if pagination != 0 {
		params.Values.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetBorrowHistory
	var resp []LoanHistory
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetDebts returns information on current outstanding pledges and loans
func (e *Exchange) GetDebts(ctx context.Context) (*DebtsResp, error) {
	path := bitgetEarn + bitgetLoan + bitgetDebts
	var resp DebtsResp
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetLiquidationRecords returns the liquidation records
func (e *Exchange) GetLiquidationRecords(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin currency.Code, status string, startTime, endTime time.Time) ([]LiquidRecs, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin.String())
	params.Values.Set("pledgeCoin", pledgeCoin.String())
	params.Values.Set("status", status)
	if pagination != 0 {
		params.Values.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetReduces
	var resp []LiquidRecs
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetLoanInfo returns information on an offered institutional loan
func (e *Exchange) GetLoanInfo(ctx context.Context, productID string) (*LoanInfo, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("productId", productID)
	path := bitgetSpot + bitgetInsLoan + bitgetProductInfos
	var resp LoanInfo
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetMarginCoinRatio returns the conversion rate various margin coins have for a particular loan
func (e *Exchange) GetMarginCoinRatio(ctx context.Context, productID string) (*MarginCoinRatio, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("productId", productID)
	path := bitgetSpot + bitgetInsLoan + bitgetEnsureCoinsConvert
	var resp MarginCoinRatio
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetSpotSymbols returns spot trading pairs that meet ======A CERTAIN CRITERIA CURRENTLY UNCLEAR TO ME======
func (e *Exchange) GetSpotSymbols(ctx context.Context, productID string) (*SpotSymbols, error) {
	if productID == "" {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("productId", productID)
	path := bitgetSpot + bitgetInsLoan + bitgetSymbols
	var resp SpotSymbols
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetLoanToValue returns the loan to value ratio for all loans on the user's account
func (e *Exchange) GetLoanToValue(ctx context.Context, riskUnitID string) (*LoanToValue, error) {
	vals := url.Values{}
	vals.Set("riskUnitId", riskUnitID)
	path := bitgetSpot + bitgetInsLoan + bitgetLTVConvert
	var resp LoanToValue
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetTransferableAmount returns the amount of a currency that can be transferred
func (e *Exchange) GetTransferableAmount(ctx context.Context, accountID string, coin currency.Code) (*TransferableAmount, error) {
	if coin.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("userId", accountID)
	vals.Set("coin", coin.String())
	path := bitgetSpot + bitgetInsLoan + bitgetTransferred
	var resp TransferableAmount
	return &resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetRiskUnit returns the IDs for all risk unit accounts
func (e *Exchange) GetRiskUnit(ctx context.Context) ([]string, error) {
	path := bitgetSpot + bitgetInsLoan + bitgetRiskUnit
	var resp struct {
		RiskUnitID []string `json:"riskUnitId"`
	}
	return resp.RiskUnitID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, nil, nil, &resp)
}

// SubaccountRiskUnitBinding binds or unbinds the provided subaccount and risk unit
func (e *Exchange) SubaccountRiskUnitBinding(ctx context.Context, subaccountID, riskUnitID string, bind bool) ([]string, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	req := map[string]any{
		"uid":        subaccountID,
		"riskUnitId": riskUnitID,
	}
	if bind {
		req["operate"] = "bind"
	} else {
		req["operate"] = "unbind"
	}
	path := bitgetSpot + bitgetInsLoan + bitgetBindUID
	var resp struct {
		RiskUnitID []string `json:"riskUnitId"`
	}
	return resp.RiskUnitID, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetLoanOrders returns a list of loan orders taken out on the user's account
func (e *Exchange) GetLoanOrders(ctx context.Context, orderID string, startTime, endTime time.Time) ([]LoanOrders, error) {
	var params Params
	params.Values = make(url.Values)
	params.Values.Set("orderId", orderID)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	path := bitgetSpot + bitgetInsLoan + bitgetLoanOrder
	var resp []LoanOrders
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, params.Values, nil, &resp)
}

// GetRepaymentOrders returns a list of repayment orders taken out on the user's account
func (e *Exchange) GetRepaymentOrders(ctx context.Context, limit int64, startTime, endTime time.Time) ([]RepaymentOrders, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetInsLoan + bitgetRepaidHistory
	var resp []RepaymentOrders
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, params.Values, nil, &resp)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, method, path string, queryParams url.Values, bodyParams map[string]any, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = common.EncodeURLValues(path, queryParams)
	newRequest := func() (*request.Item, error) {
		payload := []byte("")
		if bodyParams != nil {
			payload, err = json.Marshal(bodyParams)
			if err != nil {
				return nil, err
			}
		}
		// $ gets escaped in URLs, but the exchange reverses this before checking the signature; if we don't reverse it ourselves, they'll consider it invalid. This technically applies to other escape characters too, but $ is one we need to worry about, since it's included in some currencies supported by the exchange
		unescapedPath := strings.ReplaceAll(path, "%24", "$")
		t := strconv.FormatInt(time.Now().UnixMilli(), 10)
		message := t + method + "/api/v2/" + unescapedPath + string(payload)
		// The exchange also supports user-generated RSA keys, but we haven't implemented that yet
		var hmac []byte
		hmac, err = crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["ACCESS-KEY"] = creds.Key
		headers["ACCESS-SIGN"] = base64.StdEncoding.EncodeToString(hmac)
		headers["ACCESS-TIMESTAMP"] = t
		headers["ACCESS-PASSPHRASE"] = creds.ClientID
		headers["Content-Type"] = "application/json"
		headers["locale"] = "en-US"
		return &request.Item{
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &RespWrapper{Data: &result},
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}, nil
	}
	return e.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)
}

// SendHTTPRequest sends an unauthenticated HTTP request, with a few assumptions about the request; namely that it is a GET request with no body
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, path string, queryParams url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	path = common.EncodeURLValues(path, queryParams)
	newRequest := func() (*request.Item, error) {
		return &request.Item{
			Method:        "GET",
			Path:          endpoint + path,
			Result:        &RespWrapper{Data: &result},
			Verbose:       e.Verbose,
			HTTPDebugging: e.HTTPDebugging,
			HTTPRecording: e.HTTPRecording,
		}, nil
	}
	return e.SendPayload(ctx, rateLim, newRequest, request.UnauthenticatedRequest)
}

func (p *Params) prepareDateString(startDate, endDate time.Time, ignoreUnsetStart, ignoreUnsetEnd bool) error {
	if startDate.After(endDate) && !endDate.IsZero() && !endDate.Equal(common.ZeroValueUnix) {
		return common.ErrStartAfterEnd
	}
	if startDate.Equal(endDate) && !startDate.IsZero() && !startDate.Equal(common.ZeroValueUnix) {
		return common.ErrStartEqualsEnd
	}
	if startDate.After(time.Now()) {
		return common.ErrStartAfterTimeNow
	}
	if startDate.IsZero() || startDate.Equal(common.ZeroValueUnix) {
		if !ignoreUnsetStart {
			return fmt.Errorf("start %w", common.ErrDateUnset)
		}
	} else {
		p.Values.Set("startTime", strconv.FormatInt(startDate.UnixMilli(), 10))
	}
	if endDate.IsZero() || endDate.Equal(common.ZeroValueUnix) {
		if !ignoreUnsetEnd {
			return fmt.Errorf("end %w", common.ErrDateUnset)
		}
	} else {
		p.Values.Set("endTime", strconv.FormatInt(endDate.UnixMilli(), 10))
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON input into a YesNoBool type
func (y *YesNoBool) UnmarshalJSON(b []byte) error {
	var yn string
	err := json.Unmarshal(b, &yn)
	if err != nil {
		return err
	}
	switch yn {
	case "yes":
		*y = true
	case "no":
		*y = false
	}
	return nil
}

// MarshalJSON marshals the YesNoBool type into a JSON string
func (y YesNoBool) MarshalJSON() ([]byte, error) {
	if y {
		return json.Marshal("YES")
	}
	return json.Marshal("NO")
}

// UnmarshalJSON unmarshals the JSON input into a SuccessBool type
func (s *SuccessBool) UnmarshalJSON(b []byte) error {
	var success string
	_, typ, _, err := jsonparser.Get(b)
	if err != nil {
		return err
	}
	if typ == jsonparser.String {
		err = json.Unmarshal(b, &success)
	} else {
		// Hack fix, replace if a better one is found
		// Can't use reflect to grab the underlying json tag, as you need the struct this SuccessBool field is a part of for that
		success, err = jsonparser.GetString(b, "data")
		if err == jsonparser.KeyPathNotFoundError {
			success, err = jsonparser.GetString(b, "result")
		}
	}
	if err != nil {
		return err
	}
	switch success {
	case "success":
		*s = true
	case "failure", "fail":
		*s = false
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON input into an EmptyInt type
func (e *EmptyInt) UnmarshalJSON(b []byte) error {
	var num string
	err := json.Unmarshal(b, &num)
	if err != nil {
		return err
	}
	if num == "" {
		*e = 0
		return nil
	}
	i, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return err
	}
	*e = EmptyInt(i)
	return nil
}

// MarshalJSON marshals the EmptyInt type into a JSON string
func (e *EmptyInt) MarshalJSON() ([]byte, error) {
	if *e == 0 {
		return json.Marshal("")
	}
	return json.Marshal(strconv.FormatInt(int64(*e), 10))
}

// UnmarshalJSON unmarshals the JSON input into an OnOffBool type
func (o *OnOffBool) UnmarshalJSON(b []byte) error {
	var oS string
	err := json.Unmarshal(b, &oS)
	if err != nil {
		return err
	}
	switch oS {
	case "yes":
		*o = true
	case "no":
		*o = false
	}
	return nil
}

// MarshalJSON marshals the OnOffBool type into a JSON string
func (o OnOffBool) MarshalJSON() ([]byte, error) {
	if o {
		return json.Marshal("on")
	}
	return json.Marshal("off")
}

// spotOrderHelper is a helper function for unmarshalling spot order endpoints
func (e *Exchange) spotOrderHelper(ctx context.Context, path string, vals url.Values) ([]SpotOrderDetailData, error) {
	var temp []OrderDetailTemp
	err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, vals, nil, &temp)
	if err != nil {
		return nil, err
	}
	resp := make([]SpotOrderDetailData, len(temp))
	for i := range temp {
		err = json.Unmarshal(temp[i].FeeDetailTemp, &resp[i].FeeDetail)
		if err != nil {
			return nil, err
		}
		resp[i].UserID = temp[i].UserID
		resp[i].Symbol = temp[i].Symbol
		resp[i].OrderID = temp[i].OrderID
		resp[i].ClientOrderID = temp[i].ClientOrderID
		resp[i].Price = temp[i].Price
		resp[i].Size = temp[i].Size
		resp[i].OrderType = temp[i].OrderType
		resp[i].Side = temp[i].Side
		resp[i].Status = temp[i].Status
		resp[i].PriceAverage = temp[i].PriceAverage
		resp[i].BaseVolume = temp[i].BaseVolume
		resp[i].QuoteVolume = temp[i].QuoteVolume
		resp[i].EnterPointSource = temp[i].EnterPointSource
		resp[i].CreationTime = temp[i].CreationTime
		resp[i].UpdateTime = temp[i].UpdateTime
		resp[i].OrderSource = temp[i].OrderSource
		resp[i].TriggerPrice = temp[i].TriggerPrice
		resp[i].TPSLType = temp[i].TPSLType
		resp[i].CancelReason = temp[i].CancelReason
	}
	return resp, nil
}
