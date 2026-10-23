package coinbase

import (
	"net/http"
	"slices"
	"sync"
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
	// testDerivativesAccountInstrument is the instrument the documented position and leverage requests name
	testDerivativesAccountInstrument = "BTC-PERPETUAL"
	// testDerivativesMarginModelUserID is the account whose margin model the documented change dry runs
	testDerivativesMarginModelUserID = 3
	// testDerivativesAccessLogCount is the number of entries the documented access log request asks for
	testDerivativesAccessLogCount = 3
	// testDerivativesTransactionLogStart and testDerivativesTransactionLogEnd, in Unix milliseconds, bound the
	// documented transaction log request, which asks for every entry the documented example holds
	testDerivativesTransactionLogStart = 1613657734000
	testDerivativesTransactionLogEnd   = 1613660407000
	testDerivativesTransactionLogCount = 7
	// testDerivativesAnnouncementsStart, in Unix milliseconds, and testDerivativesAnnouncementsCount select the
	// announcement the announcements request returns, which is recorded live and still returned live
	testDerivativesAnnouncementsStart = 1791800000000
	testDerivativesAnnouncementsCount = 1
	// testDerivativesAccountIsolatedSubaccountID is the isolated margin subaccount the isolated positions guide's
	// examples name
	testDerivativesAccountIsolatedSubaccountID = 123456
)

// testDerivativesChangeMarginModelRequest is the documented margin model change, as a dry run, which the recording
// sends too
func testDerivativesChangeMarginModelRequest() *DerivativesChangeMarginModelRequest {
	return &DerivativesChangeMarginModelRequest{UserID: testDerivativesMarginModelUserID, MarginModel: DerivativesMarginModelCrossPortfolio, DryRun: true}
}

// liveDerivativesChangeMarginModelRequest dry runs a change of the account's own margin model, which changes nothing
func liveDerivativesChangeMarginModelRequest() *DerivativesChangeMarginModelRequest {
	return &DerivativesChangeMarginModelRequest{MarginModel: DerivativesMarginModelCrossStandard, DryRun: true}
}

// testDerivativesTransactionLogRequest is the documented transaction log request, which the recording sends too
func testDerivativesTransactionLogRequest() *DerivativesGetTransactionLogRequest {
	return &DerivativesGetTransactionLogRequest{
		Currency:  currency.BTC,
		StartTime: time.UnixMilli(testDerivativesTransactionLogStart),
		EndTime:   time.UnixMilli(testDerivativesTransactionLogEnd),
		Count:     testDerivativesTransactionLogCount,
	}
}

// liveDerivativesTransactionLogRequest requests the last week's USDC transaction log
func liveDerivativesTransactionLogRequest() *DerivativesGetTransactionLogRequest {
	return &DerivativesGetTransactionLogRequest{Currency: currency.USDC, StartTime: time.Now().Add(-7 * 24 * time.Hour), EndTime: time.Now(), Count: testDerivativesTransactionLogCount}
}

// testDerivativesSimulatePortfolioRequest is the documented portfolio simulation, adding a simulated position in the
// documented instrument
func testDerivativesSimulatePortfolioRequest() *DerivativesSimulatePortfolioRequest {
	return &DerivativesSimulatePortfolioRequest{Currency: currency.BTC, AddPositions: new(true), SimulatedPositions: map[string]float64{testDerivativesAccountInstrument: 1}}
}

// testDerivativesAnnouncementsRequest selects the recorded announcement
func testDerivativesAnnouncementsRequest() *DerivativesGetAnnouncementsRequest {
	return &DerivativesGetAnnouncementsRequest{StartTime: time.UnixMilli(testDerivativesAnnouncementsStart), Count: testDerivativesAnnouncementsCount}
}

// newDerivativesAccountParamsServer returns an exchange whose gateway requests are answered with a result, and a
// function returning the params of every request it has received, for request shapes no fixture records
func newDerivativesAccountParamsServer(t *testing.T, result string) (ex *Exchange, sent func() []string) {
	t.Helper()
	var mu sync.Mutex
	var params []string
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if !ok {
			return
		}
		mu.Lock()
		params = append(params, string(req.Params))
		mu.Unlock()
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":`+result+`}`)
	}))
	return ex, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(params)
	}
}

// testDerivativesMarginModelChanges is the documented margin model change's effect
func testDerivativesMarginModelChanges() []DerivativesMarginModelChangeResponse {
	return []DerivativesMarginModelChangeResponse{
		{
			Currency: currency.NewCode("eth"),
		},
		{
			Currency: currency.NewCode("btc"),
			OldState: DerivativesMarginModelState{
				AvailableBalance:      0.553590509,
				InitialMarginRate:     0.45407615,
				MaintenanceMarginRate: 0.02862727,
			},
			NewState: DerivativesMarginModelState{
				AvailableBalance:      0.98106428,
				InitialMarginRate:     0.03252245,
				MaintenanceMarginRate: 0.02710204,
			},
		},
	}
}

// testDerivativesAccessLog is the documented access log page
func testDerivativesAccessLog() *DerivativesAccessLogResponse {
	return &DerivativesAccessLogResponse{
		RecordsTotal: 34,
		Entries: []DerivativesAccessLogEntry{
			{
				ID:        45,
				Timestamp: types.Time(time.UnixMilli(1575876682576)),
				IP:        "127.0.0.1",
				Country:   "Local Country",
				City:      "Local Town",
				Result:    "success",
			},
			{
				ID:        44,
				Timestamp: types.Time(time.UnixMilli(1575876459309)),
				IP:        "127.0.0.1",
				Country:   "Local Country",
				City:      "Local Town",
				Result:    "success",
			},
			{
				ID:        43,
				Timestamp: types.Time(time.UnixMilli(1575546252774)),
				IP:        "127.0.0.1",
				Country:   "Local Country",
				City:      "Local Town",
				Result:    "disabled_tfa",
			},
		},
	}
}

// testDerivativesAccountLimits is the documented rate limits every documented account summary holds
func testDerivativesAccountLimits() DerivativesAccountLimits {
	return DerivativesAccountLimits{
		NonMatchingEngine: DerivativesAccountRateLimit{
			Rate:  1000,
			Burst: 1500,
		},
		MatchingEngine: DerivativesMatchingEngineLimits{
			Spot: DerivativesAccountRateLimit{
				Rate:  200,
				Burst: 250,
			},
			CancelAll: DerivativesAccountRateLimit{
				Rate:  200,
				Burst: 250,
			},
			Trading: map[string]DerivativesAccountRateLimit{
				"total": {
					Rate:  200,
					Burst: 250,
				},
			},
			MaximumQuotes: DerivativesAccountRateLimit{
				Rate:  500,
				Burst: 500,
			},
			MaximumMassQuotes: DerivativesAccountRateLimit{
				Rate:  10,
				Burst: 10,
			},
			GuaranteedMassQuotes: DerivativesAccountRateLimit{
				Rate:  2,
				Burst: 2,
			},
		},
	}
}

// testDerivativesBTCAccountSummary is the documented BTC summary the documented account summaries hold
func testDerivativesBTCAccountSummary() DerivativesAccountSummary {
	return DerivativesAccountSummary{
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
		OptionsGammaByIndex: map[string]float64{
			"btc_usd": 0.00001,
		},
		OptionsVegaByIndex: map[string]float64{
			"btc_usd": 0.0858,
		},
		DeltaTotal:          31.602958,
		ProjectedDeltaTotal: 32.613978,
		DeltaTotalByIndex: map[string]float64{
			"btc_usd": 31.594357699,
		},
		MarginModel: "segregated_sm",
		Fees: map[string]map[string]DerivativesAccountFee{
			"btc_usd": {
				"future": {
					Default: DerivativesAccountFeeRates{
						Type:  "fixed",
						Maker: -0.0001,
						Taker: 0.00035000000000000005,
					},
					BlockTrade: 0.3,
				},
				"option": {
					Default: DerivativesAccountFeeRates{
						Type:  "relative",
						Maker: 0.625,
						Taker: 0.625,
					},
					BlockTrade: 0.625,
				},
				"perpetual": {
					Default: DerivativesAccountFeeRates{
						Type:  "fixed",
						Maker: -0.0001,
						Taker: 0.00035000000000000005,
					},
					BlockTrade: 0.3,
				},
			},
		},
		Limits: testDerivativesAccountLimits(),
	}
}

// testDerivativesETHAccountSummary is the documented ETH summary the documented account summaries hold
func testDerivativesETHAccountSummary() DerivativesAccountSummary {
	return DerivativesAccountSummary{
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
		DeltaTotalByIndex: map[string]float64{
			"eth_usd": 0,
		},
		MarginModel: "segregated_sm",
		Fees: map[string]map[string]DerivativesAccountFee{
			"eth_usd": {
				"future": {
					Default: DerivativesAccountFeeRates{
						Type:  "fixed",
						Maker: -0.00005,
						Taker: 0.00025,
					},
					BlockTrade: 0.2,
				},
				"option": {
					Default: DerivativesAccountFeeRates{
						Type:  "relative",
						Maker: 0.5,
						Taker: 0.5,
					},
					BlockTrade: 0.5,
				},
				"perpetual": {
					Default: DerivativesAccountFeeRates{
						Type:  "fixed",
						Maker: -0.00005,
						Taker: 0.00025,
					},
					BlockTrade: 0.2,
				},
			},
		},
		Limits: testDerivativesAccountLimits(),
	}
}

// testDerivativesAccountSummaries is the documented extended account summaries, with the isolated margin subaccount
// the recording adds holding the documented ETH summary
func testDerivativesAccountSummaries() *DerivativesAccountSummariesResponse {
	return &DerivativesAccountSummariesResponse{
		ID:                          10,
		Type:                        "main",
		Username:                    "user",
		SystemName:                  "user",
		Email:                       "user@example.com",
		CreationTimestamp:           types.Time(time.UnixMilli(1687352432143)),
		SelfTradingRejectMode:       "cancel_maker",
		BlockRFQSelfMatchPrevention: true,
		Summaries:                   []DerivativesAccountSummary{testDerivativesBTCAccountSummary(), testDerivativesETHAccountSummary()},
		IsolatedAccountSummaries: []DerivativesIsolatedAccountSummaries{
			{
				ID:        testDerivativesAccountIsolatedSubaccountID,
				Summaries: []DerivativesAccountSummary{testDerivativesETHAccountSummary()},
			},
		},
	}
}

// testDerivativesAccountSummary is the documented extended BTC account summary, which also sends the options' theta
// by index
func testDerivativesAccountSummary() *DerivativesAccountSummaryResponse {
	summary := testDerivativesBTCAccountSummary()
	summary.OptionsThetaByIndex = map[string]float64{"btc_usd": 15.97071}
	return &DerivativesAccountSummaryResponse{
		DerivativesAccountSummary:   summary,
		ID:                          10,
		Type:                        "main",
		Username:                    "user",
		SystemName:                  "user",
		Email:                       "user@example.com",
		CreationTimestamp:           types.Time(time.UnixMilli(1687352432143)),
		SelfTradingRejectMode:       "cancel_maker",
		BlockRFQSelfMatchPrevention: true,
	}
}

// testDerivativesLeverages is the documented leverage of the documented instrument
func testDerivativesLeverages() []DerivativesLeverageResponse {
	return []DerivativesLeverageResponse{
		{
			InstrumentName:  "BTC-PERPETUAL",
			Leverage:        25,
			MaximumLeverage: 100,
		},
	}
}

// testDerivativesPMEParams is the documented portfolio margin restrictions
func testDerivativesPMEParams() *DerivativesPMEParamsResponse {
	return &DerivativesPMEParamsResponse{
		Enabled:                     true,
		MaximumFutureOpenOrders:     200,
		MaximumFutureOpenOrdersSize: 1000000,
		MaximumOptionOpenOrders:     200,
		MaximumOptionOpenOrdersSize: 500,
		MaximumRiskReducingOrders:   5,
		MaximumTotalOpenOrders:      300,
	}
}

// testDerivativesPosition is the documented position in the documented instrument
func testDerivativesPosition() *DerivativesPositionResponse {
	return &DerivativesPositionResponse{
		InstrumentName:   "BTC-PERPETUAL",
		Kind:             "future",
		UserID:           1234,
		Direction:        "buy",
		MarkPrice:        3556.62,
		IndexPrice:       3555.86,
		SettlementPrice:  3555.44,
		OpenOrdersMargin: 0.000165889,
		Leverage:         100,
		InterestValue:    1.7362511643080387,
	}
}

// testDerivativesPositions is the documented BTC future positions
func testDerivativesPositions() []DerivativesPositionResponse {
	return []DerivativesPositionResponse{
		{
			InstrumentName:     "BTC-PERPETUAL",
			Kind:               "future",
			UserID:             1234,
			Direction:          "buy",
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
			Leverage:           34,
			TotalProfitLoss:    0.000032781,
			RealisedProfitLoss: -0.000000009,
			RealisedFunding:    -0.00000001,
			InterestValue:      1.7362511643080387,
		},
	}
}

// testDerivativesIsolatedPositions is the isolated positions guide's documented positions in every currency,
// including the isolated margin subaccounts'
func testDerivativesIsolatedPositions() []DerivativesPositionResponse {
	return []DerivativesPositionResponse{
		{
			InstrumentName: "SOL_USDC-PERPETUAL",
			UserID:         testDerivativesAccountIsolatedSubaccountID,
			Isolated:       true,
			MainUserID:     1000,
			Direction:      "buy",
			Size:           1000,
			SizeCurrency:   10,
		},
	}
}

// testDerivativesSubaccounts is the documented accounts with their portfolios
func testDerivativesSubaccounts() []DerivativesSubaccountResponse {
	return []DerivativesSubaccountResponse{
		{
			ID:                      2,
			Type:                    "main",
			Username:                "user_1",
			SystemName:              "user_1",
			Email:                   "user_AAA@email.com",
			LoginEnabled:            true,
			IsPassword:              true,
			SecurityKeysAssignments: []string{},
			MarginModel:             "segregated_sm",
			Portfolio: map[string]DerivativesSubaccountPortfolio{
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
			ID:                      7,
			Type:                    "subaccount",
			Username:                "user_1_1",
			SystemName:              "user_1_1",
			Email:                   "user_AAA@gmail.com",
			IsPassword:              true,
			SecurityKeysAssignments: []string{},
			MarginModel:             "cross_pm",
			Portfolio: map[string]DerivativesSubaccountPortfolio{
				"btc": {
					Currency: currency.NewCode("btc"),
				},
				"eth": {
					Currency: currency.NewCode("eth"),
				},
			},
		},
	}
}

// testDerivativesIsolatedSubaccounts is the isolated positions guide's documented accounts, including an isolated
// margin subaccount
func testDerivativesIsolatedSubaccounts() []DerivativesSubaccountResponse {
	return []DerivativesSubaccountResponse{
		{
			ID:   1000,
			Type: "main",
		},
		{
			ID:                       testDerivativesAccountIsolatedSubaccountID,
			Type:                     "subaccount",
			Isolated:                 true,
			IsolatedMarginInstrument: "SOL_USDC-PERPETUAL",
		},
	}
}

// testDerivativesSubaccountsDetails is the documented accounts' positions, with the documented open order the
// recording adds, as the request asks for open orders
func testDerivativesSubaccountsDetails() []DerivativesSubaccountDetailsResponse {
	return []DerivativesSubaccountDetailsResponse{
		{
			UserID: 3,
			Positions: []DerivativesPositionResponse{
				{
					InstrumentName:     "BTC-PERPETUAL",
					Kind:               "future",
					Direction:          "buy",
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
				},
			},
			OpenOrders: []DerivativesOrderResponse{
				{
					OrderID:             "146062",
					OrderState:          "open",
					OrderType:           "limit",
					TimeInForce:         "good_til_cancelled",
					InstrumentName:      "BTC-15FEB19-3250-P",
					CreationTimestamp:   types.Time(time.UnixMilli(1550050597036)),
					LastUpdateTimestamp: types.Time(time.UnixMilli(1550050597036)),
					Direction:           "buy",
					Price: DerivativesOrderPrice{
						Value: 0.0028,
					},
					Label:  "fooBar",
					API:    true,
					Amount: 10,
				},
			},
		},
		{
			UserID: 10,
			Positions: []DerivativesPositionResponse{
				{
					InstrumentName:     "BTC-3SEP21",
					Kind:               "future",
					Direction:          "sell",
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
				},
			},
		},
	}
}

// testDerivativesTradingLimits is the documented BTC trading limits
func testDerivativesTradingLimits() *DerivativesTradingLimitsResponse {
	return &DerivativesTradingLimitsResponse{
		Currency:                          currency.BTC,
		MaximumFuturePositionSize:         5000000,
		MaximumOpenOrdersInInstrument:     100,
		MaximumTotalOpenOrders:            1000,
		MaximumTotalShortOptionsPositions: 500000,
	}
}

// testDerivativesTransactionLog is the documented transaction log page
func testDerivativesTransactionLog() *DerivativesTransactionLogResponse {
	return &DerivativesTransactionLogResponse{
		Logs: []DerivativesTransactionLogEntry{
			{
				ID:                        61313,
				UserSequence:              6010,
				UserID:                    7,
				Username:                  "TestUser",
				Currency:                  currency.BTC,
				Timestamp:                 types.Time(time.UnixMilli(1613659900000)),
				Type:                      "LSP transfer",
				Role:                      "source",
				Cashflow:                  -2.4,
				Change:                    -2.4,
				Balance:                   2998.82270418,
				Equity:                    2998.5275869,
				InstrumentName:            "BTC-PERPETUAL",
				Side:                      "sell",
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
				Username:     "TestUser",
				Currency:     currency.BTC,
				Timestamp:    types.Time(time.UnixMilli(1613659830333)),
				Type:         "transfer",
				Info:         json.RawMessage(`{"other_user":"Subaccount","other_user_id":27,"transfer_type":"subaccount"}`),
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
				Username:                    "TestUser",
				Currency:                    currency.BTC,
				Timestamp:                   types.Time(time.UnixMilli(1613659544153)),
				Type:                        "settlement",
				Info:                        json.RawMessage(`{"floating_pl":0.00220172,"settlement_price":51807}`),
				Cashflow:                    0.00215706,
				Change:                      0.00215706,
				Balance:                     3003.72270418,
				Equity:                      3003.42821428,
				InstrumentName:              "BTC-PERPETUAL",
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
				Username:     "TestUser",
				Currency:     currency.BTC,
				Timestamp:    types.Time(time.UnixMilli(1613657828414)),
				Type:         "deposit",
				Info:         json.RawMessage(`{"addr":"2N8prMvpZHr8aYqodX3S4yhz5wMxjY8La3p","deposit_type":"wallet","transaction":"de6eba075855f32c9510f338d3ca0900376cedcb9f7b142caccfbdc292d3237e"}`),
				Cashflow:     0.65,
				Change:       0.65,
				Balance:      3003.72054712,
				Equity:       3003.4876111,
				Side:         "-",
			},
			{
				ID:             61289,
				UserSequence:   6006,
				UserID:         7,
				Username:       "TestUser",
				Currency:       currency.BTC,
				Timestamp:      types.Time(time.UnixMilli(1613657734620)),
				Type:           "trade",
				Info:           json.RawMessage(`"Source: api"`),
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
			{
				ID:             61288,
				UserSequence:   6005,
				UserID:         7,
				Username:       "TestUser",
				Currency:       currency.BTC,
				Timestamp:      types.Time(time.UnixMilli(1613657734620)),
				Type:           "trade",
				Info:           json.RawMessage(`"Source: api"`),
				Cashflow:       -0.04611,
				Change:         -0.04611,
				Balance:        3003.17813712,
				Equity:         3002.83270455,
				InstrumentName: "BTC-19FEB21-49200-C",
				Side:           "close buy",
				Amount:         0.3,
				Price:          0.1537,
				PriceCurrency:  currency.BTC,
				MarkPrice:      0.04884653215049635,
				TradeID:        "28349",
				OrderID:        "67546",
				UserRole:       "maker",
			},
			{
				ID:             61287,
				UserSequence:   6004,
				UserID:         7,
				Username:       "TestUser",
				Currency:       currency.BTC,
				Timestamp:      types.Time(time.UnixMilli(1613657734620)),
				Type:           "expiry",
				Info:           json.RawMessage(`null`),
				Balance:        3003.17813712,
				Equity:         3002.83270455,
				InstrumentName: "BTC-19FEB21-52000-C",
				Side:           "close buy",
				Amount:         0.3,
				PriceCurrency:  currency.BTC,
			},
		},
		Continuation: 61282,
	}
}

// testDerivativesUserLocks is the documented account locks
func testDerivativesUserLocks() []DerivativesUserLockResponse {
	return []DerivativesUserLockResponse{
		{
			Currency: currency.BTC,
			Locked:   true,
			Message:  "locked in one currency",
		},
		{
			Currency: currency.ETH,
		},
		{
			Currency: currency.USDC,
		},
		{
			Currency: currency.SOL,
		},
	}
}

// testDerivativesPMESimulation is the documented BTC portfolio margin simulation
func testDerivativesPMESimulation() *DerivativesPMESimulateResponse {
	return &DerivativesPMESimulateResponse{
		ModelParameters: DerivativesPMEModelParameters{
			CurrencyPairs: map[string]DerivativesPMECurrencyPairParameters{
				"btc_usd": {
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
				},
			},
			Currencies: map[string]DerivativesPMECurrencyParameters{
				"btc": {
					AnnualisedMoveRisk:    0.075,
					MinimumAnnualisedMove: 0.01,
					ExtendedDampener:      100000,
					EquitySideImpact:      "both",
				},
				"usd": {
					AnnualisedMoveRisk:    0.1,
					MinimumAnnualisedMove: 0.01,
					ExtendedDampener:      25000,
					EquitySideImpact:      "none",
				},
			},
			General: DerivativesPMEGeneralParameters{
				MaintenanceMarginFactor:  0.8,
				BucketsCount:             4,
				VolatilityScenariosCount: 3,
				Timestamp:                types.Time(time.UnixMilli(1718619740501)),
			},
		},
		AggregatedRiskVectors: map[string]DerivativesPMERiskVectors{
			"btc_btc": {
				Standard: []float64{
					-0.05968587238095239, -0.05968587238095239, -0.05968587238095239,
					-0.04272965863636364, -0.04272965863636364, -0.04272965863636364,
					-0.02724789826086957, -0.02724789826086957, -0.02724789826086957,
					-0.013056284583333334, -0.013056284583333334, -0.013056284583333334,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				},
				Extended: []float64{0, 0, 0, 0, 0, 0, 0, 0},
			},
		},
		InitialRiskVectors: map[string]DerivativesPMERiskVectors{
			"BTC-28JUN24": {
				Standard: []float64{
					0.0002261969523809524, 0.0002261969523809524, 0.0002261969523809524,
					0.00016193645454545456, 0.00016193645454545456, 0.00016193645454545456,
					0.00010326382608695652, 0.00010326382608695652, 0.00010326382608695652,
					0.00004948058333333334, 0.00004948058333333334, 0.00004948058333333334,
					0, 0, 0,
					-0.00004567438461538462, -0.00004567438461538462, -0.00004567438461538462,
					-0.00008796548148148148, -0.00008796548148148148, -0.00008796548148148148,
					-0.0001272357857142857, -0.0001272357857142857, -0.0001272357857142857,
					-0.00016379779310344832, -0.00016379779310344832, -0.00016379779310344832,
				},
				Extended: []float64{
					0.0002261969523809524, 0.0002261969523809524,
					-0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483,
					-0.0001637977931034483, -0.0001637977931034483, -0.0001637977931034483,
				},
			},
			"BTC-PERPETUAL": {
				Standard: []float64{
					-0.05991206933333334, -0.05991206933333334, -0.05991206933333334,
					-0.04289159509090909, -0.04289159509090909, -0.04289159509090909,
					-0.027351162086956524, -0.027351162086956524, -0.027351162086956524,
					-0.013105765166666668, -0.013105765166666668, -0.013105765166666668,
					0, 0, 0,
					0.012097629384615385, 0.012097629384615385, 0.012097629384615385,
					0.023299138074074074, 0.023299138074074074, 0.023299138074074074,
					0.033700538999999995, 0.033700538999999995, 0.033700538999999995,
					0.043384601931034494, 0.043384601931034494, 0.043384601931034494,
				},
				Extended: []float64{
					-0.05991206933333334, -0.05991206933333334,
					0.04338460193103449, 0.04338460193103449, 0.04338460193103449,
					0.04338460193103449, 0.04338460193103449, 0.04338460193103449,
				},
			},
		},
		Margins: map[string]DerivativesPMEMargins{
			"btc": {
				InitialMargin:     0.122861343,
				MaintenanceMargin: 0.050274504784,
				InitialMarginDetails: DerivativesPMEInitialMarginDetails{
					RiskMatrixMargin: 0.06284313098,
					RiskMatrixMarginDetails: DerivativesPMERiskMatrixMarginDetails{
						WorstCaseLoss: 0.05968587238095239,
						WorstCaseBucket: DerivativesPMEWorstCaseBucket{
							Source: "standard",
							Index:  1,
							Bucket: 1,
							Side:   "left",
						},
						RollShock: 0.00315725898,
					},
					MMPMargin:        0.06,
					OpenOrdersMargin: 0.000018212,
				},
			},
		},
		Portfolio: DerivativesPMEPortfolio{
			Currencies: map[string]float64{},
			Positions: map[string]float64{
				"BTC-28JUN24":   -0.001187534,
				"BTC-PERPETUAL": 0.314538364,
			},
		},
		IndexPrices: map[string]float64{
			"btc_usd": 65666.19,
		},
		Tickers: map[string]DerivativesPMETicker{
			"BTC-28JUN24": {
				MarkPrice:  67371.75,
				IndexPrice: 65666.19,
			},
			"BTC-PERPETUAL": {
				MarkPrice:  65910.57,
				IndexPrice: 65666.19,
			},
		},
	}
}

// testDerivativesSimulatedPortfolio is the documented simulated BTC portfolio
func testDerivativesSimulatedPortfolio() *DerivativesSimulatePortfolioResponse {
	return &DerivativesSimulatePortfolioResponse{
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
		OptionsGammaByIndex: map[string]float64{
			"btc_usd": -0.03907,
		},
		OptionsThetaByIndex: map[string]float64{
			"btc_usd": 142583.29246,
		},
		OptionsVegaByIndex: map[string]float64{
			"btc_usd": -39322.23046,
		},
		DeltaTotal:          69080.932029,
		ProjectedDeltaTotal: 69080.932029,
		DeltaTotalByIndex: map[string]float64{
			"btc_usd": 68024.519462366,
		},
		MarginModel:               "cross_pm",
		PortfolioMarginingEnabled: true,
		CrossCollateralEnabled:    true,
		TotalEquityUSD:            13075634611389.318,
		TotalInitialMarginUSD:     3139528603778.822,
		TotalMaintenanceMarginUSD: 2511559381417.215,
		TotalMarginBalanceUSD:     12798550648250.61,
		TotalDeltaTotalUSD:        6157454218.3753195,
	}
}

// testDerivativesAnnouncements is the recorded live announcement
func testDerivativesAnnouncements() []DerivativesAnnouncementResponse {
	return []DerivativesAnnouncementResponse{
		{
			ID:                   1791813857051,
			Title:                "🚨 New Equity & ETF Perpetuals. Coming to Deribit.",
			Body:                 "On Wednesday, October 14, 2026 at 10:00 AM UTC, Deribit will launch the following USDC-settled equity perpetuals: <br><br>CRWV • ORCL • SOXS • DELL • PLTR • SNXX<br><br>Contract details:<br>🔹 Settlement: USDC<br>🔹&nbsp;<a href=\"https://support.deribit.com/hc/en-us/articles/38325634622493-RWA-Perpetual\" style=\"font-family: Arial; font-size: 16px;\">Full specs </a><br><br><b style=\"font-family: Arial; font-size: 16px; font-weight: bold;\">Deribit. Always Open.</b><div></div>",
			Important:            true,
			PublicationTimestamp: types.Time(time.UnixMilli(1791713204677)),
		},
	}
}

func TestDerivativesAccountCurrency(t *testing.T) {
	t.Parallel()
	codes := func(set []currency.Code) []string {
		symbols := make([]string, len(set))
		for i := range set {
			symbols[i] = set[i].String()
		}
		return symbols
	}
	assert.Equal(t, []string{"BTC", "ETH", "USDC", "USDT", "EURR"}, codes(derivativesAccountCurrencies), "derivativesAccountCurrencies should hold the currency schema's currencies")
	assert.Equal(t, []string{"BTC", "ETH", "STETH", "ETHW", "USDC", "USDT", "EURR", "SOL", "XRP", "USYC", "PAXG", "BNB", "USDE"}, codes(derivativesWalletCurrencies), "derivativesWalletCurrencies should hold the wallet_currency schema's currencies")
	assert.Equal(t, []string{"BTC", "ETH", "USDC", "USDT", "CROSS"}, codes(derivativesPMECurrencies), "derivativesPMECurrencies should hold the pme_currency schema's currencies")

	ccy, err := derivativesAccountCurrency(currency.NewCode("usdc"), derivativesAccountCurrencies)
	require.NoError(t, err, "derivativesAccountCurrency must accept a documented currency in any case")
	assert.Equal(t, "USDC", ccy, "derivativesAccountCurrency should return the currency in upper case")
	ccy, err = derivativesAccountCurrency(currency.NewCode("cross"), derivativesPMECurrencies)
	require.NoError(t, err, "derivativesAccountCurrency must accept CROSS for a portfolio margin simulation")
	assert.Equal(t, "CROSS", ccy, "derivativesAccountCurrency should return CROSS in upper case")
	_, err = derivativesAccountCurrency(currency.EMPTYCODE, derivativesAccountCurrencies)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "derivativesAccountCurrency should refuse an empty currency")
	_, err = derivativesAccountCurrency(currency.SOL, derivativesAccountCurrencies)
	assert.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "derivativesAccountCurrency should refuse a currency the method does not document")
}

func TestGlobalDerivativesChangeMarginModel(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().ChangeMarginModel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangeMarginModel must reject a nil request")
	_, err = e.GlobalDerivatives().ChangeMarginModel(t.Context(), &DerivativesChangeMarginModelRequest{MarginModel: "cross"})
	require.ErrorIs(t, err, errDerivativesInvalidMarginModel, "ChangeMarginModel must reject an undocumented margin model")

	// The live request is a dry run, which changes nothing, but needs the trade scope
	skipLiveUnlessCanManipulate(t)
	arg := testDerivativesChangeMarginModelRequest()
	if !mockTests {
		arg = liveDerivativesChangeMarginModelRequest()
	}
	resp, err := e.GlobalDerivatives().ChangeMarginModel(t.Context(), arg)
	require.NoError(t, err, "ChangeMarginModel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMarginModelChanges(), resp, "ChangeMarginModel should decode every field")
	} else {
		assert.NotEmpty(t, resp, "ChangeMarginModel should return the change's effect on each currency")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().ChangeMarginModel(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ChangeMarginModel should return the request failure")
}

func TestGlobalDerivativesGetAccessLog(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetAccessLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccessLog must reject a nil request")

	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{Count: testDerivativesAccessLogCount})
	if mockTests {
		require.NoError(t, err, "GetAccessLog must not error")
		assert.Equal(t, testDerivativesAccessLog(), resp, "GetAccessLog should decode every field")
	} else {
		// As of October 2026 the gateway answers the HTTP form with HTTP 403 and an HTML page
		assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccessLog should be refused over HTTP")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `[]`)
	resp, err = ex.GlobalDerivatives().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{Offset: 10, Count: 1000})
	require.NoError(t, err, "GetAccessLog must not error for a bare list of entries")
	assert.Equal(t, &DerivativesAccessLogResponse{Entries: []DerivativesAccessLogEntry{}}, resp, "GetAccessLog should decode the bare list the schema documents")
	params := sent()
	require.Len(t, params, 1, "GetAccessLog must send one request")
	assert.JSONEq(t, `{"offset":10,"count":1000}`, params[0], "GetAccessLog should send the offset and count")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccessLog should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccessLog should return the request failure")
}

func TestDerivativesAccessLogResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const entry = `{"id":46,"timestamp":1575876682576,"ip":"127.0.0.1","country":"Local Country","city":"Local Town","log":"new_api_key","data":"IY2D68DS"}`
	expEntry := DerivativesAccessLogEntry{
		ID:        46,
		Timestamp: types.Time(time.UnixMilli(1575876682576)),
		IP:        "127.0.0.1",
		Country:   "Local Country",
		City:      "Local Town",
		Log:       "new_api_key",
		Data:      json.RawMessage(`"IY2D68DS"`),
	}
	for _, tc := range []struct {
		input string
		exp   DerivativesAccessLogResponse
	}{
		{input: `{"records_total":1,"data":[` + entry + `]}`, exp: DerivativesAccessLogResponse{RecordsTotal: 1, Entries: []DerivativesAccessLogEntry{expEntry}}},
		// A direct call may pass the list after whitespace, which the decoder trims before it reaches UnmarshalJSON
		{input: ` [` + entry + `]`, exp: DerivativesAccessLogResponse{Entries: []DerivativesAccessLogEntry{expEntry}}},
		{input: `[]`, exp: DerivativesAccessLogResponse{Entries: []DerivativesAccessLogEntry{}}},
	} {
		var resp DerivativesAccessLogResponse
		require.NoErrorf(t, resp.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, resp, "UnmarshalJSON should decode %s", tc.input)
	}
	for _, input := range []string{`"log"`, `[1]`, `{"data":{}}`} {
		var resp DerivativesAccessLogResponse
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &resp), errDerivativesInvalidAccessLog, "UnmarshalJSON should refuse %s", input)
	}
}

func TestGlobalDerivativesGetAccountSummaries(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetAccountSummaries(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountSummaries must reject a nil request")

	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetAccountSummaries(t.Context(), &DerivativesGetAccountSummariesRequest{Extended: true, IncludeIsolated: true})
	require.NoError(t, err, "GetAccountSummaries must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAccountSummaries(), resp, "GetAccountSummaries should decode every field")
	} else {
		assert.NotEmpty(t, resp.Summaries, "GetAccountSummaries should return a summary for each currency")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetAccountSummaries(t.Context(), &DerivativesGetAccountSummariesRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSummaries should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetAccountSummaries(t.Context(), &DerivativesGetAccountSummariesRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSummaries should return the request failure")
}

func TestGlobalDerivativesGetAccountSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetAccountSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountSummary must reject a nil request")
	_, err = e.GlobalDerivatives().GetAccountSummary(t.Context(), &DerivativesGetAccountSummaryRequest{Extended: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAccountSummary must require a currency")
	_, err = e.GlobalDerivatives().GetAccountSummary(t.Context(), &DerivativesGetAccountSummaryRequest{Currency: currency.LTC})
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetAccountSummary must refuse a currency the method does not document")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesGetAccountSummaryRequest{Currency: currency.BTC, Extended: true}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := e.GlobalDerivatives().GetAccountSummary(t.Context(), arg)
	require.NoError(t, err, "GetAccountSummary must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAccountSummary(), resp, "GetAccountSummary should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(currency.USDC), "GetAccountSummary should return the currency asked for")
	}

	// The wallet currencies are documented for the account summary alone among the currency methods
	ex, sent := newDerivativesAccountParamsServer(t, `{"currency":"SOL"}`)
	_, err = ex.GlobalDerivatives().GetAccountSummary(t.Context(), &DerivativesGetAccountSummaryRequest{Currency: currency.NewCode("sol"), SubaccountID: testDerivativesAccountIsolatedSubaccountID})
	require.NoError(t, err, "GetAccountSummary must accept a wallet currency")
	params := sent()
	require.Len(t, params, 1, "GetAccountSummary must send one request")
	assert.JSONEq(t, `{"currency":"SOL","subaccount_id":123456}`, params[0], "GetAccountSummary should send the currency in upper case and the subaccount")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSummary should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSummary should return the request failure")
}

func TestDerivativesMatchingEngineLimitsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input string
		exp   DerivativesMatchingEngineLimits
	}{
		{
			name:  "limits for every currency",
			input: `{"trading":{"total":{"burst":20,"rate":5}},"spot":{"burst":250,"rate":200},"maximum_quotes":{"burst":500,"rate":500},"maximum_mass_quotes":{"burst":10,"rate":10},"guaranteed_mass_quotes":{"burst":2,"rate":2},"cancel_all":{"burst":250,"rate":200}}`,
			exp: DerivativesMatchingEngineLimits{
				Spot:                 DerivativesAccountRateLimit{Rate: 200, Burst: 250},
				CancelAll:            DerivativesAccountRateLimit{Rate: 200, Burst: 250},
				Trading:              map[string]DerivativesAccountRateLimit{"total": {Rate: 5, Burst: 20}},
				MaximumQuotes:        DerivativesAccountRateLimit{Rate: 500, Burst: 500},
				MaximumMassQuotes:    DerivativesAccountRateLimit{Rate: 10, Burst: 10},
				GuaranteedMassQuotes: DerivativesAccountRateLimit{Rate: 2, Burst: 2},
			},
		},
		{
			name:  "limits per currency",
			input: `{"cancel_all":{"burst":250,"rate":200},"spot":{"burst":250,"rate":200},"usdc":{"maximum_quotes":{"burst":500,"rate":500},"maximum_mass_quotes":{"burst":10,"rate":10},"guaranteed_mass_quotes":{"burst":2,"rate":2},"trading":{"perpetuals":{"burst":20,"rate":10},"total":{"burst":150,"rate":100}}}}`,
			exp: DerivativesMatchingEngineLimits{
				Spot:      DerivativesAccountRateLimit{Rate: 200, Burst: 250},
				CancelAll: DerivativesAccountRateLimit{Rate: 200, Burst: 250},
				Currencies: map[string]DerivativesMatchingEngineTradingLimits{
					"usdc": {
						Trading:              map[string]DerivativesAccountRateLimit{"perpetuals": {Rate: 10, Burst: 20}, "total": {Rate: 100, Burst: 150}},
						MaximumQuotes:        DerivativesAccountRateLimit{Rate: 500, Burst: 500},
						MaximumMassQuotes:    DerivativesAccountRateLimit{Rate: 10, Burst: 10},
						GuaranteedMassQuotes: DerivativesAccountRateLimit{Rate: 2, Burst: 2},
					},
				},
			},
		},
		{
			name:  "documented example names",
			input: `{"quotes":{"burst":500,"rate":500},"max_quotes":{"burst":10,"rate":10},"guaranteed_quotes":{"burst":2,"rate":2}}`,
			exp: DerivativesMatchingEngineLimits{
				MaximumQuotes:        DerivativesAccountRateLimit{Rate: 500, Burst: 500},
				MaximumMassQuotes:    DerivativesAccountRateLimit{Rate: 10, Burst: 10},
				GuaranteedMassQuotes: DerivativesAccountRateLimit{Rate: 2, Burst: 2},
			},
		},
		{name: "null", input: `null`},
	} {
		var limits DerivativesMatchingEngineLimits
		require.NoErrorf(t, json.Unmarshal([]byte(tc.input), &limits), "Unmarshal must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, limits, "UnmarshalJSON should decode %s", tc.name)
	}
	for _, input := range []string{`[]`, `{"spot":[]}`, `{"usdc":{"trading":1}}`} {
		var limits DerivativesMatchingEngineLimits
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &limits), errDerivativesInvalidMatchingEngineLimits, "UnmarshalJSON should refuse %s", input)
	}
}

func TestGlobalDerivativesGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLeverage(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLeverage must reject a nil request")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesGetLeverageRequest{InstrumentName: testDerivativesAccountInstrument}
	if !mockTests {
		arg.InstrumentName = "BTC_USDC-PERPETUAL"
	}
	resp, err := e.GlobalDerivatives().GetLeverage(t.Context(), arg)
	require.NoError(t, err, "GetLeverage must not error")
	if mockTests {
		assert.Equal(t, testDerivativesLeverages(), resp, "GetLeverage should decode every field")
	} else {
		require.Len(t, resp, 1, "GetLeverage must return the instrument's leverage")
		assert.Positive(t, resp[0].MaximumLeverage, "GetLeverage should return the maximum leverage")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `[{"instrument_name":"SOL_USDC-PERPETUAL","leverage":5,"max_leverage":50}]`)
	resp, err = ex.GlobalDerivatives().GetLeverage(t.Context(), &DerivativesGetLeverageRequest{Isolated: true})
	require.NoError(t, err, "GetLeverage must not error for every future")
	assert.Equal(t, []DerivativesLeverageResponse{{InstrumentName: "SOL_USDC-PERPETUAL", Leverage: 5, MaximumLeverage: 50}}, resp, "GetLeverage should decode the list of futures with a custom leverage")
	params := sent()
	require.Len(t, params, 1, "GetLeverage must send one request")
	assert.JSONEq(t, `{"isolated":true}`, params[0], "GetLeverage should leave the instrument out and send the isolated flag")

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLeverage(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLeverage should return the request failure")
}

func TestDerivativesLeveragesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   DerivativesLeverages
	}{
		{input: `{"instrument_name":"BTC_USDC-PERPETUAL","leverage":null,"max_leverage":50}`, exp: DerivativesLeverages{{InstrumentName: "BTC_USDC-PERPETUAL", MaximumLeverage: 50}}},
		// A direct call may pass the list after whitespace, which the decoder trims before it reaches UnmarshalJSON
		{
			input: ` [{"instrument_name":"BTC_USDC-PERPETUAL","leverage":10,"max_leverage":50},{"instrument_name":"ETH_USDC-PERPETUAL","leverage":2.5,"max_leverage":50}]`,
			exp:   DerivativesLeverages{{InstrumentName: "BTC_USDC-PERPETUAL", Leverage: 10, MaximumLeverage: 50}, {InstrumentName: "ETH_USDC-PERPETUAL", Leverage: 2.5, MaximumLeverage: 50}},
		},
		{input: `[]`, exp: DerivativesLeverages{}},
		// The gateway hides isolated margin leverage from a regular subaccount behind an empty object
		{input: `{}`, exp: DerivativesLeverages{}},
	} {
		var leverages DerivativesLeverages
		require.NoErrorf(t, leverages.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, leverages, "UnmarshalJSON should decode %s", tc.input)
	}
	leverages := DerivativesLeverages{{InstrumentName: "kept"}}
	require.NoError(t, leverages.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error for null")
	assert.Equal(t, DerivativesLeverages{{InstrumentName: "kept"}}, leverages, "UnmarshalJSON should leave the leverages unchanged for null")
	for _, input := range []string{`"leverage"`, `[1]`, `{"max_leverage":"high"}`} {
		var leverages DerivativesLeverages
		assert.ErrorIsf(t, leverages.UnmarshalJSON([]byte(input)), errDerivativesMalformedLeverage, "UnmarshalJSON should refuse %s", input)
	}
}

func TestGlobalDerivativesGetPMEParams(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetPMEParams(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPMEParams must require a currency")
	_, err = e.GlobalDerivatives().GetPMEParams(t.Context(), currency.SOL)
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetPMEParams must refuse a currency the method does not document")

	skipLiveWithoutCredentials(t)
	ccy := currency.BTC
	if !mockTests {
		ccy = currency.USDC
	}
	resp, err := e.GlobalDerivatives().GetPMEParams(t.Context(), ccy)
	require.NoError(t, err, "GetPMEParams must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPMEParams(), resp, "GetPMEParams should decode every field")
	} else if resp.Enabled {
		assert.Positive(t, resp.MaximumTotalOpenOrders, "GetPMEParams should return the open order limit with portfolio margin enabled")
	}

	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"enabled":false}}`).GlobalDerivatives().GetPMEParams(t.Context(), currency.USDC)
	require.NoError(t, err, "GetPMEParams must not error with portfolio margin disabled")
	assert.Equal(t, &DerivativesPMEParamsResponse{}, resp, "GetPMEParams should decode a disabled portfolio margin")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetPMEParams(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPMEParams should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetPMEParams(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPMEParams should return the request failure")
}

func TestGlobalDerivativesGetPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPosition must reject a nil request")
	_, err = e.GlobalDerivatives().GetPosition(t.Context(), &DerivativesGetPositionRequest{IncludeIsolated: true})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetPosition must require an instrument")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesGetPositionRequest{InstrumentName: testDerivativesAccountInstrument}
	if !mockTests {
		arg.InstrumentName = "BTC_USDC-PERPETUAL"
	}
	resp, err := e.GlobalDerivatives().GetPosition(t.Context(), arg)
	require.NoError(t, err, "GetPosition must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPosition(), resp, "GetPosition should decode every field")
	} else {
		assert.Equal(t, arg.InstrumentName, resp.InstrumentName, "GetPosition should return the instrument's position")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `{"instrument_name":"SOL_USDC-PERPETUAL","isolated":true}`)
	_, err = ex.GlobalDerivatives().GetPosition(t.Context(), &DerivativesGetPositionRequest{InstrumentName: "SOL_USDC-PERPETUAL", IncludeIsolated: true})
	require.NoError(t, err, "GetPosition must not error for an isolated position")
	params := sent()
	require.Len(t, params, 1, "GetPosition must send one request")
	assert.JSONEq(t, `{"instrument_name":"SOL_USDC-PERPETUAL","include_isolated":true}`, params[0], "GetPosition should send the isolated flag")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetPosition(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPosition should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetPosition(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPosition should return the request failure")
}

func TestGlobalDerivativesGetPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositions must reject a nil request")
	_, err = e.GlobalDerivatives().GetPositions(t.Context(), &DerivativesGetPositionsRequest{Currency: currency.SOL})
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetPositions must refuse a currency the method does not document")
	_, err = e.GlobalDerivatives().GetPositions(t.Context(), &DerivativesGetPositionsRequest{Kind: DerivativesKindSpot})
	require.ErrorIs(t, err, errDerivativesInvalidPositionKind, "GetPositions must refuse spot, which has no positions")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesGetPositionsRequest{Currency: currency.BTC, Kind: DerivativesKindFuture}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := e.GlobalDerivatives().GetPositions(t.Context(), arg)
	require.NoError(t, err, "GetPositions must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPositions(), resp, "GetPositions should decode every field")
	} else {
		for i := range resp {
			assert.Equal(t, DerivativesKindFuture, resp[i].Kind, "GetPositions should return future positions")
		}
	}

	arg = &DerivativesGetPositionsRequest{Currency: currency.NewCode(DerivativesCurrencyAny), IncludeIsolated: true}
	resp, err = e.GlobalDerivatives().GetPositions(t.Context(), arg)
	require.NoError(t, err, "GetPositions must not error for every currency")
	if mockTests {
		assert.Equal(t, testDerivativesIsolatedPositions(), resp, "GetPositions should decode an isolated margin position")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `[]`)
	for _, req := range []*DerivativesGetPositionsRequest{
		{Currency: currency.NewCode("ANY"), Kind: DerivativesKindOption},
		{Currency: currency.NewCode("usdt"), SubaccountID: testDerivativesAccountIsolatedSubaccountID},
		{},
	} {
		_, err = ex.GlobalDerivatives().GetPositions(t.Context(), req)
		require.NoError(t, err, "GetPositions must not error")
	}
	assert.Equal(t, []string{`{"currency":"any","kind":"option"}`, `{"currency":"USDT","subaccount_id":123456}`, `{}`}, sent(), "GetPositions should send any in lower case, other currencies in upper case and leave an empty currency out")

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetPositions(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPositions should return the request failure")
}

func TestGlobalDerivativesGetRiskProfile(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetRiskProfile(t.Context())
	require.NoError(t, err, "GetRiskProfile must not error")
	if mockTests {
		exp := &DerivativesRiskProfileResponse{
			PortfolioMarginingAllowed: true,
			ShortOptionAllowed:        true,
		}
		assert.Equal(t, exp, resp, "GetRiskProfile should decode every field")
	}

	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"max_leverage":20,"portfolio_margining_allowed":false,"short_option_allowed":false}}`).GlobalDerivatives().GetRiskProfile(t.Context())
	require.NoError(t, err, "GetRiskProfile must not error for a leverage ceiling")
	assert.Equal(t, &DerivativesRiskProfileResponse{MaximumLeverage: 20}, resp, "GetRiskProfile should decode the leverage ceiling")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetRiskProfile(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetRiskProfile should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetRiskProfile(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRiskProfile should return the request failure")
}

func TestGlobalDerivativesGetSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetSubaccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubaccounts must reject a nil request")

	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetSubaccounts(t.Context(), &DerivativesGetSubaccountsRequest{WithPortfolio: true})
	require.NoError(t, err, "GetSubaccounts must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSubaccounts(), resp, "GetSubaccounts should decode every field")
	} else {
		assert.NotEmpty(t, resp, "GetSubaccounts should return the main account")
	}

	resp, err = e.GlobalDerivatives().GetSubaccounts(t.Context(), &DerivativesGetSubaccountsRequest{IncludeIsolated: true})
	require.NoError(t, err, "GetSubaccounts must not error with the isolated margin subaccounts")
	if mockTests {
		assert.Equal(t, testDerivativesIsolatedSubaccounts(), resp, "GetSubaccounts should decode the isolated margin fields")
	} else {
		assert.NotEmpty(t, resp, "GetSubaccounts should return the main account")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetSubaccounts(t.Context(), &DerivativesGetSubaccountsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSubaccounts should return the request failure")
}

func TestGlobalDerivativesGetSubaccountsDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetSubaccountsDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSubaccountsDetails must reject a nil request")
	_, err = e.GlobalDerivatives().GetSubaccountsDetails(t.Context(), &DerivativesGetSubaccountsDetailsRequest{WithOpenOrders: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetSubaccountsDetails must require a currency")
	_, err = e.GlobalDerivatives().GetSubaccountsDetails(t.Context(), &DerivativesGetSubaccountsDetailsRequest{Currency: currency.SOL})
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetSubaccountsDetails must refuse a currency the method does not document")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesGetSubaccountsDetailsRequest{Currency: currency.BTC, WithOpenOrders: true}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := e.GlobalDerivatives().GetSubaccountsDetails(t.Context(), arg)
	require.NoError(t, err, "GetSubaccountsDetails must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSubaccountsDetails(), resp, "GetSubaccountsDetails should decode every field")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `[]`)
	_, err = ex.GlobalDerivatives().GetSubaccountsDetails(t.Context(), &DerivativesGetSubaccountsDetailsRequest{Currency: currency.NewCode("usdc"), IncludeIsolated: true})
	require.NoError(t, err, "GetSubaccountsDetails must not error with the isolated margin subaccounts")
	params := sent()
	require.Len(t, params, 1, "GetSubaccountsDetails must send one request")
	assert.JSONEq(t, `{"currency":"USDC","include_isolated":true}`, params[0], "GetSubaccountsDetails should send the currency in upper case and the isolated flag")

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetSubaccountsDetails(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetSubaccountsDetails should return the request failure")
}

func TestGlobalDerivativesGetTradingLimits(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTradingLimits(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTradingLimits must require a currency")
	_, err = e.GlobalDerivatives().GetTradingLimits(t.Context(), currency.SOL)
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetTradingLimits must refuse a currency the method does not document")

	skipLiveWithoutCredentials(t)
	ccy := currency.BTC
	if !mockTests {
		ccy = currency.USDC
	}
	resp, err := e.GlobalDerivatives().GetTradingLimits(t.Context(), ccy)
	require.NoError(t, err, "GetTradingLimits must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTradingLimits(), resp, "GetTradingLimits should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(ccy), "GetTradingLimits should return the currency's limits")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetTradingLimits(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTradingLimits should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTradingLimits(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradingLimits should return the request failure")
}

func TestGlobalDerivativesGetTransactionLog(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetTransactionLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransactionLog must reject a nil request")
	arg := testDerivativesTransactionLogRequest()
	arg.Currency = currency.EMPTYCODE
	_, err = e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTransactionLog must require a currency")
	arg.Currency = currency.LTC
	_, err = e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "GetTransactionLog must refuse a currency the method does not document")
	arg = testDerivativesTransactionLogRequest()
	arg.StartTime = time.Time{}
	_, err = e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTransactionLog must require a start time")
	arg = testDerivativesTransactionLogRequest()
	arg.EndTime = time.Time{}
	_, err = e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetTransactionLog must require an end time")
	arg = testDerivativesTransactionLogRequest()
	arg.StartTime, arg.EndTime = arg.EndTime, arg.StartTime
	_, err = e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTransactionLog must refuse a reversed window")

	skipLiveWithoutCredentials(t)
	arg = testDerivativesTransactionLogRequest()
	if !mockTests {
		arg = liveDerivativesTransactionLogRequest()
	}
	resp, err := e.GlobalDerivatives().GetTransactionLog(t.Context(), arg)
	require.NoError(t, err, "GetTransactionLog must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTransactionLog(), resp, "GetTransactionLog should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Logs), testDerivativesTransactionLogCount, "GetTransactionLog should return at most the entries asked for")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `{"continuation":null,"logs":[]}`)
	resp, err = ex.GlobalDerivatives().GetTransactionLog(t.Context(), &DerivativesGetTransactionLogRequest{
		Currency:     currency.NewCode("usde"),
		StartTime:    time.UnixMilli(testDerivativesTransactionLogStart),
		EndTime:      time.UnixMilli(testDerivativesTransactionLogEnd),
		Query:        "trade maker",
		Count:        250,
		SubaccountID: testDerivativesAccountIsolatedSubaccountID,
		Continuation: 61282,
	})
	require.NoError(t, err, "GetTransactionLog must not error for an isolated margin subaccount's page")
	assert.Equal(t, &DerivativesTransactionLogResponse{Logs: []DerivativesTransactionLogEntry{}}, resp, "GetTransactionLog should decode the last page, whose continuation is null")
	params := sent()
	require.Len(t, params, 1, "GetTransactionLog must send one request")
	assert.JSONEq(t, `{"currency":"USDE","start_timestamp":1613657734000,"end_timestamp":1613660407000,"query":"trade maker","count":250,"subaccount_id":123456,"continuation":61282}`, params[0], "GetTransactionLog should send every field, the window in Unix milliseconds")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetTransactionLog(t.Context(), testDerivativesTransactionLogRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransactionLog should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTransactionLog(t.Context(), testDerivativesTransactionLogRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransactionLog should return the request failure")
}

func TestGlobalDerivativesGetUserLocks(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetUserLocks(t.Context())
	require.NoError(t, err, "GetUserLocks must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserLocks(), resp, "GetUserLocks should decode every field")
	} else {
		assert.NotEmpty(t, resp, "GetUserLocks should return each currency's lock")
	}

	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[{"currency":"USDC","enabled":true,"message":"locked"}]}`).GlobalDerivatives().GetUserLocks(t.Context())
	require.NoError(t, err, "GetUserLocks must not error for the schema's field name")
	assert.Equal(t, []DerivativesUserLockResponse{{Currency: currency.USDC, Enabled: true, Message: "locked"}}, resp, "GetUserLocks should decode the lock under the schema's name")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetUserLocks(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserLocks should return the request failure")
}

func TestGlobalDerivativesPMESimulate(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().PMESimulate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PMESimulate must reject a nil request")
	_, err = e.GlobalDerivatives().PMESimulate(t.Context(), &DerivativesSimulatePortfolioRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PMESimulate must require a currency")
	_, err = e.GlobalDerivatives().PMESimulate(t.Context(), &DerivativesSimulatePortfolioRequest{Currency: currency.EURR})
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "PMESimulate must refuse a currency the method does not document")

	skipLiveWithoutCredentials(t)
	arg := &DerivativesSimulatePortfolioRequest{Currency: currency.BTC}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := e.GlobalDerivatives().PMESimulate(t.Context(), arg)
	require.NoError(t, err, "PMESimulate must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPMESimulation(), resp, "PMESimulate should decode every field")
	} else {
		assert.NotEmpty(t, resp.Margins, "PMESimulate should return the simulated margins")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `{}`)
	_, err = ex.GlobalDerivatives().PMESimulate(t.Context(), &DerivativesSimulatePortfolioRequest{Currency: currency.NewCode("cross"), AddPositions: new(false), SimulatedPositions: map[string]float64{"BTC_USDC-PERPETUAL": -1}})
	require.NoError(t, err, "PMESimulate must not error for the cross collateral portfolio")
	params := sent()
	require.Len(t, params, 1, "PMESimulate must send one request")
	assert.JSONEq(t, `{"currency":"CROSS","add_positions":false,"simulated_positions":{"BTC_USDC-PERPETUAL":-1}}`, params[0], "PMESimulate should send CROSS and simulate the positions alone")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().PMESimulate(t.Context(), arg)
	assert.ErrorIs(t, err, common.ErrNoResponse, "PMESimulate should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().PMESimulate(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "PMESimulate should return the request failure")
}

func TestGlobalDerivativesSimulatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().SimulatePortfolio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SimulatePortfolio must reject a nil request")
	_, err = e.GlobalDerivatives().SimulatePortfolio(t.Context(), &DerivativesSimulatePortfolioRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SimulatePortfolio must require a currency")
	_, err = e.GlobalDerivatives().SimulatePortfolio(t.Context(), &DerivativesSimulatePortfolioRequest{Currency: currency.NewCode("CROSS")})
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "SimulatePortfolio must refuse CROSS, which only PMESimulate takes")

	skipLiveWithoutCredentials(t)
	arg := testDerivativesSimulatePortfolioRequest()
	if !mockTests {
		arg = &DerivativesSimulatePortfolioRequest{Currency: currency.USDC, SimulatedPositions: map[string]float64{"BTC_USDC-PERPETUAL": 0.0001}}
	}
	resp, err := e.GlobalDerivatives().SimulatePortfolio(t.Context(), arg)
	require.NoError(t, err, "SimulatePortfolio must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSimulatedPortfolio(), resp, "SimulatePortfolio should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(currency.USDC), "SimulatePortfolio should return the simulated currency")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().SimulatePortfolio(t.Context(), testDerivativesSimulatePortfolioRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SimulatePortfolio should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().SimulatePortfolio(t.Context(), testDerivativesSimulatePortfolioRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SimulatePortfolio should return the request failure")
}

func TestDerivativesSimulatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := derivativesSimulatePortfolio(nil, derivativesAccountCurrencies)
	require.ErrorIs(t, err, common.ErrNilPointer, "derivativesSimulatePortfolio must reject a nil request")
	_, err = derivativesSimulatePortfolio(&DerivativesSimulatePortfolioRequest{Currency: currency.NewCode("CROSS")}, derivativesAccountCurrencies)
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "derivativesSimulatePortfolio must check the currency against the method's currencies")
	for _, tc := range []struct {
		req *DerivativesSimulatePortfolioRequest
		exp string
	}{
		{req: &DerivativesSimulatePortfolioRequest{Currency: currency.NewCode("eth")}, exp: `{"currency":"ETH"}`},
		{req: &DerivativesSimulatePortfolioRequest{Currency: currency.BTC, AddPositions: new(false)}, exp: `{"currency":"BTC","add_positions":false}`},
		{req: testDerivativesSimulatePortfolioRequest(), exp: `{"currency":"BTC","add_positions":true,"simulated_positions":{"BTC-PERPETUAL":1}}`},
	} {
		params, err := derivativesSimulatePortfolio(tc.req, derivativesAccountCurrencies)
		require.NoError(t, err, "derivativesSimulatePortfolio must not error")
		encoded, err := json.Marshal(params)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, tc.exp, string(encoded), "derivativesSimulatePortfolio should leave unset fields out and send an explicit false")
	}
}

func TestGlobalDerivativesGetAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAnnouncements must reject a nil request")

	resp, err := e.GlobalDerivatives().GetAnnouncements(t.Context(), testDerivativesAnnouncementsRequest())
	require.NoError(t, err, "GetAnnouncements must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAnnouncements(), resp, "GetAnnouncements should decode every field")
	} else {
		require.Len(t, resp, testDerivativesAnnouncementsCount, "GetAnnouncements must return the announcements asked for")
		assert.False(t, resp[0].PublicationTimestamp.Time().After(time.UnixMilli(testDerivativesAnnouncementsStart)), "GetAnnouncements should return announcements published up to the start time")
	}

	ex, sent := newDerivativesAccountParamsServer(t, `[]`)
	resp, err = ex.GlobalDerivatives().GetAnnouncements(t.Context(), &DerivativesGetAnnouncementsRequest{})
	require.NoError(t, err, "GetAnnouncements must not error for the defaults")
	assert.Empty(t, resp, "GetAnnouncements should decode an empty list")
	params := sent()
	require.Len(t, params, 1, "GetAnnouncements must send one request")
	assert.JSONEq(t, `{}`, params[0], "GetAnnouncements should leave the defaults out")

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetAnnouncements(t.Context(), testDerivativesAnnouncementsRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAnnouncements should return the request failure")
}
