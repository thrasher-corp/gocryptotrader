package coinbase

import (
	"context"
	"errors"
	"net/http"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var errAppAddressIDRequired = errors.New("address ID required")

// CreateAddress creates a receive address for a wallet account, on the network given or the account's currency's
// default network
func (a AppAPI) CreateAddress(ctx context.Context, req *AppCreateAddressRequest) (*AppAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	resp, err := sendAppRequest[*AppAddressResponse](ctx, a, http.MethodPost, "accounts/"+req.AccountID+"/addresses", nil, &AppCreateAddressWireRequest{Name: req.Name, Network: req.Network})
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// ListAddresses returns a page of an account's addresses
func (a AppAPI) ListAddresses(ctx context.Context, req *AppListAddressesRequest) (*AppListResponse[AppAddressResponse], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	params, err := req.Page.params()
	if err != nil {
		return nil, err
	}
	resp, err := sendAppRequest[[]AppAddressResponse](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/addresses", params, nil)
	if err != nil {
		return nil, err
	}
	return listResponse(resp), nil
}

// GetAddress returns one of an account's addresses
func (a AppAPI) GetAddress(ctx context.Context, req *AppGetAddressRequest) (*AppAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	if req.AddressID == "" {
		return nil, errAppAddressIDRequired
	}
	resp, err := sendAppRequest[*AppAddressResponse](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/addresses/"+req.AddressID, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// ListAddressTransactions returns a page of the transactions sent to one of an account's addresses
func (a AppAPI) ListAddressTransactions(ctx context.Context, req *AppListAddressTransactionsRequest) (*AppListResponse[AppTransactionResponse], error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.AccountID == "" {
		return nil, errAccountIDRequired
	}
	if req.AddressID == "" {
		return nil, errAppAddressIDRequired
	}
	params, err := req.Page.params()
	if err != nil {
		return nil, err
	}
	resp, err := sendAppRequest[[]AppTransactionResponse](ctx, a, http.MethodGet, "accounts/"+req.AccountID+"/addresses/"+req.AddressID+"/transactions", params, nil)
	if err != nil {
		return nil, err
	}
	return listResponse(resp), nil
}
