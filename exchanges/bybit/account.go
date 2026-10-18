package bybit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	errNoCollateralCoins      = errors.New("no collateral coins provided")
	errDeltaLimitRequired     = errors.New("delta limit required")
	errRepayAmountWithoutCoin = errors.New("repay amount cannot be set without a coin")
)

// GetWalletBalance returns the wallet balance and risk rates of the account and the assets of each coin (Get Wallet
// Balance). Coins without assets or liabilities are left out unless requested
func (e *Exchange) GetWalletBalance(ctx context.Context, req *WalletBalanceRequest) (*WalletBalanceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", req.AccountType)
	if len(req.Coins) != 0 {
		coins, err := joinCoins(req.Coins)
		if err != nil {
			return nil, err
		}
		params.Set("coin", coins)
	}
	var resp *WalletBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/wallet-balance", params, nil, &resp, getWalletBalanceEPL)
}

// GetTransferableAmount returns the amount of each coin that can be transferred out of the unified account (Get
// Transferable Amount (Unified))
func (e *Exchange) GetTransferableAmount(ctx context.Context, coins ...currency.Code) (*TransferableAmountResponse, error) {
	if len(coins) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	coinNames, err := joinCoins(coins)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("coinName", coinNames)
	var resp *TransferableAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/withdrawal", params, nil, &resp, getTransferableAmountEPL)
}

// GetTransactionLog returns the transaction logs of the unified account (Get Transaction Log)
func (e *Exchange) GetTransactionLog(ctx context.Context, req *TransactionLogRequest) (*TransactionLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.AccountType != "" {
		params.Set("accountType", req.AccountType)
	}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.TransactionSubType != "" {
		params.Set("transSubType", req.TransactionSubType)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *TransactionLogResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/transaction-log", params, nil, &resp, getTransactionLogEPL)
}

// GetAccountInfo returns the margin mode and status of the account (Get Account Info)
func (e *Exchange) GetAccountInfo(ctx context.Context) (*AccountInfoResponse, error) {
	var resp *AccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/info", nil, nil, &resp, getAccountInfoEPL)
}

// GetAccountInstrumentsInfo returns the specifications of the instruments the account can trade (Get Account
// Instruments Info)
func (e *Exchange) GetAccountInstrumentsInfo(ctx context.Context, req *AccountInstrumentsInfoRequest) (*AccountInstrumentsInfoResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Category == "" {
		return nil, errCategoryNotSet
	}
	params := url.Values{}
	params.Set("category", req.Category)
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *AccountInstrumentsInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/instruments-info", params, nil, &resp, getAccountInstrumentsInfoEPL)
}

// ManualBorrow borrows a coin at a variable rate in the unified account (Manual Borrow)
func (e *Exchange) ManualBorrow(ctx context.Context, coin currency.Code, amount float64) (*ManualBorrowResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	body := make(map[string]any)
	body["coin"] = coin.Upper().String()
	body["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	var resp *ManualBorrowResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/borrow", nil, body, &resp, manualBorrowEPL)
}

// ManualRepayWithoutAssetConversion repays a coin's liabilities from its spot balance only, without converting other
// assets (Manual Repay Without Asset Conversion)
func (e *Exchange) ManualRepayWithoutAssetConversion(ctx context.Context, req *ManualRepayRequest) (*ManualRepayResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *ManualRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/no-convert-repay", nil, manualRepayBody(req), &resp, manualRepayWithoutAssetConversionEPL)
}

// ManualRepay repays liabilities, converting other assets when the spot balance of the debt coin is not enough
// (Manual Repay)
func (e *Exchange) ManualRepay(ctx context.Context, req *ManualRepayRequest) (*ManualRepayResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Coin.IsEmpty() && req.Amount != 0 {
		return nil, errRepayAmountWithoutCoin
	}
	var resp *ManualRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/repay", nil, manualRepayBody(req), &resp, manualRepayEPL)
}

// manualRepayBody builds the body Manual Repay and Manual Repay Without Asset Conversion share
func manualRepayBody(req *ManualRepayRequest) map[string]any {
	body := map[string]any{}
	if !req.Coin.IsEmpty() {
		body["coin"] = req.Coin.Upper().String()
	}
	if req.Amount != 0 {
		body["amount"] = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	if req.RepaymentType != "" {
		body["repaymentType"] = req.RepaymentType
	}
	return body
}

// GetFeeRate returns the trading fee rates of the account (Get Fee Rate)
func (e *Exchange) GetFeeRate(ctx context.Context, req *FeeRateRequest) (*FeeRateResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if req.Symbol != "" {
		params.Set("symbol", req.Symbol)
	}
	if !req.BaseCoin.IsEmpty() {
		params.Set("baseCoin", req.BaseCoin.Upper().String())
	}
	var resp *FeeRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/fee-rate", params, nil, &resp, getFeeRateEPL)
}

// GetCollateralInfo returns the borrowing and collateral details of each coin in the unified account, of one coin
// when ccy is set (Get Collateral Info)
func (e *Exchange) GetCollateralInfo(ctx context.Context, ccy currency.Code) (*CollateralInfoResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp *CollateralInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/collateral-info", params, nil, &resp, getCollateralInfoEPL)
}

// GetDCPInfo returns the Disconnected-CancelAll-Prevention configuration of each product (Get DCP Info)
func (e *Exchange) GetDCPInfo(ctx context.Context) (*DCPInfoResponse, error) {
	var resp *DCPInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/query-dcp-info", nil, nil, &resp, getDCPInfoEPL)
}

// SetCollateralCoin switches a coin's use as collateral in the unified account on or off (Set Collateral Coin)
func (e *Exchange) SetCollateralCoin(ctx context.Context, coin currency.Code, collateralSwitch bool) error {
	if coin.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	body := map[string]any{
		"coin":             coin.Upper().String(),
		"collateralSwitch": onOff(collateralSwitch),
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-collateral-switch", nil, body, nil, defaultEPL)
}

// SetMarginMode sets the account's margin mode: ISOLATED_MARGIN, REGULAR_MARGIN (cross margin) or PORTFOLIO_MARGIN
// (Set Margin Mode). When Bybit refuses the change, the response holds the reasons it gives and comes with the error
func (e *Exchange) SetMarginMode(ctx context.Context, marginMode string) (*SetMarginModeResponse, error) {
	if marginMode == "" {
		return nil, errInvalidMode
	}
	body := map[string]any{"setMarginMode": marginMode}
	var resp *SetMarginModeResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-margin-mode", nil, body, &resp, setMarginModeEPL); err != nil {
		err = decodeErrorResult(err, &resp)
		return resp, err
	}
	return resp, nil
}

// SetSpotHedging turns spot hedging in portfolio margin on or off (Set Spot Hedging)
func (e *Exchange) SetSpotHedging(ctx context.Context, setHedgingModeOn bool) error {
	body := map[string]any{"setHedgingMode": onOff(setHedgingModeOn)}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-hedging-mode", nil, body, nil, defaultEPL)
}

// GetBorrowHistory returns the interest records of the unified account, newest first (Get Borrow History)
func (e *Exchange) GetBorrowHistory(ctx context.Context, req *BorrowHistoryRequest) (*BorrowHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("currency", req.Currency.Upper().String())
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	var resp *BorrowHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/borrow-history", params, nil, &resp, getBorrowHistoryEPL)
}

// BatchSetCollateralCoin switches the use of several coins as collateral in the unified account on or off (Batch Set
// Collateral Coin)
func (e *Exchange) BatchSetCollateralCoin(ctx context.Context, reqs []SetCollateralCoinRequest) (*BatchSetCollateralCoinResponse, error) {
	if len(reqs) == 0 {
		return nil, errNoCollateralCoins
	}
	items := make([]map[string]any, len(reqs))
	for i := range reqs {
		if reqs[i].Coin.IsEmpty() {
			return nil, fmt.Errorf("%w: request %d", currency.ErrCurrencyCodeEmpty, i)
		}
		items[i] = map[string]any{
			"coin":             reqs[i].Coin.Upper().String(),
			"collateralSwitch": onOff(reqs[i].CollateralSwitch),
		}
	}
	body := map[string]any{"request": items}
	var resp *BatchSetCollateralCoinResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-collateral-switch-batch", nil, body, &resp, batchSetCollateralCoinEPL)
}

// GetCoinGreeks returns the account's option greeks per base coin, of one base coin when baseCoin is set (Get Coin
// Greeks)
func (e *Exchange) GetCoinGreeks(ctx context.Context, baseCoin currency.Code) (*CoinGreeksResponse, error) {
	params := url.Values{}
	if !baseCoin.IsEmpty() {
		params.Set("baseCoin", baseCoin.Upper().String())
	}
	var resp *CoinGreeksResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin-greeks", params, nil, &resp, getCoinGreeksEPL)
}

// GetMMPState returns the Market Maker Protection configuration and state of a base coin (Get MMP State)
func (e *Exchange) GetMMPState(ctx context.Context, baseCoin currency.Code) (*MMPStateResponse, error) {
	if baseCoin.IsEmpty() {
		return nil, errBaseNotSet
	}
	params := url.Values{}
	params.Set("baseCoin", baseCoin.Upper().String())
	var resp *MMPStateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/mmp-state", params, nil, &resp, defaultEPL)
}

// ResetMMP unfreezes a base coin after Market Maker Protection triggers, resetting its accumulated limits (Reset MMP)
func (e *Exchange) ResetMMP(ctx context.Context, baseCoin currency.Code) error {
	if baseCoin.IsEmpty() {
		return errBaseNotSet
	}
	body := map[string]any{"baseCoin": baseCoin.Upper().String()}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", nil, body, nil, resetMMPEPL)
}

// SetMMP configures Market Maker Protection for a base coin's options (Set MMP)
func (e *Exchange) SetMMP(ctx context.Context, req *SetMMPRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.BaseCoin.IsEmpty() {
		return errBaseNotSet
	}
	if req.Window <= 0 {
		return errTimeWindowRequired
	}
	if req.QuantityLimit <= 0 {
		return errQuantityLimitRequired
	}
	if req.DeltaLimit <= 0 {
		return errDeltaLimitRequired
	}
	body := map[string]any{
		"baseCoin":     req.BaseCoin.Upper().String(),
		"window":       strconv.FormatInt(req.Window.Milliseconds(), 10),
		"frozenPeriod": strconv.FormatInt(req.FrozenPeriod.Milliseconds(), 10),
		"qtyLimit":     strconv.FormatFloat(req.QuantityLimit, 'f', -1, 64),
		"deltaLimit":   strconv.FormatFloat(req.DeltaLimit, 'f', -1, 64),
	}
	if req.VegaLimit != 0 {
		body["vegaLimit"] = strconv.FormatFloat(req.VegaLimit, 'f', -1, 64)
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-modify", nil, body, nil, setMMPEPL)
}

// GetSMPGroupID returns the account's Self-Match Prevention group ID, 0 when it has none (Get SMP Group ID)
func (e *Exchange) GetSMPGroupID(ctx context.Context) (*SMPGroupIDResponse, error) {
	var resp *SMPGroupIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/smp-group", nil, nil, &resp, getSMPGroupIDEPL)
}

// GetTradeBehaviourConfig returns how the account treats limit order prices beyond the price limit, and its delta
// neutral, MNT fee deduction and Self-Match Prevention settings (Get Trade Behaviour Config)
func (e *Exchange) GetTradeBehaviourConfig(ctx context.Context) (*TradeBehaviourConfigResponse, error) {
	var resp *TradeBehaviourConfigResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/user-setting-config", nil, nil, &resp, getTradeBehaviourConfigEPL)
}

// SetDeltaNeutralMode turns delta neutral mode on or off (Set Delta Neutral Mode)
func (e *Exchange) SetDeltaNeutralMode(ctx context.Context, enable bool) (*SetDeltaNeutralModeResponse, error) {
	deltaEnable := "0"
	if enable {
		deltaEnable = "1"
	}
	body := map[string]any{"deltaEnable": deltaEnable}
	var resp *SetDeltaNeutralModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-delta-mode", nil, body, &resp, defaultEPL)
}

// SetPriceLimitBehaviour sets whether a limit order priced beyond the price limit has its price adjusted or is
// rejected; linear and inverse each set the behaviour of all futures (Set Price Limit Behaviour)
func (e *Exchange) SetPriceLimitBehaviour(ctx context.Context, category string, modifyEnable bool) error {
	if category == "" {
		return errCategoryNotSet
	}
	body := map[string]any{
		"category":     category,
		"modifyEnable": modifyEnable,
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-limit-px-action", nil, body, nil, setPriceLimitBehaviourEPL)
}

// RepayLiability repays the unified account's liabilities, of one coin when coin is set. It does not repay manual
// borrowing, which ManualRepay and ManualRepayWithoutAssetConversion repay (Repay Liability)
func (e *Exchange) RepayLiability(ctx context.Context, coin currency.Code) (*RepayLiabilityResponse, error) {
	body := map[string]any{}
	if !coin.IsEmpty() {
		body["coin"] = coin.Upper().String()
	}
	var resp *RepayLiabilityResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/quick-repayment", nil, body, &resp, repayLiabilityEPL)
}

// UpgradeToUnifiedAccount upgrades a UTA 2.0 account to UTA 2.0 Pro (Upgrade to Unified Account Pro)
func (e *Exchange) UpgradeToUnifiedAccount(ctx context.Context) (*UnifiedAccountUpgradeResponse, error) {
	var resp *UnifiedAccountUpgradeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/upgrade-to-uta", nil, nil, &resp, upgradeToUnifiedAccountEPL)
}

// GetOptionAssetInfo returns the option profit and loss, delta and margin of each coin (Get Option Asset Info)
func (e *Exchange) GetOptionAssetInfo(ctx context.Context) (*OptionAssetInfoResponse, error) {
	var resp *OptionAssetInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/option-asset-info", nil, nil, &resp, defaultEPL)
}

// GetPayInfo returns the collateral available to repay liabilities, and the borrowing of coin when it is set; coin
// must have outstanding liabilities (Get Pay Info)
func (e *Exchange) GetPayInfo(ctx context.Context, coin currency.Code) (*PayInfoResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *PayInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/pay-info", params, nil, &resp, getPayInfoEPL)
}

// GetTradeInfoForAnalysis returns aggregated spot trade statistics for a symbol (Get Trade Info For Analysis)
func (e *Exchange) GetTradeInfoForAnalysis(ctx context.Context, req *TradeInfoForAnalysisRequest) (*TradeInfoForAnalysisResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolMissing
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *TradeInfoForAnalysisResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/trade-info-for-analysis", params, nil, &resp, getTradeInfoForAnalysisEPL)
}

// joinCoins returns coins as the comma separated, upper case list Bybit takes for several coins
func joinCoins(coins []currency.Code) (string, error) {
	names := make([]string, len(coins))
	for i := range coins {
		if coins[i].IsEmpty() {
			return "", fmt.Errorf("%w: coin %d", currency.ErrCurrencyCodeEmpty, i)
		}
		names[i] = coins[i].Upper().String()
	}
	return strings.Join(names, ","), nil
}

// onOff returns the ON or OFF value Bybit takes for a switch
func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}
