package gateio

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		pair := feeBuilder.Pair
		if pair.IsPopulated() {
			// Callers build fee requests in their own pair format, while Gate only recognises its underscored pairs
			formatted, err := e.FormatExchangeCurrency(pair, asset.Spot)
			if err != nil {
				return 0, err
			}
			pair = formatted
		}
		feePairs, err := e.GetPersonalTradingFee(ctx, pair, currency.EMPTYCODE)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = calculateTradingFee(feePairs.MakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		} else {
			fee = calculateTradingFee(feePairs.TakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		}
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(feeForPair, purchasePrice, amount float64) float64 {
	return feeForPair * purchasePrice * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}
