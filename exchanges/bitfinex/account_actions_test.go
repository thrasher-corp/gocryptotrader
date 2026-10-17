package bitfinex

import (
	"context"
	"io"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Mock inputs, which the recorded requests in testdata/http.json hold
const (
	testLoginHistoryStart int64  = 1579000000000
	testLoginHistoryEnd   int64  = 1579300000000
	testChangelogStart    int64  = 1574300000000
	testChangelogEnd      int64  = 1574400000000
	testMovementsStart    int64  = 1569340000000
	testMovementsEnd      int64  = 1569350000000
	testLNXPaymentsOffset uint64 = 10
	testLNXPaymentHash           = "05f589c28838e8f08d83ed6419ab54a6e58cfc125e1e7f63c2d350f504ed34d0"
	// testUserSettingKey is the setting the live user settings tests write and delete
	testUserSettingKey = "api:gct_test"
)

// testGenerateTokenRequest is the token request GenerateToken's mock response answers
func testGenerateTokenRequest() *GenerateTokenRequest {
	return &GenerateTokenRequest{
		TimeToLive:      5 * time.Minute,
		Capabilities:    []string{TokenCapabilityAccount, TokenCapabilityOrders, TokenCapabilityFunding},
		WritePermission: true,
		UserIP:          "192.0.2.1",
	}
}

// testWithdrawalRequest is the withdrawal RequestWithdrawal's mock response answers, with the docs' travel rule
// example details
func testWithdrawalRequest() *WithdrawalRequest {
	return &WithdrawalRequest{
		Wallet:    WalletTypeExchange,
		Method:    "ETHEREUM",
		Amount:    0.01,
		Address:   "0x6C20eBD2222CE0F809713EF06AE400C2445B72B7",
		DeductFee: true,
		Note:      "Purchase of 100 pizzas",
		TravelRule: &TravelRuleDetails{
			VASPDID:   "did:ethr:0x36c12a3d41698534463a6dcda1c549b8f178249b",
			FirstName: "John",
			LastName:  "Doe",
		},
	}
}

// testLNXInvoice is the invoice the LNX mock responses hold, the payments' alike
var testLNXInvoice = LNXInvoice{
	PaymentHash:        testLNXPaymentHash,
	CreatedAt:          types.Time(time.UnixMilli(1732125293487)),
	ExpiresAt:          types.Time(time.UnixMilli(1732128893487)),
	Invoice:            "lnbc10m1p0rn2fkpp5ng8cq7084jssenhcmdl2sczgtvja26ufz47rx4n7azsr03dk6evqdp5xy6kzwfsvfjrzd3cxcux2wfexcmrxcek8psnjdpkvyurxef38y6scqzpgxqzjcyrp0kymmeqejdelm3r586ehz5cs5ntxx96hr3ec49q70rlnljzcpag5pynsalrwmd0w8serwmmtjru6g6z3a607kqw47mqgm6m4ztkqqxgkujx",
	RecipientPublicKey: "022bd67d43b0b27d6bbef581d8d4ba214efe16307cff001b2bf10d0642ed89a973",
	Memo:               "Bitfinex deposit",
	AmountSatoshis:     1000000,
	AmountBTC:          0.01,
	Status:             "completed",
}

// newRequestBodyExchange returns an exchange whose requests a test server answers with one response, asserting their
// path and passing each body on; a request arriving before the last body is read fails rather than blocks
func newRequestBodyExchange(t *testing.T, path, response string) (ex *Exchange, bodies <-chan string) {
	t.Helper()
	received := make(chan string, 1)
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "The request should be sent to the endpoint's path")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "Reading the request body should not error")
		select {
		case received <- string(body):
		default:
			assert.Fail(t, "The test server should receive one request at a time")
		}
		_, err = w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	return ex, received
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetUserInfo(t.Context())
	require.NoError(t, err, "GetUserInfo must not error")
	if mockTests {
		lastLogin, err := time.Parse(time.RFC3339, "2022-09-06T12:14:14.600Z")
		require.NoError(t, err, "Parse must not error")
		exp := &UserInfoResponse{
			ID:                                123456,
			Email:                             "email@email.com",
			Username:                          "AccountName",
			CreatedAt:                         types.Time(time.UnixMilli(1613492801000)),
			Verified:                          true,
			VerificationLevel:                 3,
			Timezone:                          "Lisbon",
			Locale:                            "en",
			Company:                           "bitfinex",
			EmailVerified:                     true,
			MasterAccountCreatedAt:            types.Time(time.UnixMilli(1613492801000)),
			GroupID:                           79,
			MasterAccountID:                   123456,
			InheritsMasterAccountVerification: true,
			IsGroupMaster:                     true,
			GroupWithdrawEnabled:              true,
			TwoFactorModes:                    []string{"u2f"},
			LastLoginAt:                       lastLogin,
			ComplianceCountries:               []string{"FR", "FR_Paris"},
			ComplianceResidenceCountries:      []string{"FR", "FR_Paris"},
			IsMerchantEnterprise:              true,
		}
		assert.Equal(t, exp, result, "GetUserInfo should decode every field")
	} else {
		assert.NotZero(t, result.ID, "GetUserInfo should return the account ID")
	}

	_, err = newResponseServerExchange(t, `null`).GetUserInfo(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserInfo should error for a null response")
	_, err = newUnavailableServerExchange(t).GetUserInfo(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserInfo should return the request failure")
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountSummary(t.Context())
	require.NoError(t, err, "GetAccountSummary must not error")
	if mockTests {
		exp := &AccountSummaryResponse{
			FeeRates: SummaryFeeRates{
				Maker: SummaryMakerFeeRates{ToCrypto: 0.001, ToStable: 0.001, ToFiat: 0.001, DerivativeRebate: 0.0002},
				Taker: SummaryTakerFeeRates{ToCrypto: 0.002, ToStable: 0.002, ToFiat: 0.002, Derivative: 0.0065},
			},
			TradingVolumeAndFees: SummaryTradingVolumeAndFees{
				Volumes: []SummaryTradeVolume{
					{Currency: "BTCF0", Volume: 0.016},
					{Currency: "BTC", Volume: 0.765},
					{Currency: "Total (USD)", Volume: 52069.01928633, VolumeSafe: 52069.01928633, VolumeMaker: 11362.82692633, VolumeBFX: 52069.01928633, VolumeBFXSafe: 52069.01928633, VolumeBFXMaker: 11362.82692633},
				},
				Fees:         map[currency.Code]float64{currency.NewCode("USTF0"): 0.17569482, currency.USD: 38.95555663, currency.BTC: 0.0005152},
				TotalFeesUSD: 51.12300602,
			},
			FundingEarnings: SummaryFundingEarnings{
				Earnings:         map[currency.Code]float64{currency.USD: 21.87, currency.BTC: 0.04},
				TotalEarningsUSD: 82.1078,
			},
		}
		assert.Equal(t, exp, result, "GetAccountSummary should decode every field")
	} else {
		assert.Positive(t, result.FeeRates.Taker.ToCrypto, "GetAccountSummary should return the taker fee rate")
	}

	_, err = newResponseServerExchange(t, `null`).GetAccountSummary(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSummary should error for a null response")
	_, err = newUnavailableServerExchange(t).GetAccountSummary(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSummary should return the request failure")
}

func TestGetLoginHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoginHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLoginHistory must reject a nil request")
	_, err = e.GetLoginHistory(t.Context(), &LoginHistoryRequest{Start: time.UnixMilli(testLoginHistoryEnd), End: time.UnixMilli(testLoginHistoryStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLoginHistory must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &LoginHistoryRequest{Start: time.UnixMilli(testLoginHistoryStart), End: time.UnixMilli(testLoginHistoryEnd), Limit: 25}
	if !mockTests {
		req = &LoginHistoryRequest{Start: time.Now().AddDate(0, -1, 0), End: time.Now(), Limit: 25}
	}
	result, err := e.GetLoginHistory(t.Context(), req)
	require.NoError(t, err, "GetLoginHistory must not error")
	if mockTests {
		exp := []Login{{
			ID:        78404724,
			Time:      types.Time(time.UnixMilli(1579202754000)),
			IP:        "184.214.55.220",
			ExtraInfo: LoginExtraInfo(`{"asn": "28545, Cablemas Telecomunicaciones SA de CV", "geo": "Cancún, MX", "user_agent":   {"os": "Windows 10", "raw": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.117 Safari/537.36", "browser": "Chrome", "version": "79.0.3945.117", "platform": "Windows", "is_mobile": false}}`),
		}}
		assert.Equal(t, exp, result, "GetLoginHistory should decode every field, and the extra info from its string")
	}

	_, err = newUnavailableServerExchange(t).GetLoginHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLoginHistory should return the request failure")
}

func TestGetKeyPermissions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetKeyPermissions(t.Context())
	require.NoError(t, err, "GetKeyPermissions must not error")
	if mockTests {
		exp := []KeyPermission{
			{Scope: "account", Read: true, Write: true},
			{Scope: "history", Read: true, Write: true},
			{Scope: "orders", Read: true, Write: true},
			{Scope: "positions"},
			{Scope: "funding", Read: true, Write: true},
			{Scope: "settings"},
			{Scope: "wallets", Read: true, Write: true},
			{Scope: "withdraw", Write: true},
			{Scope: "ui_withdraw"},
			{Scope: "bfxpay"},
		}
		assert.Equal(t, exp, result, "GetKeyPermissions should decode every permission")
	} else {
		assert.NotEmpty(t, result, "GetKeyPermissions should return the key's permissions")
	}

	_, err = newUnavailableServerExchange(t).GetKeyPermissions(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetKeyPermissions should return the request failure")
}

func TestGenerateToken(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GenerateToken must reject a nil request")
	for _, ttl := range []time.Duration{-time.Second, 1500 * time.Millisecond, time.Millisecond} {
		_, err = e.GenerateToken(t.Context(), &GenerateTokenRequest{TimeToLive: ttl})
		require.ErrorIsf(t, err, errInvalidTokenTimeToLive, "GenerateToken must refuse time to live %s", ttl)
	}
	_, err = e.GenerateToken(t.Context(), &GenerateTokenRequest{Capabilities: []string{TokenCapabilityAccount, "x"}})
	require.ErrorIs(t, err, errInvalidTokenCapability, "GenerateToken must refuse an unknown capability")
	_, err = e.GenerateToken(t.Context(), &GenerateTokenRequest{Capabilities: []string{TokenCapabilityWithdraw}})
	require.ErrorIs(t, err, errTokenWithdrawNeedsWrite, "GenerateToken must refuse the withdraw capability without write permission")

	skipLiveUnlessCanManipulate(t)
	req := testGenerateTokenRequest()
	if !mockTests {
		req = &GenerateTokenRequest{TimeToLive: time.Minute, Capabilities: []string{TokenCapabilityAccount}}
	}
	result, err := e.GenerateToken(t.Context(), req)
	require.NoError(t, err, "GenerateToken must not error")
	if mockTests {
		assert.Equal(t, &GenerateTokenResponse{Token: "pub:api:665b6328-cc1e-4627-8caf-9ea5ddb8dfb0-caps:a:o:f-write"}, result, "GenerateToken should decode the token")
	} else {
		assert.NotEmpty(t, result.Token, "GenerateToken should return a token")
	}

	_, err = newResponseServerExchange(t, `null`).GenerateToken(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GenerateToken should error for a null response")
	_, err = newUnavailableServerExchange(t).GenerateToken(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GenerateToken should return the request failure")
}

func TestGetChangelog(t *testing.T) {
	t.Parallel()
	_, err := e.GetChangelog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetChangelog must reject a nil request")
	_, err = e.GetChangelog(t.Context(), &ChangelogRequest{Start: time.UnixMilli(testChangelogEnd), End: time.UnixMilli(testChangelogStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetChangelog must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &ChangelogRequest{Start: time.UnixMilli(testChangelogStart), End: time.UnixMilli(testChangelogEnd), Limit: 10}
	if !mockTests {
		req = &ChangelogRequest{Start: time.Now().AddDate(-1, 0, 0), End: time.Now(), Limit: 10}
	}
	result, err := e.GetChangelog(t.Context(), req)
	require.NoError(t, err, "GetChangelog must not error")
	if mockTests {
		exp := []ChangelogEntry{{
			CreatedAt: types.Time(time.UnixMilli(1574332100000)),
			Log:       "settings: api key created",
			IP:        "182.64.34.9",
			UserAgent: "Mozilla/5.0 (X11; CrOS x86_64 12499.51.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.92 Safari/537.36",
		}}
		assert.Equal(t, exp, result, "GetChangelog should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetChangelog(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetChangelog should return the request failure")
}

func TestTransferBetweenWallets(t *testing.T) {
	t.Parallel()
	_, err := e.TransferBetweenWallets(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "TransferBetweenWallets must reject a nil request")
	// trading and deposit are the old names of the margin and funding wallets
	for _, wallet := range []string{"", "trading", "deposit", "Exchange"} {
		_, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: wallet, To: WalletTypeExchange})
		require.ErrorIsf(t, err, errInvalidAccountWallet, "TransferBetweenWallets must refuse from wallet %q", wallet)
		_, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: wallet})
		require.ErrorIsf(t, err, errInvalidAccountWallet, "TransferBetweenWallets must refuse to wallet %q", wallet)
	}
	_, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeFunding})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "TransferBetweenWallets must require a currency")
	for _, amount := range []float64{0, -1} {
		_, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeFunding, Currency: currency.UST, Amount: amount})
		require.ErrorIsf(t, err, errInvalidAmount, "TransferBetweenWallets must refuse amount %v", amount)
	}
	_, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeFunding, Currency: currency.UST, Amount: 1, DestinationUserID: 1, TwoFactorToken: &TwoFactorToken{Method: "sms"}})
	require.ErrorIs(t, err, errInvalidTwoFactorMethod, "TransferBetweenWallets must refuse an invalid two factor token")

	// The recorded requests transfer between the account's own wallets; these transfer to other accounts
	ex, bodies := newRequestBodyExchange(t, "/v2/auth/w/transfer", `[1568736745789,"acc_tf",null,null,[1568736745790,"exchange","exchange",null,"UST",null,null,10],null,"SUCCESS","10.0 Tether USDt transferred from Exchange to Exchange"]`)
	_, err = ex.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeExchange, Currency: currency.UST, Amount: 10, DestinationEmail: "sub@example.com", TwoFactorToken: &TwoFactorToken{Method: TwoFactorMethodOneTimePassword, Token: "123456"}})
	require.NoError(t, err, "TransferBetweenWallets must not error to another account by email")
	assert.JSONEq(t, `{"from":"exchange","to":"exchange","currency":"UST","amount":"10","email_dst":"sub@example.com","tfaToken":{"method":"otp","token":"123456"}}`, <-bodies, "TransferBetweenWallets should send the destination email and one time password")
	assertion := &WebAuthnAssertion{ID: "id1", Response: &WebAuthnAssertionResponse{AuthenticatorData: "authenticator", ClientDataJSON: "client", Signature: "signature"}}
	_, err = ex.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeExchange, Currency: currency.UST, Amount: 10, DestinationUserID: 1234, TwoFactorToken: &TwoFactorToken{Method: TwoFactorMethodU2F, Response: assertion}})
	require.NoError(t, err, "TransferBetweenWallets must not error to another account by user ID")
	assert.JSONEq(t, `{"from":"exchange","to":"exchange","currency":"UST","amount":"10","user_id_dst":1234,"tfaToken":{"method":"u2f","response":{"id":"id1","response":{"authenticatorData":"authenticator","clientDataJSON":"client","signature":"signature"}}}}`, <-bodies, "TransferBetweenWallets should send the destination user ID and assertion")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		// Needs 1 UST in the exchange wallet, which is transferred to the funding wallet and back
		for _, req := range []*TransferBetweenWalletsRequest{
			{From: WalletTypeExchange, To: WalletTypeFunding, Currency: currency.UST, Amount: 1},
			{From: WalletTypeFunding, To: WalletTypeExchange, Currency: currency.UST, Amount: 1},
		} {
			result, err := e.TransferBetweenWallets(t.Context(), req)
			require.NoErrorf(t, err, "TransferBetweenWallets must not error from %s to %s", req.From, req.To)
			assert.Equalf(t, req.To, result.Data.WalletTo, "TransferBetweenWallets should transfer to the %s wallet", req.To)
		}
		return
	}
	result, err := e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeMargin, To: WalletTypeExchange, Currency: currency.USD, Amount: 50})
	require.NoError(t, err, "TransferBetweenWallets must not error")
	exp := &Notification[WalletTransfer]{
		Time:   types.Time(time.UnixMilli(1568736745789)),
		Type:   "acc_tf",
		Data:   WalletTransfer{UpdatedAt: types.Time(time.UnixMilli(1568736745790)), WalletFrom: WalletTypeMargin, WalletTo: WalletTypeExchange, Currency: currency.USD, Amount: 50},
		Status: NotificationStatusSuccess,
		Text:   "50.0 US Dollar transferred from Margin to Exchange",
	}
	assert.Equal(t, exp, result, "TransferBetweenWallets should decode the transfer")
	result, err = e.TransferBetweenWallets(t.Context(), &TransferBetweenWalletsRequest{From: WalletTypeExchange, To: WalletTypeMargin, Currency: currency.UST, CurrencyTo: currency.NewCode("USTF0"), Amount: 200})
	require.NoError(t, err, "TransferBetweenWallets must not error converting the currency")
	exp = &Notification[WalletTransfer]{
		Time:   types.Time(time.UnixMilli(1574173088379)),
		Type:   "acc_tf",
		Data:   WalletTransfer{UpdatedAt: types.Time(time.UnixMilli(1574173088379)), WalletFrom: WalletTypeExchange, WalletTo: WalletTypeMargin, Currency: currency.UST, CurrencyTo: currency.NewCode("USTF0"), Amount: 200},
		Status: NotificationStatusSuccess,
		Text:   "200.0 Tether USDt transfered from Exchange to Margin", //nolint:misspell // Bitfinex's notification text, as the docs give it
	}
	assert.Equal(t, exp, result, "TransferBetweenWallets should decode the converting transfer")

	req := &TransferBetweenWalletsRequest{From: WalletTypeMargin, To: WalletTypeExchange, Currency: currency.USD, Amount: 50}
	_, err = newResponseServerExchange(t, `null`).TransferBetweenWallets(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "TransferBetweenWallets should error for a null response")
	_, err = newUnavailableServerExchange(t).TransferBetweenWallets(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "TransferBetweenWallets should return the request failure")
}

func TestRequestDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.RequestDepositAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestDepositAddress must reject a nil request")
	_, err = e.RequestDepositAddress(t.Context(), &DepositAddressRequest{Wallet: "trading", Method: "bitcoin"})
	require.ErrorIs(t, err, errInvalidAccountWallet, "RequestDepositAddress must refuse an invalid wallet")
	_, err = e.RequestDepositAddress(t.Context(), &DepositAddressRequest{Wallet: WalletTypeExchange})
	require.ErrorIs(t, err, errTransactionMethodRequired, "RequestDepositAddress must require a method")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		result, err := e.RequestDepositAddress(t.Context(), &DepositAddressRequest{Wallet: WalletTypeExchange, Method: "bitcoin"})
		require.NoError(t, err, "RequestDepositAddress must not error")
		assert.NotEmpty(t, result.Data.Address, "RequestDepositAddress should return an address")
		return
	}
	exp := &Notification[DepositAddress]{
		Time:   types.Time(time.UnixMilli(1568738594687)),
		Type:   "acc_dep",
		Data:   DepositAddress{Method: "BITCOIN", Currency: currency.BTC, Address: "YOURADDRESS"},
		Status: NotificationStatusSuccess,
		Text:   "success",
	}
	// The recorded requests hold the method in lower case, and op_renew only for a renewal
	for _, req := range []*DepositAddressRequest{
		{Wallet: WalletTypeExchange, Method: "bitcoin"},
		{Wallet: WalletTypeExchange, Method: "BITCOIN", Renew: true},
	} {
		result, err := e.RequestDepositAddress(t.Context(), req)
		require.NoErrorf(t, err, "RequestDepositAddress must not error with renew %t", req.Renew)
		assert.Equalf(t, exp, result, "RequestDepositAddress should decode the address with renew %t", req.Renew)
	}

	req := &DepositAddressRequest{Wallet: WalletTypeExchange, Method: "bitcoin"}
	_, err = newResponseServerExchange(t, `null`).RequestDepositAddress(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestDepositAddress should error for a null response")
	_, err = newUnavailableServerExchange(t).RequestDepositAddress(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestDepositAddress should return the request failure")
}

func TestGetDepositAddressList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDepositAddressList must reject a nil request")
	_, err = e.GetDepositAddressList(t.Context(), &DepositAddressListRequest{})
	require.ErrorIs(t, err, errTransactionMethodRequired, "GetDepositAddressList must require a method")

	skipLiveWithoutCredentials(t)
	req := &DepositAddressListRequest{Method: "TETHERUSE", Page: 3, PageSize: 25}
	if !mockTests {
		req = &DepositAddressListRequest{Method: "bitcoin"}
	}
	result, err := e.GetDepositAddressList(t.Context(), req)
	require.NoError(t, err, "GetDepositAddressList must not error")
	if mockTests {
		exp := []WalletDepositAddress{
			{Address: "0x6C20eBD2222CE0F809713EF06AE400C2445B72B7", WalletType: WalletTypeExchange},
			{Address: "0xcd752F90494d103F993aB91472129e9DF4987ED0", WalletType: WalletTypeMargin},
			{Address: "0x01Dd2e1E9bE8C4667C645c8FA694aB89B894d734", WalletType: WalletTypeExchange},
			{Address: "0x542BECa0Fc09d2478118b0E777Ee2928F01acC0a", WalletType: WalletTypeFunding},
		}
		assert.Equal(t, exp, result, "GetDepositAddressList should decode every address")
	}

	_, err = newUnavailableServerExchange(t).GetDepositAddressList(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDepositAddressList should return the request failure")
}

func TestGenerateInvoice(t *testing.T) {
	t.Parallel()
	for _, amount := range []float64{0, -0.01} {
		_, err := e.GenerateInvoice(t.Context(), amount)
		require.ErrorIsf(t, err, errInvalidAmount, "GenerateInvoice must refuse amount %v", amount)
	}

	skipLiveUnlessCanManipulate(t)
	amount := 0.01
	if !mockTests {
		amount = 0.0001
		// An account's first invoice needs its LNX deposit address
		_, err := e.RequestDepositAddress(t.Context(), &DepositAddressRequest{Wallet: WalletTypeExchange, Method: TransactionMethodLightning})
		require.NoError(t, err, "RequestDepositAddress must return the LNX deposit address")
	}
	result, err := e.GenerateInvoice(t.Context(), amount)
	require.NoError(t, err, "GenerateInvoice must not error")
	if mockTests {
		exp := &GenerateInvoiceResponse{
			InvoiceHash: "9a0f8079e7aca10ccef8db7ea860485b25d56b89157c33567ee8a037c5b6d658",
			Invoice:     testLNXInvoice.Invoice,
			Amount:      0.01,
		}
		assert.Equal(t, exp, result, "GenerateInvoice should decode every field")
	} else {
		assert.NotEmpty(t, result.Invoice, "GenerateInvoice should return an invoice")
	}

	_, err = newResponseServerExchange(t, `null`).GenerateInvoice(t.Context(), amount)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GenerateInvoice should error for a null response")
	_, err = newUnavailableServerExchange(t).GenerateInvoice(t.Context(), amount)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GenerateInvoice should return the request failure")
}

func TestLNXInvoicePayments(t *testing.T) {
	t.Parallel()
	ex, bodies := newRequestBodyExchange(t, "/v2/auth/r/ext/invoice/payments", `{"payments":[{"payment_hash":"abc"}]}`)
	var resp lnxInvoiceListResponse
	require.NoError(t, ex.lnxInvoicePayments(t.Context(), "getPaymentsByUser", lnxInvoicePaymentsQuery{Offset: 5}, &resp), "lnxInvoicePayments must not error")
	assert.JSONEq(t, `{"action":"getPaymentsByUser","query":{"offset":5}}`, <-bodies, "lnxInvoicePayments should send the action and query")
	assert.Equal(t, lnxInvoiceListResponse{Payments: []LNXInvoice{{PaymentHash: "abc"}}}, resp, "lnxInvoicePayments should decode the response")
	require.NoError(t, ex.lnxInvoicePayments(t.Context(), "getInvoiceById", lnxInvoicePaymentsQuery{PaymentHash: "abc"}, nil), "lnxInvoicePayments must not error")
	assert.JSONEq(t, `{"action":"getInvoiceById","query":{"txid":"abc"}}`, <-bodies, "lnxInvoicePayments should send the payment hash as txid")
	require.NoError(t, ex.lnxInvoicePayments(t.Context(), "getInvoicesByUser", lnxInvoicePaymentsQuery{}, nil), "lnxInvoicePayments must not error")
	assert.JSONEq(t, `{"action":"getInvoicesByUser","query":{}}`, <-bodies, "lnxInvoicePayments should send an empty query when nothing is set")
}

func TestGetLNXInvoices(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetLNXInvoices(t.Context(), 0)
	require.NoError(t, err, "GetLNXInvoices must not error")
	if mockTests {
		assert.Equal(t, []LNXInvoice{testLNXInvoice}, result, "GetLNXInvoices should decode every invoice")
	}

	result, err = newResponseServerExchange(t, `{"payments":[{}]}`).GetLNXInvoices(t.Context(), 0)
	require.NoError(t, err, "GetLNXInvoices must not error for a response without invoices")
	assert.Empty(t, result, "GetLNXInvoices should return no invoices from a response without them")
	_, err = newUnavailableServerExchange(t).GetLNXInvoices(t.Context(), 0)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLNXInvoices should return the request failure")
}

func TestGetLNXPayments(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	offset := testLNXPaymentsOffset
	if !mockTests {
		offset = 0
	}
	result, err := e.GetLNXPayments(t.Context(), offset)
	require.NoError(t, err, "GetLNXPayments must not error")
	if mockTests {
		assert.Equal(t, []LNXInvoice{testLNXInvoice}, result, "GetLNXPayments should decode every payment")
	}

	result, err = newResponseServerExchange(t, `{"invoices":[{}]}`).GetLNXPayments(t.Context(), offset)
	require.NoError(t, err, "GetLNXPayments must not error for a response without payments")
	assert.Empty(t, result, "GetLNXPayments should return no payments from a response without them")
	_, err = newUnavailableServerExchange(t).GetLNXPayments(t.Context(), offset)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLNXPayments should return the request failure")
}

func TestGetLNXInvoice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLNXInvoice(t.Context(), "")
	require.ErrorIs(t, err, errLNXPaymentHashRequired, "GetLNXInvoice must require a payment hash")

	skipLiveWithoutCredentials(t)
	paymentHash := testLNXPaymentHash
	if !mockTests {
		invoices, err := e.GetLNXInvoices(t.Context(), 0)
		require.NoError(t, err, "GetLNXInvoices must not error")
		if len(invoices) == 0 {
			t.Skip("The account has no Lightning Network invoice to get")
		}
		paymentHash = invoices[0].PaymentHash
	}
	result, err := e.GetLNXInvoice(t.Context(), paymentHash)
	require.NoError(t, err, "GetLNXInvoice must not error")
	if mockTests {
		assert.Equal(t, &testLNXInvoice, result, "GetLNXInvoice should decode every field")
	} else {
		assert.Equal(t, paymentHash, result.PaymentHash, "GetLNXInvoice should return the invoice with the payment hash")
	}

	_, err = newResponseServerExchange(t, `null`).GetLNXInvoice(t.Context(), paymentHash)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLNXInvoice should error for a null response")
	_, err = newUnavailableServerExchange(t).GetLNXInvoice(t.Context(), paymentHash)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLNXInvoice should return the request failure")
}

func TestGetLNXPayment(t *testing.T) {
	t.Parallel()
	_, err := e.GetLNXPayment(t.Context(), "")
	require.ErrorIs(t, err, errLNXPaymentHashRequired, "GetLNXPayment must require a payment hash")

	skipLiveWithoutCredentials(t)
	paymentHash := testLNXPaymentHash
	if !mockTests {
		payments, err := e.GetLNXPayments(t.Context(), 0)
		require.NoError(t, err, "GetLNXPayments must not error")
		if len(payments) == 0 {
			t.Skip("The account has no Lightning Network payment to get")
		}
		paymentHash = payments[0].PaymentHash
	}
	result, err := e.GetLNXPayment(t.Context(), paymentHash)
	require.NoError(t, err, "GetLNXPayment must not error")
	if mockTests {
		assert.Equal(t, &testLNXInvoice, result, "GetLNXPayment should decode every field")
	} else {
		assert.Equal(t, paymentHash, result.PaymentHash, "GetLNXPayment should return the payment with the payment hash")
	}

	_, err = newResponseServerExchange(t, `null`).GetLNXPayment(t.Context(), paymentHash)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLNXPayment should error for a null response")
	_, err = newUnavailableServerExchange(t).GetLNXPayment(t.Context(), paymentHash)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLNXPayment should return the request failure")
}

func TestRequestWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.RequestWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestWithdrawal must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *WithdrawalRequest
		err  error
	}{
		{"an old wallet name", &WithdrawalRequest{Wallet: "deposit", Method: "bitcoin", Amount: 1, Address: "a"}, errInvalidAccountWallet},
		{"no method", &WithdrawalRequest{Wallet: WalletTypeExchange, Amount: 1, Address: "a"}, errTransactionMethodRequired},
		{"no amount", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "bitcoin", Address: "a"}, errInvalidAmount},
		{"a negative amount", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "bitcoin", Amount: -1, Address: "a"}, errInvalidAmount},
		{"a negative Lightning amount", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: TransactionMethodLightning, Amount: -1, Invoice: "a"}, errInvalidAmount},
		{"no address", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "bitcoin", Amount: 1, Invoice: "a"}, errWithdrawalAddressRequired},
		{"no Lightning invoice", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "LNX", Amount: 1, Address: "a"}, errWithdrawalInvoiceRequired},
		{"a first name alone", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "bitcoin", Amount: 1, Address: "a", TravelRule: &TravelRuleDetails{FirstName: "John"}}, errTravelRuleNameIncomplete},
		{"a last name alone", &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "bitcoin", Amount: 1, Address: "a", TravelRule: &TravelRuleDetails{LastName: "Doe"}}, errTravelRuleNameIncomplete},
	} {
		_, err = e.RequestWithdrawal(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "RequestWithdrawal must refuse %s", tc.name)
	}

	// The recorded request holds the travel rule details; these hold the other optional fields
	ex, bodies := newRequestBodyExchange(t, "/v2/auth/w/withdraw", `[1568742390999,"acc_wd-req",null,null,[13080092,null,"lnx",null,"exchange",0.0001,null,null,0],null,"SUCCESS","Your withdrawal request has been successfully submitted."]`)
	_, err = ex.RequestWithdrawal(t.Context(), &WithdrawalRequest{Wallet: WalletTypeExchange, Method: "LNX", Invoice: "lnbc1", TravelRule: &TravelRuleDetails{VASPName: "Strawb Exchange Ltd", BeneficiarySelf: true}})
	require.NoError(t, err, "RequestWithdrawal must not error for a Lightning withdrawal")
	assert.JSONEq(t, `{"wallet":"exchange","method":"lnx","invoice":"lnbc1","travel_rule_tos":true,"vasp_name":"Strawb Exchange Ltd","beneficiary_self":true}`, <-bodies, "RequestWithdrawal should send a Lightning withdrawal's invoice and no amount or address")
	_, err = ex.RequestWithdrawal(t.Context(), &WithdrawalRequest{Wallet: WalletTypeFunding, Method: "ripple", Amount: 25.5, Address: "rAddress", PaymentID: "1234", TravelRule: &TravelRuleDetails{CorporateName: "Yotsuba Ltd"}})
	require.NoError(t, err, "RequestWithdrawal must not error for a withdrawal with a tag")
	assert.JSONEq(t, `{"wallet":"funding","method":"ripple","amount":"25.5","address":"rAddress","payment_id":"1234","travel_rule_tos":true,"dest_corp_name":"Yotsuba Ltd"}`, <-bodies, "RequestWithdrawal should send the payment ID and corporate name")
	_, err = newResponseServerExchange(t, `null`).RequestWithdrawal(t.Context(), testWithdrawalRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestWithdrawal should error for a null response")
	_, err = newUnavailableServerExchange(t).RequestWithdrawal(t.Context(), testWithdrawalRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestWithdrawal should return the request failure")

	if !mockTests {
		t.Skip("RequestWithdrawal moves funds irreversibly, so it is not run live")
	}
	result, err := e.RequestWithdrawal(t.Context(), testWithdrawalRequest())
	require.NoError(t, err, "RequestWithdrawal must not error")
	exp := &Notification[Withdrawal]{
		Time:   types.Time(time.UnixMilli(1568742390999)),
		Type:   "acc_wd-req",
		Data:   Withdrawal{ID: 13080092, Method: "ethereum", Wallet: WalletTypeExchange, Amount: 0.01, Fee: 0.00135},
		Status: NotificationStatusSuccess,
		Text:   "Your withdrawal request has been successfully submitted.",
	}
	assert.Equal(t, exp, result, "RequestWithdrawal should decode the withdrawal")
}

func TestGetMovements(t *testing.T) {
	t.Parallel()
	_, err := e.GetMovements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMovements must reject a nil request")
	_, err = e.GetMovements(t.Context(), &MovementsRequest{Start: time.UnixMilli(testMovementsEnd), End: time.UnixMilli(testMovementsStart)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMovements must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		_, err := e.GetMovements(t.Context(), &MovementsRequest{Currency: currency.BTC, Start: time.Now().AddDate(-1, 0, 0), End: time.Now(), Limit: 25})
		require.NoError(t, err, "GetMovements must not error")
		return
	}
	result, err := e.GetMovements(t.Context(), &MovementsRequest{Currency: currency.ETH, Start: time.UnixMilli(testMovementsStart), End: time.UnixMilli(testMovementsEnd), Limit: 25, Address: "DESTINATION_ADDRESS"})
	require.NoError(t, err, "GetMovements must not error")
	exp := []Movement{{
		ID:                 13105603,
		Currency:           currency.ETH,
		Method:             "ETHEREUM",
		StartedAt:          types.Time(time.UnixMilli(1569348774000)),
		UpdatedAt:          types.Time(time.UnixMilli(1569348774000)),
		Status:             "COMPLETED",
		Amount:             0.26300954,
		Fees:               -0.00135,
		DestinationAddress: "DESTINATION_ADDRESS",
		TransactionID:      "TRANSACTION_ID",
	}}
	assert.Equal(t, exp, result, "GetMovements should decode every field of a deposit")
	// Without a currency the path leaves it out, which the recorded request's path holds
	result, err = e.GetMovements(t.Context(), &MovementsRequest{IDs: []uint64{13293039}})
	require.NoError(t, err, "GetMovements must not error without a currency")
	exp = []Movement{{
		ID:                 13293039,
		Currency:           currency.ETH,
		Method:             "ETHEREUM",
		StartedAt:          types.Time(time.UnixMilli(1574175052000)),
		UpdatedAt:          types.Time(time.UnixMilli(1574181326000)),
		Status:             "CANCELED",
		Amount:             -0.24,
		Fees:               -0.00135,
		DestinationAddress: "DESTINATION_ADDRESS",
		TransactionID:      "TRANSACTION_ID",
		Note:               "Purchase of 100 pizzas",
	}}
	assert.Equal(t, exp, result, "GetMovements should decode every field of a withdrawal")

	_, err = newUnavailableServerExchange(t).GetMovements(t.Context(), &MovementsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMovements should return the request failure")
}

func TestGetMovementInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetMovementInfo(t.Context(), 0)
	require.ErrorIs(t, err, errMovementIDRequired, "GetMovementInfo must require a movement ID")

	skipLiveWithoutCredentials(t)
	var id uint64 = 24
	if !mockTests {
		movements, err := e.GetMovements(t.Context(), &MovementsRequest{Limit: 1})
		require.NoError(t, err, "GetMovements must not error")
		if len(movements) == 0 {
			t.Skip("The account has no deposit or withdrawal to get the info of")
		}
		id = movements[0].ID
	}
	result, err := e.GetMovementInfo(t.Context(), id)
	require.NoError(t, err, "GetMovementInfo must not error")
	if mockTests {
		exp := &MovementInfoResponse{
			Movement: Movement{
				ID:        24,
				Currency:  currency.EUR,
				Method:    "WIRE",
				StartedAt: types.Time(time.UnixMilli(1677086074000)),
				UpdatedAt: types.Time(time.UnixMilli(1677086210000)),
				Status:    "COMPLETED",
				Amount:    -29.5,
				Fees:      -0.5,
				Note:      "testing note",
			},
			Remark:                          "remark related to bank details",
			BankRouterID:                    123,
			ExternalBankMovementID:          "abcd-1234",
			ExternalBankMovementStatus:      "COMPLETED",
			ExternalBankMovementDescription: "finished withdrawal in platform",
			ExternalBankAccountInfo:         json.RawMessage(`{"router":"my-router","meta":{"foo":"bar"}}`),
		}
		assert.Equal(t, exp, result, "GetMovementInfo should decode every field")
	} else {
		assert.Equal(t, id, result.Movement.ID, "GetMovementInfo should return the movement with the ID")
	}

	_, err = newResponseServerExchange(t, `null`).GetMovementInfo(t.Context(), id)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMovementInfo should error for a null response")
	_, err = newUnavailableServerExchange(t).GetMovementInfo(t.Context(), id)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMovementInfo should return the request failure")
}

func TestGetAlertList(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAlertList(t.Context())
	require.NoError(t, err, "GetAlertList must not error")
	if mockTests {
		assert.Equal(t, []Alert{{Key: "price:tETHUSD:220", Type: alertTypePrice, Symbol: "tETHUSD", Price: 220, Countdown: 100}}, result, "GetAlertList should decode every alert")
	}

	_, err = newUnavailableServerExchange(t).GetAlertList(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAlertList should return the request failure")
}

func TestSetAlert(t *testing.T) {
	t.Parallel()
	_, err := e.SetAlert(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetAlert must reject a nil request")
	for _, symbol := range []string{"", "fUSD", "BTCUSD"} {
		_, err = e.SetAlert(t.Context(), &SetAlertRequest{Symbol: symbol, Price: 1})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "SetAlert must refuse symbol %q", symbol)
	}
	for _, price := range []float64{0, -1} {
		_, err = e.SetAlert(t.Context(), &SetAlertRequest{Symbol: "tBTCUSD", Price: price})
		require.ErrorIsf(t, err, errInvalidPrice, "SetAlert must refuse price %v", price)
	}

	skipLiveUnlessCanManipulate(t)
	req := &SetAlertRequest{Symbol: "tETHUSD", Price: 185, Count: 100}
	if !mockTests {
		// Priced far below the market, so it does not trigger before it is deleted
		req = &SetAlertRequest{Symbol: "tBTCUSD", Price: 1.5, Count: 1}
		t.Cleanup(func() {
			// The test's context is cancelled before its cleanups run
			_, err := e.DeleteAlert(context.WithoutCancel(t.Context()), &DeleteAlertRequest{Symbol: req.Symbol, Price: req.Price})
			assert.NoError(t, err, "DeleteAlert should delete the alert SetAlert set")
		})
	}
	result, err := e.SetAlert(t.Context(), req)
	require.NoError(t, err, "SetAlert must not error")
	if mockTests {
		assert.Equal(t, &Alert{Key: "price:tETHUSD:185", Type: alertTypePrice, Symbol: "tETHUSD", Price: 185, Countdown: 100}, result, "SetAlert should decode the alert")
	} else {
		assert.Equal(t, "price:tBTCUSD:1.5", result.Key, "SetAlert should key the alert by its symbol and price")
	}

	_, err = newResponseServerExchange(t, `null`).SetAlert(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetAlert should error for a null response")
	_, err = newUnavailableServerExchange(t).SetAlert(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetAlert should return the request failure")
}

func TestDeleteAlert(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteAlert(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "DeleteAlert must reject a nil request")
	_, err = e.DeleteAlert(t.Context(), &DeleteAlertRequest{Symbol: "fUSD", Price: 1})
	require.ErrorIs(t, err, errTradingSymbolRequired, "DeleteAlert must require a trading pair")
	_, err = e.DeleteAlert(t.Context(), &DeleteAlertRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errInvalidPrice, "DeleteAlert must require a price")

	skipLiveUnlessCanManipulate(t)
	req := &DeleteAlertRequest{Symbol: "tBTCUSD", Price: 600}
	if !mockTests {
		req = &DeleteAlertRequest{Symbol: "tBTCUSD", Price: 2.5}
		_, err := e.SetAlert(t.Context(), &SetAlertRequest{Symbol: req.Symbol, Price: req.Price, Count: 1})
		require.NoError(t, err, "SetAlert must set the alert DeleteAlert deletes")
	}
	result, err := e.DeleteAlert(t.Context(), req)
	require.NoError(t, err, "DeleteAlert must not error")
	assert.Equal(t, &DeleteAlertResponse{Deleted: true}, result, "DeleteAlert should report the alert deleted")

	_, err = newResponseServerExchange(t, `null`).DeleteAlert(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "DeleteAlert should error for a null response")
	_, err = newUnavailableServerExchange(t).DeleteAlert(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DeleteAlert should return the request failure")
}

func TestGetBalanceAvailable(t *testing.T) {
	t.Parallel()
	_, err := e.GetBalanceAvailable(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBalanceAvailable must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *BalanceAvailableRequest
		err  error
	}{
		{"no symbol", &BalanceAvailableRequest{Type: BalanceAvailableTypeFunding}, errSymbolRequired},
		{"an unknown type", &BalanceAvailableRequest{Symbol: "tBTCUSD", Type: "exchange", Direction: 1, Rate: 1}, errInvalidBalanceAvailableType},
		{"an exchange order without a direction", &BalanceAvailableRequest{Symbol: "tBTCUSD", Type: BalanceAvailableTypeExchange, Rate: 1}, errInvalidBalanceAvailableDirection},
		{"a margin order without a direction", &BalanceAvailableRequest{Symbol: "tBTCUSD", Type: BalanceAvailableTypeMargin, Rate: 1}, errInvalidBalanceAvailableDirection},
		{"an invalid direction", &BalanceAvailableRequest{Symbol: "tBTCF0:USTF0", Type: BalanceAvailableTypeDerivative, Direction: 2}, errInvalidBalanceAvailableDirection},
		{"an exchange order without a price", &BalanceAvailableRequest{Symbol: "tBTCUSD", Type: BalanceAvailableTypeExchange, Direction: -1}, errInvalidPrice},
		{"a negative price", &BalanceAvailableRequest{Symbol: "tBTCF0:USTF0", Type: BalanceAvailableTypeDerivative, Rate: -1}, errInvalidPrice},
	} {
		_, err = e.GetBalanceAvailable(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetBalanceAvailable must refuse %s", tc.name)
	}

	skipLiveWithoutCredentials(t)
	req := &BalanceAvailableRequest{Symbol: "tBTCUSD", Type: BalanceAvailableTypeExchange, Direction: 1, Rate: 800}
	result, err := e.GetBalanceAvailable(t.Context(), req)
	require.NoError(t, err, "GetBalanceAvailable must not error")
	if mockTests {
		assert.Equal(t, &BalanceAvailableResponse{Amount: 0.8056309}, result, "GetBalanceAvailable should decode the amount")
		result, err = e.GetBalanceAvailable(t.Context(), &BalanceAvailableRequest{Symbol: "tBTCF0:USTF0", Type: BalanceAvailableTypeDerivative, Direction: -1, Rate: 30000, Leverage: 10})
		require.NoError(t, err, "GetBalanceAvailable must not error with leverage")
		assert.Equal(t, &BalanceAvailableResponse{Amount: 0.8056309}, result, "GetBalanceAvailable should decode the amount with leverage")
	}

	_, err = newResponseServerExchange(t, `null`).GetBalanceAvailable(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBalanceAvailable should error for a null response")
	_, err = newUnavailableServerExchange(t).GetBalanceAvailable(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBalanceAvailable should return the request failure")
}

func TestWriteUserSettings(t *testing.T) {
	t.Parallel()
	_, err := e.WriteUserSettings(t.Context(), nil)
	require.ErrorIs(t, err, errUserSettingsRequired, "WriteUserSettings must require a setting")
	_, err = e.WriteUserSettings(t.Context(), map[string]json.RawMessage{"api:a": json.RawMessage(`1`), "key": json.RawMessage(`2`)})
	require.ErrorIs(t, err, errInvalidUserSettingKey, "WriteUserSettings must refuse a key without the api: prefix")

	skipLiveUnlessCanManipulate(t)
	settings := map[string]json.RawMessage{"api:key1": json.RawMessage(`"foo"`), "api:key2": json.RawMessage(`{"bar":2}`)}
	if !mockTests {
		settings = map[string]json.RawMessage{testUserSettingKey: json.RawMessage(`"gct"`)}
		t.Cleanup(func() {
			// The test's context is cancelled before its cleanups run
			_, err := e.DeleteUserSettings(context.WithoutCancel(t.Context()), []string{testUserSettingKey})
			assert.NoError(t, err, "DeleteUserSettings should delete the setting WriteUserSettings wrote")
		})
	}
	result, err := e.WriteUserSettings(t.Context(), settings)
	require.NoError(t, err, "WriteUserSettings must not error")
	if mockTests {
		exp := &Notification[UserSettingsCount]{Time: types.Time(time.UnixMilli(1574340932016)), Type: "acc_ss", Data: UserSettingsCount{Count: 2}, Status: NotificationStatusSuccess}
		assert.Equal(t, exp, result, "WriteUserSettings should decode the notification")
	} else {
		assert.Equal(t, uint64(1), result.Data.Count, "WriteUserSettings should write the setting")
	}

	_, err = newResponseServerExchange(t, `null`).WriteUserSettings(t.Context(), settings)
	assert.ErrorIs(t, err, common.ErrNoResponse, "WriteUserSettings should error for a null response")
	_, err = newUnavailableServerExchange(t).WriteUserSettings(t.Context(), settings)
	assert.ErrorIs(t, err, request.ErrBadStatus, "WriteUserSettings should return the request failure")
}

func TestGetUserSettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserSettings(t.Context(), nil)
	require.ErrorIs(t, err, errUserSettingsRequired, "GetUserSettings must require a key")
	_, err = e.GetUserSettings(t.Context(), []string{"api:a", "api:b c"})
	require.ErrorIs(t, err, errInvalidUserSettingKey, "GetUserSettings must refuse an invalid key")

	skipLiveWithoutCredentials(t)
	keys := []string{"api:bit"}
	if !mockTests {
		keys = []string{testUserSettingKey}
	}
	result, err := e.GetUserSettings(t.Context(), keys)
	require.NoError(t, err, "GetUserSettings must not error")
	if mockTests {
		assert.Equal(t, map[string]json.RawMessage{"bit": json.RawMessage(`"finex"`)}, result, "GetUserSettings should decode every setting")
	}

	result, err = newResponseServerExchange(t, `[["api:a",{"b":[1,2]}],["api:c",null]]`).GetUserSettings(t.Context(), keys)
	require.NoError(t, err, "GetUserSettings must not error")
	assert.Equal(t, map[string]json.RawMessage{"api:a": json.RawMessage(`{"b":[1,2]}`), "api:c": nil}, result, "GetUserSettings should keep each value as sent, and a null value unset")
	_, err = newUnavailableServerExchange(t).GetUserSettings(t.Context(), keys)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserSettings should return the request failure")
}

func TestDeleteUserSettings(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteUserSettings(t.Context(), []string{})
	require.ErrorIs(t, err, errUserSettingsRequired, "DeleteUserSettings must require a key")
	_, err = e.DeleteUserSettings(t.Context(), []string{"api:key1", "API:key2"})
	require.ErrorIs(t, err, errInvalidUserSettingKey, "DeleteUserSettings must refuse an invalid key")

	skipLiveUnlessCanManipulate(t)
	keys := []string{"api:key1", "api:key2"}
	if !mockTests {
		keys = []string{testUserSettingKey}
		_, err := e.WriteUserSettings(t.Context(), map[string]json.RawMessage{testUserSettingKey: json.RawMessage(`"gct"`)})
		require.NoError(t, err, "WriteUserSettings must write the setting DeleteUserSettings deletes")
	}
	result, err := e.DeleteUserSettings(t.Context(), keys)
	require.NoError(t, err, "DeleteUserSettings must not error")
	if mockTests {
		exp := &Notification[json.RawMessage]{Time: types.Time(time.UnixMilli(1617005634199)), Type: "acc_sd", Data: json.RawMessage(`[1]`), Status: NotificationStatusSuccess}
		assert.Equal(t, exp, result, "DeleteUserSettings should decode the notification")
	} else {
		assert.Equal(t, NotificationStatusSuccess, result.Status, "DeleteUserSettings should delete the setting")
	}

	_, err = newResponseServerExchange(t, `null`).DeleteUserSettings(t.Context(), keys)
	assert.ErrorIs(t, err, common.ErrNoResponse, "DeleteUserSettings should error for a null response")
	_, err = newUnavailableServerExchange(t).DeleteUserSettings(t.Context(), keys)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DeleteUserSettings should return the request failure")
}

func TestCheckAccountWallet(t *testing.T) {
	t.Parallel()
	for _, wallet := range []string{WalletTypeExchange, WalletTypeMargin, WalletTypeFunding} {
		assert.NoErrorf(t, checkAccountWallet(wallet), "checkAccountWallet should accept wallet %q", wallet)
	}
	for _, wallet := range []string{"", "trading", "deposit", "EXCHANGE", "spot"} {
		assert.ErrorIsf(t, checkAccountWallet(wallet), errInvalidAccountWallet, "checkAccountWallet should refuse wallet %q", wallet)
	}
}

func TestCheckUserSettingKeys(t *testing.T) {
	t.Parallel()
	valid := []string{"api:", "api:key1", "api:A-z_0-9", "api:---"}
	assert.NoError(t, checkUserSettingKeys(slices.Values(valid)), "checkUserSettingKeys should accept keys matching the pattern")
	assert.NoError(t, checkUserSettingKeys(maps.Keys(map[string]int{"api:x": 1})), "checkUserSettingKeys should accept a map's keys")
	for _, key := range []string{"", "key1", "api", "API:key1", "api:key 1", "api:key.1", "api:kéy", "api:key:1", " api:key1", "xapi:key1"} {
		assert.ErrorIsf(t, checkUserSettingKeys(slices.Values([]string{"api:ok", key})), errInvalidUserSettingKey, "checkUserSettingKeys should refuse key %q", key)
	}
}

func TestTwoFactorTokenValidate(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&TwoFactorToken{Method: TwoFactorMethodOneTimePassword, Token: "123456"}).validate(), "validate should accept a one time password")
	assert.NoError(t, (&TwoFactorToken{Method: TwoFactorMethodU2F, Response: &WebAuthnAssertion{ID: "id"}}).validate(), "validate should accept an assertion")
	assert.ErrorIs(t, (&TwoFactorToken{Method: TwoFactorMethodOneTimePassword}).validate(), errTwoFactorPasswordRequired, "validate should require the one time password")
	assert.ErrorIs(t, (&TwoFactorToken{Method: TwoFactorMethodU2F, Token: "123456"}).validate(), errTwoFactorAssertionRequired, "validate should require the assertion")
	for _, method := range []string{"", "OTP", "sms"} {
		assert.ErrorIsf(t, (&TwoFactorToken{Method: method, Token: "123456"}).validate(), errInvalidTwoFactorMethod, "validate should refuse method %q", method)
	}
}

func TestLoginExtraInfoUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]LoginExtraInfo{
		`"{\"geo\": \"Cancún, MX\"}"`: LoginExtraInfo(`{"geo": "Cancún, MX"}`),
		`" {\"a\":1} "`:               LoginExtraInfo(`{"a":1}`),
		`{"geo":"Cancún, MX"}`:        LoginExtraInfo(`{"geo":"Cancún, MX"}`),
		`"text"`:                      LoginExtraInfo(`"text"`),
		`"123"`:                       LoginExtraInfo(`"123"`),
		`"{not json}"`:                LoginExtraInfo(`"{not json}"`),
		`""`:                          LoginExtraInfo(`""`),
		`null`:                        LoginExtraInfo(`null`),
	} {
		var got LoginExtraInfo
		require.NoErrorf(t, got.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for %s", data)
		assert.Equalf(t, exp, got, "UnmarshalJSON should decode %s", data)
	}
	var got LoginExtraInfo
	assert.Error(t, got.UnmarshalJSON([]byte(`"\x"`)), "UnmarshalJSON should error for a malformed string")
}

func TestLoginExtraInfoMarshalJSON(t *testing.T) {
	t.Parallel()
	out, err := json.Marshal(struct{ ExtraInfo LoginExtraInfo }{LoginExtraInfo(`{"geo":"Cancún, MX"}`)})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"ExtraInfo":{"geo":"Cancún, MX"}}`, string(out), "Marshal should encode the extra info as the object it holds")
	out, err = LoginExtraInfo(nil).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, "null", string(out), "MarshalJSON should encode unset info as null")
}

// TestAccountActionsRecordsEveryPosition decodes each account actions record from an array with a distinct value at
// every tagged position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestAccountActionsRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	lastLogin, err := time.Parse(time.RFC3339, "2023-11-14T22:13:20.044Z")
	require.NoError(t, err, "Parse must not error")
	testTupleDecode(t, `[1000,"s1","s2",1700000000003,1,1005,null,"s7","s8","s9",1,null,{"k":12},null,1700000000014,1015,1016,1,1,1,null,1,1,1,null,null,["s26"],null,1,1,1,1,1,null,null,null,null,null,1,1,null,null,1,null,"2023-11-14T22:13:20.044Z",null,null,1047,null,["s49"],["s50"],"s51",null,1,1]`,
		UserInfoResponse{ID: 1000, Email: "s1", Username: "s2", CreatedAt: types.Time(time.UnixMilli(1700000000003)), Verified: true, VerificationLevel: 1005, Timezone: "s7", Locale: "s8", Company: "s9", EmailVerified: true, SubAccountType: json.RawMessage(`{"k":12}`), MasterAccountCreatedAt: types.Time(time.UnixMilli(1700000000014)), GroupID: 1015, MasterAccountID: 1016, InheritsMasterAccountVerification: true, IsGroupMaster: true, GroupWithdrawEnabled: true, PaperTradingEnabled: true, MerchantEnabled: true, CompetitionEnabled: true, TwoFactorModes: []string{"s26"}, IsSecuritiesMaster: true, SecuritiesEnabled: true, IsSecuritiesInvestorAccredited: true, IsSecuritiesElSalvador: true, IsSecuritiesAIFC: true, AllowDisableContextSwitch: true, ContextSwitchDisabled: true, HighNetWorth: true, LastLoginAt: lastLogin, VerificationLevelSubmitted: 1047, ComplianceCountries: []string{"s49"}, ComplianceResidenceCountries: []string{"s50"}, ComplianceAccountType: "s51", PermittedBeneficiary: true, IsMerchantEnterprise: true})
	testTupleDecode(t, `[null,null,null,null,[[0.5,1.5,2.5,null,null,5.5],[10.5,11.5,12.5,null,null,15.5]],[[{"curr":"s0","vol":0.25,"vol_safe":1.25,"vol_maker":2.25,"vol_BFX":3.25,"vol_BFX_safe":4.25,"vol_BFX_maker":5.25}],{"BTC":1.5},2.5],[null,{"ETH":1.75},2.75],null,null,{"leo_lev":1009,"leo_amount_avg":9.5}]`,
		AccountSummaryResponse{
			FeeRates: SummaryFeeRates{
				Maker: SummaryMakerFeeRates{ToCrypto: 0.5, ToStable: 1.5, ToFiat: 2.5, DerivativeRebate: 5.5},
				Taker: SummaryTakerFeeRates{ToCrypto: 10.5, ToStable: 11.5, ToFiat: 12.5, Derivative: 15.5},
			},
			TradingVolumeAndFees: SummaryTradingVolumeAndFees{
				Volumes:      []SummaryTradeVolume{{Currency: "s0", Volume: 0.25, VolumeSafe: 1.25, VolumeMaker: 2.25, VolumeBFX: 3.25, VolumeBFXSafe: 4.25, VolumeBFXMaker: 5.25}},
				Fees:         map[currency.Code]float64{currency.BTC: 1.5},
				TotalFeesUSD: 2.5,
			},
			FundingEarnings: SummaryFundingEarnings{Earnings: map[currency.Code]float64{currency.ETH: 1.75}, TotalEarningsUSD: 2.75},
			LEOInfo:         SummaryLEOInfo{Level: 1009, AverageAmount: 9.5},
		})
	testTupleDecode(t, `[[0.5,1.5,2.5,null,null,5.5],[10.5,11.5,12.5,null,null,15.5]]`,
		SummaryFeeRates{Maker: SummaryMakerFeeRates{ToCrypto: 0.5, ToStable: 1.5, ToFiat: 2.5, DerivativeRebate: 5.5}, Taker: SummaryTakerFeeRates{ToCrypto: 10.5, ToStable: 11.5, ToFiat: 12.5, Derivative: 15.5}})
	testTupleDecode(t, `[0.5,1.5,2.5,null,null,5.5]`,
		SummaryMakerFeeRates{ToCrypto: 0.5, ToStable: 1.5, ToFiat: 2.5, DerivativeRebate: 5.5})
	testTupleDecode(t, `[0.5,1.5,2.5,null,null,5.5]`,
		SummaryTakerFeeRates{ToCrypto: 0.5, ToStable: 1.5, ToFiat: 2.5, Derivative: 5.5})
	testTupleDecode(t, `[[{"curr":"s0","vol":0.25}],{"BTC":1.5},2.5]`,
		SummaryTradingVolumeAndFees{Volumes: []SummaryTradeVolume{{Currency: "s0", Volume: 0.25}}, Fees: map[currency.Code]float64{currency.BTC: 1.5}, TotalFeesUSD: 2.5})
	testTupleDecode(t, `[null,{"ETH":1.75},2.75]`,
		SummaryFundingEarnings{Earnings: map[currency.Code]float64{currency.ETH: 1.75}, TotalEarningsUSD: 2.75})
	testTupleDecode(t, `[1000,null,1700000000002,null,"s4",null,null,{"k":7}]`,
		Login{ID: 1000, Time: types.Time(time.UnixMilli(1700000000002)), IP: "s4", ExtraInfo: LoginExtraInfo(`{"k":7}`)})
	testTupleDecode(t, `["s0",1,1]`,
		KeyPermission{Scope: "s0", Read: true, Write: true})
	testTupleDecode(t, `["s0"]`,
		GenerateTokenResponse{Token: "s0"})
	testTupleDecode(t, `[1700000000000,null,"s2",null,null,"s5","s6"]`,
		ChangelogEntry{CreatedAt: types.Time(time.UnixMilli(1700000000000)), Log: "s2", IP: "s5", UserAgent: "s6"})
	testTupleDecode(t, `[1700000000000,"s1","s2",null,"UST","BTC",null,7.5]`,
		WalletTransfer{UpdatedAt: types.Time(time.UnixMilli(1700000000000)), WalletFrom: "s1", WalletTo: "s2", Currency: currency.UST, CurrencyTo: currency.BTC, Amount: 7.5})
	testTupleDecode(t, `[null,"s1","UST",null,"s4","s5"]`,
		DepositAddress{Method: "s1", Currency: currency.UST, Address: "s4", PoolAddress: "s5"})
	testTupleDecode(t, `["s0","s1"]`,
		WalletDepositAddress{Address: "s0", WalletType: "s1"})
	testTupleDecode(t, `["s0","s1",null,null,"4.5"]`,
		GenerateInvoiceResponse{InvoiceHash: "s0", Invoice: "s1", Amount: 4.5})
	testTupleDecode(t, `[1000,null,"s2","s3","s4",5.5,null,null,8.5]`,
		Withdrawal{ID: 1000, Method: "s2", PaymentID: "s3", Wallet: "s4", Amount: 5.5, Fee: 8.5})
	testTupleDecode(t, `[1000,"UST","s2",null,null,1700000000005,1700000000006,null,null,"s9",null,null,"12.5",13.5,null,null,"s16","s17",null,null,"s20","s21"]`,
		Movement{ID: 1000, Currency: currency.UST, Method: "s2", StartedAt: types.Time(time.UnixMilli(1700000000005)), UpdatedAt: types.Time(time.UnixMilli(1700000000006)), Status: "s9", Amount: 12.5, Fees: 13.5, DestinationAddress: "s16", PaymentID: "s17", TransactionID: "s20", Note: "s21"})
	testTupleDecode(t, `[1000,"UST","s2",null,"s4",1700000000005,1700000000006,null,null,"s9",null,null,12.5,"13.5",null,null,"s16","s17",null,null,"s20","s21",null,null,"24.5",1025,null,null,"s28","s29","s30",{"k":31}]`,
		MovementInfoResponse{
			Movement:                        Movement{ID: 1000, Currency: currency.UST, Method: "s2", StartedAt: types.Time(time.UnixMilli(1700000000005)), UpdatedAt: types.Time(time.UnixMilli(1700000000006)), Status: "s9", Amount: 12.5, Fees: 13.5, DestinationAddress: "s16", PaymentID: "s17", TransactionID: "s20", Note: "s21"},
			Remark:                          "s4",
			BankFees:                        24.5,
			BankRouterID:                    1025,
			ExternalBankMovementID:          "s28",
			ExternalBankMovementStatus:      "s29",
			ExternalBankMovementDescription: "s30",
			ExternalBankAccountInfo:         json.RawMessage(`{"k":31}`),
		})
	testTupleDecode(t, `["s0","s1","s2",3.5,1004]`,
		Alert{Key: "s0", Type: "s1", Symbol: "s2", Price: 3.5, Countdown: 1004})
	testTupleDecode(t, `[true]`,
		DeleteAlertResponse{Deleted: true})
	testTupleDecode(t, `[0.5]`,
		BalanceAvailableResponse{Amount: 0.5})
	testTupleDecode(t, `[1000]`,
		UserSettingsCount{Count: 1000})
	testTupleDecode(t, `["s0",{"k":1}]`,
		userSetting{Key: "s0", Value: json.RawMessage(`{"k":1}`)})
}
