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
)

var errToAccountTypeRequired = errors.New("to account type required")

// GetFundingAccountTransactionHistory returns the Funding wallet's transaction log (Funding Account Transaction
// History). Without a window the API returns the last 7 days; a window needs both bounds and spans at most 7 days
func (e *Exchange) GetFundingAccountTransactionHistory(ctx context.Context, arg *FundingAccountTransactionHistoryRequest) (*FundingAccountTransactionHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.StartTime.IsZero() || !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		params.Set("createTimeFrom", strconv.FormatInt(arg.StartTime.Unix(), 10))
		params.Set("createTimeTo", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *FundingAccountTransactionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/fundinghistory", params, nil, &resp, getFundingAccountTransactionHistoryEPL)
}

// GetDeliveryRecord returns delivery records of inverse futures, USDT and USDC futures and options, newest first (Get
// Delivery Record)
func (e *Exchange) GetDeliveryRecord(ctx context.Context, arg *DeliveryRecordRequest) (*DeliveryRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Category == "" {
		return nil, errCategoryNotSet
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", arg.Category)
	if arg.Symbol != "" {
		params.Set("symbol", arg.Symbol)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if !arg.ExpiryDate.IsZero() {
		params.Set("expDate", expiryDate(arg.ExpiryDate))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *DeliveryRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/delivery-record", params, nil, &resp, getDeliveryRecordEPL)
}

// GetUSDCSessionSettlement returns session settlement records of USDC perpetuals and futures (Get USDC Session
// Settlement)
func (e *Exchange) GetUSDCSessionSettlement(ctx context.Context, arg *USDCSessionSettlementRequest) (*USDCSessionSettlementResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Category == "" {
		return nil, errCategoryNotSet
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("category", arg.Category)
	if arg.Symbol != "" {
		params.Set("symbol", arg.Symbol)
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *USDCSessionSettlementResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/settlement-record", params, nil, &resp, getUSDCSessionSettlementEPL)
}

// GetCoinExchangeRecords returns the coin exchange records (Get Coin Exchange Records)
func (e *Exchange) GetCoinExchangeRecords(ctx context.Context, arg *CoinExchangeRecordsRequest) (*CoinExchangeRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.FromCoin.IsEmpty() {
		params.Set("fromCoin", arg.FromCoin.Upper().String())
	}
	if !arg.ToCoin.IsEmpty() {
		params.Set("toCoin", arg.ToCoin.Upper().String())
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Cursor != "" {
		params.Set("cursor", arg.Cursor)
	}
	var resp *CoinExchangeRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/order-record", params, nil, &resp, getCoinExchangeRecordsEPL)
}

// GetCoinInfo returns coin information, including each chain's deposit and withdrawal status, for one coin or for
// every coin when coin is empty (Get Coin Info)
func (e *Exchange) GetCoinInfo(ctx context.Context, coin currency.Code) (*CoinInfoResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *CoinInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin/query-info", params, nil, &resp, getCoinInfoEPL)
}

// GetSubUID returns up to 2000 sub UIDs under the master UID; it needs the master UID's API key (Get Sub UID)
func (e *Exchange) GetSubUID(ctx context.Context) (*SubUIDResponse, error) {
	var resp *SubUIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-sub-member-list", nil, nil, &resp, getSubUIDEPL)
}

// GetPortfolioMarginInfo returns the portfolio margin account's wallet balance and its P&L ranges per base coin, for
// every base coin when baseCoin is empty (Get Portfolio Margin Info)
func (e *Exchange) GetPortfolioMarginInfo(ctx context.Context, baseCoin currency.Code) (*PortfolioMarginInfoResponse, error) {
	params := url.Values{}
	if !baseCoin.IsEmpty() {
		params.Set("baseCoin", baseCoin.Upper().String())
	}
	var resp *PortfolioMarginInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/portfolio-margin", params, nil, &resp, getPortfolioMarginInfoEPL)
}

// GetTotalMembersAssets returns the total assets of the master account and its sub-accounts, valued in coin, or in
// BTC when coin is empty (Get Total Members Assets)
func (e *Exchange) GetTotalMembersAssets(ctx context.Context, coin currency.Code) (*TotalMembersAssetsResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
	}
	var resp *TotalMembersAssetsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/total-members-assets", params, nil, &resp, getTotalMembersAssetsEPL)
}

// GetSingleCoinBalance returns the balance of one coin in one account type, including a sub UID's balance and the
// amount transferable to another UID or account type (Get Single Coin Balance)
func (e *Exchange) GetSingleCoinBalance(ctx context.Context, arg *SingleCoinBalanceRequest) (*SingleCoinBalanceResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.WithLTVTransferSafeAmount && arg.ToAccountType == "" {
		return nil, errToAccountTypeRequired
	}
	params := url.Values{}
	if arg.MemberID != "" {
		params.Set("memberId", arg.MemberID)
	}
	if arg.ToMemberID != "" {
		params.Set("toMemberId", arg.ToMemberID)
	}
	params.Set("accountType", arg.AccountType)
	if arg.ToAccountType != "" {
		params.Set("toAccountType", arg.ToAccountType)
	}
	params.Set("coin", arg.Coin.Upper().String())
	if arg.WithBonus {
		params.Set("withBonus", "1")
	}
	if arg.WithTransferSafeAmount {
		params.Set("withTransferSafeAmount", "1")
	}
	if arg.WithLTVTransferSafeAmount {
		params.Set("withLtvTransferSafeAmount", "1")
	}
	var resp *SingleCoinBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coin-balance", params, nil, &resp, getSingleCoinBalanceEPL)
}

// GetAllCoinsBalance returns the balance of every coin, or of up to 10 listed coins, in one account type of the master
// account or a sub-account; the Unified account needs the coins listed (Get All Coins Balance)
func (e *Exchange) GetAllCoinsBalance(ctx context.Context, arg *AllCoinsBalanceRequest) (*AllCoinsBalanceResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AccountType == "" {
		return nil, errMissingAccountType
	}
	// The unified account is queried coin by coin, up to ten at a time
	if arg.AccountType == "UNIFIED" && len(arg.Coins) == 0 {
		return nil, fmt.Errorf("%w for the UNIFIED account", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if arg.MemberID != "" {
		params.Set("memberId", arg.MemberID)
	}
	params.Set("accountType", arg.AccountType)
	if len(arg.Coins) > 0 {
		coins := make([]string, len(arg.Coins))
		for i := range arg.Coins {
			if arg.Coins[i].IsEmpty() {
				return nil, fmt.Errorf("%w: coin %d", currency.ErrCurrencyCodeEmpty, i)
			}
			coins[i] = arg.Coins[i].Upper().String()
		}
		params.Set("coin", strings.Join(coins, ","))
	}
	if arg.WithBonus {
		params.Set("withBonus", "1")
	}
	var resp *AllCoinsBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coins-balance", params, nil, &resp, getAllCoinsBalanceEPL)
}

// GetWithdrawableAmount returns the amount of coin each wallet can withdraw, and the amount frozen by risk control
// (Get Withdrawable Amount)
func (e *Exchange) GetWithdrawableAmount(ctx context.Context, coin currency.Code) (*WithdrawableAmountResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Upper().String())
	var resp *WithdrawableAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/withdrawable-amount", params, nil, &resp, getWithdrawableAmountEPL)
}

// GetAssetOverview returns the master account's or a sub-account's total equity and its holdings per account type and
// product category; regional accounts such as Bybit EU are not supported (Asset Overview)
func (e *Exchange) GetAssetOverview(ctx context.Context, arg *AssetOverviewRequest) (*AssetOverviewResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.MemberID != "" {
		params.Set("memberId", arg.MemberID)
	}
	if !arg.ValuationCurrency.IsEmpty() {
		params.Set("valuationCurrency", arg.ValuationCurrency.Upper().String())
	}
	if arg.AccountType != "" {
		params.Set("accountType", arg.AccountType)
	}
	var resp *AssetOverviewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/asset-overview", params, nil, &resp, getAssetOverviewEPL)
}
