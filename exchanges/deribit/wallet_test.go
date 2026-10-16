package deribit

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The mock wallet requests use the values of Deribit's documented examples, which the private method fixtures answer
const (
	// mockLegacyAddress is the BTC address of the legacy wallet type address book and beneficiary examples
	mockLegacyAddress = "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf0uyj"
	// mockWithdrawalAddress is the BTC address of the legacy withdrawal and deposit clearance examples
	mockWithdrawalAddress = "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz"
	// mockCoinbaseAddress, mockEthereumNetwork and mockAddressBookEntryID are the ETH address, network and address book
	// entry of the Coinbase wallet type examples
	mockCoinbaseAddress    = "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
	mockEthereumNetwork    = "networks/ethereum-mainnet"
	mockAddressBookEntryID = "0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5b"
	// mockCounterpartyID is the Coinbase Travel Network counterparty of the Coinbase wallet type examples
	mockCounterpartyID = "cp-8f3e2a1b9c4d"
	// mockBitcoinNetwork is the network of the withdrawal fee estimate example
	mockBitcoinNetwork = "networks/bitcoin-mainnet"
	// mockTransferID and mockWithdrawalID are the transfer and withdrawal the cancellation examples cancel
	mockTransferID   = 2
	mockWithdrawalID = 1
	// mockDestinationSubaccountID and mockSourceSubaccountID are the subaccounts of the subaccount transfer examples
	mockDestinationSubaccountID = 20
	mockSourceSubaccountID      = 10
	// mockUserTransferDestination is the address book address the example transfer to another user is sent to
	mockUserTransferDestination = "0x4aa0753d798d668056920094d65321a8e8913e26"
	// mockIdempotencyNonce is the nonce example, which the mock transfer between subaccounts and withdrawal send
	mockIdempotencyNonce = "bF1_gfgcsd"
	// mockDepositUserID and mockDepositTransactionHash identify the deposit the clearance example sets the originator
	// of
	mockDepositUserID          = 123
	mockDepositTransactionHash = "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda"
	// mockBeneficiariesCreatedAfter and mockBeneficiariesUpdatedBefore bound the mock beneficiary listing around the
	// example beneficiary, in Unix milliseconds
	mockBeneficiariesCreatedAfter  = 1536569522000
	mockBeneficiariesUpdatedBefore = 1536569523000
)

// testLegacyAddToAddressBookRequest is the documented legacy wallet type request adding a withdrawal address
func testLegacyAddToAddressBookRequest() *AddToAddressBookRequest {
	return &AddToAddressBookRequest{
		Currency:             currency.BTC,
		Type:                 AddressBookTypeWithdrawal,
		Address:              mockLegacyAddress,
		Label:                "Main address",
		BeneficiaryVASPName:  "Money`s Gone",
		BeneficiaryVASPDID:   "did:example:123456789abcdefghi",
		BeneficiaryFirstName: "John",
		BeneficiaryLastName:  "Doe",
		BeneficiaryAddress:   "NL, Amsterdam, Street, 1",
		Agreed:               new(true),
		Personal:             new(false),
	}
}

// testCoinbaseAddToAddressBookRequest is the documented Coinbase wallet type request adding an on-chain address
func testCoinbaseAddToAddressBookRequest() *AddToAddressBookRequest {
	return &AddToAddressBookRequest{
		Currency: currency.ETH,
		Type:     AddressBookTypeWithdrawal,
		Address:  mockCoinbaseAddress,
		Network:  mockEthereumNetwork,
		Label:    "Main address",
		Name:     "Alice Smith",
		Country:  "NL",
	}
}

// testCounterpartyAddToAddressBookRequest is the documented Coinbase wallet type request adding a Coinbase Travel
// Network counterparty, which leaves the currency out
func testCounterpartyAddToAddressBookRequest() *AddToAddressBookRequest {
	return &AddToAddressBookRequest{Type: AddressBookTypeWithdrawal, CounterpartyID: mockCounterpartyID, Label: "CTN Partner"}
}

// testCounterpartyAddressBookEntry is the documented Coinbase Travel Network counterparty entry, listed under every
// currency
func testCounterpartyAddressBookEntry() *AddressBookEntry {
	return &AddressBookEntry{
		ID:                  "0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5c",
		Currency:            currency.NewCode("all"),
		Address:             mockCounterpartyID,
		Type:                AddressBookTypeWithdrawal,
		Kind:                "ctn",
		Label:               "CTN Partner",
		CreationTimestamp:   types.Time(time.UnixMilli(1719993032041)),
		UpdatedTimestamp:    types.Time(time.UnixMilli(1719993032041)),
		Status:              "ready",
		RequiresSecurityKey: true,
		InfoRequired:        true,
	}
}

// testLegacyAddressBookEntry is the documented legacy wallet type address book entry the mock adds
func testLegacyAddressBookEntry() *AddressBookEntry {
	return &AddressBookEntry{
		Currency:             currency.BTC,
		Address:              mockLegacyAddress,
		Type:                 AddressBookTypeWithdrawal,
		Label:                "Main address",
		CreationTimestamp:    types.Time(time.UnixMilli(1536569522277)),
		BeneficiaryVASPName:  "Money`s Gone",
		BeneficiaryVASPDID:   "did:example:123456789abcdefghi",
		BeneficiaryFirstName: "John",
		BeneficiaryLastName:  "Doe",
		BeneficiaryAddress:   "NL, Amsterdam, Street, 1",
		Agreed:               true,
	}
}

// testCoinbaseAddressBookEntry is the documented Coinbase wallet type address book entry the mock adds; the fixture
// recorder blanks every name field, so the beneficiary's name is empty
func testCoinbaseAddressBookEntry() *AddressBookEntry {
	return &AddressBookEntry{
		ID:                          mockAddressBookEntryID,
		Currency:                    currency.ETH,
		Address:                     mockCoinbaseAddress,
		Type:                        AddressBookTypeWithdrawal,
		Kind:                        "onchain",
		Label:                       "Main address",
		CreationTimestamp:           types.Time(time.UnixMilli(1719993033041)),
		UpdatedTimestamp:            types.Time(time.UnixMilli(1719993033041)),
		Status:                      "waiting",
		WaitingTimestamp:            AddressWaitingTime(time.UnixMilli(1720252232860)),
		RequiresSecurityKey:         true,
		Network:                     mockEthereumNetwork,
		NetworkDisplayName:          "Ethereum",
		Country:                     "NL",
		BeneficiaryCreatedTimestamp: types.Time(time.UnixMilli(1719993033041)),
		BeneficiaryUpdatedTimestamp: types.Time(time.UnixMilli(1719993033041)),
	}
}

func TestAddToAddressBook(t *testing.T) {
	t.Parallel()
	_, err := e.AddToAddressBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddToAddressBook must reject a nil request")
	_, err = e.AddToAddressBook(t.Context(), &AddToAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeDeposit, Address: mockLegacyAddress, Label: "label"})
	require.ErrorIs(t, err, errInvalidAddressBookType, "AddToAddressBook must reject an address book type it cannot add")
	_, err = e.AddToAddressBook(t.Context(), &AddToAddressBookRequest{Type: AddressBookTypeWithdrawal, Address: mockLegacyAddress, Label: "label"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AddToAddressBook must reject an address without a currency")
	_, err = e.AddToAddressBook(t.Context(), &AddToAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal, Label: "label"})
	require.ErrorIs(t, err, errAddressOrCounterpartyRequired, "AddToAddressBook must reject a request without an address or counterparty")
	_, err = e.AddToAddressBook(t.Context(), &AddToAddressBookRequest{Type: AddressBookTypeWithdrawal, CounterpartyID: mockCounterpartyID})
	require.ErrorIs(t, err, errLabelRequired, "AddToAddressBook must reject an empty label")
	_, err = e.AddToAddressBook(t.Context(), &AddToAddressBookRequest{Type: AddressBookTypeWithdrawal, CounterpartyID: mockCounterpartyID, Label: "label", EntityType: "company"})
	require.ErrorIs(t, err, errInvalidBeneficiaryEntityType, "AddToAddressBook must reject an undocumented beneficiary entity type")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to add a withdrawal address, which live tests cannot give")
	}
	result, err := e.AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	require.NoError(t, err, "AddToAddressBook must not error")
	assert.Equal(t, testLegacyAddressBookEntry(), result, "AddToAddressBook should decode every field")

	result, err = e.AddToAddressBook(t.Context(), testCoinbaseAddToAddressBookRequest())
	require.NoError(t, err, "AddToAddressBook must not error for a Coinbase wallet type address")
	assert.Equal(t, testCoinbaseAddressBookEntry(), result, "AddToAddressBook should decode a Coinbase wallet type entry")

	// The fixture is recorded without a currency, so it only matches a request leaving the counterparty's currency out
	result, err = e.AddToAddressBook(t.Context(), testCounterpartyAddToAddressBookRequest())
	require.NoError(t, err, "AddToAddressBook must not error for a counterparty")
	assert.Equal(t, testCounterpartyAddressBookEntry(), result, "AddToAddressBook should decode a counterparty entry")

	// The fixture recorder blanks name fields, so the documented entry is served whole to show its name decodes
	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"id":"0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5b","currency":"ETH","address":"0x742d35Cc6634C0532925a3b844Bc454e4438f44e","type":"withdrawal","kind":"onchain","label":"Main address","creation_timestamp":1719993033041,"updated_timestamp":1719993033041,"status":"waiting","waiting_timestamp":1720252232860,"requires_confirmation":false,"requires_confirmation_change":false,"requires_security_key":true,"info_required":false,"network":"networks/ethereum-mainnet","network_display_name":"Ethereum","name":"Alice Smith","country":"NL","beneficiary_created":1719993033041,"beneficiary_updated":1719993033041}}`).AddToAddressBook(t.Context(), testCoinbaseAddToAddressBookRequest())
	require.NoError(t, err, "AddToAddressBook must not error for the documented Coinbase wallet type entry")
	exp := testCoinbaseAddressBookEntry()
	exp.Name = "Alice Smith"
	assert.Equal(t, exp, result, "AddToAddressBook should decode the beneficiary's name")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "AddToAddressBook should reject a null result")
	_, err = newUnavailableServerExchange(t).AddToAddressBook(t.Context(), testLegacyAddToAddressBookRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "AddToAddressBook should return the request failure")
}

// testTransfer is the documented transfer the mock transfer methods return
func testTransfer() *Transfer {
	return &Transfer{
		ID:               mockTransferID,
		Type:             "user",
		Currency:         currency.BTC,
		Amount:           0.2,
		OtherSide:        "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy",
		State:            "cancelled",
		Direction:        "payment",
		CreatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
		UpdatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
	}
}

func TestCancelTransferByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTransferByID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelTransferByID must reject a nil request")
	_, err = e.CancelTransferByID(t.Context(), &CancelTransferByIDRequest{ID: mockTransferID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelTransferByID must reject an empty currency")
	_, err = e.CancelTransferByID(t.Context(), &CancelTransferByIDRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errTransferIDRequired, "CancelTransferByID must reject an empty transfer ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("CancelTransferByID needs a pending transfer, which only a transfer to another user, which live tests cannot undo, leaves")
	}
	arg := &CancelTransferByIDRequest{Currency: currency.BTC, ID: mockTransferID}
	result, err := e.CancelTransferByID(t.Context(), arg)
	require.NoError(t, err, "CancelTransferByID must not error")
	assert.Equal(t, testTransfer(), result, "CancelTransferByID should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelTransferByID(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelTransferByID should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelTransferByID(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelTransferByID should return the request failure")
}

// testLegacyWithdrawal is the documented legacy wallet type withdrawal the mock cancels
func testLegacyWithdrawal() *Withdrawal {
	return &Withdrawal{
		ID:               "1",
		Currency:         currency.BTC,
		State:            "cancelled",
		Amount:           0.5,
		Address:          mockWithdrawalAddress,
		Fee:              0.0001,
		Priority:         0.15,
		CreatedTimestamp: types.Time(time.UnixMilli(1550571443070)),
		UpdatedTimestamp: types.Time(time.UnixMilli(1550571443070)),
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelWithdrawal must reject a nil request")
	_, err = e.CancelWithdrawal(t.Context(), &CancelWithdrawalRequest{ID: mockWithdrawalID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CancelWithdrawal must reject an empty currency")
	_, err = e.CancelWithdrawal(t.Context(), &CancelWithdrawalRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errWithdrawalIDRequired, "CancelWithdrawal must reject an empty withdrawal ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("CancelWithdrawal needs a pending withdrawal, which only a withdrawal, which live tests cannot undo, leaves")
	}
	arg := &CancelWithdrawalRequest{Currency: currency.BTC, ID: mockWithdrawalID}
	result, err := e.CancelWithdrawal(t.Context(), arg)
	require.NoError(t, err, "CancelWithdrawal must not error")
	assert.Equal(t, testLegacyWithdrawal(), result, "CancelWithdrawal should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CancelWithdrawal(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelWithdrawal should reject a null result")
	_, err = newUnavailableServerExchange(t).CancelWithdrawal(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelWithdrawal should return the request failure")
}

// testWithdrawalFeeEstimate is the documented withdrawal fee estimate the mock returns
func testWithdrawalFeeEstimate(t *testing.T) *WithdrawalFeeEstimateResponse {
	t.Helper()
	expiry, err := time.Parse(time.RFC3339, "2026-04-06T12:00:00Z")
	require.NoError(t, err, "Parse must not error")
	return &WithdrawalFeeEstimateResponse{EstimatedFee: 0.0005, EstimatedFeeBeforeSubsidy: 0.001, ExpiresAt: expiry}
}

func TestCoinbaseEstimateWithdrawalFee(t *testing.T) {
	t.Parallel()
	_, err := e.CoinbaseEstimateWithdrawalFee(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinbaseEstimateWithdrawalFee must reject a nil request")
	_, err = e.CoinbaseEstimateWithdrawalFee(t.Context(), &CoinbaseEstimateWithdrawalFeeRequest{NetworkName: mockBitcoinNetwork})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CoinbaseEstimateWithdrawalFee must reject an empty currency")
	_, err = e.CoinbaseEstimateWithdrawalFee(t.Context(), &CoinbaseEstimateWithdrawalFeeRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errNetworkNameRequired, "CoinbaseEstimateWithdrawalFee must reject an empty network name")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		t.Skip("CoinbaseEstimateWithdrawalFee needs an institutional Coinbase wallet type account")
	}
	arg := &CoinbaseEstimateWithdrawalFeeRequest{Currency: currency.BTC, NetworkName: mockBitcoinNetwork}
	result, err := e.CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	require.NoError(t, err, "CoinbaseEstimateWithdrawalFee must not error")
	assert.Equal(t, testWithdrawalFeeEstimate(t), result, "CoinbaseEstimateWithdrawalFee should decode every field")

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"estimated_fee":"0.0005","estimated_fee_before_subsidy":"0.0005","expires_at":null}}`).CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	require.NoError(t, err, "CoinbaseEstimateWithdrawalFee must not error without an expiry")
	assert.Equal(t, &WithdrawalFeeEstimateResponse{EstimatedFee: 0.0005, EstimatedFeeBeforeSubsidy: 0.0005}, result, "CoinbaseEstimateWithdrawalFee should decode a null expiry as the zero time")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CoinbaseEstimateWithdrawalFee should reject a null result")
	_, err = newUnavailableServerExchange(t).CoinbaseEstimateWithdrawalFee(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CoinbaseEstimateWithdrawalFee should return the request failure")
}

// testLegacyDepositAddress is the documented legacy wallet type deposit address the mock creates and returns
func testLegacyDepositAddress() *AddressBookEntry {
	return &AddressBookEntry{
		Currency:          currency.BTC,
		Address:           "2N8udZGBc1hLRCFsU9kGwMPpmYUwMFTuCwB",
		Type:              AddressBookTypeDeposit,
		CreationTimestamp: types.Time(time.UnixMilli(1550575165170)),
	}
}

// testCoinbaseDepositAddress is the documented Coinbase wallet type deposit address the mock creates and returns
func testCoinbaseDepositAddress() *AddressBookEntry {
	return &AddressBookEntry{
		ID:                 mockAddressBookEntryID,
		Currency:           currency.ETH,
		Address:            mockCoinbaseAddress,
		Type:               AddressBookTypeDeposit,
		Kind:               "onchain",
		Label:              "deposit address",
		CreationTimestamp:  types.Time(time.UnixMilli(1719993033041)),
		UpdatedTimestamp:   types.Time(time.UnixMilli(1719993033041)),
		Status:             "active",
		InfoRequired:       true,
		Network:            mockEthereumNetwork,
		NetworkDisplayName: "Ethereum",
	}
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.CreateDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateDepositAddress must reject a nil request")
	_, err = e.CreateDepositAddress(t.Context(), &CreateDepositAddressRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateDepositAddress must reject an empty currency")

	skipLiveUnlessCanManipulate(t)
	arg := &CreateDepositAddressRequest{Currency: currency.BTC}
	result, err := e.CreateDepositAddress(t.Context(), arg)
	if !mockTests && errors.Is(err, common.ErrNoResponse) {
		t.Skip("Deribit creates at most one BTC deposit address a day")
	}
	require.NoError(t, err, "CreateDepositAddress must not error")
	if mockTests {
		assert.Equal(t, testLegacyDepositAddress(), result, "CreateDepositAddress should decode every field")
	} else {
		assert.Equal(t, currency.BTC, result.Currency, "CreateDepositAddress should return a BTC address")
		assert.NotEmpty(t, result.Address, "CreateDepositAddress should return the address")
		return
	}

	result, err = e.CreateDepositAddress(t.Context(), &CreateDepositAddressRequest{Currency: currency.ETH, Network: mockEthereumNetwork})
	require.NoError(t, err, "CreateDepositAddress must not error for a Coinbase wallet type network")
	assert.Equal(t, testCoinbaseDepositAddress(), result, "CreateDepositAddress should decode a Coinbase wallet type deposit address")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CreateDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateDepositAddress should report that no address was created")
	_, err = newUnavailableServerExchange(t).CreateDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateDepositAddress should return the request failure")
}

func TestDeleteAddressBeneficiary(t *testing.T) {
	t.Parallel()
	err := e.DeleteAddressBeneficiary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DeleteAddressBeneficiary must reject a nil request")
	err = e.DeleteAddressBeneficiary(t.Context(), &AddressBeneficiaryRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errAddressRequired, "DeleteAddressBeneficiary must reject an empty address")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("DeleteAddressBeneficiary would delete an address's travel rule information, which a live test cannot restore")
	}
	arg := &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}
	require.NoError(t, e.DeleteAddressBeneficiary(t.Context(), arg), "DeleteAddressBeneficiary must not error")

	err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"failed"}`).DeleteAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, errUnexpectedResult, "DeleteAddressBeneficiary should reject a result other than ok")
	err = newUnavailableServerExchange(t).DeleteAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DeleteAddressBeneficiary should return the request failure")
}

// testAddressBeneficiary is the documented address beneficiary the mock saves and returns
func testAddressBeneficiary() *AddressBeneficiary {
	return &AddressBeneficiary{
		Currency:               currency.BTC,
		Address:                mockLegacyAddress,
		UserID:                 1016,
		Agreed:                 true,
		BeneficiaryVASPName:    "Money's Gone",
		BeneficiaryVASPDID:     "did:example:123456789abcdefghi",
		BeneficiaryVASPWebsite: "https://example.com",
		BeneficiaryFirstName:   "John",
		BeneficiaryLastName:    "Doe",
		BeneficiaryCompanyName: "Example Corp",
		BeneficiaryAddress:     "NL, Amsterdam, Street, 1",
		CreatedTimestamp:       types.Time(time.UnixMilli(1536569522277)),
		UpdatedTimestamp:       types.Time(time.UnixMilli(1536569522277)),
	}
}

// testAddressBeneficiaryRequest selects the mock beneficiary or, live, the account's first BTC address beneficiary,
// skipping the test when it has none
func testAddressBeneficiaryRequest(t *testing.T) *AddressBeneficiaryRequest {
	t.Helper()
	if mockTests {
		return &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}
	}
	result, err := e.ListAddressBeneficiaries(t.Context(), &ListAddressBeneficiariesRequest{Currency: currency.BTC, Limit: 1})
	require.NoError(t, err, "ListAddressBeneficiaries must not error")
	if len(result.Beneficiaries) == 0 {
		t.Skip("the account has no BTC address beneficiary to read")
	}
	return &AddressBeneficiaryRequest{Currency: currency.BTC, Address: result.Beneficiaries[0].Address, Tag: result.Beneficiaries[0].Tag}
}

func TestGetAddressBeneficiary(t *testing.T) {
	t.Parallel()
	_, err := e.GetAddressBeneficiary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAddressBeneficiary must reject a nil request")
	_, err = e.GetAddressBeneficiary(t.Context(), &AddressBeneficiaryRequest{Currency: currency.BTC})
	require.ErrorIs(t, err, errAddressRequired, "GetAddressBeneficiary must reject an empty address")

	skipLiveWithoutCredentials(t)
	arg := testAddressBeneficiaryRequest(t)
	result, err := e.GetAddressBeneficiary(t.Context(), arg)
	require.NoError(t, err, "GetAddressBeneficiary must not error")
	if mockTests {
		assert.Equal(t, testAddressBeneficiary(), result, "GetAddressBeneficiary should decode every field")
	} else {
		assert.Equal(t, arg.Address, result.Address, "GetAddressBeneficiary should return the address's beneficiary")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAddressBeneficiary should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAddressBeneficiary(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAddressBeneficiary should return the request failure")
}

func TestAddressBeneficiaryRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AddressBeneficiaryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty currency", arg: &AddressBeneficiaryRequest{Address: mockLegacyAddress}, err: currency.ErrCurrencyCodeEmpty},
		{name: "empty address", arg: &AddressBeneficiaryRequest{Currency: currency.BTC}, err: errAddressRequired},
		{name: "address", arg: &AddressBeneficiaryRequest{Currency: currency.BTC, Address: mockLegacyAddress}},
		{name: "tagged address", arg: &AddressBeneficiaryRequest{Currency: currency.XRP, Address: "rDsbeomae4FXwgQTJp9Rs64Qg9vDiTCdBv", Tag: "12345"}},
	} {
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

// testLegacyAddressBook is the documented legacy wallet type address book the mock returns
func testLegacyAddressBook() []AddressBookEntry {
	return []AddressBookEntry{
		{
			Currency:             currency.BTC,
			Address:              mockWithdrawalAddress,
			Type:                 AddressBookTypeWithdrawal,
			Label:                "Main Address",
			CreationTimestamp:    types.Time(time.UnixMilli(1719993033041)),
			Status:               "waiting",
			WaitingTimestamp:     AddressWaitingTime(time.UnixMilli(1720252232860)),
			RequiresConfirmation: true,
			BeneficiaryFirstName: "John",
			BeneficiaryLastName:  "Doe",
			BeneficiaryAddress:   "NL, Amsterdam, Street, 1",
			Agreed:               true,
			Personal:             true,
		},
		{
			Currency:               currency.BTC,
			Address:                mockLegacyAddress,
			Type:                   AddressBookTypeWithdrawal,
			Label:                  "One More Address",
			CreationTimestamp:      types.Time(time.UnixMilli(1719993032041)),
			Status:                 "waiting",
			WaitingTimestamp:       AddressWaitingTime(time.UnixMilli(1720252232760)),
			BeneficiaryCompanyName: "MyCompany",
			BeneficiaryAddress:     "NL, Haarlem, Street, 5",
			Agreed:                 true,
			Personal:               true,
		},
	}
}

func TestGetAddressBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetAddressBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAddressBook must reject a nil request")
	_, err = e.GetAddressBook(t.Context(), &GetAddressBookRequest{Type: AddressBookTypeWithdrawal})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAddressBook must reject an empty currency")
	_, err = e.GetAddressBook(t.Context(), &GetAddressBookRequest{Currency: currency.BTC, Type: "deposits"})
	require.ErrorIs(t, err, errInvalidAddressBookType, "GetAddressBook must reject an undocumented address book type")

	skipLiveWithoutCredentials(t)
	arg := &GetAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal}
	result, err := e.GetAddressBook(t.Context(), arg)
	require.NoError(t, err, "GetAddressBook must not error")
	if !mockTests {
		for i := range result {
			assert.Equal(t, AddressBookTypeWithdrawal, result[i].Type, "GetAddressBook should return withdrawal entries")
		}
		return
	}
	assert.Equal(t, testLegacyAddressBook(), result, "GetAddressBook should decode every field")

	result, err = e.GetAddressBook(t.Context(), &GetAddressBookRequest{Currency: currency.ETH, Type: AddressBookTypeWithdrawal})
	require.NoError(t, err, "GetAddressBook must not error for a Coinbase wallet type account")
	assert.Equal(t, []AddressBookEntry{*testCoinbaseAddressBookEntry(), *testCounterpartyAddressBookEntry()}, result, "GetAddressBook should decode Coinbase wallet type entries and counterparties")

	_, err = newUnavailableServerExchange(t).GetAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAddressBook should return the request failure")
}

func TestGetCurrentDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCurrentDepositAddress must reject a nil request")
	_, err = e.GetCurrentDepositAddress(t.Context(), &GetCurrentDepositAddressRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrentDepositAddress must reject an empty currency")

	skipLiveWithoutCredentials(t)
	arg := &GetCurrentDepositAddressRequest{Currency: currency.BTC}
	result, err := e.GetCurrentDepositAddress(t.Context(), arg)
	if !mockTests && errors.Is(err, common.ErrNoResponse) {
		t.Skip("the account has no BTC deposit address")
	}
	require.NoError(t, err, "GetCurrentDepositAddress must not error")
	if !mockTests {
		assert.Equal(t, currency.BTC, result.Currency, "GetCurrentDepositAddress should return a BTC address")
		return
	}
	assert.Equal(t, testLegacyDepositAddress(), result, "GetCurrentDepositAddress should decode every field")

	result, err = e.GetCurrentDepositAddress(t.Context(), &GetCurrentDepositAddressRequest{Currency: currency.ETH, Network: mockEthereumNetwork})
	require.NoError(t, err, "GetCurrentDepositAddress must not error for a Coinbase wallet type network")
	assert.Equal(t, testCoinbaseDepositAddress(), result, "GetCurrentDepositAddress should decode a Coinbase wallet type deposit address")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetCurrentDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCurrentDepositAddress should report that no address exists")
	_, err = newUnavailableServerExchange(t).GetCurrentDepositAddress(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCurrentDepositAddress should return the request failure")
}

// testWalletHistoryRequest requests the documented page of BTC wallet history
func testWalletHistoryRequest() *WalletHistoryRequest {
	return &WalletHistoryRequest{Currency: currency.BTC, Count: 10}
}

// testDeposits is the documented page of deposits the mock returns
func testDeposits() *DepositsResponse {
	return &DepositsResponse{
		Deposits: []Deposit{{
			Currency:          currency.BTC,
			Address:           "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax",
			Amount:            5,
			State:             "completed",
			TransactionID:     mockDepositTransactionHash,
			ReceivedTimestamp: types.Time(time.UnixMilli(1549295017670)),
			UpdatedTimestamp:  types.Time(time.UnixMilli(1549295130159)),
		}},
		Count: 1,
	}
}

func TestGetDeposits(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeposits(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeposits must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetDeposits(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "GetDeposits must not error")
	if mockTests {
		assert.Equal(t, testDeposits(), result, "GetDeposits should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Deposits), 10, "GetDeposits should return at most the requested count")
		for i := range result.Deposits {
			assert.Equal(t, currency.BTC, result.Deposits[i].Currency, "GetDeposits should return BTC deposits")
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetDeposits(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDeposits should reject a null result")
	_, err = newUnavailableServerExchange(t).GetDeposits(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDeposits should return the request failure")
}

func TestWalletHistoryRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *WalletHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty currency", arg: &WalletHistoryRequest{Count: 10}, err: currency.ErrCurrencyCodeEmpty},
		{name: "count over 1000", arg: &WalletHistoryRequest{Currency: currency.BTC, Count: 1001}, err: errInvalidCount},
		{name: "default count", arg: &WalletHistoryRequest{Currency: currency.BTC}},
		{name: "maximum count", arg: &WalletHistoryRequest{Currency: currency.BTC, Count: 1000, Offset: 10}},
	} {
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

// testRewardEligibility is the documented reward eligibility the mock returns
func testRewardEligibility() RewardEligibilityResponse {
	return RewardEligibilityResponse{
		currency.USDC:  {EligibilityStatus: "non_eligible", APRSimpleMovingAverage7Day: 4},
		currency.USDE:  {EligibilityStatus: "eligible", APRSimpleMovingAverage7Day: 7},
		currency.BUIDL: {EligibilityStatus: "eligible", APRSimpleMovingAverage7Day: 3.943606},
		currency.STETH: {EligibilityStatus: "eligible", APRSimpleMovingAverage7Day: 2.692285714285714},
	}
}

func TestGetRewardEligibility(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetRewardEligibility(t.Context())
	require.NoError(t, err, "GetRewardEligibility must not error")
	if mockTests {
		assert.Equal(t, testRewardEligibility(), result, "GetRewardEligibility should decode every currency's eligibility")
	} else {
		assert.NotEmpty(t, result, "GetRewardEligibility should return the reward bearing currencies")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetRewardEligibility(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetRewardEligibility should reject a null result")
	_, err = newUnavailableServerExchange(t).GetRewardEligibility(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRewardEligibility should return the request failure")
}

// testTransfers is the documented page of transfers the mock returns
func testTransfers() *TransfersResponse {
	return &TransfersResponse{
		Transfers: []Transfer{
			{
				ID:               2,
				Type:             "user",
				Currency:         currency.BTC,
				Amount:           0.2,
				OtherSide:        "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy",
				State:            "prepared",
				Direction:        "payment",
				CreatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
				UpdatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
			},
			{
				ID:               1,
				Type:             "subaccount",
				Currency:         currency.BTC,
				Amount:           0.3,
				OtherSide:        "new_user_1_1",
				State:            "confirmed",
				Direction:        "payment",
				CreatedTimestamp: types.Time(time.UnixMilli(1550579255800)),
				UpdatedTimestamp: types.Time(time.UnixMilli(1550579255800)),
			},
		},
		Count: 2,
	}
}

func TestGetTransfers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransfers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransfers must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetTransfers(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "GetTransfers must not error")
	if mockTests {
		assert.Equal(t, testTransfers(), result, "GetTransfers should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Transfers), 10, "GetTransfers should return at most the requested count")
		for i := range result.Transfers {
			assert.Equal(t, currency.BTC, result.Transfers[i].Currency, "GetTransfers should return BTC transfers")
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTransfers(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransfers should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTransfers(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransfers should return the request failure")
}

// testWithdrawals is the documented page of withdrawals the mock returns
func testWithdrawals() *WithdrawalsResponse {
	withdrawal := testLegacyWithdrawal()
	withdrawal.State = "unconfirmed"
	return &WithdrawalsResponse{Withdrawals: []Withdrawal{*withdrawal}, Count: 1}
}

func TestGetWithdrawals(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawals(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawals must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetWithdrawals(t.Context(), testWalletHistoryRequest())
	require.NoError(t, err, "GetWithdrawals must not error")
	if mockTests {
		assert.Equal(t, testWithdrawals(), result, "GetWithdrawals should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Withdrawals), 10, "GetWithdrawals should return at most the requested count")
		for i := range result.Withdrawals {
			assert.Equal(t, currency.BTC, result.Withdrawals[i].Currency, "GetWithdrawals should return BTC withdrawals")
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetWithdrawals(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetWithdrawals should reject a null result")
	_, err = newUnavailableServerExchange(t).GetWithdrawals(t.Context(), testWalletHistoryRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetWithdrawals should return the request failure")
}

// mockListAddressBeneficiariesRequest filters the mock beneficiaries by currency, creation and update times and a flag
func mockListAddressBeneficiariesRequest() *ListAddressBeneficiariesRequest {
	return &ListAddressBeneficiariesRequest{
		Currency:      currency.BTC,
		CreatedAfter:  time.UnixMilli(mockBeneficiariesCreatedAfter),
		UpdatedBefore: time.UnixMilli(mockBeneficiariesUpdatedBefore),
		Unhosted:      new(false),
		Limit:         10,
	}
}

// testListAddressBeneficiariesRequest filters the mock beneficiaries or, live, lists the account's BTC beneficiaries
func testListAddressBeneficiariesRequest() *ListAddressBeneficiariesRequest {
	if mockTests {
		return mockListAddressBeneficiariesRequest()
	}
	return &ListAddressBeneficiariesRequest{Currency: currency.BTC, Limit: 10}
}

// testAddressBeneficiaries is the documented page of address beneficiaries the mock returns
func testAddressBeneficiaries() *AddressBeneficiariesResponse {
	beneficiary := testAddressBeneficiary()
	beneficiary.UserID = 1026
	return &AddressBeneficiariesResponse{
		Beneficiaries: []AddressBeneficiary{*beneficiary},
		Continuation:  "xY7T6cutS3t2B9YtaDkE6TS379oKnkzTvmEDUnEUP2Msa9xKWNNaT",
		Count:         1,
	}
}

func TestListAddressBeneficiaries(t *testing.T) {
	t.Parallel()
	_, err := e.ListAddressBeneficiaries(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ListAddressBeneficiaries must reject a nil request")
	_, err = e.ListAddressBeneficiaries(t.Context(), &ListAddressBeneficiariesRequest{Limit: 1001})
	require.ErrorIs(t, err, errInvalidAddressBeneficiariesLimit, "ListAddressBeneficiaries must reject a limit over 1000")
	now := time.Now()
	_, err = e.ListAddressBeneficiaries(t.Context(), &ListAddressBeneficiariesRequest{CreatedAfter: now, CreatedBefore: now.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "ListAddressBeneficiaries must reject a creation range ending before it starts")
	_, err = e.ListAddressBeneficiaries(t.Context(), &ListAddressBeneficiariesRequest{UpdatedAfter: now, UpdatedBefore: now.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "ListAddressBeneficiaries must reject an update range ending before it starts")

	skipLiveWithoutCredentials(t)
	result, err := e.ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	require.NoError(t, err, "ListAddressBeneficiaries must not error")
	if mockTests {
		assert.Equal(t, testAddressBeneficiaries(), result, "ListAddressBeneficiaries should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Beneficiaries), 10, "ListAddressBeneficiaries should return at most the requested limit")
		for i := range result.Beneficiaries {
			assert.Equal(t, currency.BTC, result.Beneficiaries[i].Currency, "ListAddressBeneficiaries should return BTC beneficiaries")
		}
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "ListAddressBeneficiaries should reject a null result")
	_, err = newUnavailableServerExchange(t).ListAddressBeneficiaries(t.Context(), testListAddressBeneficiariesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAddressBeneficiaries should return the request failure")
}

func TestListAddressBeneficiariesParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	arg := &ListAddressBeneficiariesRequest{
		Currency:      currency.BTC,
		CreatedBefore: time.UnixMilli(1700000003000),
		CreatedAfter:  time.UnixMilli(1700000000000),
		UpdatedBefore: time.UnixMilli(1700000004000),
		UpdatedAfter:  time.UnixMilli(1700000001000),
		Personal:      new(false),
	}
	encoded, err := json.Marshal(&listAddressBeneficiariesParams{
		ListAddressBeneficiariesRequest: arg,
		CreatedBefore:                   arg.CreatedBefore.UnixMilli(),
		CreatedAfter:                    arg.CreatedAfter.UnixMilli(),
		UpdatedBefore:                   arg.UpdatedBefore.UnixMilli(),
		UpdatedAfter:                    arg.UpdatedAfter.UnixMilli(),
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"currency":"BTC","created_before":1700000003000,"created_after":1700000000000,"updated_before":1700000004000,"updated_after":1700000001000,"personal":false}`, string(encoded), "listAddressBeneficiariesParams should send each time once in Unix milliseconds and a false filter")

	encoded, err = json.Marshal(&listAddressBeneficiariesParams{ListAddressBeneficiariesRequest: &ListAddressBeneficiariesRequest{}})
	require.NoError(t, err, "Marshal must not error for an empty request")
	assert.JSONEq(t, `{}`, string(encoded), "listAddressBeneficiariesParams should leave out every unset filter")
}

func TestRemoveFromAddressBook(t *testing.T) {
	t.Parallel()
	err := e.RemoveFromAddressBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RemoveFromAddressBook must reject a nil request")
	err = e.RemoveFromAddressBook(t.Context(), &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeDepositSource, Address: mockLegacyAddress})
	require.ErrorIs(t, err, errInvalidAddressBookType, "RemoveFromAddressBook must reject an address book type it cannot remove")
	err = e.RemoveFromAddressBook(t.Context(), &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeTransfer})
	require.ErrorIs(t, err, errAddressOrEntryIDRequired, "RemoveFromAddressBook must reject a request without an address or entry ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("RemoveFromAddressBook would remove an entry, which adding again restarts the address delay of")
	}
	arg := &RemoveFromAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeTransfer, Address: mockLegacyAddress}
	require.NoError(t, e.RemoveFromAddressBook(t.Context(), arg), "RemoveFromAddressBook must not error")

	err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"failed"}`).RemoveFromAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, errUnexpectedResult, "RemoveFromAddressBook should reject a result other than ok")
	err = newUnavailableServerExchange(t).RemoveFromAddressBook(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RemoveFromAddressBook should return the request failure")
}

// testSaveAddressBeneficiaryRequest is the documented request saving an address's beneficiary information
func testSaveAddressBeneficiaryRequest() *SaveAddressBeneficiaryRequest {
	return &SaveAddressBeneficiaryRequest{
		Currency:               currency.BTC,
		Address:                mockLegacyAddress,
		Agreed:                 true,
		BeneficiaryVASPName:    "Money's Gone",
		BeneficiaryVASPDID:     "did:example:123456789abcdefghi",
		BeneficiaryVASPWebsite: "https://example.com",
		BeneficiaryFirstName:   "John",
		BeneficiaryLastName:    "Doe",
		BeneficiaryCompanyName: "Example Corp",
		BeneficiaryAddress:     "NL, Amsterdam, Street, 1",
	}
}

func TestSaveAddressBeneficiary(t *testing.T) {
	t.Parallel()
	_, err := e.SaveAddressBeneficiary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SaveAddressBeneficiary must reject a nil request")
	arg := testSaveAddressBeneficiaryRequest()
	arg.Currency = currency.EMPTYCODE
	_, err = e.SaveAddressBeneficiary(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SaveAddressBeneficiary must reject an empty currency")
	arg = testSaveAddressBeneficiaryRequest()
	arg.Address = ""
	_, err = e.SaveAddressBeneficiary(t.Context(), arg)
	require.ErrorIs(t, err, errAddressRequired, "SaveAddressBeneficiary must reject an empty address")
	arg = testSaveAddressBeneficiaryRequest()
	arg.BeneficiaryVASPName = ""
	_, err = e.SaveAddressBeneficiary(t.Context(), arg)
	require.ErrorIs(t, err, errBeneficiaryVASPNameRequired, "SaveAddressBeneficiary must reject an empty VASP name")
	arg = testSaveAddressBeneficiaryRequest()
	arg.BeneficiaryVASPDID = ""
	_, err = e.SaveAddressBeneficiary(t.Context(), arg)
	require.ErrorIs(t, err, errBeneficiaryVASPDIDRequired, "SaveAddressBeneficiary must reject an empty VASP DID")
	arg = testSaveAddressBeneficiaryRequest()
	arg.BeneficiaryAddress = ""
	_, err = e.SaveAddressBeneficiary(t.Context(), arg)
	require.ErrorIs(t, err, errBeneficiaryAddressRequired, "SaveAddressBeneficiary must reject an empty beneficiary address")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SaveAddressBeneficiary would overwrite an address's travel rule information, which a live test cannot restore")
	}
	result, err := e.SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	require.NoError(t, err, "SaveAddressBeneficiary must not error")
	assert.Equal(t, testAddressBeneficiary(), result, "SaveAddressBeneficiary should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SaveAddressBeneficiary should reject a null result")
	_, err = newUnavailableServerExchange(t).SaveAddressBeneficiary(t.Context(), testSaveAddressBeneficiaryRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SaveAddressBeneficiary should return the request failure")
}

// testLegacySecurityKeyRequest is the documented legacy wallet type request making withdrawals to an address need a
// security key
func testLegacySecurityKeyRequest() *SetAddressRequiresSecurityKeyRequest {
	return &SetAddressRequiresSecurityKeyRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal, Address: mockLegacyAddress, RequiresSecurityKey: true}
}

// testLegacySecurityKeyEntry is the documented legacy wallet type entry the mock sets the security key requirement of
func testLegacySecurityKeyEntry() *AddressBookEntry {
	return &AddressBookEntry{
		Currency:             currency.BTC,
		Address:              mockLegacyAddress,
		Type:                 AddressBookTypeWithdrawal,
		Label:                "Main address",
		CreationTimestamp:    types.Time(time.UnixMilli(1536569522277)),
		Status:               "ready",
		RequiresConfirmation: true,
		RequiresSecurityKey:  true,
	}
}

func TestSetAddressRequiresSecurityKey(t *testing.T) {
	t.Parallel()
	_, err := e.SetAddressRequiresSecurityKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetAddressRequiresSecurityKey must reject a nil request")
	_, err = e.SetAddressRequiresSecurityKey(t.Context(), &SetAddressRequiresSecurityKeyRequest{Type: AddressBookTypeWithdrawal, Address: mockLegacyAddress})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetAddressRequiresSecurityKey must reject an empty currency")
	_, err = e.SetAddressRequiresSecurityKey(t.Context(), &SetAddressRequiresSecurityKeyRequest{Currency: currency.BTC, Type: AddressBookTypeDepositSource, Address: mockLegacyAddress})
	require.ErrorIs(t, err, errInvalidAddressBookType, "SetAddressRequiresSecurityKey must reject an address book type it cannot set")
	_, err = e.SetAddressRequiresSecurityKey(t.Context(), &SetAddressRequiresSecurityKeyRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal})
	require.ErrorIs(t, err, errAddressOrEntryIDRequired, "SetAddressRequiresSecurityKey must reject a request without an address or entry ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to change a withdrawal address, which live tests cannot give")
	}
	result, err := e.SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	require.NoError(t, err, "SetAddressRequiresSecurityKey must not error")
	assert.Equal(t, testLegacySecurityKeyEntry(), result, "SetAddressRequiresSecurityKey should decode every field")

	result, err = e.SetAddressRequiresSecurityKey(t.Context(), &SetAddressRequiresSecurityKeyRequest{Currency: currency.ETH, Type: AddressBookTypeWithdrawal, EntryID: mockAddressBookEntryID, RequiresSecurityKey: true})
	require.NoError(t, err, "SetAddressRequiresSecurityKey must not error for a Coinbase wallet type entry")
	exp := &AddressBookEntry{
		ID:                  mockAddressBookEntryID,
		Currency:            currency.ETH,
		Address:             mockCoinbaseAddress,
		Type:                AddressBookTypeWithdrawal,
		Kind:                "onchain",
		Label:               "Main address",
		CreationTimestamp:   types.Time(time.UnixMilli(1719993033041)),
		UpdatedTimestamp:    types.Time(time.UnixMilli(1720252232860)),
		Status:              "ready",
		RequiresSecurityKey: true,
		Network:             mockEthereumNetwork,
		NetworkDisplayName:  "Ethereum",
	}
	assert.Equal(t, exp, result, "SetAddressRequiresSecurityKey should decode a Coinbase wallet type entry")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetAddressRequiresSecurityKey should reject a null result")
	_, err = newUnavailableServerExchange(t).SetAddressRequiresSecurityKey(t.Context(), testLegacySecurityKeyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetAddressRequiresSecurityKey should return the request failure")
}

// testSetClearanceOriginatorRequest is the documented request setting a deposit's originator
func testSetClearanceOriginatorRequest() *SetClearanceOriginatorRequest {
	return &SetClearanceOriginatorRequest{
		Deposit: ClearanceDepositID{
			Currency:        currency.BTC,
			UserID:          mockDepositUserID,
			Address:         mockWithdrawalAddress,
			TransactionHash: mockDepositTransactionHash,
		},
		Originator: ClearanceOriginator{
			FirstName:   "First",
			LastName:    "Last",
			CompanyName: "Company Name",
			Address:     "NL, Amsterdam, Street, 1",
		},
	}
}

// testClearedDeposit is the documented deposit the mock sets the originator of
func testClearedDeposit() *Deposit {
	return &Deposit{
		Currency:          currency.BTC,
		UserID:            mockDepositUserID,
		Address:           mockWithdrawalAddress,
		Amount:            0.4,
		State:             "completed",
		TransactionID:     mockDepositTransactionHash,
		SourceAddress:     "A3BqqD5GRJ8wHy1PYyCXTe9ke5226Fha123",
		ReceivedTimestamp: types.Time(time.UnixMilli(1550574558607)),
		UpdatedTimestamp:  types.Time(time.UnixMilli(1550574558807)),
		Note:              "Note",
		ClearanceState:    "in_progress",
	}
}

func TestSetClearanceOriginator(t *testing.T) {
	t.Parallel()
	_, err := e.SetClearanceOriginator(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetClearanceOriginator must reject a nil request")
	arg := testSetClearanceOriginatorRequest()
	arg.Deposit.Currency = currency.EMPTYCODE
	_, err = e.SetClearanceOriginator(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetClearanceOriginator must reject a deposit without a currency")
	arg = testSetClearanceOriginatorRequest()
	arg.Deposit.UserID = 0
	_, err = e.SetClearanceOriginator(t.Context(), arg)
	require.ErrorIs(t, err, errDepositUserIDRequired, "SetClearanceOriginator must reject a deposit without a user ID")
	arg = testSetClearanceOriginatorRequest()
	arg.Deposit.Address = ""
	_, err = e.SetClearanceOriginator(t.Context(), arg)
	require.ErrorIs(t, err, errAddressRequired, "SetClearanceOriginator must reject a deposit without an address")
	arg = testSetClearanceOriginatorRequest()
	arg.Deposit.TransactionHash = ""
	_, err = e.SetClearanceOriginator(t.Context(), arg)
	require.ErrorIs(t, err, errTransactionHashRequired, "SetClearanceOriginator must reject a deposit without a transaction hash")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SetClearanceOriginator needs a deposit awaiting its originator, which live tests do not have")
	}
	result, err := e.SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	require.NoError(t, err, "SetClearanceOriginator must not error")
	assert.Equal(t, testClearedDeposit(), result, "SetClearanceOriginator should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetClearanceOriginator should reject a null result")
	_, err = newUnavailableServerExchange(t).SetClearanceOriginator(t.Context(), testSetClearanceOriginatorRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetClearanceOriginator should return the request failure")
}

func TestSetCoinbaseSubaccountWithdrawalsAllowed(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SetCoinbaseSubaccountWithdrawalsAllowed needs a Coinbase wallet type main account")
	}
	result, err := e.SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	require.NoError(t, err, "SetCoinbaseSubaccountWithdrawalsAllowed must not error")
	assert.Equal(t, &CoinbaseSubaccountWithdrawalsResponse{Enabled: true}, result, "SetCoinbaseSubaccountWithdrawalsAllowed should decode the setting")
	result, err = e.SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), false)
	require.NoError(t, err, "SetCoinbaseSubaccountWithdrawalsAllowed must not error when forbidding withdrawals")
	assert.Equal(t, &CoinbaseSubaccountWithdrawalsResponse{}, result, "SetCoinbaseSubaccountWithdrawalsAllowed should send and decode a false setting")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetCoinbaseSubaccountWithdrawalsAllowed should reject a null result")
	_, err = newUnavailableServerExchange(t).SetCoinbaseSubaccountWithdrawalsAllowed(t.Context(), true)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetCoinbaseSubaccountWithdrawalsAllowed should return the request failure")
}

// testSubmitTransferBetweenSubaccountsRequest is the documented transfer between subaccounts, with the nonce example
func testSubmitTransferBetweenSubaccountsRequest() *SubmitTransferBetweenSubaccountsRequest {
	return &SubmitTransferBetweenSubaccountsRequest{
		Currency:    currency.ETH,
		Amount:      12.1234,
		Destination: mockDestinationSubaccountID,
		Source:      mockSourceSubaccountID,
		Nonce:       mockIdempotencyNonce,
	}
}

// testTransferBetweenSubaccounts is the documented transfer the mock transfer between subaccounts returns, with the
// nonce it was requested with
func testTransferBetweenSubaccounts() *Transfer {
	transfer := testTransfer()
	transfer.Nonce = mockIdempotencyNonce
	return transfer
}

func TestSubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferBetweenSubaccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitTransferBetweenSubaccounts must reject a nil request")
	arg := testSubmitTransferBetweenSubaccountsRequest()
	arg.Currency = currency.EMPTYCODE
	_, err = e.SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubmitTransferBetweenSubaccounts must reject an empty currency")
	arg = testSubmitTransferBetweenSubaccountsRequest()
	arg.Amount = 0
	_, err = e.SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "SubmitTransferBetweenSubaccounts must reject a zero amount")
	arg = testSubmitTransferBetweenSubaccountsRequest()
	arg.Destination = 0
	_, err = e.SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errTransferDestinationRequired, "SubmitTransferBetweenSubaccounts must reject an empty destination")
	arg = testSubmitTransferBetweenSubaccountsRequest()
	arg.Nonce = "short"
	_, err = e.SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidIdempotencyNonce, "SubmitTransferBetweenSubaccounts must reject a nonce under 8 characters")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SubmitTransferBetweenSubaccounts needs subaccounts, which live tests do not have")
	}
	result, err := e.SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	require.NoError(t, err, "SubmitTransferBetweenSubaccounts must not error")
	assert.Equal(t, testTransferBetweenSubaccounts(), result, "SubmitTransferBetweenSubaccounts should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SubmitTransferBetweenSubaccounts should reject a null result")
	_, err = newUnavailableServerExchange(t).SubmitTransferBetweenSubaccounts(t.Context(), testSubmitTransferBetweenSubaccountsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitTransferBetweenSubaccounts should return the request failure")
}

// testSubmitTransferToSubaccountRequest is the documented transfer to a subaccount
func testSubmitTransferToSubaccountRequest() *SubmitTransferToSubaccountRequest {
	return &SubmitTransferToSubaccountRequest{Currency: currency.ETH, Amount: 12.1234, Destination: mockDestinationSubaccountID}
}

func TestSubmitTransferToSubaccount(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferToSubaccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitTransferToSubaccount must reject a nil request")
	_, err = e.SubmitTransferToSubaccount(t.Context(), &SubmitTransferToSubaccountRequest{Amount: 1, Destination: mockDestinationSubaccountID})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubmitTransferToSubaccount must reject an empty currency")
	_, err = e.SubmitTransferToSubaccount(t.Context(), &SubmitTransferToSubaccountRequest{Currency: currency.ETH, Amount: -1, Destination: mockDestinationSubaccountID})
	require.ErrorIs(t, err, errInvalidAmount, "SubmitTransferToSubaccount must reject a negative amount")
	_, err = e.SubmitTransferToSubaccount(t.Context(), &SubmitTransferToSubaccountRequest{Currency: currency.ETH, Amount: 1})
	require.ErrorIs(t, err, errTransferDestinationRequired, "SubmitTransferToSubaccount must reject an empty destination")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SubmitTransferToSubaccount needs a subaccount, which live tests do not have")
	}
	result, err := e.SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	require.NoError(t, err, "SubmitTransferToSubaccount must not error")
	assert.Equal(t, testTransfer(), result, "SubmitTransferToSubaccount should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SubmitTransferToSubaccount should reject a null result")
	_, err = newUnavailableServerExchange(t).SubmitTransferToSubaccount(t.Context(), testSubmitTransferToSubaccountRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitTransferToSubaccount should return the request failure")
}

// testSubmitTransferToUserRequest is the documented transfer to another user
func testSubmitTransferToUserRequest() *SubmitTransferToUserRequest {
	return &SubmitTransferToUserRequest{Currency: currency.ETH, Amount: 13.456, Destination: mockUserTransferDestination}
}

func TestSubmitTransferToUser(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitTransferToUser(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitTransferToUser must reject a nil request")
	_, err = e.SubmitTransferToUser(t.Context(), &SubmitTransferToUserRequest{Amount: 1, Destination: mockUserTransferDestination})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubmitTransferToUser must reject an empty currency")
	_, err = e.SubmitTransferToUser(t.Context(), &SubmitTransferToUserRequest{Currency: currency.ETH, Destination: mockUserTransferDestination})
	require.ErrorIs(t, err, errInvalidAmount, "SubmitTransferToUser must reject a zero amount")
	_, err = e.SubmitTransferToUser(t.Context(), &SubmitTransferToUserRequest{Currency: currency.ETH, Amount: 1})
	require.ErrorIs(t, err, errTransferDestinationRequired, "SubmitTransferToUser must reject an empty destination")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SubmitTransferToUser moves funds to another user, which a live test cannot undo")
	}
	result, err := e.SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	require.NoError(t, err, "SubmitTransferToUser must not error")
	assert.Equal(t, testTransfer(), result, "SubmitTransferToUser should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SubmitTransferToUser should reject a null result")
	_, err = newUnavailableServerExchange(t).SubmitTransferToUser(t.Context(), testSubmitTransferToUserRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitTransferToUser should return the request failure")
}

// testLegacyUpdateInAddressBookRequest is the documented legacy wallet type request updating a withdrawal address
func testLegacyUpdateInAddressBookRequest() *UpdateInAddressBookRequest {
	return &UpdateInAddressBookRequest{
		Currency:             currency.BTC,
		Type:                 AddressBookTypeWithdrawal,
		Address:              mockLegacyAddress,
		Label:                "Main address",
		BeneficiaryVASPName:  "Money`s Gone",
		BeneficiaryVASPDID:   "did:example:123456789abcdefghi",
		BeneficiaryFirstName: "John",
		BeneficiaryLastName:  "Doe",
		BeneficiaryAddress:   "NL, Amsterdam, Street, 1",
		Agreed:               new(true),
		Personal:             new(false),
	}
}

// testCoinbaseUpdateInAddressBookRequest is the documented Coinbase wallet type request relabelling an entry
func testCoinbaseUpdateInAddressBookRequest() *UpdateInAddressBookRequest {
	return &UpdateInAddressBookRequest{Currency: currency.ETH, Type: AddressBookTypeWithdrawal, EntryID: mockAddressBookEntryID, Label: "Updated label"}
}

func TestUpdateInAddressBook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateInAddressBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateInAddressBook must reject a nil request")
	_, err = e.UpdateInAddressBook(t.Context(), &UpdateInAddressBookRequest{Type: AddressBookTypeWithdrawal, Address: mockLegacyAddress})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateInAddressBook must reject an empty currency")
	_, err = e.UpdateInAddressBook(t.Context(), &UpdateInAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeDeposit, Address: mockLegacyAddress})
	require.ErrorIs(t, err, errInvalidAddressBookType, "UpdateInAddressBook must reject an address book type it cannot update")
	_, err = e.UpdateInAddressBook(t.Context(), &UpdateInAddressBookRequest{Currency: currency.BTC, Type: AddressBookTypeWithdrawal})
	require.ErrorIs(t, err, errAddressOrEntryIDRequired, "UpdateInAddressBook must reject a request without an address or entry ID")
	_, err = e.UpdateInAddressBook(t.Context(), &UpdateInAddressBookRequest{Currency: currency.ETH, Type: AddressBookTypeWithdrawal, EntryID: mockAddressBookEntryID, WalletType: "custodial"})
	require.ErrorIs(t, err, errInvalidBeneficiaryWalletType, "UpdateInAddressBook must reject an undocumented beneficiary wallet type")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Deribit asks for a security key to update a withdrawal address, which live tests cannot give")
	}
	result, err := e.UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	require.NoError(t, err, "UpdateInAddressBook must not error")
	assert.Equal(t, &UpdateInAddressBookResponse{}, result, "UpdateInAddressBook should accept a legacy wallet type acknowledgement")

	result, err = e.UpdateInAddressBook(t.Context(), testCoinbaseUpdateInAddressBookRequest())
	require.NoError(t, err, "UpdateInAddressBook must not error for a Coinbase wallet type entry")
	exp := testCoinbaseAddressBookEntry()
	exp.Label = "Updated label"
	exp.UpdatedTimestamp = types.Time(time.UnixMilli(1720252232860))
	exp.Status = "ready"
	exp.WaitingTimestamp = AddressWaitingTime{}
	exp.BeneficiaryUpdatedTimestamp = types.Time(time.UnixMilli(1720252232860))
	assert.Equal(t, &UpdateInAddressBookResponse{Entry: exp}, result, "UpdateInAddressBook should decode a Coinbase wallet type entry")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "UpdateInAddressBook should reject a null result")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"failed"}`).UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	assert.ErrorIs(t, err, errUnexpectedResult, "UpdateInAddressBook should reject an acknowledgement other than ok")
	_, err = newUnavailableServerExchange(t).UpdateInAddressBook(t.Context(), testLegacyUpdateInAddressBookRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "UpdateInAddressBook should return the request failure")
}

// testLegacyWithdrawRequest is the documented legacy wallet type withdrawal, with the nonce example
func testLegacyWithdrawRequest() *WithdrawRequest {
	return &WithdrawRequest{
		Currency: currency.BTC,
		Amount:   0.4,
		Address:  mockWithdrawalAddress,
		Priority: WithdrawalPriorityMid,
		Nonce:    mockIdempotencyNonce,
	}
}

// testRequestedLegacyWithdrawal is the documented legacy wallet type withdrawal the mock withdrawal returns, with the
// nonce it was requested with
func testRequestedLegacyWithdrawal() *Withdrawal {
	withdrawal := testLegacyWithdrawal()
	withdrawal.Nonce = mockIdempotencyNonce
	return withdrawal
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.Withdraw(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Withdraw must reject a nil request")
	arg := testLegacyWithdrawRequest()
	arg.Currency = currency.EMPTYCODE
	_, err = e.Withdraw(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "Withdraw must reject an empty currency")
	arg = testLegacyWithdrawRequest()
	arg.Amount = 0
	_, err = e.Withdraw(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "Withdraw must reject a zero amount")
	arg = testLegacyWithdrawRequest()
	arg.Address = ""
	_, err = e.Withdraw(t.Context(), arg)
	require.ErrorIs(t, err, errAddressOrEntryIDRequired, "Withdraw must reject a request without an address or entry ID")
	arg = testLegacyWithdrawRequest()
	arg.Priority = "normal"
	_, err = e.Withdraw(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidWithdrawalPriority, "Withdraw must reject an undocumented priority")
	arg = testLegacyWithdrawRequest()
	arg.Nonce = strings.Repeat("n", 129)
	_, err = e.Withdraw(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidIdempotencyNonce, "Withdraw must reject a nonce over 128 characters")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Withdraw moves funds off Deribit, which a live test cannot undo")
	}
	result, err := e.Withdraw(t.Context(), testLegacyWithdrawRequest())
	require.NoError(t, err, "Withdraw must not error")
	assert.Equal(t, testRequestedLegacyWithdrawal(), result, "Withdraw should decode every field")

	result, err = e.Withdraw(t.Context(), &WithdrawRequest{Currency: currency.ETH, Amount: 0.5, EntryID: mockAddressBookEntryID})
	require.NoError(t, err, "Withdraw must not error for a Coinbase wallet type entry")
	exp := &Withdrawal{
		ID:                     mockAddressBookEntryID,
		TransferIdempotencyKey: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
		Currency:               currency.ETH,
		State:                  "unconfirmed",
		Amount:                 0.5,
		Address:                mockCoinbaseAddress,
		Network:                mockEthereumNetwork,
		NetworkDisplayName:     "Ethereum",
		CreatedTimestamp:       types.Time(time.UnixMilli(1719993033041)),
		UpdatedTimestamp:       types.Time(time.UnixMilli(1719993033041)),
	}
	assert.Equal(t, exp, result, "Withdraw should decode a Coinbase wallet type withdrawal")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).Withdraw(t.Context(), testLegacyWithdrawRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "Withdraw should reject a null result")
	_, err = newUnavailableServerExchange(t).Withdraw(t.Context(), testLegacyWithdrawRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "Withdraw should return the request failure")
}

func TestValidateAddressBookType(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateAddressBookType(AddressBookTypeTransfer, AddressBookTypeTransfer, AddressBookTypeWithdrawal), "validateAddressBookType should accept an accepted type")
	for _, addressBookType := range []string{"", "Withdrawal", AddressBookTypeDepositSource} {
		assert.ErrorIsf(t, validateAddressBookType(addressBookType, AddressBookTypeTransfer, AddressBookTypeWithdrawal), errInvalidAddressBookType, "validateAddressBookType should reject type %q", addressBookType)
	}
}

func TestValidateAddressBookEntry(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		currency currency.Code
		addrType string
		address  string
		entryID  string
		err      error
	}{
		{name: "empty currency", addrType: AddressBookTypeWithdrawal, address: mockLegacyAddress, err: currency.ErrCurrencyCodeEmpty},
		{name: "unaccepted type", currency: currency.BTC, addrType: AddressBookTypeDeposit, address: mockLegacyAddress, err: errInvalidAddressBookType},
		{name: "no address or entry ID", currency: currency.BTC, addrType: AddressBookTypeWithdrawal, err: errAddressOrEntryIDRequired},
		{name: "address", currency: currency.BTC, addrType: AddressBookTypeWithdrawal, address: mockLegacyAddress},
		{name: "entry ID", currency: currency.NewCode("all"), addrType: AddressBookTypeTransfer, entryID: mockAddressBookEntryID},
	} {
		err := validateAddressBookEntry(tc.currency, tc.addrType, tc.address, tc.entryID, AddressBookTypeTransfer, AddressBookTypeWithdrawal)
		assert.ErrorIsf(t, err, tc.err, "validateAddressBookEntry should return the expected error for %s", tc.name)
	}
}

func TestValidateBeneficiaryTypes(t *testing.T) {
	t.Parallel()
	for _, entityType := range []string{"", BeneficiaryEntityTypeOrganisation, BeneficiaryEntityTypeIndividual} {
		for _, walletType := range []string{"", BeneficiaryWalletTypeExchange, BeneficiaryWalletTypeSelfHosted} {
			assert.NoErrorf(t, validateBeneficiaryTypes(entityType, walletType), "validateBeneficiaryTypes should accept entity type %q and wallet type %q", entityType, walletType)
		}
	}
	assert.ErrorIs(t, validateBeneficiaryTypes("organisation", ""), errInvalidBeneficiaryEntityType, "validateBeneficiaryTypes should reject an entity type Deribit does not document")
	assert.ErrorIs(t, validateBeneficiaryTypes("", "hosted"), errInvalidBeneficiaryWalletType, "validateBeneficiaryTypes should reject a wallet type Deribit does not document")
}

func TestValidateIdempotencyNonce(t *testing.T) {
	t.Parallel()
	for _, nonce := range []string{"", strings.Repeat("n", 8), strings.Repeat("n", 128), strings.Repeat("é", 8)} {
		assert.NoErrorf(t, validateIdempotencyNonce(nonce), "validateIdempotencyNonce should accept a nonce of %d characters", utf8.RuneCountInString(nonce))
	}
	for _, nonce := range []string{strings.Repeat("n", 7), strings.Repeat("n", 129), strings.Repeat("é", 7)} {
		assert.ErrorIsf(t, validateIdempotencyNonce(nonce), errInvalidIdempotencyNonce, "validateIdempotencyNonce should reject a nonce of %d characters", utf8.RuneCountInString(nonce))
	}
}

func TestAddressWaitingTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   time.Time
	}{
		{input: `1720252232860`, exp: time.UnixMilli(1720252232860)},
		{input: `null`},
		{input: `true`},
		{input: `false`},
	} {
		var waiting AddressWaitingTime
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &waiting), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, waiting.Time(), "UnmarshalJSON should decode %s", tc.input)
	}
	var waiting AddressWaitingTime
	assert.Error(t, json.Unmarshal([]byte(`"soon"`), &waiting), "UnmarshalJSON should reject a value that is not a timestamp or boolean")
}

func TestWithdrawalIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   WithdrawalID
	}{
		{input: `1`, exp: "1"},
		{input: `12345678901234`, exp: "12345678901234"},
		{input: `"0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5b"`, exp: "0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5b"},
		{input: `null`},
	} {
		var id WithdrawalID
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &id), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, id, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`-1`, `1.5`, `true`, `{}`} {
		var id WithdrawalID
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &id), errInvalidWithdrawalID, "UnmarshalJSON should reject %s", input)
	}
}

func TestRewardEligibilityResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result RewardEligibilityResponse
	require.NoError(t, json.Unmarshal([]byte(`{"usdc":{"eligibility_status":"partially_eligible","apr_sma7":4.156},"steth":{"eligibility_status":"eligible","apr_sma7":2.5}}`), &result), "Unmarshal must not error")
	exp := RewardEligibilityResponse{
		currency.USDC:  {EligibilityStatus: "partially_eligible", APRSimpleMovingAverage7Day: 4.156},
		currency.STETH: {EligibilityStatus: "eligible", APRSimpleMovingAverage7Day: 2.5},
	}
	assert.Equal(t, exp, result, "UnmarshalJSON should key each eligibility by its upper case currency code, so the package's currency codes find it")

	result = nil
	require.NoError(t, json.Unmarshal([]byte(`null`), &result), "Unmarshal must not error for null")
	assert.Nil(t, result, "UnmarshalJSON should leave a null result nil")
	require.NoError(t, json.Unmarshal([]byte(`{}`), &result), "Unmarshal must not error for an empty object")
	assert.Equal(t, RewardEligibilityResponse{}, result, "UnmarshalJSON should decode an empty object as an empty map")
	assert.Error(t, json.Unmarshal([]byte(`[]`), &result), "UnmarshalJSON should reject a result that is not an object")
}

func TestUpdateInAddressBookResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result UpdateInAddressBookResponse
	require.NoError(t, json.Unmarshal([]byte(`"ok"`), &result), "Unmarshal must not error for an acknowledgement")
	assert.Nil(t, result.Entry, "UnmarshalJSON should leave Entry nil for an acknowledgement")

	require.NoError(t, json.Unmarshal([]byte(`{"id":"0190a1b2-7c3d-7e5f-8a9b-0c1d2e3f4a5b","currency":"ETH","label":"Updated label"}`), &result), "Unmarshal must not error for an entry")
	assert.Equal(t, UpdateInAddressBookResponse{Entry: &AddressBookEntry{ID: mockAddressBookEntryID, Currency: currency.ETH, Label: "Updated label"}}, result, "UnmarshalJSON should decode the updated entry")

	require.NoError(t, json.Unmarshal([]byte(`"ok"`), &result), "Unmarshal must not error for an acknowledgement after an entry")
	assert.Nil(t, result.Entry, "UnmarshalJSON should clear a previously decoded entry for an acknowledgement")

	assert.ErrorIs(t, json.Unmarshal([]byte(`"failed"`), &result), errUnexpectedResult, "UnmarshalJSON should reject an acknowledgement other than ok")
	assert.Error(t, json.Unmarshal([]byte(`[]`), &result), "UnmarshalJSON should reject a result that is neither an acknowledgement nor an entry")
}
