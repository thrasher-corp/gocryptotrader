package binance

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// PMProBNBTransfer transfers BNB between the portfolio margin pro margin and USDⓈ-M accounts; transferSide is TO_UM or
// FROM_UM. The endpoint can be called twice per 10 minutes
func (e *Exchange) PMProBNBTransfer(ctx context.Context, amount float64, transferSide string) (*PMTransactionResponse, error) {
	return e.pmTransferBNB(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/bnb-transfer", amount, transferSide, transferBNBRate)
}

// GetPMProAutoRepayFuturesStatus returns whether portfolio margin pro futures negative balances are repaid
// automatically
func (e *Exchange) GetPMProAutoRepayFuturesStatus(ctx context.Context) (*PMAutoRepayFuturesStatusResponse, error) {
	var resp *PMAutoRepayFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/repay-futures-switch", nil, getAutoRepayFuturesStatusRate, &resp)
}

// ChangePMProAutoRepayFuturesStatus turns the automatic repayment of portfolio margin pro futures negative balances on
// or off. The endpoint can be called 20 times a day
func (e *Exchange) ChangePMProAutoRepayFuturesStatus(ctx context.Context, autoRepay bool) (*PMMessageResponse, error) {
	return e.pmChangeAutoRepayFuturesStatus(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/repay-futures-switch", autoRepay, changeAutoRepayFuturesStatusRate)
}

// PMProFundAutoCollection collects every asset except BNB from the portfolio margin pro futures accounts into the
// margin account. The endpoint can be called 500 times per hour
func (e *Exchange) PMProFundAutoCollection(ctx context.Context) (*PMMessageResponse, error) {
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/auto-collection", nil, fundAutoCollectionRate, &resp)
}

// PMProFundCollectionByAsset collects an asset other than BNB from the portfolio margin pro futures accounts into the
// margin account
func (e *Exchange) PMProFundCollectionByAsset(ctx context.Context, ccy currency.Code) (*PMMessageResponse, error) {
	return e.pmFundCollectionByAsset(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/asset-collection", ccy, fundCollectionByAssetRate)
}

// GetPMProAccountInfo returns the portfolio margin pro account information
func (e *Exchange) GetPMProAccountInfo(ctx context.Context) (*PMProAccountInfoResponse, error) {
	var resp *PMProAccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/account", nil, classicPMAccountInfoRate, &resp)
}

// RepayPMProBankruptcyLoan repays the portfolio margin pro bankruptcy loan from the SPOT (default) or MARGIN account;
// the API key needs spot and margin trading permissions
func (e *Exchange) RepayPMProBankruptcyLoan(ctx context.Context, from string) (*PMTransactionResponse, error) {
	params := url.Values{}
	if from != "" {
		params.Set("from", from)
	}
	var resp *PMTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/repay", params, repayClassicPMBankruptcyLoanRate, &resp)
}

// GetPMProBankruptcyLoanAmount returns the portfolio margin pro bankruptcy loan amount
func (e *Exchange) GetPMProBankruptcyLoanAmount(ctx context.Context) (*PMProBankruptcyLoanAmountResponse, error) {
	var resp *PMProBankruptcyLoanAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/pmLoan", nil, getClassicPMBankruptcyLoanAmountRate, &resp)
}

// GetPMProNegativeBalanceInterestHistory returns the interest charged on portfolio margin pro negative balances
func (e *Exchange) GetPMProNegativeBalanceInterestHistory(ctx context.Context, arg *PMNegativeBalanceInterestHistoryRequest) ([]PMProNegativeBalanceInterest, error) {
	params, err := pmNegativeBalanceInterestHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []PMProNegativeBalanceInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/interest-history", params, classicPMNegativeBalanceInterestHistoryRate, &resp)
}

// RepayPMProFuturesNegativeBalance repays portfolio margin pro futures negative balances from the SPOT (default) or
// MARGIN account
func (e *Exchange) RepayPMProFuturesNegativeBalance(ctx context.Context, from string) (*PMMessageResponse, error) {
	params := url.Values{}
	if from != "" {
		params.Set("from", from)
	}
	var resp *PMMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/repay-futures-negative-balance", params, repayFuturesNegativeBalanceRate, &resp)
}

// GetPortfolioMarginAssetLeverage returns the leverage of every portfolio margin asset
func (e *Exchange) GetPortfolioMarginAssetLeverage(ctx context.Context) ([]PMAssetLeverage, error) {
	var resp []PMAssetLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/margin-asset-leverage", nil, pmAssetLeverageRate, &resp)
}

// GetPortfolioMarginCollateralRate returns the collateral rate of every portfolio margin asset
func (e *Exchange) GetPortfolioMarginCollateralRate(ctx context.Context) ([]PMCollateralRate, error) {
	var resp []PMCollateralRate
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/collateralRate", classicPMCollateralRate, &resp)
}

// GetPortfolioMarginAssetIndexPrice returns the USD index price of one portfolio margin asset, or of every asset when
// ccy is empty
func (e *Exchange) GetPortfolioMarginAssetIndexPrice(ctx context.Context, ccy currency.Code) ([]PortfolioMarginAssetIndexPrice, error) {
	params := url.Values{}
	f := pmAssetIndexPriceRate
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
		f = sapiDefaultRate
	}
	var resp []PortfolioMarginAssetIndexPrice
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/portfolio/asset-index-price", params), f, &resp)
}
