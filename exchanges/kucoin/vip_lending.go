package kucoin

import (
	"context"
	"net/http"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetVIPLendingCollateralRatio returns each currency's gradient collateral ratios (Get Collateral Ratio)
func (e *Exchange) GetVIPLendingCollateralRatio(ctx context.Context) ([]VIPLendingCollateralRatio, error) {
	var resp []VIPLendingCollateralRatio
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getVIPLendingCollateralRatioEPL, http.MethodGet, "/v1/otc-loan/discount-rate-configs", nil, &resp)
}

// GetInformationOnOffExchangeFundingAndLoans returns the account's off-exchange (VIP) loans, their loan-to-value
// ratios and margins; only accounts currently involved in a loan have one (Get Loan Info)
func (e *Exchange) GetInformationOnOffExchangeFundingAndLoans(ctx context.Context) (*VIPLendingLoanInfoResponse, error) {
	var resp *VIPLendingLoanInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getInformationOnOffExchangeFundingAndLoansEPL, http.MethodGet, "/v1/otc-loan/loan", nil, &resp)
}

// GetInformationOnAccountInvolvedInOffExchangeLoans returns the accounts currently involved in off-exchange (VIP)
// lending (Get Accounts)
func (e *Exchange) GetInformationOnAccountInvolvedInOffExchangeLoans(ctx context.Context) ([]VIPLendingAccount, error) {
	var resp []VIPLendingAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, getInformationOnAccountInvolvedInOffExchangeLoansEPL, http.MethodGet, "/v1/otc-loan/accounts", nil, &resp)
}
