package coinbase

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// ChangeMarginModel changes the margin model as GlobalDerivativesAPI.ChangeMarginModel does, over the websocket
func (w WebsocketAPI) ChangeMarginModel(ctx context.Context, req *DerivativesChangeMarginModelRequest) ([]DerivativesMarginModelChangeResponse, error) {
	return w.e.GlobalDerivatives().changeMarginModel(ctx, true, req)
}

// GetAccessLog returns a page of the account's access log as GlobalDerivativesAPI.GetAccessLog does, over the
// websocket; the gateway blocks the HTTP form, so this is the form that may reach it
func (w WebsocketAPI) GetAccessLog(ctx context.Context, req *DerivativesGetAccessLogRequest) (*DerivativesAccessLogResponse, error) {
	return w.e.GlobalDerivatives().getAccessLog(ctx, true, req)
}

// GetAccountSummaries returns the account's summaries as GlobalDerivativesAPI.GetAccountSummaries does, over the
// websocket
func (w WebsocketAPI) GetAccountSummaries(ctx context.Context, req *DerivativesGetAccountSummariesRequest) (*DerivativesAccountSummariesResponse, error) {
	return w.e.GlobalDerivatives().getAccountSummaries(ctx, true, req)
}

// GetAccountSummary returns the account's summary in one currency as GlobalDerivativesAPI.GetAccountSummary does, over
// the websocket
func (w WebsocketAPI) GetAccountSummary(ctx context.Context, req *DerivativesGetAccountSummaryRequest) (*DerivativesAccountSummaryResponse, error) {
	return w.e.GlobalDerivatives().getAccountSummary(ctx, true, req)
}

// GetLeverage returns futures' custom and maximum leverage as GlobalDerivativesAPI.GetLeverage does, over the websocket
func (w WebsocketAPI) GetLeverage(ctx context.Context, req *DerivativesGetLeverageRequest) ([]DerivativesLeverageResponse, error) {
	return w.e.GlobalDerivatives().getLeverage(ctx, true, req)
}

// GetPMEParams returns the account's portfolio margin restrictions as GlobalDerivativesAPI.GetPMEParams does, over the
// websocket
func (w WebsocketAPI) GetPMEParams(ctx context.Context, ccy currency.Code) (*DerivativesPMEParamsResponse, error) {
	return w.e.GlobalDerivatives().getPMEParams(ctx, true, ccy)
}

// GetPosition returns the account's position in an instrument as GlobalDerivativesAPI.GetPosition does, over the
// websocket
func (w WebsocketAPI) GetPosition(ctx context.Context, req *DerivativesGetPositionRequest) (*DerivativesPositionResponse, error) {
	return w.e.GlobalDerivatives().getPosition(ctx, true, req)
}

// GetPositions returns the open positions as GlobalDerivativesAPI.GetPositions does, over the websocket
func (w WebsocketAPI) GetPositions(ctx context.Context, req *DerivativesGetPositionsRequest) ([]DerivativesPositionResponse, error) {
	return w.e.GlobalDerivatives().getPositions(ctx, true, req)
}

// GetRiskProfile returns the account's risk profile as GlobalDerivativesAPI.GetRiskProfile does, over the websocket
func (w WebsocketAPI) GetRiskProfile(ctx context.Context) (*DerivativesRiskProfileResponse, error) {
	return w.e.GlobalDerivatives().getRiskProfile(ctx, true)
}

// GetSubaccounts returns the account's and its subaccounts' details as GlobalDerivativesAPI.GetSubaccounts does, over
// the websocket
func (w WebsocketAPI) GetSubaccounts(ctx context.Context, req *DerivativesGetSubaccountsRequest) ([]DerivativesSubaccountResponse, error) {
	return w.e.GlobalDerivatives().getSubaccounts(ctx, true, req)
}

// GetSubaccountsDetails returns the accounts' positions as GlobalDerivativesAPI.GetSubaccountsDetails does, over the
// websocket
func (w WebsocketAPI) GetSubaccountsDetails(ctx context.Context, req *DerivativesGetSubaccountsDetailsRequest) ([]DerivativesSubaccountDetailsResponse, error) {
	return w.e.GlobalDerivatives().getSubaccountsDetails(ctx, true, req)
}

// GetTradingLimits returns the account's trading limits as GlobalDerivativesAPI.GetTradingLimits does, over the
// websocket
func (w WebsocketAPI) GetTradingLimits(ctx context.Context, ccy currency.Code) (*DerivativesTradingLimitsResponse, error) {
	return w.e.GlobalDerivatives().getTradingLimits(ctx, true, ccy)
}

// GetTransactionLog returns a page of the account's transaction log as GlobalDerivativesAPI.GetTransactionLog does,
// over the websocket
func (w WebsocketAPI) GetTransactionLog(ctx context.Context, req *DerivativesGetTransactionLogRequest) (*DerivativesTransactionLogResponse, error) {
	return w.e.GlobalDerivatives().getTransactionLog(ctx, true, req)
}

// GetUserLocks returns the account's locks as GlobalDerivativesAPI.GetUserLocks does, over the websocket
func (w WebsocketAPI) GetUserLocks(ctx context.Context) ([]DerivativesUserLockResponse, error) {
	return w.e.GlobalDerivatives().getUserLocks(ctx, true)
}

// PMESimulate calculates a portfolio margin simulation as GlobalDerivativesAPI.PMESimulate does, over the websocket
func (w WebsocketAPI) PMESimulate(ctx context.Context, req *DerivativesSimulatePortfolioRequest) (*DerivativesPMESimulateResponse, error) {
	return w.e.GlobalDerivatives().pmeSimulate(ctx, true, req)
}

// SimulatePortfolio calculates the margins of a simulated portfolio as GlobalDerivativesAPI.SimulatePortfolio does,
// over the websocket
func (w WebsocketAPI) SimulatePortfolio(ctx context.Context, req *DerivativesSimulatePortfolioRequest) (*DerivativesSimulatePortfolioResponse, error) {
	return w.e.GlobalDerivatives().simulatePortfolio(ctx, true, req)
}

// GetAnnouncements returns the platform's announcements as GlobalDerivativesAPI.GetAnnouncements does, over the
// websocket
func (w WebsocketAPI) GetAnnouncements(ctx context.Context, req *DerivativesGetAnnouncementsRequest) ([]DerivativesAnnouncementResponse, error) {
	return w.e.GlobalDerivatives().getAnnouncements(ctx, true, req)
}
