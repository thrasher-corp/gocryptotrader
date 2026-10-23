package gateio

import (
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// ListCurrencyChain retrieves the chains supported by a currency
func (e *Exchange) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]*CurrencyChainResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []*CurrencyChainResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyChainEPL, common.EncodeURLValues("wallet/currency_chains", params), &resp)
}

// GenerateCurrencyDepositAddress generates a deposit address for a currency
func (e *Exchange) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfoResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyDepositAddressInfoResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositAddressEPL, http.MethodGet, "wallet/deposit_address", params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. The time range cannot exceed 30 days and defaults to the last 7 days
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, arg *WithdrawalRecordsRequest) ([]*WithdrawalResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
			return nil, err
		}
		if !arg.Currency.IsEmpty() {
			params.Set("currency", arg.Currency.String())
		}
		if arg.WithdrawalID != "" {
			params.Set("withdraw_id", arg.WithdrawalID)
		}
		if arg.AssetClass != "" {
			params.Set("asset_class", arg.AssetClass)
		}
		if arg.WithdrawalOrderID != "" {
			params.Set("withdraw_order_id", arg.WithdrawalOrderID)
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Offset > 0 {
			params.Set("offset", strconv.FormatUint(arg.Offset, 10))
		}
	}
	var withdrawals []*WithdrawalResponse
	return withdrawals, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawalRecordsEPL, http.MethodGet, "wallet/withdrawals", params, nil, &withdrawals)
}

// GetDepositRecords retrieves deposit records. The time range cannot exceed 30 days and defaults to the last 7 days
func (e *Exchange) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]*DepositRecordResponse, error) {
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var depositHistories []*DepositRecordResponse
	return depositHistories, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositRecordsEPL, http.MethodGet, "wallet/deposits", params, nil, &depositHistories)
}

// TransferCurrency transfers funds between the user's trading accounts. Gate supports transfers between the spot
// account and the margin, perpetual futures, delivery futures and options accounts
func (e *Exchange) TransferCurrency(ctx context.Context, arg *TransferCurrencyRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	from, err := walletAccountType(arg.From)
	if err != nil {
		return nil, fmt.Errorf("%w: source account", err)
	}
	to, err := walletAccountType(arg.To)
	if err != nil {
		return nil, fmt.Errorf("%w: destination account", err)
	}
	if (from == "margin" || to == "margin") && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: currency pair is required for margin account transfer", currency.ErrCurrencyPairEmpty)
	}
	if (from == futuresAccount || from == deliveryAccount || to == futuresAccount || to == deliveryAccount) && arg.Settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settle is required for futures account transfer", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var response *TransactionIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferCurrencyEPL, http.MethodPost, "wallet/transfers", nil, arg, &response)
}

// GetTradingAccountTransfer retrieves a transfer between the user's trading accounts by the transaction ID returned
// from TransferCurrency
func (e *Exchange) GetTradingAccountTransfer(ctx context.Context, transactionID string) (*TradingAccountTransferResponse, error) {
	if transactionID == "" {
		return nil, errTransactionIDRequired
	}
	params := url.Values{}
	params.Set("tx_id", transactionID)
	var resp *TradingAccountTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletGetTransferEPL, http.MethodGet, "wallet/transfers", params, nil, &resp)
}

// SubAccountTransfer transfers funds between the main account and a sub-account's trading account. The main
// account side always uses its spot account, whichever sub-account account is chosen
func (e *Exchange) SubAccountTransfer(ctx context.Context, arg *SubAccountTransferRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccount == "" {
		return nil, errInvalidSubAccount
	}
	arg.Direction = strings.ToLower(arg.Direction)
	if arg.Direction != "to" && arg.Direction != "from" {
		return nil, errInvalidTransferDirection
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	switch arg.SubAccountType {
	case "", spotAccount, futuresAccount, deliveryAccount, "options":
	default:
		return nil, fmt.Errorf("%w %q for SubAccountTransfer; Supported: [spot, futures, delivery, options]", asset.ErrNotSupported, arg.SubAccountType)
	}
	var resp *TransactionIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferEPL, http.MethodPost, "wallet/sub_account_transfers", nil, arg, &resp)
}

// GetSubAccountTransferHistory retrieves transfer records between main and sub accounts.
// Records begin on 2020-04-10 and the query range cannot exceed 30 days.
func (e *Exchange) GetSubAccountTransferHistory(ctx context.Context, subAccountUserID string, from, to time.Time, offset, limit uint64) ([]*SubAccountTransferResponse, error) {
	if !from.IsZero() && from.Before(time.Date(2020, time.April, 10, 0, 0, 0, 0, time.UTC)) {
		return nil, errSubAccountTransferHistoryStart
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	// Only whole seconds are sent, so the limit applies to those
	if !from.IsZero() && !to.IsZero() && to.Truncate(time.Second).Sub(from.Truncate(time.Second)) > 30*24*time.Hour {
		return nil, errSubAccountTransferHistoryRange
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*SubAccountTransferResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferHistoryEPL, http.MethodGet, "wallet/sub_account_transfers", params, nil, &response)
}

// SubAccountTransferToSubAccount transfers funds between two sub-accounts of the same main account, using either the
// main account's API key or the source sub-account's
func (e *Exchange) SubAccountTransferToSubAccount(ctx context.Context, arg *InterSubAccountTransferRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccountFromUserID == "" {
		return nil, fmt.Errorf("%w: sub-account from user-id is required", errInvalidSubAccountUserID)
	}
	if err := checkSubAccountTransferAccountType(arg.SubAccountFromAssetType); err != nil {
		return nil, fmt.Errorf("%w: source sub-account", err)
	}
	if arg.SubAccountToUserID == "" {
		return nil, fmt.Errorf("%w: sub-account to user-id is required", errInvalidSubAccountUserID)
	}
	if err := checkSubAccountTransferAccountType(arg.SubAccountToAssetType); err != nil {
		return nil, fmt.Errorf("%w: destination sub-account", err)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *TransactionIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountToSubAccountTransferEPL, http.MethodPost, "wallet/sub_account_to_sub_account", nil, arg, &resp)
}

// checkSubAccountTransferAccountType checks that a sub-account to sub-account transfer uses an account Gate supports
func checkSubAccountTransferAccountType(a asset.Item) error {
	account, err := walletAccountType(a)
	if err != nil {
		return err
	}
	if account != spotAccount && account != futuresAccount && account != deliveryAccount {
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return nil
}

// GetTransferOrderStatus retrieves the status of a main and sub-account transfer by its client order ID or by the
// transaction ID the transfer returned
func (e *Exchange) GetTransferOrderStatus(ctx context.Context, clientOrderID, transactionID string) (*TransferStatusResponse, error) {
	if clientOrderID == "" && transactionID == "" {
		return nil, errTransferStatusIDRequired
	}
	params := url.Values{}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if transactionID != "" {
		params.Set("tx_id", transactionID)
	}
	var resp *TransferStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferOrderStatusEPL, http.MethodGet, "wallet/order_status", params, nil, &resp)
}

// GetWithdrawalStatus retrieves withdrawal fees and limits, for all currencies when ccy is empty
func (e *Exchange) GetWithdrawalStatus(ctx context.Context, ccy currency.Code) ([]*WithdrawalStatusResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*WithdrawalStatusResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawStatusEPL, http.MethodGet, "wallet/withdraw_status", params, nil, &response)
}

// GetSubAccountBalances retrieves sub-account spot balances. subAccountUserID accepts several comma-separated user IDs
// and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountBalances(ctx context.Context, subAccountUserID string, page, limit uint64) ([]*SubAccountBalancesResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*SubAccountBalancesResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountBalancesEPL, http.MethodGet, "wallet/sub_account_balances", params, nil, &resp)
}

// GetSubAccountMarginBalances retrieves sub-account isolated margin balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountMarginBalanceResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountMarginBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountMarginBalancesEPL, http.MethodGet, "wallet/sub_account_margin_balances", params, nil, &response)
}

// GetSubAccountFuturesBalances retrieves sub-account perpetual futures balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountFuturesBalances(ctx context.Context, subAccountUserID string, settle currency.Code) ([]*FuturesSubAccountBalanceResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response []*FuturesSubAccountBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountFuturesBalancesEPL, http.MethodGet, "wallet/sub_account_futures_balances", params, nil, &response)
}

// GetSubAccountCrossMarginBalances retrieves sub-account cross margin balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountCrossMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountCrossMarginInfoResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountCrossMarginInfoResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountCrossMarginBalancesEPL, http.MethodGet, "wallet/sub_account_cross_margin_balances", params, nil, &response)
}

// GetSavedAddresses retrieves saved withdrawal addresses
func (e *Exchange) GetSavedAddresses(ctx context.Context, arg *SavedAddressesRequest) ([]*WalletSavedAddressResponse, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.Currency.IsEmpty() {
			params.Set("currency", arg.Currency.String())
		}
		if arg.Chain != "" {
			params.Set("chain", arg.Chain)
		}
		if arg.Verified != "" {
			params.Set("verified", arg.Verified)
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
	}
	var response []*WalletSavedAddressResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSavedAddressesEPL, http.MethodGet, "wallet/saved_address", params, nil, &response)
}

// GetPersonalTradingFee retrieves the user's trading fee rates. Fees are usually the same for every currency pair and
// settlement currency, which narrow the query when set
func (e *Exchange) GetPersonalTradingFee(ctx context.Context, currencyPair currency.Pair, settle currency.Code) (*PersonalTradingFeeResponse, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Upper().String())
	}
	var response *PersonalTradingFeeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTradingFeeEPL, http.MethodGet, "wallet/fee", params, nil, &response)
}

// GetUsersTotalBalance retrieves the estimated value of every account converted to ccy, which accepts BTC, CNY, USD
// and USDT and defaults to USDT. Gate caches the values for up to a minute, so use the account balance endpoints for
// real-time figures
func (e *Exchange) GetUsersTotalBalance(ctx context.Context, ccy currency.Code) (*UsersAllAccountBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response *UsersAllAccountBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTotalBalanceEPL, http.MethodGet, "wallet/total_balance", params, nil, &response)
}

// GetConvertibleSmallBalanceCurrencies retrieves the small balances that can be converted into GT
func (e *Exchange) GetConvertibleSmallBalanceCurrencies(ctx context.Context) ([]*ConvertibleSmallBalanceResponse, error) {
	var resp []*ConvertibleSmallBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSmallBalancesEPL, http.MethodGet, "wallet/small_balance", nil, nil, &resp)
}

// ConvertSmallBalances converts small balances of provided currencies into GT.
// If no currencies are provided, all supported currencies will be converted
// See [this documentation](https://www.gate.io/help/guide/functional_guidelines/22367) for details and restrictions.
func (e *Exchange) ConvertSmallBalances(ctx context.Context, currs ...currency.Code) error {
	currencyList := make([]string, len(currs))
	for i := range currs {
		if currs[i].IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
		currencyList[i] = currs[i].Upper().String()
	}

	payload := &convertSmallBalancesRequest{
		Currency: currencyList,
		IsAll:    len(currs) == 0,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletConvertSmallBalancesEPL, http.MethodPost, "wallet/small_balance", nil, payload, nil)
}

// GetConvertibleSmallBalanceCurrencyHistory retrieves small balance conversion records
func (e *Exchange) GetConvertibleSmallBalanceCurrencyHistory(ctx context.Context, ccy currency.Code, page, limit uint64) ([]*SmallCurrencyBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*SmallCurrencyBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSmallBalanceHistoryEPL, http.MethodGet, "wallet/small_balance_history", params, nil, &resp)
}

// GetUIDTransferHistory retrieves UID transfer records. The time range defaults to the last 7 days
func (e *Exchange) GetUIDTransferHistory(ctx context.Context, arg *UIDTransferHistoryRequest) ([]*UIDTransferRecordResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
			return nil, err
		}
		if arg.ID != 0 {
			params.Set("id", strconv.FormatUint(arg.ID, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Offset > 0 {
			params.Set("offset", strconv.FormatUint(arg.Offset, 10))
		}
		if arg.TransactionType != "" {
			params.Set("transaction_type", arg.TransactionType)
		}
	}
	var resp []*UIDTransferRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletUIDTransferHistoryEPL, http.MethodGet, "wallet/push", params, nil, &resp)
}

// GetLowCapExchangeList retrieves the low-liquidity or low-cap currencies that the API does not support, which must be
// managed through the website or app instead
func (e *Exchange) GetLowCapExchangeList(ctx context.Context) ([]currency.Code, error) {
	var resp []currency.Code
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletLowCapExchangeListEPL, http.MethodGet, "wallet/getLowCapExchangeList", nil, nil, &resp)
}
