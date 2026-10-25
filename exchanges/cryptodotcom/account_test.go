package cryptodotcom

import (
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

func TestGetUserBalance(t *testing.T) {
	t.Parallel()
	ex, sent := newSignedRecorder(t, "private/user-balance", `{"data":[]}`)
	_, err := ex.GetUserBalance(t.Context())
	require.NoError(t, err, "GetUserBalance must not error")
	params := sent()
	require.Len(t, params, 1, "GetUserBalance must send one request")
	assert.JSONEq(t, `{}`, params[0], "GetUserBalance should send empty params")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/user-balance","code":0,"result":null}`).GetUserBalance(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserBalance should error for a null result")
	_, err = newUnavailableServerExchange(t).GetUserBalance(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserBalance should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetUserBalance(t.Context())
	require.NoError(t, err, "GetUserBalance must not error")
	if mockTests {
		exp := []UserBalanceResponse{{
			InstrumentName:             "USD",
			TotalAvailableBalance:      preciseNumber("4721.05898582"),
			TotalMarginBalance:         preciseNumber("7595.42571782"),
			TotalInitialMargin:         preciseNumber("2874.36673202"),
			TotalMaintenanceMargin:     preciseNumber("1437.18336601"),
			TotalPositionCost:          preciseNumber("14517.54641301"),
			TotalCashBalance:           preciseNumber("7890.00320721"),
			TotalCollateralValue:       preciseNumber("7651.18811483"),
			TotalSessionUnrealisedPNL:  preciseNumber("-55.76239701"),
			TotalSessionRealisedPNL:    preciseNumber("3.21000000"),
			TotalEffectiveLeverage:     1.9040123,
			PositionLimit:              preciseNumber("3000000.00000000"),
			UsedPositionLimit:          preciseNumber("40674.69622001"),
			TotalIsolatedCashBalance:   preciseNumber("0.00000000"),
			TotalPositionInitialMargin: preciseNumber("486.31273202"),
			TotalHaircut:               preciseNumber("2388.054"),
			TotalRiskExposure:          preciseNumber("15003.85914503"),
			TotalBorrow:                preciseNumber("120.50000000"),
			MarginScore:                0.18922107,
			HasRisk:                    true,
			CreditLimits:               []json.RawMessage{},
			PositionBalances: []CollateralBalance{
				{
					InstrumentName:       "CRO",
					Quantity:             preciseNumber("24422.72427884"),
					MarketValue:          preciseNumber("4776.107959969951"),
					CollateralEligible:   true,
					Haircut:              0.5,
					CollateralAmount:     preciseNumber("2388.053979984976"),
					MaxWithdrawalBalance: preciseNumber("24400.00000000"),
					ReservedQuantity:     preciseNumber("22.72427884"),
				},
				{
					InstrumentName:       "USD",
					Quantity:             preciseNumber("3113.50747209"),
					MarketValue:          preciseNumber("3113.50747209"),
					CollateralEligible:   true,
					CollateralAmount:     preciseNumber("3113.50747209"),
					MaxWithdrawalBalance: preciseNumber("3113.50747209"),
					ReservedQuantity:     preciseNumber("12.50000000"),
					HourlyInterestRate:   0.00000571,
				},
			},
			IsolatedPositions: []IsolatedPositionBalance{},
		}}
		assert.Equal(t, exp, result, "GetUserBalance should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetUserBalance should return the account's balance")
	}
}

func TestGetUserBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserBalanceHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserBalanceHistory must refuse a nil request")
	_, err = e.GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{EndTime: time.Unix(0, -1)})
	require.ErrorIs(t, err, errTimeBeforeUnixEpoch, "GetUserBalanceHistory must refuse an end time before 1970")
	_, err = e.GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{Timeframe: "M1"})
	require.ErrorIs(t, err, errInvalidBalanceHistoryTimeframe, "GetUserBalanceHistory must refuse an unknown timeframe")
	for _, req := range []*UserBalanceHistoryRequest{{Limit: 31}, {Timeframe: BalanceHistoryTimeframeDaily, Limit: 31}, {Timeframe: BalanceHistoryTimeframeHourly, Limit: 121}} {
		_, err = e.GetUserBalanceHistory(t.Context(), req)
		require.ErrorIsf(t, err, errInvalidCount, "GetUserBalanceHistory must refuse %d %s points", req.Limit, req.Timeframe)
	}

	ex, sent := newSignedRecorder(t, "private/user-balance-history", `{"instrument_name":"USD","data":[]}`)
	_, err = ex.GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{Timeframe: BalanceHistoryTimeframeHourly, EndTime: time.Unix(0, 1629565200123456789), Limit: 120})
	require.NoError(t, err, "GetUserBalanceHistory must not error with every param")
	_, err = ex.GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{})
	require.NoError(t, err, "GetUserBalanceHistory must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetUserBalanceHistory must send two requests")
	assert.JSONEq(t, `{"timeframe":"H1","end_time":"1629565200123456789","limit":"120"}`, params[0], "GetUserBalanceHistory should send every param, the end time in nanoseconds")
	assert.JSONEq(t, `{}`, params[1], "GetUserBalanceHistory should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/user-balance-history","code":0,"result":null}`).GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserBalanceHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserBalanceHistory should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetUserBalanceHistory(t.Context(), &UserBalanceHistoryRequest{Timeframe: BalanceHistoryTimeframeDaily, EndTime: time.Unix(0, 1629565200000000000), Limit: 2})
	require.NoError(t, err, "GetUserBalanceHistory must not error")
	if mockTests {
		exp := &UserBalanceHistoryResponse{
			InstrumentName: "USD",
			Balances: []BalanceHistoryPoint{
				{Time: types.Time(time.UnixMilli(1629392400000)), TotalCashBalance: preciseNumber("798.354209")},
				{Time: types.Time(time.UnixMilli(1629478800000)), TotalCashBalance: preciseNumber("811.621851")},
			},
		}
		assert.Equal(t, exp, result, "GetUserBalanceHistory should decode every field")
	} else {
		assert.NotEmpty(t, result.InstrumentName, "GetUserBalanceHistory should name the balance's currency")
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccounts must refuse a nil request")

	ex, sent := newSignedRecorder(t, "private/get-accounts", `{"master_account":{},"sub_account_list":[]}`)
	_, err = ex.GetAccounts(t.Context(), &AccountsRequest{PageSize: 30, Page: 2})
	require.NoError(t, err, "GetAccounts must not error with every param")
	_, err = ex.GetAccounts(t.Context(), &AccountsRequest{})
	require.NoError(t, err, "GetAccounts must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetAccounts must send two requests")
	assert.JSONEq(t, `{"page_size":"30","page":"2"}`, params[0], "GetAccounts should send every param")
	assert.JSONEq(t, `{}`, params[1], "GetAccounts should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-accounts","code":0,"result":null}`).GetAccounts(t.Context(), &AccountsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccounts should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAccounts(t.Context(), &AccountsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccounts should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAccounts(t.Context(), &AccountsRequest{PageSize: 20})
	require.NoError(t, err, "GetAccounts must not error")
	if mockTests {
		exp := &AccountsResponse{
			ThisAccount: CurrentAccount{UUID: "243d3f39-b193-4eb9-1d60-e98f2fc17707", Hierarchy: AccountHierarchyMaster},
			MasterAccount: Account{
				UUID:                           "243d3f39-b193-4eb9-1d60-e98f2fc17707",
				Hierarchy:                      AccountHierarchyMaster,
				MasterAccountUUID:              "291879ae-b769-4eb3-4d75-3366ebee7dd6",
				MarginAccountUUID:              "69c9ab41-5b95-4d75-b769-e45f2fc16507",
				Enabled:                        true,
				Tradable:                       true,
				Email:                          "trader@example.com",
				MobileNumber:                   "+61491570156",
				CountryCode:                    "AU",
				Address:                        "1 Example Street, Sydney NSW 2000",
				MarginAccess:                   AccessLevelDefault,
				DerivativesAccess:              AccessLevelDisabled,
				CreateTime:                     types.Time(time.UnixMilli(1620962543792)),
				UpdateTime:                     types.Time(time.UnixMilli(1622019525960)),
				TwoFactorAuthenticationEnabled: true,
				KYCLevel:                       "ADVANCED",
			},
			SubAccounts: []Account{{
				UUID:              "69c9ab41-5b95-4d75-b769-e45f2fc16507",
				Hierarchy:         AccountHierarchySubAccount,
				MasterAccountUUID: "291879ae-b769-4eb3-4d75-3366ebee7dd6",
				MarginAccountUUID: "69c9ab41-5b95-4d75-b769-e45f2fc16507",
				Label:             "Sub Account",
				Enabled:           true,
				Tradable:          true,
				Name:              "Hedging",
				Email:             "hedging@example.com",
				MobileNumber:      "+61491570157",
				CountryCode:       "AU",
				Address:           "2 Example Street, Sydney NSW 2000",
				MarginAccess:      AccessLevelDefault,
				DerivativesAccess: AccessLevelDisabled,
				CreateTime:        types.Time(time.UnixMilli(1621034103478)),
				UpdateTime:        types.Time(time.UnixMilli(1621034103478)),
				KYCLevel:          "ADVANCED",
			}},
		}
		assert.Equal(t, exp, result, "GetAccounts should decode every field")
	} else {
		assert.NotEmpty(t, result.MasterAccount.UUID, "GetAccounts should return the master account's UUID")
	}
}

func TestCreateSubAccountTransfer(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *SubAccountTransferRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &SubAccountTransferRequest{ToAccountUUID: "b", Currency: currency.CRO, Amount: 1}, err: errFromAccountRequired},
		{req: &SubAccountTransferRequest{FromAccountUUID: "a", Currency: currency.CRO, Amount: 1}, err: errToAccountRequired},
		{req: &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
		{req: &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Currency: currency.CRO}, err: errInvalidAmount},
		{req: &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Currency: currency.CRO, Amount: -1}, err: errInvalidAmount},
	} {
		require.ErrorIsf(t, e.CreateSubAccountTransfer(t.Context(), tc.req), tc.err, "CreateSubAccountTransfer must refuse %+v", tc.req)
	}

	ex, sent := newSignedRecorder(t, "private/create-subaccount-transfer", `null`)
	require.NoError(t, ex.CreateSubAccountTransfer(t.Context(), &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Currency: currency.NewCode("cro"), Amount: 0.00000001}), "CreateSubAccountTransfer must not error")
	params := sent()
	require.Len(t, params, 1, "CreateSubAccountTransfer must send one request")
	assert.JSONEq(t, `{"from":"a","to":"b","currency":"CRO","amount":"0.00000001"}`, params[0], "CreateSubAccountTransfer should send every param, the currency in upper case")

	err := newUnavailableServerExchange(t).CreateSubAccountTransfer(t.Context(), &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Currency: currency.CRO, Amount: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateSubAccountTransfer should return the request's failure")

	skipLiveUnlessCanManipulate(t)
	req := &SubAccountTransferRequest{FromAccountUUID: "243d3f39-b193-4eb9-1d60-e98f2fc17707", ToAccountUUID: "69c9ab41-5b95-4d75-b769-e45f2fc16507", Currency: currency.CRO, Amount: 500}
	if !mockTests {
		accounts, err := e.GetAccounts(t.Context(), &AccountsRequest{})
		require.NoError(t, err, "GetAccounts must not error")
		if len(accounts.SubAccounts) == 0 {
			t.Skip("the account has no sub-account to transfer to")
		}
		// Live, one USD goes to the first sub-account and back, leaving the balances as they were
		req = &SubAccountTransferRequest{FromAccountUUID: accounts.MasterAccount.UUID, ToAccountUUID: accounts.SubAccounts[0].UUID, Currency: currency.USD, Amount: 1}
	}
	require.NoError(t, e.CreateSubAccountTransfer(t.Context(), req), "CreateSubAccountTransfer must not error")
	if !mockTests {
		back := &SubAccountTransferRequest{FromAccountUUID: req.ToAccountUUID, ToAccountUUID: req.FromAccountUUID, Currency: req.Currency, Amount: req.Amount}
		assert.NoError(t, e.CreateSubAccountTransfer(t.Context(), back), "CreateSubAccountTransfer should transfer the funds back")
	}
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	ex, sent := newSignedRecorder(t, "private/get-subaccount-balances", `{"data":[]}`)
	_, err := ex.GetSubAccountBalances(t.Context())
	require.NoError(t, err, "GetSubAccountBalances must not error")
	params := sent()
	require.Len(t, params, 1, "GetSubAccountBalances must send one request")
	assert.JSONEq(t, `{}`, params[0], "GetSubAccountBalances should send empty params")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-subaccount-balances","code":0,"result":null}`).GetSubAccountBalances(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetSubAccountBalances should error for a null result")
	_, err = newUnavailableServerExchange(t).GetSubAccountBalances(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSubAccountBalances should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetSubAccountBalances(t.Context())
	require.NoError(t, err, "GetSubAccountBalances must not error")
	if mockTests {
		exp := []UserBalanceResponse{
			{
				InstrumentName:            "USD",
				TotalAvailableBalance:     preciseNumber("0.00000000"),
				TotalMarginBalance:        preciseNumber("0.00000000"),
				TotalInitialMargin:        preciseNumber("0.00000000"),
				TotalMaintenanceMargin:    preciseNumber("0.00000000"),
				TotalPositionCost:         preciseNumber("0.00000000"),
				TotalCashBalance:          preciseNumber("0.00000000"),
				TotalCollateralValue:      preciseNumber("0.00000000"),
				TotalSessionUnrealisedPNL: preciseNumber("0.00000000"),
				TotalSessionRealisedPNL:   preciseNumber("0.00000000"),
				PositionLimit:             preciseNumber("3000000.00000000"),
				UsedPositionLimit:         preciseNumber("0.00000000"),
				TotalIsolatedCashBalance:  preciseNumber("0.00000000"),
				PositionBalances:          []CollateralBalance{},
				IsolatedPositions:         []IsolatedPositionBalance{},
				SubAccountID:              "a0d206a1-6b06-47c5-9cd3-8bc6ef0915c5",
			},
			{
				InstrumentName:            "USD",
				TotalAvailableBalance:     preciseNumber("20922.62250000"),
				TotalMarginBalance:        preciseNumber("20922.62250000"),
				TotalInitialMargin:        preciseNumber("0.00000000"),
				TotalMaintenanceMargin:    preciseNumber("0.00000000"),
				TotalPositionCost:         preciseNumber("0.00000000"),
				TotalCashBalance:          preciseNumber("22018.55000000"),
				TotalCollateralValue:      preciseNumber("20922.62250000"),
				TotalSessionUnrealisedPNL: preciseNumber("0.00000000"),
				TotalSessionRealisedPNL:   preciseNumber("0.00000000"),
				PositionLimit:             preciseNumber("3000000.00000000"),
				UsedPositionLimit:         preciseNumber("0.00000000"),
				TotalIsolatedCashBalance:  preciseNumber("8.96326705"),
				PositionBalances: []CollateralBalance{
					{InstrumentName: "BTC", Quantity: preciseNumber("1.0000000000"), MarketValue: preciseNumber("21918.5500000000"), CollateralEligible: true, Haircut: 0.55, CollateralAmount: preciseNumber("21918.0000000000"), MaxWithdrawalBalance: preciseNumber("1.0000000000")},
					{InstrumentName: "USD", Quantity: preciseNumber("100.00000000"), MarketValue: preciseNumber("100.00000000"), CollateralEligible: true, Haircut: 1, CollateralAmount: preciseNumber("99.00000000"), MaxWithdrawalBalance: preciseNumber("0.00000000")},
				},
				IsolatedPositions: []IsolatedPositionBalance{{
					InstrumentName:             "USD",
					TotalAvailableBalance:      preciseNumber("7.80487225"),
					TotalMarginBalance:         preciseNumber("8.96026705"),
					TotalInitialMargin:         preciseNumber("1.09620001"),
					TotalMaintenanceMargin:     preciseNumber("0.27405"),
					TotalPositionCost:          preciseNumber("10.959"),
					TotalCashBalance:           preciseNumber("8.96326705"),
					TotalCollateralValue:       preciseNumber("8.96326705"),
					TotalSessionUnrealisedPNL:  preciseNumber("-0.003"),
					TotalSessionRealisedPNL:    preciseNumber("-0.043"),
					TotalEffectiveLeverage:     1.231537,
					PositionLimit:              preciseNumber("89.01072259"),
					UsedPositionLimit:          preciseNumber("10.962"),
					TotalPositionInitialMargin: preciseNumber("0.5481"),
					TotalHaircut:               preciseNumber("0"),
					PositionBalances: []CollateralBalance{{
						InstrumentName:       "USD",
						Quantity:             preciseNumber("8.9632670590686"),
						MarketValue:          preciseNumber("8.96326705"),
						CollateralEligible:   true,
						CollateralAmount:     preciseNumber("8.96326705"),
						MaxWithdrawalBalance: preciseNumber("7.80487225"),
						ReservedQuantity:     preciseNumber("0.00000000"),
					}},
					IsolationID:   "19848526",
					Leverage:      10,
					IsolationType: IsolationTypeIsolatedMargin,
				}},
				SubAccountID: "507d3f7d-37c3-4a09-9076-b83c2fcbb638",
			},
		}
		assert.Equal(t, exp, result, "GetSubAccountBalances should decode every field, a quoted collateral eligibility among them")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].SubAccountID, "GetSubAccountBalances should name sub-account %d", i)
		}
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositions must refuse a nil request")

	ex, sent := newSignedRecorder(t, "private/get-positions", `{"data":[]}`)
	_, err = ex.GetPositions(t.Context(), &PositionsRequest{InstrumentName: "ETHUSD-PERP"})
	require.NoError(t, err, "GetPositions must not error with every param")
	_, err = ex.GetPositions(t.Context(), &PositionsRequest{})
	require.NoError(t, err, "GetPositions must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetPositions must send two requests")
	assert.JSONEq(t, `{"instrument_name":"ETHUSD-PERP"}`, params[0], "GetPositions should send every param")
	assert.JSONEq(t, `{}`, params[1], "GetPositions should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-positions","code":0,"result":null}`).GetPositions(t.Context(), &PositionsRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPositions should error for a null result")
	_, err = newUnavailableServerExchange(t).GetPositions(t.Context(), &PositionsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositions should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetPositions(t.Context(), &PositionsRequest{InstrumentName: "BTCUSD-PERP"})
	require.NoError(t, err, "GetPositions must not error")
	if mockTests {
		exp := []PositionResponse{
			{
				AccountID:        "858dbc8b-22fd-49fa-bff4-d342d98a8acb",
				InstrumentName:   "BTCUSD-PERP",
				InstrumentType:   InstrumentTypePerpetualSwap,
				Quantity:         preciseNumber("-0.1984"),
				Cost:             preciseNumber("-10159.573500"),
				OpenPositionPNL:  preciseNumber("-497.743736"),
				OpenPositionCost: preciseNumber("-10159.352200"),
				SessionPNL:       preciseNumber("2.236145"),
				UpdateTime:       types.Time(time.UnixMilli(1613552240770)),
			},
			{
				AccountID:        "858dbc8b-22fd-49fa-bff4-d342d98a8acb",
				InstrumentName:   "BTCUSD-PERP",
				InstrumentType:   InstrumentTypePerpetualSwap,
				Quantity:         preciseNumber("0.0002"),
				Cost:             preciseNumber("10.959000"),
				OpenPositionPNL:  preciseNumber("-0.003000"),
				OpenPositionCost: preciseNumber("10.959000"),
				SessionPNL:       preciseNumber("-0.043000"),
				UpdateTime:       types.Time(time.UnixMilli(1613552240771)),
				IsolationID:      "19848526",
				IsolationType:    IsolationTypeIsolatedMargin,
			},
		}
		assert.Equal(t, exp, result, "GetPositions should decode every field")
	} else {
		for i := range result {
			assert.Equalf(t, "BTCUSD-PERP", result[i].InstrumentName, "GetPositions should only return BTCUSD-PERP positions, not %s", result[i].InstrumentName)
		}
	}
}

func TestGetFeeCreditBalances(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeCreditBalances(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeCreditBalances must refuse a nil request")

	ex, sent := newSignedRecorder(t, "private/get-fee-credit-balances", `{"fee_credits":[]}`)
	_, err = ex.GetFeeCreditBalances(t.Context(), &FeeCreditBalancesRequest{Page: 1, PageSize: 50})
	require.NoError(t, err, "GetFeeCreditBalances must not error with every param")
	_, err = ex.GetFeeCreditBalances(t.Context(), &FeeCreditBalancesRequest{})
	require.NoError(t, err, "GetFeeCreditBalances must not error without params")
	params := sent()
	require.Len(t, params, 2, "GetFeeCreditBalances must send two requests")
	assert.JSONEq(t, `{"page":1,"page_size":50}`, params[0], "GetFeeCreditBalances should send every param, as the integers the docs type them")
	assert.JSONEq(t, `{}`, params[1], "GetFeeCreditBalances should send no params when none are set")

	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-fee-credit-balances","code":0,"result":null}`).GetFeeCreditBalances(t.Context(), &FeeCreditBalancesRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFeeCreditBalances should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFeeCreditBalances(t.Context(), &FeeCreditBalancesRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFeeCreditBalances should return the request's failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetFeeCreditBalances(t.Context(), &FeeCreditBalancesRequest{PageSize: 200})
	require.NoError(t, err, "GetFeeCreditBalances must not error")
	if mockTests {
		exp := []FeeCreditResponse{{
			ExpiryDate:        types.Time(time.UnixMilli(1735689540000)),
			CampaignNames:     "Campaign1, Campaign2",
			DistributedAmount: preciseNumber("200.00000000"),
			AvailableBalance:  preciseNumber("150.00000000"),
			CampaignDistributions: []FeeCreditDistribution{
				{CampaignName: "Campaign1", DistributionDate: types.Time(time.UnixMilli(1704067140000)), DistributedAmount: preciseNumber("100.00000000")},
				{CampaignName: "Campaign2", DistributionDate: types.Time(time.UnixMilli(1706745540000)), DistributedAmount: preciseNumber("100.00000000")},
			},
		}}
		assert.Equal(t, exp, result, "GetFeeCreditBalances should decode every field")
	} else {
		for i := range result {
			assert.Falsef(t, result[i].ExpiryDate.Time().IsZero(), "GetFeeCreditBalances should return fee credit %d's expiry date", i)
		}
	}
}

// TestPositionAndBalanceChannelFields proves PositionResponse and CollateralBalance decode the fields only the user
// channels send, so the user.positions, user.account_risk and user.position_balance channels can reuse them
func TestPositionAndBalanceChannelFields(t *testing.T) {
	t.Parallel()
	var position PositionResponse
	require.NoError(t, json.Unmarshal([]byte(`{
		"account_id": "858dbc8b-22fd-49fa-bff4-d342d98a8acb",
		"instrument_name": "BTCUSD-PERP",
		"type": "PERPETUAL_SWAP",
		"quantity": "0.0002",
		"cost": "10.959",
		"open_position_pnl": "-0.003",
		"open_pos_cost": "10.958",
		"session_pnl": "-0.043",
		"update_timestamp_ms": 1613552240771,
		"isolation_id": "19848526",
		"isolation_type": "ISOLATED_MARGIN",
		"session_unrealized_pnl": "-0.004",
		"pos_initial_margin": "0.5481",
		"pos_maintenance_margin": "0.27405",
		"market_value": "10.956",
		"mark_price": "54780",
		"target_leverage": "10",
		"leverage": "9.5",
		"liquidation_price": "50120.5",
		"isolated_margin_balance": "8.96026705",
		"effective_leverage": "1.231537"
	}`), &position), "Unmarshal must not error for a user.positions entry")
	exp := PositionResponse{
		AccountID:                 "858dbc8b-22fd-49fa-bff4-d342d98a8acb",
		InstrumentName:            "BTCUSD-PERP",
		InstrumentType:            InstrumentTypePerpetualSwap,
		Quantity:                  preciseNumber("0.0002"),
		Cost:                      preciseNumber("10.959"),
		OpenPositionPNL:           preciseNumber("-0.003"),
		OpenPositionCost:          preciseNumber("10.958"),
		SessionPNL:                preciseNumber("-0.043"),
		UpdateTime:                types.Time(time.UnixMilli(1613552240771)),
		IsolationID:               "19848526",
		IsolationType:             IsolationTypeIsolatedMargin,
		SessionUnrealisedPNL:      preciseNumber("-0.004"),
		PositionInitialMargin:     preciseNumber("0.5481"),
		PositionMaintenanceMargin: preciseNumber("0.27405"),
		MarketValue:               preciseNumber("10.956"),
		MarkPrice:                 54780,
		TargetLeverage:            10,
		Leverage:                  9.5,
		LiquidationPrice:          50120.5,
		IsolatedMarginBalance:     preciseNumber("8.96026705"),
		EffectiveLeverage:         1.231537,
	}
	assert.Equal(t, exp, position, "PositionResponse should decode every field of a user.positions entry")

	var balance CollateralBalance
	require.NoError(t, json.Unmarshal([]byte(`{"instrument_name":"USD","quantity":"8.96326705","update_timestamp_ms":1613552240771}`), &balance), "Unmarshal must not error for a user.position_balance balance")
	assert.Equal(t, CollateralBalance{InstrumentName: "USD", Quantity: preciseNumber("8.96326705"), UpdateTime: types.Time(time.UnixMilli(1613552240771))}, balance, "CollateralBalance should decode every field of a user.position_balance balance")
}
