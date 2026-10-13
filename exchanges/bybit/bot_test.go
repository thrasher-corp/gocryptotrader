package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const testBotBanReason = "Bots are not available to your account in its current region"

func TestBotStatusError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		statusCode uint64
		mapped     error
		exp        string
	}{
		{statusCode: 0},
		{statusCode: 200},
		{statusCode: 400, mapped: errBotInvalidParameters, exp: "bot request failed: invalid bot request parameters: status code 400: detail"},
		{statusCode: 401, exp: "bot request failed: status code 401: detail"},
		{statusCode: 404, mapped: errBotNotFound, exp: "bot request failed: bot not found: status code 404: detail"},
		{statusCode: 405, mapped: errBotNotClosable, exp: "bot request failed: bot cannot be closed in its current state: status code 405: detail"},
		{statusCode: 421, mapped: errBotUserBanned, exp: "bot request failed: user banned from bots: status code 421: detail"},
		{statusCode: 429, mapped: errBotRateLimitExceeded, exp: "bot request failed: bot rate limit exceeded: status code 429: detail"},
		{statusCode: 503, mapped: errBotNotClosable, exp: "bot request failed: bot cannot be closed in its current state: status code 503: detail"},
	} {
		err := botStatusError(tc.statusCode, "", "detail")
		if tc.exp == "" {
			assert.NoErrorf(t, err, "botStatusError should accept status code %d", tc.statusCode)
			continue
		}
		require.ErrorIsf(t, err, errBotRequestFailed, "botStatusError must reject status code %d", tc.statusCode)
		if tc.mapped != nil {
			assert.ErrorIsf(t, err, tc.mapped, "botStatusError should map status code %d", tc.statusCode)
		}
		assert.EqualErrorf(t, err, tc.exp, "botStatusError should describe status code %d", tc.statusCode)
	}
	assert.EqualError(t, botStatusError(421, "banned", "", "trace"), "bot request failed: user banned from bots: status code 421: banned: trace", "botStatusError should join every non-empty detail")
}

// testFuturesComboBotRequest returns the futures combo bot settings the fixtures record
func testFuturesComboBotRequest() *CreateFuturesComboBotRequest {
	return &CreateFuturesComboBotRequest{
		Leverage:           5,
		InitialMargin:      500,
		AdjustPositionMode: 3,
		SymbolSettings: []FuturesComboSymbolSettingRequest{
			{Symbol: "BTCUSDT", TargetPositionPercent: 0.5, Side: 1},
			{Symbol: "ETHUSDT", TargetPositionPercent: 0.5, Side: 2},
		},
		AdjustPositionPercent:      0.05,
		AdjustPositionTimeInterval: time.Hour,
		SLPercent:                  0.2,
		TPPercent:                  0.5,
		TrailingStopPercent:        0.05,
	}
}

// requireFuturesComboBotValidation checks call rejects each setting Get Bot Parameter Limit and Create Bot require
func requireFuturesComboBotValidation(t *testing.T, method string, call func(*CreateFuturesComboBotRequest) error) {
	t.Helper()
	for _, tc := range []struct {
		name   string
		mutate func(*CreateFuturesComboBotRequest)
		err    error
	}{
		{name: "leverage", mutate: func(r *CreateFuturesComboBotRequest) { r.Leverage = 0 }, err: errInvalidLeverage},
		{name: "initial margin", mutate: func(r *CreateFuturesComboBotRequest) { r.InitialMargin = 0 }, err: errBotInitialMarginRequired},
		{name: "adjust position mode", mutate: func(r *CreateFuturesComboBotRequest) { r.AdjustPositionMode = 0 }, err: errAdjustPositionModeRequired},
		{name: "symbol settings", mutate: func(r *CreateFuturesComboBotRequest) { r.SymbolSettings = nil }, err: errSymbolSettingsRequired},
		{name: "a setting's symbol", mutate: func(r *CreateFuturesComboBotRequest) { r.SymbolSettings[1].Symbol = "" }, err: errSymbolMissing},
		{name: "a setting's target position percent", mutate: func(r *CreateFuturesComboBotRequest) { r.SymbolSettings[1].TargetPositionPercent = 0 }, err: errTargetPositionPercentRequired},
		{name: "a setting's side", mutate: func(r *CreateFuturesComboBotRequest) { r.SymbolSettings[1].Side = 0 }, err: order.ErrSideIsInvalid},
	} {
		r := testFuturesComboBotRequest()
		tc.mutate(r)
		require.ErrorIsf(t, call(r), tc.err, "%s must reject a request without %s", method, tc.name)
	}
}

func TestFuturesComboBotBody(t *testing.T) {
	t.Parallel()
	requireFuturesComboBotValidation(t, "futuresComboBotBody", func(r *CreateFuturesComboBotRequest) error {
		_, err := futuresComboBotBody(r)
		return err
	})
	r := testFuturesComboBotRequest()
	r.SymbolSettings[1].Side = 0
	_, err := futuresComboBotBody(r)
	assert.ErrorContains(t, err, "symbol setting 1", "futuresComboBotBody should name the symbol setting it rejects")

	body, err := futuresComboBotBody(testFuturesComboBotRequest())
	require.NoError(t, err, "futuresComboBotBody must not error")
	exp := map[string]any{
		"leverage":             "5",
		"init_margin":          "500",
		"adjust_position_mode": uint64(3),
		"symbol_settings": []map[string]any{
			{"symbol": "BTCUSDT", "target_position_percent": "0.5", "side": uint64(1)},
			{"symbol": "ETHUSDT", "target_position_percent": "0.5", "side": uint64(2)},
		},
		"adjust_position_percent":       "0.05",
		"adjust_position_time_interval": int64(3600),
		"sl_percent":                    "0.2",
		"tp_percent":                    "0.5",
		"trailing_stop_percent":         "0.05",
	}
	assert.Equal(t, exp, body, "futuresComboBotBody should send every setting, with decimals as strings and the rebalancing interval in seconds")

	body, err = futuresComboBotBody(&CreateFuturesComboBotRequest{
		Leverage:           2,
		InitialMargin:      100,
		AdjustPositionMode: 4,
		SymbolSettings:     []FuturesComboSymbolSettingRequest{{Symbol: "SOLUSDT", TargetPositionPercent: 1, Side: 1}},
	})
	require.NoError(t, err, "futuresComboBotBody must not error")
	exp = map[string]any{
		"leverage":             "2",
		"init_margin":          "100",
		"adjust_position_mode": uint64(4),
		"symbol_settings":      []map[string]any{{"symbol": "SOLUSDT", "target_position_percent": "1", "side": uint64(1)}},
	}
	assert.Equal(t, exp, body, "futuresComboBotBody should leave out the optional settings left unset")
}

func TestGetFuturesComboBotLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesComboBotLimit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesComboBotLimit must reject a nil request")
	requireFuturesComboBotValidation(t, "GetFuturesComboBotLimit", func(r *CreateFuturesComboBotRequest) error {
		_, err := e.GetFuturesComboBotLimit(t.Context(), &GetFuturesComboBotLimitRequest{CreateFuturesComboBotRequest: *r})
		return err
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesComboBotLimit(t.Context(), &GetFuturesComboBotLimitRequest{CreateFuturesComboBotRequest: *testFuturesComboBotRequest(), NeedToSlippage: true})
	require.NoError(t, err, "GetFuturesComboBotLimit must not error")
	if mockTests {
		exp := &FuturesComboBotLimitResponse{
			StatusCode:   200,
			DebugMessage: "limit check passed",
			CheckCode:    "LIMIT_CHECK_CODE_SUCCESS_UNSPECIFIED",
			InitialMargin: BotLimitRange{
				Min: 352.3,
				Max: 576840,
			},
			Leverage: BotLimitRange{
				Min: 1,
				Max: 100,
			},
			AdjustPositionPercent: BotLimitRange{
				Min: 0.01,
				Max: 0.5,
			},
			AdjustPositionTimeInterval: BotLimitRange{
				Min: 1800,
				Max: 2419200,
			},
			SLPercent: BotLimitRange{
				Min: 0.01,
				Max: 0.99,
			},
			TPPercent: BotLimitRange{
				Min: 0.01,
				Max: 100,
			},
			TrailingStopPercent: BotLimitRange{
				Min: 0.05,
				Max: 0.99,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesComboBotLimit should decode every field")
		unbalanced := testFuturesComboBotRequest()
		unbalanced.SymbolSettings[1].TargetPositionPercent = 0.7
		result, err = e.GetFuturesComboBotLimit(t.Context(), &GetFuturesComboBotLimitRequest{CreateFuturesComboBotRequest: *unbalanced})
		assert.ErrorIs(t, err, errBotInvalidParameters, "GetFuturesComboBotLimit should report target position percents that do not sum to 1")
		assert.ErrorContains(t, err, "target position percents do not sum to 1", "GetFuturesComboBotLimit should include the debug message")
		assert.Nil(t, result, "GetFuturesComboBotLimit should return no limits with a failed status code")
		denied := testFuturesComboBotRequest()
		denied.Leverage = 10
		result, err = e.GetFuturesComboBotLimit(t.Context(), &GetFuturesComboBotLimitRequest{CreateFuturesComboBotRequest: *denied})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesComboBotLimit should return an error reply")
		assert.Nil(t, result, "GetFuturesComboBotLimit should return no limits with an error reply")
		return
	}
	assert.NotEmpty(t, result.CheckCode, "GetFuturesComboBotLimit should return a check code")
}

func TestCreateFuturesComboBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesComboBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFuturesComboBot must reject a nil request")
	requireFuturesComboBotValidation(t, "CreateFuturesComboBot", func(r *CreateFuturesComboBotRequest) error {
		_, err := e.CreateFuturesComboBot(t.Context(), r)
		return err
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFuturesComboBot(t.Context(), testFuturesComboBotRequest())
	require.NoError(t, err, "CreateFuturesComboBot must not error")
	if mockTests {
		exp := &CreateBotResponse{
			StatusCode:   200,
			BotID:        "612323083000239723",
			DebugMessage: "bot created",
		}
		assert.Equal(t, exp, result, "CreateFuturesComboBot should decode every field")
		banned := testFuturesComboBotRequest()
		banned.InitialMargin = 1000
		_, err = e.CreateFuturesComboBot(t.Context(), banned)
		assert.ErrorIs(t, err, errBotUserBanned, "CreateFuturesComboBot should report a banned user")
		assert.ErrorContains(t, err, testBotBanReason, "CreateFuturesComboBot should include the ban reason")
		denied := testFuturesComboBotRequest()
		denied.InitialMargin = 2000
		result, err = e.CreateFuturesComboBot(t.Context(), denied)
		assert.ErrorIs(t, err, errAPIResponse, "CreateFuturesComboBot should return an error reply")
		assert.Nil(t, result, "CreateFuturesComboBot should return no bot with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CreateFuturesComboBot should return a bot ID")
}

func TestCloseFuturesComboBot(t *testing.T) {
	t.Parallel()
	_, err := e.CloseFuturesComboBot(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "CloseFuturesComboBot must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseFuturesComboBot(t.Context(), "612324421335537259")
	require.NoError(t, err, "CloseFuturesComboBot must not error")
	if mockTests {
		exp := &CloseBotResponse{
			StatusCode:   200,
			DebugMessage: "close request accepted",
		}
		assert.Equal(t, exp, result, "CloseFuturesComboBot should decode every field")
		result, err = e.CloseFuturesComboBot(t.Context(), "612324421335537260")
		assert.ErrorIs(t, err, errBotInvalidParameters, "CloseFuturesComboBot should report an unknown bot")
		assert.Nil(t, result, "CloseFuturesComboBot should return no result with a failed status code")
		result, err = e.CloseFuturesComboBot(t.Context(), "612324421335537261")
		assert.ErrorIs(t, err, errAPIResponse, "CloseFuturesComboBot should return an error reply")
		assert.Nil(t, result, "CloseFuturesComboBot should return no result with an error reply")
		return
	}
	assert.NotNil(t, result, "CloseFuturesComboBot should return a result")
}

func TestGetFuturesComboBotDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesComboBotDetail(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "GetFuturesComboBotDetail must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesComboBotDetail(t.Context(), "612324421335537259")
	require.NoError(t, err, "GetFuturesComboBotDetail must not error")
	if mockTests {
		exp := &FuturesComboBotDetailResponse{
			StatusCode:   200,
			DebugMessage: "query detail ok",
			Detail: FuturesComboBotDetail{
				BotID:                 "612324421335537259",
				BotMode:               "BOT_MODE_MIX",
				BotDisplayStatus:      "BOT_DISPLAY_STATUS_COMPLETED",
				BotName:               "BTC ETH",
				Leverage:              5,
				TotalMargin:           500,
				Equity:                496.24,
				TotalPNL:              -3.7543,
				TotalPNLPercent:       -0.0075,
				ROI:                   -0.0075,
				TotalAPR:              -0.1825,
				RealizedPNL:           -3.7543,
				UnrealizedPNL:         0.0012,
				TotalExecutionFee:     0.7061,
				FundingFee:            -0.0213,
				TotalPositionValue:    2481.2,
				MarginBalance:         496.24,
				AvailableBalance:      120.5,
				InitialMarginRate:     0.2,
				MaintenanceMarginRate: 0.005,
				SymbolSettings: []FuturesComboSymbolSetting{
					{
						Symbol:                "BTCUSDT",
						BaseToken:             currency.BTC,
						QuoteToken:            currency.USDT,
						TargetPositionPercent: 0.5,
						Side:                  "SIDE_LONG",
						SymbolID:              5,
					},
					{
						Symbol:                "ETHUSDT",
						BaseToken:             currency.ETH,
						QuoteToken:            currency.USDT,
						TargetPositionPercent: 0.5,
						Side:                  "SIDE_SHORT",
						SymbolID:              6,
					},
				},
				AdjustPositionMode:         "ADJUST_POSITION_MODE_TIME_OR_PERCENT",
				AdjustPositionPercent:      0.05,
				AdjustPositionTimeInterval: 3600,
				AdjustedPositionNumber:     2,
				LastAdjustPositionTime:     types.Time(time.Unix(1774503406, 0)),
				SLPercent:                  0.2,
				TPPercent:                  0.5,
				TrailingStopPercent:        0.05,
				StopType:                   "BOT_STOP_TYPE_BY_USER",
				CloseCode:                  "BOT_CLOSE_CODE_CANCELED_MANUALLY",
				ExitEquity:                 496.24,
				SLExitEquity:               400,
				TPExitEquity:               750,
				MidwayTransfer:             20,
				SettlementAssets:           "496.24566099 USDT",
				FollowNumber:               3,
				UsedRewardAmount:           10,
				UsedRewardID:               "80209",
				TotalBonus:                 1.5,
				RunTimeDuration:            14,
				CreateTime:                 types.Time(time.Unix(1774503396, 0)),
				EndTime:                    types.Time(time.Unix(1774503410, 0)),
				LastModifySettingTime:      types.Time(time.Unix(1774503400, 0)),
				WebsocketToken:             "a3f9c2e1b7d04e5f",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesComboBotDetail should decode every field")
		result, err = e.GetFuturesComboBotDetail(t.Context(), "612324421335537260")
		assert.ErrorIs(t, err, errBotInvalidParameters, "GetFuturesComboBotDetail should report an unknown bot")
		assert.Nil(t, result, "GetFuturesComboBotDetail should return no detail with a failed status code")
		result, err = e.GetFuturesComboBotDetail(t.Context(), "612324421335537261")
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesComboBotDetail should return an error reply")
		assert.Nil(t, result, "GetFuturesComboBotDetail should return no detail with an error reply")
		return
	}
	assert.NotEmpty(t, result.Detail.BotID, "GetFuturesComboBotDetail should return the bot")
}

func TestValidateFuturesGridInput(t *testing.T) {
	t.Parallel()
	_, err := e.ValidateFuturesGridInput(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ValidateFuturesGridInput must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*ValidateFuturesGridInputRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *ValidateFuturesGridInputRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "cell number", mutate: func(r *ValidateFuturesGridInputRequest) { r.CellNumber = 0 }, err: errGridCellNumberRequired},
		{name: "min price", mutate: func(r *ValidateFuturesGridInputRequest) { r.MinPrice = 0 }, err: errGridMinPriceRequired},
		{name: "max price", mutate: func(r *ValidateFuturesGridInputRequest) { r.MaxPrice = 0 }, err: errGridMaxPriceRequired},
		{name: "leverage", mutate: func(r *ValidateFuturesGridInputRequest) { r.Leverage = 0 }, err: errInvalidLeverage},
		{name: "grid type", mutate: func(r *ValidateFuturesGridInputRequest) { r.GridType = 0 }, err: errGridTypeRequired},
		{name: "grid mode", mutate: func(r *ValidateFuturesGridInputRequest) { r.GridMode = 0 }, err: errGridModeRequired},
	} {
		r := &ValidateFuturesGridInputRequest{Symbol: "BTCUSDT", CellNumber: 88, MinPrice: 230000, MaxPrice: 800000, Leverage: 6, GridType: 1, GridMode: 2}
		tc.mutate(r)
		_, err = e.ValidateFuturesGridInput(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "ValidateFuturesGridInput must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ValidateFuturesGridInput(t.Context(), &ValidateFuturesGridInputRequest{
		Symbol:              "BTCUSDT",
		CellNumber:          88,
		MinPrice:            230000,
		MaxPrice:            800000,
		Leverage:            6,
		GridType:            1,
		GridMode:            2,
		StopLossPrice:       200000,
		TakeProfitPrice:     900000,
		TPSLType:            2,
		EntryPrice:          370000,
		StopLossPercent:     10,
		TakeProfitPercent:   20,
		TrailingStopPercent: 0.18,
		InitialMargin:       950,
		MoveUpPrice:         850000,
		MoveDownPrice:       220000,
	})
	require.NoError(t, err, "ValidateFuturesGridInput must not error")
	if mockTests {
		exp := &ValidateFuturesGridInputResponse{
			StatusCode:   200,
			DebugMessage: "validation passed",
			CheckCode:    "FGRID_CHECK_CODE_SUCCESS",
			Investment: GridInputRange{
				From: 7900.3349,
				To:   26748800,
			},
			Profit: GridInputRange{
				From: 8.1959,
				To:   8.1959,
			},
			CellNumber: GridInputRange{
				From: 3,
				To:   1562,
			},
			MinPrice: GridInputRange{
				From: 56930.2,
				To:   1997760.4,
			},
			MaxPrice: GridInputRange{
				From: 479430.7,
				To:   1999999.8,
			},
			Leverage: GridInputRange{
				From: 1,
				To:   50,
			},
			StopLoss: GridInputRange{
				From: 0.5,
				To:   100,
			},
			TakeProfit: GridInputRange{
				From: 0.5,
				To:   500,
			},
			TakeProfitPrice: GridInputRange{
				From: 800000,
				To:   1999999.8,
			},
			StopLossPrice: GridInputRange{
				From: 56930.2,
				To:   230000,
			},
			EntryPrice: GridInputRange{
				From: 142325.4,
				To:   479162.1,
			},
			TrailingStopPercent: GridInputRange{
				From: 0.05,
				To:   0.99,
			},
			LongLiquidationPrice:  195412.6,
			ShortLiquidationPrice: 1305228.4,
			MoveUpPrice: GridInputRange{
				From: 811412.8,
				To:   1999999.8,
			},
			MoveDownPrice: GridInputRange{
				From: 0.1,
				To:   226764.9,
			},
			CellGTV0Percent: 71.88,
		}
		assert.Equal(t, exp, result, "ValidateFuturesGridInput should decode every field")
		result, err = e.ValidateFuturesGridInput(t.Context(), &ValidateFuturesGridInputRequest{Symbol: "XYZUSDT", CellNumber: 88, MinPrice: 230000, MaxPrice: 800000, Leverage: 6, GridType: 1, GridMode: 2})
		assert.ErrorIs(t, err, errBotInvalidParameters, "ValidateFuturesGridInput should report a symbol that does not exist")
		assert.ErrorContains(t, err, "FGRID_CHECK_CODE_SYMBOL_NOT_EXISTS", "ValidateFuturesGridInput should include the check code")
		assert.Nil(t, result, "ValidateFuturesGridInput should return no ranges with a failed status code")
		result, err = e.ValidateFuturesGridInput(t.Context(), &ValidateFuturesGridInputRequest{Symbol: "BTCUSDT", CellNumber: 88, MinPrice: 230000, MaxPrice: 800000, Leverage: 6, GridType: 1, GridMode: 2})
		assert.ErrorIs(t, err, errAPIResponse, "ValidateFuturesGridInput should return an error reply")
		assert.Nil(t, result, "ValidateFuturesGridInput should return no ranges with an error reply")
		return
	}
	assert.NotEmpty(t, result.CheckCode, "ValidateFuturesGridInput should return a check code")
}

// testFuturesGridBotRequest returns the futures grid bot settings the fixtures record
func testFuturesGridBotRequest() *CreateFuturesGridBotRequest {
	return &CreateFuturesGridBotRequest{
		Symbol:              "BTCUSDT",
		GridMode:            2,
		MinPrice:            230000,
		MaxPrice:            800000,
		CellNumber:          88,
		Leverage:            6,
		GridType:            1,
		TotalInvestment:     950,
		TakeProfitPercent:   0.28,
		StopLossPercent:     0.1,
		TakeProfitPrice:     900000,
		StopLossPrice:       200000,
		TPSLType:            4,
		EntryPrice:          370000,
		TrailingStopPercent: 0.18,
		MoveUpPrice:         850000,
		MoveDownPrice:       220000,
	}
}

func TestCreateFuturesGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFuturesGridBot must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*CreateFuturesGridBotRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *CreateFuturesGridBotRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "cell number", mutate: func(r *CreateFuturesGridBotRequest) { r.CellNumber = 0 }, err: errGridCellNumberRequired},
		{name: "min price", mutate: func(r *CreateFuturesGridBotRequest) { r.MinPrice = 0 }, err: errGridMinPriceRequired},
		{name: "max price", mutate: func(r *CreateFuturesGridBotRequest) { r.MaxPrice = 0 }, err: errGridMaxPriceRequired},
		{name: "leverage", mutate: func(r *CreateFuturesGridBotRequest) { r.Leverage = 0 }, err: errInvalidLeverage},
		{name: "grid type", mutate: func(r *CreateFuturesGridBotRequest) { r.GridType = 0 }, err: errGridTypeRequired},
		{name: "grid mode", mutate: func(r *CreateFuturesGridBotRequest) { r.GridMode = 0 }, err: errGridModeRequired},
		{name: "total investment", mutate: func(r *CreateFuturesGridBotRequest) { r.TotalInvestment = 0 }, err: errGridTotalInvestmentRequired},
	} {
		r := testFuturesGridBotRequest()
		tc.mutate(r)
		_, err = e.CreateFuturesGridBot(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "CreateFuturesGridBot must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFuturesGridBot(t.Context(), testFuturesGridBotRequest())
	require.NoError(t, err, "CreateFuturesGridBot must not error")
	if mockTests {
		exp := &CreateFuturesGridBotResponse{
			StatusCode:   200,
			BotID:        "612330315406398322",
			CheckCode:    "FGRID_CHECK_CODE_SUCCESS",
			DebugMessage: "bot created",
		}
		assert.Equal(t, exp, result, "CreateFuturesGridBot should decode every field")
		banned := testFuturesGridBotRequest()
		banned.TotalInvestment = 1900
		_, err = e.CreateFuturesGridBot(t.Context(), banned)
		assert.ErrorIs(t, err, errBotUserBanned, "CreateFuturesGridBot should report a banned user")
		assert.ErrorContains(t, err, testBotBanReason, "CreateFuturesGridBot should include the ban reason")
		denied := testFuturesGridBotRequest()
		denied.TotalInvestment = 2850
		result, err = e.CreateFuturesGridBot(t.Context(), denied)
		assert.ErrorIs(t, err, errAPIResponse, "CreateFuturesGridBot should return an error reply")
		assert.Nil(t, result, "CreateFuturesGridBot should return no bot with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CreateFuturesGridBot should return a bot ID")
}

func TestCloseFuturesGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CloseFuturesGridBot(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "CloseFuturesGridBot must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseFuturesGridBot(t.Context(), "612330315406398322")
	require.NoError(t, err, "CloseFuturesGridBot must not error")
	if mockTests {
		exp := &CloseFuturesGridBotResponse{
			StatusCode:   200,
			BotID:        "612330315406398322",
			DebugMessage: "bot has already been canceled",
		}
		assert.Equal(t, exp, result, "CloseFuturesGridBot should decode every field")
		result, err = e.CloseFuturesGridBot(t.Context(), "612330315406398323")
		assert.ErrorIs(t, err, errBotInvalidParameters, "CloseFuturesGridBot should report an unknown bot")
		assert.Nil(t, result, "CloseFuturesGridBot should return no result with a failed status code")
		result, err = e.CloseFuturesGridBot(t.Context(), "612330315406398324")
		assert.ErrorIs(t, err, errAPIResponse, "CloseFuturesGridBot should return an error reply")
		assert.Nil(t, result, "CloseFuturesGridBot should return no result with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CloseFuturesGridBot should return the closed bot's ID")
}

func TestGetFuturesGridBotDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesGridBotDetail(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "GetFuturesGridBotDetail must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesGridBotDetail(t.Context(), "612330315406398322")
	require.NoError(t, err, "GetFuturesGridBotDetail must not error")
	if mockTests {
		exp := &FuturesGridBotDetailResponse{
			StatusCode:   200,
			DebugMessage: "query detail ok",
			Detail: FuturesGridBotDetail{
				BotID:                    "612330315406398322",
				Symbol:                   "BTCUSDT",
				BaseToken:                currency.BTC,
				QuoteToken:               currency.USDT,
				Status:                   "FUTURE_GRID_STATUS_COMPLETED",
				GridMode:                 "FUTURE_GRID_MODE_LONG",
				GridType:                 "FUTURE_GRID_TYPE_GEOMETRIC",
				MinPrice:                 230000,
				MaxPrice:                 800000,
				CellNumber:               88,
				Leverage:                 6,
				RealLeverage:             5.8,
				TotalInvestment:          950,
				TotalValue:               5510.4,
				CurrentPosition:          0.012,
				PNL:                      12.5,
				PNLPercent:               0.0132,
				GridProfit:               10.2,
				GridAPR:                  0.365,
				TotalAPR:                 0.42,
				RealisedPNL:              10.2,
				UnrealisedPNL:            2.3,
				FundingFee:               -0.42,
				PositionBalance:          918.4,
				AvailableBalance:         31.6,
				TotalOrderBalance:        112.8,
				Equity:                   962.5,
				LastPrice:                459201.5,
				MarkPrice:                459188.2,
				LiquidationPrice:         195412.6,
				StopLossPercent:          0.1,
				TakeProfitPercent:        0.28,
				StopLossPrice:            200000,
				TakeProfitPrice:          900000,
				TPSLType:                 "TP_SL_TYPE_TP_PERCENT_SL_PRICE",
				EntryPrice:               370000,
				RealClosePrice:           459100.3,
				MinPricePrecision:        0.01,
				TickSize:                 0.1,
				TrailingStopPercent:      0.18,
				TrailingStopExitEquity:   955.1,
				CloseReason:              "F_GRID_BOT_STOP_TYPE_TRIGGER_FBU_FAIL",
				BotCloseCode:             "BOT_CLOSE_CODE_FAILED_INITIATION",
				FuturesPositionSide:      "FUTURES_POSITION_SIDE_LONG",
				ArbitrageNumber:          14,
				ArbitrageNumber24Hour:    3,
				MinProfit:                8.1959,
				MaxProfit:                8.1959,
				MoveUpPrice:              850000,
				MoveDownPrice:            220000,
				CurrentMinPrice:          230000,
				CurrentMaxPrice:          800000,
				InitialMarginRate:        0.1,
				MaintenanceMarginRate:    0.005,
				AccountType:              "BOT_ACCOUNT_TYPE_FUND",
				AllowFollow:              true,
				FollowNumber:             2,
				CopyTradeIsMaster:        true,
				CopyTradeFollowerNumber:  2,
				UsedRewardAmount:         5,
				UsedRewardID:             "80209",
				InitialBonus:             1,
				UsedBonusAmount:          0.5,
				SettlementAssets:         "950 USDT",
				CumulativeWithdrewAmount: 1.5,
				CurrentProfit:            12.5,
				CurrentPercent:           0.032,
				TakerFeeRate:             0.00028,
				MakerFeeRate:             0.0001,
				OperationTime:            1110,
				CreateTime:               types.Time(time.UnixMilli(1774506909414)),
				ModifyTime:               types.Time(time.UnixMilli(1774506910524)),
				EndTime:                  types.Time(time.UnixMilli(1774506910522)),
				ADLRankIndicator:         2,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesGridBotDetail should decode every field")
		result, err = e.GetFuturesGridBotDetail(t.Context(), "612330315406398323")
		assert.ErrorIs(t, err, errBotInvalidParameters, "GetFuturesGridBotDetail should report an unknown bot")
		assert.Nil(t, result, "GetFuturesGridBotDetail should return no detail with a failed status code")
		result, err = e.GetFuturesGridBotDetail(t.Context(), "612330315406398324")
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesGridBotDetail should return an error reply")
		assert.Nil(t, result, "GetFuturesGridBotDetail should return no detail with an error reply")
		return
	}
	assert.NotEmpty(t, result.Detail.BotID, "GetFuturesGridBotDetail should return the bot")
}

func TestGetFuturesMartingaleBotLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMartingaleBotLimit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesMartingaleBotLimit must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*GetFuturesMartingaleBotLimitRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *GetFuturesMartingaleBotLimitRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "martingale mode", mutate: func(r *GetFuturesMartingaleBotLimitRequest) { r.MartingaleMode = 0 }, err: errMartingaleModeRequired},
		{name: "leverage", mutate: func(r *GetFuturesMartingaleBotLimitRequest) { r.Leverage = 0 }, err: errInvalidLeverage},
	} {
		r := &GetFuturesMartingaleBotLimitRequest{Symbol: "MNTUSDT", MartingaleMode: 1, Leverage: 5}
		tc.mutate(r)
		_, err = e.GetFuturesMartingaleBotLimit(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "GetFuturesMartingaleBotLimit must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesMartingaleBotLimit(t.Context(), &GetFuturesMartingaleBotLimitRequest{
		Symbol:             "MNTUSDT",
		MartingaleMode:     1,
		Leverage:           5,
		PriceFloatPercent:  0.16,
		AddPositionPercent: 1.8,
		AddPositionNumber:  5,
		InitialMargin:      1000,
		RoundTPPercent:     0.1,
		SLPercent:          0.5,
		EntryPrice:         0.7,
		NeedToSlippage:     true,
	})
	require.NoError(t, err, "GetFuturesMartingaleBotLimit must not error")
	if mockTests {
		exp := &FuturesMartingaleBotLimitResponse{
			StatusCode:   200,
			DebugMessage: "limit check passed",
			CheckCode:    "F_MART_LIMIT_CHECK_CODE_F_MART_CHECK_CODE_SUCCESS_UNSPECIFIED",
			PriceFloatPercent: BotLimitRange{
				Min: 0.001,
				Max: 0.199,
			},
			AddPositionPercent: BotLimitRange{
				Min: 1,
				Max: 2,
			},
			AddPositionNumber: BotLimitRange{
				Min: 1,
				Max: 10,
			},
			InitialMargin: BotLimitRange{
				Min: 0.7682,
				Max: 34004.22,
			},
			RoundTPPercent: BotLimitRange{
				Min: 0.01,
				Max: 4.45,
			},
			SLPercent: BotLimitRange{
				Min: 0.01,
				Max: 1,
			},
			EntryPrice: BotLimitRange{
				Min: 0.2105,
				Max: 0.7086,
			},
			Leverage: BotLimitRange{
				Min: 1,
				Max: 20,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesMartingaleBotLimit should decode every field")
		result, err = e.GetFuturesMartingaleBotLimit(t.Context(), &GetFuturesMartingaleBotLimitRequest{Symbol: "MNTUSDT", MartingaleMode: 3, Leverage: 5})
		assert.ErrorIs(t, err, errBotInvalidParameters, "GetFuturesMartingaleBotLimit should report a Martingale mode that is neither long nor short")
		assert.ErrorContains(t, err, "invalid martingale_mode", "GetFuturesMartingaleBotLimit should include the debug message")
		assert.Nil(t, result, "GetFuturesMartingaleBotLimit should return no limits with a failed status code")
		result, err = e.GetFuturesMartingaleBotLimit(t.Context(), &GetFuturesMartingaleBotLimitRequest{Symbol: "MNTUSDT", MartingaleMode: 1, Leverage: 5})
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesMartingaleBotLimit should return an error reply")
		assert.Nil(t, result, "GetFuturesMartingaleBotLimit should return no limits with an error reply")
		return
	}
	assert.NotEmpty(t, result.CheckCode, "GetFuturesMartingaleBotLimit should return a check code")
}

// testFuturesMartingaleBotRequest returns the futures Martingale bot settings the fixtures record
func testFuturesMartingaleBotRequest() *CreateFuturesMartingaleBotRequest {
	return &CreateFuturesMartingaleBotRequest{
		Symbol:             "MNTUSDT",
		MartingaleMode:     1,
		Leverage:           5,
		PriceFloatPercent:  0.16,
		AddPositionPercent: 1.8,
		AddPositionNumber:  5,
		InitialMargin:      650,
		RoundTPPercent:     0.1,
		AutoCycleToggle:    2,
		SLPercent:          0.5,
		EntryPrice:         0.56,
	}
}

func TestCreateFuturesMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateFuturesMartingaleBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateFuturesMartingaleBot must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*CreateFuturesMartingaleBotRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "martingale mode", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.MartingaleMode = 0 }, err: errMartingaleModeRequired},
		{name: "leverage", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.Leverage = 0 }, err: errInvalidLeverage},
		{name: "price float percent", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.PriceFloatPercent = 0 }, err: errPriceFloatPercentRequired},
		{name: "add position percent", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.AddPositionPercent = 0 }, err: errAddPositionPercentRequired},
		{name: "add position number", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.AddPositionNumber = 0 }, err: errAddPositionNumberRequired},
		{name: "initial margin", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.InitialMargin = 0 }, err: errBotInitialMarginRequired},
		{name: "round take-profit percent", mutate: func(r *CreateFuturesMartingaleBotRequest) { r.RoundTPPercent = 0 }, err: errRoundTPPercentRequired},
	} {
		r := testFuturesMartingaleBotRequest()
		tc.mutate(r)
		_, err = e.CreateFuturesMartingaleBot(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "CreateFuturesMartingaleBot must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateFuturesMartingaleBot(t.Context(), testFuturesMartingaleBotRequest())
	require.NoError(t, err, "CreateFuturesMartingaleBot must not error")
	if mockTests {
		exp := &CreateBotResponse{
			StatusCode:   200,
			BotID:        "612335280740902531",
			DebugMessage: "bot created",
		}
		assert.Equal(t, exp, result, "CreateFuturesMartingaleBot should decode every field")
		banned := testFuturesMartingaleBotRequest()
		banned.InitialMargin = 1300
		_, err = e.CreateFuturesMartingaleBot(t.Context(), banned)
		assert.ErrorIs(t, err, errBotUserBanned, "CreateFuturesMartingaleBot should report a banned user")
		assert.ErrorContains(t, err, testBotBanReason, "CreateFuturesMartingaleBot should include the ban reason")
		denied := testFuturesMartingaleBotRequest()
		denied.InitialMargin = 1950
		result, err = e.CreateFuturesMartingaleBot(t.Context(), denied)
		assert.ErrorIs(t, err, errAPIResponse, "CreateFuturesMartingaleBot should return an error reply")
		assert.Nil(t, result, "CreateFuturesMartingaleBot should return no bot with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CreateFuturesMartingaleBot should return a bot ID")
}

func TestCloseFuturesMartingaleBot(t *testing.T) {
	t.Parallel()
	_, err := e.CloseFuturesMartingaleBot(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "CloseFuturesMartingaleBot must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseFuturesMartingaleBot(t.Context(), "612335280740902531")
	require.NoError(t, err, "CloseFuturesMartingaleBot must not error")
	if mockTests {
		exp := &CloseBotResponse{
			StatusCode:   200,
			DebugMessage: "close request accepted",
		}
		assert.Equal(t, exp, result, "CloseFuturesMartingaleBot should decode every field")
		result, err = e.CloseFuturesMartingaleBot(t.Context(), "612335280740902532")
		assert.ErrorIs(t, err, errBotInvalidParameters, "CloseFuturesMartingaleBot should report an unknown bot")
		assert.Nil(t, result, "CloseFuturesMartingaleBot should return no result with a failed status code")
		result, err = e.CloseFuturesMartingaleBot(t.Context(), "612335280740902533")
		assert.ErrorIs(t, err, errAPIResponse, "CloseFuturesMartingaleBot should return an error reply")
		assert.Nil(t, result, "CloseFuturesMartingaleBot should return no result with an error reply")
		return
	}
	assert.NotNil(t, result, "CloseFuturesMartingaleBot should return a result")
}

func TestGetFuturesMartingaleBotDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMartingaleBotDetail(t.Context(), "")
	require.ErrorIs(t, err, errBotIDRequired, "GetFuturesMartingaleBotDetail must reject an empty bot ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesMartingaleBotDetail(t.Context(), "612335724699611830")
	require.NoError(t, err, "GetFuturesMartingaleBotDetail must not error")
	if mockTests {
		exp := &FuturesMartingaleBotDetailResponse{
			StatusCode:   200,
			DebugMessage: "query detail ok",
			FuturesMartingaleDetail: FuturesMartingaleBotDetail{
				BotID:                          "612335724699611830",
				Symbol:                         "MNTUSDT",
				BaseToken:                      currency.NewCode("MNT"),
				QuoteToken:                     currency.USDT,
				BotDisplayStatus:               "F_MART_BOT_DISPLAY_STATUS_RUNNING",
				FuturesMartingaleMode:          "F_MART_MODE_MARTINGALE_MODE_LONG",
				Leverage:                       5,
				ActualLeverage:                 4.64,
				SingleRoundTargetProfitPercent: 0.1,
				AddPositionNumber:              5,
				AddPositionPercent:             1.8,
				PriceFloatPercent:              0.16,
				EntryPrice:                     0.7039,
				SLPercent:                      0.5,
				TotalMargin:                    1000,
				PNL:                            1.41687,
				PNLPercent:                     0.0014,
				RealizedPNL:                    -0.02684394,
				UnrealizedPNL:                  1.44372,
				NetFundingFee:                  -0.0021,
				TotalAPR:                       0.511,
				CompletedRounds:                2,
				CurrentRound:                   3,
				CurrentAddedPositionNumber:     1,
				PositionSize:                   136.2,
				AveragePositionPrice:           0.7039,
				PositionBalance:                19.36433252,
				OrderBalance:                   34.68510157,
				AvailableBalance:               946.66328797,
				MarkPrice:                      0.70966,
				CurrentPrice:                   0.7145,
				EstimatedLiquidationPrice:      0.5634,
				AutoCycle:                      "AUTO_CYCLE_TOGGLE_AUTO_CYCLE_TOGGLE_DISABLED",
				StopType:                       "F_MART_BOT_STOP_TYPE_STOP_TYPE_BY_USER",
				CloseCode:                      "BOT_CLOSE_CODE_CANCELED_MANUALLY",
				ClosePrice:                     0.7145,
				TickSize:                       0.0001,
				SettlementAssets:               "1001.41687 USDT",
				FollowNumber:                   1,
				UsedRewardAmount:               5,
				UsedRewardID:                   "80209",
				TotalBonus:                     0.5,
				OperationTime:                  56,
				CreateTime:                     types.Time(time.Unix(1774510133, 0)),
				ModifyTime:                     types.Time(time.Unix(1774510150, 0)),
				EndTime:                        types.Time(time.Unix(1774510189, 0)),
				RunningDuration:                56,
				ADLRankIndicator:               2,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesMartingaleBotDetail should decode every field")
		result, err = e.GetFuturesMartingaleBotDetail(t.Context(), "612335724699611831")
		assert.ErrorIs(t, err, errBotInvalidParameters, "GetFuturesMartingaleBotDetail should report an unknown bot")
		assert.Nil(t, result, "GetFuturesMartingaleBotDetail should return no detail with a failed status code")
		result, err = e.GetFuturesMartingaleBotDetail(t.Context(), "612335724699611832")
		assert.ErrorIs(t, err, errAPIResponse, "GetFuturesMartingaleBotDetail should return an error reply")
		assert.Nil(t, result, "GetFuturesMartingaleBotDetail should return no detail with an error reply")
		return
	}
	assert.NotEmpty(t, result.FuturesMartingaleDetail.BotID, "GetFuturesMartingaleBotDetail should return the bot")
}

func TestValidateSpotGridInput(t *testing.T) {
	t.Parallel()
	_, err := e.ValidateSpotGridInput(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ValidateSpotGridInput must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*ValidateSpotGridInputRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *ValidateSpotGridInputRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "cell number", mutate: func(r *ValidateSpotGridInputRequest) { r.CellNumber = 0 }, err: errGridCellNumberRequired},
		{name: "min price", mutate: func(r *ValidateSpotGridInputRequest) { r.MinPrice = 0 }, err: errGridMinPriceRequired},
		{name: "max price", mutate: func(r *ValidateSpotGridInputRequest) { r.MaxPrice = 0 }, err: errGridMaxPriceRequired},
	} {
		r := &ValidateSpotGridInputRequest{Symbol: "MNTUSDT", CellNumber: 66, MinPrice: 0.3, MaxPrice: 1.2}
		tc.mutate(r)
		_, err = e.ValidateSpotGridInput(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "ValidateSpotGridInput must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ValidateSpotGridInput(t.Context(), &ValidateSpotGridInputRequest{
		Symbol:              "MNTUSDT",
		CellNumber:          66,
		MinPrice:            0.3,
		MaxPrice:            1.2,
		InvestMode:          2,
		BaseInvestment:      300,
		QuoteInvestment:     100,
		StopLoss:            0.25,
		TakeProfit:          1.5,
		EntryPrice:          0.4,
		TrailingStopPercent: 0.05,
		EnableTrailing:      true,
		LimitUpPrice:        2,
	})
	require.NoError(t, err, "ValidateSpotGridInput must not error")
	if mockTests {
		exp := &ValidateSpotGridInputResponse{
			StatusCode:   200,
			DebugMessage: "validation passed",
			CheckCode:    "SPOT_CHECK_CODE_SUCCESS_UNSPECIFIED",
			Investment: GridInputRange{
				From: 15.5,
				To:   1366590.08,
			},
			Profit: GridInputRange{
				From: 0.0114,
				To:   0.0453,
			},
			CellNumber: GridInputRange{
				From: 2,
				To:   562,
			},
			MinPrice: GridInputRange{
				From: 0.1568,
				To:   3.92,
			},
			MaxPrice: GridInputRange{
				From: 0.3012,
				To:   40,
			},
			StopLoss: GridInputRange{
				From: 0.01,
				To:   0.3,
			},
			TakeProfit: GridInputRange{
				From: 1.2,
				To:   40,
			},
			EntryPrice: GridInputRange{
				From: 0.2352,
				To:   0.784,
			},
			BaseInvestment: GridInputRange{
				From: 228.36,
				To:   1366590.08,
			},
			CellDistance: 0.0136,
			TrailingStopPercent: GridInputRange{
				From: 0.03,
				To:   0.99,
			},
			LimitUpPrice: GridInputRange{
				From: 1.2112,
				To:   3.9065,
			},
		}
		assert.Equal(t, exp, result, "ValidateSpotGridInput should decode every field")
		result, err = e.ValidateSpotGridInput(t.Context(), &ValidateSpotGridInputRequest{Symbol: "MNTUSDT", CellNumber: 66, MinPrice: 0.3, MaxPrice: 1.2, InvestMode: 1})
		assert.ErrorIs(t, err, errBotInvalidParameters, "ValidateSpotGridInput should report a base investment mode without a base investment")
		assert.ErrorContains(t, err, "base_investment is required when invest_mode is 1", "ValidateSpotGridInput should include the debug message")
		assert.Nil(t, result, "ValidateSpotGridInput should return no ranges with a failed status code")
		result, err = e.ValidateSpotGridInput(t.Context(), &ValidateSpotGridInputRequest{Symbol: "MNTUSDT", CellNumber: 66, MinPrice: 0.3, MaxPrice: 1.2})
		assert.ErrorIs(t, err, errAPIResponse, "ValidateSpotGridInput should return an error reply")
		assert.Nil(t, result, "ValidateSpotGridInput should return no ranges with an error reply")
		return
	}
	assert.NotEmpty(t, result.CheckCode, "ValidateSpotGridInput should return a check code")
}

// testSpotGridBotRequest returns the spot grid bot settings the fixtures record
func testSpotGridBotRequest() *CreateSpotGridBotRequest {
	return &CreateSpotGridBotRequest{
		Symbol:              "MNTUSDT",
		MaxPrice:            1.0968,
		MinPrice:            0.3656,
		CellNumber:          50,
		InvestMode:          2,
		BaseInvestment:      100,
		QuoteInvestment:     200,
		EntryPrice:          0.5,
		StopLossPrice:       0.3,
		TakeProfitPrice:     1.2,
		TrailingStopPercent: 0.05,
		EnableTrailing:      true,
		LimitUpPrice:        1.5,
	}
}

func TestCreateSpotGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpotGridBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpotGridBot must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*CreateSpotGridBotRequest)
		err    error
	}{
		{name: "symbol", mutate: func(r *CreateSpotGridBotRequest) { r.Symbol = "" }, err: errSymbolMissing},
		{name: "cell number", mutate: func(r *CreateSpotGridBotRequest) { r.CellNumber = 0 }, err: errGridCellNumberRequired},
		{name: "min price", mutate: func(r *CreateSpotGridBotRequest) { r.MinPrice = 0 }, err: errGridMinPriceRequired},
		{name: "max price", mutate: func(r *CreateSpotGridBotRequest) { r.MaxPrice = 0 }, err: errGridMaxPriceRequired},
	} {
		r := testSpotGridBotRequest()
		tc.mutate(r)
		_, err = e.CreateSpotGridBot(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "CreateSpotGridBot must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateSpotGridBot(t.Context(), testSpotGridBotRequest())
	require.NoError(t, err, "CreateSpotGridBot must not error")
	if mockTests {
		exp := &CreateSpotGridBotResponse{
			StatusCode:   200,
			GridID:       "612340768081708828",
			DebugMessage: "grid created",
		}
		assert.Equal(t, exp, result, "CreateSpotGridBot should decode every field")
		banned := testSpotGridBotRequest()
		banned.QuoteInvestment = 400
		_, err = e.CreateSpotGridBot(t.Context(), banned)
		assert.ErrorIs(t, err, errBotUserBanned, "CreateSpotGridBot should report a banned user")
		assert.ErrorContains(t, err, testBotBanReason, "CreateSpotGridBot should include the ban reason")
		denied := testSpotGridBotRequest()
		denied.QuoteInvestment = 600
		result, err = e.CreateSpotGridBot(t.Context(), denied)
		assert.ErrorIs(t, err, errAPIResponse, "CreateSpotGridBot should return an error reply")
		assert.Nil(t, result, "CreateSpotGridBot should return no grid with an error reply")
		return
	}
	assert.NotEmpty(t, result.GridID, "CreateSpotGridBot should return a grid ID")
}

func TestCloseSpotGridBot(t *testing.T) {
	t.Parallel()
	_, err := e.CloseSpotGridBot(t.Context(), "", 3)
	require.ErrorIs(t, err, errGridIDRequired, "CloseSpotGridBot must reject an empty grid ID")
	_, err = e.CloseSpotGridBot(t.Context(), "612340768081708828", 0)
	require.ErrorIs(t, err, errBotCloseModeRequired, "CloseSpotGridBot must reject an empty close mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseSpotGridBot(t.Context(), "612340768081708828", 3)
	require.NoError(t, err, "CloseSpotGridBot must not error")
	if mockTests {
		exp := &CloseBotResponse{
			StatusCode:   200,
			DebugMessage: "close request accepted",
		}
		assert.Equal(t, exp, result, "CloseSpotGridBot should decode every field")
		_, err = e.CloseSpotGridBot(t.Context(), "612340768081708829", 3)
		assert.ErrorIs(t, err, errBotNotClosable, "CloseSpotGridBot should report a grid that cannot be closed")
		result, err = e.CloseSpotGridBot(t.Context(), "612340768081708830", 3)
		assert.ErrorIs(t, err, errAPIResponse, "CloseSpotGridBot should return an error reply")
		assert.Nil(t, result, "CloseSpotGridBot should return no result with an error reply")
		return
	}
	assert.NotNil(t, result, "CloseSpotGridBot should return a result")
}

func TestGetSpotGridBotDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotGridBotDetail(t.Context(), "")
	require.ErrorIs(t, err, errGridIDRequired, "GetSpotGridBotDetail must reject an empty grid ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotGridBotDetail(t.Context(), "612340768081708828")
	require.NoError(t, err, "GetSpotGridBotDetail must not error")
	if mockTests {
		exp := &SpotGridBotDetailResponse{
			StatusCode:   200,
			DebugMessage: "query detail ok",
			Detail: SpotGridBotDetail{
				GridID:                   "612340768081708828",
				Symbol:                   "MNTUSDT",
				BaseToken:                currency.NewCode("MNT"),
				QuoteToken:               currency.USDT,
				Status:                   "RUNNING",
				MaxPrice:                 1.0968,
				MinPrice:                 0.3656,
				CellNumber:               50,
				TotalInvestment:          200,
				TotalProfit:              40.4186,
				GridProfit:               61.3113,
				GridAPR:                  111.893143923675,
				TotalAPR:                 73.763945,
				ArbitrageNumber:          291,
				ArbitrageNumber24Hour:    12,
				CurrentPrice:             0.7279,
				CurrentProfit:            40.4186,
				CurrentPercent:           0.202093,
				Equity:                   237.412782,
				EntryPrice:               0.5,
				StopLossPrice:            0.3,
				TakeProfitPrice:          1.2,
				CloseReason:              "UNKNOWN_GridCloseReason",
				BotCloseCode:             "BOT_CLOSE_CODE_UNKNOWN",
				TrailingStopPercent:      0.05,
				IsSupportTrailingStop:    true,
				EnableTrailing:           true,
				LimitUpPrice:             1.5,
				OriginalMaxPrice:         1.0512,
				OriginalMinPrice:         0.32,
				OriginalCellNumber:       48,
				AllowFollow:              true,
				FollowNumber:             4,
				OperationTime:            986587,
				CreateTime:               types.Time(time.Unix(1774513139, 0)),
				EndTime:                  types.Time(time.Unix(1774514125, 0)),
				UsedRewardAmount:         5,
				SettlementAssets:         "237.41 USDT",
				AccountType:              "BOT_ACCOUNT_TYPE_FUND",
				CumulativeWithdrewAmount: 2.5,
				SpotSymbolStatus:         "ONLINE",
				UsedRewardID:             "80209",
				TakerFeeRate:             0.0006,
				MakerFeeRate:             0.0004,
				TrailingStopExitEquity:   230.1,
			},
		}
		assert.Equal(t, exp, result, "GetSpotGridBotDetail should decode every field")
		_, err = e.GetSpotGridBotDetail(t.Context(), "612340768081708829")
		assert.ErrorIs(t, err, errBotNotFound, "GetSpotGridBotDetail should report a missing grid")
		result, err = e.GetSpotGridBotDetail(t.Context(), "612340768081708830")
		assert.ErrorIs(t, err, errAPIResponse, "GetSpotGridBotDetail should return an error reply")
		assert.Nil(t, result, "GetSpotGridBotDetail should return no detail with an error reply")
		return
	}
	assert.NotEmpty(t, result.Detail.GridID, "GetSpotGridBotDetail should return the grid")
}

func TestCreateDCABot(t *testing.T) {
	t.Parallel()
	_, err := e.CreateDCABot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateDCABot must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*CreateDCABotRequest)
		err    error
	}{
		{name: "frequency", mutate: func(r *CreateDCABotRequest) { r.Frequency = time.Millisecond }, err: errDCAFrequencyRequired},
		{name: "quote coin", mutate: func(r *CreateDCABotRequest) { r.QuoteCoin = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "pairs", mutate: func(r *CreateDCABotRequest) { r.Pairs = nil }, err: errDCAPairsRequired},
		{name: "a pair's base", mutate: func(r *CreateDCABotRequest) { r.Pairs[0].Base = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "a pair's amount", mutate: func(r *CreateDCABotRequest) { r.Pairs[0].Amount = 0 }, err: order.ErrAmountIsInvalid},
	} {
		r := &CreateDCABotRequest{Frequency: time.Hour, QuoteCoin: currency.USDT, Pairs: []DCABotPairRequest{{Base: currency.BTC, Amount: 10}}}
		tc.mutate(r)
		_, err = e.CreateDCABot(t.Context(), r)
		require.ErrorIsf(t, err, tc.err, "CreateDCABot must reject a request without %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CreateDCABot(t.Context(), &CreateDCABotRequest{
		Frequency:       10 * time.Minute,
		QuoteCoin:       currency.USDT,
		Pairs:           []DCABotPairRequest{{Base: currency.BTC, Amount: 66}, {Base: currency.ETH, Amount: 88}},
		MaxInvestAmount: 2000,
	})
	require.NoError(t, err, "CreateDCABot must not error")
	if mockTests {
		exp := &CreateBotResponse{
			StatusCode:   200,
			BotID:        "612349292702654770",
			DebugMessage: "bot created",
		}
		assert.Equal(t, exp, result, "CreateDCABot should decode every field")
		_, err = e.CreateDCABot(t.Context(), &CreateDCABotRequest{Frequency: time.Hour, QuoteCoin: currency.USDT, Pairs: []DCABotPairRequest{{Base: currency.BTC, Amount: 10}}})
		assert.ErrorIs(t, err, errBotUserBanned, "CreateDCABot should report a banned user")
		assert.ErrorContains(t, err, testBotBanReason, "CreateDCABot should include the ban reason")
		result, err = e.CreateDCABot(t.Context(), &CreateDCABotRequest{Frequency: time.Hour, QuoteCoin: currency.USDT, Pairs: []DCABotPairRequest{{Base: currency.ETH, Amount: 10}}})
		assert.ErrorIs(t, err, errAPIResponse, "CreateDCABot should return an error reply")
		assert.Nil(t, result, "CreateDCABot should return no bot with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CreateDCABot should return a bot ID")
}

func TestCloseDCABot(t *testing.T) {
	t.Parallel()
	_, err := e.CloseDCABot(t.Context(), "", 2)
	require.ErrorIs(t, err, errBotIDRequired, "CloseDCABot must reject an empty bot ID")
	_, err = e.CloseDCABot(t.Context(), "612349292702654770", 0)
	require.ErrorIs(t, err, errBotCloseModeRequired, "CloseDCABot must reject an empty close mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CloseDCABot(t.Context(), "612349292702654770", 2)
	require.NoError(t, err, "CloseDCABot must not error")
	if mockTests {
		exp := &CloseDCABotResponse{
			StatusCode:   200,
			BotID:        "612349292702654770",
			CloseMode:    "DCA_BASE_MODE",
			DebugMessage: "close request accepted",
		}
		assert.Equal(t, exp, result, "CloseDCABot should decode every field")
		_, err = e.CloseDCABot(t.Context(), "612349292702654771", 3)
		assert.ErrorIs(t, err, errBotNotClosable, "CloseDCABot should report a bot that cannot be closed yet")
		result, err = e.CloseDCABot(t.Context(), "612349292702654772", 2)
		assert.ErrorIs(t, err, errAPIResponse, "CloseDCABot should return an error reply")
		assert.Nil(t, result, "CloseDCABot should return no result with an error reply")
		return
	}
	assert.NotEmpty(t, result.BotID, "CloseDCABot should return the closed bot's ID")
}
