package binance

import (
	"context"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// PortfolioMarginProBNBTransfer transfers BNB between the portfolio margin pro margin and USDⓈ-M accounts; transferSide
// is TO_UM or FROM_UM. The endpoint can be called twice per 10 minutes
func (e *Exchange) PortfolioMarginProBNBTransfer(ctx context.Context, amount float64, transferSide string) (*PortfolioMarginTransactionResponse, error) {
	return e.portfolioMarginTransferBNB(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/bnb-transfer", amount, transferSide, transferBNBRate)
}

// GetPortfolioMarginProAutoRepayFuturesStatus returns whether portfolio margin pro futures negative balances are repaid
// automatically
func (e *Exchange) GetPortfolioMarginProAutoRepayFuturesStatus(ctx context.Context) (*PortfolioMarginAutoRepayFuturesStatusResponse, error) {
	var resp *PortfolioMarginAutoRepayFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/repay-futures-switch", nil, getAutoRepayFuturesStatusRate, &resp)
}

// ChangePortfolioMarginProAutoRepayFuturesStatus turns the automatic repayment of portfolio margin pro futures negative
// balances on or off. The endpoint can be called 20 times a day
func (e *Exchange) ChangePortfolioMarginProAutoRepayFuturesStatus(ctx context.Context, autoRepay bool) (*PortfolioMarginMessageResponse, error) {
	return e.portfolioMarginChangeAutoRepayFuturesStatus(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/repay-futures-switch", autoRepay, changeAutoRepayFuturesStatusRate)
}

// PortfolioMarginProFundAutoCollection collects every asset except BNB from the portfolio margin pro futures accounts
// into the margin account. The endpoint can be called 500 times per hour
func (e *Exchange) PortfolioMarginProFundAutoCollection(ctx context.Context) (*PortfolioMarginMessageResponse, error) {
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/auto-collection", nil, fundAutoCollectionRate, &resp)
}

// PortfolioMarginProFundCollectionByAsset collects an asset other than BNB from the portfolio margin pro futures
// accounts into the margin account
func (e *Exchange) PortfolioMarginProFundCollectionByAsset(ctx context.Context, ccy currency.Code) (*PortfolioMarginMessageResponse, error) {
	return e.portfolioMarginFundCollectionByAsset(ctx, exchange.RestSpotSupplementary, "/sapi/v1/portfolio/asset-collection", ccy, fundCollectionByAssetRate)
}

// GetPortfolioMarginProAccountInfo returns the portfolio margin pro account information
func (e *Exchange) GetPortfolioMarginProAccountInfo(ctx context.Context) (*PortfolioMarginProAccountInfoResponse, error) {
	var resp *PortfolioMarginProAccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/account", nil, classicPortfolioMarginAccountInfoRate, &resp)
}

// RepayPortfolioMarginProBankruptcyLoan repays the portfolio margin pro bankruptcy loan from the SPOT (default) or
// MARGIN account; the API key needs spot and margin trading permissions
func (e *Exchange) RepayPortfolioMarginProBankruptcyLoan(ctx context.Context, from string) (*PortfolioMarginTransactionResponse, error) {
	params := url.Values{}
	if from != "" {
		params.Set("from", from)
	}
	var resp *PortfolioMarginTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/repay", params, repayClassicPortfolioMarginBankruptcyLoanRate, &resp)
}

// GetPortfolioMarginProBankruptcyLoanAmount returns the portfolio margin pro bankruptcy loan amount
func (e *Exchange) GetPortfolioMarginProBankruptcyLoanAmount(ctx context.Context) (*PortfolioMarginProBankruptcyLoanAmountResponse, error) {
	var resp *PortfolioMarginProBankruptcyLoanAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/pmLoan", nil, getClassicPortfolioMarginBankruptcyLoanAmountRate, &resp)
}

// GetPortfolioMarginProNegativeBalanceInterestHistory returns the interest charged on portfolio margin pro negative
// balances
func (e *Exchange) GetPortfolioMarginProNegativeBalanceInterestHistory(ctx context.Context, arg *PortfolioMarginNegativeBalanceInterestHistoryRequest) ([]PortfolioMarginProNegativeBalanceInterestResponse, error) {
	params, err := portfolioMarginNegativeBalanceInterestHistoryParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []PortfolioMarginProNegativeBalanceInterestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/interest-history", params, classicPortfolioMarginNegativeBalanceInterestHistoryRate, &resp)
}

// RepayPortfolioMarginProFuturesNegativeBalance repays portfolio margin pro futures negative balances from the SPOT
// (default) or MARGIN account
func (e *Exchange) RepayPortfolioMarginProFuturesNegativeBalance(ctx context.Context, from string) (*PortfolioMarginMessageResponse, error) {
	params := url.Values{}
	if from != "" {
		params.Set("from", from)
	}
	var resp *PortfolioMarginMessageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/portfolio/repay-futures-negative-balance", params, repayFuturesNegativeBalanceRate, &resp)
}

// GetPortfolioMarginAssetLeverage returns the leverage of every portfolio margin asset
func (e *Exchange) GetPortfolioMarginAssetLeverage(ctx context.Context) ([]PortfolioMarginAssetLeverageResponse, error) {
	var resp []PortfolioMarginAssetLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/margin-asset-leverage", nil, portfolioMarginAssetLeverageRate, &resp)
}

// GetPortfolioMarginCollateralRate returns the collateral rate of every portfolio margin asset
func (e *Exchange) GetPortfolioMarginCollateralRate(ctx context.Context) ([]PortfolioMarginCollateralRateResponse, error) {
	var resp []PortfolioMarginCollateralRateResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/portfolio/collateralRate", classicPortfolioMarginCollateralRate, &resp)
}

// GetPortfolioMarginAssetIndexPrice returns the USD index price of one portfolio margin asset, or of every asset when
// ccy is empty
func (e *Exchange) GetPortfolioMarginAssetIndexPrice(ctx context.Context, ccy currency.Code) ([]PortfolioMarginAssetIndexPriceResponse, error) {
	params := url.Values{}
	f := portfolioMarginAssetIndexPriceRate
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.Upper().String())
		f = sapiDefaultRate
	}
	var resp []PortfolioMarginAssetIndexPriceResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/portfolio/asset-index-price", params), f, &resp)
}
