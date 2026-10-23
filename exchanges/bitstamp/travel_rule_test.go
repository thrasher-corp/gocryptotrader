package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetContact(t *testing.T) {
	t.Parallel()
	_, err := e.GetContact(t.Context(), "")
	assert.ErrorIs(t, err, errContactIDRequired, "GetContact should error without a contact ID")

	skipIfLiveWithoutCredentials(t)
	contact, err := e.GetContact(t.Context(), testContact.ID)
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GetContact should error for an unknown contact")
		return
	}
	require.NoError(t, err, "GetContact must not error")
	assert.Equal(t, &testContact, contact, "GetContact should return the contact")
}

func TestGetContacts(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	_, err := e.GetContacts(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetContacts should error on a nil request")
	contacts, err := e.GetContacts(t.Context(), &PaginationRequest{Page: 1, PerPage: 10})
	require.NoError(t, err, "GetContacts must not error")
	if mockTests {
		assert.Equal(t, []ContactResponse{testContact}, contacts, "GetContacts should return the contacts")
	}
}

func TestCreateContact(t *testing.T) {
	t.Parallel()
	_, err := e.CreateContact(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "CreateContact should error on a nil request")
	_, err = e.CreateContact(t.Context(), &ContactRequest{})
	assert.ErrorIs(t, err, errContactInfoRequired, "CreateContact should error without contact information")

	if !mockTests {
		t.Skip("CreateContact is not tested live to avoid creating contacts")
	}
	contact, err := e.CreateContact(t.Context(), &ContactRequest{RetailInfo: testContact.RetailInfo, Description: testContact.Description})
	require.NoError(t, err, "CreateContact must not error")
	assert.Equal(t, &testContact, contact, "CreateContact should return the contact")
}

func TestSubmitCounterpartyInfo(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *CounterpartyAddressRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &CounterpartyAddressRequest{}, err: common.ErrAddressIsEmptyOrInvalid},
		{req: &CounterpartyAddressRequest{Address: core.BitcoinDonationAddress}, err: errNetworkRequired},
		{req: &CounterpartyAddressRequest{Address: core.BitcoinDonationAddress, Network: "bitcoin", ContactThirdParty: true}, err: errContactUUIDRequired},
	} {
		_, err := e.SubmitCounterpartyInfo(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "SubmitCounterpartyInfo should return the correct error")
	}

	if !mockTests {
		t.Skip("SubmitCounterpartyInfo is not tested live to avoid registering addresses")
	}
	resp, err := e.SubmitCounterpartyInfo(t.Context(), &CounterpartyAddressRequest{
		Address:           core.BitcoinDonationAddress,
		Network:           "bitcoin",
		ContactThirdParty: true,
		ContactUUID:       testContact.ID,
		VASPUUID:          "48feffe5-a5d0-44fe-bf4c-734e7d721249",
	})
	require.NoError(t, err, "SubmitCounterpartyInfo must not error")
	exp := &CounterpartyAddressResponse{
		Address:           core.BitcoinDonationAddress,
		Network:           "bitcoin",
		ContactThirdParty: true,
		ContactUUID:       testContact.ID,
		VASPUUID:          "48feffe5-a5d0-44fe-bf4c-734e7d721249",
	}
	assert.Equal(t, exp, resp, "SubmitCounterpartyInfo should return the address")
}

func TestGetSatoshiTests(t *testing.T) {
	t.Parallel()
	_, err := e.GetSatoshiTests(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetSatoshiTests should error on a nil request")

	skipIfLiveWithoutCredentials(t)
	tests, err := e.GetSatoshiTests(t.Context(), &SatoshiTestsRequest{Network: "bitcoin", Status: "pending"})
	require.NoError(t, err, "GetSatoshiTests must not error")
	if mockTests {
		assert.Equal(t, []SatoshiTestResponse{testSatoshiTest}, tests, "GetSatoshiTests should return the tests")
	}
}

func TestCreateSatoshiTest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *SatoshiTestRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &SatoshiTestRequest{}, err: common.ErrAddressIsEmptyOrInvalid},
		{req: &SatoshiTestRequest{Address: core.BitcoinDonationAddress}, err: errNetworkRequired},
		{req: &SatoshiTestRequest{Address: core.BitcoinDonationAddress, Network: "bitcoin"}, err: currency.ErrCurrencyCodeEmpty},
	} {
		_, err := e.CreateSatoshiTest(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "CreateSatoshiTest should return the correct error")
	}

	if !mockTests {
		t.Skip("CreateSatoshiTest is not tested live to avoid creating tests")
	}
	resp, err := e.CreateSatoshiTest(t.Context(), &SatoshiTestRequest{Address: core.BitcoinDonationAddress, Network: "bitcoin", Currency: currency.BTC})
	require.NoError(t, err, "CreateSatoshiTest must not error")
	assert.Equal(t, &testSatoshiTest, resp, "CreateSatoshiTest should return the test")
}

func TestGetSatoshiTest(t *testing.T) {
	t.Parallel()
	_, err := e.GetSatoshiTest(t.Context(), "")
	assert.ErrorIs(t, err, errSatoshiTestIDRequired, "GetSatoshiTest should error without an ID")

	skipIfLiveWithoutCredentials(t)
	resp, err := e.GetSatoshiTest(t.Context(), testSatoshiTest.ID)
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GetSatoshiTest should error for an unknown test")
		return
	}
	require.NoError(t, err, "GetSatoshiTest must not error")
	assert.Equal(t, &testSatoshiTest, resp, "GetSatoshiTest should return the test")
}

func TestGetXpubRegistrations(t *testing.T) {
	t.Parallel()
	_, err := e.GetXpubRegistrations(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetXpubRegistrations should error on a nil request")
	_, err = e.GetXpubRegistrations(t.Context(), &XpubRegistrationsRequest{Limit: 1001})
	assert.ErrorIs(t, err, errInvalidLimit, "GetXpubRegistrations should error on an invalid limit")

	skipIfLiveWithoutCredentials(t)
	registrations, err := e.GetXpubRegistrations(t.Context(), &XpubRegistrationsRequest{Network: "bitcoin", Limit: 10})
	require.NoError(t, err, "GetXpubRegistrations must not error")
	if mockTests {
		require.Len(t, registrations, 1, "GetXpubRegistrations must return the registrations")
		exp := XpubRegistrationResponse{
			ID:        testXpubRegistrationID,
			Network:   "bitcoin",
			Label:     "btc cold wallet",
			Status:    "PENDING",
			Proof:     XpubRegistrationProof{ID: testSatoshiTest.ID, Method: "satoshi_test"},
			CreatedAt: DateTime(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)),
		}
		assert.Equal(t, exp, registrations[0], "GetXpubRegistrations should return the registration")
	}
}

func TestCreateXpubRegistration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *XpubRegistrationRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &XpubRegistrationRequest{}, err: errNetworkRequired},
		{req: &XpubRegistrationRequest{Network: "bitcoin"}, err: errPublicKeyRequired},
	} {
		_, err := e.CreateXpubRegistration(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "CreateXpubRegistration should return the correct error")
	}

	if !mockTests {
		t.Skip("CreateXpubRegistration is not tested live to avoid registering keys")
	}
	resp, err := e.CreateXpubRegistration(t.Context(), &XpubRegistrationRequest{
		Network:           "bitcoin",
		ExtendedPublicKey: "xpub6CUGRUonZSQ4TWtTMmzXdrXDtypWKiKrhko4egpiMZbpiaQL2jkwSB1icqYh2cfDfVxdx4df189oLKnC5fSwqPfgyP3hooxujYzAu3fDVmz",
		Label:             "btc cold wallet",
	})
	require.NoError(t, err, "CreateXpubRegistration must not error")
	exp := &XpubRegistrationResponse{
		ID:        testXpubRegistrationID,
		Network:   "bitcoin",
		Label:     "btc cold wallet",
		Status:    XpubRegistrationStatusPending,
		Proof:     XpubRegistrationProof{ID: testSatoshiTest.ID, Method: "satoshi_test"},
		CreatedAt: DateTime(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)),
	}
	assert.Equal(t, exp, resp, "CreateXpubRegistration should return the registration")
}

func TestRevokeXpubRegistration(t *testing.T) {
	t.Parallel()
	_, err := e.RevokeXpubRegistration(t.Context(), "")
	assert.ErrorIs(t, err, errRegistrationIDRequired, "RevokeXpubRegistration should error without an ID")

	if !mockTests {
		t.Skip("RevokeXpubRegistration is not tested live to avoid revoking registrations")
	}
	resp, err := e.RevokeXpubRegistration(t.Context(), testXpubRegistrationID)
	require.NoError(t, err, "RevokeXpubRegistration must not error")
	assert.Equal(t, &XpubRevocationResponse{ID: testXpubRegistrationID, Status: "REVOKED"}, resp, "RevokeXpubRegistration should return the revocation")
}

func TestGetXpubRegistration(t *testing.T) {
	t.Parallel()
	_, err := e.GetXpubRegistration(t.Context(), "")
	assert.ErrorIs(t, err, errRegistrationIDRequired, "GetXpubRegistration should error without an ID")

	skipIfLiveWithoutCredentials(t)
	resp, err := e.GetXpubRegistration(t.Context(), testXpubRegistrationID)
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GetXpubRegistration should error for an unknown registration")
		return
	}
	require.NoError(t, err, "GetXpubRegistration must not error")
	exp := &XpubRegistrationResponse{
		ID:          testXpubRegistrationID,
		Network:     "bitcoin",
		Label:       "btc cold wallet",
		Status:      XpubRegistrationStatusActive,
		Proof:       XpubRegistrationProof{ID: testSatoshiTest.ID, Method: "satoshi_test"},
		CreatedAt:   DateTime(time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)),
		ActivatedAt: DateTime(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)),
	}
	assert.Equal(t, exp, resp, "GetXpubRegistration should return the registration")
}

func TestGetAddressVerification(t *testing.T) {
	t.Parallel()
	_, err := e.GetAddressVerification(t.Context(), "", core.BitcoinDonationAddress)
	assert.ErrorIs(t, err, errNetworkRequired, "GetAddressVerification should error without a network")
	_, err = e.GetAddressVerification(t.Context(), "bitcoin", "")
	assert.ErrorIs(t, err, common.ErrAddressIsEmptyOrInvalid, "GetAddressVerification should error without an address")

	skipIfLiveWithoutCredentials(t)
	resp, err := e.GetAddressVerification(t.Context(), "bitcoin", core.BitcoinDonationAddress)
	require.NoError(t, err, "GetAddressVerification must not error")
	if mockTests {
		exp := &AddressVerificationResponse{
			Network:    "bitcoin",
			Address:    core.BitcoinDonationAddress,
			Verified:   true,
			Method:     "satoshi_test",
			VerifiedAt: DateTime(time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)),
		}
		assert.Equal(t, exp, resp, "GetAddressVerification should return the verification")
	}
}
