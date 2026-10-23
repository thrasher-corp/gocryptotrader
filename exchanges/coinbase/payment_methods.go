package coinbase

import (
	"context"
	"errors"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var errPaymentMethodIDRequired = errors.New("payment method ID required")

// GetPaymentMethod returns one of the user's payment methods by its ID, as ListPaymentMethods lists them
func (e *Exchange) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethod, error) {
	if paymentMethodID == "" {
		return nil, errPaymentMethodIDRequired
	}
	var resp *PaymentMethodResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"payment_methods/"+paymentMethodID, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.PaymentMethod == nil {
		return nil, common.ErrNoResponse
	}
	return resp.PaymentMethod, nil
}

// ListPaymentMethods returns the user's payment methods, such as bank accounts, which fiat moves into and out of
// Coinbase through
func (e *Exchange) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	var resp *PaymentMethodsResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"payment_methods", nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.PaymentMethods, nil
}
