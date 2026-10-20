package deribit

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// ChangeAPIKeyName renames an API key as Exchange.ChangeAPIKeyName does, over the websocket
func (w WebsocketAPI) ChangeAPIKeyName(ctx context.Context, arg *ChangeAPIKeyNameRequest) (*APIKey, error) {
	return w.e.changeAPIKeyName(ctx, arg, true)
}

// ChangeMarginModel changes the margin model as Exchange.ChangeMarginModel does, over the websocket
func (w WebsocketAPI) ChangeMarginModel(ctx context.Context, arg *ChangeMarginModelRequest) ([]MarginModelChange, error) {
	return w.e.changeMarginModel(ctx, arg, true)
}

// ChangeScopeInAPIKey changes an API key's maximum scope as Exchange.ChangeScopeInAPIKey does, over the websocket
func (w WebsocketAPI) ChangeScopeInAPIKey(ctx context.Context, arg *ChangeScopeInAPIKeyRequest) (*APIKey, error) {
	return w.e.changeScopeInAPIKey(ctx, arg, true)
}

// ChangeSubaccountName renames a subaccount as Exchange.ChangeSubaccountName does, over the websocket
func (w WebsocketAPI) ChangeSubaccountName(ctx context.Context, arg *ChangeSubaccountNameRequest) error {
	return w.e.changeSubaccountName(ctx, arg, true)
}

// CreateAPIKey creates an API key as Exchange.CreateAPIKey does, over the websocket
func (w WebsocketAPI) CreateAPIKey(ctx context.Context, arg *CreateAPIKeyRequest) (*APIKey, error) {
	return w.e.createAPIKey(ctx, arg, true)
}

// CreateSubaccount creates a subaccount as Exchange.CreateSubaccount does, over the websocket
func (w WebsocketAPI) CreateSubaccount(ctx context.Context) (*Subaccount, error) {
	return w.e.createSubaccount(ctx, true)
}

// DeleteMember deletes a Direct Access member as Exchange.DeleteMember does, over the websocket
func (w WebsocketAPI) DeleteMember(ctx context.Context, memberID uint64) (*MembersResponse, error) {
	return w.e.deleteMember(ctx, memberID, true)
}

// DisableAPIKey disables an API key as Exchange.DisableAPIKey does, over the websocket
func (w WebsocketAPI) DisableAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return w.e.disableAPIKey(ctx, id, true)
}

// EditAPIKey edits an API key as Exchange.EditAPIKey does, over the websocket
func (w WebsocketAPI) EditAPIKey(ctx context.Context, arg *EditAPIKeyRequest) (*APIKey, error) {
	return w.e.editAPIKey(ctx, arg, true)
}

// EnableAffiliateProgram enables the account's affiliate program as Exchange.EnableAffiliateProgram does, over the
// websocket
func (w WebsocketAPI) EnableAffiliateProgram(ctx context.Context) error {
	return w.e.sendOKRequest(ctx, true, "private/enable_affiliate_program", nil)
}

// EnableAPIKey enables an API key as Exchange.EnableAPIKey does, over the websocket
func (w WebsocketAPI) EnableAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return w.e.enableAPIKey(ctx, id, true)
}

// GetAccessLog returns a page of the account's access log as Exchange.GetAccessLog does, over the websocket
func (w WebsocketAPI) GetAccessLog(ctx context.Context, arg *GetAccessLogRequest) (*AccessLogResponse, error) {
	return w.e.getAccessLog(ctx, arg, true)
}

// GetAccountSummaries returns the account's summaries as Exchange.GetAccountSummaries does, over the websocket
func (w WebsocketAPI) GetAccountSummaries(ctx context.Context, arg *GetAccountSummariesRequest) (*AccountSummariesResponse, error) {
	return w.e.getAccountSummaries(ctx, arg, true)
}

// GetAccountSummary returns the account's summary in one currency as Exchange.GetAccountSummary does, over the
// websocket
func (w WebsocketAPI) GetAccountSummary(ctx context.Context, arg *GetAccountSummaryRequest) (*AccountSummaryResponse, error) {
	return w.e.getAccountSummary(ctx, arg, true)
}

// GetAffiliateProgramInfo returns the account's affiliate program information as Exchange.GetAffiliateProgramInfo does,
// over the websocket
func (w WebsocketAPI) GetAffiliateProgramInfo(ctx context.Context) (*AffiliateProgramInfoResponse, error) {
	return w.e.getAffiliateProgramInfo(ctx, true)
}

// GetAccountCurrencies returns the currencies available to the account as Exchange.GetAccountCurrencies does, over the
// websocket
func (w WebsocketAPI) GetAccountCurrencies(ctx context.Context) ([]AccountCurrency, error) {
	return w.e.getAccountCurrencies(ctx, true)
}

// GetEmailLanguage returns the language of the account's notification emails as Exchange.GetEmailLanguage does, over
// the websocket
func (w WebsocketAPI) GetEmailLanguage(ctx context.Context) (string, error) {
	return w.e.getEmailLanguage(ctx, true)
}

// GetMembers returns the account's Direct Access members as Exchange.GetMembers does, over the websocket
func (w WebsocketAPI) GetMembers(ctx context.Context) (*MembersResponse, error) {
	return w.e.getMembers(ctx, true)
}

// GetNewAnnouncements returns the account's unread announcements as Exchange.GetNewAnnouncements does, over the
// websocket
func (w WebsocketAPI) GetNewAnnouncements(ctx context.Context) ([]Announcement, error) {
	return w.e.getNewAnnouncements(ctx, true)
}

// GetPosition returns the account's position in an instrument as Exchange.GetPosition does, over the websocket
func (w WebsocketAPI) GetPosition(ctx context.Context, instrumentName string) (*Position, error) {
	return w.e.getPosition(ctx, instrumentName, true)
}

// GetPositions returns the open positions as Exchange.GetPositions does, over the websocket
func (w WebsocketAPI) GetPositions(ctx context.Context, arg *GetPositionsRequest) ([]Position, error) {
	return w.e.getPositions(ctx, arg, true)
}

// GetSubaccounts returns the account's and its subaccounts' details as Exchange.GetSubaccounts does, over the websocket
func (w WebsocketAPI) GetSubaccounts(ctx context.Context, withPortfolio bool) ([]Subaccount, error) {
	return w.e.getSubaccounts(ctx, withPortfolio, true)
}

// GetSubaccountsDetails returns the accounts' positions as Exchange.GetSubaccountsDetails does, over the websocket
func (w WebsocketAPI) GetSubaccountsDetails(ctx context.Context, arg *GetSubaccountsDetailsRequest) ([]SubaccountDetails, error) {
	return w.e.getSubaccountsDetails(ctx, arg, true)
}

// GetTransactionLog returns a page of the account's transaction log as Exchange.GetTransactionLog does, over the
// websocket
func (w WebsocketAPI) GetTransactionLog(ctx context.Context, arg *GetTransactionLogRequest) (*TransactionLogResponse, error) {
	return w.e.getTransactionLog(ctx, arg, true)
}

// GetUserLocks returns the account's locks as Exchange.GetUserLocks does, over the websocket
func (w WebsocketAPI) GetUserLocks(ctx context.Context) ([]UserLock, error) {
	return w.e.getUserLocks(ctx, true)
}

// ListAPIKeys returns the account's API keys as Exchange.ListAPIKeys does, over the websocket
func (w WebsocketAPI) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	return w.e.listAPIKeys(ctx, true)
}

// ListCustodyAccounts returns the account's custody accounts as Exchange.ListCustodyAccounts does, over the websocket
func (w WebsocketAPI) ListCustodyAccounts(ctx context.Context, ccy currency.Code) ([]CustodyAccount, error) {
	return w.e.listCustodyAccounts(ctx, ccy, true)
}

// PMESimulate calculates a portfolio margin simulation as Exchange.PMESimulate does, over the websocket
func (w WebsocketAPI) PMESimulate(ctx context.Context, arg *SimulatePortfolioRequest) (*PMESimulateResponse, error) {
	return w.e.pmeSimulate(ctx, arg, true)
}

// RemoveAPIKey removes an API key as Exchange.RemoveAPIKey does, over the websocket
func (w WebsocketAPI) RemoveAPIKey(ctx context.Context, id uint64) error {
	return w.e.removeAPIKey(ctx, id, true)
}

// RemoveSubaccount removes a subaccount as Exchange.RemoveSubaccount does, over the websocket
func (w WebsocketAPI) RemoveSubaccount(ctx context.Context, subaccountID uint64) error {
	return w.e.removeSubaccount(ctx, subaccountID, true)
}

// ResetAPIKey replaces an API key's secret as Exchange.ResetAPIKey does, over the websocket
func (w WebsocketAPI) ResetAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return w.e.resetAPIKey(ctx, id, true)
}

// SetAnnouncementAsRead marks an announcement read as Exchange.SetAnnouncementAsRead does, over the websocket
func (w WebsocketAPI) SetAnnouncementAsRead(ctx context.Context, announcementID uint64) error {
	return w.e.setAnnouncementAsRead(ctx, announcementID, true)
}

// SetDisabledTradingProducts sets a subaccount's disabled trading products as Exchange.SetDisabledTradingProducts does,
// over the websocket
func (w WebsocketAPI) SetDisabledTradingProducts(ctx context.Context, arg *SetDisabledTradingProductsRequest) error {
	return w.e.setDisabledTradingProducts(ctx, arg, true)
}

// SetEmailForSubaccount sets the email address of a subaccount as Exchange.SetEmailForSubaccount does, over the
// websocket
func (w WebsocketAPI) SetEmailForSubaccount(ctx context.Context, arg *SetEmailForSubaccountRequest) error {
	return w.e.setEmailForSubaccount(ctx, arg, true)
}

// SetEmailLanguage sets the language of the account's notification emails as Exchange.SetEmailLanguage does, over the
// websocket
func (w WebsocketAPI) SetEmailLanguage(ctx context.Context, language string) error {
	return w.e.setEmailLanguage(ctx, language, true)
}

// SetMember creates or edits a Direct Access member as Exchange.SetMember does, over the websocket
func (w WebsocketAPI) SetMember(ctx context.Context, arg *SetMemberRequest) (*MembersResponse, error) {
	return w.e.setMember(ctx, arg, true)
}

// SetSelfTradingConfig configures self trading prevention as Exchange.SetSelfTradingConfig does, over the websocket
func (w WebsocketAPI) SetSelfTradingConfig(ctx context.Context, arg *SetSelfTradingConfigRequest) error {
	return w.e.setSelfTradingConfig(ctx, arg, true)
}

// SimulatePortfolio calculates the margins of a simulated portfolio as Exchange.SimulatePortfolio does, over the
// websocket
func (w WebsocketAPI) SimulatePortfolio(ctx context.Context, arg *SimulatePortfolioRequest) (*SimulatePortfolioResponse, error) {
	return w.e.simulatePortfolio(ctx, arg, true)
}

// ToggleNotificationsFromSubaccount enables or disables a subaccount's notifications as
// Exchange.ToggleNotificationsFromSubaccount does, over the websocket
func (w WebsocketAPI) ToggleNotificationsFromSubaccount(ctx context.Context, arg *ToggleNotificationsFromSubaccountRequest) error {
	return w.e.toggleNotificationsFromSubaccount(ctx, arg, true)
}

// ToggleSubaccountLogin enables or disables a subaccount's login as Exchange.ToggleSubaccountLogin does, over the
// websocket
func (w WebsocketAPI) ToggleSubaccountLogin(ctx context.Context, arg *ToggleSubaccountLoginRequest) error {
	return w.e.toggleSubaccountLogin(ctx, arg, true)
}

// GetAnnouncements returns the platform's announcements as Exchange.GetAnnouncements does, over the websocket
func (w WebsocketAPI) GetAnnouncements(ctx context.Context, arg *GetAnnouncementsRequest) ([]Announcement, error) {
	return w.e.getAnnouncements(ctx, arg, true)
}
