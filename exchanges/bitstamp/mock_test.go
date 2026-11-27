//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitstamp

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	if useTestNet {
		log.Fatal("cannot use testnet with mock tests")
	}

	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bitstamp Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(e, "api"); err != nil {
		log.Fatalf("Bitstamp MockHTTPInstance error: %s", err)
	}

	os.Exit(m.Run())
}

// TestRecordingSkipsMockOnlySubtests checks that the tests whose mocked subtests change the account send no request
// through e while it records. It runs serially, as recording changes the exchange every test shares
func TestRecordingSkipsMockOnlySubtests(t *testing.T) {
	sent := testexch.RequestsWhileRecording(t, e, map[string]func(*testing.T){
		"TestCreateContact":                        TestCreateContact,
		"TestCreateExtendedPublicKeyRegistration":  TestCreateExtendedPublicKeyRegistration,
		"TestCreateInstantConvertAddress":          TestCreateInstantConvertAddress,
		"TestCreateSatoshiTest":                    TestCreateSatoshiTest,
		"TestCryptoWithdrawal":                     TestCryptoWithdrawal,
		"TestEarnSubscribe":                        TestEarnSubscribe,
		"TestEarnUnsubscribe":                      TestEarnUnsubscribe,
		"TestRejectCryptoDeposit":                  TestRejectCryptoDeposit,
		"TestRevokeAllAPIKeys":                     TestRevokeAllAPIKeys,
		"TestRevokeExtendedPublicKeyRegistration":  TestRevokeExtendedPublicKeyRegistration,
		"TestRippleIOUWithdrawal":                  TestRippleIOUWithdrawal,
		"TestSetEarnSubscriptionSetting":           TestSetEarnSubscriptionSetting,
		"TestSubmitCounterpartyInfo":               TestSubmitCounterpartyInfo,
		"TestTransferFromMainAccount":              TestTransferFromMainAccount,
		"TestUpdateCryptoDepositOriginator":        TestUpdateCryptoDepositOriginator,
		"TestWithdrawCryptocurrencyFunds":          TestWithdrawCryptocurrencyFunds,
		"TestWithdrawFiatFunds":                    TestWithdrawFiatFunds,
		"TestWithdrawFiatFundsToInternationalBank": TestWithdrawFiatFundsToInternationalBank,
	}, "api")
	assert.Empty(t, sent, "These tests should send no request through e while it records")
}
