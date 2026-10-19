package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// paymentMethodsResponse holds the payment methods ListPaymentMethods returns
type paymentMethodsResponse struct {
	PaymentMethods []PaymentMethod `json:"payment_methods"`
}

// paymentMethodResponse holds the payment method GetPaymentMethod returns
type paymentMethodResponse struct {
	PaymentMethod *PaymentMethod `json:"payment_method"`
}

// PaymentMethod is a way the user moves fiat into and out of Coinbase, and what it may be used for
type PaymentMethod struct {
	ID string `json:"id"`
	// Type is the payment method's kind, such as ACH
	Type string `json:"type"`
	// Name names the payment method as the user sees it, such as ALLY BANK ******1234
	Name          string        `json:"name"`
	Currency      currency.Code `json:"currency"`
	Verified      bool          `json:"verified"`
	AllowBuy      bool          `json:"allow_buy"`
	AllowSell     bool          `json:"allow_sell"`
	AllowDeposit  bool          `json:"allow_deposit"`
	AllowWithdraw bool          `json:"allow_withdraw"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
