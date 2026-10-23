package bitstamp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetContact returns a Travel Rule contact
func (e *Exchange) GetContact(ctx context.Context, contactID string) (*ContactResponse, error) {
	if contactID == "" {
		return nil, errContactIDRequired
	}
	var resp *ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/contacts/"+url.PathEscape(contactID)+"/", nil, nil, &resp)
}

// GetContacts returns the account's Travel Rule contacts
func (e *Exchange) GetContacts(ctx context.Context, req *PaginationRequest) ([]ContactResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := paginationParams(req)
	var resp []ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/contacts/", params, nil, &resp)
}

// CreateContact creates a Travel Rule contact for use as a withdrawal beneficiary or deposit originator
func (e *Exchange) CreateContact(ctx context.Context, req *ContactRequest) (*ContactResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validateContact(req.RetailInfo, req.CorporateInfo); err != nil {
		return nil, err
	}
	var resp *ContactResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/contacts/", nil, req, &resp)
}

// SubmitCounterpartyInfo registers Travel Rule counterparty details for a cryptocurrency address
func (e *Exchange) SubmitCounterpartyInfo(ctx context.Context, req *CounterpartyAddressRequest) (*CounterpartyAddressResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.ContactThirdParty && req.ContactUUID == "" {
		return nil, errContactUUIDRequired
	}
	var resp *CounterpartyAddressResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/addresses/", nil, req, &resp)
}

// GetSatoshiTests returns the account's Satoshi tests for verifying ownership of external addresses
func (e *Exchange) GetSatoshiTests(ctx context.Context, req *SatoshiTestsRequest) ([]SatoshiTestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	for k, v := range map[string]string{"network": req.Network, "address": req.Address, "status": req.Status} {
		if v != "" {
			params.Set(k, v)
		}
	}
	var resp []SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/satoshi_test/", params, nil, &resp)
}

// CreateSatoshiTest starts a Satoshi test to verify ownership of an external address
func (e *Exchange) CreateSatoshiTest(ctx context.Context, req *SatoshiTestRequest) (*SatoshiTestResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	body := &satoshiTestBody{
		Address:  req.Address,
		Network:  req.Network,
		Currency: req.Currency.Upper().String(),
	}
	var resp *SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/satoshi_test/", nil, body, &resp)
}

// GetSatoshiTest returns a Satoshi test
func (e *Exchange) GetSatoshiTest(ctx context.Context, satoshiTestID string) (*SatoshiTestResponse, error) {
	if satoshiTestID == "" {
		return nil, errSatoshiTestIDRequired
	}
	var resp *SatoshiTestResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/satoshi_test/"+url.PathEscape(satoshiTestID)+"/", nil, nil, &resp)
}

// GetXpubRegistrations returns the account's extended public key registrations
func (e *Exchange) GetXpubRegistrations(ctx context.Context, req *XpubRegistrationsRequest) ([]XpubRegistrationResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := validatePaging(req.Limit, req.Offset); err != nil {
		return nil, err
	}
	params := url.Values{}
	if req.Network != "" {
		params.Set("network", req.Network)
	}
	if req.Status != "" {
		params.Set("status", req.Status)
	}
	if req.Offset != 0 {
		params.Set("offset", strconv.FormatUint(req.Offset, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/utxo/xpub_registrations/", params, nil, &resp)
}

// CreateXpubRegistration registers an extended public key for UTXO address derivation
// The registration remains pending until ownership is proven with a Satoshi test
func (e *Exchange) CreateXpubRegistration(ctx context.Context, req *XpubRegistrationRequest) (*XpubRegistrationResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Network == "" {
		return nil, errNetworkRequired
	}
	if req.ExtendedPublicKey == "" {
		return nil, errPublicKeyRequired
	}
	var resp *XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/utxo/xpub_registrations/", nil, req, &resp)
}

// RevokeXpubRegistration revokes an extended public key registration
func (e *Exchange) RevokeXpubRegistration(ctx context.Context, registrationID string) (*XpubRevocationResponse, error) {
	if registrationID == "" {
		return nil, errRegistrationIDRequired
	}
	var resp *XpubRevocationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v2/travel_rule/utxo/xpub_registrations/"+url.PathEscape(registrationID)+"/revoke/", nil, nil, &resp)
}

// GetXpubRegistration returns an extended public key registration
func (e *Exchange) GetXpubRegistration(ctx context.Context, registrationID string) (*XpubRegistrationResponse, error) {
	if registrationID == "" {
		return nil, errRegistrationIDRequired
	}
	var resp *XpubRegistrationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/utxo/xpub_registrations/"+url.PathEscape(registrationID)+"/", nil, nil, &resp)
}

// GetAddressVerification returns whether ownership of a cryptocurrency address has been verified for the account
func (e *Exchange) GetAddressVerification(ctx context.Context, network, address string) (*AddressVerificationResponse, error) {
	if network == "" {
		return nil, errNetworkRequired
	}
	if address == "" {
		return nil, common.ErrAddressIsEmptyOrInvalid
	}
	params := url.Values{}
	params.Set("network", network)
	params.Set("address", address)
	var resp *AddressVerificationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v2/travel_rule/address_verification/", params, nil, &resp)
}
