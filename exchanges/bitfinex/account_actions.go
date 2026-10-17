package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	errInvalidAccountWallet             = errors.New("wallet must be exchange, margin or funding")
	errTransactionMethodRequired        = errors.New("deposit or withdrawal method required, such as bitcoin")
	errWithdrawalAddressRequired        = errors.New("withdrawal address required")
	errWithdrawalInvoiceRequired        = errors.New("lightning withdrawal invoice required")
	errTravelRuleNameIncomplete         = errors.New("travel rule beneficiary needs both a first and a last name")
	errInvalidTokenTimeToLive           = errors.New("token time to live must be a whole number of seconds")
	errInvalidTokenCapability           = errors.New("invalid token capability")
	errTokenWithdrawNeedsWrite          = errors.New("token withdraw capability needs write permission")
	errInvalidTwoFactorMethod           = errors.New("two factor method must be otp or u2f")
	errTwoFactorPasswordRequired        = errors.New("one time password required for the otp two factor method")
	errTwoFactorAssertionRequired       = errors.New("assertion required for the u2f two factor method")
	errLNXPaymentHashRequired           = errors.New("lightning payment hash required")
	errMovementIDRequired               = errors.New("movement ID required")
	errInvalidBalanceAvailableType      = errors.New("type must be EXCHANGE, MARGIN, DERIV or FUNDING")
	errInvalidBalanceAvailableDirection = errors.New("direction must be 1 to buy or -1 to sell")
	errUserSettingsRequired             = errors.New("user settings required")
	errInvalidUserSettingKey            = errors.New("user setting key must be api: followed by letters, digits, underscores or hyphens")
)

// GetUserInfo returns the account's details, such as its verification level and two factor methods
func (e *Exchange) GetUserInfo(ctx context.Context) (*UserInfoResponse, error) {
	var resp *UserInfoResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, userInfoEPL, "auth/r/info/user", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAccountSummary returns the account's maker and taker fee rates, its trading volume, fees and funding earnings
// over the last 30 days, and its LEO discount level
func (e *Exchange) GetAccountSummary(ctx context.Context) (*AccountSummaryResponse, error) {
	var resp *AccountSummaryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, summaryEPL, "auth/r/summary", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLoginHistory returns the account's logins, at most 250 a request
func (e *Exchange) GetLoginHistory(ctx context.Context, req *LoginHistoryRequest) ([]Login, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []Login
	return resp, e.SendAuthenticatedHTTPRequest(ctx, loginHistoryEPL, "auth/r/logins/hist", &body, &resp)
}

// GetKeyPermissions returns whether the API key reads and writes in each scope
func (e *Exchange) GetKeyPermissions(ctx context.Context) ([]KeyPermission, error) {
	var resp []KeyPermission
	return resp, e.SendAuthenticatedHTTPRequest(ctx, keyPermissionsEPL, "auth/r/permissions", nil, &resp)
}

// GenerateToken generates an authentication token for API requests, which lasts 120 seconds unless the request says
// otherwise
func (e *Exchange) GenerateToken(ctx context.Context, req *GenerateTokenRequest) (*GenerateTokenResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TimeToLive < 0 || req.TimeToLive%time.Second != 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidTokenTimeToLive, req.TimeToLive)
	}
	for _, c := range req.Capabilities {
		switch c {
		case TokenCapabilityAccount, TokenCapabilityOrders, TokenCapabilityFunding, TokenCapabilitySettings,
			TokenCapabilityWallets, TokenCapabilityWithdraw, TokenCapabilityBitfinexPay:
		default:
			return nil, fmt.Errorf("%w: %q", errInvalidTokenCapability, c)
		}
	}
	if !req.WritePermission && slices.Contains(req.Capabilities, TokenCapabilityWithdraw) {
		return nil, errTokenWithdrawNeedsWrite
	}
	body := &generateTokenBody{
		// The docs give api as the only scope
		Scope:           "api",
		TimeToLive:      int64(req.TimeToLive / time.Second),
		Capabilities:    req.Capabilities,
		WritePermission: req.WritePermission,
		UserIP:          req.UserIP,
	}
	var resp *GenerateTokenResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, generateTokenEPL, "auth/w/token", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetChangelog returns the changes made to the account, such as API keys created, at most 500 a request
func (e *Exchange) GetChangelog(ctx context.Context, req *ChangelogRequest) ([]ChangelogEntry, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	body, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	var resp []ChangelogEntry
	return resp, e.SendAuthenticatedHTTPRequest(ctx, changelogEPL, "auth/r/audit/hist", &body, &resp)
}

// TransferBetweenWallets transfers funds between the account's exchange, margin and funding wallets, or to a wallet of
// its sub-account or master account, converting them when CurrencyTo is set, such as UST to USTF0 for derivatives
func (e *Exchange) TransferBetweenWallets(ctx context.Context, req *TransferBetweenWalletsRequest) (*Notification[WalletTransfer], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkAccountWallet(req.From); err != nil {
		return nil, err
	}
	if err := checkAccountWallet(req.To); err != nil {
		return nil, err
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if req.TwoFactorToken != nil {
		if err := req.TwoFactorToken.validate(); err != nil {
			return nil, err
		}
	}
	body := &transferBody{
		From:              req.From,
		To:                req.To,
		Currency:          req.Currency.Upper().String(),
		Amount:            strconv.FormatFloat(req.Amount, 'f', -1, 64),
		DestinationEmail:  req.DestinationEmail,
		DestinationUserID: req.DestinationUserID,
		TwoFactorToken:    req.TwoFactorToken,
	}
	if !req.CurrencyTo.IsEmpty() {
		body.CurrencyTo = req.CurrencyTo.Upper().String()
	}
	return sendNotificationRequest[WalletTransfer](ctx, e, transferEPL, "auth/w/transfer", body)
}

// RequestDepositAddress returns the account's deposit address for a method and wallet, or with Renew a new one, the
// old ones staying valid. For a currency needing a tag, memo or payment ID, the address returned is that tag
func (e *Exchange) RequestDepositAddress(ctx context.Context, req *DepositAddressRequest) (*Notification[DepositAddress], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkAccountWallet(req.Wallet); err != nil {
		return nil, err
	}
	if req.Method == "" {
		return nil, errTransactionMethodRequired
	}
	// The docs list methods in lower case, and the pub:map:tx:method configuration in upper case
	body := &depositAddressBody{Wallet: req.Wallet, Method: strings.ToLower(req.Method)}
	if req.Renew {
		body.Renew = 1
	}
	return sendNotificationRequest[DepositAddress](ctx, e, depositAddressEPL, "auth/w/deposit/address", body)
}

// GetDepositAddressList returns a page of the account's deposit addresses for a method, with the wallet each credits
func (e *Exchange) GetDepositAddressList(ctx context.Context, req *DepositAddressListRequest) ([]WalletDepositAddress, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Method == "" {
		return nil, errTransactionMethodRequired
	}
	// The docs list methods in lower case, and the pub:map:tx:method configuration in upper case
	body := &depositAddressListBody{Method: strings.ToLower(req.Method), Page: req.Page, PageSize: req.PageSize}
	var resp []WalletDepositAddress
	return resp, e.SendAuthenticatedHTTPRequest(ctx, depositAddressListEPL, "auth/r/deposit/address/all", body, &resp)
}

// GenerateInvoice generates a Lightning Network invoice for a deposit of an amount of BTC, from 0.000001 to 0.02, into
// the exchange wallet, the only wallet Bitfinex invoices for, in LNX, the only currency. An account's first invoice
// needs an LNX deposit address, from RequestDepositAddress with the lnx method and the exchange wallet
func (e *Exchange) GenerateInvoice(ctx context.Context, amount float64) (*GenerateInvoiceResponse, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, amount)
	}
	body := &generateInvoiceBody{Wallet: WalletTypeExchange, Currency: "LNX", Amount: strconv.FormatFloat(amount, 'f', -1, 64)}
	var resp *GenerateInvoiceResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, generateInvoiceEPL, "auth/w/deposit/invoice", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// lnxInvoicePayments sends an LNX Invoice Payments query, whose response depends on its action
func (e *Exchange) lnxInvoicePayments(ctx context.Context, action string, query lnxInvoicePaymentsQuery, result any) error {
	return e.SendAuthenticatedHTTPRequest(ctx, lnxInvoicePaymentsEPL, "auth/r/ext/invoice/payments", &lnxInvoicePaymentsBody{Action: action, Query: query}, result)
}

// GetLNXInvoices returns the account's Lightning Network invoices, from an offset into them
func (e *Exchange) GetLNXInvoices(ctx context.Context, offset uint64) ([]LNXInvoice, error) {
	var resp lnxInvoiceListResponse
	if err := e.lnxInvoicePayments(ctx, "getInvoicesByUser", lnxInvoicePaymentsQuery{Offset: offset}, &resp); err != nil {
		return nil, err
	}
	return resp.Invoices, nil
}

// GetLNXPayments returns the account's Lightning Network payments, from an offset into them
func (e *Exchange) GetLNXPayments(ctx context.Context, offset uint64) ([]LNXInvoice, error) {
	var resp lnxInvoiceListResponse
	if err := e.lnxInvoicePayments(ctx, "getPaymentsByUser", lnxInvoicePaymentsQuery{Offset: offset}, &resp); err != nil {
		return nil, err
	}
	return resp.Payments, nil
}

// GetLNXInvoice returns the account's Lightning Network invoice with a payment hash
func (e *Exchange) GetLNXInvoice(ctx context.Context, paymentHash string) (*LNXInvoice, error) {
	if paymentHash == "" {
		return nil, errLNXPaymentHashRequired
	}
	var resp *LNXInvoice
	if err := e.lnxInvoicePayments(ctx, "getInvoiceById", lnxInvoicePaymentsQuery{PaymentHash: paymentHash}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetLNXPayment returns the account's Lightning Network payment with a payment hash
func (e *Exchange) GetLNXPayment(ctx context.Context, paymentHash string) (*LNXInvoice, error) {
	if paymentHash == "" {
		return nil, errLNXPaymentHashRequired
	}
	var resp *LNXInvoice
	if err := e.lnxInvoicePayments(ctx, "getPaymentById", lnxInvoicePaymentsQuery{PaymentHash: paymentHash}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// RequestWithdrawal requests a withdrawal from one of the account's wallets to an address, or with
// TransactionMethodLightning the payment of an invoice. Withdrawals worth $1,000 or more need TravelRule details
func (e *Exchange) RequestWithdrawal(ctx context.Context, req *WithdrawalRequest) (*Notification[Withdrawal], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkAccountWallet(req.Wallet); err != nil {
		return nil, err
	}
	// The docs want methods in lower case, and the pub:map:tx:method configuration lists them in upper case
	method := strings.ToLower(req.Method)
	if method == "" {
		return nil, errTransactionMethodRequired
	}
	if req.Amount < 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
	}
	if method == TransactionMethodLightning {
		if req.Invoice == "" {
			return nil, errWithdrawalInvoiceRequired
		}
	} else {
		if req.Amount == 0 {
			return nil, fmt.Errorf("%w: %v", errInvalidAmount, req.Amount)
		}
		if req.Address == "" {
			return nil, errWithdrawalAddressRequired
		}
	}
	body := &withdrawalBody{
		Wallet:    req.Wallet,
		Method:    method,
		Address:   req.Address,
		Invoice:   req.Invoice,
		PaymentID: req.PaymentID,
		Note:      req.Note,
	}
	if req.Amount != 0 {
		body.Amount = strconv.FormatFloat(req.Amount, 'f', -1, 64)
	}
	if req.DeductFee {
		body.DeductFee = 1
	}
	if t := req.TravelRule; t != nil {
		if (t.FirstName == "") != (t.LastName == "") {
			return nil, errTravelRuleNameIncomplete
		}
		body.TravelRuleTermsOfService = true
		body.VASPDID = t.VASPDID
		body.VASPName = t.VASPName
		body.BeneficiarySelf = t.BeneficiarySelf
		body.DestinationFirstName = t.FirstName
		body.DestinationLastName = t.LastName
		body.DestinationCorporateName = t.CorporateName
	}
	return sendNotificationRequest[Withdrawal](ctx, e, withdrawalEPL, "auth/w/withdraw", body)
}

// GetMovements returns the account's deposits and withdrawals in a currency, or recent ones in every currency when
// Currency is unset, at most 2500 a request
func (e *Exchange) GetMovements(ctx context.Context, req *MovementsRequest) ([]Movement, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	history, err := newHistoryBody(req.Start, req.End, req.Limit, 0)
	if err != nil {
		return nil, err
	}
	path := "auth/r/movements/hist"
	if !req.Currency.IsEmpty() {
		path = "auth/r/movements/" + req.Currency.Upper().String() + "/hist"
	}
	body := &movementsBody{historyBody: history, IDs: req.IDs, Address: req.Address}
	var resp []Movement
	return resp, e.SendAuthenticatedHTTPRequest(ctx, movementsEPL, path, body, &resp)
}

// GetMovementInfo returns the details of a deposit or withdrawal, such as a wire's bank fees and its external
// provider's status
func (e *Exchange) GetMovementInfo(ctx context.Context, id uint64) (*MovementInfoResponse, error) {
	if id == 0 {
		return nil, errMovementIDRequired
	}
	var resp *MovementInfoResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, movementInfoEPL, "auth/r/movements/info", &movementInfoBody{ID: id}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAlertList returns the account's active price alerts
func (e *Exchange) GetAlertList(ctx context.Context) ([]Alert, error) {
	var resp []Alert
	// The docs give no body, but Bitfinex's own client sends the alert type
	return resp, e.SendAuthenticatedHTTPRequest(ctx, alertListEPL, "auth/r/alerts", &alertListBody{Type: alertTypePrice}, &resp)
}

// SetAlert sets a price alert on a trading pair, which Bitfinex removes once it has triggered Count times, 100 by
// default
func (e *Exchange) SetAlert(ctx context.Context, req *SetAlertRequest) (*Alert, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.Price <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, req.Price)
	}
	body := &setAlertBody{Type: alertTypePrice, Symbol: req.Symbol, Price: strconv.FormatFloat(req.Price, 'f', -1, 64), Count: req.Count}
	var resp *Alert
	if err := e.SendAuthenticatedHTTPRequest(ctx, alertSetEPL, "auth/w/alert/set", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// DeleteAlert deletes the price alert on a trading pair at a price
func (e *Exchange) DeleteAlert(ctx context.Context, req *DeleteAlertRequest) (*DeleteAlertResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !isTradingSymbol(req.Symbol) {
		return nil, errTradingSymbolRequired
	}
	if req.Price <= 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, req.Price)
	}
	// The path is the alert's key, which formats its price as SetAlert sends it
	path := "auth/w/alert/" + alertTypePrice + ":" + req.Symbol + ":" + strconv.FormatFloat(req.Price, 'f', -1, 64) + "/del"
	var resp *DeleteAlertResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, alertDeleteEPL, path, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetBalanceAvailable returns the largest amount an order on a trading pair, or a funding offer of a currency, can be
// placed with: exchange and margin orders need their direction and price, and a derivative order may give its leverage
func (e *Exchange) GetBalanceAvailable(ctx context.Context, req *BalanceAvailableRequest) (*BalanceAvailableResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol == "" {
		return nil, errSymbolRequired
	}
	var isOrder bool
	switch req.Type {
	case BalanceAvailableTypeExchange, BalanceAvailableTypeMargin:
		isOrder = true
	case BalanceAvailableTypeDerivative, BalanceAvailableTypeFunding:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBalanceAvailableType, req.Type)
	}
	if req.Direction < -1 || req.Direction > 1 || isOrder && req.Direction == 0 {
		return nil, fmt.Errorf("%w: %d", errInvalidBalanceAvailableDirection, req.Direction)
	}
	if req.Rate < 0 || isOrder && req.Rate == 0 {
		return nil, fmt.Errorf("%w: %v", errInvalidPrice, req.Rate)
	}
	body := &balanceAvailableBody{Symbol: req.Symbol, Direction: req.Direction, Type: req.Type}
	if req.Rate != 0 {
		body.Rate = strconv.FormatFloat(req.Rate, 'f', -1, 64)
	}
	if req.Leverage != 0 {
		body.Leverage = strconv.FormatFloat(req.Leverage, 'f', -1, 64)
	}
	var resp *BalanceAvailableResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, balanceAvailableEPL, "auth/calc/order/avail", body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// WriteUserSettings creates or changes the account's custom settings, whose keys are api: followed by letters, digits,
// underscores and hyphens, and whose values are any JSON
func (e *Exchange) WriteUserSettings(ctx context.Context, settings map[string]json.RawMessage) (*Notification[UserSettingsCount], error) {
	if len(settings) == 0 {
		return nil, errUserSettingsRequired
	}
	if err := checkUserSettingKeys(maps.Keys(settings)); err != nil {
		return nil, err
	}
	return sendNotificationRequest[UserSettingsCount](ctx, e, userSettingsWriteEPL, "auth/w/settings/set", &userSettingsWriteBody{Settings: settings})
}

// GetUserSettings returns the values of the account's custom settings with the given keys, keyed as Bitfinex returns
// them, which the docs' example shows without the api: prefix
func (e *Exchange) GetUserSettings(ctx context.Context, keys []string) (map[string]json.RawMessage, error) {
	if len(keys) == 0 {
		return nil, errUserSettingsRequired
	}
	if err := checkUserSettingKeys(slices.Values(keys)); err != nil {
		return nil, err
	}
	var resp []userSetting
	if err := e.SendAuthenticatedHTTPRequest(ctx, userSettingsReadEPL, "auth/r/settings", &userSettingKeysBody{Keys: keys}, &resp); err != nil {
		return nil, err
	}
	settings := make(map[string]json.RawMessage, len(resp))
	for i := range resp {
		settings[resp[i].Key] = resp[i].Value
	}
	return settings, nil
}

// DeleteUserSettings deletes the account's custom settings with the given keys
func (e *Exchange) DeleteUserSettings(ctx context.Context, keys []string) (*Notification[json.RawMessage], error) {
	if len(keys) == 0 {
		return nil, errUserSettingsRequired
	}
	if err := checkUserSettingKeys(slices.Values(keys)); err != nil {
		return nil, err
	}
	return sendNotificationRequest[json.RawMessage](ctx, e, userSettingsDeleteEPL, "auth/w/settings/del", &userSettingKeysBody{Keys: keys})
}

// checkAccountWallet returns an error unless a wallet is the exchange, margin or funding wallet
func checkAccountWallet(wallet string) error {
	switch wallet {
	case WalletTypeExchange, WalletTypeMargin, WalletTypeFunding:
		return nil
	}
	return fmt.Errorf("%w: %q", errInvalidAccountWallet, wallet)
}

// checkUserSettingKeys returns an error for a key not matching the pattern Bitfinex requires, ^api:[A-Za-z0-9_-]*$
func checkUserSettingKeys(keys iter.Seq[string]) error {
	for key := range keys {
		rest, ok := strings.CutPrefix(key, "api:")
		if !ok || strings.ContainsFunc(rest, func(r rune) bool {
			return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-'
		}) {
			return fmt.Errorf("%w: %q", errInvalidUserSettingKey, key)
		}
	}
	return nil
}

// validate returns an error when the token lacks what its method needs
func (t *TwoFactorToken) validate() error {
	switch t.Method {
	case TwoFactorMethodOneTimePassword:
		if t.Token == "" {
			return errTwoFactorPasswordRequired
		}
	case TwoFactorMethodU2F:
		if t.Response == nil {
			return errTwoFactorAssertionRequired
		}
	default:
		return fmt.Errorf("%w: %q", errInvalidTwoFactorMethod, t.Method)
	}
	return nil
}
