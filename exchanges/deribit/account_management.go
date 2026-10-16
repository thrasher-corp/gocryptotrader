package deribit

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Margin models an account or subaccount can use
const (
	MarginModelCrossPortfolio      = "cross_pm"
	MarginModelCrossStandard       = "cross_sm"
	MarginModelSegregatedPortfolio = "segregated_pm"
	MarginModelSegregatedStandard  = "segregated_sm"
)

// Enabled features of an API key: restricted_block_trades limits the key's block trade read scope to the block trades
// it made, and block_trade_approval needs the block trades it creates approved by the user
const (
	EnabledFeatureRestrictedBlockTrades = "restricted_block_trades"
	EnabledFeatureBlockTradeApproval    = "block_trade_approval"
)

// Trading products, which can be disabled for a subaccount
const (
	TradingProductPerpetual    = "perpetual"
	TradingProductFutures      = "futures"
	TradingProductOptions      = "options"
	TradingProductFutureCombos = "future_combos"
	TradingProductOptionCombos = "option_combos"
	TradingProductSpots        = "spots"
)

// Self trading prevention modes, which reject the incoming order or cancel the resting order it would trade with
const (
	SelfTradingModeRejectTaker = "reject_taker"
	SelfTradingModeCancelMaker = "cancel_maker"
)

// Subaccount login states
const (
	SubaccountLoginEnable  = "enable"
	SubaccountLoginDisable = "disable"
)

var (
	errAPIKeyIDRequired              = errors.New("API key ID is required")
	errAPIKeyNameRequired            = errors.New("API key name is required")
	errAPIKeyMaxScopeRequired        = errors.New("API key maximum scope is required")
	errInvalidAPIKeyFeature          = errors.New("invalid API key feature")
	errInvalidMarginModel            = errors.New("invalid margin model")
	errSubaccountIDRequired          = errors.New("subaccount ID is required")
	errSubaccountNameRequired        = errors.New("subaccount name is required")
	errSubaccountEmailRequired       = errors.New("subaccount email is required")
	errInvalidSubaccountLoginState   = errors.New("invalid subaccount login state")
	errMemberIDRequired              = errors.New("member ID is required")
	errMemberNameRequired            = errors.New("member name is required")
	errMemberAccountsRequired        = errors.New("accounts are required to create a member")
	errMemberCreatedInactive         = errors.New("a member cannot be created inactive")
	errAnnouncementIDRequired        = errors.New("announcement ID is required")
	errEmailLanguageRequired         = errors.New("email language is required")
	errTradingProductsUserIDRequired = errors.New("user ID is required to set disabled trading products")
	errInvalidTradingProduct         = errors.New("invalid trading product")
	errInvalidSelfTradingMode        = errors.New("invalid self trading mode")
)

// ChangeAPIKeyName renames an API key, which identifies it in lists and the transaction log
func (e *Exchange) ChangeAPIKeyName(ctx context.Context, arg *ChangeAPIKeyNameRequest) (*APIKey, error) {
	return e.changeAPIKeyName(ctx, arg, false)
}

func (e *Exchange) changeAPIKeyName(ctx context.Context, arg *ChangeAPIKeyNameRequest, overWebsocket bool) (*APIKey, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ID == 0 {
		return nil, errAPIKeyIDRequired
	}
	if arg.Name == "" {
		return nil, errAPIKeyNameRequired
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/change_api_key_name", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ChangeMarginModel changes the margin model of the account or a subaccount, which changes its margin requirements and
// available funds, and returns the change's effect on each currency; a dry run returns the effect without the change
func (e *Exchange) ChangeMarginModel(ctx context.Context, arg *ChangeMarginModelRequest) ([]MarginModelChange, error) {
	return e.changeMarginModel(ctx, arg, false)
}

func (e *Exchange) changeMarginModel(ctx context.Context, arg *ChangeMarginModelRequest, overWebsocket bool) ([]MarginModelChange, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	switch arg.MarginModel {
	case MarginModelCrossPortfolio, MarginModelCrossStandard, MarginModelSegregatedPortfolio, MarginModelSegregatedStandard:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidMarginModel, arg.MarginModel)
	}
	var resp []MarginModelChange
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/change_margin_model", arg, &resp)
}

// ChangeScopeInAPIKey changes an API key's maximum scope, removing its access outside the new scope at once; on an
// account with two-factor authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) ChangeScopeInAPIKey(ctx context.Context, arg *ChangeScopeInAPIKeyRequest) (*APIKey, error) {
	return e.changeScopeInAPIKey(ctx, arg, false)
}

func (e *Exchange) changeScopeInAPIKey(ctx context.Context, arg *ChangeScopeInAPIKeyRequest, overWebsocket bool) (*APIKey, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ID == 0 {
		return nil, errAPIKeyIDRequired
	}
	if len(arg.MaxScope) == 0 {
		return nil, errAPIKeyMaxScopeRequired
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/change_scope_in_api_key", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ChangeSubaccountName renames a subaccount; only the main account can
func (e *Exchange) ChangeSubaccountName(ctx context.Context, arg *ChangeSubaccountNameRequest) error {
	return e.changeSubaccountName(ctx, arg, false)
}

func (e *Exchange) changeSubaccountName(ctx context.Context, arg *ChangeSubaccountNameRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SubaccountID == 0 {
		return errSubaccountIDRequired
	}
	if arg.Name == "" {
		return errSubaccountNameRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/change_subaccount_name", arg)
}

// CreateAPIKey creates an API key, whose scope cannot exceed the scope of the key creating it, and returns it with its
// secret, which Deribit shows only once; on an account with two-factor authentication, it returns a
// SecurityKeyChallengeError until the request is confirmed with WithSecurityKeyAuthorisation
func (e *Exchange) CreateAPIKey(ctx context.Context, arg *CreateAPIKeyRequest) (*APIKey, error) {
	return e.createAPIKey(ctx, arg, false)
}

func (e *Exchange) createAPIKey(ctx context.Context, arg *CreateAPIKeyRequest, overWebsocket bool) (*APIKey, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if len(arg.MaxScope) == 0 {
		return nil, errAPIKeyMaxScopeRequired
	}
	if err := validateAPIKeyFeatures(arg.EnabledFeatures); err != nil {
		return nil, err
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/create_api_key", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// validateAPIKeyFeatures checks every API key feature is one Deribit documents
func validateAPIKeyFeatures(features []string) error {
	for _, feature := range features {
		switch feature {
		case EnabledFeatureRestrictedBlockTrades, EnabledFeatureBlockTradeApproval:
		default:
			return fmt.Errorf("%w: %q", errInvalidAPIKeyFeature, feature)
		}
	}
	return nil
}

// CreateSubaccount creates a subaccount of the main account with default settings
func (e *Exchange) CreateSubaccount(ctx context.Context) (*Subaccount, error) {
	return e.createSubaccount(ctx, false)
}

func (e *Exchange) createSubaccount(ctx context.Context, overWebsocket bool) (*Subaccount, error) {
	var resp *Subaccount
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/create_subaccount", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// DeleteMember deletes a Direct Access member of the main account and returns the remaining members; it needs Direct
// Access trading enabled
func (e *Exchange) DeleteMember(ctx context.Context, memberID uint64) (*MembersResponse, error) {
	return e.deleteMember(ctx, memberID, false)
}

func (e *Exchange) deleteMember(ctx context.Context, memberID uint64, overWebsocket bool) (*MembersResponse, error) {
	if memberID == 0 {
		return nil, errMemberIDRequired
	}
	var resp *MembersResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/delete_member", &memberIDParams{MemberID: memberID}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// DisableAPIKey disables an API key, rejecting its requests until it is enabled again; on an account with two-factor
// authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) DisableAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return e.disableAPIKey(ctx, id, false)
}

func (e *Exchange) disableAPIKey(ctx context.Context, id uint64, overWebsocket bool) (*APIKey, error) {
	if id == 0 {
		return nil, errAPIKeyIDRequired
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/disable_api_key", &apiKeyIDParams{ID: id}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EditAPIKey edits an API key's maximum scope and, optionally, its name, state, features and IP whitelist, which take
// effect at once; on an account with two-factor authentication, it returns a SecurityKeyChallengeError until the
// request is confirmed with WithSecurityKeyAuthorisation
func (e *Exchange) EditAPIKey(ctx context.Context, arg *EditAPIKeyRequest) (*APIKey, error) {
	return e.editAPIKey(ctx, arg, false)
}

func (e *Exchange) editAPIKey(ctx context.Context, arg *EditAPIKeyRequest, overWebsocket bool) (*APIKey, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.ID == 0 {
		return nil, errAPIKeyIDRequired
	}
	if len(arg.MaxScope) == 0 {
		return nil, errAPIKeyMaxScopeRequired
	}
	if err := validateAPIKeyFeatures(arg.EnabledFeatures); err != nil {
		return nil, err
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/edit_api_key", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// EnableAffiliateProgram enables the account's affiliate program, which pays commission on the trading of users it
// refers; it cannot be disabled through the API
func (e *Exchange) EnableAffiliateProgram(ctx context.Context) error {
	return e.sendOKRequest(ctx, false, nonMatchingEPL, "private/enable_affiliate_program", nil)
}

// EnableAPIKey enables a disabled API key, which can authenticate at once
func (e *Exchange) EnableAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return e.enableAPIKey(ctx, id, false)
}

func (e *Exchange) enableAPIKey(ctx context.Context, id uint64, overWebsocket bool) (*APIKey, error) {
	if id == 0 {
		return nil, errAPIKeyIDRequired
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/enable_api_key", &apiKeyIDParams{ID: id}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccessLog returns a page of the account's access log, newest first: its logins and the changes to its security
// and API keys
func (e *Exchange) GetAccessLog(ctx context.Context, arg *GetAccessLogRequest) (*AccessLogResponse, error) {
	return e.getAccessLog(ctx, arg, false)
}

func (e *Exchange) getAccessLog(ctx context.Context, arg *GetAccessLogRequest, overWebsocket bool) (*AccessLogResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp *AccessLogResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_access_log", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountSummaries returns the summary of the account, or a subaccount, in every currency it holds, which Deribit
// prefers to requesting each currency's summary
func (e *Exchange) GetAccountSummaries(ctx context.Context, arg *GetAccountSummariesRequest) (*AccountSummariesResponse, error) {
	return e.getAccountSummaries(ctx, arg, false)
}

func (e *Exchange) getAccountSummaries(ctx context.Context, arg *GetAccountSummariesRequest, overWebsocket bool) (*AccountSummariesResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	var resp *AccountSummariesResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_account_summaries", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountSummary returns the summary of the account, or a subaccount, in one currency: its cash balance, equity,
// margins, profit and loss, greeks and rate limits
func (e *Exchange) GetAccountSummary(ctx context.Context, arg *GetAccountSummaryRequest) (*AccountSummaryResponse, error) {
	return e.getAccountSummary(ctx, arg, false)
}

func (e *Exchange) getAccountSummary(ctx context.Context, arg *GetAccountSummaryRequest, overWebsocket bool) (*AccountSummaryResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *AccountSummaryResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_account_summary", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAffiliateProgramInfo returns the account's affiliate program status, referral link, number of affiliates and the
// payouts received
func (e *Exchange) GetAffiliateProgramInfo(ctx context.Context) (*AffiliateProgramInfoResponse, error) {
	return e.getAffiliateProgramInfo(ctx, false)
}

func (e *Exchange) getAffiliateProgramInfo(ctx context.Context, overWebsocket bool) (*AffiliateProgramInfoResponse, error) {
	var resp *AffiliateProgramInfoResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_affiliate_program_info", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountCurrencies returns the currencies available to the account, which Deribit recommends driving wallet and
// balance requests from, as it leaves out currencies the account cannot hold
func (e *Exchange) GetAccountCurrencies(ctx context.Context) ([]AccountCurrency, error) {
	return e.getAccountCurrencies(ctx, false)
}

func (e *Exchange) getAccountCurrencies(ctx context.Context, overWebsocket bool) ([]AccountCurrency, error) {
	var resp []AccountCurrency
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_currencies", nil, &resp)
}

// GetEmailLanguage returns the language of the account's notification emails, such as en, ko, zh, ja or ru
func (e *Exchange) GetEmailLanguage(ctx context.Context) (string, error) {
	return e.getEmailLanguage(ctx, false)
}

func (e *Exchange) getEmailLanguage(ctx context.Context, overWebsocket bool) (string, error) {
	var resp string
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_email_language", nil, &resp); err != nil {
		return "", err
	}
	if resp == "" {
		return "", common.ErrNoResponse
	}
	return resp, nil
}

// GetMembers returns the account's Direct Access members; a subaccount is only sent the names and states of the
// members that may trade on it
func (e *Exchange) GetMembers(ctx context.Context) (*MembersResponse, error) {
	return e.getMembers(ctx, false)
}

func (e *Exchange) getMembers(ctx context.Context, overWebsocket bool) (*MembersResponse, error) {
	var resp *MembersResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_members", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetNewAnnouncements returns the announcements the account has not read, which SetAnnouncementAsRead or viewing them
// on the website marks read
func (e *Exchange) GetNewAnnouncements(ctx context.Context) ([]Announcement, error) {
	return e.getNewAnnouncements(ctx, false)
}

func (e *Exchange) getNewAnnouncements(ctx context.Context, overWebsocket bool) ([]Announcement, error) {
	var resp []Announcement
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_new_announcements", nil, &resp)
}

// GetPosition returns the account's position in an instrument, which has a zero size when there is none
func (e *Exchange) GetPosition(ctx context.Context, instrumentName string) (*Position, error) {
	return e.getPosition(ctx, instrumentName, false)
}

func (e *Exchange) getPosition(ctx context.Context, instrumentName string, overWebsocket bool) (*Position, error) {
	if instrumentName == "" {
		return nil, errInstrumentNameRequired
	}
	var resp *Position
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_position", &instrumentNameParams{InstrumentName: instrumentName}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetPositions returns the open positions of the account or a subaccount, optionally in one settlement currency and of
// one kind; spot holdings are balances, which GetAccountSummary reports
func (e *Exchange) GetPositions(ctx context.Context, arg *GetPositionsRequest) ([]Position, error) {
	return e.getPositions(ctx, arg, false)
}

func (e *Exchange) getPositions(ctx context.Context, arg *GetPositionsRequest, overWebsocket bool) ([]Position, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	switch arg.Kind {
	case "", KindFuture, KindOption, KindFutureCombo, KindOptionCombo:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidKind, arg.Kind)
	}
	var resp []Position
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_positions", arg, &resp)
}

// GetSubaccounts returns the main account's and its subaccounts' details, with their portfolios when withPortfolio is
// set; a subaccount is sent limited details of the main account
func (e *Exchange) GetSubaccounts(ctx context.Context, withPortfolio bool) ([]Subaccount, error) {
	return e.getSubaccounts(ctx, withPortfolio, false)
}

func (e *Exchange) getSubaccounts(ctx context.Context, withPortfolio, overWebsocket bool) ([]Subaccount, error) {
	var resp []Subaccount
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_subaccounts", &withPortfolioParams{WithPortfolio: withPortfolio}, &resp)
}

// GetSubaccountsDetails returns the positions of the main account and each subaccount in a currency and, optionally,
// their open orders
func (e *Exchange) GetSubaccountsDetails(ctx context.Context, arg *GetSubaccountsDetailsRequest) ([]SubaccountDetails, error) {
	return e.getSubaccountsDetails(ctx, arg, false)
}

func (e *Exchange) getSubaccountsDetails(ctx context.Context, arg *GetSubaccountsDetailsRequest, overWebsocket bool) ([]SubaccountDetails, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []SubaccountDetails
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_subaccounts_details", arg, &resp)
}

// GetTransactionLog returns a page of the account's transaction log in a currency, newest first, back to the account's
// creation: its trades, deposits, withdrawals, transfers, settlements, fees and other balance changes. Deribit limits
// it to about one request a second
func (e *Exchange) GetTransactionLog(ctx context.Context, arg *GetTransactionLogRequest) (*TransactionLogResponse, error) {
	return e.getTransactionLog(ctx, arg, false)
}

func (e *Exchange) getTransactionLog(ctx context.Context, arg *GetTransactionLogRequest, overWebsocket bool) (*TransactionLogResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.StartTime.IsZero() {
		return nil, fmt.Errorf("start %w", common.ErrDateUnset)
	}
	if arg.EndTime.IsZero() {
		return nil, fmt.Errorf("end %w", common.ErrDateUnset)
	}
	if arg.StartTime.After(arg.EndTime) {
		return nil, common.ErrStartAfterEnd
	}
	params := &getTransactionLogParams{
		GetTransactionLogRequest: arg,
		StartTimestamp:           arg.StartTime.UnixMilli(),
		EndTimestamp:             arg.EndTime.UnixMilli(),
	}
	var resp *TransactionLogResponse
	if err := e.sendRequest(ctx, overWebsocket, transactionLogEPL, "private/get_transaction_log", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetUserLocks returns whether the account is locked in each currency, and why
func (e *Exchange) GetUserLocks(ctx context.Context) ([]UserLock, error) {
	return e.getUserLocks(ctx, false)
}

func (e *Exchange) getUserLocks(ctx context.Context, overWebsocket bool) ([]UserLock, error) {
	var resp []UserLock
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/get_user_locks", nil, &resp)
}

// ListAPIKeys returns the account's API keys; on an account with two-factor authentication, it returns a
// SecurityKeyChallengeError until the request is confirmed with WithSecurityKeyAuthorisation
func (e *Exchange) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	return e.listAPIKeys(ctx, false)
}

func (e *Exchange) listAPIKeys(ctx context.Context, overWebsocket bool) ([]APIKey, error) {
	var resp []APIKey
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/list_api_keys", nil, &resp)
}

// ListCustodyAccounts returns the account's custody accounts in a currency, which hold its assets segregated
func (e *Exchange) ListCustodyAccounts(ctx context.Context, ccy currency.Code) ([]CustodyAccount, error) {
	return e.listCustodyAccounts(ctx, ccy, false)
}

func (e *Exchange) listCustodyAccounts(ctx context.Context, ccy currency.Code, overWebsocket bool) ([]CustodyAccount, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp []CustodyAccount
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/list_custody_accounts", &currencyParams{Currency: ccy}, &resp)
}

// PMESimulate calculates a portfolio margin account's extended risk matrix and margins for a currency, or for its
// cross collateral portfolio with the currency CROSS, optionally with simulated positions sized in the base currency
func (e *Exchange) PMESimulate(ctx context.Context, arg *SimulatePortfolioRequest) (*PMESimulateResponse, error) {
	return e.pmeSimulate(ctx, arg, false)
}

func (e *Exchange) pmeSimulate(ctx context.Context, arg *SimulatePortfolioRequest, overWebsocket bool) (*PMESimulateResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *PMESimulateResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/pme/simulate", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// RemoveAPIKey removes an API key permanently, so the applications using it lose access; DisableAPIKey suspends one
// instead
func (e *Exchange) RemoveAPIKey(ctx context.Context, id uint64) error {
	return e.removeAPIKey(ctx, id, false)
}

func (e *Exchange) removeAPIKey(ctx context.Context, id uint64, overWebsocket bool) error {
	if id == 0 {
		return errAPIKeyIDRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/remove_api_key", &apiKeyIDParams{ID: id})
}

// RemoveSubaccount removes a subaccount permanently, which must hold no positions, open orders or balance; on an
// account with two-factor authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) RemoveSubaccount(ctx context.Context, subaccountID uint64) error {
	return e.removeSubaccount(ctx, subaccountID, false)
}

func (e *Exchange) removeSubaccount(ctx context.Context, subaccountID uint64, overWebsocket bool) error {
	if subaccountID == 0 {
		return errSubaccountIDRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/remove_subaccount", &subaccountIDParams{SubaccountID: subaccountID})
}

// ResetAPIKey replaces an API key's secret, invalidating the old one at once, and returns the key with its new secret,
// which Deribit shows only once
func (e *Exchange) ResetAPIKey(ctx context.Context, id uint64) (*APIKey, error) {
	return e.resetAPIKey(ctx, id, false)
}

func (e *Exchange) resetAPIKey(ctx context.Context, id uint64, overWebsocket bool) (*APIKey, error) {
	if id == 0 {
		return nil, errAPIKeyIDRequired
	}
	var resp *APIKey
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/reset_api_key", &apiKeyIDParams{ID: id}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetAnnouncementAsRead marks an announcement read, which removes it from GetNewAnnouncements but not from
// GetAnnouncements
func (e *Exchange) SetAnnouncementAsRead(ctx context.Context, announcementID uint64) error {
	return e.setAnnouncementAsRead(ctx, announcementID, false)
}

func (e *Exchange) setAnnouncementAsRead(ctx context.Context, announcementID uint64, overWebsocket bool) error {
	if announcementID == 0 {
		return errAnnouncementIDRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_announcement_as_read", &announcementIDParams{AnnouncementID: announcementID})
}

// SetDisabledTradingProducts sets the trading products a subaccount cannot trade, which only the main account can; on
// an account with two-factor authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) SetDisabledTradingProducts(ctx context.Context, arg *SetDisabledTradingProductsRequest) error {
	return e.setDisabledTradingProducts(ctx, arg, false)
}

func (e *Exchange) setDisabledTradingProducts(ctx context.Context, arg *SetDisabledTradingProductsRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.UserID == 0 {
		return errTradingProductsUserIDRequired
	}
	for _, product := range arg.TradingProducts {
		switch product {
		case TradingProductPerpetual, TradingProductFutures, TradingProductOptions, TradingProductFutureCombos, TradingProductOptionCombos, TradingProductSpots:
		default:
			return fmt.Errorf("%w: %q", errInvalidTradingProduct, product)
		}
	}
	params := *arg
	// Deribit requires the list, which a nil slice would send as null
	if params.TradingProducts == nil {
		params.TradingProducts = []string{}
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_disabled_trading_products", &params)
}

// SetEmailForSubaccount sets the email address of a subaccount, which it can log in and receive notifications with once
// its owner confirms it; on an account with two-factor authentication, it returns a SecurityKeyChallengeError until the
// request is confirmed with WithSecurityKeyAuthorisation
func (e *Exchange) SetEmailForSubaccount(ctx context.Context, arg *SetEmailForSubaccountRequest) error {
	return e.setEmailForSubaccount(ctx, arg, false)
}

func (e *Exchange) setEmailForSubaccount(ctx context.Context, arg *SetEmailForSubaccountRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SubaccountID == 0 {
		return errSubaccountIDRequired
	}
	if arg.Email == "" {
		return errSubaccountEmailRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_email_for_subaccount", arg)
}

// SetEmailLanguage sets the language of the account's notification emails, such as en, ko, zh, ja or ru; it does not
// change the website's or the API's language
func (e *Exchange) SetEmailLanguage(ctx context.Context, language string) error {
	return e.setEmailLanguage(ctx, language, false)
}

func (e *Exchange) setEmailLanguage(ctx context.Context, language string, overWebsocket bool) error {
	if language == "" {
		return errEmailLanguageRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_email_language", &emailLanguageParams{Language: language})
}

// SetMember creates a Direct Access member of the main account, or edits an existing one, and returns every member; it
// needs Direct Access trading enabled
func (e *Exchange) SetMember(ctx context.Context, arg *SetMemberRequest) (*MembersResponse, error) {
	return e.setMember(ctx, arg, false)
}

func (e *Exchange) setMember(ctx context.Context, arg *SetMemberRequest, overWebsocket bool) (*MembersResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Name == "" {
		return nil, errMemberNameRequired
	}
	if arg.ID == 0 {
		if len(arg.Accounts) == 0 {
			return nil, errMemberAccountsRequired
		}
		if arg.IsActive != nil && !*arg.IsActive {
			return nil, errMemberCreatedInactive
		}
	}
	var resp *MembersResponse
	if err := e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_member", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetSelfTradingConfig configures how the account prevents its orders trading with each other, and optionally Block
// RFQ trades between accounts of the same legal entity
func (e *Exchange) SetSelfTradingConfig(ctx context.Context, arg *SetSelfTradingConfigRequest) error {
	return e.setSelfTradingConfig(ctx, arg, false)
}

func (e *Exchange) setSelfTradingConfig(ctx context.Context, arg *SetSelfTradingConfigRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	switch arg.Mode {
	case SelfTradingModeRejectTaker, SelfTradingModeCancelMaker:
	default:
		return fmt.Errorf("%w: %q", errInvalidSelfTradingMode, arg.Mode)
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/set_self_trading_config", arg)
}

// SimulatePortfolio calculates the margins of the account's portfolio in a currency, optionally with simulated
// positions sized in USD for futures and in the base currency for options; Deribit limits it to one request a second
func (e *Exchange) SimulatePortfolio(ctx context.Context, arg *SimulatePortfolioRequest) (*SimulatePortfolioResponse, error) {
	return e.simulatePortfolio(ctx, arg, false)
}

func (e *Exchange) simulatePortfolio(ctx context.Context, arg *SimulatePortfolioRequest, overWebsocket bool) (*SimulatePortfolioResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *SimulatePortfolioResponse
	if err := e.sendRequest(ctx, overWebsocket, simulatePortfolioEPL, "private/simulate_portfolio", arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ToggleNotificationsFromSubaccount enables or disables the notifications sent to a subaccount's email; on an account
// with two-factor authentication, it returns a SecurityKeyChallengeError until the request is confirmed with
// WithSecurityKeyAuthorisation
func (e *Exchange) ToggleNotificationsFromSubaccount(ctx context.Context, arg *ToggleNotificationsFromSubaccountRequest) error {
	return e.toggleNotificationsFromSubaccount(ctx, arg, false)
}

func (e *Exchange) toggleNotificationsFromSubaccount(ctx context.Context, arg *ToggleNotificationsFromSubaccountRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SubaccountID == 0 {
		return errSubaccountIDRequired
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/toggle_notifications_from_subaccount", arg)
}

// ToggleSubaccountLogin enables or disables a subaccount's login with its email and password, ending any session it has
// when disabled; on an account with two-factor authentication, it returns a SecurityKeyChallengeError until the request
// is confirmed with WithSecurityKeyAuthorisation
func (e *Exchange) ToggleSubaccountLogin(ctx context.Context, arg *ToggleSubaccountLoginRequest) error {
	return e.toggleSubaccountLogin(ctx, arg, false)
}

func (e *Exchange) toggleSubaccountLogin(ctx context.Context, arg *ToggleSubaccountLoginRequest, overWebsocket bool) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.SubaccountID == 0 {
		return errSubaccountIDRequired
	}
	switch arg.State {
	case SubaccountLoginEnable, SubaccountLoginDisable:
	default:
		return fmt.Errorf("%w: %q", errInvalidSubaccountLoginState, arg.State)
	}
	return e.sendOKRequest(ctx, overWebsocket, nonMatchingEPL, "private/toggle_subaccount_login", arg)
}

// GetAnnouncements returns the platform's announcements published up to a time, newest first
func (e *Exchange) GetAnnouncements(ctx context.Context, arg *GetAnnouncementsRequest) ([]Announcement, error) {
	return e.getAnnouncements(ctx, arg, false)
}

func (e *Exchange) getAnnouncements(ctx context.Context, arg *GetAnnouncementsRequest, overWebsocket bool) ([]Announcement, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	params := &getAnnouncementsParams{GetAnnouncementsRequest: arg}
	if !arg.StartTime.IsZero() {
		params.StartTimestamp = arg.StartTime.UnixMilli()
	}
	var resp []Announcement
	return resp, e.sendRequest(ctx, overWebsocket, nonMatchingEPL, "public/get_announcements", params, &resp)
}
