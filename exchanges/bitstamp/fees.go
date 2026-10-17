package bitstamp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		tradingFee, err := e.getTradingFee(ctx, feeBuilder)
		if err != nil {
			return 0, fmt.Errorf("error getting trading fee: %w", err)
		}
		fee = tradingFee
	case exchange.CryptocurrencyWithdrawalFee:
		withdrawalFee, err := e.GetWithdrawalFee(ctx, feeBuilder.Pair.Base, "")
		if err != nil {
			return 0, fmt.Errorf("error getting withdrawal fee: %w", err)
		}
		fee = withdrawalFee.Fee.Float64()
	case exchange.CryptocurrencyDepositFee:
		fee = 0
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getTradingFee returns a trading fee based on a currency
func (e *Exchange) getTradingFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	tradingFee, err := e.GetTradingFee(ctx, feeBuilder.Pair)
	if err != nil {
		return 0, err
	}
	fee := tradingFee.Fees.Taker.Float64()
	if feeBuilder.IsMaker {
		fee = tradingFee.Fees.Maker.Float64()
	}
	return fee / 100 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0025 * price * amount
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.0009

	if fee < 15 {
		return 15
	}
	return fee
}

// getInternationalBankDepositFee returns international deposit fee
func getInternationalBankDepositFee(amount float64) float64 {
	fee := amount * 0.0005

	if fee < 7.5 {
		return 7.5
	}
	if fee > 300 {
		return 300
	}
	return fee
}

// GetTradingFees returns the trading fees for all markets
func (e *Exchange) GetTradingFees(ctx context.Context) ([]TradingFeeResponse, error) {
	var resp []TradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/trading/", nil, nil, &resp)
}

// GetTradingFee returns the trading fees for a market
func (e *Exchange) GetTradingFee(ctx context.Context, pair currency.Pair) (*TradingFeeResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *TradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/trading/"+formatMarketSymbol(pair)+"/", nil, nil, &resp)
}

// GetWithdrawalFees returns the withdrawal fees for all currencies
func (e *Exchange) GetWithdrawalFees(ctx context.Context) ([]WithdrawalFeeResponse, error) {
	var resp []WithdrawalFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/withdrawal/", nil, nil, &resp)
}

// GetWithdrawalFee returns the withdrawal fee for a currency
// network is optional and selects the currency network the fee applies to
func (e *Exchange) GetWithdrawalFee(ctx context.Context, c currency.Code, network string) (*WithdrawalFeeResponse, error) {
	if c.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if network != "" {
		params.Set("network", network)
	}
	var resp *WithdrawalFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/fees/withdrawal/"+c.Lower().String()+"/", nil, params, &resp)
}
