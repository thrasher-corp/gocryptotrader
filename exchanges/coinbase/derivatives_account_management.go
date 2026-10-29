package coinbase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Gateway margin models, chosen per portfolio: cross margin pools collateral across positions while segregated margin
// keeps it apart, and portfolio margin nets related positions' risk while standard margin sizes each position alone
const (
	DerivativesMarginModelCrossPortfolio      = "cross_pm"
	DerivativesMarginModelCrossStandard       = "cross_sm"
	DerivativesMarginModelSegregatedPortfolio = "segregated_pm"
	DerivativesMarginModelSegregatedStandard  = "segregated_sm"
)

// Gateway account types
const (
	DerivativesAccountTypeMain       = "main"
	DerivativesAccountTypeSubaccount = "subaccount"
)

// Gateway self trading prevention modes, which reject the incoming order or cancel the resting order it would trade
// with
const (
	DerivativesSelfTradingModeRejectTaker = "reject_taker"
	DerivativesSelfTradingModeCancelMaker = "cancel_maker"
)

// DerivativesPositionDirectionZero is a closed position's direction; open positions are buy or sell
const DerivativesPositionDirectionZero = "zero"

// Gateway trade roles, as the transaction log reports a trade's role and the role its fee was charged at
const (
	DerivativesTradeRoleMaker = "maker"
	DerivativesTradeRoleTaker = "taker"
)

// Gateway position move roles, which tell the liquidated account's transaction log entry from the receiving
// account's for an LSP, ADL or liquidation transfer
const (
	DerivativesPositionMoveRoleSource      = "source"
	DerivativesPositionMoveRoleDestination = "destination"
)

var (
	errDerivativesInvalidMarginModel     = errors.New("invalid gateway margin model")
	errDerivativesInvalidAccountCurrency = errors.New("invalid gateway account currency")
	errDerivativesInvalidPositionKind    = errors.New("invalid gateway position kind")
)

var (
	// derivativesAccountCurrencies are the currencies most gateway account and wallet methods take
	derivativesAccountCurrencies = []currency.Code{currency.BTC, currency.ETH, currency.USDC, currency.USDT, currency.EURR}
	// derivativesWalletCurrencies are the currencies private/get_account_summary and private/get_transaction_log take
	derivativesWalletCurrencies = []currency.Code{
		currency.BTC, currency.ETH, currency.STETH, currency.ETHW, currency.USDC, currency.USDT, currency.EURR, currency.SOL,
		currency.XRP, currency.NewCode("USYC"), currency.PAXG, currency.BNB, currency.USDE,
	}
	// derivativesPMECurrencies are the currencies private/pme/simulate takes, CROSS simulating the cross collateral
	// portfolio
	derivativesPMECurrencies = []currency.Code{currency.BTC, currency.ETH, currency.USDC, currency.USDT, currency.NewCode("CROSS")}
)

// derivativesAccountCurrency checks a currency is one a gateway account method documents, and returns it in the upper
// case the gateway takes
func derivativesAccountCurrency(ccy currency.Code, documented []currency.Code) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if !slices.ContainsFunc(documented, ccy.Equal) {
		return "", fmt.Errorf("%w: %s", errDerivativesInvalidAccountCurrency, ccy)
	}
	return ccy.Upper().String(), nil
}

// ChangeMarginModel changes the margin model of the account or a subaccount, which changes its margin requirements,
// available funds and trading capabilities, and returns the change's effect on each currency; a dry run returns the
// effect without making the change
func (g GlobalDerivativesAPI) ChangeMarginModel(ctx context.Context, req *DerivativesChangeMarginModelRequest) ([]DerivativesMarginModelChangeResponse, error) {
	return g.changeMarginModel(ctx, false, req)
}

func (g GlobalDerivativesAPI) changeMarginModel(ctx context.Context, overWebsocket bool, req *DerivativesChangeMarginModelRequest) ([]DerivativesMarginModelChangeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch req.MarginModel {
	case DerivativesMarginModelCrossPortfolio, DerivativesMarginModelCrossStandard, DerivativesMarginModelSegregatedPortfolio, DerivativesMarginModelSegregatedStandard:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidMarginModel, req.MarginModel)
	}
	params := &DerivativesChangeMarginModelWireRequest{UserID: req.UserID, MarginModel: req.MarginModel, DryRun: req.DryRun}
	var resp []DerivativesMarginModelChangeResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/change_margin_model", params, &resp)
}

// GetAccessLog returns a page of the account's access log, newest first: its logins and the changes to its security
// and API keys. As of October 2026 the gateway blocks it over HTTP, answering HTTP 403 with an HTML page, so only
// WebsocketAPI.GetAccessLog may reach it
func (g GlobalDerivativesAPI) GetAccessLog(ctx context.Context, req *DerivativesGetAccessLogRequest) (*DerivativesAccessLogResponse, error) {
	return g.getAccessLog(ctx, false, req)
}

func (g GlobalDerivativesAPI) getAccessLog(ctx context.Context, overWebsocket bool, req *DerivativesGetAccessLogRequest) (*DerivativesAccessLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp *DerivativesAccessLogResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_access_log", &DerivativesAccessLogWireRequest{Offset: req.Offset, Count: req.Count}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountSummaries returns the account's or a subaccount's summary in every currency it holds, and the active
// isolated margin subaccounts' summaries when asked for; it takes no currency, as it always returns every currency
func (g GlobalDerivativesAPI) GetAccountSummaries(ctx context.Context, req *DerivativesGetAccountSummariesRequest) (*DerivativesAccountSummariesResponse, error) {
	return g.getAccountSummaries(ctx, false, req)
}

func (g GlobalDerivativesAPI) getAccountSummaries(ctx context.Context, overWebsocket bool, req *DerivativesGetAccountSummariesRequest) (*DerivativesAccountSummariesResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := &DerivativesAccountSummariesWireRequest{SubaccountID: req.SubaccountID, Extended: req.Extended, IncludeIsolated: req.IncludeIsolated}
	var resp *DerivativesAccountSummariesResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_account_summaries", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountSummary returns the account's or a subaccount's summary in one currency: its cash balance, equity,
// margins, profit and loss, greeks and rate limits. Under standard margin the margin balance is the equity less the
// options' value, under portfolio margin the equity; the session fields reset at each daily settlement
func (g GlobalDerivativesAPI) GetAccountSummary(ctx context.Context, req *DerivativesGetAccountSummaryRequest) (*DerivativesAccountSummaryResponse, error) {
	return g.getAccountSummary(ctx, false, req)
}

func (g GlobalDerivativesAPI) getAccountSummary(ctx context.Context, overWebsocket bool, req *DerivativesGetAccountSummaryRequest) (*DerivativesAccountSummaryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ccy, err := derivativesAccountCurrency(req.Currency, derivativesWalletCurrencies)
	if err != nil {
		return nil, err
	}
	params := &DerivativesAccountSummaryWireRequest{Currency: ccy, SubaccountID: req.SubaccountID, Extended: req.Extended}
	var resp *DerivativesAccountSummaryResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_account_summary", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLeverage returns a future's custom and maximum leverage or, without an instrument, those of every future with a
// custom leverage, leaving out futures at their maximum. A regular subaccount reading isolated margin leverage, which
// only the main account sees, receives none
func (g GlobalDerivativesAPI) GetLeverage(ctx context.Context, req *DerivativesGetLeverageRequest) ([]DerivativesLeverageResponse, error) {
	return g.getLeverage(ctx, false, req)
}

func (g GlobalDerivativesAPI) getLeverage(ctx context.Context, overWebsocket bool, req *DerivativesGetLeverageRequest) ([]DerivativesLeverageResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := &DerivativesLeverageWireRequest{InstrumentName: req.InstrumentName, SubaccountID: req.SubaccountID, Isolated: req.Isolated}
	var resp DerivativesLeverages
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_leverage", params, &resp)
}

// GetPMEParams returns the account's portfolio margin open order restrictions and risk limits in BTC, ETH, USDC, USDT
// or EURR; with portfolio margin disabled, only Enabled is set
func (g GlobalDerivativesAPI) GetPMEParams(ctx context.Context, ccy currency.Code) (*DerivativesPMEParamsResponse, error) {
	return g.getPMEParams(ctx, false, ccy)
}

func (g GlobalDerivativesAPI) getPMEParams(ctx context.Context, overWebsocket bool, ccy currency.Code) (*DerivativesPMEParamsResponse, error) {
	code, err := derivativesAccountCurrency(ccy, derivativesAccountCurrencies)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesPMEParamsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_pme_params", &DerivativesCurrencyWireRequest{Currency: code}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPosition returns the account's position in an instrument, which has a zero size when there is none. With
// IncludeIsolated, an instrument bound to an isolated margin subaccount returns that subaccount's position in place of
// the cross margin one; GetPositions returns both
func (g GlobalDerivativesAPI) GetPosition(ctx context.Context, req *DerivativesGetPositionRequest) (*DerivativesPositionResponse, error) {
	return g.getPosition(ctx, false, req)
}

func (g GlobalDerivativesAPI) getPosition(ctx context.Context, overWebsocket bool, req *DerivativesGetPositionRequest) (*DerivativesPositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.InstrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *DerivativesPositionResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_position", &DerivativesPositionWireRequest{InstrumentName: req.InstrumentName, IncludeIsolated: req.IncludeIsolated}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPositions returns the open positions of the account or a subaccount, optionally in one settlement currency and of
// one kind; spot holdings are balances, which GetAccountSummary reports. Isolated margin positions are owned by their
// subaccounts, so they are only returned with IncludeIsolated or the subaccount's ID
func (g GlobalDerivativesAPI) GetPositions(ctx context.Context, req *DerivativesGetPositionsRequest) ([]DerivativesPositionResponse, error) {
	return g.getPositions(ctx, false, req)
}

func (g GlobalDerivativesAPI) getPositions(ctx context.Context, overWebsocket bool, req *DerivativesGetPositionsRequest) ([]DerivativesPositionResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := &DerivativesPositionsWireRequest{Kind: req.Kind, SubaccountID: req.SubaccountID, IncludeIsolated: req.IncludeIsolated}
	switch {
	case req.Currency.IsEmpty():
	case strings.EqualFold(req.Currency.String(), DerivativesCurrencyAny):
		params.Currency = DerivativesCurrencyAny
	default:
		ccy, err := derivativesAccountCurrency(req.Currency, derivativesAccountCurrencies)
		if err != nil {
			return nil, err
		}
		params.Currency = ccy
	}
	switch req.Kind {
	case "", DerivativesKindFuture, DerivativesKindOption, DerivativesKindFutureCombo, DerivativesKindOptionCombo:
	default:
		return nil, fmt.Errorf("%w: %q", errDerivativesInvalidPositionKind, req.Kind)
	}
	var resp []DerivativesPositionResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_positions", params, &resp)
}

// GetRiskProfile returns the account's risk profile: whether it may use portfolio margin or sell options, and its
// leverage ceiling
func (g GlobalDerivativesAPI) GetRiskProfile(ctx context.Context) (*DerivativesRiskProfileResponse, error) {
	return g.getRiskProfile(ctx, false)
}

func (g GlobalDerivativesAPI) getRiskProfile(ctx context.Context, overWebsocket bool) (*DerivativesRiskProfileResponse, error) {
	var resp *DerivativesRiskProfileResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_risk_profile", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetSubaccounts returns the main account's and its subaccounts' details, with their portfolios when asked for; a
// subaccount is sent limited details of the main account. Isolated margin subaccounts are created on an account's first
// isolated order, recycled once flat and hidden unless asked for, so their bindings are to be read afresh, not cached
func (g GlobalDerivativesAPI) GetSubaccounts(ctx context.Context, req *DerivativesGetSubaccountsRequest) ([]DerivativesSubaccountResponse, error) {
	return g.getSubaccounts(ctx, false, req)
}

func (g GlobalDerivativesAPI) getSubaccounts(ctx context.Context, overWebsocket bool, req *DerivativesGetSubaccountsRequest) ([]DerivativesSubaccountResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	var resp []DerivativesSubaccountResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_subaccounts", &DerivativesSubaccountsWireRequest{WithPortfolio: req.WithPortfolio, IncludeIsolated: req.IncludeIsolated}, &resp)
}

// GetSubaccountsDetails returns the positions of the main account and each subaccount in a currency and, when asked
// for, their open orders
func (g GlobalDerivativesAPI) GetSubaccountsDetails(ctx context.Context, req *DerivativesGetSubaccountsDetailsRequest) ([]DerivativesSubaccountDetailsResponse, error) {
	return g.getSubaccountsDetails(ctx, false, req)
}

func (g GlobalDerivativesAPI) getSubaccountsDetails(ctx context.Context, overWebsocket bool, req *DerivativesGetSubaccountsDetailsRequest) ([]DerivativesSubaccountDetailsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ccy, err := derivativesAccountCurrency(req.Currency, derivativesAccountCurrencies)
	if err != nil {
		return nil, err
	}
	params := &DerivativesSubaccountsDetailsWireRequest{Currency: ccy, WithOpenOrders: req.WithOpenOrders, IncludeIsolated: req.IncludeIsolated}
	var resp []DerivativesSubaccountDetailsResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_subaccounts_details", params, &resp)
}

// GetTradingLimits returns the account's open order and position limits in BTC, ETH, USDC, USDT or EURR under its
// margin model
func (g GlobalDerivativesAPI) GetTradingLimits(ctx context.Context, ccy currency.Code) (*DerivativesTradingLimitsResponse, error) {
	return g.getTradingLimits(ctx, false, ccy)
}

func (g GlobalDerivativesAPI) getTradingLimits(ctx context.Context, overWebsocket bool, ccy currency.Code) (*DerivativesTradingLimitsResponse, error) {
	code, err := derivativesAccountCurrency(ccy, derivativesAccountCurrencies)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesTradingLimitsResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_trading_limits", &DerivativesCurrencyWireRequest{Currency: code}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetTransactionLog returns a page of the account's transaction log in a currency, newest first, back to the
// account's creation: its trades, deposits, withdrawals, transfers, settlements, fees and other balance changes. The
// gateway allows it about one request a second
func (g GlobalDerivativesAPI) GetTransactionLog(ctx context.Context, req *DerivativesGetTransactionLogRequest) (*DerivativesTransactionLogResponse, error) {
	return g.getTransactionLog(ctx, false, req)
}

func (g GlobalDerivativesAPI) getTransactionLog(ctx context.Context, overWebsocket bool, req *DerivativesGetTransactionLogRequest) (*DerivativesTransactionLogResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ccy, err := derivativesAccountCurrency(req.Currency, derivativesWalletCurrencies)
	if err != nil {
		return nil, err
	}
	// The gateway refuses a request without either end of the window
	if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	startTimestamp, endTimestamp, err := derivativesUnixMilliRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	params := &DerivativesTransactionLogWireRequest{
		Currency:       ccy,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Query:          req.Query,
		Count:          req.Count,
		SubaccountID:   req.SubaccountID,
		Continuation:   req.Continuation,
	}
	var resp *DerivativesTransactionLogResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesTransactionLogEPL, "private/get_transaction_log", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserLocks returns whether the account is locked in each currency, and why; a lock may stop trading, withdrawals
// or other operations
func (g GlobalDerivativesAPI) GetUserLocks(ctx context.Context) ([]DerivativesUserLockResponse, error) {
	return g.getUserLocks(ctx, false)
}

func (g GlobalDerivativesAPI) getUserLocks(ctx context.Context, overWebsocket bool) ([]DerivativesUserLockResponse, error) {
	var resp []DerivativesUserLockResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "private/get_user_locks", nil, &resp)
}

// PMESimulate calculates a portfolio margin account's extended risk matrix and margins in BTC, ETH, USDC or USDT, or
// for its cross collateral portfolio with the currency CROSS, optionally with simulated positions sized in the base
// currency
func (g GlobalDerivativesAPI) PMESimulate(ctx context.Context, req *DerivativesSimulatePortfolioRequest) (*DerivativesPMESimulateResponse, error) {
	return g.pmeSimulate(ctx, false, req)
}

func (g GlobalDerivativesAPI) pmeSimulate(ctx context.Context, overWebsocket bool, req *DerivativesSimulatePortfolioRequest) (*DerivativesPMESimulateResponse, error) {
	params, err := derivativesSimulatePortfolio(req, derivativesPMECurrencies)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesPMESimulateResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesSimulatePortfolioEPL, "private/pme/simulate", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SimulatePortfolio calculates the margins of the account's portfolio in BTC, ETH, USDC, USDT or EURR, optionally with
// simulated positions sized in USD for futures and in the base currency for options; the gateway allows it once a
// second
func (g GlobalDerivativesAPI) SimulatePortfolio(ctx context.Context, req *DerivativesSimulatePortfolioRequest) (*DerivativesSimulatePortfolioResponse, error) {
	return g.simulatePortfolio(ctx, false, req)
}

func (g GlobalDerivativesAPI) simulatePortfolio(ctx context.Context, overWebsocket bool, req *DerivativesSimulatePortfolioRequest) (*DerivativesSimulatePortfolioResponse, error) {
	params, err := derivativesSimulatePortfolio(req, derivativesAccountCurrencies)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesSimulatePortfolioResponse
	if err := g.sendRequest(ctx, overWebsocket, derivativesSimulatePortfolioEPL, "private/simulate_portfolio", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// derivativesSimulatePortfolio validates a portfolio simulation request against the currencies its method documents
// and returns its params
func derivativesSimulatePortfolio(req *DerivativesSimulatePortfolioRequest, documented []currency.Code) (*DerivativesSimulatePortfolioWireRequest, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ccy, err := derivativesAccountCurrency(req.Currency, documented)
	if err != nil {
		return nil, err
	}
	return &DerivativesSimulatePortfolioWireRequest{Currency: ccy, AddPositions: req.AddPositions, SimulatedPositions: req.SimulatedPositions}, nil
}

// GetAnnouncements returns the platform's announcements published up to a time, newest first; the gateway returns at
// most 50, whatever the count
func (g GlobalDerivativesAPI) GetAnnouncements(ctx context.Context, req *DerivativesGetAnnouncementsRequest) ([]DerivativesAnnouncementResponse, error) {
	return g.getAnnouncements(ctx, false, req)
}

func (g GlobalDerivativesAPI) getAnnouncements(ctx context.Context, overWebsocket bool, req *DerivativesGetAnnouncementsRequest) ([]DerivativesAnnouncementResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	startTimestamp, err := derivativesUnixMilli(req.StartTime)
	if err != nil {
		return nil, err
	}
	params := &DerivativesAnnouncementsWireRequest{Count: req.Count, StartTimestamp: startTimestamp}
	var resp []DerivativesAnnouncementResponse
	return resp, g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, "public/get_announcements", params, &resp)
}
