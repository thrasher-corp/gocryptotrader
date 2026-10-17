package bitstamp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// TransferToMainAccount transfers a balance from a sub account to the main account
// SubAccount must be supplied when called with main account credentials
func (e *Exchange) TransferToMainAccount(ctx context.Context, req *TransferRequest) error {
	params, err := transferParams(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/transfer-to-main/", nil, params, nil)
}

// TransferFromMainAccount transfers a balance from the main account to a sub account
func (e *Exchange) TransferFromMainAccount(ctx context.Context, req *TransferRequest) error {
	params, err := transferParams(req)
	if err != nil {
		return err
	}
	if req.SubAccount == 0 {
		return errSubAccountRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/transfer-from-main/", nil, params, nil)
}

func transferParams(req *TransferRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	setFloat(params, "amount", req.Amount)
	params.Set("currency", req.Currency.Upper().String())
	if req.SubAccount != 0 {
		params.Set("subAccount", strconv.FormatUint(req.SubAccount, 10))
	}
	return params, nil
}
