package kucoin

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetUTAVIPLendingCollateralRatio returns each collateral currency's discount rate tiers. The account type is UNIFIED
// for the unified account, or empty for the classic account (Get Collateral Ratio)
func (e *Exchange) GetUTAVIPLendingCollateralRatio(ctx context.Context, accountType string) ([]UTAVIPLendingCollateralRatio, error) {
	var resp []UTAVIPLendingCollateralRatio
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAVIPLendingCollateralRatioEPL, http.MethodGet, utaVIPLendingPath("/ua/v2/otc-loan/discount-rate", accountType), nil, &resp)
}

// GetUTAVIPLendingLoanInfo returns the off-exchange loans of the accounts in a loan, their loan-to-value ratios and
// margin. The account type is UNIFIED for the unified account, or empty for the classic account (Get Loan Info)
func (e *Exchange) GetUTAVIPLendingLoanInfo(ctx context.Context, accountType string) (*UTAVIPLendingLoanInfoResponse, error) {
	var resp *UTAVIPLendingLoanInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAVIPLendingLoanInfoEPL, http.MethodGet, utaVIPLendingPath("/ua/v2/otc-loan/loan", accountType), nil, &resp)
}

// GetUTAVIPLendingAccounts returns the accounts in an off-exchange loan and their margin. The account type is UNIFIED
// for the unified account, or empty for the classic account (Get Accounts)
func (e *Exchange) GetUTAVIPLendingAccounts(ctx context.Context, accountType string) ([]UTAVIPLendingAccount, error) {
	var resp []UTAVIPLendingAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getUTAVIPLendingAccountsEPL, http.MethodGet, utaVIPLendingPath("/ua/v2/otc-loan/account", accountType), nil, &resp)
}

// utaVIPLendingPath adds the optional account type the VIP lending endpoints share to a path
func utaVIPLendingPath(path, accountType string) string {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	return common.EncodeURLValues(path, params)
}
