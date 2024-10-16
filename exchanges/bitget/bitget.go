package bitget

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
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitget is the overarching type across this package
type Bitget struct {
	exchange.Base
}

const (
	bitgetAPIURL = "https://api.bitget.com/api/v2/"

	// Public endpoints
	bitgetPublic            = "public/"
	bitgetAnnouncements     = "annoucements" // sic
	bitgetTime              = "time"
	bitgetMarket            = "market/"
	bitgetWhaleNetFlow      = "whale-net-flow"
	bitgetTakerBuySell      = "taker-buy-sell"
	bitgetPositionLongShort = "position-long-short"
	// bitgetLongShortRatio    = "long-short-ratio"
	// bitgetLoanGrowth          = "loan-growth"
	// bitgetIsolatedBorrowRate  = "isolated-borrow-rate"
	bitgetLongShort           = "long-short"
	bitgetFundFlow            = "fund-flow"
	bitgetSupportSymbols      = "support-symbols"
	bitgetFundNetFlow         = "fund-net-flow"
	bitgetAccountLongShort    = "account-long-short"
	bitgetCoins               = "coins"
	bitgetSymbols             = "symbols"
	bitgetVIPFeeRate          = "vip-fee-rate"
	bitgetTickers             = "tickers"
	bitgetMergeDepth          = "merge-depth"
	bitgetOrderbook           = "orderbook"
	bitgetCandles             = "candles"
	bitgetHistoryCandles      = "history-candles"
	bitgetFillsHistory        = "fills-history"
	bitgetTicker              = "ticker"
	bitgetHistoryIndexCandles = "history-index-candles"
	bitgetHistoryMarkCandles  = "history-mark-candles"
	bitgetOpenInterest        = "open-interest"
	bitgetFundingTime         = "funding-time"
	bitgetSymbolPrice         = "symbol-price"
	bitgetHistoryFundRate     = "history-fund-rate"
	bitgetCurrentFundRate     = "current-fund-rate"
	bitgetContracts           = "contracts"
	bitgetQueryPositionLever  = "query-position-lever"
	bitgetCoinInfos           = "coinInfos"
	bitgetHourInterest        = "hour-interest"

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
	bitgetBatchCreateSubAccApi     = "batch-create-subaccount-and-apikey"
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
	bitgetSetMarginMode            = "/set-margin-mode"
	bitgetSetPositionMode          = "/set-position-mode"
	bitgetBill                     = "/bill"
	bitgetPosition                 = "position/"
	bitgetSinglePosition           = "single-position"
	bitgetAllPositions             = "all-position" // sic
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

	errIntervalNotSupported           = "interval not supported"
	errAuthenticatedWebsocketDisabled = "%v AuthenticatedWebsocketAPISupport not enabled"
)

var (
	errBusinessTypeEmpty             = errors.New("businessType cannot be empty")
	errPairEmpty                     = errors.New("currency pair cannot be empty")
	errCurrencyEmpty                 = errors.New("currency cannot be empty")
	errProductTypeEmpty              = errors.New("productType cannot be empty")
	errSubaccountEmpty               = errors.New("subaccounts cannot be empty")
	errNewStatusEmpty                = errors.New("newStatus cannot be empty")
	errNewPermsEmpty                 = errors.New("newPerms cannot be empty")
	errPassphraseEmpty               = errors.New("passphrase cannot be empty")
	errLabelEmpty                    = errors.New("label cannot be empty")
	errAPIKeyEmpty                   = errors.New("apiKey cannot be empty")
	errFromToMutex                   = errors.New("exactly one of fromAmount and toAmount must be set")
	errTraceIDEmpty                  = errors.New("traceID cannot be empty")
	errAmountEmpty                   = errors.New("amount cannot be empty")
	errPriceEmpty                    = errors.New("price cannot be empty")
	errTypeAssertTimestamp           = errors.New("unable to type assert timestamp")
	errTypeAssertOpenPrice           = errors.New("unable to type assert opening price")
	errTypeAssertHighPrice           = errors.New("unable to type assert high price")
	errTypeAssertLowPrice            = errors.New("unable to type assert low price")
	errTypeAssertClosePrice          = errors.New("unable to type assert close price")
	errTypeAssertBaseVolume          = errors.New("unable to type assert base volume")
	errTypeAssertQuoteVolume         = errors.New("unable to type assert quote volume")
	errTypeAssertUSDTVolume          = errors.New("unable to type assert USDT volume")
	errGranEmpty                     = errors.New("granularity cannot be empty")
	errEndTimeEmpty                  = errors.New("endTime cannot be empty")
	errSideEmpty                     = errors.New("side cannot be empty")
	errOrderTypeEmpty                = errors.New("orderType cannot be empty")
	errStrategyEmpty                 = errors.New("strategy cannot be empty")
	errLimitPriceEmpty               = errors.New("price cannot be empty for limit orders")
	errOrderClientEmpty              = errors.New("at least one of orderID and clientOrderID must not be empty")
	errOrderIDEmpty                  = errors.New("orderID cannot be empty")
	errOrdersEmpty                   = errors.New("orders cannot be empty")
	errTriggerPriceEmpty             = errors.New("triggerPrice cannot be empty")
	errTriggerTypeEmpty              = errors.New("triggerType cannot be empty")
	errAccountTypeEmpty              = errors.New("accountType cannot be empty")
	errFromTypeEmpty                 = errors.New("fromType cannot be empty")
	errToTypeEmpty                   = errors.New("toType cannot be empty")
	errCurrencyAndPairEmpty          = errors.New("currency and pair cannot both be empty")
	errFromIDEmpty                   = errors.New("fromID cannot be empty")
	errToIDEmpty                     = errors.New("toID cannot be empty")
	errTransferTypeEmpty             = errors.New("transferType cannot be empty")
	errAddressEmpty                  = errors.New("address cannot be empty")
	errNoCandleData                  = errors.New("no candle data")
	errMarginCoinEmpty               = errors.New("marginCoin cannot be empty")
	errOpenAmountEmpty               = errors.New("openAmount cannot be empty")
	errOpenPriceEmpty                = errors.New("openPrice cannot be empty")
	errLeverageEmpty                 = errors.New("leverage cannot be empty")
	errMarginModeEmpty               = errors.New("marginMode cannot be empty")
	errPositionModeEmpty             = errors.New("positionMode cannot be empty")
	errNewClientOrderIDEmpty         = errors.New("newClientOrderID cannot be empty")
	errPlanTypeEmpty                 = errors.New("planType cannot be empty")
	errPlanOrderIDEmpty              = errors.New("planOrderID cannot be empty")
	errHoldSideEmpty                 = errors.New("holdSide cannot be empty")
	errExecutePriceEmpty             = errors.New("executePrice cannot be empty")
	errTakeProfitParamsInconsistency = errors.New("takeProfitTriggerPrice, takeProfitExecutePrice, and takeProfitTriggerType must either all be set or all be empty")
	errStopLossParamsInconsistency   = errors.New("stopLossTriggerPrice, stopLossExecutePrice, and stopLossTriggerType must either all be set or all be empty")
	errIDListEmpty                   = errors.New("idList cannot be empty")
	errLoanTypeEmpty                 = errors.New("loanType cannot be empty")
	errProductIDEmpty                = errors.New("productID cannot be empty")
	errPeriodTypeEmpty               = errors.New("periodType cannot be empty")
	errLoanCoinEmpty                 = errors.New("loanCoin cannot be empty")
	errCollateralCoinEmpty           = errors.New("collateralCoin cannot be empty")
	errTermEmpty                     = errors.New("term cannot be empty")
	errCollateralAmountEmpty         = errors.New("collateralAmount cannot be empty")
	errCollateralLoanMutex           = errors.New("exactly one of collateralAmount and loanAmount must be set")
	errReviseTypeEmpty               = errors.New("reviseType cannot be empty")
	errUnknownPairQuote              = errors.New("unknown pair quote; pair can't be split due to lack of delimiter and unclear base length")
	errStrategyMutex                 = errors.New("only one of immediate or cancel, fill or kill, and post only can be set to true")
	errOrderNotFound                 = errors.New("order not found")
	errReturnEmpty                   = errors.New("returned data unexpectedly empty")
	errInvalidChecksum               = errors.New("invalid checksum")

	prodTypes = []string{"USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES"}
	planTypes = []string{"normal_plan", "track_plan", "profit_loss"}
)

// QueryAnnouncement returns announcements from the exchange, filtered by type and time
func (bi *Bitget) QueryAnnouncements(ctx context.Context, annType string, startTime, endTime time.Time) ([]AnnResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("annType", annType)
	params.Values.Set("language", "en_US")
	var resp struct {
		AnnResp []AnnResp `json:"data"`
	}
	return resp.AnnResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetAnnouncements, params.Values, &resp)
}

// GetTime returns the server's time
func (bi *Bitget) GetTime(ctx context.Context) (*TimeResp, error) {
	var resp struct {
		TimeResp `json:"data"`
	}
	return &resp.TimeResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, bitgetPublic+bitgetTime, nil, &resp)
}

// GetTradeRate returns the fees the user would face for trading a given symbol
func (bi *Bitget) GetTradeRate(ctx context.Context, pair, businessType string) (*TradeRateResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if businessType == "" {
		return nil, errBusinessTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("businessType", businessType)
	var resp struct {
		TradeRateResp `json:"data"`
	}
	return &resp.TradeRateResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetCommon+bitgetTradeRate, vals, nil, &resp)
}

// GetSpotTransactionRecords returns the user's spot transaction records
func (bi *Bitget) GetSpotTransactionRecords(ctx context.Context, currency string, startTime, endTime time.Time, limit, pagination int64) ([]SpotTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp struct {
		SpotTrResp []SpotTrResp `json:"data"`
	}
	return resp.SpotTrResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetSpotRecord, params.Values, nil, &resp)
}

// GetFuturesTransactionRecords returns the user's futures transaction records
func (bi *Bitget) GetFuturesTransactionRecords(ctx context.Context, productType, currency string, startTime, endTime time.Time, limit, pagination int64) ([]FutureTrResp, error) {
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
	params.Values.Set("marginCoin", currency)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp struct {
		FutureTrResp []FutureTrResp `json:"data"`
	}
	return resp.FutureTrResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetFutureRecord, params.Values, nil, &resp)
}

// GetMarginTransactionRecords returns the user's margin transaction records
func (bi *Bitget) GetMarginTransactionRecords(ctx context.Context, marginType, currency string, startTime, endTime time.Time, limit, pagination int64) ([]MarginTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("marginType", marginType)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp struct {
		MarginTrResp []MarginTrResp `json:"data"`
	}
	return resp.MarginTrResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetMarginRecord, params.Values, nil, &resp)
}

// GetP2PTransactionRecords returns the user's P2P transaction records
func (bi *Bitget) GetP2PTransactionRecords(ctx context.Context, currency string, startTime, endTime time.Time, limit, pagination int64) ([]P2PTrResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp struct {
		P2PTrResp []P2PTrResp `json:"data"`
	}
	return resp.P2PTrResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetTax+bitgetP2PRecord, params.Values, nil, &resp)
}

// GetP2PMerchantList returns detailed information on merchants
func (bi *Bitget) GetP2PMerchantList(ctx context.Context, online string, limit, pagination int64) (*P2PMerListResp, error) {
	vals := url.Values{}
	vals.Set("online", online)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	var resp struct {
		P2PMerListResp `json:"data"`
	}
	return &resp.P2PMerListResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetMerchantList, vals, nil, &resp)
}

// GetMerchantInfo returns detailed information on the user as a merchant
func (bi *Bitget) GetMerchantInfo(ctx context.Context) (*P2PMerInfoResp, error) {
	var resp struct {
		P2PMerInfoResp `json:"data"`
	}
	return &resp.P2PMerInfoResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetMerchantInfo, nil, nil, &resp)
}

// GetMerchantP2POrders returns information on the user's P2P orders
func (bi *Bitget) GetMerchantP2POrders(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, ordNum int64, status, side, cryptoCurrency, fiatCurrency string) (*P2POrdersResp, error) {
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
	if cryptoCurrency != "" {
		params.Values.Set("coin", cryptoCurrency)
	}
	params.Values.Set("fiat", fiatCurrency)
	var resp struct {
		P2POrdersResp `json:"data"`
	}
	return &resp.P2POrdersResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetOrderList, params.Values, nil, &resp)
}

// GetMerchantAdvertisementList returns information on a variety of merchant advertisements
func (bi *Bitget) GetMerchantAdvertisementList(ctx context.Context, startTime, endTime time.Time, limit, pagination, adNum, payMethodID int64, status, side, cryptoCurrency, fiatCurrency, orderBy, sourceType string) ([]P2PAdListResp, error) {
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
	if cryptoCurrency != "" {
		params.Values.Set("coin", cryptoCurrency)
	}
	params.Values.Set("fiat", fiatCurrency)
	params.Values.Set("orderBy", orderBy)
	params.Values.Set("sourceType", sourceType)
	var resp struct {
		Data struct {
			AdList []P2PAdListResp `json:"advList"`
		} `json:"data"`
	}
	return resp.Data.AdList, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetP2P+bitgetAdvList, params.Values, nil, &resp)
}

// GetSpotWhaleNetFlow returns the amount whales have been trading in a specified pair recently
func (bi *Bitget) GetSpotWhaleNetFlow(ctx context.Context, pair string) ([]WhaleNetFlowResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetMarket + bitgetWhaleNetFlow
	var resp struct {
		WhaleNetFlowResp []WhaleNetFlowResp `json:"data"`
	}
	return resp.WhaleNetFlowResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesActiveVolume returns the active volume of a specified pair
func (bi *Bitget) GetFuturesActiveVolume(ctx context.Context, pair, period string) ([]ActiveVolumeResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetTakerBuySell
	var resp struct {
		ActiveVolumeResp []ActiveVolumeResp `json:"data"`
	}
	return resp.ActiveVolumeResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesPositionRatios returns the ratio of long to short positions for a specified pair
func (bi *Bitget) GetFuturesPositionRatios(ctx context.Context, pair, period string) ([]PosRatFutureResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetPositionLongShort
	var resp struct {
		PosRatFutureResp []PosRatFutureResp `json:"data"`
	}
	return resp.PosRatFutureResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// When queried, the exchange claims that this endpoint doesn't exist, despite having a documentation page
// // GetMarginPositionRatios returns the ratio of long to short positions for a specified pair in margin accounts
// func (bi *Bitget) GetMarginPositionRatios(ctx context.Context, pair, period, currency string) (*PosRatMarginResp, error) {
// 	if pair == "" {
// 		return nil, errPairEmpty
// 	}
// 	vals := url.Values{}
// 	vals.Set("symbol", pair)
// 	vals.Set("period", period)
// 	vals.Set("coin", currency)
// 	path := bitgetMix + bitgetMarket + bitgetLongShortRatio
// 	var resp *PosRatMarginResp
// 	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
// }

// When queried, the exchange claims that this endpoint doesn't exist, despite having a documentation page
// // GetMarginLoanGrowth returns the growth rate of borrowed funds for a specified pair in margin accounts
// func (bi *Bitget) GetMarginLoanGrowth(ctx context.Context, pair, period, currency string) (*LoanGrowthResp, error) {
// 	if pair == "" {
// 		return nil, errPairEmpty
// 	}
// 	vals := url.Values{}
// 	vals.Set("symbol", pair)
// 	vals.Set("period", period)
// 	vals.Set("coin", currency)
// 	path := bitgetMix + bitgetMarket + bitgetLoanGrowth
// 	var resp *LoanGrowthResp
// 	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
// }

// When queried, the exchange claims that this endpoint doesn't exist, despite having a documentation page
// // GetIsolatedBorrowingRatio returns the ratio of borrowed funds between base and quote currencies, after
// // converting to USDT, within isolated margin accounts
// func (bi *Bitget) GetIsolatedBorrowingRatio(ctx context.Context, pair, period string) (*BorrowRatioResp, error) {
// 	if pair == "" {
// 		return nil, errPairEmpty
// 	}
// 	vals := url.Values{}
// 	vals.Set("symbol", pair)
// 	vals.Set("period", period)
// 	path := bitgetMix + bitgetMarket + bitgetIsolatedBorrowRate
// 	var resp *BorrowRatioResp
// 	return resp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
// }

// GetFuturesRatios returns the ratio of long to short positions for a specified pair
func (bi *Bitget) GetFuturesRatios(ctx context.Context, pair, period string) ([]RatioResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetLongShort
	var resp struct {
		RatioResp []RatioResp `json:"data"`
	}
	return resp.RatioResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetSpotFundFlows returns information on volumes and buy/sell ratios for whales, dolphins, and fish for a particular pair
func (bi *Bitget) GetSpotFundFlows(ctx context.Context, pair string) (*FundFlowResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetMarket + bitgetFundFlow
	var resp struct {
		FundFlowResp `json:"data"`
	}
	return &resp.FundFlowResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetTradeSupportSymbols returns a list of supported symbols
func (bi *Bitget) GetTradeSupportSymbols(ctx context.Context) (*SymbolsResp, error) {
	path := bitgetSpot + bitgetMarket + bitgetSupportSymbols
	var resp struct {
		SymbolsResp `json:"data"`
	}
	return &resp.SymbolsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, nil, &resp)
}

// GetSpotWhaleFundFlows returns the amount whales have been trading in a specified pair recently
func (bi *Bitget) GetSpotWhaleFundFlows(ctx context.Context, pair string) ([]WhaleFundFlowResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetMarket + bitgetFundNetFlow
	var resp struct {
		WhaleFundFlowResp []WhaleFundFlowResp `json:"data"`
	}
	return resp.WhaleFundFlowResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// GetFuturesAccountRatios returns the ratio of long to short positions for a specified pair
func (bi *Bitget) GetFuturesAccountRatios(ctx context.Context, pair, period string) ([]AccountRatioResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("period", period)
	path := bitgetMix + bitgetMarket + bitgetAccountLongShort
	var resp struct {
		AccountRatioResp []AccountRatioResp `json:"data"`
	}
	return resp.AccountRatioResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate1, path, vals, &resp)
}

// CreateVirtualSubaccounts creates a batch of virtual subaccounts. These names must use English letters, no spaces, no numbers, and be exactly 8 characters long.
func (bi *Bitget) CreateVirtualSubaccounts(ctx context.Context, subaccounts []string) (*CrVirSubResp, error) {
	if len(subaccounts) == 0 {
		return nil, errSubaccountEmpty
	}
	path := bitgetUser + bitgetCreate + bitgetVirtualSubaccount
	req := map[string]any{
		"subAccountList": subaccounts,
	}
	var resp struct {
		CrVirSubResp `json:"data"`
	}
	return &resp.CrVirSubResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ModifyVirtualSubaccount changes the permissions and/or status of a virtual subaccount
func (bi *Bitget) ModifyVirtualSubaccount(ctx context.Context, subaccountID, newStatus string, newPerms []string) (*SuccessBool, error) {
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
	var resp struct {
		SuccessBool `json:"data"`
	}
	return &resp.SuccessBool, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// CreateSubaccountAndAPIKey creates a subaccounts and an API key. Every account can have up to 20 sub-accounts, and every API key can have up to 10 API keys. The name of the sub-account must be exactly 8 English letters. The passphrase of the API key must be 8-32 letters and/or numbers. The label must be 20 or fewer characters. A maximum of 30 IPs can be a part of the whitelist.
func (bi *Bitget) CreateSubaccountAndAPIKey(ctx context.Context, subaccountName, passphrase, label string, whiteList, permList []string) ([]CrSubAccAPIKeyResp, error) {
	if subaccountName == "" {
		return nil, errSubaccountEmpty
	}
	req := map[string]any{
		"subAccountName": subaccountName,
		"passphrase":     passphrase,
		"label":          label,
		"ipList":         whiteList,
		"permList":       permList,
	}
	var resp struct {
		CrSubAccAPIKeyResp []CrSubAccAPIKeyResp `json:"data"`
	}
	return resp.CrSubAccAPIKeyResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, bitgetUser+bitgetBatchCreateSubAccApi, nil, req, &resp)
}

// GetVirtualSubaccounts returns a list of the user's virtual sub-accounts
func (bi *Bitget) GetVirtualSubaccounts(ctx context.Context, limit, pagination int64, status string) (*GetVirSubResp, error) {
	vals := url.Values{}
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	vals.Set("status", status)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetList
	var resp struct {
		GetVirSubResp `json:"data"`
	}
	return &resp.GetVirSubResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CreateAPIKey creates an API key for the selected virtual sub-account
func (bi *Bitget) CreateAPIKey(ctx context.Context, subaccountID, passphrase, label string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
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
	var resp struct {
		AlterAPIKeyResp `json:"data"`
	}
	return &resp.AlterAPIKeyResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ModifyAPIKey modifies the label, IP whitelist, and/or permissions of the API key associated with the selected virtual sub-account
func (bi *Bitget) ModifyAPIKey(ctx context.Context, subaccountID, passphrase, label, apiKey string, whiteList, permList []string) (*AlterAPIKeyResp, error) {
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
	req := make(map[string]any)
	req["subAccountUid"] = subaccountID
	req["passphrase"] = passphrase
	req["label"] = label
	req["subAccountApiKey"] = apiKey
	req["ipList"] = whiteList
	req["permList"] = permList
	var resp struct {
		AlterAPIKeyResp `json:"data"`
	}
	return &resp.AlterAPIKeyResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetAPIKeys lists the API keys associated with the selected virtual sub-account
func (bi *Bitget) GetAPIKeys(ctx context.Context, subaccountID string) ([]GetAPIKeyResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	vals := url.Values{}
	vals.Set("subAccountUid", subaccountID)
	path := bitgetUser + bitgetVirtualSubaccount + "-" + bitgetAPIKey + "-" + bitgetList
	var resp struct {
		GetAPIKeyResp []GetAPIKeyResp `json:"data"`
	}
	return resp.GetAPIKeyResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetFundingAssets returns the user's assets
func (bi *Bitget) GetFundingAssets(ctx context.Context, currency string) ([]FundingAssetsResp, error) {
	vals := url.Values{}
	if currency != "" {
		vals.Set("coin", currency)
	}
	var resp struct {
		FundingAssetsResp []FundingAssetsResp `json:"data"`
	}
	return resp.FundingAssetsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetAccount+bitgetFundingAssets, vals, nil, &resp)
}

// GetBotAccountAssets returns the user's bot account assets
func (bi *Bitget) GetBotAccountAssets(ctx context.Context, accountType string) ([]BotAccAssetsResp, error) {
	vals := url.Values{}
	if accountType != "" {
		vals.Set("accountType", accountType)
	}
	var resp struct {
		BotAccAssetsResp []BotAccAssetsResp `json:"data"`
	}
	return resp.BotAccAssetsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetAccount+bitgetBotAssets, vals, nil, &resp)
}

// GetAssetOverview returns an overview of the user's assets across various account types
func (bi *Bitget) GetAssetOverview(ctx context.Context) ([]AssetOverviewResp, error) {
	var resp struct {
		AssetOverviewResp []AssetOverviewResp `json:"data"`
	}
	return resp.AssetOverviewResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, bitgetAccount+bitgetAllAccountBalance, nil, nil, &resp)
}

// GetConvertCoins returns a list of supported currencies, your balance in those currencies, and the maximum and minimum tradable amounts of those currencies
func (bi *Bitget) GetConvertCoins(ctx context.Context) ([]ConvertCoinsResp, error) {
	var resp struct {
		ConvertCoinsResp []ConvertCoinsResp `json:"data"`
	}
	return resp.ConvertCoinsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetCurrencies, nil, nil, &resp)
}

// GetQuotedPrice returns the price of a given amount of one currency in terms of another currency, and an ID for this quote, to be used in a subsequent conversion
func (bi *Bitget) GetQuotedPrice(ctx context.Context, fromCurrency, toCurrency string, fromAmount, toAmount float64) (*QuotedPriceResp, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, errCurrencyEmpty
	}
	if (fromAmount == 0 && toAmount == 0) || (fromAmount != 0 && toAmount != 0) {
		return nil, errFromToMutex
	}
	vals := url.Values{}
	vals.Set("fromCoin", fromCurrency)
	vals.Set("toCoin", toCurrency)
	if fromAmount != 0 {
		vals.Set("fromCoinSize", strconv.FormatFloat(fromAmount, 'f', -1, 64))
	} else {
		vals.Set("toCoinSize", strconv.FormatFloat(toAmount, 'f', -1, 64))
	}
	var resp struct {
		QuotedPriceResp `json:"data"`
	}
	return &resp.QuotedPriceResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetQuotedPrice, vals, nil, &resp)
}

// CommitConversion commits a conversion previously quoted by GetQuotedPrice. This quote has to have been issued within the last 8 seconds.
func (bi *Bitget) CommitConversion(ctx context.Context, fromCurrency, toCurrency, traceID string, fromAmount, toAmount, price float64) (*CommitConvResp, error) {
	if fromCurrency == "" || toCurrency == "" {
		return nil, errCurrencyEmpty
	}
	if traceID == "" {
		return nil, errTraceIDEmpty
	}
	if fromAmount == 0 || toAmount == 0 {
		return nil, errAmountEmpty
	}
	if price == 0 {
		return nil, errPriceEmpty
	}
	req := map[string]any{
		"fromCoin":     fromCurrency,
		"toCoin":       toCurrency,
		"traceId":      traceID,
		"fromCoinSize": strconv.FormatFloat(fromAmount, 'f', -1, 64),
		"toCoinSize":   strconv.FormatFloat(toAmount, 'f', -1, 64),
		"cnvtPrice":    strconv.FormatFloat(price, 'f', -1, 64),
	}
	var resp struct {
		CommitConvResp `json:"data"`
	}
	return &resp.CommitConvResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, bitgetConvert+bitgetTrade, nil, req, &resp)
}

// GetConvertHistory returns a list of the user's previous conversions
func (bi *Bitget) GetConvertHistory(ctx context.Context, startTime, endTime time.Time, limit, pagination int64) (*ConvHistResp, error) {
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
	var resp struct {
		ConvHistResp `json:"data"`
	}
	return &resp.ConvHistResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetConvertRecord, params.Values, nil, &resp)
}

// GetBGBConvertCoins returns a list of available currencies, with information on converting them to BGB
func (bi *Bitget) GetBGBConvertCoins(ctx context.Context) ([]BGBConvertCoinsResp, error) {
	var resp struct {
		Data struct {
			CoinList []BGBConvertCoinsResp `json:"coinList"`
		} `json:"data"`
	}
	return resp.Data.CoinList, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetConvertCoinList, nil, nil, &resp)
}

// ConvertBGB converts all funds in the listed currencies to BGB
func (bi *Bitget) ConvertBGB(ctx context.Context, currencies []string) ([]ConvertBGBResp, error) {
	if len(currencies) == 0 {
		return nil, errCurrencyEmpty
	}
	req := map[string]any{
		"coinList": currencies,
	}
	var resp struct {
		Data struct {
			OrderList []ConvertBGBResp `json:"orderList"`
		} `json:"data"`
	}
	return resp.Data.OrderList, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, bitgetConvert+bitgetBGBConvert, nil, req, &resp)
}

// GetBGBConvertHistory returns a list of the user's previous BGB conversions
func (bi *Bitget) GetBGBConvertHistory(ctx context.Context, orderID, limit, pagination int64, startTime, endTime time.Time) ([]BGBConvHistResp, error) {
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
	var resp struct {
		BGBConvHistResp []BGBConvHistResp `json:"data"`
	}
	return resp.BGBConvHistResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, bitgetConvert+bitgetBGBConvertRecords, params.Values, nil, &resp)
}

// GetCoinInfo returns information on all supported spot currencies, or a single currency of the user's choice
func (bi *Bitget) GetCoinInfo(ctx context.Context, currency string) ([]CoinInfoResp, error) {
	vals := url.Values{}
	if currency != "" {
		vals.Set("coin", currency)
	}
	path := bitgetSpot + bitgetPublic + bitgetCoins
	var resp struct {
		CoinInfoResp []CoinInfoResp `json:"data"`
	}
	return resp.CoinInfoResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate3, path, vals, &resp)
}

// GetSymbolInfo returns information on all supported spot trading pairs, or a single pair of the user's choice
func (bi *Bitget) GetSymbolInfo(ctx context.Context, pair string) ([]SymbolInfoResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetPublic + bitgetSymbols
	var resp struct {
		SymbolInfoResp []SymbolInfoResp `json:"data"`
	}
	return resp.SymbolInfoResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotVIPFeeRate returns the different levels of VIP fee rates for spot trading
func (bi *Bitget) GetSpotVIPFeeRate(ctx context.Context) ([]VIPFeeRateResp, error) {
	path := bitgetSpot + bitgetMarket + bitgetVIPFeeRate
	var resp struct {
		VIPFeeRateResp []VIPFeeRateResp `json:"data"`
	}
	return resp.VIPFeeRateResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetSpotTickerInformation returns the ticker information for all trading pairs, or a single pair of the user's choice
func (bi *Bitget) GetSpotTickerInformation(ctx context.Context, pair string) ([]TickerResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetSpot + bitgetMarket + bitgetTickers
	var resp struct {
		TickerResp []TickerResp `json:"data"`
	}
	return resp.TickerResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotMergeDepth returns part of the orderbook, with options to merge orders of similar price levels together, and to change how many results are returned. Limit's a string instead of the typical int64 because the API will accept a value of "max"
func (bi *Bitget) GetSpotMergeDepth(ctx context.Context, pair, precision, limit string) (*DepthResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("precision", precision)
	vals.Set("limit", limit)
	path := bitgetSpot + bitgetMarket + bitgetMergeDepth
	var resp struct {
		DepthResp `json:"data"`
	}
	return &resp.DepthResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetOrderbookDepth returns the orderbook for a given trading pair, with options to merge orders of similar price levels together, and to change how many results are returned.
func (bi *Bitget) GetOrderbookDepth(ctx context.Context, pair, step string, limit uint8) (*OrderbookResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("type", step)
	vals.Set("limit", strconv.FormatUint(uint64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetOrderbook
	var resp struct {
		OrderbookResp `json:"data"`
	}
	return &resp.OrderbookResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetSpotCandlestickData returns candlestick data for a given trading pair
func (bi *Bitget) GetSpotCandlestickData(ctx context.Context, pair, granularity string, startTime, endTime time.Time, limit uint16, historic bool) (*CandleData, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	return bi.candlestickHelper(ctx, pair, granularity, path, limit, params)
}

// GetRecentSpotFills returns the most recent trades for a given pair
func (bi *Bitget) GetRecentSpotFills(ctx context.Context, pair string, limit uint16) ([]MarketFillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("limit", strconv.FormatInt(int64(limit), 10))
	path := bitgetSpot + bitgetMarket + bitgetFills
	var resp struct {
		MarketFillsResp []MarketFillsResp `json:"data"`
	}
	return resp.MarketFillsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetSpotMarketTrades returns trades for a given pair within a particular time range, and/or before a certain ID
func (bi *Bitget) GetSpotMarketTrades(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination int64) ([]MarketFillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetSpot + bitgetMarket + bitgetFillsHistory
	var resp struct {
		MarketFillsResp []MarketFillsResp `json:"data"`
	}
	return resp.MarketFillsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, params.Values, &resp)
}

// PlaceSpotOrder places a spot order on the exchange
func (bi *Bitget) PlaceSpotOrder(ctx context.Context, pair, side, orderType, strategy, clientOrderID string, price, amount float64, isCopyTradeLeader bool) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	if orderType == "limit" && price == 0 {
		return nil, errLimitPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"side":      side,
		"orderType": orderType,
		"force":     strategy,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
		"size":      strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid": clientOrderID,
	}
	path := bitgetSpot + bitgetTrade + bitgetPlaceOrder
	var resp *OrderIDResp
	// I suspect the two rate limits have to do with distinguishing ordinary traders, and traders who are also copy trade leaders. Since this isn't detectable, it'll be handled in the relevant functions through a bool
	rLim := Rate10
	if isCopyTradeLeader {
		rLim = Rate1
	}
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// CancelSpotOrderByID cancels an order on the exchange
func (bi *Bitget) CancelSpotOrderByID(ctx context.Context, pair, clientOrderID string, orderID int64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	req := map[string]any{
		"symbol": pair,
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceSpotOrders places up to fifty orders on the exchange
func (bi *Bitget) BatchPlaceSpotOrders(ctx context.Context, pair string, orders []PlaceSpotOrderStruct, isCopyTradeLeader bool) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchOrders
	var resp struct {
		BatchOrderResp `json:"data"`
	}
	rLim := Rate5
	if isCopyTradeLeader {
		rLim = Rate1
	}
	return &resp.BatchOrderResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelOrders cancels up to fifty orders on the exchange
func (bi *Bitget) BatchCancelOrders(ctx context.Context, pair string, orderIDs []OrderIDStruct) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orderIDs) == 0 {
		return nil, errOrderIDEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orderIDs,
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchCancel
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelOrdersBySymbol cancels orders for a given symbol. Doesn't return information on failures/successes
func (bi *Bitget) CancelOrdersBySymbol(ctx context.Context, pair string) (string, error) {
	if pair == "" {
		return "", errPairEmpty
	}
	req := map[string]any{
		"symbol": pair,
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelSymbolOrder
	var resp struct {
		Data struct {
			Symbol string `json:"symbol"`
		} `json:"data"`
	}
	return resp.Data.Symbol, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetSpotOrderDetails returns information on a single order
func (bi *Bitget) GetSpotOrderDetails(ctx context.Context, orderID int64, clientOrderID string) ([]SpotOrderDetailData, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	vals := url.Values{}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if clientOrderID != "" {
		vals.Set("clientOid", clientOrderID)
	}
	vals.Set("requestTime", strconv.FormatInt(time.Now().UnixMilli(), 10))
	vals.Set("receiveWindow", "60000")
	path := bitgetSpot + bitgetTrade + bitgetOrderInfo
	return bi.spotOrderHelper(ctx, path, vals)
}

// GetUnfilledOrders returns information on the user's unfilled orders
func (bi *Bitget) GetUnfilledOrders(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination, orderID int64) ([]UnfilledOrdersResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetSpot + bitgetTrade + bitgetUnfilledOrders
	var resp struct {
		UnfilledOrdersResp []UnfilledOrdersResp `json:"data"`
	}
	return resp.UnfilledOrdersResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetHistoricalSpotOrders returns the user's spot order history
func (bi *Bitget) GetHistoricalSpotOrders(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination, orderID int64) ([]SpotOrderDetailData, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetSpot + bitgetTrade + bitgetHistoryOrders
	return bi.spotOrderHelper(ctx, path, params.Values)
}

// GetSpotFills returns information on the user's fulfilled orders in a certain pair
func (bi *Bitget) GetSpotFills(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination, orderID int64) ([]SpotFillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetSpot + bitgetTrade + "/" + bitgetFills
	var resp struct {
		SpotFillsResp []SpotFillsResp `json:"data"`
	}
	return resp.SpotFillsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// PlacePlanSpotOrder sets up an order to be placed after certain conditions are met
func (bi *Bitget) PlacePlanSpotOrder(ctx context.Context, pair, side, orderType, planType, triggerType, clientOrderID, strategy string, triggerPrice, executePrice, amount float64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if triggerPrice == 0 {
		return nil, errTriggerPriceEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if orderType == "limit" && executePrice == 0 {
		return nil, errLimitPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
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
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetTrade + bitgetPlacePlanOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &resp)
}

// ModifyPlanSpotOrder alters the price, trigger price, amount, or order type of a plan order
func (bi *Bitget) ModifyPlanSpotOrder(ctx context.Context, orderID int64, clientOrderID, orderType string, triggerPrice, executePrice, amount float64) (*OrderIDResp, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if triggerPrice == 0 {
		return nil, errTriggerPriceEmpty
	}
	if orderType == "limit" && executePrice == 0 {
		return nil, errLimitPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &resp)
}

// CancelPlanSpotOrder cancels a plan order
func (bi *Bitget) CancelPlanSpotOrder(ctx context.Context, orderID int64, clientOrderID string) (*SuccessBool, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	req := make(map[string]any)
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	if orderID != 0 {
		req["orderId"] = strconv.FormatInt(orderID, 10)
	}
	path := bitgetSpot + bitgetTrade + bitgetCancelPlanOrder
	var resp struct {
		SuccessBool `json:"success"`
	}
	return &resp.SuccessBool, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodPost, path, nil, req, &resp)
}

// GetCurrentSpotPlanOrders returns the user's current plan orders
func (bi *Bitget) GetCurrentSpotPlanOrders(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination int64) (*PlanSpotOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetTrade + bitgetCurrentPlanOrder
	var resp struct {
		PlanSpotOrderResp `json:"data"`
	}
	return &resp.PlanSpotOrderResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSpotPlanSubOrder returns the sub-orders of a triggered plan order
func (bi *Bitget) GetSpotPlanSubOrder(ctx context.Context, orderID string) (*SubOrderResp, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	vals := url.Values{}
	vals.Set("planOrderId", orderID)
	path := bitgetSpot + bitgetTrade + bitgetPlanSubOrder
	var resp struct {
		SubOrderResp `json:"data"`
	}
	return &resp.SubOrderResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, vals, nil, &resp)
}

// GetSpotPlanOrderHistory returns the user's plan order history
func (bi *Bitget) GetSpotPlanOrderHistory(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination int64) (*PlanSpotOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	path := bitgetSpot + bitgetTrade + bitgetPlanOrderHistory
	var resp struct {
		PlanSpotOrderResp `json:"data"`
	}
	return &resp.PlanSpotOrderResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// BatchCancelSpotPlanOrders cancels all plan orders, with the option to restrict to only those for particular pairs
func (bi *Bitget) BatchCancelSpotPlanOrders(ctx context.Context, pairs []string) (*BatchOrderResp, error) {
	req := make(map[string]any)
	if len(pairs) > 0 {
		req["symbolList"] = pairs
	}
	path := bitgetSpot + bitgetTrade + bitgetBatchCancelPlanOrder
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetAccountInfo returns the user's account information
func (bi *Bitget) GetAccountInfo(ctx context.Context) (*AccountInfoResp, error) {
	path := bitgetSpot + bitgetAccount + bitgetInfo
	var resp struct {
		AccountInfoResp `json:"data"`
	}
	return &resp.AccountInfoResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path, nil, nil, &resp)
}

// GetAccountAssets returns information on the user's assets
func (bi *Bitget) GetAccountAssets(ctx context.Context, currency, assetType string) ([]AssetData, error) {
	vals := url.Values{}
	if currency != "" {
		vals.Set("coin", currency)
	}
	vals.Set("type", assetType)
	path := bitgetSpot + bitgetAccount + bitgetAssets
	var resp struct {
		AssetData []AssetData `json:"data"`
	}
	return resp.AssetData, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSpotSubaccountAssets returns information on assets in the user's sub-accounts
func (bi *Bitget) GetSpotSubaccountAssets(ctx context.Context) ([]SubaccountAssetsResp, error) {
	path := bitgetSpot + bitgetAccount + bitgetSubaccountAssets
	var resp struct {
		SubaccountAssetsResp []SubaccountAssetsResp `json:"data"`
	}
	return resp.SubaccountAssetsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// ModifyDepositAccount changes which account is automatically used for deposits of a particular currency
func (bi *Bitget) ModifyDepositAccount(ctx context.Context, accountType, currency string) (*SuccessBool, error) {
	if accountType == "" {
		return nil, errAccountTypeEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	req := map[string]any{
		"coin":        currency,
		"accountType": accountType,
	}
	path := bitgetSpot + bitgetWallet + bitgetModifyDepositAccount
	var resp struct {
		SuccessBool `json:"data"`
	}
	return &resp.SuccessBool, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSpotAccountBills returns a section of the user's billing history
func (bi *Bitget) GetSpotAccountBills(ctx context.Context, currency, groupType, businessType string, startTime, endTime time.Time, limit, pagination int64) ([]SpotAccBillResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if currency != "" {
		params.Values.Set("coin", currency)
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
	var resp struct {
		SpotAccBillResp []SpotAccBillResp `json:"data"`
	}
	return resp.SpotAccBillResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// TransferAsset transfers a certain amount of a currency or pair between different productType accounts
func (bi *Bitget) TransferAsset(ctx context.Context, fromType, toType, currency, pair, clientOrderID string, amount float64) (*TransferResp, error) {
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	if toType == "" {
		return nil, errToTypeEmpty
	}
	if currency == "" && pair == "" {
		return nil, errCurrencyAndPairEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
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
	var resp struct {
		TransferResp `json:"data"`
	}
	return &resp.TransferResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetTransferableCoinList returns a list of coins that can be transferred between the provided accounts
func (bi *Bitget) GetTransferableCoinList(ctx context.Context, fromType, toType string) ([]string, error) {
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
	var resp struct {
		Data []string `json:"data"`
	}
	return resp.Data, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubaccountTransfer transfers assets between sub-accounts
func (bi *Bitget) SubaccountTransfer(ctx context.Context, fromType, toType, currency, pair, clientOrderID, fromID, toID string, amount float64) (*TransferResp, error) {
	if fromType == "" {
		return nil, errFromTypeEmpty
	}
	if toType == "" {
		return nil, errToTypeEmpty
	}
	if currency == "" && pair == "" {
		return nil, errCurrencyAndPairEmpty
	}
	if fromID == "" {
		return nil, errFromIDEmpty
	}
	if toID == "" {
		return nil, errToIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"fromType": fromType,
		"toType":   toType,
		"amount":   strconv.FormatFloat(amount, 'f', -1, 64),
		"coin":     currency,
		"symbol":   pair,
		"fromId":   fromID,
		"toId":     toID,
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetWallet + bitgetSubaccountTransfer
	var resp struct {
		TransferResp `json:"data"`
	}
	return &resp.TransferResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// WithdrawFunds withdraws funds from the user's account
func (bi *Bitget) WithdrawFunds(ctx context.Context, currency, transferType, address, chain, innerAddressType, areaCode, tag, note, clientOrderID string, amount float64) (*OrderIDResp, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if transferType == "" {
		return nil, errTransferTypeEmpty
	}
	if address == "" {
		return nil, errAddressEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
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
	}
	if clientOrderID != "" {
		req["clientOid"] = clientOrderID
	}
	path := bitgetSpot + bitgetWallet + bitgetWithdrawal
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSubaccountTransferRecord returns the user's sub-account transfer history
func (bi *Bitget) GetSubaccountTransferRecord(ctx context.Context, currency, subaccountID, clientOrderID string, startTime, endTime time.Time, limit, pagination int64) ([]SubaccTfrRecResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	params.Values.Set("subUid", subaccountID)
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
	var resp struct {
		SubaccTfrRecResp []SubaccTfrRecResp `json:"data"`
	}
	return resp.SubaccTfrRecResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// GetTransferRecord returns the user's transfer history
func (bi *Bitget) GetTransferRecord(ctx context.Context, currency, fromType, clientOrderID string, startTime, endTime time.Time, limit, pagination int64) ([]TransferRecResp, error) {
	if currency == "" {
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
	params.Values.Set("coin", currency)
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
	path := bitgetSpot + bitgetAccount + bitgetTransferRecord
	var resp struct {
		TransferRecResp []TransferRecResp `json:"data"`
	}
	return resp.TransferRecResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// SwitchBGBDeductionStatus switches the deduction of BGB for trading fees on and off
func (bi *Bitget) SwitchBGBDeductionStatus(ctx context.Context, deduct bool) (bool, error) {
	req := make(map[string]any)
	if deduct {
		req["deduct"] = "on"
	} else {
		req["deduct"] = "off"
	}
	path := bitgetSpot + bitgetAccount + bitgetSwitchDeduct
	var resp struct {
		Data bool `json:"data"`
	}
	return resp.Data, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, path, nil, req, &resp)
}

// GetDepositAddressForCurrency returns the user's deposit address for a particular currency
func (bi *Bitget) GetDepositAddressForCurrency(ctx context.Context, currency, chain string) (*DepositAddressResp, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	vals.Set("chain", chain)
	path := bitgetSpot + bitgetWallet + bitgetDepositAddress
	var resp struct {
		DepositAddressResp `json:"data"`
	}
	return &resp.DepositAddressResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSubaccountDepositAddress returns the deposit address for a particular currency and sub-account
func (bi *Bitget) GetSubaccountDepositAddress(ctx context.Context, subaccountID, currency, chain string) (*DepositAddressResp, error) {
	if subaccountID == "" {
		return nil, errSubaccountEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("subUid", subaccountID)
	vals.Set("coin", currency)
	vals.Set("chain", chain)
	path := bitgetSpot + bitgetWallet + bitgetSubaccountDepositAddress
	var resp struct {
		DepositAddressResp `json:"data"`
	}
	return &resp.DepositAddressResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetBGBDeductionStatus returns the user's current BGB deduction status
func (bi *Bitget) GetBGBDeductionStatus(ctx context.Context) (string, error) {
	path := bitgetSpot + bitgetAccount + bitgetDeductInfo
	var resp struct {
		Data struct {
			Deduct string `json:"deduct"`
		} `json:"data"`
	}
	return resp.Data.Deduct, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, nil, nil, &resp)
}

// CancelWithdrawal cancels a large withdrawal request that was placed in the last minute
func (bi *Bitget) CancelWithdrawal(ctx context.Context, orderID string) (*SuccessBool, error) {
	if orderID == "" {
		return nil, errOrderIDEmpty
	}
	req := map[string]any{
		"orderId": orderID,
	}
	path := bitgetSpot + bitgetWallet + bitgetCancelWithdrawal
	var resp *SuccessBool
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSubaccountDepositRecords returns the deposit history for a sub-account
func (bi *Bitget) GetSubaccountDepositRecords(ctx context.Context, subaccountID, currency string, orderID, pagination, limit int64, startTime, endTime time.Time) ([]SubaccDepRecResp, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetWallet + bitgetSubaccountDepositRecord
	var resp struct {
		SubaccDepRecResp []SubaccDepRecResp `json:"data"`
	}
	return resp.SubaccDepRecResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetWithdrawalRecords returns the user's withdrawal history
func (bi *Bitget) GetWithdrawalRecords(ctx context.Context, currency, clientOrderID string, startTime, endTime time.Time, pagination, orderID, limit int64) ([]WithdrawRecordsResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if currency != "" {
		params.Values.Set("coin", currency)
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
	var resp struct {
		WithdrawRecordsResp []WithdrawRecordsResp `json:"data"`
	}
	return resp.WithdrawRecordsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetDepositRecords returns the user's cryptocurrency deposit history
func (bi *Bitget) GetDepositRecords(ctx context.Context, crypto string, orderID, pagination, limit int64, startTime, endTime time.Time) ([]CryptoDepRecResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	if crypto != "" {
		params.Values.Set("coin", crypto)
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetSpot + bitgetWallet + bitgetDepositRecord
	var resp struct {
		CryptoDepRecResp []CryptoDepRecResp `json:"data"`
	}
	return resp.CryptoDepRecResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetFuturesVIPFeeRate returns the different levels of VIP fee rates for futures trading
func (bi *Bitget) GetFuturesVIPFeeRate(ctx context.Context) ([]VIPFeeRateResp, error) {
	path := bitgetMix + bitgetMarket + bitgetVIPFeeRate
	var resp struct {
		VIPFeeRateResp []VIPFeeRateResp `json:"data"`
	}
	return resp.VIPFeeRateResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetFuturesMergeDepth returns part of the orderbook, with options to merge orders of similar price levels together, and to change how many results are returned. Limit's a string instead of the typical int64 because the API will accept a value of "max"
func (bi *Bitget) GetFuturesMergeDepth(ctx context.Context, pair, productType, precision, limit string) (*DepthResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	vals.Set("precision", precision)
	vals.Set("limit", limit)
	path := bitgetMix + bitgetMarket + bitgetMergeDepth
	var resp struct {
		DepthResp `json:"data"`
	}
	return &resp.DepthResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesTicker returns the ticker information for a pair of the user's choice
func (bi *Bitget) GetFuturesTicker(ctx context.Context, pair, productType string) ([]FutureTickerResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetTicker
	var resp struct {
		FutureTickerResp []FutureTickerResp `json:"data"`
	}
	return resp.FutureTickerResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetAllFuturesTickers returns the ticker information for all pairs
func (bi *Bitget) GetAllFuturesTickers(ctx context.Context, productType string) ([]FutureTickerResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetTickers
	var resp struct {
		FutureTickerResp []FutureTickerResp `json:"data"`
	}
	return resp.FutureTickerResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetRecentFuturesFills returns the most recent trades for a given pair
func (bi *Bitget) GetRecentFuturesFills(ctx context.Context, pair, productType string, limit int64) ([]MarketFillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetFills
	var resp struct {
		MarketFillsResp []MarketFillsResp `json:"data"`
	}
	return resp.MarketFillsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesMarketTrades returns trades for a given pair within a particular time range, and/or before a certain ID
func (bi *Bitget) GetFuturesMarketTrades(ctx context.Context, pair, productType string, limit, pagination int64, startTime, endTime time.Time) ([]MarketFillsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	params.Values.Set("symbol", pair)
	params.Values.Set("productType", productType)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetFillsHistory
	var resp struct {
		MarketFillsResp []MarketFillsResp `json:"data"`
	}
	return resp.MarketFillsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, params.Values, &resp)
}

// GetFuturesCandlestickData returns candlestick data for a given pair within a particular time range
func (bi *Bitget) GetFuturesCandlestickData(ctx context.Context, pair, productType, granularity string, startTime, endTime time.Time, limit uint16, mode CallMode) (*CandleData, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	case CallModeHistory:
		path += bitgetHistoryCandles
	case CallModeIndex:
		path += bitgetHistoryIndexCandles
	case CallModeMark:
		path += bitgetHistoryMarkCandles
	}
	return bi.candlestickHelper(ctx, pair, granularity, path, limit, params)
}

// GetOpenPositions returns the total positions of a particular pair
func (bi *Bitget) GetOpenPositions(ctx context.Context, pair, productType string) (*OpenPositionsResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetOpenInterest
	var resp struct {
		OpenPositionsResp `json:"data"`
	}
	return &resp.OpenPositionsResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetNextFundingTime returns the settlement time and period of a particular contract
func (bi *Bitget) GetNextFundingTime(ctx context.Context, pair, productType string) ([]FundingTimeResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetFundingTime
	var resp struct {
		FundingTimeResp []FundingTimeResp `json:"data"`
	}
	return resp.FundingTimeResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFuturesPrices returns the current market, index, and mark prices for a given pair
func (bi *Bitget) GetFuturesPrices(ctx context.Context, pair, productType string) ([]FuturesPriceResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetSymbolPrice
	var resp struct {
		FuturesPriceResp []FuturesPriceResp `json:"data"`
	}
	return resp.FuturesPriceResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFundingHistorical returns the historical funding rates for a given pair
func (bi *Bitget) GetFundingHistorical(ctx context.Context, pair, productType string, limit, pagination int64) ([]FundingHistoryResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	if limit != 0 {
		vals.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("pageNo", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMix + bitgetMarket + bitgetHistoryFundRate
	var resp struct {
		FundingHistoryResp []FundingHistoryResp `json:"data"`
	}
	return resp.FundingHistoryResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetFundingCurrent returns the current funding rate for a given pair
func (bi *Bitget) GetFundingCurrent(ctx context.Context, pair, productType string) ([]FundingCurrentResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetCurrentFundRate
	var resp struct {
		FundingCurrentResp []FundingCurrentResp `json:"data"`
	}
	return resp.FundingCurrentResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetContractConfig returns details for a given contract
func (bi *Bitget) GetContractConfig(ctx context.Context, pair, productType string) ([]ContractConfigResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	path := bitgetMix + bitgetMarket + bitgetContracts
	var resp struct {
		ContractConfigResp []ContractConfigResp `json:"data"`
	}
	return resp.ContractConfigResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, vals, &resp)
}

// GetOneFuturesAccount returns details for the account associated with a given pair, margin coin, and product type
func (bi *Bitget) GetOneFuturesAccount(ctx context.Context, pair, productType, marginCoin string) (*OneAccResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin)
	path := bitgetMix + bitgetAccount + "/" + bitgetAccount
	var resp struct {
		OneAccResp `json:"data"`
	}
	return &resp.OneAccResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetAllFuturesAccounts returns details for all accounts
func (bi *Bitget) GetAllFuturesAccounts(ctx context.Context, productType string) ([]FutureAccDetails, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetAccount + bitgetAccounts
	var resp struct {
		FutureAccDetails []FutureAccDetails `json:"data"`
	}
	return resp.FutureAccDetails, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetFuturesSubaccountAssets returns details on the assets of all sub-accounts
func (bi *Bitget) GetFuturesSubaccountAssets(ctx context.Context, productType string) ([]SubaccountFuturesResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	path := bitgetMix + bitgetAccount + bitgetSubaccountAssets2
	var resp struct {
		SubaccountFuturesResp []SubaccountFuturesResp `json:"data"`
	}
	return resp.SubaccountFuturesResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodGet, path, vals, nil, &resp)
}

// GetEstimatedOpenCount returns the estimated size of open orders for a given pair
func (bi *Bitget) GetEstimatedOpenCount(ctx context.Context, pair, productType, marginCoin string, openAmount, openPrice, leverage float64) (float64, error) {
	if pair == "" {
		return 0, errPairEmpty
	}
	if productType == "" {
		return 0, errProductTypeEmpty
	}
	if marginCoin == "" {
		return 0, errMarginCoinEmpty
	}
	if openAmount == 0 {
		return 0, errOpenAmountEmpty
	}
	if openPrice == 0 {
		return 0, errOpenPriceEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin)
	vals.Set("openAmount", strconv.FormatFloat(openAmount, 'f', -1, 64))
	vals.Set("openPrice", strconv.FormatFloat(openPrice, 'f', -1, 64))
	if leverage != 0 {
		vals.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	}
	path := bitgetMix + bitgetAccount + bitgetOpenCount
	var resp struct {
		Data struct {
			Size float64 `json:"size,string"`
		} `json:"data"`
	}
	return resp.Data.Size, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// ChangeLeverage changes the leverage for the given pair and product type
func (bi *Bitget) ChangeLeverage(ctx context.Context, pair, productType, marginCoin, holdSide string, leverage float64) (*LeverageResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin == "" {
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
	var resp struct {
		LeverageResp `json:"data"`
	}
	return &resp.LeverageResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// AdjustIsolatedAutoMargin adjusts the auto margin for a specified isolated margin account
func (bi *Bitget) AdjustIsolatedAutoMargin(ctx context.Context, pair, marginCoin, holdSide string, autoMargin bool, amount float64) error {
	if pair == "" {
		return errPairEmpty
	}
	if marginCoin == "" {
		return errMarginCoinEmpty
	}
	req := map[string]any{
		"symbol":     pair,
		"marginCoin": marginCoin,
		"holdSide":   holdSide,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if autoMargin {
		req["autoMargin"] = "on"
	} else {
		req["autoMargin"] = "off"
	}
	path := bitgetMix + bitgetAccount + bitgetSetAutoMargin
	return bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, nil)
}

// AdjustMargin adds or subtracts margin from a position
func (bi *Bitget) AdjustMargin(ctx context.Context, pair, productType, marginCoin, holdSide string, amount float64) error {
	if pair == "" {
		return errPairEmpty
	}
	if productType == "" {
		return errProductTypeEmpty
	}
	if marginCoin == "" {
		return errMarginCoinEmpty
	}
	if amount == 0 {
		return errAmountEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"marginCoin":  marginCoin,
		"amount":      strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if holdSide != "" {
		req["holdSide"] = holdSide
	}
	path := bitgetMix + bitgetAccount + bitgetSetMargin
	return bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, nil)
}

// ChangeMarginMode changes the margin mode for a given pair. Can only be done when there the user has no open positions or orders
func (bi *Bitget) ChangeMarginMode(ctx context.Context, pair, productType, marginCoin, marginMode string) (*LeverageResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin == "" {
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
	var resp struct {
		LeverageResp `json:"data"`
	}
	return &resp.LeverageResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// ChangePositionMode changes the position mode for any pair. Having any positions or orders on any side of any pair may cause this to fail.
func (bi *Bitget) ChangePositionMode(ctx context.Context, productType, positionMode string) (string, error) {
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
		Data struct {
			PosMode string `json:"posMode"`
		} `json:"data"`
	}
	return resp.Data.PosMode, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesAccountBills returns a section of the user's billing history
func (bi *Bitget) GetFuturesAccountBills(ctx context.Context, productType, pair, currency, businessType string, pagination, limit int64, startTime, endTime time.Time) ([]FutureAccBillResp, error) {
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
	params.Values.Set("symbol", pair)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	params.Values.Set("businessType", businessType)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetAccount + bitgetBill
	var resp struct {
		Data struct {
			Bills []FutureAccBillResp `json:"bills"`
		} `json:"data"`
	}
	return resp.Data.Bills, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetPositionTier returns the position configuration for a given pair
func (bi *Bitget) GetPositionTier(ctx context.Context, productType, pair string) ([]PositionTierResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("symbol", pair)
	path := bitgetMix + bitgetMarket + bitgetQueryPositionLever
	var resp struct {
		PositionTierResp []PositionTierResp `json:"data"`
	}
	return resp.PositionTierResp, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetSinglePosition returns position details for a given productType, pair, and marginCoin. The exchange recommends using the websocket feed instead, as information from this endpoint may be delayed during settlement or market fluctuations
func (bi *Bitget) GetSinglePosition(ctx context.Context, productType, pair, marginCoin string) ([]PositionResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("symbol", pair)
	vals.Set("marginCoin", marginCoin)
	path := bitgetMix + bitgetPosition + bitgetSinglePosition
	var resp struct {
		PositionResp []PositionResp `json:"data"`
	}
	return resp.PositionResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetAllPositions returns position details for a given productType and marginCoin. The exchange recommends using the websocket feed instead, as information from this endpoint may be delayed during settlement or market fluctuations
func (bi *Bitget) GetAllPositions(ctx context.Context, productType, marginCoin string) ([]PositionResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	vals := url.Values{}
	vals.Set("productType", productType)
	vals.Set("marginCoin", marginCoin)
	path := bitgetMix + bitgetPosition + bitgetAllPositions
	var resp struct {
		PositionResp []PositionResp `json:"data"`
	}
	return resp.PositionResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetHistoricalPositions returns historical position details, up to a maximum of three months ago
func (bi *Bitget) GetHistoricalPositions(ctx context.Context, pair, productType string, pagination, limit int64, startTime, endTime time.Time) (*HistPositionResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
	params.Values.Set("productType", productType)
	fmt.Printf("pagination %v\n", pagination)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetPosition + bitgetHistoryPosition
	var resp struct {
		HistPositionResp `json:"data"`
	}
	return &resp.HistPositionResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, params.Values, nil, &resp)
}

// PlaceFuturesOrder places a futures order on the exchange
func (bi *Bitget) PlaceFuturesOrder(ctx context.Context, pair, productType, marginMode, marginCoin, side, tradeSide, orderType, strategy, clientOID string, stopSurplusPrice, stopLossPrice, amount, price float64, reduceOnly, isCopyTradeLeader bool) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	if side == "" {
		return nil, errSideEmpty
	}
	if orderType == "" {
		return nil, errOrderTypeEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if orderType == "limit" && price == 0 {
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// PlaceReversal attempts to close a position, in part or in whole, and opens a position of corresponding size on the opposite side. This operation may only be done in part under certain margin levels, market conditions, or other unspecified factors. If a reversal is attempted for an amount greater than the current outstanding position, that position will be closed, and a new position will be opened for the amount of the closed position; not the amount specified in the request. The side specified in the parameter should correspond to the side of the position you're attempting to close; if the original is open_long, use close_long; if the original is open_short, use close_short; if the original is sell_single, use buy_single. If the position is sell_single or buy_single, the amount parameter will be ignored, and the entire position will be closed, with a corresponding amount opened on the opposite side.
func (bi *Bitget) PlaceReversal(ctx context.Context, pair, marginCoin, productType, side, tradeSide, clientOID string, amount float64, isCopyTradeLeader bool) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"marginCoin":  marginCoin,
		"productType": productType,
		"side":        side,
		"tradeSide":   tradeSide,
		"size":        strconv.FormatFloat(amount, 'f', -1, 64),
		"orderType":   "market",
	}
	if clientOID != "" {
		req["clientOid"] = clientOID
	}
	path := bitgetMix + bitgetOrder + bitgetClickBackhand
	rLim := Rate10
	if isCopyTradeLeader {
		rLim = Rate1
	}
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceFuturesOrders places multiple orders at once. Can also be used to modify the take-profit and stop-loss of an open position.
func (bi *Bitget) BatchPlaceFuturesOrders(ctx context.Context, pair, productType, marginCoin, marginMode string, orders []PlaceFuturesOrderStruct, isCopyTradeLeader bool) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginCoin == "" {
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, rLim, http.MethodPost, path, nil, req, &resp)
}

// ModifyFuturesOrder can change the size, price, take-profit, and stop-loss of an order. Size and price have to be modified at the same time, or the request will fail. If size and price are altered, the old order will be cancelled, and a new one will be created asynchronously. Due to the asynchronous creation of a new order, a new ClientOrderID must be supplied so it can be tracked.
func (bi *Bitget) ModifyFuturesOrder(ctx context.Context, orderID int64, clientOrderID, pair, productType, newClientOrderID string, newAmount, newPrice, newTakeProfit, newStopLoss float64) (*OrderIDResp, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelFuturesOrder cancels an order on the exchange
func (bi *Bitget) CancelFuturesOrder(ctx context.Context, pair, productType, marginCoin, clientOrderID string, orderID int64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, errOrderClientEmpty
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
	if marginCoin != "" {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetCancelOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelFuturesOrders cancels multiple orders at once
func (bi *Bitget) BatchCancelFuturesOrders(ctx context.Context, orderIDs []OrderIDStruct, pair, productType, marginCoin string) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"symbol":      pair,
		"productType": productType,
		"orderList":   orderIDs,
	}
	if marginCoin != "" {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetBatchCancelOrders
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// FlashClosePosition attempts to close a position at the best available price
func (bi *Bitget) FlashClosePosition(ctx context.Context, pair, holdSide, productType string) (*BatchOrderResp, error) {
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate1, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesOrderDetails returns details on a given order
func (bi *Bitget) GetFuturesOrderDetails(ctx context.Context, pair, productType, clientOrderID string, orderID int64) (*FuturesOrderDetailResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, errOrderClientEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	vals.Set("productType", productType)
	if clientOrderID != "" {
		vals.Set("clientOid", clientOrderID)
	}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetDetail
	var resp *FuturesOrderDetailResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetFuturesFills returns fill details
func (bi *Bitget) GetFuturesFills(ctx context.Context, orderID, pagination, limit int64, pair, productType string, startTime, endTime time.Time) (*FuturesFillsResp, error) {
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
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + "/" + bitgetFills
	var resp struct {
		FuturesFillsResp `json:"data"`
	}
	return &resp.FuturesFillsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetFuturesOrderFillHistory returns historical fill details
func (bi *Bitget) GetFuturesOrderFillHistory(ctx context.Context, pair, productType string, orderID, pagination, limit int64, startTime, endTime time.Time) (*FuturesFillsResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		FuturesFillsResp `json:"data"`
	}
	return &resp.FuturesFillsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetPendingFuturesOrders returns detailed information on pending futures orders
func (bi *Bitget) GetPendingFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, pair, productType, status string, startTime, endTime time.Time) (*FuturesOrdResp, error) {
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
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetMix + bitgetOrder + bitgetOrdersPending
	var resp struct {
		FuturesOrdResp `json:"data"`
	}
	return &resp.FuturesOrdResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetHistoricalFuturesOrders returns information on futures orders that are no longer pending
func (bi *Bitget) GetHistoricalFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, pair, productType string, startTime, endTime time.Time) (*FuturesOrdResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	path := bitgetMix + bitgetOrder + bitgetOrdersHistory
	var resp struct {
		FuturesOrdResp `json:"data"`
	}
	return &resp.FuturesOrdResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// CancelAllFuturesOrders cancels all pending orders
func (bi *Bitget) CancelAllFuturesOrders(ctx context.Context, pair, productType, marginCoin string, acceptableDelay time.Duration) (*BatchOrderResp, error) {
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	req := map[string]any{
		"productType":   productType,
		"symbol":        pair,
		"requestTime":   time.Now().UnixMilli(),
		"receiveWindow": time.Unix(0, 0).Add(acceptableDelay).UnixMilli(),
	}
	if marginCoin != "" {
		req["marginCoin"] = marginCoin
	}
	path := bitgetMix + bitgetOrder + bitgetCancelAllOrders
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetFuturesTriggerOrderByID returns information on a particular trigger order
func (bi *Bitget) GetFuturesTriggerOrderByID(ctx context.Context, planType, productType string, planOrderID int64) (*SubOrderResp, error) {
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
	var resp struct {
		SubOrderResp `json:"data"`
	}
	return &resp.SubOrderResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// PlaceTPSLFuturesOrder places a take-profit or stop-loss futures order
func (bi *Bitget) PlaceTPSLFuturesOrder(ctx context.Context, marginCoin, productType, pair, planType, triggerType, holdSide, rangeRate, clientOrderID string, triggerPrice, executePrice, amount float64) (*OrderIDResp, error) {
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if holdSide == "" {
		return nil, errHoldSideEmpty
	}
	if triggerPrice == 0 {
		return nil, errTriggerPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"marginCoin":   marginCoin,
		"productType":  productType,
		"symbol":       pair,
		"planType":     planType,
		"triggerType":  triggerType,
		"holdSide":     holdSide,
		"rangeRate":    rangeRate,
		"clientOid":    clientOrderID,
		"triggerPrice": strconv.FormatFloat(triggerPrice, 'f', -1, 64),
		"executePrice": strconv.FormatFloat(executePrice, 'f', -1, 64),
		"size":         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetMix + bitgetOrder + bitgetPlaceTPSLOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceTriggerFuturesOrder places a trigger futures order
func (bi *Bitget) PlaceTriggerFuturesOrder(ctx context.Context, planType, pair, productType, marginMode, marginCoin, triggerType, side, tradeSide, orderType, clientOrderID, takeProfitTriggerType, stopLossTriggerType string, amount, executePrice, callbackRatio, triggerPrice, takeProfitTriggerPrice, takeProfitExecutePrice, stopLossTriggerPrice, stopLossExecutePrice float64, reduceOnly bool) (*OrderIDResp, error) {
	if planType == "" {
		return nil, errPlanTypeEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if marginMode == "" {
		return nil, errMarginModeEmpty
	}
	if marginCoin == "" {
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
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if executePrice == 0 {
		return nil, errExecutePriceEmpty
	}
	if triggerPrice == 0 {
		return nil, errTriggerPriceEmpty
	}
	if (takeProfitTriggerType != "" || takeProfitTriggerPrice != 0 || takeProfitExecutePrice != 0) && (takeProfitTriggerType == "" || takeProfitTriggerPrice == 0 || takeProfitExecutePrice == 0) {
		return nil, errTakeProfitParamsInconsistency
	}
	if (stopLossTriggerType != "" || stopLossTriggerPrice != 0 || stopLossExecutePrice != 0) && (stopLossTriggerType == "" || stopLossTriggerPrice == 0 || stopLossExecutePrice == 0) {
		return nil, errStopLossParamsInconsistency
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// ModifyTPSLFuturesOrder modifies a take-profit or stop-loss futures order
func (bi *Bitget) ModifyTPSLFuturesOrder(ctx context.Context, orderID int64, clientOrderID, marginCoin, productType, pair, triggerType string, triggerPrice, executePrice, amount, rangeRate float64) (*OrderIDResp, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
	}
	if marginCoin == "" {
		return nil, errMarginCoinEmpty
	}
	if productType == "" {
		return nil, errProductTypeEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	if triggerPrice == 0 {
		return nil, errTriggerPriceEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// ModifyTriggerFuturesOrder modifies a trigger futures order
func (bi *Bitget) ModifyTriggerFuturesOrder(ctx context.Context, orderID int64, clientOrderID, productType, triggerType, takeProfitTriggerType, stopLossTriggerType string, amount, executePrice, callbackRatio, triggerPrice, takeProfitTriggerPrice, takeProfitExecutePrice, stopLossTriggerPrice, stopLossExecutePrice float64) (*OrderIDResp, error) {
	if orderID == 0 && clientOrderID == "" {
		return nil, errOrderClientEmpty
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetPendingTriggerFuturesOrders returns information on pending trigger orders
func (bi *Bitget) GetPendingTriggerFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, pair, planType, productType string, startTime, endTime time.Time) (*PlanFuturesOrdResp, error) {
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
	params.Values.Set("symbol", pair)
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
	var resp struct {
		PlanFuturesOrdResp `json:"data"`
	}
	return &resp.PlanFuturesOrdResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// CancelTriggerFuturesOrders cancels trigger futures orders
func (bi *Bitget) CancelTriggerFuturesOrders(ctx context.Context, orderIDList []OrderIDStruct, pair, productType, marginCoin, planType string) (*BatchOrderResp, error) {
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetHistoricalTriggerFuturesOrders returns information on historical trigger orders
func (bi *Bitget) GetHistoricalTriggerFuturesOrders(ctx context.Context, orderID, pagination, limit int64, clientOrderID, planType, planStatus, pair, productType string, startTime, endTime time.Time) (*HistTriggerFuturesOrdResp, error) {
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
	params.Values.Set("symbol", pair)
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
	var resp struct {
		HistTriggerFuturesOrdResp `json:"data"`
	}
	return &resp.HistTriggerFuturesOrdResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSupportedCurrencies returns information on the currencies supported by the exchange
func (bi *Bitget) GetSupportedCurrencies(ctx context.Context) ([]SupCurrencyResp, error) {
	path := bitgetMargin + bitgetCurrencies
	var resp struct {
		Data []SupCurrencyResp `json:"data"`
	}
	return resp.Data, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, nil, &resp)
}

// GetCrossBorrowHistory returns the borrowing history for cross margin
func (bi *Bitget) GetCrossBorrowHistory(ctx context.Context, loanID, limit, pagination int64, currency string, startTime, endTime time.Time) (*BorrowHistCross, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetCrossed + bitgetBorrowHistory
	var resp struct {
		BorrowHistCross `json:"data"`
	}
	return &resp.BorrowHistCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossRepayHistory returns the repayment history for cross margin
func (bi *Bitget) GetCrossRepayHistory(ctx context.Context, repayID, limit, pagination int64, currency string, startTime, endTime time.Time) (*RepayHistResp, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetCrossed + bitgetRepayHistory
	var resp struct {
		RepayHistResp `json:"data"`
	}
	return &resp.RepayHistResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossInterestHistory returns the interest history for cross margin
func (bi *Bitget) GetCrossInterestHistory(ctx context.Context, currency string, startTime, endTime time.Time, limit, pagination int64) (*InterHistCross, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetCrossed + bitgetInterestHistory
	var resp struct {
		InterHistCross `json:"data"`
	}
	return &resp.InterHistCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossLiquidationHistory returns the liquidation history for cross margin
func (bi *Bitget) GetCrossLiquidationHistory(ctx context.Context, startTime, endTime time.Time, limit, pagination int64) (*LiquidHistCross, error) {
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
	var resp struct {
		LiquidHistCross `json:"data"`
	}
	return &resp.LiquidHistCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossFinancialHistory returns the financial history for cross margin
func (bi *Bitget) GetCrossFinancialHistory(ctx context.Context, marginType, currency string, startTime, endTime time.Time, limit, pagination int64) (*FinHistCrossResp, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetCrossed + bitgetFinancialRecords
	var resp struct {
		FinHistCrossResp `json:"data"`
	}
	return &resp.FinHistCrossResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossAccountAssets returns the account assets for cross margin
func (bi *Bitget) GetCrossAccountAssets(ctx context.Context, currency string) ([]CrossAssetResp, error) {
	vals := url.Values{}
	if currency != "" {
		vals.Set("coin", currency)
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetAssets
	var resp struct {
		CrossAssetResp []CrossAssetResp `json:"data"`
	}
	return resp.CrossAssetResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CrossBorrow borrows funds for cross margin
func (bi *Bitget) CrossBorrow(ctx context.Context, currency, clientOrderID string, amount float64) (*BorrowCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"coin":         currency,
		"borrowAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid":    clientOrderID,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetBorrow
	var resp struct {
		BorrowCross `json:"data"`
	}
	return &resp.BorrowCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CrossRepay repays funds for cross margin
func (bi *Bitget) CrossRepay(ctx context.Context, currency string, amount float64) (*RepayCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"coin":        currency,
		"repayAmount": strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetRepay
	var resp struct {
		RepayCross `json:"data"`
	}
	return &resp.RepayCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossRiskRate returns the risk rate for cross margin
func (bi *Bitget) GetCrossRiskRate(ctx context.Context) (float64, error) {
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetRiskRate
	var resp struct {
		Data struct {
			RiskRateRatio float64 `json:"riskRateRatio"`
		} `json:"data"`
	}
	return resp.Data.RiskRateRatio, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetCrossMaxBorrowable returns the maximum amount that can be borrowed for cross margin
func (bi *Bitget) GetCrossMaxBorrowable(ctx context.Context, currency string) (*MaxBorrowCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetMaxBorrowableAmount
	var resp struct {
		MaxBorrowCross `json:"data"`
	}
	return &resp.MaxBorrowCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossMaxTransferable returns the maximum amount that can be transferred out of cross margin
func (bi *Bitget) GetCrossMaxTransferable(ctx context.Context, currency string) (*MaxTransferCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetMaxTransferOutAmount
	var resp struct {
		MaxTransferCross `json:"data"`
	}
	return &resp.MaxTransferCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossInterestRateAndMaxBorrowable returns the interest rate and maximum borrowable amount for cross margin
func (bi *Bitget) GetCrossInterestRateAndMaxBorrowable(ctx context.Context, currency string) ([]IntRateMaxBorrowCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetMargin + bitgetCrossed + bitgetInterestRateAndLimit
	var resp struct {
		IntRateMaxBorrowCross []IntRateMaxBorrowCross `json:"data"`
	}
	return resp.IntRateMaxBorrowCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetCrossTierConfiguration returns tier information for the user's VIP level
func (bi *Bitget) GetCrossTierConfiguration(ctx context.Context, currency string) ([]TierConfigCross, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetMargin + bitgetCrossed + bitgetTierData
	var resp struct {
		TierConfigCross []TierConfigCross `json:"data"`
	}
	return resp.TierConfigCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// CrossFlashRepay repays funds for cross margin, with the option to only repay for a particular currency
func (bi *Bitget) CrossFlashRepay(ctx context.Context, currency string) (*FlashRepayCross, error) {
	req := map[string]any{
		"coin": currency,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetFlashRepay
	var resp struct {
		FlashRepayCross `json:"data"`
	}
	return &resp.FlashRepayCross, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossFlashRepayResult returns the result of the supplied flash repayments for cross margin
func (bi *Bitget) GetCrossFlashRepayResult(ctx context.Context, idList []int64) ([]FlashRepayResult, error) {
	if len(idList) == 0 {
		return nil, errIDListEmpty
	}
	req := map[string]any{
		"repayIdList": idList,
	}
	path := bitgetMargin + bitgetCrossed + "/" + bitgetAccount + bitgetQueryFlashRepayStatus
	var resp struct {
		FlashRepayResult []FlashRepayResult `json:"data"`
	}
	return resp.FlashRepayResult, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceCrossOrder places an order using cross margin
func (bi *Bitget) PlaceCrossOrder(ctx context.Context, pair, orderType, loanType, strategy, clientOrderID, side string, price, baseAmount, quoteAmount float64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	if baseAmount == 0 && quoteAmount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderType": orderType,
		"loanType":  loanType,
		"force":     strategy,
		"clientOid": clientOrderID,
		"side":      side,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
	}
	if baseAmount != 0 {
		req["baseSize"] = strconv.FormatFloat(baseAmount, 'f', -1, 64)
	}
	if quoteAmount != 0 {
		req["quoteSize"] = strconv.FormatFloat(quoteAmount, 'f', -1, 64)
	}
	path := bitgetMargin + bitgetCrossed + bitgetPlaceOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceCrossOrders places multiple orders using cross margin
func (bi *Bitget) BatchPlaceCrossOrders(ctx context.Context, pair string, orders []MarginOrderData) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelCrossOrder cancels an order using cross margin
func (bi *Bitget) CancelCrossOrder(ctx context.Context, pair, clientOrderID string, orderID int64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, errOrderClientEmpty
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
	path := bitgetMargin + bitgetCrossed + bitgetCancelOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelCrossOrders cancels multiple orders using cross margin
func (bi *Bitget) BatchCancelCrossOrders(ctx context.Context, pair string, orders []OrderIDStruct) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetMargin + bitgetCrossed + bitgetBatchCancelOrder
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetCrossOpenOrders returns the open orders for cross margin
func (bi *Bitget) GetCrossOpenOrders(ctx context.Context, pair, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOpenOrds, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginOpenOrds `json:"data"`
	}
	return &resp.MarginOpenOrds, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossHistoricalOrders returns the historical orders for cross margin
func (bi *Bitget) GetCrossHistoricalOrders(ctx context.Context, pair, enterPointSource, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginHistOrds, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginHistOrds `json:"data"`
	}
	return &resp.MarginHistOrds, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossOrderFills returns the fills for cross margin orders
func (bi *Bitget) GetCrossOrderFills(ctx context.Context, pair string, orderID, pagination, limit int64, startTime, endTime time.Time) (*MarginOrderFills, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginOrderFills `json:"data"`
	}
	return &resp.MarginOrderFills, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetCrossLiquidationOrders returns the liquidation orders for cross margin
func (bi *Bitget) GetCrossLiquidationOrders(ctx context.Context, orderType, pair, fromCoin, toCoin string, startTime, endTime time.Time, limit, pagination int64) (*LiquidationResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderType", orderType)
	params.Values.Set("symbol", pair)
	params.Values.Set("fromCoin", fromCoin)
	params.Values.Set("toCoin", toCoin)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetCrossed + bitgetLiquidationOrder
	var resp struct {
		LiquidationResp `json:"data"`
	}
	return &resp.LiquidationResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedRepayHistory returns the repayment history for isolated margin
func (bi *Bitget) GetIsolatedRepayHistory(ctx context.Context, pair, currency string, repayID, limit, pagination int64, startTime, endTime time.Time) (*RepayHistResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	params.Values.Set("symbol", pair)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetIsolated + bitgetRepayHistory
	var resp struct {
		RepayHistResp `json:"data"`
	}
	return &resp.RepayHistResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedBorrowHistory returns the borrowing history for isolated margin
func (bi *Bitget) GetIsolatedBorrowHistory(ctx context.Context, pair, currency string, loanID, limit, pagination int64, startTime, endTime time.Time) (*BorrowHistIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
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
	params.Values.Set("symbol", pair)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetIsolated + bitgetBorrowHistory
	var resp struct {
		BorrowHistIso `json:"data"`
	}
	return &resp.BorrowHistIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedInterestHistory returns the interest history for isolated margin
func (bi *Bitget) GetIsolatedInterestHistory(ctx context.Context, pair, currency string, startTime, endTime time.Time, limit, pagination int64) (*InterHistIso, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	params.Values.Set("symbol", pair)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetIsolated + bitgetInterestHistory
	var resp struct {
		InterHistIso `json:"data"`
	}
	return &resp.InterHistIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedLiquidationHistory returns the liquidation history for isolated margin
func (bi *Bitget) GetIsolatedLiquidationHistory(ctx context.Context, pair string, startTime, endTime time.Time, limit, pagination int64) (*LiquidHistIso, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	params.Values.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + bitgetLiquidationHistory
	var resp struct {
		LiquidHistIso `json:"data"`
	}
	return &resp.LiquidHistIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedFinancialHistory returns the financial history for isolated margin
func (bi *Bitget) GetIsolatedFinancialHistory(ctx context.Context, pair, marginType, currency string, startTime, endTime time.Time, limit, pagination int64) (*FinHistIsoResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	params.Values.Set("symbol", pair)
	params.Values.Set("marginType", marginType)
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	path := bitgetMargin + bitgetIsolated + bitgetFinancialRecords
	var resp struct {
		FinHistIsoResp `json:"data"`
	}
	return &resp.FinHistIsoResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedAccountAssets returns the account assets for isolated margin
func (bi *Bitget) GetIsolatedAccountAssets(ctx context.Context, pair string) ([]IsoAssetResp, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetAssets
	var resp struct {
		IsoAssetResp []IsoAssetResp `json:"data"`
	}
	return resp.IsoAssetResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// IsolatedBorrow borrows funds for isolated margin
func (bi *Bitget) IsolatedBorrow(ctx context.Context, pair, currency, clientOrderID string, amount float64) (*BorrowIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"symbol":       pair,
		"coin":         currency,
		"borrowAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"clientOid":    clientOrderID,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetBorrow
	var resp struct {
		BorrowIso `json:"data"`
	}
	return &resp.BorrowIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// IsolatedRepay repays funds for isolated margin
func (bi *Bitget) IsolatedRepay(ctx context.Context, amount float64, currency, pair, clientOrderID string) (*RepayIso, error) {
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	if pair == "" {
		return nil, errPairEmpty
	}
	req := map[string]any{
		"coin":        currency,
		"repayAmount": strconv.FormatFloat(amount, 'f', -1, 64),
		"symbol":      pair,
		"clientOid":   clientOrderID,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetRepay
	var resp struct {
		RepayIso `json:"data"`
	}
	return &resp.RepayIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedRiskRate returns the risk rate for isolated margin
func (bi *Bitget) GetIsolatedRiskRate(ctx context.Context, pair string, pagination, limit int64) ([]RiskRateIso, error) {
	vals := url.Values{}
	vals.Set("symbol", pair)
	if limit != 0 {
		vals.Set("pageSize", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("pageNum", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetRiskRate
	var resp struct {
		RiskRateIso []RiskRateIso `json:"data"`
	}
	return resp.RiskRateIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedInterestRateAndMaxBorrowable returns the interest rate and maximum borrowable amount for isolated margin
func (bi *Bitget) GetIsolatedInterestRateAndMaxBorrowable(ctx context.Context, pair string) ([]IntRateMaxBorrowIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + bitgetInterestRateAndLimit
	var resp struct {
		IntRateMaxBorrowIso []IntRateMaxBorrowIso `json:"data"`
	}
	return resp.IntRateMaxBorrowIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedTierConfiguration returns tier information for the user's VIP level
func (bi *Bitget) GetIsolatedTierConfiguration(ctx context.Context, pair string) ([]TierConfigIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + bitgetTierData
	var resp struct {
		TierConfigIso []TierConfigIso `json:"data"`
	}
	return resp.TierConfigIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedMaxBorrowable returns the maximum amount that can be borrowed for isolated margin
func (bi *Bitget) GetIsolatedMaxBorrowable(ctx context.Context, pair string) (*MaxBorrowIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetMaxBorrowableAmount
	var resp struct {
		MaxBorrowIso `json:"data"`
	}
	return &resp.MaxBorrowIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetIsolatedMaxTransferable returns the maximum amount that can be transferred out of isolated margin
func (bi *Bitget) GetIsolatedMaxTransferable(ctx context.Context, pair string) (*MaxTransferIso, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	vals := url.Values{}
	vals.Set("symbol", pair)
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetMaxTransferOutAmount
	var resp struct {
		MaxTransferIso `json:"data"`
	}
	return &resp.MaxTransferIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// IsolatedFlashRepay repays funds for isolated margin, with the option to only repay for a set of up to 100 pairs
func (bi *Bitget) IsolatedFlashRepay(ctx context.Context, pairs []string) ([]FlashRepayIso, error) {
	req := map[string]any{
		"symbolList": pairs,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetFlashRepay
	var resp struct {
		FlashRepayIso []FlashRepayIso `json:"data"`
	}
	return resp.FlashRepayIso, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedFlashRepayResult returns the result of the supplied flash repayments for isolated margin
func (bi *Bitget) GetIsolatedFlashRepayResult(ctx context.Context, idList []int64) ([]FlashRepayResult, error) {
	if len(idList) == 0 {
		return nil, errIDListEmpty
	}
	req := map[string]any{
		"repayIdList": idList,
	}
	path := bitgetMargin + bitgetIsolated + "/" + bitgetAccount + bitgetQueryFlashRepayStatus
	var resp struct {
		FlashRepayResult []FlashRepayResult `json:"data"`
	}
	return resp.FlashRepayResult, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// PlaceIsolatedOrder places an order using isolated margin
func (bi *Bitget) PlaceIsolatedOrder(ctx context.Context, pair, orderType, loanType, strategy, clientOrderID, side string, price, baseAmount, quoteAmount float64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
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
	if baseAmount == 0 && quoteAmount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderType": orderType,
		"loanType":  loanType,
		"force":     strategy,
		"clientOid": clientOrderID,
		"side":      side,
		"price":     strconv.FormatFloat(price, 'f', -1, 64),
	}
	if baseAmount != 0 {
		req["baseSize"] = strconv.FormatFloat(baseAmount, 'f', -1, 64)
	}
	if quoteAmount != 0 {
		req["quoteSize"] = strconv.FormatFloat(quoteAmount, 'f', -1, 64)
	}
	path := bitgetMargin + bitgetIsolated + bitgetPlaceOrder
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchPlaceIsolatedOrders places multiple orders using isolated margin
func (bi *Bitget) BatchPlaceIsolatedOrders(ctx context.Context, pair string, orders []MarginOrderData) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetMargin + bitgetIsolated + bitgetBatchPlaceOrder
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// CancelIsolatedOrder cancels an order using isolated margin
func (bi *Bitget) CancelIsolatedOrder(ctx context.Context, pair, clientOrderID string, orderID int64) (*OrderIDResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if clientOrderID == "" && orderID == 0 {
		return nil, errOrderClientEmpty
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
	var resp *OrderIDResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// BatchCancelIsolatedOrders cancels multiple orders using isolated margin
func (bi *Bitget) BatchCancelIsolatedOrders(ctx context.Context, pair string, orders []OrderIDStruct) (*BatchOrderResp, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	if len(orders) == 0 {
		return nil, errOrdersEmpty
	}
	req := map[string]any{
		"symbol":    pair,
		"orderList": orders,
	}
	path := bitgetMargin + bitgetIsolated + bitgetBatchCancelOrder
	var resp *BatchOrderResp
	return resp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetIsolatedOpenOrders returns the open orders for isolated margin
func (bi *Bitget) GetIsolatedOpenOrders(ctx context.Context, pair, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginOpenOrds, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginOpenOrds `json:"data"`
	}
	return &resp.MarginOpenOrds, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedHistoricalOrders returns the historical orders for isolated margin
func (bi *Bitget) GetIsolatedHistoricalOrders(ctx context.Context, pair, enterPointSource, clientOrderID string, orderID, limit, pagination int64, startTime, endTime time.Time) (*MarginHistOrds, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginHistOrds `json:"data"`
	}
	return &resp.MarginHistOrds, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedOrderFills returns the fills for isolated margin orders
func (bi *Bitget) GetIsolatedOrderFills(ctx context.Context, pair string, orderID, pagination, limit int64, startTime, endTime time.Time) (*MarginOrderFills, error) {
	if pair == "" {
		return nil, errPairEmpty
	}
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("symbol", pair)
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
	var resp struct {
		MarginOrderFills `json:"data"`
	}
	return &resp.MarginOrderFills, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetIsolatedLiquidationOrders returns the liquidation orders for isolated margin
func (bi *Bitget) GetIsolatedLiquidationOrders(ctx context.Context, orderType, pair, fromCoin, toCoin string, startTime, endTime time.Time, limit, pagination int64) (*LiquidationResp, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, true, true)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderType", orderType)
	params.Values.Set("symbol", pair)
	params.Values.Set("fromCoin", fromCoin)
	params.Values.Set("toCoin", toCoin)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetMargin + bitgetIsolated + bitgetLiquidationOrder
	var resp struct {
		LiquidationResp `json:"data"`
	}
	return &resp.LiquidationResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsProductList returns the list of savings products for a particular currency
func (bi *Bitget) GetSavingsProductList(ctx context.Context, currency, filter string) ([]SavingsProductList, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	vals.Set("filter", filter)
	path := bitgetEarn + bitgetSavings + bitgetProduct
	var resp struct {
		SavingsProductList []SavingsProductList `json:"data"`
	}
	return resp.SavingsProductList, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSavingsBalance returns the savings balance and amount earned in BTC and USDT
func (bi *Bitget) GetSavingsBalance(ctx context.Context) (*SavingsBalance, error) {
	path := bitgetEarn + bitgetSavings + "/" + bitgetAccount
	var resp struct {
		SavingsBalance `json:"data"`
	}
	return &resp.SavingsBalance, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetSavingsAssets returns information on assets held over the last three months
func (bi *Bitget) GetSavingsAssets(ctx context.Context, periodType string, startTime, endTime time.Time, limit, pagination int64) (*SavingsAssetsResp, error) {
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
	var resp struct {
		SavingsAssetsResp `json:"data"`
	}
	return &resp.SavingsAssetsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsRecords returns information on transactions performed over the last three months
func (bi *Bitget) GetSavingsRecords(ctx context.Context, currency, periodType, orderType string, startTime, endTime time.Time, limit, pagination int64) (*SavingsRecords, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	params.Values.Set("periodType", periodType)
	params.Values.Set("orderType", orderType)
	path := bitgetEarn + bitgetSavings + bitgetRecords
	var resp struct {
		SavingsRecords `json:"data"`
	}
	return &resp.SavingsRecords, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSavingsSubscriptionDetail returns detailed information on subscribing, for a single product
func (bi *Bitget) GetSavingsSubscriptionDetail(ctx context.Context, productID int64, periodType string) (*SavingsSubDetail, error) {
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
	var resp struct {
		SavingsSubDetail `json:"data"`
	}
	return &resp.SavingsSubDetail, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubscribeSavings applies funds to a savings product
func (bi *Bitget) SubscribeSavings(ctx context.Context, productID int64, periodType string, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"productId":  productID,
		"periodType": periodType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSavings + bitgetSubscribe
	var resp struct {
		SaveResp `json:"data"`
	}
	return &resp.SaveResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSavingsSubscriptionResult returns the result of a subscription attempt
func (bi *Bitget) GetSavingsSubscriptionResult(ctx context.Context, orderID int64, periodType string) (*SaveResult, error) {
	if orderID == 0 {
		return nil, errOrderIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	vals := url.Values{}
	vals.Set("orderId", strconv.FormatInt(orderID, 10))
	vals.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetSubscribeResult
	var resp struct {
		SaveResult `json:"data"`
	}
	return &resp.SaveResult, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// RedeemSavings redeems funds from a savings product
func (bi *Bitget) RedeemSavings(ctx context.Context, productID, orderID int64, periodType string, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"productId":  productID,
		"orderId":    orderID,
		"periodType": periodType,
		"amount":     strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSavings + bitgetRedeem
	var resp struct {
		SaveResp `json:"data"`
	}
	return &resp.SaveResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSavingsRedemptionResult returns the result of a redemption attempt
func (bi *Bitget) GetSavingsRedemptionResult(ctx context.Context, orderID int64, periodType string) (*SaveResult, error) {
	if orderID == 0 {
		return nil, errOrderIDEmpty
	}
	if periodType == "" {
		return nil, errPeriodTypeEmpty
	}
	vals := url.Values{}
	vals.Set("orderId", strconv.FormatInt(orderID, 10))
	vals.Set("periodType", periodType)
	path := bitgetEarn + bitgetSavings + bitgetRedeemResult
	var resp struct {
		SaveResult `json:"data"`
	}
	return &resp.SaveResult, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetEarnAccountAssets returns the assets in the earn account
func (bi *Bitget) GetEarnAccountAssets(ctx context.Context, currency string) ([]EarnAssets, error) {
	vals := url.Values{}
	if currency != "" {
		vals.Set("coin", currency)
	}
	path := bitgetEarn + bitgetAccount + bitgetAssets
	var resp struct {
		EarnAssets []EarnAssets `json:"data"`
	}
	return resp.EarnAssets, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSharkFinProducts returns information on Shark Fin products
func (bi *Bitget) GetSharkFinProducts(ctx context.Context, currency string, limit, pagination int64) (*SharkFinProductResp, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	if limit != 0 {
		vals.Set("limit", strconv.FormatInt(limit, 10))
	}
	if pagination != 0 {
		vals.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	path := bitgetEarn + bitgetSharkFin + bitgetProduct
	var resp struct {
		SharkFinProductResp `json:"data"`
	}
	return &resp.SharkFinProductResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// GetSharkFinBalance returns the balance and amount earned in BTC and USDT for Shark Fin products
func (bi *Bitget) GetSharkFinBalance(ctx context.Context) (*SharkFinBalance, error) {
	path := bitgetEarn + bitgetSharkFin + "/" + bitgetAccount
	var resp struct {
		SharkFinBalance `json:"data"`
	}
	return &resp.SharkFinBalance, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetSharkFinAssets returns information on assets held over the last three months for Shark Fin products
func (bi *Bitget) GetSharkFinAssets(ctx context.Context, status string, startTime, endTime time.Time, limit, pagination int64) (*SharkFinAssetsResp, error) {
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
	var resp struct {
		SharkFinAssetsResp `json:"data"`
	}
	return &resp.SharkFinAssetsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSharkFinRecords returns information on transactions performed over the last three months for Shark Fin products
func (bi *Bitget) GetSharkFinRecords(ctx context.Context, currency, transactionType string, startTime, endTime time.Time, limit, pagination int64) ([]SharkFinRecords, error) {
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
	if currency != "" {
		params.Values.Set("coin", currency)
	}
	params.Values.Set("type", transactionType)
	path := bitgetEarn + bitgetSharkFin + bitgetRecords
	var resp struct {
		Data struct {
			ResultList []SharkFinRecords `json:"resultList"`
		} `json:"data"`
	}
	return resp.Data.ResultList, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetSharkFinSubscriptionDetail returns detailed information on subscribing, for a single product
func (bi *Bitget) GetSharkFinSubscriptionDetail(ctx context.Context, productID int64) (*SharkFinSubDetail, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	vals := url.Values{}
	vals.Set("productId", strconv.FormatInt(productID, 10))
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribeInfo
	var resp struct {
		SharkFinSubDetail `json:"data"`
	}
	return &resp.SharkFinSubDetail, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// SubscribeSharkFin applies funds to a Shark Fin product
func (bi *Bitget) SubscribeSharkFin(ctx context.Context, productID int64, amount float64) (*SaveResp, error) {
	if productID == 0 {
		return nil, errProductIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	req := map[string]any{
		"productId": productID,
		"amount":    strconv.FormatFloat(amount, 'f', -1, 64),
	}
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribe
	var resp struct {
		SaveResp `json:"data"`
	}
	return &resp.SaveResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetSharkFinSubscriptionResult returns the result of a subscription attempt
func (bi *Bitget) GetSharkFinSubscriptionResult(ctx context.Context, orderID int64) (*SaveResult, error) {
	if orderID == 0 {
		return nil, errOrderIDEmpty
	}
	vals := url.Values{}
	vals.Set("orderId", strconv.FormatInt(orderID, 10))
	path := bitgetEarn + bitgetSharkFin + bitgetSubscribeResult
	var resp struct {
		SaveResult `json:"data"`
	}
	return &resp.SaveResult, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate5, http.MethodGet, path, vals, nil, &resp)
}

// GetLoanCurrencyList returns the list of currencies available for loan
func (bi *Bitget) GetLoanCurrencyList(ctx context.Context, currency string) (*LoanCurList, error) {
	if currency == "" {
		return nil, errCurrencyEmpty
	}
	vals := url.Values{}
	vals.Set("coin", currency)
	path := bitgetEarn + bitgetLoan + "/" + bitgetPublic + bitgetCoinInfos
	var resp struct {
		LoanCurList `json:"data"`
	}
	return &resp.LoanCurList, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// GetEstimatedInterestAndBorrowable returns the estimated interest and borrowable amount for a currency
func (bi *Bitget) GetEstimatedInterestAndBorrowable(ctx context.Context, loanCoin, collateralCoin, term string, collateralAmount float64) (*EstimateInterest, error) {
	if loanCoin == "" {
		return nil, errLoanCoinEmpty
	}
	if collateralCoin == "" {
		return nil, errCollateralCoinEmpty
	}
	if term == "" {
		return nil, errTermEmpty
	}
	if collateralAmount == 0 {
		return nil, errCollateralAmountEmpty
	}
	vals := url.Values{}
	vals.Set("loanCoin", loanCoin)
	vals.Set("pledgeCoin", collateralCoin)
	vals.Set("daily", term)
	vals.Set("pledgeAmount", strconv.FormatFloat(collateralAmount, 'f', -1, 64))
	path := bitgetEarn + bitgetLoan + "/" + bitgetPublic + bitgetHourInterest
	var resp struct {
		EstimateInterest `json:"data"`
	}
	return &resp.EstimateInterest, bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate10, path, vals, &resp)
}

// BorrowFunds borrows funds for a currency, supplying a certain amount of currency as collateral
func (bi *Bitget) BorrowFunds(ctx context.Context, loanCoin, collateralCoin, term string, collateralAmount, loanAmount float64) (*BorrowResp, error) {
	if loanCoin == "" {
		return nil, errLoanCoinEmpty
	}
	if collateralCoin == "" {
		return nil, errCollateralCoinEmpty
	}
	if term == "" {
		return nil, errTermEmpty
	}
	if (collateralAmount == 0 && loanAmount == 0) || (collateralAmount != 0 && loanAmount != 0) {
		return nil, errCollateralLoanMutex
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
		BorrowResp `json:"data"`
	}
	return &resp.BorrowResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetOngoingLoans returns the ongoing loans, optionally filtered by currency
func (bi *Bitget) GetOngoingLoans(ctx context.Context, orderID int64, loanCoin, collateralCoin string) ([]OngoingLoans, error) {
	vals := url.Values{}
	if orderID != 0 {
		vals.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	vals.Set("loanCoin", loanCoin)
	vals.Set("pledgeCoin", collateralCoin)
	path := bitgetEarn + bitgetLoan + bitgetOngoingOrders
	var resp struct {
		OngoingLoans []OngoingLoans `json:"data"`
	}
	return resp.OngoingLoans, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, vals, nil, &resp)
}

// RepayLoan repays a loan
func (bi *Bitget) RepayLoan(ctx context.Context, orderID int64, amount float64, repayUnlock, repayAll bool) (*RepayResp, error) {
	if orderID == 0 {
		return nil, errOrderIDEmpty
	}
	if amount == 0 && !repayAll {
		return nil, errAmountEmpty
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
	var resp struct {
		RepayResp `json:"data"`
	}
	return &resp.RepayResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetLoanRepayHistory returns the repayment records for a loan
func (bi *Bitget) GetLoanRepayHistory(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin string, startTime, endTime time.Time) ([]RepayRecords, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin)
	params.Values.Set("pledgeCoin", pledgeCoin)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetRepayHistory
	var resp struct {
		RepayRecords []RepayRecords `json:"data"`
	}
	return resp.RepayRecords, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// ModifyPledgeRate modifies the amount of collateral pledged for a loan
func (bi *Bitget) ModifyPledgeRate(ctx context.Context, orderID int64, amount float64, pledgeCoin, reviseType string) (*ModPledgeResp, error) {
	if orderID == 0 {
		return nil, errOrderIDEmpty
	}
	if amount == 0 {
		return nil, errAmountEmpty
	}
	if pledgeCoin == "" {
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
	var resp struct {
		ModPledgeResp `json:"data"`
	}
	return &resp.ModPledgeResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodPost, path, nil, req, &resp)
}

// GetPledgeRateHistory returns the history of pledged rates for loans
func (bi *Bitget) GetPledgeRateHistory(ctx context.Context, orderID, pagination, limit int64, reviseSide, pledgeCoin string, startTime, endTime time.Time) ([]PledgeRateHist, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("reviseSide", reviseSide)
	params.Values.Set("pledgeCoin", pledgeCoin)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetReviseHistory
	var resp struct {
		PledgeRateHist []PledgeRateHist `json:"data"`
	}
	return resp.PledgeRateHist, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetLoanHistory returns the loan history
func (bi *Bitget) GetLoanHistory(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin, status string, startTime, endTime time.Time) ([]LoanHistory, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin)
	params.Values.Set("pledgeCoin", pledgeCoin)
	params.Values.Set("status", status)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetBorrowHistory
	var resp struct {
		LoanHistory []LoanHistory `json:"data"`
	}
	return resp.LoanHistory, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// GetDebts returns information on current outstanding pledges and loans
func (bi *Bitget) GetDebts(ctx context.Context) (*DebtsResp, error) {
	path := bitgetEarn + bitgetLoan + bitgetDebts
	var resp struct {
		DebtsResp `json:"data"`
	}
	return &resp.DebtsResp, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, nil, nil, &resp)
}

// GetLiquidationRecords returns the liquidation records
func (bi *Bitget) GetLiquidationRecords(ctx context.Context, orderID, pagination, limit int64, loanCoin, pledgeCoin, status string, startTime, endTime time.Time) ([]LiquidRecs, error) {
	var params Params
	params.Values = make(url.Values)
	err := params.prepareDateString(startTime, endTime, false, false)
	if err != nil {
		return nil, err
	}
	params.Values.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Values.Set("loanCoin", loanCoin)
	params.Values.Set("pledgeCoin", pledgeCoin)
	params.Values.Set("status", status)
	if pagination != 0 {
		params.Values.Set("idLessThan", strconv.FormatInt(pagination, 10))
	}
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatInt(limit, 10))
	}
	path := bitgetEarn + bitgetLoan + bitgetReduces
	var resp struct {
		LiquidRecs []LiquidRecs `json:"data"`
	}
	return resp.LiquidRecs, bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate10, http.MethodGet, path, params.Values, nil, &resp)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request
func (bi *Bitget) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, method, path string, queryParams url.Values, bodyParams map[string]any, result any) error {
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := bi.API.Endpoints.GetURL(ep)
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
		headers["ACCESS-SIGN"] = crypto.Base64Encode(hmac)
		headers["ACCESS-TIMESTAMP"] = t
		headers["ACCESS-PASSPHRASE"] = creds.ClientID
		headers["Content-Type"] = "application/json"
		headers["locale"] = "en-US"
		return &request.Item{
			Method:        method,
			Path:          endpoint + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			Verbose:       bi.Verbose,
			HTTPDebugging: bi.HTTPDebugging,
			HTTPRecording: bi.HTTPRecording,
		}, nil
	}
	return bi.SendPayload(ctx, rateLim, newRequest, request.AuthenticatedRequest)
}

// SendHTTPRequest sends an unauthenticated HTTP request, with a few assumptions about the request; namely that it is a GET request with no body
func (bi *Bitget) SendHTTPRequest(ctx context.Context, ep exchange.URL, rateLim request.EndpointLimit, path string, queryParams url.Values, result any) error {
	endpoint, err := bi.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	newRequest := func() (*request.Item, error) {
		path = common.EncodeURLValues(path, queryParams)
		return &request.Item{
			Method:        "GET",
			Path:          endpoint + path,
			Result:        &result,
			Verbose:       bi.Verbose,
			HTTPDebugging: bi.HTTPDebugging,
			HTTPRecording: bi.HTTPRecording,
		}, nil
	}
	return bi.SendPayload(ctx, rateLim, newRequest, request.UnauthenticatedRequest)
}

func (p *Params) prepareDateString(startDate, endDate time.Time, ignoreUnsetStart, ignoreUnsetEnd bool) error {
	if startDate.After(endDate) && !(endDate.IsZero() || endDate.Equal(common.ZeroValueUnix)) {
		return common.ErrStartAfterEnd
	}
	if startDate.Equal(endDate) && !(startDate.IsZero() || startDate.Equal(common.ZeroValueUnix)) {
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

// UnmarshalJSON unmarshals the JSON input into a UnixTimestamp type
func (t *UnixTimestamp) UnmarshalJSON(b []byte) error {
	var timestampStr string
	err := json.Unmarshal(b, &timestampStr)
	if err != nil {
		return err
	}
	if timestampStr == "" {
		*t = UnixTimestamp(time.Time{})
		return nil
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return err
	}
	*t = UnixTimestamp(time.UnixMilli(timestamp).UTC())
	return nil
}

// String implements the stringer interface
func (t *UnixTimestamp) String() string {
	return t.Time().String()
}

// Time returns the time.Time representation of the UnixTimestamp
func (t *UnixTimestamp) Time() time.Time {
	return time.Time(*t)
}

// UnmarshalJSON unmarshals the JSON input into a UnixTimestampNumber type
func (t *UnixTimestampNumber) UnmarshalJSON(b []byte) error {
	var timestampNum uint64
	err := json.Unmarshal(b, &timestampNum)
	if err != nil {
		return err
	}
	*t = UnixTimestampNumber(time.UnixMilli(int64(timestampNum)).UTC())
	return nil
}

// String implements the stringer interface
func (t *UnixTimestampNumber) String() string {
	return t.Time().String()
}

// Time returns the time.Time representation of the UnixTimestampNumber
func (t *UnixTimestampNumber) Time() time.Time {
	return time.Time(*t)
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
	err := json.Unmarshal(b, &success)
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

// CandlestickHelper pulls out common candlestick functionality to avoid repetition
func (bi *Bitget) candlestickHelper(ctx context.Context, pair, granularity, path string, limit uint16, params Params) (*CandleData, error) {
	params.Values.Set("symbol", pair)
	params.Values.Set("granularity", granularity)
	if limit != 0 {
		params.Values.Set("limit", strconv.FormatUint(uint64(limit), 10))
	}
	var resp *CandleResponse
	err := bi.SendHTTPRequest(ctx, exchange.RestSpot, Rate20, path, params.Values, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errNoCandleData
	}
	var spot bool
	var data CandleData
	if resp.Data[0][7] == nil {
		data.FuturesCandles = make([]OneFuturesCandle, len(resp.Data))
	} else {
		spot = true
		data.SpotCandles = make([]OneSpotCandle, len(resp.Data))
	}
	for i := range resp.Data {
		timeTemp, ok := resp.Data[i][0].(string)
		if !ok {
			return nil, errTypeAssertTimestamp
		}
		timeTemp = (timeTemp)[1 : len(timeTemp)-1]
		timeTemp2, err := strconv.ParseInt(timeTemp, 10, 64)
		if err != nil {
			return nil, err
		}
		openTemp, ok := resp.Data[i][1].(string)
		if !ok {
			return nil, errTypeAssertOpenPrice
		}
		highTemp, ok := resp.Data[i][2].(string)
		if !ok {
			return nil, errTypeAssertHighPrice
		}
		lowTemp, ok := resp.Data[i][3].(string)
		if !ok {
			return nil, errTypeAssertLowPrice
		}
		closeTemp, ok := resp.Data[i][4].(string)
		if !ok {
			return nil, errTypeAssertClosePrice
		}
		baseVolumeTemp := resp.Data[i][5].(string)
		if !ok {
			return nil, errTypeAssertBaseVolume
		}
		quoteVolumeTemp := resp.Data[i][6].(string)
		if !ok {
			return nil, errTypeAssertQuoteVolume
		}
		if spot {
			usdtVolumeTemp := resp.Data[i][7].(string)
			if !ok {
				return nil, errTypeAssertUSDTVolume
			}
			data.SpotCandles[i].Timestamp = time.Time(UnixTimestamp(time.UnixMilli(timeTemp2).UTC()))
			data.SpotCandles[i].Open, err = strconv.ParseFloat(openTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].High, err = strconv.ParseFloat(highTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].Low, err = strconv.ParseFloat(lowTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].Close, err = strconv.ParseFloat(closeTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].BaseVolume, err = strconv.ParseFloat(baseVolumeTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].QuoteVolume, err = strconv.ParseFloat(quoteVolumeTemp, 64)
			if err != nil {
				return nil, err
			}
			data.SpotCandles[i].USDTVolume, err = strconv.ParseFloat(usdtVolumeTemp, 64)
			if err != nil {
				return nil, err
			}
		} else {
			data.FuturesCandles[i].Timestamp = time.Time(UnixTimestamp(time.UnixMilli(timeTemp2).UTC()))
			data.FuturesCandles[i].Entry, err = strconv.ParseFloat(openTemp, 64)
			if err != nil {
				return nil, err
			}
			data.FuturesCandles[i].High, err = strconv.ParseFloat(highTemp, 64)
			if err != nil {
				return nil, err
			}
			data.FuturesCandles[i].Low, err = strconv.ParseFloat(lowTemp, 64)
			if err != nil {
				return nil, err
			}
			data.FuturesCandles[i].Exit, err = strconv.ParseFloat(closeTemp, 64)
			if err != nil {
				return nil, err
			}
			data.FuturesCandles[i].BaseVolume, err = strconv.ParseFloat(baseVolumeTemp, 64)
			if err != nil {
				return nil, err
			}
			data.FuturesCandles[i].QuoteVolume, err = strconv.ParseFloat(quoteVolumeTemp, 64)
			if err != nil {
				return nil, err
			}
		}
	}
	return &data, nil
}

// spotOrderHelper is a helper function for unmarshalling spot order endpoints
func (bi *Bitget) spotOrderHelper(ctx context.Context, path string, vals url.Values) ([]SpotOrderDetailData, error) {
	var temp struct {
		Data []OrderDetailTemp `json:"data"`
	}
	err := bi.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, Rate20, http.MethodGet, path, vals, nil, &temp)
	if err != nil {
		return nil, err
	}
	resp := make([]SpotOrderDetailData, len(temp.Data))
	for i := range temp.Data {
		resp[i].UserID = temp.Data[i].UserID
		resp[i].Symbol = temp.Data[i].Symbol
		resp[i].OrderID = temp.Data[i].OrderID
		resp[i].ClientOrderID = temp.Data[i].ClientOrderID
		resp[i].Price = temp.Data[i].Price
		resp[i].Size = temp.Data[i].Size
		resp[i].OrderType = temp.Data[i].OrderType
		resp[i].Side = temp.Data[i].Side
		resp[i].Status = temp.Data[i].Status
		resp[i].PriceAverage = temp.Data[i].PriceAverage
		resp[i].BaseVolume = temp.Data[i].BaseVolume
		resp[i].QuoteVolume = temp.Data[i].QuoteVolume
		resp[i].EnterPointSource = temp.Data[i].EnterPointSource
		resp[i].CreationTime = temp.Data[i].CreationTime
		resp[i].UpdateTime = temp.Data[i].UpdateTime
		resp[i].OrderSource = temp.Data[i].OrderSource
		err = json.Unmarshal(temp.Data[i].FeeDetailTemp, &resp[i].FeeDetail)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}
