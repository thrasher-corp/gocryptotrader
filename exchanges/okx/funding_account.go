package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errRemittingAccountTypeRequired        = errors.New("remitting account type is required")
	errBeneficiaryAccountTypeRequired      = errors.New("beneficiary account type is required")
	errTransferIDOrClientIDRequired        = errors.New("either transfer ID or client ID is required")
	errAfterBillIDAndEndTimeSet            = errors.New("after bill ID and end time cannot both be set")
	errDepositIDRequired                   = errors.New("deposit ID is required")
	errTravelRuleQuestionnaireRequired     = errors.New("travel rule questionnaire is required")
	errPersonTypeRequired                  = errors.New("person type is required")
	errWalletTypeRequired                  = errors.New("wallet type is required")
	errExchangeIDRequired                  = errors.New("exchange ID is required")
	errExchangeNameRequired                = errors.New("exchange name is required")
	errTravelRuleDepositUnlockFailed       = errors.New("travel rule deposit unlock failed")
	errWithdrawalDestinationRequired       = errors.New("withdrawal destination is required")
	errAddressRequired                     = errors.New("address is required")
	errWithdrawalIDRequired                = errors.New("withdrawal ID is required")
	errWithdrawalIDOrTransactionIDRequired = errors.New("exactly one of withdrawal ID and transaction ID is required")
	errChainRequired                       = errors.New("chain is required")
	errMonthRequired                       = errors.New("month is required")
	errInvalidMonth                        = errors.New("invalid month")
	errQuoteIDRequired                     = errors.New("quote ID is required")
	errPaymentAccountIDRequired            = errors.New("payment account ID is required")
	errPaymentMethodRequired               = errors.New("payment method is required")
	errClientIDRequired                    = errors.New("client ID is required")
)

// maxFundingCurrencyFilter is the most currencies a funding balance filter takes
const maxFundingCurrencyFilter = 20

// GetFundingCurrencies returns the currencies available to the account's KYC entity, one entry for each chain
func (e *Exchange) GetFundingCurrencies(ctx context.Context, req *FundingCurrenciesRequest) ([]FundingCurrencyResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	currencies, err := formatFundingCurrencies(req.Currencies, 0)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if currencies != "" {
		params.Set("ccy", currencies)
	}
	var resp []FundingCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCurrenciesEPL, http.MethodGet, common.EncodeURLValues("asset/currencies", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFundingBalances returns the funding account's balances, listing only currencies with a balance above zero
func (e *Exchange) GetFundingBalances(ctx context.Context, req *FundingBalancesRequest) ([]FundingBalanceResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	currencies, err := formatFundingCurrencies(req.Currencies, maxFundingCurrencyFilter)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if currencies != "" {
		params.Set("ccy", currencies)
	}
	var resp []FundingBalanceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBalanceEPL, http.MethodGet, common.EncodeURLValues("asset/balances", params), nil, &resp, request.AuthenticatedRequest)
}

// GetNonTradableAssets returns the funding account's non-tradable assets
func (e *Exchange) GetNonTradableAssets(ctx context.Context, req *NonTradableAssetsRequest) ([]NonTradableAssetResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	currencies, err := formatFundingCurrencies(req.Currencies, maxFundingCurrencyFilter)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if currencies != "" {
		params.Set("ccy", currencies)
	}
	var resp []NonTradableAssetResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getNonTradableAssetsEPL, http.MethodGet, common.EncodeURLValues("asset/non-tradable-assets", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAccountAssetValuation returns the valuation of the account's assets
func (e *Exchange) GetAccountAssetValuation(ctx context.Context, req *AccountAssetValuationRequest) (*AccountAssetValuationResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.Upper().String())
	}
	var resp *AccountAssetValuationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAccountAssetValuationEPL, http.MethodGet, common.EncodeURLValues("asset/asset-valuation", params), nil, &resp, request.AuthenticatedRequest)
}

// FundingTransfer transfers funds between the funding and trading accounts, or between a master account and its
// sub-accounts. OKX accepting a transfer does not mean it completed: GetFundsTransferState reports its state
func (e *Exchange) FundingTransfer(ctx context.Context, req *FundingTransferRequest) (*FundingTransferResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.RemittingAccountType == "" {
		return nil, errRemittingAccountTypeRequired
	}
	if req.BeneficiaryAccountType == "" {
		return nil, errBeneficiaryAccountTypeRequired
	}
	switch req.TransferType {
	case "1", "2", "4":
		if req.SubAccountName == "" {
			return nil, fmt.Errorf("%w: transfer type %s", errSubAccountNameRequired, req.TransferType)
		}
	}
	var resp *FundingTransferResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, fundsTransferEPL, http.MethodPost, "asset/transfer", req, &resp, request.AuthenticatedRequest)
}

// GetFundsTransferState returns the state of a funds transfer from the last two weeks
func (e *Exchange) GetFundsTransferState(ctx context.Context, req *FundsTransferStateRequest) ([]FundsTransferStateResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.TransferID == "" && req.ClientID == "" {
		return nil, errTransferIDOrClientIDRequired
	}
	params := url.Values{}
	if req.TransferID != "" {
		params.Set("transId", req.TransferID)
	}
	if req.ClientID != "" {
		params.Set("clientId", req.ClientID)
	}
	if req.TransferType != "" {
		params.Set("type", req.TransferType)
	}
	var resp []FundsTransferStateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFundsTransferStateEPL, http.MethodGet, common.EncodeURLValues("asset/transfer-state", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAssetBillsDetails returns the funding account's bills from the last month
func (e *Exchange) GetAssetBillsDetails(ctx context.Context, req *AssetBillsDetailsRequest) ([]AssetBillResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := assetBillsParams(req.Currency, req.BillType, req.ThirdPartyType, req.ClientID, req.Limit)
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	var resp []AssetBillResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, assetBillsDetailsEPL, http.MethodGet, common.EncodeURLValues("asset/bills", params), nil, &resp, request.AuthenticatedRequest)
}

// GetAssetBillsHistory returns the funding account's bills since 1 February 2021, which OKX updates every 30 seconds
func (e *Exchange) GetAssetBillsHistory(ctx context.Context, req *AssetBillsHistoryRequest) ([]AssetBillResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.AfterBillID != "" && !req.EndTime.IsZero() {
		return nil, errAfterBillIDAndEndTimeSet
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := assetBillsParams(req.Currency, req.BillType, req.ThirdPartyType, req.ClientID, req.Limit)
	switch {
	case req.AfterBillID != "":
		params.Set("after", req.AfterBillID)
		params.Set("pagingType", "2")
	case !req.EndTime.IsZero():
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	var resp []AssetBillResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAssetBillsHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/bills-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCurrencyDepositAddress returns a currency's deposit addresses, including previously used ones
func (e *Exchange) GetCurrencyDepositAddress(ctx context.Context, ccy currency.Code) ([]DepositAddressResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp []DepositAddressResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositAddressEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-address", params), nil, &resp, request.AuthenticatedRequest)
}

// GetCurrencyDepositHistory returns deposit records newest first, the latest 100 by default
func (e *Exchange) GetCurrencyDepositHistory(ctx context.Context, req *DepositHistoryRequest) ([]DepositHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.DepositID != "" {
		params.Set("depId", req.DepositID)
	}
	if req.FromWithdrawalID != "" {
		params.Set("fromWdId", req.FromWithdrawalID)
	}
	if req.TransactionID != "" {
		params.Set("txId", req.TransactionID)
	}
	if req.DepositType != "" {
		params.Set("type", req.DepositType)
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []DepositHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetLockedTravelRuleDeposits returns the deposits locked under the Travel Rule, at most 50
func (e *Exchange) GetLockedTravelRuleDeposits(ctx context.Context) ([]LockedTravelRuleDepositResponse, error) {
	var resp []LockedTravelRuleDepositResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getLockedTravelRuleDepositsEPL, http.MethodGet, "asset/travel-rule/get-locked", nil, &resp, request.AuthenticatedRequest)
}

// UnlockTravelRuleDeposit submits a deposit originator's information to unlock a deposit locked under the Travel Rule.
// OKX answers success without reprocessing a deposit that is already unlocked
func (e *Exchange) UnlockTravelRuleDeposit(ctx context.Context, req *UnlockTravelRuleDepositRequest) (*UnlockTravelRuleDepositResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.DepositID == "" {
		return nil, errDepositIDRequired
	}
	if req.Questionnaire == nil {
		return nil, errTravelRuleQuestionnaireRequired
	}
	if req.Questionnaire.PersonType == "" {
		return nil, errPersonTypeRequired
	}
	if req.Questionnaire.WalletType == "" {
		return nil, errWalletTypeRequired
	}
	if req.Questionnaire.WalletType == "exchange" && req.Questionnaire.ExchangeID == "" {
		return nil, errExchangeIDRequired
	}
	if req.Questionnaire.ExchangeID == "0" && req.Questionnaire.ExchangeName == "" {
		return nil, errExchangeNameRequired
	}
	var resp []UnlockTravelRuleDepositResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, unlockTravelRuleDepositEPL, http.MethodPost, "asset/travel-rule/unlock", req, &resp, request.AuthenticatedRequest); err != nil {
		return nil, err
	}
	// OKX reports a failed unlock, which leaves the deposit locked, as an empty result
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w: deposit %s", errTravelRuleDepositUnlockFailed, req.DepositID)
	}
	return &resp[0], nil
}

// Withdrawal withdraws from the funding account to a verified address or account; common sub-accounts cannot withdraw
func (e *Exchange) Withdrawal(ctx context.Context, req *WithdrawalRequest) (*WithdrawalResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.Destination == "" {
		return nil, errWithdrawalDestinationRequired
	}
	if req.ToAddress == "" {
		return nil, errAddressRequired
	}
	if req.RecipientInformation != nil && req.RecipientInformation.WalletType == "" {
		return nil, errWalletTypeRequired
	}
	var resp *WithdrawalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, withdrawalEPL, http.MethodPost, "asset/withdrawal", req, &resp, request.AuthenticatedRequest)
}

// CancelWithdrawal cancels a pending withdrawal; Lightning withdrawals cannot be cancelled
func (e *Exchange) CancelWithdrawal(ctx context.Context, withdrawalID string) (*CancelWithdrawalResponse, error) {
	if withdrawalID == "" {
		return nil, errWithdrawalIDRequired
	}
	var resp *CancelWithdrawalResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalEPL, http.MethodPost, "asset/cancel-withdrawal", &CancelWithdrawalRequest{WithdrawalID: withdrawalID}, &resp, request.AuthenticatedRequest)
}

// GetWithdrawalHistory returns withdrawal records newest first, the latest 100 by default
func (e *Exchange) GetWithdrawalHistory(ctx context.Context, req *WithdrawalHistoryRequest) ([]WithdrawalHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.WithdrawalID != "" {
		params.Set("wdId", req.WithdrawalID)
	}
	if req.ClientID != "" {
		params.Set("clientId", req.ClientID)
	}
	if req.TransactionID != "" {
		params.Set("txId", req.TransactionID)
	}
	if req.WithdrawalType != "" {
		params.Set("type", req.WithdrawalType)
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []WithdrawalHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/withdrawal-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetDepositWithdrawalStatus returns a deposit's or withdrawal's detailed status and estimated completion time
func (e *Exchange) GetDepositWithdrawalStatus(ctx context.Context, req *DepositWithdrawalStatusRequest) ([]DepositWithdrawalStatusResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if (req.WithdrawalID == "") == (req.TransactionID == "") {
		return nil, errWithdrawalIDOrTransactionIDRequired
	}
	if req.TransactionID != "" {
		if req.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if req.ToAddress == "" {
			return nil, errAddressRequired
		}
		if req.Chain == "" {
			return nil, errChainRequired
		}
	}
	params := url.Values{}
	if req.WithdrawalID != "" {
		params.Set("wdId", req.WithdrawalID)
	}
	if req.TransactionID != "" {
		params.Set("txId", req.TransactionID)
	}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.ToAddress != "" {
		params.Set("to", req.ToAddress)
	}
	if req.Chain != "" {
		params.Set("chain", req.Chain)
	}
	var resp []DepositWithdrawalStatusResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositWithdrawalStatusEPL, http.MethodGet, common.EncodeURLValues("asset/deposit-withdraw-status", params), nil, &resp, request.AuthenticatedRequest)
}

// GetPublicExchangeList returns the exchanges a withdrawal or Travel Rule questionnaire can name
func (e *Exchange) GetPublicExchangeList(ctx context.Context) ([]ExchangeResponse, error) {
	var resp []ExchangeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getPublicExchangeListEPL, http.MethodGet, "asset/exchange-list", nil, &resp, request.UnauthenticatedRequest)
}

// ApplyForMonthlyStatement applies for a monthly statement from the past year
func (e *Exchange) ApplyForMonthlyStatement(ctx context.Context, req *ApplyForMonthlyStatementRequest) (*ApplyForMonthlyStatementResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.Month != 0 {
		if _, err := formatStatementMonth(req.Month); err != nil {
			return nil, err
		}
	}
	var resp *ApplyForMonthlyStatementResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, applyForMonthlyStatementEPL, http.MethodPost, "asset/monthly-statement", req, &resp, request.AuthenticatedRequest)
}

// GetMonthlyStatement returns a monthly statement from the past year
func (e *Exchange) GetMonthlyStatement(ctx context.Context, month time.Month) ([]MonthlyStatementResponse, error) {
	if month == 0 {
		return nil, errMonthRequired
	}
	statementMonth, err := formatStatementMonth(month)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("month", statementMonth)
	var resp []MonthlyStatementResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMonthlyStatementEPL, http.MethodGet, common.EncodeURLValues("asset/monthly-statement", params), nil, &resp, request.AuthenticatedRequest)
}

// GetConvertCurrencies returns the currencies available to convert
func (e *Exchange) GetConvertCurrencies(ctx context.Context) ([]ConvertCurrencyResponse, error) {
	var resp []ConvertCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrenciesEPL, http.MethodGet, "asset/convert/currencies", nil, &resp, request.AuthenticatedRequest)
}

// GetConvertCurrencyPair returns the convert limits of a currency pair
func (e *Exchange) GetConvertCurrencyPair(ctx context.Context, req *ConvertCurrencyPairRequest) (*ConvertCurrencyPairResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("fromCcy", req.FromCurrency.String())
	params.Set("toCcy", req.ToCurrency.String())
	if req.ConvertMode != "" {
		params.Set("convertMode", req.ConvertMode)
	}
	var resp *ConvertCurrencyPairResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getConvertCurrencyPairEPL, http.MethodGet, common.EncodeURLValues("asset/convert/currency-pair", params), nil, &resp, request.AuthenticatedRequest)
}

// EstimateQuote requests a convert quote, which ConvertTrade then fills
func (e *Exchange) EstimateQuote(ctx context.Context, req *EstimateQuoteRequest) (*EstimateQuoteResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: base currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: quote currency", currency.ErrCurrencyCodeEmpty)
	}
	side, err := formatFundingSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.RequestForQuoteSize <= 0 {
		return nil, fmt.Errorf("%w: request for quote size must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.RequestForQuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: request for quote currency", currency.ErrCurrencyCodeEmpty)
	}
	body := *req
	body.Side = side
	var resp *EstimateQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, estimateQuoteEPL, http.MethodPost, "asset/convert/estimate-quote", &body, &resp, request.AuthenticatedRequest)
}

// ConvertTrade converts between a base and quote currency at a quote from EstimateQuote, using the trading account's
// assets
func (e *Exchange) ConvertTrade(ctx context.Context, req *ConvertTradeRequest) (*ConvertTradeResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.BaseCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: base currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.QuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: quote currency", currency.ErrCurrencyCodeEmpty)
	}
	side, err := formatFundingSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.Size <= 0 {
		return nil, fmt.Errorf("%w: size must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.SizeCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: size currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.QuoteID == "" {
		return nil, errQuoteIDRequired
	}
	body := *req
	body.Side = side
	var resp *ConvertTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, convertTradeEPL, http.MethodPost, "asset/convert/trade", &body, &resp, request.AuthenticatedRequest)
}

// GetConvertHistory returns convert trades newest first
func (e *Exchange) GetConvertHistory(ctx context.Context, req *ConvertHistoryRequest) ([]ConvertHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.ClientTradeRequestID != "" {
		params.Set("clTReqId", req.ClientTradeRequestID)
	}
	if !req.EndTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.Tag != "" {
		params.Set("tag", req.Tag)
	}
	var resp []ConvertHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getConvertHistoryEPL, http.MethodGet, common.EncodeURLValues("asset/convert/history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatDepositPaymentMethods returns the payment methods available for fiat deposits in a currency
func (e *Exchange) GetFiatDepositPaymentMethods(ctx context.Context, ccy currency.Code) ([]FiatPaymentMethodResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp []FiatPaymentMethodResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFiatDepositPaymentMethodsEPL, http.MethodGet, common.EncodeURLValues("fiat/deposit-payment-methods", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatWithdrawalPaymentMethods returns the payment methods available for fiat withdrawals in a currency
func (e *Exchange) GetFiatWithdrawalPaymentMethods(ctx context.Context, ccy currency.Code) ([]FiatPaymentMethodResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("ccy", ccy.String())
	var resp []FiatPaymentMethodResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalPaymentMethodsEPL, http.MethodGet, common.EncodeURLValues("fiat/withdrawal-payment-methods", params), nil, &resp, request.AuthenticatedRequest)
}

// CreateFiatWithdrawalOrder withdraws fiat from the funding account to a payment account; the API key needs the
// withdraw permission
func (e *Exchange) CreateFiatWithdrawalOrder(ctx context.Context, req *CreateFiatWithdrawalOrderRequest) (*FiatOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.PaymentAccountID == "" {
		return nil, errPaymentAccountIDRequired
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.PaymentMethod == "" {
		return nil, errPaymentMethodRequired
	}
	if req.ClientID == "" {
		return nil, errClientIDRequired
	}
	var resp *FiatOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, createWithdrawalOrderEPL, http.MethodPost, "fiat/create-withdrawal", req, &resp, request.AuthenticatedRequest)
}

// CancelFiatWithdrawalOrder cancels a pending fiat withdrawal order, which OKX supports for TRY only
func (e *Exchange) CancelFiatWithdrawalOrder(ctx context.Context, orderID string) (*CancelFiatWithdrawalOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *CancelFiatWithdrawalOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, cancelWithdrawalOrderEPL, http.MethodPost, "fiat/cancel-withdrawal", &CancelFiatWithdrawalOrderRequest{OrderID: orderID}, &resp, request.AuthenticatedRequest)
}

// GetFiatWithdrawalOrderHistory returns fiat withdrawal orders
func (e *Exchange) GetFiatWithdrawalOrderHistory(ctx context.Context, req *FiatOrderHistoryRequest) ([]FiatOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params, err := fiatOrderHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp []FiatOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getFiatWithdrawalOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("fiat/withdrawal-order-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatWithdrawalOrderDetail returns a fiat withdrawal order
func (e *Exchange) GetFiatWithdrawalOrderDetail(ctx context.Context, orderID string) (*FiatOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	var resp *FiatOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getWithdrawalOrderDetailEPL, http.MethodGet, common.EncodeURLValues("fiat/withdrawal", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatDepositOrderHistory returns fiat deposit orders
func (e *Exchange) GetFiatDepositOrderHistory(ctx context.Context, req *FiatOrderHistoryRequest) ([]FiatOrderResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	params, err := fiatOrderHistoryParams(req)
	if err != nil {
		return nil, err
	}
	var resp []FiatOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositOrderHistoryEPL, http.MethodGet, common.EncodeURLValues("fiat/deposit-order-history", params), nil, &resp, request.AuthenticatedRequest)
}

// GetFiatDepositOrderDetail returns a fiat deposit order
func (e *Exchange) GetFiatDepositOrderDetail(ctx context.Context, orderID string) (*FiatOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("ordId", orderID)
	var resp *FiatOrderResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getDepositOrderDetailEPL, http.MethodGet, common.EncodeURLValues("fiat/deposit", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBuySellCurrencies returns the fiat and crypto currencies fiat buy and sell trades support
func (e *Exchange) GetBuySellCurrencies(ctx context.Context) (*BuySellCurrenciesResponse, error) {
	var resp *BuySellCurrenciesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBuySellCurrenciesEPL, http.MethodGet, "fiat/buy-sell/currencies", nil, &resp, request.AuthenticatedRequest)
}

// GetBuySellCurrencyPair returns a fiat buy and sell currency pair's trade limits and payment methods
func (e *Exchange) GetBuySellCurrencyPair(ctx context.Context, req *BuySellCurrencyPairRequest) (*BuySellCurrencyPairResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("fromCcy", req.FromCurrency.String())
	params.Set("toCcy", req.ToCurrency.String())
	var resp *BuySellCurrencyPairResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBuySellCurrencyPairEPL, http.MethodGet, common.EncodeURLValues("fiat/buy-sell/currency-pair", params), nil, &resp, request.AuthenticatedRequest)
}

// GetBuySellQuote requests a fiat buy or sell quote, which PlaceBuySellTrade then fills
func (e *Exchange) GetBuySellQuote(ctx context.Context, req *BuySellQuoteRequest) (*BuySellQuoteResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	side, err := formatFundingSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.RequestForQuoteAmount <= 0 {
		return nil, fmt.Errorf("%w: request for quote amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.RequestForQuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: request for quote currency", currency.ErrCurrencyCodeEmpty)
	}
	body := *req
	body.Side = side
	var resp *BuySellQuoteResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBuySellQuoteEPL, http.MethodPost, "fiat/buy-sell/quote", &body, &resp, request.AuthenticatedRequest)
}

// PlaceBuySellTrade fills a fiat buy or sell quote from GetBuySellQuote
func (e *Exchange) PlaceBuySellTrade(ctx context.Context, req *BuySellTradeRequest) (*BuySellTradeResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if req.QuoteID == "" {
		return nil, errQuoteIDRequired
	}
	side, err := formatFundingSide(req.Side)
	if err != nil {
		return nil, err
	}
	if req.FromCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: from currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.ToCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: to currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.RequestForQuoteAmount <= 0 {
		return nil, fmt.Errorf("%w: request for quote amount must be greater than 0", limits.ErrAmountBelowMin)
	}
	if req.RequestForQuoteCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w: request for quote currency", currency.ErrCurrencyCodeEmpty)
	}
	if req.PaymentMethod == "" {
		return nil, errPaymentMethodRequired
	}
	if req.ClientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	body := *req
	body.Side = side
	var resp *BuySellTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, placeBuySellTradeEPL, http.MethodPost, "fiat/buy-sell/trade", &body, &resp, request.AuthenticatedRequest)
}

// GetBuySellTradeHistory returns fiat buy and sell trades
func (e *Exchange) GetBuySellTradeHistory(ctx context.Context, req *BuySellTradeHistoryRequest) ([]BuySellTradeHistoryResponse, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.OrderID != "" {
		params.Set("ordId", req.OrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("clOrdId", req.ClientOrderID)
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if !req.StartTime.IsZero() {
		params.Set("begin", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("end", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []BuySellTradeHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getBuySellTradeHistoryEPL, http.MethodGet, common.EncodeURLValues("fiat/buy-sell/history", params), nil, &resp, request.AuthenticatedRequest)
}

// formatFundingCurrencies joins a currency filter into OKX's comma-separated form, rejecting empty codes and, when
// maximum is above zero, a list longer than it
func formatFundingCurrencies(currencies currency.Currencies, maximum int) (string, error) {
	if maximum > 0 && len(currencies) > maximum {
		return "", fmt.Errorf("%w: at most %d currencies", errExceedLimit, maximum)
	}
	for i := range currencies {
		if currencies[i].IsEmpty() {
			return "", currency.ErrCurrencyCodeEmpty
		}
	}
	return currencies.Join(), nil
}

// checkFundingTimeWindow rejects a time filter whose bounds are both set and out of order
func checkFundingTimeWindow(start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return common.ErrStartAfterEnd
	}
	return nil
}

// assetBillsParams returns the filters asset bills details and asset bills history share
func assetBillsParams(ccy currency.Code, billType, thirdPartyType, clientID string, limit uint64) url.Values {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("ccy", ccy.String())
	}
	if billType != "" {
		params.Set("type", billType)
	}
	if thirdPartyType != "" {
		params.Set("thirdPartyType", thirdPartyType)
	}
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params
}

// fiatOrderHistoryParams returns the query of the fiat order history endpoints, whose after and before filters are
// the inclusive start and end of the window, unlike the paging cursors of other endpoints
func fiatOrderHistoryParams(req *FiatOrderHistoryRequest) (url.Values, error) {
	if err := checkFundingTimeWindow(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !req.Currency.IsEmpty() {
		params.Set("ccy", req.Currency.String())
	}
	if req.PaymentMethod != "" {
		params.Set("paymentMethod", req.PaymentMethod)
	}
	if req.State != "" {
		params.Set("state", req.State)
	}
	if !req.StartTime.IsZero() {
		params.Set("after", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("before", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	return params, nil
}

// formatFundingSide returns a convert or fiat buy and sell side in the lower case OKX takes
func formatFundingSide(side string) (string, error) {
	switch s := strings.ToLower(side); s {
	case order.Buy.Lower(), order.Sell.Lower():
		return s, nil
	default:
		return "", fmt.Errorf("%w: %q", order.ErrSideIsInvalid, side)
	}
}

// formatStatementMonth returns the three letter month name the monthly statement endpoints take
func formatStatementMonth(month time.Month) (string, error) {
	if month < time.January || month > time.December {
		return "", fmt.Errorf("%w: %d", errInvalidMonth, month)
	}
	return month.String()[:3], nil
}
