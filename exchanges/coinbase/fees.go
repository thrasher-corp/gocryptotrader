package coinbase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errInvalidTransactionSummaryProductType        = errors.New("invalid product type, want SPOT, FUTURE, EQUITY, OPTION_GROUP or FUTURE_GROUP")
	errInvalidTransactionSummaryContractExpiryType = errors.New("invalid contract expiry type, want EXPIRING or PERPETUAL")
	errInvalidTransactionSummaryProductVenue       = errors.New("invalid product venue, want CBE, FCM or INTX")
	errInvalidDecimalValue                         = errors.New("invalid decimal value")
)

// GetTransactionSummary returns the user's fee tier, with their trading volume and fees in USD, across every product
// type unless the request selects some; DeribitVenueData is only set for INTX derivatives
func (e *Exchange) GetTransactionSummary(ctx context.Context, req *TransactionSummaryRequest) (*TransactionSummaryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := req.params()
	if err != nil {
		return nil, err
	}
	var resp *TransactionSummaryResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"transaction_summary", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// params validates a transaction summary request and returns its query; Coinbase reads the UNKNOWN values its enums
// list as unset, so they are refused in favour of leaving a field empty
func (r *TransactionSummaryRequest) params() (url.Values, error) {
	switch r.ProductType {
	case "", ProductTypeSpot, ProductTypeFuture, ProductTypeEquity, ProductTypeOptionGroup, ProductTypeFutureGroup:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTransactionSummaryProductType, r.ProductType)
	}
	switch r.ContractExpiryType {
	case "", ContractExpiryTypeExpiring, ContractExpiryTypePerpetual:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTransactionSummaryContractExpiryType, r.ContractExpiryType)
	}
	switch r.ProductVenue {
	case "", ProductVenueCBE, ProductVenueFCM, ProductVenueINTX:
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTransactionSummaryProductVenue, r.ProductVenue)
	}
	params := url.Values{}
	if r.ProductType != "" {
		params.Set("product_type", r.ProductType)
	}
	if r.ContractExpiryType != "" {
		params.Set("contract_expiry_type", r.ContractExpiryType)
	}
	if r.ProductVenue != "" {
		params.Set("product_venue", r.ProductVenue)
	}
	return params, nil
}
