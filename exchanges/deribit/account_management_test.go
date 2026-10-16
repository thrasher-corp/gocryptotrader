package deribit

import (
	"context"
	"io"
	"net/http"
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

const (
	// mockAPIKeyID is the API key the mock API key requests act on; mockAPIKeyName and mockEditedAPIKeyName are the
	// names the mock rename and edit give it
	mockAPIKeyID         = 3
	mockAPIKeyName       = "KeyName3"
	mockEditedAPIKeyName = "NewKeyName"
	// mockCreatedAPIKeyName and mockCreatedAPIKeyPublicKey create the mock asymmetric API key
	mockCreatedAPIKeyName      = "Public key 1"
	mockCreatedAPIKeyPublicKey = "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEAM7FWhKquNqLmTOV4hfYT5r3AjrYiORTT6Tn5HIfFNV8=\n-----END PUBLIC KEY-----"
	// mockAccountUserID is the account whose margin model the mock dry run changes
	mockAccountUserID = 3
	// mockAccountSubaccountID is the subaccount the mock subaccount requests act on, with mockSubaccountName and
	// mockSubaccountEmail
	mockAccountSubaccountID = 7
	mockSubaccountName      = "new_user_1_1"
	mockSubaccountEmail     = "user_1_1@email.com"
	// mockMemberID is the Direct Access member the mock member requests act on, which may trade on
	// mockMemberAccountID
	mockMemberID        = 195
	mockMemberName      = "Member1"
	mockMemberAccountID = 85867
	// mockAnnouncementID is the announcement the mock request marks read
	mockAnnouncementID = 1550058362418
	// mockAnnouncementsStartTime and mockAnnouncementsCount select the announcement the announcements request returns,
	// which is recorded live and still returned live
	mockAnnouncementsStartTime = 1791800000000
	mockAnnouncementsCount     = 1
	// mockAccessLogCount is the number of access log entries the access log requests ask for
	mockAccessLogCount = 3
	// mockPositionInstrument is the instrument of the position requests and simulated positions
	mockPositionInstrument = "BTC-PERPETUAL"
	// mockTransactionLogStartTime and mockTransactionLogEndTime, in Unix milliseconds, bound the mock transaction log
	// request, which asks for mockTransactionLogCount entries
	mockTransactionLogStartTime = 1613657734000
	mockTransactionLogEndTime   = 1613660407000
	mockTransactionLogCount     = 5
	// mockEmailLanguage is the language the mock request sets
	mockEmailLanguage = "en"
)

// testAPIKeyScopes returns the maximum scope the mock API key requests set
func testAPIKeyScopes() APIKeyScopes {
	return APIKeyScopes{"account:read_write", "wallet:read_write", "block_trade:read", "trade:read_write"}
}

// testCreateAPIKeyRequest returns the mock request creating an asymmetric API key
func testCreateAPIKeyRequest() *CreateAPIKeyRequest {
	return &CreateAPIKeyRequest{
		MaxScope:  APIKeyScopes{"account:read", "trade:read", "block_trade:read_write", "wallet:none"},
		Name:      mockCreatedAPIKeyName,
		PublicKey: mockCreatedAPIKeyPublicKey,
	}
}

// testSetMemberRequest returns the mock request editing a Direct Access member
func testSetMemberRequest() *SetMemberRequest {
	active := true
	return &SetMemberRequest{ID: mockMemberID, Name: mockMemberName, IsActive: &active, Accounts: []uint64{mockMemberAccountID}}
}

// testSetSelfTradingConfigRequest returns the mock request configuring self trading prevention
func testSetSelfTradingConfigRequest() *SetSelfTradingConfigRequest {
	prevention := true
	return &SetSelfTradingConfigRequest{Mode: SelfTradingModeCancelMaker, ExtendedToSubaccounts: true, BlockRFQSelfMatchPrevention: &prevention}
}

// testSimulatePortfolioRequest returns the request simulating a perpetual position added to the portfolio
func testSimulatePortfolioRequest() *SimulatePortfolioRequest {
	addPositions := true
	return &SimulatePortfolioRequest{Currency: currency.BTC, AddPositions: &addPositions, SimulatedPositions: map[string]float64{mockPositionInstrument: 1}}
}

// liveTestAPIKeyRequest returns the request creating the API keys live tests act on, with the narrowest scope
func liveTestAPIKeyRequest() *CreateAPIKeyRequest {
	return &CreateAPIKeyRequest{MaxScope: APIKeyScopes{"account:read"}, Name: "gct_test"}
}

// liveTestAPIKey creates an API key for a live test to act on, and removes it once the test ends
func liveTestAPIKey(t *testing.T) *APIKey {
	t.Helper()
	key, err := e.CreateAPIKey(t.Context(), liveTestAPIKeyRequest())
	require.NoError(t, err, "CreateAPIKey must create an API key to test with")
	t.Cleanup(func() {
		assert.NoError(t, e.RemoveAPIKey(context.WithoutCancel(t.Context()), key.ID), "RemoveAPIKey should remove the test's API key")
	})
	return key
}

// testAPIKeyID returns the mock API key or, live, an API key created for the test
func testAPIKeyID(t *testing.T) uint64 {
	t.Helper()
	if mockTests {
		return mockAPIKeyID
	}
	return liveTestAPIKey(t).ID
}

// liveTestSubaccount returns a subaccount of the live account, skipping the test when it has none
func liveTestSubaccount(t *testing.T) *Subaccount {
	t.Helper()
	accounts, err := e.GetSubaccounts(t.Context(), false)
	require.NoError(t, err, "GetSubaccounts must not error")
	for i := range accounts {
		if accounts[i].Type == "subaccount" {
			return &accounts[i]
		}
	}
	t.Skip("the live account has no subaccount to test with")
	return nil
}

// testChangeAPIKeyNameRequest returns the mock rename or, live, a rename of an API key created for the test
func testChangeAPIKeyNameRequest(t *testing.T) *ChangeAPIKeyNameRequest {
	t.Helper()
	if mockTests {
		return &ChangeAPIKeyNameRequest{ID: mockAPIKeyID, Name: mockAPIKeyName}
	}
	return &ChangeAPIKeyNameRequest{ID: liveTestAPIKey(t).ID, Name: "gct_renamed"}
}

// testChangeMarginModelRequest returns the mock dry run or, live, a dry run of the account's margin model change,
// which changes nothing
func testChangeMarginModelRequest() *ChangeMarginModelRequest {
	if mockTests {
		return &ChangeMarginModelRequest{UserID: mockAccountUserID, MarginModel: MarginModelCrossPortfolio, DryRun: true}
	}
	return &ChangeMarginModelRequest{MarginModel: MarginModelCrossStandard, DryRun: true}
}

// testChangeScopeInAPIKeyRequest returns the mock scope change or, live, a scope change of an API key created for the
// test
func testChangeScopeInAPIKeyRequest(t *testing.T) *ChangeScopeInAPIKeyRequest {
	t.Helper()
	if mockTests {
		return &ChangeScopeInAPIKeyRequest{ID: mockAPIKeyID, MaxScope: testAPIKeyScopes()}
	}
	return &ChangeScopeInAPIKeyRequest{ID: liveTestAPIKey(t).ID, MaxScope: APIKeyScopes{"account:read", "trade:read"}}
}

// testChangeSubaccountNameRequest returns the mock rename or, live, a rename of a subaccount to its own name, which
// changes nothing and leaves concurrent tests unaffected
func testChangeSubaccountNameRequest(t *testing.T) *ChangeSubaccountNameRequest {
	t.Helper()
	if mockTests {
		return &ChangeSubaccountNameRequest{SubaccountID: mockAccountSubaccountID, Name: mockSubaccountName}
	}
	subaccount := liveTestSubaccount(t)
	return &ChangeSubaccountNameRequest{SubaccountID: subaccount.ID, Name: subaccount.Username}
}

// testEditAPIKeyRequest returns the mock edit or, live, an edit of an API key created for the test
func testEditAPIKeyRequest(t *testing.T) *EditAPIKeyRequest {
	t.Helper()
	if mockTests {
		return &EditAPIKeyRequest{ID: mockAPIKeyID, MaxScope: testAPIKeyScopes(), Name: mockEditedAPIKeyName}
	}
	return &EditAPIKeyRequest{ID: liveTestAPIKey(t).ID, MaxScope: APIKeyScopes{"account:read"}, Name: "gct_edited"}
}

// testTransactionLogRequest returns the mock transaction log request or, live, one for the past week
func testTransactionLogRequest() *GetTransactionLogRequest {
	if mockTests {
		return &GetTransactionLogRequest{
			Currency:  currency.BTC,
			StartTime: time.UnixMilli(mockTransactionLogStartTime),
			EndTime:   time.UnixMilli(mockTransactionLogEndTime),
			Count:     mockTransactionLogCount,
		}
	}
	return &GetTransactionLogRequest{Currency: currency.BTC, StartTime: time.Now().Add(-7 * 24 * time.Hour), EndTime: time.Now(), Count: mockTransactionLogCount}
}

// testEmailLanguage returns the mock language or, live, the account's language, which setting changes nothing
func testEmailLanguage(t *testing.T) string {
	t.Helper()
	if mockTests {
		return mockEmailLanguage
	}
	language, err := e.GetEmailLanguage(t.Context())
	require.NoError(t, err, "GetEmailLanguage must not error")
	return language
}

// testSetMemberRequestFor returns the mock member edit or, live, an edit of a member to its own settings, which
// changes nothing, skipping the test without Direct Access or a member
func testSetMemberRequestFor(t *testing.T) *SetMemberRequest {
	t.Helper()
	if mockTests {
		return testSetMemberRequest()
	}
	members, err := e.GetMembers(t.Context())
	require.NoError(t, err, "GetMembers must not error")
	if !members.IsDirectAccessAllowed || len(members.Members) == 0 {
		t.Skip("the live account has no Direct Access member to test with")
	}
	member := members.Members[0]
	return &SetMemberRequest{ID: member.ID, Name: member.Name, IsActive: &member.IsActive, Accounts: member.Accounts}
}

// testSetSelfTradingConfigRequestFor returns the mock configuration or, live, the account's configuration, which
// setting changes nothing
func testSetSelfTradingConfigRequestFor(t *testing.T) *SetSelfTradingConfigRequest {
	t.Helper()
	if mockTests {
		return testSetSelfTradingConfigRequest()
	}
	summary, err := e.GetAccountSummary(t.Context(), &GetAccountSummaryRequest{Currency: currency.BTC, Extended: true})
	require.NoError(t, err, "GetAccountSummary must not error")
	return &SetSelfTradingConfigRequest{
		Mode:                        summary.SelfTradingRejectMode,
		ExtendedToSubaccounts:       summary.SelfTradingExtendedToSubaccounts.Bool(),
		BlockRFQSelfMatchPrevention: &summary.BlockRFQSelfMatchPrevention,
	}
}

// testToggleSubaccountLoginRequest returns the mock request or, live, a request setting a subaccount's login to its
// current state, which changes nothing
func testToggleSubaccountLoginRequest(t *testing.T) *ToggleSubaccountLoginRequest {
	t.Helper()
	if mockTests {
		return &ToggleSubaccountLoginRequest{SubaccountID: mockAccountSubaccountID, State: SubaccountLoginEnable}
	}
	subaccount := liveTestSubaccount(t)
	state := SubaccountLoginDisable
	if subaccount.LoginEnabled {
		state = SubaccountLoginEnable
	}
	return &ToggleSubaccountLoginRequest{SubaccountID: subaccount.ID, State: state}
}

// testAPIKey returns the documented API key the mock API key requests return; the mock recording blanks its name
func testAPIKey() *APIKey {
	return &APIKey{
		ID:           mockAPIKeyID,
		Timestamp:    types.Time(time.UnixMilli(1560242482758)),
		ClientID:     "1sXMQBhM",
		ClientSecret: "B6RsF9rrLY5ezEGBQkyLlV-UC7whyPJ34BMA-kKYpes",
		MaxScope:     APIKeyScopes{"account:read_write", "block_trade:read", "trade:read_write", "wallet:read_write"},
		Enabled:      true,
	}
}

// testMarginModelChanges returns the documented margin model changes the mock dry run returns
func testMarginModelChanges() []MarginModelChange {
	return []MarginModelChange{
		{Currency: currency.NewCode("eth")},
		{
			Currency: currency.NewCode("btc"),
			OldState: MarginModelState{MaintenanceMarginRate: 0.02862727, InitialMarginRate: 0.45407615, AvailableBalance: 0.553590509},
			NewState: MarginModelState{MaintenanceMarginRate: 0.02710204, InitialMarginRate: 0.03252245, AvailableBalance: 0.98106428},
		},
	}
}

// testCreatedSubaccount returns the documented subaccount the mock creation returns; the mock recording blanks its
// username
func testCreatedSubaccount() *Subaccount {
	return &Subaccount{
		ID:         13,
		Type:       "subaccount",
		SystemName: "user_1_4",
		Email:      "user_AAA@email.com",
		Portfolio: map[string]SubaccountPortfolio{
			"btc": {Currency: currency.NewCode("btc")},
			"eth": {Currency: currency.NewCode("eth")},
		},
	}
}

// testMembers returns the documented members the mock member requests return; the mock recording blanks their names
func testMembers() *MembersResponse {
	return &MembersResponse{
		Members: []Member{{
			ID:                        mockMemberID,
			MainUserID:                57874,
			IsActive:                  true,
			Accounts:                  []uint64{mockMemberAccountID},
			LastModificationTimestamp: types.Time(time.UnixMilli(1690000000000)),
		}},
		IsDirectAccessAllowed: true,
	}
}

// testAccessLog returns the documented access log page the mock request returns
func testAccessLog() *AccessLogResponse {
	return &AccessLogResponse{
		RecordsTotal: 34,
		Entries: []AccessLogEntry{
			{ID: 45, Timestamp: types.Time(time.UnixMilli(1575876682576)), IP: "127.0.0.1", Country: "Local Country", City: "Local Town", Result: "success"},
			{ID: 44, Timestamp: types.Time(time.UnixMilli(1575876459309)), IP: "127.0.0.1", Country: "Local Country", City: "Local Town", Result: "success"},
			{ID: 43, Timestamp: types.Time(time.UnixMilli(1575546252774)), IP: "127.0.0.1", Country: "Local Country", City: "Local Town", Result: "disabled_tfa"},
		},
	}
}

// testAccountLimits returns the documented rate limits every mock account summary returns
func testAccountLimits() AccountLimits {
	return AccountLimits{
		NonMatchingEngine: AccountRateLimit{Rate: 1000, Burst: 1500},
		MatchingEngine: MatchingEngineLimits{
			Spot:      AccountRateLimit{Rate: 200, Burst: 250},
			CancelAll: AccountRateLimit{Rate: 200, Burst: 250},
			MatchingEngineTradingLimits: MatchingEngineTradingLimits{
				Trading:              map[string]AccountRateLimit{"total": {Rate: 200, Burst: 250}},
				MaximumQuotes:        AccountRateLimit{Rate: 500, Burst: 500},
				MaximumMassQuotes:    AccountRateLimit{Rate: 10, Burst: 10},
				GuaranteedMassQuotes: AccountRateLimit{Rate: 2, Burst: 2},
			},
		},
	}
}

// testAccountDetails returns the documented account details the mock extended summaries return; the mock recording
// blanks the username
func testAccountDetails() AccountDetails {
	return AccountDetails{
		ID:                          10,
		Type:                        "main",
		SystemName:                  "user",
		Email:                       "user@example.com",
		CreationTimestamp:           types.Time(time.UnixMilli(1687352432143)),
		SelfTradingRejectMode:       SelfTradingModeCancelMaker,
		BlockRFQSelfMatchPrevention: true,
	}
}

// testBTCAccountSummary returns the documented BTC summary the mock account summary request returns
func testBTCAccountSummary() AccountSummary {
	return AccountSummary{
		Currency:                           currency.BTC,
		Balance:                            302.60065765,
		Equity:                             302.61869214,
		MarginBalance:                      302.62729214,
		AvailableFunds:                     301.38059622,
		AvailableWithdrawalFunds:           301.35396172,
		InitialMargin:                      1.24669592,
		MaintenanceMargin:                  0.8857841,
		ProjectedInitialMargin:             1.01529592,
		ProjectedMaintenanceMargin:         0.7543841,
		TotalProfitLoss:                    -0.33084225,
		SessionRealisedProfitLoss:          -0.03258105,
		SessionUnrealisedProfitLoss:        0.05271555,
		FuturesProfitLoss:                  -0.32434225,
		FuturesSessionRealisedProfitLoss:   -0.03258105,
		FuturesSessionUnrealisedProfitLoss: 0.05921555,
		OptionsProfitLoss:                  -0.0065,
		OptionsSessionUnrealisedProfitLoss: -0.0065,
		OptionsValue:                       -0.0086,
		OptionsDelta:                       -1.01962,
		OptionsGamma:                       0.00001,
		OptionsTheta:                       15.97071,
		OptionsVega:                        0.0858,
		OptionsGammaByIndex:                map[string]float64{"btc_usd": 0.00001},
		OptionsThetaByIndex:                map[string]float64{"btc_usd": 15.97071},
		OptionsVegaByIndex:                 map[string]float64{"btc_usd": 0.0858},
		DeltaTotal:                         31.602958,
		ProjectedDeltaTotal:                32.613978,
		DeltaTotalByIndex:                  map[string]float64{"btc_usd": 31.594357699},
		MarginModel:                        MarginModelSegregatedStandard,
		Limits:                             testAccountLimits(),
		Fees: map[string]map[string]AccountFee{"btc_usd": {
			"future":    {Default: AccountFeeRates{Type: "fixed", Maker: -0.0001, Taker: 0.00035000000000000005}, BlockTrade: 0.3},
			"option":    {Default: AccountFeeRates{Type: "relative", Maker: 0.625, Taker: 0.625}, BlockTrade: 0.625},
			"perpetual": {Default: AccountFeeRates{Type: "fixed", Maker: -0.0001, Taker: 0.00035000000000000005}, BlockTrade: 0.3},
		}},
	}
}

// testAccountSummaries returns the documented summaries the mock account summaries request returns; its BTC summary
// leaves out the options' theta map
func testAccountSummaries() *AccountSummariesResponse {
	btc := testBTCAccountSummary()
	btc.OptionsThetaByIndex = nil
	return &AccountSummariesResponse{
		AccountDetails: testAccountDetails(),
		Summaries: []AccountSummary{
			btc,
			{
				Currency:                 currency.ETH,
				Balance:                  100,
				Equity:                   100,
				MarginBalance:            100,
				AvailableFunds:           99.999598,
				AvailableWithdrawalFunds: 99.999597,
				SpotReserve:              0.0002,
				InitialMargin:            0.000402,
				ProjectedInitialMargin:   0.0002,
				OptionsGammaByIndex:      map[string]float64{},
				OptionsThetaByIndex:      map[string]float64{},
				OptionsVegaByIndex:       map[string]float64{},
				DeltaTotalByIndex:        map[string]float64{"eth_usd": 0},
				MarginModel:              MarginModelSegregatedStandard,
				Limits:                   testAccountLimits(),
				Fees: map[string]map[string]AccountFee{"eth_usd": {
					"future":    {Default: AccountFeeRates{Type: "fixed", Maker: -0.00005, Taker: 0.00025}, BlockTrade: 0.2},
					"option":    {Default: AccountFeeRates{Type: "relative", Maker: 0.5, Taker: 0.5}, BlockTrade: 0.5},
					"perpetual": {Default: AccountFeeRates{Type: "fixed", Maker: -0.00005, Taker: 0.00025}, BlockTrade: 0.2},
				}},
			},
		},
	}
}

// testAffiliateProgramInfo returns the documented affiliate program information the mock request returns
func testAffiliateProgramInfo() *AffiliateProgramInfoResponse {
	return &AffiliateProgramInfoResponse{
		IsEnabled:          true,
		NumberOfAffiliates: 1,
		Link:               "https://www.deribit.com/reg-xxx.zxyq",
		Received:           AffiliateProgramPayouts{BTC: 0.000001, ETH: 0.00004},
	}
}

// testNewAnnouncements returns the documented announcements the mock unread announcements request returns
func testNewAnnouncements() []Announcement {
	return []Announcement{{
		ID:                   mockAnnouncementID,
		Title:                "Example announcement",
		Body:                 "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		PublicationTimestamp: types.Time(time.UnixMilli(1550058362418)),
	}}
}

// testPosition returns the documented position the mock position request returns
func testPosition() *Position {
	return &Position{
		InstrumentName:   mockPositionInstrument,
		Kind:             KindFuture,
		UserID:           1234,
		Direction:        DirectionBuy,
		MarkPrice:        3556.62,
		IndexPrice:       3555.86,
		SettlementPrice:  3555.44,
		OpenOrdersMargin: 0.000165889,
		Leverage:         100,
		InterestValue:    1.7362511643080387,
	}
}

// testPositions returns the documented positions the mock positions request returns
func testPositions() []Position {
	return []Position{{
		InstrumentName:     mockPositionInstrument,
		Kind:               KindFuture,
		UserID:             1234,
		Direction:          DirectionBuy,
		Size:               50,
		SizeCurrency:       0.006687487,
		AveragePrice:       7440.18,
		MarkPrice:          7476.65,
		IndexPrice:         7466.79,
		SettlementPrice:    7476.65,
		Delta:              0.006687487,
		InitialMargin:      0.000197283,
		MaintenanceMargin:  0.000143783,
		OpenOrdersMargin:   0.000197288,
		Leverage:           35.639422,
		TotalProfitLoss:    0.000032781,
		RealisedProfitLoss: -0.000000009,
		RealisedFunding:    -0.00000001,
		InterestValue:      1.7362511643080387,
	}}
}

// testSubaccounts returns the documented accounts the mock subaccounts request returns; the mock recording blanks
// their usernames
func testSubaccounts() []Subaccount {
	return []Subaccount{
		{
			ID:                      2,
			Type:                    "main",
			SystemName:              "user_1",
			Email:                   "user_AAA@email.com",
			LoginEnabled:            true,
			IsPassword:              true,
			SecurityKeysAssignments: []string{},
			MarginModel:             MarginModelSegregatedStandard,
			Portfolio: map[string]SubaccountPortfolio{
				"btc": {
					Currency:                 currency.NewCode("btc"),
					Balance:                  5.000593987,
					Equity:                   5.000571846,
					MarginBalance:            5.000571846,
					InitialMargin:            0.000158771,
					MaintenanceMargin:        0.000115715,
					AvailableFunds:           5.000413075,
					AvailableWithdrawalFunds: 5.000413075,
				},
				"eth": {
					Currency:                 currency.NewCode("eth"),
					Balance:                  5,
					Equity:                   5,
					MarginBalance:            5,
					AvailableFunds:           5,
					AvailableWithdrawalFunds: 5,
				},
			},
		},
		{
			ID:                      mockAccountSubaccountID,
			Type:                    "subaccount",
			SystemName:              "user_1_1",
			Email:                   "user_AAA@gmail.com",
			IsPassword:              true,
			SecurityKeysAssignments: []string{},
			MarginModel:             MarginModelCrossPortfolio,
			Portfolio: map[string]SubaccountPortfolio{
				"btc": {Currency: currency.NewCode("btc")},
				"eth": {Currency: currency.NewCode("eth")},
			},
		},
	}
}

// testSubaccountsDetails returns the documented accounts' positions the mock subaccounts details request returns, with
// the documented open order the recording added
func testSubaccountsDetails() []SubaccountDetails {
	return []SubaccountDetails{
		{
			UserID: 3,
			Positions: []Position{{
				InstrumentName:     mockPositionInstrument,
				Kind:               KindFuture,
				Direction:          DirectionBuy,
				Size:               200,
				SizeCurrency:       0.004152776,
				AveragePrice:       49571.3,
				MarkPrice:          48160.55,
				IndexPrice:         47897.12,
				SettlementPrice:    48150.36,
				Delta:              0.004152776,
				InitialMargin:      0.000122508,
				MaintenanceMargin:  0.000089286,
				Leverage:           34,
				TotalProfitLoss:    -0.000118183,
				FloatingProfitLoss: -0.00003451,
				RealisedProfitLoss: -0.000000879,
				RealisedFunding:    -0.00000088,
			}},
			OpenOrders: []Order{{
				OrderID:             "146062",
				OrderState:          "open",
				OrderType:           "limit",
				TimeInForce:         "good_til_cancelled",
				InstrumentName:      "BTC-15FEB19-3250-P",
				CreationTimestamp:   types.Time(time.UnixMilli(1550050597036)),
				LastUpdateTimestamp: types.Time(time.UnixMilli(1550050597036)),
				Direction:           DirectionBuy,
				Price:               OrderPrice{Value: 0.0028},
				Label:               "fooBar",
				API:                 true,
				Amount:              10,
			}},
		},
		{
			UserID: 10,
			Positions: []Position{{
				InstrumentName:     "BTC-3SEP21",
				Kind:               KindFuture,
				Direction:          DirectionSell,
				Size:               -60,
				SizeCurrency:       -0.001308984,
				AveragePrice:       47182.76,
				MarkPrice:          45837.07,
				IndexPrice:         47897.12,
				SettlementPrice:    47886.98,
				Delta:              -0.001308984,
				InitialMargin:      0.000038615,
				MaintenanceMargin:  0.000028143,
				Leverage:           34,
				TotalProfitLoss:    0.000037333,
				FloatingProfitLoss: 0.000037333,
			}},
		},
	}
}

// testTransactionLog returns the documented transaction log page the mock request returns, trimmed to five entries;
// the mock recording blanks the usernames
func testTransactionLog() *TransactionLogResponse {
	return &TransactionLogResponse{
		Logs: []TransactionLogEntry{
			{
				ID:                        61313,
				UserSequence:              6010,
				UserID:                    7,
				Currency:                  currency.BTC,
				Timestamp:                 types.Time(time.UnixMilli(1613659900000)),
				Type:                      "LSP transfer",
				Role:                      "source",
				Cashflow:                  -2.4,
				Change:                    -2.4,
				Balance:                   2998.82270418,
				Equity:                    2998.5275869,
				InstrumentName:            mockPositionInstrument,
				Side:                      DirectionSell,
				Amount:                    5000,
				Price:                     63481.75,
				PriceCurrency:             currency.USD,
				SessionRealisedProfitLoss: -12.4,
				InterestProfitLoss:        0.00000512,
				TotalInterestProfitLoss:   0.00000512,
			},
			{
				ID:           61312,
				UserSequence: 6009,
				UserID:       7,
				Currency:     currency.BTC,
				Timestamp:    types.Time(time.UnixMilli(1613659830333)),
				Type:         "transfer",
				Info:         map[string]any{"other_user": "Subaccount", "other_user_id": float64(27), "transfer_type": "subaccount"},
				Cashflow:     -2.5,
				Change:       -2.5,
				Balance:      3001.22270418,
				Equity:       3000.9275869,
				Side:         "-",
			},
			{
				ID:                          61311,
				UserSequence:                6008,
				UserID:                      7,
				Currency:                    currency.BTC,
				Timestamp:                   types.Time(time.UnixMilli(1613659544153)),
				Type:                        "settlement",
				Info:                        map[string]any{"floating_pl": 0.00220172, "settlement_price": float64(51807)},
				Cashflow:                    0.00215706,
				Change:                      0.00215706,
				Balance:                     3003.72270418,
				Equity:                      3003.42821428,
				InstrumentName:              mockPositionInstrument,
				Side:                        "long",
				Amount:                      1520,
				Price:                       51807.07,
				PriceCurrency:               currency.USD,
				Position:                    1520,
				SessionRealisedProfitLoss:   -0.00004467,
				SessionUnrealisedProfitLoss: 0.00220172,
				InterestProfitLoss:          0.00000993,
				TotalInterestProfitLoss:     0.00001243,
			},
			{
				ID:           61291,
				UserSequence: 6007,
				UserID:       7,
				Currency:     currency.BTC,
				Timestamp:    types.Time(time.UnixMilli(1613657828414)),
				Type:         "deposit",
				Info: map[string]any{
					"addr":         "2N8prMvpZHr8aYqodX3S4yhz5wMxjY8La3p",
					"deposit_type": "wallet",
					"transaction":  "de6eba075855f32c9510f338d3ca0900376cedcb9f7b142caccfbdc292d3237e",
				},
				Cashflow: 0.65,
				Change:   0.65,
				Balance:  3003.72054712,
				Equity:   3003.4876111,
				Side:     "-",
			},
			{
				ID:             61289,
				UserSequence:   6006,
				UserID:         7,
				Currency:       currency.BTC,
				Timestamp:      types.Time(time.UnixMilli(1613657734620)),
				Type:           "trade",
				Info:           "Source: api",
				Cashflow:       -0.10759,
				Change:         -0.10759,
				Balance:        3003.07054712,
				Equity:         3002.83270455,
				InstrumentName: "BTC-19FEB21-49200-C",
				Side:           "open buy",
				Amount:         0.7,
				Price:          0.1537,
				PriceCurrency:  currency.BTC,
				Position:       0.7,
				MarkPrice:      0.04884653215049635,
				TradeID:        "28349",
				OrderID:        "67546",
				UserRole:       "maker",
				IP:             "11.222.33.44",
			},
		},
		Continuation: 61282,
	}
}

// testUserLocks returns the documented locks the mock request returns
func testUserLocks() []UserLock {
	return []UserLock{
		{Currency: currency.BTC, Locked: true, Message: "locked in one currency"},
		{Currency: currency.ETH},
		{Currency: currency.USDC},
		{Currency: currency.SOL},
	}
}

// testAPIKeys returns the documented API keys the mock list returns; the mock recording blanks their names
func testAPIKeys() []APIKey {
	return []APIKey{
		{
			ID:              1,
			Timestamp:       types.Time(time.UnixMilli(1560236001108)),
			ClientID:        "TiA4AyLPq3",
			ClientSecret:    "SjM57m1T2CfXZ4vZ76X1APjqRlJdtzHI8IwVXoQnfoM",
			MaxScope:        APIKeyScopes{"account:read", "block_trade:read", "trade:read_write", "wallet:read"},
			EnabledFeatures: []string{},
		},
		{
			ID:              2,
			Timestamp:       types.Time(time.UnixMilli(1560236287708)),
			ClientID:        "aD-KFx-H",
			ClientSecret:    "mwNOvbUVyQczytQ5IVM8CbzmgqNJ81WvLKfu6MXcJPs",
			MaxScope:        APIKeyScopes{"account:read_write", "block_trade:read_write", "trade:read_write", "wallet:read_write"},
			Enabled:         true,
			Default:         true,
			EnabledFeatures: []string{},
		},
	}
}

// testCustodyAccounts returns the documented custody accounts the mock request returns; the mock recording blanks
// their names
func testCustodyAccounts() []CustodyAccount {
	return []CustodyAccount{{
		Currency:                 currency.BTC,
		ClientID:                 "4KVcFrrzmXBR",
		ExternalID:               "24f97d44-1d72-4641-8527-811268a0bdd3",
		Balance:                  0.5,
		PendingWithdrawalBalance: 0.1,
	}}
}

// testPMESimulation returns the documented portfolio margin simulation the mock request returns
func testPMESimulation() *PMESimulateResponse {
	return &PMESimulateResponse{
		ModelParameters: PMEModelParameters{
			CurrencyPairs: map[string]PMECurrencyPairParameters{"btc_usd": {
				PriceRange:                        0.16,
				ExtendedTableFactor:               1,
				VolatilityRangeUp:                 0.5,
				VolatilityRangeDown:               0.25,
				MinimumVolatilityForShockUp:       0.5,
				ShortTermVegaPower:                0.3,
				LongTermVegaPower:                 0.13,
				MarginIncrement:                   0.00005,
				DeltaTotalLiquidityShockThreshold: 20000000,
				MaximumDeltaShock:                 0.1,
			}},
			Currencies: map[string]PMECurrencyParameters{
				"btc": {AnnualisedMoveRisk: 0.075, MinimumAnnualisedMove: 0.01, ExtendedDampener: 100000, EquitySideImpact: "both"},
				"usd": {AnnualisedMoveRisk: 0.1, MinimumAnnualisedMove: 0.01, ExtendedDampener: 25000, EquitySideImpact: "none"},
			},
			General: PMEGeneralParameters{
				MaintenanceMarginFactor:  0.8,
				BucketsCount:             4,
				VolatilityScenariosCount: 3,
				Timestamp:                types.Time(time.UnixMilli(1718619740501)),
			},
		},
		AggregatedRiskVectors: map[string]PMERiskVectors{"btc_btc": {
			Standard: []float64{-0.05968587238095239, -0.05968587238095239, -0.05968587238095239, -0.04272965863636364, -0.04272965863636364, -0.04272965863636364, -0.02724789826086957, -0.02724789826086957, -0.02724789826086957, -0.013056284583333334, -0.013056284583333334, -0.013056284583333334, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Extended: []float64{0, 0, 0, 0, 0, 0, 0, 0},
		}},
		InitialRiskVectors: map[string]PMERiskVectors{
			"BTC-28JUN24": {
				Standard: []float64{0.0002261969523809524, 0.0002261969523809524, 0.0002261969523809524, 0.00016193645454545456, 0.00016193645454545456, 0.00016193645454545456, 0.00010326382608695652, 0.00010326382608695652, 0.00010326382608695652, 4.948058333333334e-05, 4.948058333333334e-05, 4.948058333333334e-05, 0, 0, 0, -4.567438461538462e-05, -4.567438461538462e-05, -4.567438461538462e-05, -8.796548148148148e-05, -8.796548148148148e-05, -8.796548148148148e-05, -0.0001272357857142857, -0.0001272357857142857, -0.0001272357857142857, -0.00016379779310344832, -0.00016379779310344832, -0.00016379779310344832},
				Extended: []float64{0.0002261969523809524, 0.0002261969523809524, -0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483},
			},
			mockPositionInstrument: {
				Standard: []float64{-0.05991206933333334, -0.05991206933333334, -0.05991206933333334, -0.04289159509090909, -0.04289159509090909, -0.04289159509090909, -0.027351162086956524, -0.027351162086956524, -0.027351162086956524, -0.013105765166666668, -0.013105765166666668, -0.013105765166666668, 0, 0, 0, 0.012097629384615385, 0.012097629384615385, 0.012097629384615385, 0.023299138074074074, 0.023299138074074074, 0.023299138074074074, 0.033700538999999995, 0.033700538999999995, 0.033700538999999995, 0.043384601931034494, 0.043384601931034494, 0.043384601931034494},
				Extended: []float64{-0.05991206933333334, -0.05991206933333334, 0.04338460193103449, 0.04338460193103449, 0.04338460193103449, 0.04338460193103449, 0.04338460193103449, 0.04338460193103449},
			},
		},
		Margins: map[string]PMEMargins{"btc": {
			InitialMargin:     0.122861343,
			MaintenanceMargin: 0.050274504784,
			InitialMarginDetails: PMEInitialMarginDetails{
				RiskMatrixMargin: 0.06284313098,
				RiskMatrixMarginDetails: PMERiskMatrixMarginDetails{
					WorstCaseLoss:   0.05968587238095239,
					WorstCaseBucket: PMEWorstCaseBucket{Source: "standard", Index: 1, Bucket: 1, Side: "left"},
					RollShock:       0.00315725898,
				},
				MMPMargin:        0.06,
				OpenOrdersMargin: 0.000018212,
			},
		}},
		Portfolio: PMEPortfolio{
			Currencies: map[string]float64{},
			Positions:  map[string]float64{"BTC-28JUN24": -0.001187534, mockPositionInstrument: 0.314538364},
		},
		IndexPrices: map[string]float64{"btc_usd": 65666.19},
		Tickers: map[string]PMETicker{
			"BTC-28JUN24":          {MarkPrice: 67371.75, IndexPrice: 65666.19},
			mockPositionInstrument: {MarkPrice: 65910.57, IndexPrice: 65666.19},
		},
	}
}

// testSimulatedPortfolio returns the documented portfolio simulation the mock request returns
func testSimulatedPortfolio() *SimulatePortfolioResponse {
	return &SimulatePortfolioResponse{
		Currency:                           currency.BTC,
		Balance:                            150076473.4995114,
		Equity:                             150075253.91354558,
		MarginBalance:                      153534213.79481918,
		AvailableFunds:                     115871741.76065847,
		AvailableWithdrawalFunds:           115871741.76065847,
		InitialMargin:                      37662472.03416069,
		MaintenanceMargin:                  30129215.84817124,
		ProjectedInitialMargin:             37662472.03416069,
		ProjectedMaintenanceMargin:         30129215.84817124,
		TotalProfitLoss:                    40419.10179263,
		SessionRealisedProfitLoss:          1.309136,
		SessionUnrealisedProfitLoss:        -339.16214185,
		FuturesProfitLoss:                  39497.54616685,
		FuturesSessionRealisedProfitLoss:   1.309136,
		FuturesSessionUnrealisedProfitLoss: -164.48253509,
		OptionsProfitLoss:                  921.55562578,
		OptionsSessionUnrealisedProfitLoss: -174.67960675,
		OptionsValue:                       -1056.41256672,
		OptionsDelta:                       2883.38481,
		OptionsGamma:                       -0.03907,
		OptionsTheta:                       142583.29246,
		OptionsVega:                        -39322.23046,
		OptionsGammaByIndex:                map[string]float64{"btc_usd": -0.03907},
		OptionsThetaByIndex:                map[string]float64{"btc_usd": 142583.29246},
		OptionsVegaByIndex:                 map[string]float64{"btc_usd": -39322.23046},
		DeltaTotal:                         69080.932029,
		ProjectedDeltaTotal:                69080.932029,
		DeltaTotalByIndex:                  map[string]float64{"btc_usd": 68024.519462366},
		MarginModel:                        MarginModelCrossPortfolio,
		PortfolioMarginingEnabled:          true,
		CrossCollateralEnabled:             true,
		TotalEquityUSD:                     13075634611389.318,
		TotalInitialMarginUSD:              3139528603778.822,
		TotalMaintenanceMarginUSD:          2511559381417.215,
		TotalMarginBalanceUSD:              12798550648250.61,
		TotalDeltaTotalUSD:                 6157454218.3753195,
	}
}

// testAnnouncements returns the announcement the announcements request returned when recorded live
func testAnnouncements() []Announcement {
	return []Announcement{{
		ID:                   1791813857051,
		Title:                "🚨 New Equity & ETF Perpetuals. Coming to Deribit.",
		Body:                 `On Wednesday, October 14, 2026 at 10:00 AM UTC, Deribit will launch the following USDC-settled equity perpetuals: <br><br>CRWV • ORCL • SOXS • DELL • PLTR • SNXX<br><br>Contract details:<br>🔹 Settlement: USDC<br>🔹&nbsp;<a href="https://support.deribit.com/hc/en-us/articles/38325634622493-RWA-Perpetual" style="font-family: Arial; font-size: 16px;">Full specs </a><br><br><b style="font-family: Arial; font-size: 16px; font-weight: bold;">Deribit. Always Open.</b><div></div>`,
		Important:            true,
		PublicationTimestamp: types.Time(time.UnixMilli(1791713204677)),
	}}
}

// testAnnouncementsRequest returns the request selecting the recorded announcement
func testAnnouncementsRequest() *GetAnnouncementsRequest {
	return &GetAnnouncementsRequest{StartTime: time.UnixMilli(mockAnnouncementsStartTime), Count: mockAnnouncementsCount}
}

func TestChangeAPIKeyName(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeAPIKeyName(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeAPIKeyName must reject a nil request")
	_, err = e.ChangeAPIKeyName(t.Context(), &ChangeAPIKeyNameRequest{Name: mockAPIKeyName})
	require.ErrorIs(t, err, errAPIKeyIDRequired, "ChangeAPIKeyName must reject an empty API key ID")
	_, err = e.ChangeAPIKeyName(t.Context(), &ChangeAPIKeyNameRequest{ID: mockAPIKeyID})
	require.ErrorIs(t, err, errAPIKeyNameRequired, "ChangeAPIKeyName must reject an empty name")

	skipLiveUnlessCanManipulate(t)
	arg := testChangeAPIKeyNameRequest(t)
	result, err := e.ChangeAPIKeyName(t.Context(), arg)
	require.NoError(t, err, "ChangeAPIKeyName must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "ChangeAPIKeyName should decode every field")
	} else {
		assert.Equal(t, arg.Name, result.Name, "ChangeAPIKeyName should return the new name")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ChangeAPIKeyName(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ChangeAPIKeyName should reject a null result")
	_, err = newUnavailableServerExchange(t).ChangeAPIKeyName(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ChangeAPIKeyName should return the request failure")
}

func TestChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeMarginModel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeMarginModel must reject a nil request")
	_, err = e.ChangeMarginModel(t.Context(), &ChangeMarginModelRequest{MarginModel: "cross"})
	require.ErrorIs(t, err, errInvalidMarginModel, "ChangeMarginModel must reject an undocumented margin model")

	// The live request is a dry run, which changes nothing, but needs the account write scope
	skipLiveUnlessCanManipulate(t)
	result, err := e.ChangeMarginModel(t.Context(), testChangeMarginModelRequest())
	require.NoError(t, err, "ChangeMarginModel must not error")
	if mockTests {
		assert.Equal(t, testMarginModelChanges(), result, "ChangeMarginModel should decode every field")
	} else {
		assert.NotEmpty(t, result, "ChangeMarginModel should return the change's effect on each currency")
	}

	_, err = newUnavailableServerExchange(t).ChangeMarginModel(t.Context(), testChangeMarginModelRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ChangeMarginModel should return the request failure")
}

func TestChangeScopeInAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeScopeInAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeScopeInAPIKey must reject a nil request")
	_, err = e.ChangeScopeInAPIKey(t.Context(), &ChangeScopeInAPIKeyRequest{MaxScope: testAPIKeyScopes()})
	require.ErrorIs(t, err, errAPIKeyIDRequired, "ChangeScopeInAPIKey must reject an empty API key ID")
	_, err = e.ChangeScopeInAPIKey(t.Context(), &ChangeScopeInAPIKeyRequest{ID: mockAPIKeyID})
	require.ErrorIs(t, err, errAPIKeyMaxScopeRequired, "ChangeScopeInAPIKey must reject an empty scope")

	skipLiveUnlessCanManipulate(t)
	arg := testChangeScopeInAPIKeyRequest(t)
	result, err := e.ChangeScopeInAPIKey(t.Context(), arg)
	require.NoError(t, err, "ChangeScopeInAPIKey must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "ChangeScopeInAPIKey should decode every field")
	} else {
		assert.ElementsMatch(t, arg.MaxScope, result.MaxScope, "ChangeScopeInAPIKey should return the new scope")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ChangeScopeInAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ChangeScopeInAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).ChangeScopeInAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ChangeScopeInAPIKey should return the request failure")
}

func TestChangeSubaccountName(t *testing.T) {
	t.Parallel()
	err := e.ChangeSubaccountName(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeSubaccountName must reject a nil request")
	err = e.ChangeSubaccountName(t.Context(), &ChangeSubaccountNameRequest{Name: mockSubaccountName})
	require.ErrorIs(t, err, errSubaccountIDRequired, "ChangeSubaccountName must reject an empty subaccount ID")
	err = e.ChangeSubaccountName(t.Context(), &ChangeSubaccountNameRequest{SubaccountID: mockAccountSubaccountID})
	require.ErrorIs(t, err, errSubaccountNameRequired, "ChangeSubaccountName must reject an empty name")

	skipLiveUnlessCanManipulate(t)
	arg := testChangeSubaccountNameRequest(t)
	require.NoError(t, e.ChangeSubaccountName(t.Context(), arg), "ChangeSubaccountName must not error")

	err = newUnavailableServerExchange(t).ChangeSubaccountName(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ChangeSubaccountName should return the request failure")
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.CreateAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAPIKey must reject a nil request")
	_, err = e.CreateAPIKey(t.Context(), &CreateAPIKeyRequest{Name: mockCreatedAPIKeyName})
	require.ErrorIs(t, err, errAPIKeyMaxScopeRequired, "CreateAPIKey must reject an empty scope")
	_, err = e.CreateAPIKey(t.Context(), &CreateAPIKeyRequest{MaxScope: testAPIKeyScopes(), EnabledFeatures: []string{"bogus"}})
	require.ErrorIs(t, err, errInvalidAPIKeyFeature, "CreateAPIKey must reject an undocumented feature")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		result, err := e.CreateAPIKey(t.Context(), testCreateAPIKeyRequest())
		require.NoError(t, err, "CreateAPIKey must not error")
		assert.Equal(t, testAPIKey(), result, "CreateAPIKey should decode every field")
	} else {
		result, err := e.CreateAPIKey(t.Context(), liveTestAPIKeyRequest())
		require.NoError(t, err, "CreateAPIKey must not error")
		assert.NotEmpty(t, result.ClientSecret, "CreateAPIKey should return the key's secret")
		require.NoError(t, e.RemoveAPIKey(t.Context(), result.ID), "RemoveAPIKey must remove the created API key")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CreateAPIKey(t.Context(), testCreateAPIKeyRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).CreateAPIKey(t.Context(), testCreateAPIKeyRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateAPIKey should return the request failure")
}

func TestValidateAPIKeyFeatures(t *testing.T) {
	t.Parallel()
	for _, features := range [][]string{nil, {EnabledFeatureRestrictedBlockTrades}, {EnabledFeatureRestrictedBlockTrades, EnabledFeatureBlockTradeApproval}} {
		assert.NoErrorf(t, validateAPIKeyFeatures(features), "validateAPIKeyFeatures should accept features %v", features)
	}
	err := validateAPIKeyFeatures([]string{EnabledFeatureBlockTradeApproval, "Block_Trade_Approval"})
	assert.ErrorIs(t, err, errInvalidAPIKeyFeature, "validateAPIKeyFeatures should reject a feature Deribit does not document")
}

func TestCreateSubaccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("CreateSubaccount would create a subaccount, which live tests cannot undo")
	}
	result, err := e.CreateSubaccount(t.Context())
	require.NoError(t, err, "CreateSubaccount must not error")
	assert.Equal(t, testCreatedSubaccount(), result, "CreateSubaccount should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CreateSubaccount(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateSubaccount should reject a null result")
	_, err = newUnavailableServerExchange(t).CreateSubaccount(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateSubaccount should return the request failure")
}

func TestDeleteMember(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteMember(t.Context(), 0)
	require.ErrorIs(t, err, errMemberIDRequired, "DeleteMember must reject an empty member ID")

	if !mockTests {
		t.Skip("DeleteMember would delete a Direct Access member, which live tests cannot recreate")
	}
	result, err := e.DeleteMember(t.Context(), mockMemberID)
	require.NoError(t, err, "DeleteMember must not error")
	assert.Equal(t, &MembersResponse{Members: []Member{}, IsDirectAccessAllowed: true}, result, "DeleteMember should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).DeleteMember(t.Context(), mockMemberID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "DeleteMember should reject a null result")
	_, err = newUnavailableServerExchange(t).DeleteMember(t.Context(), mockMemberID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DeleteMember should return the request failure")
}

func TestDisableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.DisableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errAPIKeyIDRequired, "DisableAPIKey must reject an empty API key ID")

	skipLiveUnlessCanManipulate(t)
	id := testAPIKeyID(t)
	result, err := e.DisableAPIKey(t.Context(), id)
	require.NoError(t, err, "DisableAPIKey must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "DisableAPIKey should decode every field")
	} else {
		assert.False(t, result.Enabled, "DisableAPIKey should return the key disabled")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).DisableAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, common.ErrNoResponse, "DisableAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).DisableAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, request.ErrBadStatus, "DisableAPIKey should return the request failure")
}

func TestEditAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.EditAPIKey(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditAPIKey must reject a nil request")
	_, err = e.EditAPIKey(t.Context(), &EditAPIKeyRequest{MaxScope: testAPIKeyScopes()})
	require.ErrorIs(t, err, errAPIKeyIDRequired, "EditAPIKey must reject an empty API key ID")
	_, err = e.EditAPIKey(t.Context(), &EditAPIKeyRequest{ID: mockAPIKeyID})
	require.ErrorIs(t, err, errAPIKeyMaxScopeRequired, "EditAPIKey must reject an empty scope")
	_, err = e.EditAPIKey(t.Context(), &EditAPIKeyRequest{ID: mockAPIKeyID, MaxScope: testAPIKeyScopes(), EnabledFeatures: []string{"bogus"}})
	require.ErrorIs(t, err, errInvalidAPIKeyFeature, "EditAPIKey must reject an undocumented feature")

	skipLiveUnlessCanManipulate(t)
	arg := testEditAPIKeyRequest(t)
	result, err := e.EditAPIKey(t.Context(), arg)
	require.NoError(t, err, "EditAPIKey must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "EditAPIKey should decode every field")
	} else {
		assert.Equal(t, arg.Name, result.Name, "EditAPIKey should return the new name")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).EditAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EditAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).EditAPIKey(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EditAPIKey should return the request failure")
}

func TestEditAPIKeyRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	enabled := false
	for _, tc := range []struct {
		arg *EditAPIKeyRequest
		exp string
	}{
		{arg: &EditAPIKeyRequest{ID: mockAPIKeyID, MaxScope: APIKeyScopes{"account:read"}}, exp: `{"id":3,"max_scope":"account:read"}`},
		{
			arg: &EditAPIKeyRequest{ID: mockAPIKeyID, MaxScope: APIKeyScopes{"account:read"}, Enabled: &enabled, EnabledFeatures: []string{}, IPWhitelist: []string{}},
			exp: `{"id":3,"max_scope":"account:read","enabled":false,"enabled_features":[],"ip_whitelist":[]}`,
		},
	} {
		encoded, err := json.Marshal(tc.arg)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, tc.exp, string(encoded), "EditAPIKeyRequest should send a set false state and empty lists, which clear the key's")
	}
}

func TestEnableAffiliateProgram(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("EnableAffiliateProgram would enable the affiliate program, which live tests cannot disable")
	}
	require.NoError(t, e.EnableAffiliateProgram(t.Context()), "EnableAffiliateProgram must not error")

	err := newUnavailableServerExchange(t).EnableAffiliateProgram(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "EnableAffiliateProgram should return the request failure")
}

func TestEnableAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.EnableAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errAPIKeyIDRequired, "EnableAPIKey must reject an empty API key ID")

	skipLiveUnlessCanManipulate(t)
	id := testAPIKeyID(t)
	result, err := e.EnableAPIKey(t.Context(), id)
	require.NoError(t, err, "EnableAPIKey must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "EnableAPIKey should decode every field")
	} else {
		assert.True(t, result.Enabled, "EnableAPIKey should return the key enabled")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).EnableAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, common.ErrNoResponse, "EnableAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).EnableAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, request.ErrBadStatus, "EnableAPIKey should return the request failure")
}

func TestGetAccessLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccessLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccessLog must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAccessLog(t.Context(), &GetAccessLogRequest{Count: mockAccessLogCount})
	require.NoError(t, err, "GetAccessLog must not error")
	if mockTests {
		assert.Equal(t, testAccessLog(), result, "GetAccessLog should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Entries), mockAccessLogCount, "GetAccessLog should return at most the entries asked for")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAccessLog(t.Context(), &GetAccessLogRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccessLog should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAccessLog(t.Context(), &GetAccessLogRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccessLog should return the request failure")
}

func TestAccessLogResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	entry := AccessLogEntry{ID: 46, Timestamp: types.Time(time.UnixMilli(1575876682576)), IP: "127.0.0.1", Country: "Local Country", City: "Local Town", Log: "new_api_key", Data: "IY2D68DS"}
	for _, tc := range []struct {
		input string
		exp   AccessLogResponse
	}{
		{
			input: `{"records_total":1,"data":[{"id":46,"timestamp":1575876682576,"ip":"127.0.0.1","country":"Local Country","city":"Local Town","log":"new_api_key","data":"IY2D68DS"}]}`,
			exp:   AccessLogResponse{RecordsTotal: 1, Entries: []AccessLogEntry{entry}},
		},
		{
			input: `[{"id":46,"timestamp":1575876682576,"ip":"127.0.0.1","country":"Local Country","city":"Local Town","log":"new_api_key","data":"IY2D68DS"}]`,
			exp:   AccessLogResponse{Entries: []AccessLogEntry{entry}},
		},
		{input: `[]`, exp: AccessLogResponse{Entries: []AccessLogEntry{}}},
	} {
		var result AccessLogResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, result, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`"log"`, `[1]`, `{"data":{}}`} {
		var result AccessLogResponse
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &result), errInvalidAccessLog, "UnmarshalJSON should reject %s", input)
	}
}

func TestGetAccountSummaries(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountSummaries(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountSummaries must reject a nil request")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{Extended: true})
	require.NoError(t, err, "GetAccountSummaries must not error")
	if mockTests {
		assert.Equal(t, testAccountSummaries(), result, "GetAccountSummaries should decode every field")
	} else {
		assert.NotEmpty(t, result.Summaries, "GetAccountSummaries should return a summary for each currency the account holds")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSummaries should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAccountSummaries(t.Context(), &GetAccountSummariesRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSummaries should return the request failure")
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountSummary must reject a nil request")
	_, err = e.GetAccountSummary(t.Context(), &GetAccountSummaryRequest{Extended: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAccountSummary must reject an empty currency")

	skipLiveWithoutCredentials(t)
	arg := &GetAccountSummaryRequest{Currency: currency.BTC, Extended: true}
	result, err := e.GetAccountSummary(t.Context(), arg)
	require.NoError(t, err, "GetAccountSummary must not error")
	if mockTests {
		exp := &AccountSummaryResponse{AccountSummary: testBTCAccountSummary(), AccountDetails: testAccountDetails()}
		assert.Equal(t, exp, result, "GetAccountSummary should decode every field")
	} else {
		assert.Equal(t, currency.BTC, result.Currency, "GetAccountSummary should return the summary in the currency asked for")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSummary should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSummary should return the request failure")
}

func TestMatchingEngineLimitsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
		exp   MatchingEngineLimits
	}{
		{
			name:  "global limits",
			input: `{"trading":{"total":{"burst":20,"rate":5}},"spot":{"burst":250,"rate":200},"maximum_quotes":{"burst":500,"rate":500},"maximum_mass_quotes":{"burst":10,"rate":10},"guaranteed_mass_quotes":{"burst":2,"rate":2},"cancel_all":{"burst":250,"rate":200}}`,
			exp: MatchingEngineLimits{
				Spot:      AccountRateLimit{Rate: 200, Burst: 250},
				CancelAll: AccountRateLimit{Rate: 200, Burst: 250},
				MatchingEngineTradingLimits: MatchingEngineTradingLimits{
					Trading:              map[string]AccountRateLimit{"total": {Rate: 5, Burst: 20}},
					MaximumQuotes:        AccountRateLimit{Rate: 500, Burst: 500},
					MaximumMassQuotes:    AccountRateLimit{Rate: 10, Burst: 10},
					GuaranteedMassQuotes: AccountRateLimit{Rate: 2, Burst: 2},
				},
			},
		},
		{
			name:  "per currency limits",
			input: `{"cancel_all":{"burst":250,"rate":200},"spot":{"burst":250,"rate":200},"usdt":{"maximum_quotes":{"burst":500,"rate":500},"maximum_mass_quotes":{"burst":10,"rate":10},"guaranteed_mass_quotes":{"burst":2,"rate":2},"trading":{"total":{"burst":250,"rate":200}}},"btc":{"maximum_quotes":{"burst":500,"rate":500},"maximum_mass_quotes":{"burst":10,"rate":10},"guaranteed_mass_quotes":{"burst":2,"rate":2},"trading":{"perpetuals":{"burst":20,"rate":10},"total":{"burst":150,"rate":100}}}}`,
			exp: MatchingEngineLimits{
				Spot:      AccountRateLimit{Rate: 200, Burst: 250},
				CancelAll: AccountRateLimit{Rate: 200, Burst: 250},
				Currencies: map[string]MatchingEngineTradingLimits{
					"usdt": {
						Trading:              map[string]AccountRateLimit{"total": {Rate: 200, Burst: 250}},
						MaximumQuotes:        AccountRateLimit{Rate: 500, Burst: 500},
						MaximumMassQuotes:    AccountRateLimit{Rate: 10, Burst: 10},
						GuaranteedMassQuotes: AccountRateLimit{Rate: 2, Burst: 2},
					},
					"btc": {
						Trading:              map[string]AccountRateLimit{"perpetuals": {Rate: 10, Burst: 20}, "total": {Rate: 100, Burst: 150}},
						MaximumQuotes:        AccountRateLimit{Rate: 500, Burst: 500},
						MaximumMassQuotes:    AccountRateLimit{Rate: 10, Burst: 10},
						GuaranteedMassQuotes: AccountRateLimit{Rate: 2, Burst: 2},
					},
				},
			},
		},
		{
			name:  "documented example names",
			input: `{"quotes":{"burst":500,"rate":500},"max_quotes":{"burst":10,"rate":10},"guaranteed_quotes":{"burst":2,"rate":2}}`,
			exp: MatchingEngineLimits{MatchingEngineTradingLimits: MatchingEngineTradingLimits{
				MaximumQuotes:        AccountRateLimit{Rate: 500, Burst: 500},
				MaximumMassQuotes:    AccountRateLimit{Rate: 10, Burst: 10},
				GuaranteedMassQuotes: AccountRateLimit{Rate: 2, Burst: 2},
			}},
		},
		{name: "null", input: `null`},
	} {
		var result MatchingEngineLimits
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &result), "Unmarshal must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, result, "UnmarshalJSON should decode %s", tc.name)
	}
	for _, input := range []string{`[]`, `{"spot":[]}`, `{"btc":{"trading":1}}`} {
		var result MatchingEngineLimits
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &result), errInvalidMatchingEngineLimits, "UnmarshalJSON should reject %s", input)
	}
}

func TestGetAffiliateProgramInfo(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAffiliateProgramInfo(t.Context())
	require.NoError(t, err, "GetAffiliateProgramInfo must not error")
	if mockTests {
		assert.Equal(t, testAffiliateProgramInfo(), result, "GetAffiliateProgramInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetAffiliateProgramInfo should return the affiliate program information")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetAffiliateProgramInfo(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAffiliateProgramInfo should reject a null result")
	_, err = newUnavailableServerExchange(t).GetAffiliateProgramInfo(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAffiliateProgramInfo should return the request failure")
}

func TestGetAccountCurrencies(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountCurrencies(t.Context())
	require.NoError(t, err, "GetAccountCurrencies must not error")
	if mockTests {
		exp := []AccountCurrency{{Currency: currency.NewCode("btc")}, {Currency: currency.NewCode("eth")}, {Currency: currency.NewCode("usdc")}}
		assert.Equal(t, exp, result, "GetAccountCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetAccountCurrencies should return the account's currencies")
	}

	_, err = newUnavailableServerExchange(t).GetAccountCurrencies(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountCurrencies should return the request failure")
}

func TestGetEmailLanguage(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetEmailLanguage(t.Context())
	require.NoError(t, err, "GetEmailLanguage must not error")
	if mockTests {
		assert.Equal(t, mockEmailLanguage, result, "GetEmailLanguage should decode the language")
	} else {
		assert.NotEmpty(t, result, "GetEmailLanguage should return the language")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetEmailLanguage(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetEmailLanguage should reject a null result")
	_, err = newUnavailableServerExchange(t).GetEmailLanguage(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetEmailLanguage should return the request failure")
}

func TestGetMembers(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetMembers(t.Context())
	require.NoError(t, err, "GetMembers must not error")
	if mockTests {
		assert.Equal(t, testMembers(), result, "GetMembers should decode every field")
	} else {
		assert.NotNil(t, result, "GetMembers should return the account's members")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetMembers(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMembers should reject a null result")
	_, err = newUnavailableServerExchange(t).GetMembers(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMembers should return the request failure")
}

func TestGetNewAnnouncements(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetNewAnnouncements(t.Context())
	require.NoError(t, err, "GetNewAnnouncements must not error")
	if mockTests {
		assert.Equal(t, testNewAnnouncements(), result, "GetNewAnnouncements should decode every field")
	}

	_, err = newUnavailableServerExchange(t).GetNewAnnouncements(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetNewAnnouncements should return the request failure")
}

func TestGetPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetPosition(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetPosition must reject an empty instrument name")

	skipLiveWithoutCredentials(t)
	result, err := e.GetPosition(t.Context(), mockPositionInstrument)
	require.NoError(t, err, "GetPosition must not error")
	if mockTests {
		assert.Equal(t, testPosition(), result, "GetPosition should decode every field")
	} else {
		assert.Equal(t, mockPositionInstrument, result.InstrumentName, "GetPosition should return the position in the instrument asked for")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetPosition(t.Context(), mockPositionInstrument)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPosition should reject a null result")
	_, err = newUnavailableServerExchange(t).GetPosition(t.Context(), mockPositionInstrument)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPosition should return the request failure")
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositions must reject a nil request")
	_, err = e.GetPositions(t.Context(), &GetPositionsRequest{Kind: KindSpot})
	require.ErrorIs(t, err, errInvalidKind, "GetPositions must reject spot, which holds balances rather than positions")

	skipLiveWithoutCredentials(t)
	arg := &GetPositionsRequest{Currency: currency.BTC, Kind: KindFuture}
	result, err := e.GetPositions(t.Context(), arg)
	require.NoError(t, err, "GetPositions must not error")
	if mockTests {
		assert.Equal(t, testPositions(), result, "GetPositions should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, KindFuture, result[i].Kind, "GetPositions should return positions of the kind asked for")
		}
	}

	_, err = newUnavailableServerExchange(t).GetPositions(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositions should return the request failure")
}

func TestGetSubaccounts(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetSubaccounts(t.Context(), true)
	require.NoError(t, err, "GetSubaccounts must not error")
	if mockTests {
		assert.Equal(t, testSubaccounts(), result, "GetSubaccounts should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetSubaccounts should return the main account")
	}

	_, err = newUnavailableServerExchange(t).GetSubaccounts(t.Context(), true)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSubaccounts should return the request failure")
}

func TestGetSubaccountsDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountsDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubaccountsDetails must reject a nil request")
	_, err = e.GetSubaccountsDetails(t.Context(), &GetSubaccountsDetailsRequest{WithOpenOrders: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSubaccountsDetails must reject an empty currency")

	skipLiveWithoutCredentials(t)
	arg := &GetSubaccountsDetailsRequest{Currency: currency.BTC, WithOpenOrders: true}
	result, err := e.GetSubaccountsDetails(t.Context(), arg)
	require.NoError(t, err, "GetSubaccountsDetails must not error")
	if mockTests {
		assert.Equal(t, testSubaccountsDetails(), result, "GetSubaccountsDetails should decode every field")
	} else {
		for i := range result {
			assert.NotZero(t, result[i].UserID, "GetSubaccountsDetails should identify each account")
		}
	}

	_, err = newUnavailableServerExchange(t).GetSubaccountsDetails(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSubaccountsDetails should return the request failure")
}

func TestGetTransactionLog(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(mockTransactionLogStartTime), time.UnixMilli(mockTransactionLogEndTime)
	_, err := e.GetTransactionLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransactionLog must reject a nil request")
	_, err = e.GetTransactionLog(t.Context(), &GetTransactionLogRequest{StartTime: start, EndTime: end})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTransactionLog must reject an empty currency")
	_, err = e.GetTransactionLog(t.Context(), &GetTransactionLogRequest{Currency: currency.BTC, EndTime: end})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTransactionLog must reject an empty start time")
	_, err = e.GetTransactionLog(t.Context(), &GetTransactionLogRequest{Currency: currency.BTC, StartTime: start})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTransactionLog must reject an empty end time")
	_, err = e.GetTransactionLog(t.Context(), &GetTransactionLogRequest{Currency: currency.BTC, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTransactionLog must reject a start time after the end time")

	skipLiveWithoutCredentials(t)
	result, err := e.GetTransactionLog(t.Context(), testTransactionLogRequest())
	require.NoError(t, err, "GetTransactionLog must not error")
	if mockTests {
		assert.Equal(t, testTransactionLog(), result, "GetTransactionLog should decode every field")
	} else {
		assert.LessOrEqual(t, len(result.Logs), mockTransactionLogCount, "GetTransactionLog should return at most the entries asked for")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTransactionLog(t.Context(), testTransactionLogRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransactionLog should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTransactionLog(t.Context(), testTransactionLogRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransactionLog should return the request failure")
}

func TestGetUserLocks(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetUserLocks(t.Context())
	require.NoError(t, err, "GetUserLocks must not error")
	if mockTests {
		assert.Equal(t, testUserLocks(), result, "GetUserLocks should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetUserLocks should return the account's state in each currency")
	}

	_, err = newUnavailableServerExchange(t).GetUserLocks(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserLocks should return the request failure")
}

func TestListAPIKeys(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.ListAPIKeys(t.Context())
	require.NoError(t, err, "ListAPIKeys must not error")
	if mockTests {
		assert.Equal(t, testAPIKeys(), result, "ListAPIKeys should decode every field")
	} else {
		assert.NotEmpty(t, result, "ListAPIKeys should return the API key in use")
	}

	_, err = newUnavailableServerExchange(t).ListAPIKeys(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListAPIKeys should return the request failure")
}

func TestAPIKeyScopesMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		scopes APIKeyScopes
		exp    string
	}{
		{scopes: testAPIKeyScopes(), exp: `"account:read_write wallet:read_write block_trade:read trade:read_write"`},
		{scopes: APIKeyScopes{"account:read"}, exp: `"account:read"`},
		{exp: `""`},
	} {
		encoded, err := json.Marshal(tc.scopes)
		require.NoErrorf(t, err, "Marshal must not error for %v", tc.scopes)
		assert.JSONEqf(t, tc.exp, string(encoded), "MarshalJSON should send scopes %v as a space separated string", tc.scopes)
	}
}

func TestAPIKeyScopesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   APIKeyScopes
	}{
		{input: `"account:read_write block_trade:read trade:read_write"`, exp: APIKeyScopes{"account:read_write", "block_trade:read", "trade:read_write"}},
		{input: `["account:read","trade:read"]`, exp: APIKeyScopes{"account:read", "trade:read"}},
		{input: `""`, exp: APIKeyScopes{}},
		{input: `null`},
	} {
		var scopes APIKeyScopes
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &scopes), "Unmarshal must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, scopes, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`1`, `[1]`, `{}`} {
		var scopes APIKeyScopes
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &scopes), errInvalidAPIKeyScopes, "UnmarshalJSON should reject %s", input)
	}
}

func TestListCustodyAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.ListCustodyAccounts(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "ListCustodyAccounts must reject an empty currency")

	skipLiveWithoutCredentials(t)
	result, err := e.ListCustodyAccounts(t.Context(), currency.BTC)
	require.NoError(t, err, "ListCustodyAccounts must not error")
	if mockTests {
		assert.Equal(t, testCustodyAccounts(), result, "ListCustodyAccounts should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, currency.BTC, result[i].Currency, "ListCustodyAccounts should return custody accounts in the currency asked for")
		}
	}

	_, err = newUnavailableServerExchange(t).ListCustodyAccounts(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListCustodyAccounts should return the request failure")
}

func TestPMESimulate(t *testing.T) {
	t.Parallel()
	_, err := e.PMESimulate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PMESimulate must reject a nil request")
	_, err = e.PMESimulate(t.Context(), &SimulatePortfolioRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PMESimulate must reject an empty currency")

	skipLiveWithoutCredentials(t)
	result, err := e.PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
	require.NoError(t, err, "PMESimulate must not error")
	if mockTests {
		assert.Equal(t, testPMESimulation(), result, "PMESimulate should decode every field")
	} else {
		assert.NotEmpty(t, result.IndexPrices, "PMESimulate should return the index prices simulated")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
	assert.ErrorIs(t, err, common.ErrNoResponse, "PMESimulate should reject a null result")
	_, err = newUnavailableServerExchange(t).PMESimulate(t.Context(), &SimulatePortfolioRequest{Currency: currency.BTC})
	assert.ErrorIs(t, err, request.ErrBadStatus, "PMESimulate should return the request failure")
}

func TestRemoveAPIKey(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.RemoveAPIKey(t.Context(), 0), errAPIKeyIDRequired, "RemoveAPIKey must reject an empty API key ID")

	skipLiveUnlessCanManipulate(t)
	id := uint64(mockAPIKeyID)
	if !mockTests {
		// The key is created for the test, so removing it restores the account's keys
		key, err := e.CreateAPIKey(t.Context(), liveTestAPIKeyRequest())
		require.NoError(t, err, "CreateAPIKey must create an API key to remove")
		id = key.ID
	}
	require.NoError(t, e.RemoveAPIKey(t.Context(), id), "RemoveAPIKey must not error")

	err := newUnavailableServerExchange(t).RemoveAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RemoveAPIKey should return the request failure")
}

func TestRemoveSubaccount(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.RemoveSubaccount(t.Context(), 0), errSubaccountIDRequired, "RemoveSubaccount must reject an empty subaccount ID")

	if !mockTests {
		t.Skip("RemoveSubaccount would remove a subaccount, which live tests cannot recreate")
	}
	require.NoError(t, e.RemoveSubaccount(t.Context(), mockAccountSubaccountID), "RemoveSubaccount must not error")

	err := newUnavailableServerExchange(t).RemoveSubaccount(t.Context(), mockAccountSubaccountID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RemoveSubaccount should return the request failure")
}

func TestResetAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ResetAPIKey(t.Context(), 0)
	require.ErrorIs(t, err, errAPIKeyIDRequired, "ResetAPIKey must reject an empty API key ID")

	skipLiveUnlessCanManipulate(t)
	id := testAPIKeyID(t)
	result, err := e.ResetAPIKey(t.Context(), id)
	require.NoError(t, err, "ResetAPIKey must not error")
	if mockTests {
		assert.Equal(t, testAPIKey(), result, "ResetAPIKey should decode every field")
	} else {
		assert.NotEmpty(t, result.ClientSecret, "ResetAPIKey should return the key's new secret")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ResetAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ResetAPIKey should reject a null result")
	_, err = newUnavailableServerExchange(t).ResetAPIKey(t.Context(), id)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ResetAPIKey should return the request failure")
}

func TestSetAnnouncementAsRead(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetAnnouncementAsRead(t.Context(), 0), errAnnouncementIDRequired, "SetAnnouncementAsRead must reject an empty announcement ID")

	if !mockTests {
		t.Skip("SetAnnouncementAsRead would mark an announcement read, which live tests cannot mark unread")
	}
	require.NoError(t, e.SetAnnouncementAsRead(t.Context(), mockAnnouncementID), "SetAnnouncementAsRead must not error")

	err := newUnavailableServerExchange(t).SetAnnouncementAsRead(t.Context(), mockAnnouncementID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetAnnouncementAsRead should return the request failure")
}

func TestSetDisabledTradingProducts(t *testing.T) {
	t.Parallel()
	err := e.SetDisabledTradingProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetDisabledTradingProducts must reject a nil request")
	err = e.SetDisabledTradingProducts(t.Context(), &SetDisabledTradingProductsRequest{TradingProducts: []string{TradingProductSpots}})
	require.ErrorIs(t, err, errTradingProductsUserIDRequired, "SetDisabledTradingProducts must reject an empty user ID")
	err = e.SetDisabledTradingProducts(t.Context(), &SetDisabledTradingProductsRequest{UserID: mockAccountSubaccountID, TradingProducts: []string{TradingProductSpots, "spot"}})
	require.ErrorIs(t, err, errInvalidTradingProduct, "SetDisabledTradingProducts must reject an undocumented trading product")

	// A nil list is sent as an empty list, as Deribit requires the list
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"jsonrpc":"2.0","method":"private/set_disabled_trading_products","params":{"user_id":7,"trading_products":[]}}`, string(body), "SetDisabledTradingProducts should send a nil list as an empty list")
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":"ok"}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	require.NoError(t, ex.SetDisabledTradingProducts(t.Context(), &SetDisabledTradingProductsRequest{UserID: mockAccountSubaccountID}), "SetDisabledTradingProducts must not error for a nil list")

	if !mockTests {
		t.Skip("SetDisabledTradingProducts would change a subaccount's products, which live tests cannot read back to restore")
	}
	arg := &SetDisabledTradingProductsRequest{UserID: mockAccountSubaccountID, TradingProducts: []string{TradingProductFutures, TradingProductOptions}}
	require.NoError(t, e.SetDisabledTradingProducts(t.Context(), arg), "SetDisabledTradingProducts must not error")

	err = newUnavailableServerExchange(t).SetDisabledTradingProducts(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetDisabledTradingProducts should return the request failure")
}

func TestSetEmailForSubaccount(t *testing.T) {
	t.Parallel()
	err := e.SetEmailForSubaccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetEmailForSubaccount must reject a nil request")
	err = e.SetEmailForSubaccount(t.Context(), &SetEmailForSubaccountRequest{Email: mockSubaccountEmail})
	require.ErrorIs(t, err, errSubaccountIDRequired, "SetEmailForSubaccount must reject an empty subaccount ID")
	err = e.SetEmailForSubaccount(t.Context(), &SetEmailForSubaccountRequest{SubaccountID: mockAccountSubaccountID})
	require.ErrorIs(t, err, errSubaccountEmailRequired, "SetEmailForSubaccount must reject an empty email")

	if !mockTests {
		t.Skip("SetEmailForSubaccount would send a confirmation email and replace a subaccount's email, which live tests cannot undo")
	}
	arg := &SetEmailForSubaccountRequest{SubaccountID: mockAccountSubaccountID, Email: mockSubaccountEmail}
	require.NoError(t, e.SetEmailForSubaccount(t.Context(), arg), "SetEmailForSubaccount must not error")

	err = newUnavailableServerExchange(t).SetEmailForSubaccount(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetEmailForSubaccount should return the request failure")
}

func TestSetEmailLanguage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetEmailLanguage(t.Context(), ""), errEmailLanguageRequired, "SetEmailLanguage must reject an empty language")

	skipLiveUnlessCanManipulate(t)
	language := testEmailLanguage(t)
	require.NoError(t, e.SetEmailLanguage(t.Context(), language), "SetEmailLanguage must not error")

	err := newUnavailableServerExchange(t).SetEmailLanguage(t.Context(), language)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetEmailLanguage should return the request failure")
}

func TestSetMember(t *testing.T) {
	t.Parallel()
	inactive := false
	_, err := e.SetMember(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetMember must reject a nil request")
	_, err = e.SetMember(t.Context(), &SetMemberRequest{ID: mockMemberID, Accounts: []uint64{mockMemberAccountID}})
	require.ErrorIs(t, err, errMemberNameRequired, "SetMember must reject an empty name")
	_, err = e.SetMember(t.Context(), &SetMemberRequest{Name: mockMemberName})
	require.ErrorIs(t, err, errMemberAccountsRequired, "SetMember must reject creating a member without accounts")
	_, err = e.SetMember(t.Context(), &SetMemberRequest{Name: mockMemberName, IsActive: &inactive, Accounts: []uint64{mockMemberAccountID}})
	require.ErrorIs(t, err, errMemberCreatedInactive, "SetMember must reject creating an inactive member")

	skipLiveUnlessCanManipulate(t)
	arg := testSetMemberRequestFor(t)
	result, err := e.SetMember(t.Context(), arg)
	require.NoError(t, err, "SetMember must not error")
	if mockTests {
		assert.Equal(t, testMembers(), result, "SetMember should decode every field")
	} else {
		assert.NotEmpty(t, result.Members, "SetMember should return the account's members")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SetMember(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SetMember should reject a null result")
	_, err = newUnavailableServerExchange(t).SetMember(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetMember should return the request failure")
}

func TestSetSelfTradingConfig(t *testing.T) {
	t.Parallel()
	err := e.SetSelfTradingConfig(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetSelfTradingConfig must reject a nil request")
	err = e.SetSelfTradingConfig(t.Context(), &SetSelfTradingConfigRequest{Mode: "reject_maker"})
	require.ErrorIs(t, err, errInvalidSelfTradingMode, "SetSelfTradingConfig must reject an undocumented mode")

	skipLiveUnlessCanManipulate(t)
	arg := testSetSelfTradingConfigRequestFor(t)
	require.NoError(t, e.SetSelfTradingConfig(t.Context(), arg), "SetSelfTradingConfig must not error")

	err = newUnavailableServerExchange(t).SetSelfTradingConfig(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SetSelfTradingConfig should return the request failure")
}

func TestSimulatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.SimulatePortfolio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SimulatePortfolio must reject a nil request")
	_, err = e.SimulatePortfolio(t.Context(), &SimulatePortfolioRequest{SimulatedPositions: map[string]float64{mockPositionInstrument: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SimulatePortfolio must reject an empty currency")

	skipLiveWithoutCredentials(t)
	result, err := e.SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
	require.NoError(t, err, "SimulatePortfolio must not error")
	if mockTests {
		assert.Equal(t, testSimulatedPortfolio(), result, "SimulatePortfolio should decode every field")
	} else {
		assert.Equal(t, currency.BTC, result.Currency, "SimulatePortfolio should simulate the currency asked for")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SimulatePortfolio should reject a null result")
	_, err = newUnavailableServerExchange(t).SimulatePortfolio(t.Context(), testSimulatePortfolioRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SimulatePortfolio should return the request failure")
}

func TestSimulatePortfolioRequestMarshalJSON(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(&SimulatePortfolioRequest{Currency: currency.BTC})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"currency":"BTC"}`, string(encoded), "SimulatePortfolioRequest should leave out unset positions, so Deribit adds them to the account's")
	addPositions := false
	encoded, err = json.Marshal(&SimulatePortfolioRequest{Currency: currency.BTC, AddPositions: &addPositions, SimulatedPositions: map[string]float64{mockPositionInstrument: -1}})
	require.NoError(t, err, "Marshal must not error with simulated positions alone")
	assert.JSONEq(t, `{"currency":"BTC","add_positions":false,"simulated_positions":{"BTC-PERPETUAL":-1}}`, string(encoded), "SimulatePortfolioRequest should send a false add_positions and the positions as an object")
}

func TestToggleNotificationsFromSubaccount(t *testing.T) {
	t.Parallel()
	err := e.ToggleNotificationsFromSubaccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ToggleNotificationsFromSubaccount must reject a nil request")
	err = e.ToggleNotificationsFromSubaccount(t.Context(), &ToggleNotificationsFromSubaccountRequest{State: true})
	require.ErrorIs(t, err, errSubaccountIDRequired, "ToggleNotificationsFromSubaccount must reject an empty subaccount ID")

	if !mockTests {
		t.Skip("ToggleNotificationsFromSubaccount would change a subaccount's notifications, whose state live tests cannot read back to restore")
	}
	arg := &ToggleNotificationsFromSubaccountRequest{SubaccountID: mockAccountSubaccountID, State: true}
	require.NoError(t, e.ToggleNotificationsFromSubaccount(t.Context(), arg), "ToggleNotificationsFromSubaccount must not error")

	err = newUnavailableServerExchange(t).ToggleNotificationsFromSubaccount(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ToggleNotificationsFromSubaccount should return the request failure")
}

func TestToggleSubaccountLogin(t *testing.T) {
	t.Parallel()
	err := e.ToggleSubaccountLogin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ToggleSubaccountLogin must reject a nil request")
	err = e.ToggleSubaccountLogin(t.Context(), &ToggleSubaccountLoginRequest{State: SubaccountLoginEnable})
	require.ErrorIs(t, err, errSubaccountIDRequired, "ToggleSubaccountLogin must reject an empty subaccount ID")
	err = e.ToggleSubaccountLogin(t.Context(), &ToggleSubaccountLoginRequest{SubaccountID: mockAccountSubaccountID, State: "true"})
	require.ErrorIs(t, err, errInvalidSubaccountLoginState, "ToggleSubaccountLogin must reject an undocumented state")

	skipLiveUnlessCanManipulate(t)
	arg := testToggleSubaccountLoginRequest(t)
	require.NoError(t, e.ToggleSubaccountLogin(t.Context(), arg), "ToggleSubaccountLogin must not error")

	err = newUnavailableServerExchange(t).ToggleSubaccountLogin(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ToggleSubaccountLogin should return the request failure")
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.GetAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAnnouncements must reject a nil request")

	result, err := e.GetAnnouncements(t.Context(), testAnnouncementsRequest())
	require.NoError(t, err, "GetAnnouncements must not error")
	if mockTests {
		assert.Equal(t, testAnnouncements(), result, "GetAnnouncements should decode every field")
	} else {
		require.Len(t, result, mockAnnouncementsCount, "GetAnnouncements must return the announcements asked for")
		assert.False(t, result[0].PublicationTimestamp.Time().After(time.UnixMilli(mockAnnouncementsStartTime)), "GetAnnouncements should return announcements published up to the start time")
	}

	_, err = newUnavailableServerExchange(t).GetAnnouncements(t.Context(), testAnnouncementsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAnnouncements should return the request failure")
}

func TestGetAnnouncementsParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		params *getAnnouncementsParams
		exp    string
	}{
		{params: &getAnnouncementsParams{GetAnnouncementsRequest: &GetAnnouncementsRequest{}}, exp: `{}`},
		{
			params: &getAnnouncementsParams{GetAnnouncementsRequest: testAnnouncementsRequest(), StartTimestamp: mockAnnouncementsStartTime},
			exp:    `{"count":1,"start_timestamp":1791800000000}`,
		},
	} {
		encoded, err := json.Marshal(tc.params)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, tc.exp, string(encoded), "getAnnouncementsParams should send the start time once, in Unix milliseconds, and leave out unset params")
	}
}
