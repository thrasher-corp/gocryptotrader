package btcc

import (
	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	btccAuthRate   = 0
	btccUnauthRate = 0
)

// BTCC is the main overaching type across the BTCC package
// NOTE this package is websocket connection dependant, the REST endpoints have
// been dropped
type BTCC struct {
	exchange.Base
	Conn *websocket.Conn
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTCC) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency, feeBuilder.Amount)
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
	return 0.001 * price * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64) float64 {
	return WithdrawalFees[c] * amount
}
