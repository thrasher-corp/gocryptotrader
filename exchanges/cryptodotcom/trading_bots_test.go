package cryptodotcom

import (
	"context"
	"math"
	"net/http"
	"strconv"
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

// dcaBotSettings returns the settings of the docs' DCA bot example
func dcaBotSettings() *TradingBotDollarCostAveragingSettings {
	return &TradingBotDollarCostAveragingSettings{
		Allocations:             []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}},
		InvestmentCurrency:      currency.USDT,
		Side:                    SideBuy,
		OrderType:               OrderTypeMarket,
		Quantity:                0.01,
		InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeFixedDuration,
		InvestmentFrequency:     5 * time.Minute,
		FirstInvestmentTime:     time.UnixMilli(1691510460000),
		MaximumOrders:           10,
	}
}

// twapBotSettings returns the settings of the docs' TWAP bot example
func twapBotSettings() *TradingBotTimeWeightedAveragePriceSettings {
	return &TradingBotTimeWeightedAveragePriceSettings{
		InstrumentName: "BTC_USDT",
		Side:           SideBuy,
		OrderType:      OrderTypeMarket,
		Quantity:       1,
		StartTime:      time.UnixMilli(1691510460000),
		EndTime:        time.UnixMilli(1691513460000),
	}
}

// gridBotSettings returns the settings of the docs' grid bot example
func gridBotSettings() *TradingBotGridSettings {
	return &TradingBotGridSettings{
		InstrumentName:  "BTC_USDT",
		TotalInvestment: 1000,
		LowerPrice:      25000,
		UpperPrice:      35000,
		GridMode:        TradingBotGridModeArithmetic,
		GridLevels:      10,
	}
}

// fundingArbitrageBotSettings returns the settings of the docs' funding arbitrage bot example
func fundingArbitrageBotSettings() *TradingBotFundingArbitrageSettings {
	return &TradingBotFundingArbitrageSettings{
		TotalInvestment:    5000,
		InvestmentCurrency: currency.USDT,
		SpotLeg:            TradingBotLeg{InstrumentName: "BTC_USDT", Side: SideBuy},
		PerpetualLeg:       TradingBotLeg{InstrumentName: "BTCUSD-PERP", Side: SideSell},
	}
}

// dcaBotEverySetting returns a selling DCA bot's settings, setting everything but IsBasket
func dcaBotEverySetting() *TradingBotDollarCostAveragingSettings {
	return &TradingBotDollarCostAveragingSettings{
		Allocations:             []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}},
		InvestmentCurrency:      currency.USDT,
		Side:                    SideSell,
		OrderType:               OrderTypeLimit,
		AutoStake:               true,
		Price:                   30000,
		Quantity:                0.01,
		TotalInvestment:         300,
		InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeFixedDuration,
		InvestmentFrequency:     time.Hour,
		FirstInvestmentTime:     time.UnixMilli(1792890000000),
		InvestmentDuration:      86400000,
		MaximumOrders:           24,
		StopPrice:               20000,
		TakeProfitPrice:         40000,
		SimplifiedMode:          true,
	}
}

// dcaBasketBotSettings returns a basket DCA bot's settings, which name no side or order type
func dcaBasketBotSettings() *TradingBotDollarCostAveragingSettings {
	return &TradingBotDollarCostAveragingSettings{
		IsBasket:                true,
		Allocations:             []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}},
		InvestmentCurrency:      currency.USDT,
		TotalInvestment:         1000,
		InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeMonthly,
		InvestmentDuration:      12,
	}
}

// twapLimitBotSettings returns a TWAP bot's settings for selling with limit orders
func twapLimitBotSettings() *TradingBotTimeWeightedAveragePriceSettings {
	return &TradingBotTimeWeightedAveragePriceSettings{
		InstrumentName: "BTC_USDT",
		Side:           SideSell,
		OrderType:      OrderTypeLimit,
		Price:          30000,
		Quantity:       0.5,
		StartTime:      time.UnixMilli(1792890000000),
		EndTime:        time.UnixMilli(1792893600000),
	}
}

// gridBotEverySetting returns a trailing geometric grid bot's settings, setting everything
func gridBotEverySetting() *TradingBotGridSettings {
	return &TradingBotGridSettings{
		InstrumentName:     "BTC_USDT",
		TotalInvestment:    1000,
		LowerPrice:         25000,
		UpperPrice:         35000,
		GridMode:           TradingBotGridModeGeometric,
		GridLevels:         20,
		MarginEnabled:      true,
		TrailingUpEnabled:  true,
		StopPrice:          24000,
		TakeProfitPrice:    36000,
		TrailingUpPriceCap: 50000,
	}
}

// The DCA and TWAP settings above as Crypto.com takes and reports them
const (
	dcaBotSettingsJSON       = `{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"BUY","type":"MARKET","auto_stake":false,"quantity":"0.01","investment_frequency_mode":"FIXED_DURATION","investment_frequency":300000,"first_investment_time":1691510460000,"max_order":"10"}`
	dcaBotEverySettingJSON   = `{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"SELL","type":"LIMIT","auto_stake":true,"price":"30000","quantity":"0.01","total_investment":"300","investment_frequency_mode":"FIXED_DURATION","investment_frequency":3600000,"first_investment_time":1792890000000,"investment_duration":86400000,"max_order":"24","stop_price":"20000","take_profit_price":"40000","simplified_mode":true}`
	dcaBasketBotSettingsJSON = `{"is_basket":true,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"60"},{"instrument_name":"ETH_USDT","allocated_percentage":"40"}],"investment_currency":"USDT","auto_stake":false,"total_investment":"1000","investment_frequency_mode":"MONTHLY","investment_duration":12}`
	twapBotSettingsJSON      = `{"instrument_name":"BTC_USDT","side":"BUY","type":"MARKET","quantity":"1","start_time_millis":1691510460000,"end_time_millis":1691513460000}`
	twapLimitBotSettingsJSON = `{"instrument_name":"BTC_USDT","side":"SELL","type":"LIMIT","price":"30000","quantity":"0.5","start_time_millis":1792890000000,"end_time_millis":1792893600000}`
)

// invalidTradingBotSettingsCase is settings a bot type's validation refuses, and the error it refuses them with
type invalidTradingBotSettingsCase struct {
	name     string
	settings TradingBotSettings
	err      error
}

// invalidTradingBotSettingsCases returns settings each bot type's validation refuses, one case for each check
func invalidTradingBotSettingsCases() []invalidTradingBotSettingsCase {
	dca := func(modify func(*TradingBotDollarCostAveragingSettings)) TradingBotSettings {
		s := dcaBotSettings()
		modify(s)
		return s
	}
	twap := func(modify func(*TradingBotTimeWeightedAveragePriceSettings)) TradingBotSettings {
		s := twapBotSettings()
		modify(s)
		return s
	}
	grid := func(modify func(*TradingBotGridSettings)) TradingBotSettings {
		s := gridBotSettings()
		modify(s)
		return s
	}
	fundingArbitrage := func(modify func(*TradingBotFundingArbitrageSettings)) TradingBotSettings {
		s := fundingArbitrageBotSettings()
		modify(s)
		return s
	}
	return []invalidTradingBotSettingsCase{
		{name: "typed nil DCA settings", settings: (*TradingBotDollarCostAveragingSettings)(nil), err: common.ErrNilPointer},
		{name: "a DCA bot without allocations", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations = nil }), err: errTradingBotAllocationsRequired},
		{name: "a DCA allocation without an instrument", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations[0].InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "a DCA allocation without a percentage", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations[0].AllocatedPercentage = 0 }), err: errInvalidTradingBotAllocation},
		{name: "DCA allocations totalling 150 percent", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations[0].AllocatedPercentage = 150 }), err: errInvalidTradingBotAllocation},
		{name: "DCA allocations totalling 99.9 percent", settings: dca(func(s *TradingBotDollarCostAveragingSettings) {
			s.Allocations = []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 50}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 49.9}}
		}), err: errInvalidTradingBotAllocation},
		{name: "a DCA bot without an investment currency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentCurrency = currency.EMPTYCODE }), err: currency.ErrCurrencyCodeEmpty},
		{name: "a DCA bot without a side", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Side = "" }), err: errInvalidSide},
		{name: "a DCA bot with a stop loss order type", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = OrderTypeStopLoss }), err: errInvalidOrderType},
		{name: "a DCA bot's limit order without a price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = OrderTypeLimit }), err: errInvalidPrice},
		{name: "a DCA bot with a negative quantity", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Quantity = -1 }), err: errInvalidQuantity},
		{name: "a DCA bot with a negative total investment", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.TotalInvestment = -1 }), err: errInvalidAmount},
		{name: "a DCA bot without a quantity or a total investment", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Quantity = 0 }), err: errTradingBotQuantityOrInvestmentRequired},
		{name: "a DCA bot without an investment frequency mode", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequencyMode = "" }), err: errInvalidTradingBotInvestmentFrequencyMode},
		{name: "a fixed duration DCA bot without a frequency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequency = 0 }), err: errInvalidTradingBotInvestmentFrequency},
		{name: "a monthly DCA bot with a negative frequency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) {
			s.InvestmentFrequencyMode = TradingBotInvestmentFrequencyModeMonthly
			s.InvestmentFrequency = -time.Hour
		}), err: errInvalidTradingBotInvestmentFrequency},
		{name: "a DCA bot first investing before 1970", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.FirstInvestmentTime = time.UnixMilli(-1) }), err: errTimeBeforeUnixEpoch},
		{name: "typed nil TWAP settings", settings: (*TradingBotTimeWeightedAveragePriceSettings)(nil), err: common.ErrNilPointer},
		{name: "a TWAP bot without an instrument", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "a TWAP bot without a side", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.Side = "" }), err: errInvalidSide},
		{name: "a TWAP bot without an order type", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType = "" }), err: errInvalidOrderType},
		{name: "a TWAP bot's limit order without a price", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType = OrderTypeLimit }), err: errInvalidPrice},
		{name: "a TWAP bot without a quantity", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.Quantity = 0 }), err: errInvalidQuantity},
		{name: "a TWAP bot without a start time", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = time.Time{} }), err: common.ErrDateUnset},
		{name: "a TWAP bot without an end time", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.EndTime = time.Time{} }), err: common.ErrDateUnset},
		{name: "a TWAP bot starting after it ends", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = s.EndTime.Add(time.Minute) }), err: common.ErrStartAfterEnd},
		{name: "a TWAP bot starting as it ends", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = s.EndTime }), err: common.ErrStartEqualsEnd},
		{name: "a TWAP bot starting before 1970", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = time.UnixMilli(-1) }), err: errTimeBeforeUnixEpoch},
		{name: "typed nil grid settings", settings: (*TradingBotGridSettings)(nil), err: common.ErrNilPointer},
		{name: "a grid bot without an instrument", settings: grid(func(s *TradingBotGridSettings) { s.InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "a grid bot without a total investment", settings: grid(func(s *TradingBotGridSettings) { s.TotalInvestment = 0 }), err: errInvalidAmount},
		{name: "a grid bot without a lower price", settings: grid(func(s *TradingBotGridSettings) { s.LowerPrice = 0 }), err: errInvalidPrice},
		{name: "a grid bot whose upper price is not above its lower", settings: grid(func(s *TradingBotGridSettings) { s.UpperPrice = s.LowerPrice }), err: errInvalidTradingBotPriceRange},
		{name: "a grid bot without a grid mode", settings: grid(func(s *TradingBotGridSettings) { s.GridMode = "" }), err: errInvalidTradingBotGridMode},
		{name: "a grid bot with one grid level", settings: grid(func(s *TradingBotGridSettings) { s.GridLevels = 1 }), err: errInvalidTradingBotGridLevels},
		{name: "typed nil funding arbitrage settings", settings: (*TradingBotFundingArbitrageSettings)(nil), err: common.ErrNilPointer},
		{name: "a funding arbitrage bot without a total investment", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.TotalInvestment = 0 }), err: errInvalidAmount},
		{name: "a funding arbitrage bot without an investment currency", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.InvestmentCurrency = currency.EMPTYCODE }), err: currency.ErrCurrencyCodeEmpty},
		{name: "a funding arbitrage bot's spot leg without an instrument", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.SpotLeg.InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "a funding arbitrage bot's perpetual leg without a side", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.PerpetualLeg.Side = "" }), err: errInvalidSide},
	}
}

// startTradingTestBotLater starts a DCA or TWAP bot's investing an hour from now rather than when the docs' settings
// start it, which has passed, so that a bot a live test creates invests nothing before the test terminates it
func startTradingTestBotLater(settings TradingBotSettings) {
	start := time.Now().Add(time.Hour)
	switch s := settings.(type) {
	case *TradingBotDollarCostAveragingSettings:
		s.FirstInvestmentTime = start
	case *TradingBotTimeWeightedAveragePriceSettings:
		s.StartTime, s.EndTime = start, start.Add(s.EndTime.Sub(s.StartTime))
	}
}

// createTradingTestBot creates a bot from settings for a live test to act on, and returns its ID
func createTradingTestBot(t *testing.T, settings TradingBotSettings) string {
	t.Helper()
	startTradingTestBotLater(settings)
	result, err := e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: settings.BotType(), Settings: settings})
	require.NoError(t, err, "CreateTradingBot must create the bot the test acts on")
	require.NotEmpty(t, result.BotID, "CreateTradingBot must return the ID of the bot the test acts on")
	return result.BotID
}

// terminateTradingBotAfterTest terminates a bot a live test created once the test ends, closing a funding arbitrage
// bot's positions
func terminateTradingBotAfterTest(t *testing.T, botID, botType string) {
	t.Helper()
	t.Cleanup(func() {
		err := e.TerminateTradingBot(context.WithoutCancel(t.Context()), &TerminateTradingBotRequest{BotID: botID, BotType: botType, ClosePosition: botType == TradingBotTypeFundingArbitrage})
		assert.NoErrorf(t, err, "TerminateTradingBot should terminate bot %s, which the test created", botID)
	})
}

func TestCreateTradingBot(t *testing.T) { //nolint:tparallel // the cases run in order on e and before the later calls of CreateTradingBot
	t.Parallel()
	_, err := e.CreateTradingBot(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateTradingBot must error for a nil request")
	_, err = e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{Settings: dcaBotSettings()})
	require.ErrorIs(t, err, errInvalidTradingBotType, "CreateTradingBot must error without a bot type")
	_, err = e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: "MARTINGALE", Settings: dcaBotSettings()})
	require.ErrorIs(t, err, errInvalidTradingBotType, "CreateTradingBot must error for an unknown bot type")
	_, err = e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging})
	require.ErrorIs(t, err, errTradingBotSettingsRequired, "CreateTradingBot must error without settings")
	_, err = e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: TradingBotTypeGrid, Settings: dcaBotSettings()})
	require.ErrorIs(t, err, errTradingBotSettingsMismatch, "CreateTradingBot must error for settings of another bot type")
	// The cases run in order on e, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range invalidTradingBotSettingsCases() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: tc.settings.BotType(), Settings: tc.settings})
			require.ErrorIsf(t, err, tc.err, "CreateTradingBot must error for %s", tc.name)
		})
	}
	// The cases run in order on e, which every test shares, before the calls of CreateTradingBot below
	for _, tc := range []struct {
		settings TradingBotSettings
		botID    string
	}{
		{settings: dcaBotSettings(), botID: "4611686018431234561"},
		{settings: twapBotSettings(), botID: "4611686018431234562"},
		{settings: gridBotSettings(), botID: "4611686018431234563"},
		{settings: fundingArbitrageBotSettings(), botID: "4611686018431234564"},
	} {
		t.Run(tc.settings.BotType(), func(t *testing.T) {
			skipLiveUnlessCanManipulate(t)
			if !mockTests {
				startTradingTestBotLater(tc.settings)
			}
			result, err := e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: tc.settings.BotType(), NotifyOnBotChange: true, Settings: tc.settings})
			require.NoErrorf(t, err, "CreateTradingBot must not error for the docs' %s bot", tc.settings.BotType())
			if mockTests {
				assert.Equalf(t, &CreateTradingBotResponse{BotID: tc.botID}, result, "CreateTradingBot should return the %s bot's ID", tc.settings.BotType())
				return
			}
			require.NotEmptyf(t, result.BotID, "CreateTradingBot must return the %s bot's ID", tc.settings.BotType())
			terminateTradingBotAfterTest(t, result.BotID, tc.settings.BotType())
		})
	}

	// The cases run in order before the calls of CreateTradingBot below, which parallel cases would follow
	for _, tc := range []struct {
		name string
		req  *CreateTradingBotRequest
		exp  string
	}{
		{
			name: "a selling DCA bot with every setting",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging, NotifyOnBotChange: true, DisplayName: "Selling DCA", Settings: dcaBotEverySetting()},
			exp:  `{"bot_type":"DCA","notify_on_bot_change":true,"display_name":"Selling DCA","settings":{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"SELL","type":"LIMIT","auto_stake":true,"price":"30000","quantity":"0.01","total_investment":"300","investment_frequency_mode":"FIXED_DURATION","investment_frequency":3600000,"first_investment_time":1792890000000,"investment_duration":86400000,"max_order":"24","stop_price":"20000","take_profit_price":"40000","simplified_mode":true}}`,
		},
		{
			name: "a basket DCA bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging, Settings: dcaBasketBotSettings()},
			exp:  `{"bot_type":"DCA","notify_on_bot_change":false,"settings":{"is_basket":true,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"60"},{"instrument_name":"ETH_USDT","allocated_percentage":"40"}],"investment_currency":"USDT","auto_stake":false,"total_investment":"1000","investment_frequency_mode":"MONTHLY","investment_duration":12}}`,
		},
		{
			name: "a limit TWAP bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeTimeWeightedAveragePrice, Settings: twapLimitBotSettings()},
			exp:  `{"bot_type":"TWAP","notify_on_bot_change":false,"settings":{"instrument_name":"BTC_USDT","side":"SELL","type":"LIMIT","price":"30000","quantity":"0.5","start_time_millis":1792890000000,"end_time_millis":1792893600000}}`,
		},
		{
			name: "the docs' grid bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeGrid, Settings: gridBotSettings()},
			exp:  `{"bot_type":"GRID","notify_on_bot_change":false,"settings":{"instrument_name":"BTC_USDT","total_investment":"1000","lower_price":"25000","upper_price":"35000","grid_mode":"ARITHMETIC","grid_no":10,"margin_enabled":false,"trailing_up_enabled":false}}`,
		},
		{
			name: "a trailing grid bot with every setting",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeGrid, NotifyOnBotChange: true, DisplayName: "Trailing Grid", Settings: gridBotEverySetting()},
			exp:  `{"bot_type":"GRID","notify_on_bot_change":true,"display_name":"Trailing Grid","settings":{"instrument_name":"BTC_USDT","total_investment":"1000","lower_price":"25000","upper_price":"35000","grid_mode":"GEOMETRIC","grid_no":20,"margin_enabled":true,"trailing_up_enabled":true,"stop_price":"24000","take_profit_price":"36000","trailing_up_price_cap":"50000"}}`,
		},
		{
			name: "the docs' funding arbitrage bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeFundingArbitrage, Settings: fundingArbitrageBotSettings()},
			exp:  `{"bot_type":"FUNDING_ARBITRAGE","notify_on_bot_change":false,"settings":{"total_investment":"5000","investment_currency":"USDT","spot_leg":{"instrument_name":"BTC_USDT","side":"BUY"},"perp_leg":{"instrument_name":"BTCUSD-PERP","side":"SELL"}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/bot/create-trading-bot", tc.exp, `{"id":1,"method":"private/bot/create-trading-bot","code":0,"result":{"bot_id":"1"}}`)
			_, err := ex.CreateTradingBot(t.Context(), tc.req)
			require.NoErrorf(t, err, "CreateTradingBot must not error against the test server for a %s bot", tc.req.BotType)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/bot/create-trading-bot","code":0,"result":null}`).CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: TradingBotTypeGrid, Settings: gridBotSettings()})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateTradingBot should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: TradingBotTypeGrid, Settings: gridBotSettings()})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateTradingBot should return a request failure")
}

func TestTradingBotRequestSignature(t *testing.T) {
	t.Parallel()
	const exp = "bot_typeDCAdisplay_nameBasket DCAnotify_on_bot_changetruesettings" +
		"allocationsallocated_percentage60instrument_nameBTC_USDTallocated_percentage40instrument_nameETH_USDT" +
		"auto_stakefalsefirst_investment_time1792890000000investment_currencyUSDTinvestment_duration86400000" +
		"investment_frequency3600000investment_frequency_modeFIXED_DURATIONis_baskettruemax_order24total_investment1000"
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// readSignedRequest checks the signature covers the params string of the params sent
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		signed, err := paramString(body.Params)
		if !assert.NoError(t, err, "paramString should not error for the params sent") {
			return
		}
		assert.Equal(t, exp, signed, "The signature should cover the nested settings and each allocation, keys in order")
		_, err = w.Write([]byte(`{"id":1,"method":"private/bot/create-trading-bot","code":0,"result":{"bot_id":"1"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err := ex.CreateTradingBot(t.Context(), &CreateTradingBotRequest{
		BotType:           TradingBotTypeDollarCostAveraging,
		NotifyOnBotChange: true,
		DisplayName:       "Basket DCA",
		Settings: &TradingBotDollarCostAveragingSettings{
			IsBasket:                true,
			Allocations:             []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}},
			InvestmentCurrency:      currency.USDT,
			TotalInvestment:         1000,
			InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeFixedDuration,
			InvestmentFrequency:     time.Hour,
			FirstInvestmentTime:     time.UnixMilli(1792890000000),
			InvestmentDuration:      86400000,
			MaximumOrders:           24,
		},
	})
	require.NoError(t, err, "CreateTradingBot must not error against the test server")
}

func TestUpdateTradingBot(t *testing.T) { //nolint:tparallel // the cases run in order on e and before the later calls of UpdateTradingBot
	t.Parallel()
	require.ErrorIs(t, e.UpdateTradingBot(t.Context(), nil), common.ErrNilPointer, "UpdateTradingBot must error for a nil request")
	// The cases run in order on e, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *UpdateTradingBotRequest
		err  error
	}{
		{name: "without a bot ID", req: &UpdateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging}, err: errTradingBotIDRequired},
		{name: "for a bot ID that is not a number", req: &UpdateTradingBotRequest{BotID: "bot-1", BotType: TradingBotTypeDollarCostAveraging}, err: errInvalidTradingBotID},
		{name: "without a bot type", req: &UpdateTradingBotRequest{BotID: "1"}, err: errInvalidTradingBotType},
		{name: "for grid settings on a DCA bot", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotGridUpdateSettings{}}, err: errTradingBotSettingsMismatch},
		{name: "for DCA settings on a TWAP bot", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeTimeWeightedAveragePrice, Settings: &TradingBotDollarCostAveragingUpdateSettings{}}, err: errTradingBotSettingsMismatch},
		{name: "for typed nil DCA settings", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: (*TradingBotDollarCostAveragingUpdateSettings)(nil)}, err: common.ErrNilPointer},
		{name: "for an allocation without an instrument", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{Allocations: []TradingBotAllocation{{AllocatedPercentage: 100}}}}, err: errInstrumentNameRequired},
		{name: "for an allocation without a percentage", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{Allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT"}}}}, err: errInvalidTradingBotAllocation},
		{name: "for a stop loss order type", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{OrderType: OrderTypeStopLoss}}, err: errInvalidOrderType},
		{name: "for a negative DCA price", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{StopPrice: -1}}, err: errInvalidPrice},
		{name: "for a negative DCA quantity", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{Quantity: -1}}, err: errInvalidQuantity},
		{name: "for a negative investment frequency", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, Settings: &TradingBotDollarCostAveragingUpdateSettings{InvestmentFrequency: -time.Hour}}, err: errInvalidTradingBotInvestmentFrequency},
		{name: "for typed nil grid settings", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeGrid, Settings: (*TradingBotGridUpdateSettings)(nil)}, err: common.ErrNilPointer},
		{name: "for a negative grid price", req: &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeGrid, Settings: &TradingBotGridUpdateSettings{TrailingUpPriceCap: -1}}, err: errInvalidPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIsf(t, e.UpdateTradingBot(t.Context(), tc.req), tc.err, "UpdateTradingBot must error %s", tc.name)
		})
	}
	// The cases run in order on e, which every test shares, before the calls of UpdateTradingBot below
	for _, tc := range []struct {
		req      *UpdateTradingBotRequest
		settings TradingBotSettings
	}{
		{
			req: &UpdateTradingBotRequest{
				BotID:             "4611686018431234561",
				BotType:           TradingBotTypeDollarCostAveraging,
				NotifyOnBotChange: true,
				Settings: &TradingBotDollarCostAveragingUpdateSettings{
					Allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}},
					StopPrice:   20000,
				},
			},
			settings: dcaBotSettings(),
		},
		{req: &UpdateTradingBotRequest{BotID: "4611686018431234562", BotType: TradingBotTypeTimeWeightedAveragePrice, DisplayName: "My TWAP Bot"}, settings: twapBotSettings()},
		{req: &UpdateTradingBotRequest{BotID: "4611686018431234563", BotType: TradingBotTypeGrid, NotifyOnBotChange: true, Settings: &TradingBotGridUpdateSettings{StopPrice: 24000}}, settings: gridBotSettings()},
		{req: &UpdateTradingBotRequest{BotID: "4611686018431234564", BotType: TradingBotTypeFundingArbitrage, NotifyOnBotChange: true, DisplayName: "Funding Arb Bot"}, settings: fundingArbitrageBotSettings()},
	} {
		t.Run(tc.req.BotType, func(t *testing.T) {
			skipLiveUnlessCanManipulate(t)
			if !mockTests {
				tc.req.BotID = createTradingTestBot(t, tc.settings)
				terminateTradingBotAfterTest(t, tc.req.BotID, tc.req.BotType)
			}
			require.NoErrorf(t, e.UpdateTradingBot(t.Context(), tc.req), "UpdateTradingBot must not error for the docs' %s bot", tc.req.BotType)
		})
	}

	autoStake := false
	// The cases run in order, one using autoStake, before the call of UpdateTradingBot below, which parallel cases
	// would follow
	for _, tc := range []struct {
		name string
		req  *UpdateTradingBotRequest
		exp  string
	}{
		{
			name: "a DCA bot with every setting",
			req: &UpdateTradingBotRequest{
				BotID:             "4611686018431234561",
				BotType:           TradingBotTypeDollarCostAveraging,
				NotifyOnBotChange: true,
				DisplayName:       "Updated DCA",
				Settings: &TradingBotDollarCostAveragingUpdateSettings{
					Allocations:         []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}},
					OrderType:           OrderTypeLimit,
					Price:               29000,
					MaximumOrders:       12,
					StopPrice:           20000,
					TakeProfitPrice:     40000,
					AutoStake:           &autoStake,
					InvestmentDuration:  43200000,
					InvestmentFrequency: 30 * time.Minute,
					Quantity:            0.02,
				},
			},
			exp: `{"bot_id":4611686018431234561,"bot_type":"DCA","notify_on_bot_change":true,"display_name":"Updated DCA","settings":{"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"type":"LIMIT","price":"29000","max_order":"12","stop_price":"20000","take_profit_price":"40000","auto_stake":false,"investment_duration":43200000,"investment_frequency":1800000,"quantity":"0.02"}}`,
		},
		{
			name: "a grid bot's prices",
			req:  &UpdateTradingBotRequest{BotID: "4611686018431234563", BotType: TradingBotTypeGrid, Settings: &TradingBotGridUpdateSettings{StopPrice: 24000, TakeProfitPrice: 36000, TrailingUpPriceCap: 50000}},
			exp:  `{"bot_id":4611686018431234563,"bot_type":"GRID","notify_on_bot_change":false,"settings":{"stop_price":"24000","take_profit_price":"36000","trailing_up_price_cap":"50000"}}`,
		},
		{
			name: "a TWAP bot without settings",
			req:  &UpdateTradingBotRequest{BotID: "4611686018431234562", BotType: TradingBotTypeTimeWeightedAveragePrice},
			exp:  `{"bot_id":4611686018431234562,"bot_type":"TWAP","notify_on_bot_change":false}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/bot/update-trading-bot", tc.exp, `{"id":1,"method":"private/bot/update-trading-bot","code":0}`)
			require.NoErrorf(t, ex.UpdateTradingBot(t.Context(), tc.req), "UpdateTradingBot must not error against the test server for a %s bot", tc.req.BotType)
		})
	}
	err := newUnavailableServerExchange(t).UpdateTradingBot(t.Context(), &UpdateTradingBotRequest{BotID: "1", BotType: TradingBotTypeTimeWeightedAveragePrice})
	assert.ErrorIs(t, err, request.ErrBadStatus, "UpdateTradingBot should return a request failure")
}

// requireTradingBotIdentityValidation checks a method taking a bot's ID and type refuses each request its validation
// rejects
func requireTradingBotIdentityValidation(t *testing.T, method string, call func(botID, botType string) error) {
	t.Helper()
	require.ErrorIsf(t, call("", TradingBotTypeDollarCostAveraging), errTradingBotIDRequired, "%s must error without a bot ID", method)
	require.ErrorIsf(t, call("bot-1", TradingBotTypeDollarCostAveraging), errInvalidTradingBotID, "%s must error for a bot ID that is not a number", method)
	require.ErrorIsf(t, call("1", ""), errInvalidTradingBotType, "%s must error without a bot type", method)
	require.ErrorIsf(t, call("1", "MARTINGALE"), errInvalidTradingBotType, "%s must error for an unknown bot type", method)
}

func TestTerminateTradingBot(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.TerminateTradingBot(t.Context(), nil), common.ErrNilPointer, "TerminateTradingBot must error for a nil request")
	requireTradingBotIdentityValidation(t, "TerminateTradingBot", func(botID, botType string) error {
		return e.TerminateTradingBot(t.Context(), &TerminateTradingBotRequest{BotID: botID, BotType: botType})
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		req := &TerminateTradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}
		if !mockTests {
			req.BotID = createTradingTestBot(t, dcaBotSettings())
			t.Cleanup(func() {
				if !t.Failed() {
					return
				}
				if err := e.TerminateTradingBot(context.WithoutCancel(t.Context()), req); err != nil {
					t.Logf("TerminateTradingBot could not terminate bot %s, which the test created: %v", req.BotID, err)
				}
			})
		}
		require.NoError(t, e.TerminateTradingBot(t.Context(), req), "TerminateTradingBot must not error")
	})

	// The cases run in order, before the call of TerminateTradingBot below, which parallel cases would follow
	for _, tc := range []struct {
		req *TerminateTradingBotRequest
		exp string
	}{
		{
			req: &TerminateTradingBotRequest{BotID: "4611686018431234564", BotType: TradingBotTypeFundingArbitrage, ClosePosition: true},
			exp: `{"bot_id":4611686018431234564,"bot_type":"FUNDING_ARBITRAGE","close_position":true}`,
		},
		{
			req: &TerminateTradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging},
			exp: `{"bot_id":4611686018431234561,"bot_type":"DCA"}`,
		},
	} {
		t.Run(tc.req.BotType, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/bot/terminate-trading-bot", tc.exp, `{"id":1,"method":"private/bot/terminate-trading-bot","code":0}`)
			require.NoErrorf(t, ex.TerminateTradingBot(t.Context(), tc.req), "TerminateTradingBot must not error against the test server for a %s bot", tc.req.BotType)
		})
	}
	err := newUnavailableServerExchange(t).TerminateTradingBot(t.Context(), &TerminateTradingBotRequest{BotID: "1", BotType: TradingBotTypeDollarCostAveraging})
	assert.ErrorIs(t, err, request.ErrBadStatus, "TerminateTradingBot should return a request failure")
}

func TestPauseTradingBot(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.PauseTradingBot(t.Context(), nil), common.ErrNilPointer, "PauseTradingBot must error for a nil request")
	requireTradingBotIdentityValidation(t, "PauseTradingBot", func(botID, botType string) error {
		return e.PauseTradingBot(t.Context(), &TradingBotRequest{BotID: botID, BotType: botType})
	})
	req := &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		live := *req
		if !mockTests {
			live.BotID = createTradingTestBot(t, dcaBotSettings())
			terminateTradingBotAfterTest(t, live.BotID, live.BotType)
		}
		require.NoError(t, e.PauseTradingBot(t.Context(), &live), "PauseTradingBot must not error")
	})

	ex := newSignedParamsServer(t, "private/bot/pause-trading-bot", `{"bot_id":4611686018431234561,"bot_type":"DCA"}`, `{"id":1,"method":"private/bot/pause-trading-bot","code":0}`)
	require.NoError(t, ex.PauseTradingBot(t.Context(), req), "PauseTradingBot must not error against the test server")
	assert.ErrorIs(t, newUnavailableServerExchange(t).PauseTradingBot(t.Context(), req), request.ErrBadStatus, "PauseTradingBot should return a request failure")
}

func TestResumeTradingBot(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ResumeTradingBot(t.Context(), nil), common.ErrNilPointer, "ResumeTradingBot must error for a nil request")
	requireTradingBotIdentityValidation(t, "ResumeTradingBot", func(botID, botType string) error {
		return e.ResumeTradingBot(t.Context(), &TradingBotRequest{BotID: botID, BotType: botType})
	})
	req := &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		live := *req
		if !mockTests {
			live.BotID = createTradingTestBot(t, dcaBotSettings())
			terminateTradingBotAfterTest(t, live.BotID, live.BotType)
			require.NoError(t, e.PauseTradingBot(t.Context(), &live), "PauseTradingBot must pause the bot the test resumes")
		}
		require.NoError(t, e.ResumeTradingBot(t.Context(), &live), "ResumeTradingBot must not error")
	})

	ex := newSignedParamsServer(t, "private/bot/resume-trading-bot", `{"bot_id":4611686018431234561,"bot_type":"DCA"}`, `{"id":1,"method":"private/bot/resume-trading-bot","code":0}`)
	require.NoError(t, ex.ResumeTradingBot(t.Context(), req), "ResumeTradingBot must not error against the test server")
	assert.ErrorIs(t, newUnavailableServerExchange(t).ResumeTradingBot(t.Context(), req), request.ErrBadStatus, "ResumeTradingBot should return a request failure")
}

func TestGetTradingBots(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingBots(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingBots must error for a nil request")
	// The cases run in order on e, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *TradingBotsRequest
		err  error
	}{
		{name: "for a bot ID that is not a number", req: &TradingBotsRequest{BotID: "bot-1", BotTypes: []string{TradingBotTypeDollarCostAveraging}}, err: errInvalidTradingBotID},
		{name: "without bot types", req: &TradingBotsRequest{States: []string{TradingBotStateRunning}}, err: errTradingBotTypesRequired},
		{name: "for an unknown bot type", req: &TradingBotsRequest{BotTypes: []string{"MARTINGALE"}, States: []string{TradingBotStateRunning}}, err: errInvalidTradingBotType},
		{name: "without a bot ID or states", req: &TradingBotsRequest{BotTypes: []string{TradingBotTypeDollarCostAveraging}}, err: errTradingBotStatesRequired},
		{name: "for an unknown state", req: &TradingBotsRequest{BotTypes: []string{TradingBotTypeDollarCostAveraging}, States: []string{"STOPPED"}}, err: errInvalidTradingBotState},
		{name: "for a page size above the maximum", req: &TradingBotsRequest{BotTypes: []string{TradingBotTypeDollarCostAveraging}, States: []string{TradingBotStateRunning}, PageSize: tradingBotMaximumPageSize + 1}, err: errInvalidCount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetTradingBots(t.Context(), tc.req)
			require.ErrorIsf(t, err, tc.err, "GetTradingBots must error %s", tc.name)
		})
	}
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		req := &TradingBotsRequest{
			BotTypes: []string{TradingBotTypeDollarCostAveraging, TradingBotTypeTimeWeightedAveragePrice, TradingBotTypeGrid, TradingBotTypeFundingArbitrage},
			States:   []string{TradingBotStateRunning, TradingBotStatePaused, TradingBotStateTerminated},
			PageSize: 20,
		}
		result, err := e.GetTradingBots(t.Context(), req)
		require.NoError(t, err, "GetTradingBots must not error")
		if mockTests {
			exp := []TradingBotResponse{
				{
					BotID:       "4611686018431234564",
					BotType:     TradingBotTypeFundingArbitrage,
					DisplayName: "Funding Arb Bot",
					State:       TradingBotStateRunning,
					CreateTime:  types.Time(time.UnixMilli(1691600000000)),
					UpdateTime:  types.Time(time.UnixMilli(1691600000000)),
					Settings:    fundingArbitrageBotSettings(),
				},
				{
					BotID:       "4611686018431234563",
					BotType:     TradingBotTypeGrid,
					DisplayName: "My Grid Bot",
					State:       TradingBotStatePaused,
					CreateTime:  types.Time(time.UnixMilli(1691520000000)),
					UpdateTime:  types.Time(time.UnixMilli(1691590000000)),
					Migrated:    true,
					Settings:    gridBotSettings(),
				},
				{
					BotID:       "4611686018431234562",
					BotType:     TradingBotTypeTimeWeightedAveragePrice,
					DisplayName: "My TWAP Bot",
					State:       TradingBotStateTerminated,
					Reason:      "User terminated",
					CreateTime:  types.Time(time.UnixMilli(1691510460000)),
					UpdateTime:  types.Time(time.UnixMilli(1691513460000)),
					EndTime:     types.Time(time.UnixMilli(1691513460000)),
					Settings:    twapBotSettings(),
				},
				{
					BotID:            "4611686018431234561",
					BotType:          TradingBotTypeDollarCostAveraging,
					DisplayName:      "My DCA Bot",
					State:            TradingBotStateRunning,
					CreateTime:       types.Time(time.UnixMilli(1691510460000)),
					UpdateTime:       types.Time(time.UnixMilli(1691510460000)),
					Settings:         dcaBotSettings(),
					ExecutionSummary: TradingBotExecutionSummary{ExecutedOrders: 5, TotalInvested: preciseNumber("500"), TotalQuantity: preciseNumber("0.05"), AveragePrice: preciseNumber("10000")},
				},
			}
			assert.Equal(t, exp, result, "GetTradingBots should decode every field, and each bot's settings by its type")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].BotID, "GetTradingBots should return bot %d's ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetTradingBots below, which parallel cases would follow
	for _, tc := range []struct {
		name string
		req  *TradingBotsRequest
		exp  string
	}{
		{
			name: "every param",
			req:  &TradingBotsRequest{BotID: "4611686018431234561", BotTypes: []string{TradingBotTypeDollarCostAveraging}, States: []string{TradingBotStateRunning}, Page: 2, PageSize: 50},
			exp:  `{"bot_id":4611686018431234561,"bot_types":["DCA"],"state":["RUNNING"],"page":2,"page_size":50}`,
		},
		{
			name: "bot types and states",
			req:  &TradingBotsRequest{BotTypes: []string{TradingBotTypeGrid}, States: []string{TradingBotStatePaused}},
			exp:  `{"bot_types":["GRID"],"state":["PAUSED"]}`,
		},
		{
			name: "a bot ID without states",
			req:  &TradingBotsRequest{BotID: "4611686018431234563", BotTypes: []string{TradingBotTypeGrid}},
			exp:  `{"bot_id":4611686018431234563,"bot_types":["GRID"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/bot/get-trading-bots", tc.exp, `{"id":1,"method":"private/bot/get-trading-bots","code":0,"result":{"bot_list":[]}}`)
			_, err := ex.GetTradingBots(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetTradingBots must not error against the test server for %+v", tc.req)
		})
	}
	// Serial too, so the case runs before the GetTradingBots calls below
	t.Run("bare executed order count", func(t *testing.T) {
		ex := newResponseServerExchange(t, testResponse("private/bot/get-trading-bots",
			`{"bot_list":[{"bot_id":"1","bot_type":"DCA","state":"RUNNING","execution_summary":{"executed_order":5}}]}`))
		got, err := ex.GetTradingBots(t.Context(), &TradingBotsRequest{BotTypes: []string{TradingBotTypeDollarCostAveraging}, States: []string{TradingBotStateRunning}})
		require.NoError(t, err, "GetTradingBots must accept a bare executed order count")
		exp := []TradingBotResponse{{BotID: "1", BotType: TradingBotTypeDollarCostAveraging, State: TradingBotStateRunning, ExecutionSummary: TradingBotExecutionSummary{ExecutedOrders: 5}}}
		assert.Equal(t, exp, got, "GetTradingBots should decode a bare executed order count")
	})
	minimal := &TradingBotsRequest{BotTypes: []string{TradingBotTypeDollarCostAveraging}, States: []string{TradingBotStateRunning}}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/bot/get-trading-bots","code":0,"result":null}`).GetTradingBots(t.Context(), minimal)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTradingBots should error for a null result")
	_, err = newUnavailableServerExchange(t).GetTradingBots(t.Context(), minimal)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradingBots should return a request failure")
}

func TestGetTradingBotExecutions(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingBotExecutions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingBotExecutions must error for a nil request")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{})
	require.ErrorIs(t, err, errTradingBotIDRequired, "GetTradingBotExecutions must error without a bot ID")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "bot-1"})
	require.ErrorIs(t, err, errInvalidTradingBotID, "GetTradingBotExecutions must error for a bot ID that is not a number")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "1", PageSize: tradingBotMaximumPageSize + 1})
	require.ErrorIs(t, err, errInvalidCount, "GetTradingBotExecutions must error for a page size above the maximum")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		result, err := e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "4611686018431234561", PageSize: 20})
		require.NoError(t, err, "GetTradingBotExecutions must not error")
		if mockTests {
			exp := []TradingBotExecutionResponse{
				{
					InstrumentName:     "BTC/USDT",
					Symbol:             "BTC_USDT",
					ExecutionTime:      types.Time(time.UnixMilli(1691510460123)),
					ScheduleTime:       types.Time(time.UnixMilli(1691510460000)),
					OrderStatus:        OrderStatusFilled,
					Side:               SideBuy,
					Price:              preciseNumber("10000.00"),
					Quantity:           preciseNumber("0.01"),
					AveragePrice:       preciseNumber("10000.00"),
					CumulativeQuantity: preciseNumber("0.01"),
					OrderID:            "6142909897021751347",
					UpdateTime:         types.Time(time.UnixMilli(1691510460456)),
					ExecutedPercentage: 100,
				},
			}
			assert.Equal(t, exp, result, "GetTradingBotExecutions should decode every field")
		} else {
			for i := range result {
				assert.NotEmptyf(t, result[i].OrderID, "GetTradingBotExecutions should return execution %d's order ID", i)
			}
		}
	})

	// The cases run in order, before the calls of GetTradingBotExecutions below, which parallel cases would follow
	for _, tc := range []struct {
		name string
		req  *TradingBotExecutionsRequest
		exp  string
	}{
		{
			name: "every param",
			req:  &TradingBotExecutionsRequest{BotID: "4611686018431234561", OrderStatuses: []string{OrderStatusFilled, OrderStatusCanceled}, BasketInstrumentNames: []string{"BTC_USDT", "ETH_USDT"}, Page: 1, PageSize: 100},
			exp:  `{"bot_id":4611686018431234561,"order_status":["FILLED","CANCELED"],"basket_bot_filter":{"include_child_symbols":["BTC_USDT","ETH_USDT"]},"page":1,"page_size":100}`,
		},
		{
			name: "only the bot ID",
			req:  &TradingBotExecutionsRequest{BotID: "4611686018431234561"},
			exp:  `{"bot_id":4611686018431234561}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/bot/get-trading-bot-executions", tc.exp, `{"id":1,"method":"private/bot/get-trading-bot-executions","code":0,"result":{"execution_list":[]}}`)
			_, err := ex.GetTradingBotExecutions(t.Context(), tc.req)
			require.NoErrorf(t, err, "GetTradingBotExecutions must not error against the test server for %+v", tc.req)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/bot/get-trading-bot-executions","code":0,"result":null}`).GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "1"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTradingBotExecutions should error for a null result")
	_, err = newUnavailableServerExchange(t).GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "1"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTradingBotExecutions should return a request failure")
}

func TestTradingBotDollarCostAveragingSettingsValidateSettings(t *testing.T) {
	t.Parallel()
	dca := func(modify func(*TradingBotDollarCostAveragingSettings)) *TradingBotDollarCostAveragingSettings {
		s := dcaBotSettings()
		modify(s)
		return s
	}
	basket := func(modify func(*TradingBotDollarCostAveragingSettings)) *TradingBotDollarCostAveragingSettings {
		s := dcaBasketBotSettings()
		modify(s)
		return s
	}
	for _, tc := range []struct {
		name     string
		settings *TradingBotDollarCostAveragingSettings
		err      error
	}{
		{name: "the docs' DCA bot", settings: dcaBotSettings()},
		{name: "every setting", settings: dcaBotEverySetting()},
		{name: "a basket without a side or an order type", settings: dcaBasketBotSettings()},
		{name: "a selling bot", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Side = SideSell })},
		{name: "a limit order with a price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType, s.Price = OrderTypeLimit, 30000 })},
		{name: "a total investment without a quantity", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Quantity, s.TotalInvestment = 0, 300 })},
		{name: "a fixed duration of a millisecond", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequency = time.Millisecond })},
		{name: "a monthly bot without a frequency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) {
			s.InvestmentFrequencyMode, s.InvestmentFrequency = TradingBotInvestmentFrequencyModeMonthly, 0
		})},
		{name: "no first investment time", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.FirstInvestmentTime = time.Time{} })},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "no allocations", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations = nil }), err: errTradingBotAllocationsRequired},
		{name: "a negative price on a market order", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Price = -1 }), err: errInvalidPrice},
		{name: "a negative stop price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.StopPrice = -1 }), err: errInvalidPrice},
		{name: "a negative take profit price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.TakeProfitPrice = -1 }), err: errInvalidPrice},
		{name: "an empty list of allocations", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations = []TradingBotAllocation{} }), err: errTradingBotAllocationsRequired},
		{name: "an allocation without an instrument", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations[0].InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "allocations totalling 150 percent", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Allocations[0].AllocatedPercentage = 150 }), err: errInvalidTradingBotAllocation},
		{name: "no investment currency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentCurrency = currency.EMPTYCODE }), err: currency.ErrCurrencyCodeEmpty},
		{name: "no side", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Side = "" }), err: errInvalidSide},
		{name: "no order type", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = "" }), err: errInvalidOrderType},
		{name: "a stop loss order type", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = OrderTypeStopLoss }), err: errInvalidOrderType},
		{name: "a limit order without a price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = OrderTypeLimit }), err: errInvalidPrice},
		{name: "a limit order with a negative price", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType, s.Price = OrderTypeLimit, -1 }), err: errInvalidPrice},
		{name: "a basket's limit order without a price", settings: basket(func(s *TradingBotDollarCostAveragingSettings) { s.OrderType = OrderTypeLimit }), err: errInvalidPrice},
		{name: "a negative quantity", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Quantity = -1 }), err: errInvalidQuantity},
		{name: "a negative total investment", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.TotalInvestment = -1 }), err: errInvalidAmount},
		{name: "neither a quantity nor a total investment", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.Quantity = 0 }), err: errTradingBotQuantityOrInvestmentRequired},
		{name: "no investment frequency mode", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequencyMode = "" }), err: errInvalidTradingBotInvestmentFrequencyMode},
		{name: "an unknown investment frequency mode", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequencyMode = "WEEKLY" }), err: errInvalidTradingBotInvestmentFrequencyMode},
		{name: "a fixed duration without a frequency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.InvestmentFrequency = 0 }), err: errInvalidTradingBotInvestmentFrequency},
		{name: "a fixed duration under a millisecond", settings: dca(func(s *TradingBotDollarCostAveragingSettings) {
			s.InvestmentFrequency = time.Millisecond - time.Nanosecond
		}), err: errInvalidTradingBotInvestmentFrequency},
		{name: "a monthly bot with a negative frequency", settings: dca(func(s *TradingBotDollarCostAveragingSettings) {
			s.InvestmentFrequencyMode, s.InvestmentFrequency = TradingBotInvestmentFrequencyModeMonthly, -time.Nanosecond
		}), err: errInvalidTradingBotInvestmentFrequency},
		{name: "a first investment before the Unix epoch", settings: dca(func(s *TradingBotDollarCostAveragingSettings) { s.FirstInvestmentTime = time.UnixMilli(-1) }), err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.settings.validateSettings(), tc.err, "validateSettings should return the expected error for %s", tc.name)
		})
	}
}

func TestTradingBotTimeWeightedAveragePriceSettingsValidateSettings(t *testing.T) {
	t.Parallel()
	twap := func(modify func(*TradingBotTimeWeightedAveragePriceSettings)) *TradingBotTimeWeightedAveragePriceSettings {
		s := twapBotSettings()
		modify(s)
		return s
	}
	now := time.Now()
	for _, tc := range []struct {
		name     string
		settings *TradingBotTimeWeightedAveragePriceSettings
		err      error
	}{
		{name: "the docs' TWAP bot", settings: twapBotSettings()},
		{name: "a limit order with a price", settings: twapLimitBotSettings()},
		{name: "a start in the future", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) {
			s.StartTime, s.EndTime = now.Add(time.Hour), now.Add(2*time.Hour)
		})},
		{name: "a start a millisecond before the end", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = s.EndTime.Add(-time.Millisecond) })},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "no instrument", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "no side", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.Side = "" }), err: errInvalidSide},
		{name: "no order type", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType = "" }), err: errInvalidOrderType},
		{name: "a stop loss order type", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType = OrderTypeStopLoss }), err: errInvalidOrderType},
		{name: "a limit order without a price", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType = OrderTypeLimit }), err: errInvalidPrice},
		{name: "a limit order with a negative price", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.OrderType, s.Price = OrderTypeLimit, -1 }), err: errInvalidPrice},
		{name: "no quantity", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.Quantity = 0 }), err: errInvalidQuantity},
		{name: "a negative quantity", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.Quantity = -1 }), err: errInvalidQuantity},
		{name: "no start time", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = time.Time{} }), err: common.ErrDateUnset},
		{name: "no end time", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.EndTime = time.Time{} }), err: common.ErrDateUnset},
		{name: "a start after the end", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = s.EndTime.Add(time.Millisecond) }), err: common.ErrStartAfterEnd},
		{name: "a start at the end", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = s.EndTime }), err: common.ErrStartEqualsEnd},
		{name: "a start before the Unix epoch", settings: twap(func(s *TradingBotTimeWeightedAveragePriceSettings) { s.StartTime = time.UnixMilli(-1) }), err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.settings.validateSettings(), tc.err, "validateSettings should return the expected error for %s", tc.name)
		})
	}
}

func TestTradingBotGridSettingsValidateSettings(t *testing.T) {
	t.Parallel()
	grid := func(modify func(*TradingBotGridSettings)) *TradingBotGridSettings {
		s := gridBotSettings()
		modify(s)
		return s
	}
	for _, tc := range []struct {
		name     string
		settings *TradingBotGridSettings
		err      error
	}{
		{name: "the docs' grid bot", settings: gridBotSettings()},
		{name: "every setting", settings: gridBotEverySetting()},
		{name: "the fewest grid levels", settings: grid(func(s *TradingBotGridSettings) { s.GridLevels = tradingBotGridMinimumLevels })},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "no instrument", settings: grid(func(s *TradingBotGridSettings) { s.InstrumentName = "" }), err: errInstrumentNameRequired},
		{name: "no total investment", settings: grid(func(s *TradingBotGridSettings) { s.TotalInvestment = 0 }), err: errInvalidAmount},
		{name: "a negative total investment", settings: grid(func(s *TradingBotGridSettings) { s.TotalInvestment = -1 }), err: errInvalidAmount},
		{name: "no lower price", settings: grid(func(s *TradingBotGridSettings) { s.LowerPrice = 0 }), err: errInvalidPrice},
		{name: "a negative lower price", settings: grid(func(s *TradingBotGridSettings) { s.LowerPrice = -1 }), err: errInvalidPrice},
		{name: "an upper price at the lower", settings: grid(func(s *TradingBotGridSettings) { s.UpperPrice = s.LowerPrice }), err: errInvalidTradingBotPriceRange},
		{name: "an upper price below the lower", settings: grid(func(s *TradingBotGridSettings) { s.UpperPrice = s.LowerPrice - 1 }), err: errInvalidTradingBotPriceRange},
		{name: "no grid mode", settings: grid(func(s *TradingBotGridSettings) { s.GridMode = "" }), err: errInvalidTradingBotGridMode},
		{name: "an unknown grid mode", settings: grid(func(s *TradingBotGridSettings) { s.GridMode = "LOGARITHMIC" }), err: errInvalidTradingBotGridMode},
		{name: "fewer grid levels than the fewest", settings: grid(func(s *TradingBotGridSettings) { s.GridLevels = tradingBotGridMinimumLevels - 1 }), err: errInvalidTradingBotGridLevels},
		{name: "no grid levels", settings: grid(func(s *TradingBotGridSettings) { s.GridLevels = 0 }), err: errInvalidTradingBotGridLevels},
		{name: "a negative stop price", settings: grid(func(s *TradingBotGridSettings) { s.StopPrice = -1 }), err: errInvalidPrice},
		{name: "a negative take profit price", settings: grid(func(s *TradingBotGridSettings) { s.TakeProfitPrice = -1 }), err: errInvalidPrice},
		{name: "a negative trailing up price cap", settings: grid(func(s *TradingBotGridSettings) { s.TrailingUpPriceCap = -1 }), err: errInvalidPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.settings.validateSettings(), tc.err, "validateSettings should return the expected error for %s", tc.name)
		})
	}
}

func TestTradingBotFundingArbitrageSettingsValidateSettings(t *testing.T) {
	t.Parallel()
	fundingArbitrage := func(modify func(*TradingBotFundingArbitrageSettings)) *TradingBotFundingArbitrageSettings {
		s := fundingArbitrageBotSettings()
		modify(s)
		return s
	}
	for _, tc := range []struct {
		name     string
		settings *TradingBotFundingArbitrageSettings
		err      error
		leg      string
	}{
		{name: "the docs' funding arbitrage bot", settings: fundingArbitrageBotSettings()},
		{name: "a short spot leg and a long perpetual leg", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.SpotLeg.Side, s.PerpetualLeg.Side = SideSell, SideBuy })},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "no total investment", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.TotalInvestment = 0 }), err: errInvalidAmount},
		{name: "a negative total investment", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.TotalInvestment = -1 }), err: errInvalidAmount},
		{name: "no investment currency", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.InvestmentCurrency = currency.EMPTYCODE }), err: currency.ErrCurrencyCodeEmpty},
		{name: "a spot leg without an instrument", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.SpotLeg.InstrumentName = "" }), err: errInstrumentNameRequired, leg: "spot leg"},
		{name: "a spot leg without a side", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.SpotLeg.Side = "" }), err: errInvalidSide, leg: "spot leg"},
		{name: "a perpetual leg without an instrument", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.PerpetualLeg.InstrumentName = "" }), err: errInstrumentNameRequired, leg: "perpetual leg"},
		{name: "a perpetual leg without a side", settings: fundingArbitrage(func(s *TradingBotFundingArbitrageSettings) { s.PerpetualLeg.Side = "" }), err: errInvalidSide, leg: "perpetual leg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.settings.validateSettings()
			require.ErrorIsf(t, err, tc.err, "validateSettings must return the expected error for %s", tc.name)
			if tc.leg != "" {
				assert.ErrorContainsf(t, err, tc.leg+": ", "validateSettings should name the %s", tc.leg)
			}
		})
	}
}

func TestTradingBotLegValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		leg  TradingBotLeg
		err  error
	}{
		{name: "a long spot leg", leg: TradingBotLeg{InstrumentName: "BTC_USDT", Side: SideBuy}},
		{name: "a short perpetual leg", leg: TradingBotLeg{InstrumentName: "BTCUSD-PERP", Side: SideSell}},
		{name: "no instrument", leg: TradingBotLeg{Side: SideBuy}, err: errInstrumentNameRequired},
		{name: "no instrument or side", err: errInstrumentNameRequired},
		{name: "no side", leg: TradingBotLeg{InstrumentName: "BTC_USDT"}, err: errInvalidSide},
		{name: "an unknown side", leg: TradingBotLeg{InstrumentName: "BTC_USDT", Side: "LONG"}, err: errInvalidSide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.leg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
		})
	}
}

func TestTradingBotDollarCostAveragingUpdateSettingsUpdateSettingsParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings *TradingBotDollarCostAveragingUpdateSettings
		exp      any
		expJSON  string
		err      error
	}{
		{name: "no settings", settings: &TradingBotDollarCostAveragingUpdateSettings{}, exp: &tradingBotDollarCostAveragingUpdateSettingsWire{}, expJSON: `{}`},
		{
			name:     "a frequency and maximum orders",
			settings: &TradingBotDollarCostAveragingUpdateSettings{InvestmentFrequency: 90 * time.Second, MaximumOrders: 3},
			exp:      &tradingBotDollarCostAveragingUpdateSettingsWire{MaximumOrders: 3, InvestmentFrequencyMilliseconds: 90000},
			expJSON:  `{"max_order":"3","investment_frequency":90000}`,
		},
		{
			name: "every setting",
			settings: &TradingBotDollarCostAveragingUpdateSettings{
				Allocations:         []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}},
				OrderType:           OrderTypeLimit,
				Price:               29000,
				MaximumOrders:       12,
				StopPrice:           20000,
				TakeProfitPrice:     40000,
				AutoStake:           new(false),
				InvestmentDuration:  43200000,
				InvestmentFrequency: 30 * time.Minute,
				Quantity:            0.02,
			},
			exp: &tradingBotDollarCostAveragingUpdateSettingsWire{
				Allocations:                     []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}},
				OrderType:                       OrderTypeLimit,
				Price:                           29000,
				MaximumOrders:                   12,
				StopPrice:                       20000,
				TakeProfitPrice:                 40000,
				AutoStake:                       new(false),
				InvestmentDuration:              43200000,
				InvestmentFrequencyMilliseconds: 1800000,
				Quantity:                        0.02,
			},
			expJSON: `{"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"type":"LIMIT","price":"29000","max_order":"12","stop_price":"20000","take_profit_price":"40000","auto_stake":false,"investment_duration":43200000,"investment_frequency":1800000,"quantity":"0.02"}`,
		},
		{name: "auto-staking turned on", settings: &TradingBotDollarCostAveragingUpdateSettings{AutoStake: new(true)}, exp: &tradingBotDollarCostAveragingUpdateSettingsWire{AutoStake: new(true)}, expJSON: `{"auto_stake":true}`},
		{name: "a market order type", settings: &TradingBotDollarCostAveragingUpdateSettings{OrderType: OrderTypeMarket}, exp: &tradingBotDollarCostAveragingUpdateSettingsWire{OrderType: OrderTypeMarket}, expJSON: `{"type":"MARKET"}`},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "an allocation without an instrument", settings: &TradingBotDollarCostAveragingUpdateSettings{Allocations: []TradingBotAllocation{{AllocatedPercentage: 100}}}, err: errInstrumentNameRequired},
		{name: "allocations totalling 90 percent", settings: &TradingBotDollarCostAveragingUpdateSettings{Allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 90}}}, err: errInvalidTradingBotAllocation},
		{name: "a stop loss order type", settings: &TradingBotDollarCostAveragingUpdateSettings{OrderType: OrderTypeStopLoss}, err: errInvalidOrderType},
		{name: "a negative price", settings: &TradingBotDollarCostAveragingUpdateSettings{Price: -1}, err: errInvalidPrice},
		{name: "a negative stop price", settings: &TradingBotDollarCostAveragingUpdateSettings{StopPrice: -1}, err: errInvalidPrice},
		{name: "a negative take profit price", settings: &TradingBotDollarCostAveragingUpdateSettings{TakeProfitPrice: -1}, err: errInvalidPrice},
		{name: "a negative quantity", settings: &TradingBotDollarCostAveragingUpdateSettings{Quantity: -1}, err: errInvalidQuantity},
		{name: "a negative frequency", settings: &TradingBotDollarCostAveragingUpdateSettings{InvestmentFrequency: -time.Hour}, err: errInvalidTradingBotInvestmentFrequency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.settings.updateSettingsParams()
			require.ErrorIsf(t, err, tc.err, "updateSettingsParams must return the expected error for %s", tc.name)
			require.Equalf(t, tc.exp, params, "updateSettingsParams must return the update of %s", tc.name)
			if tc.err != nil {
				return
			}
			encoded, err := json.Marshal(params)
			require.NoErrorf(t, err, "Marshal must not error for the update of %s", tc.name)
			assert.JSONEqf(t, tc.expJSON, string(encoded), "updateSettingsParams should give %s as Crypto.com takes it", tc.name)
		})
	}
}

func TestTradingBotGridUpdateSettingsUpdateSettingsParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings *TradingBotGridUpdateSettings
		exp      any
		expJSON  string
		err      error
	}{
		{name: "no settings", settings: &TradingBotGridUpdateSettings{}, exp: &TradingBotGridUpdateSettings{}, expJSON: `{}`},
		{name: "a take profit price", settings: &TradingBotGridUpdateSettings{TakeProfitPrice: 36000}, exp: &TradingBotGridUpdateSettings{TakeProfitPrice: 36000}, expJSON: `{"take_profit_price":"36000"}`},
		{
			name:     "every price",
			settings: &TradingBotGridUpdateSettings{StopPrice: 24000, TakeProfitPrice: 36000, TrailingUpPriceCap: 50000},
			exp:      &TradingBotGridUpdateSettings{StopPrice: 24000, TakeProfitPrice: 36000, TrailingUpPriceCap: 50000},
			expJSON:  `{"stop_price":"24000","take_profit_price":"36000","trailing_up_price_cap":"50000"}`,
		},
		{name: "nil settings", err: common.ErrNilPointer},
		{name: "a negative stop price", settings: &TradingBotGridUpdateSettings{StopPrice: -1}, err: errInvalidPrice},
		{name: "a negative take profit price", settings: &TradingBotGridUpdateSettings{TakeProfitPrice: -1}, err: errInvalidPrice},
		{name: "a negative trailing up price cap", settings: &TradingBotGridUpdateSettings{TrailingUpPriceCap: -1}, err: errInvalidPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.settings.updateSettingsParams()
			require.ErrorIsf(t, err, tc.err, "updateSettingsParams must return the expected error for %s", tc.name)
			require.Equalf(t, tc.exp, params, "updateSettingsParams must return the update of %s", tc.name)
			if tc.err != nil {
				return
			}
			encoded, err := json.Marshal(params)
			require.NoErrorf(t, err, "Marshal must not error for the update of %s", tc.name)
			assert.JSONEqf(t, tc.expJSON, string(encoded), "updateSettingsParams should give %s as Crypto.com takes it", tc.name)
		})
	}
}

func TestValidateTradingBotAllocations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		allocations []TradingBotAllocation
		err         error
		allocation  string
	}{
		{name: "no allocations"},
		{name: "an empty list of allocations", allocations: []TradingBotAllocation{}},
		{name: "one allocation of 100 percent", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 100}}},
		{name: "allocations totalling 100 percent", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}}},
		{
			name:        "fractions totalling 100 percent that a float sum misses",
			allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 0.1}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 64.1}, {InstrumentName: "CRO_USDT", AllocatedPercentage: 35.8}},
		},
		{
			name:        "fractions over 100 percent that a float sum totals 100",
			allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 33.333333333333336}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 33.333333333333336}, {InstrumentName: "CRO_USDT", AllocatedPercentage: 33.33333333333333}},
			err:         errInvalidTradingBotAllocation,
		},
		{name: "an allocation without an instrument", allocations: []TradingBotAllocation{{AllocatedPercentage: 100}}, err: errInstrumentNameRequired, allocation: "allocation 0"},
		{name: "a second allocation without an instrument", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {AllocatedPercentage: 40}}, err: errInstrumentNameRequired, allocation: "allocation 1"},
		{name: "an allocation without a percentage", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT"}}, err: errInvalidTradingBotAllocation, allocation: "allocation 0"},
		{
			name:        "a negative allocation in a total of 100 percent",
			allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 150}, {InstrumentName: "ETH_USDT", AllocatedPercentage: -50}},
			err:         errInvalidTradingBotAllocation,
			allocation:  "allocation 1",
		},
		{name: "allocations totalling 99.9 percent", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 50}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 49.9}}, err: errInvalidTradingBotAllocation},
		{name: "allocations totalling 150 percent", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 150}}, err: errInvalidTradingBotAllocation},
		{name: "a percentage that is not a number", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: types.Number(math.NaN())}}, err: errInvalidTradingBotAllocation, allocation: "allocation 0"},
		{name: "an infinite percentage", allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: types.Number(math.Inf(1))}}, err: errInvalidTradingBotAllocation, allocation: "allocation 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateTradingBotAllocations(tc.allocations)
			require.ErrorIsf(t, err, tc.err, "validateTradingBotAllocations must return the expected error for %s", tc.name)
			if tc.allocation != "" {
				assert.ErrorContainsf(t, err, tc.allocation+": ", "validateTradingBotAllocations should name %s", tc.allocation)
			}
		})
	}
}

func TestValidateTradingBotType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		botType string
		err     error
	}{
		{botType: TradingBotTypeDollarCostAveraging},
		{botType: TradingBotTypeTimeWeightedAveragePrice},
		{botType: TradingBotTypeGrid},
		{botType: TradingBotTypeFundingArbitrage},
		{botType: "", err: errInvalidTradingBotType},
		{botType: "dca", err: errInvalidTradingBotType},
		{botType: "MARTINGALE", err: errInvalidTradingBotType},
	} {
		t.Run(subtestName(tc.botType), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateTradingBotType(tc.botType), tc.err, "validateTradingBotType should return the expected error for %q", tc.botType)
		})
	}
}

func TestValidateTradingBotSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side string
		err  error
	}{
		{side: SideBuy},
		{side: SideSell},
		{side: "", err: errInvalidSide},
		{side: "buy", err: errInvalidSide},
		{side: "HOLD", err: errInvalidSide},
	} {
		t.Run(subtestName(tc.side), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateTradingBotSide(tc.side), tc.err, "validateTradingBotSide should return the expected error for %q", tc.side)
		})
	}
}

func TestValidateTradingBotOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType string
		err       error
	}{
		{orderType: OrderTypeMarket},
		{orderType: OrderTypeLimit},
		{orderType: "", err: errInvalidOrderType},
		{orderType: "limit", err: errInvalidOrderType},
		{orderType: OrderTypeStopLoss, err: errInvalidOrderType},
		{orderType: OrderTypeTakeProfitLimit, err: errInvalidOrderType},
	} {
		t.Run(subtestName(tc.orderType), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateTradingBotOrderType(tc.orderType), tc.err, "validateTradingBotOrderType should return the expected error for %q", tc.orderType)
		})
	}
}

func TestNewTradingBotParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *TradingBotRequest
		exp  *tradingBotParams
		err  error
	}{
		{
			name: "a DCA bot",
			req:  &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging},
			exp:  &tradingBotParams{BotID: 4611686018431234561, BotType: TradingBotTypeDollarCostAveraging},
		},
		{
			name: "a funding arbitrage bot",
			req:  &TradingBotRequest{BotID: "4611686018431234564", BotType: TradingBotTypeFundingArbitrage},
			exp:  &tradingBotParams{BotID: 4611686018431234564, BotType: TradingBotTypeFundingArbitrage},
		},
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "no bot ID", req: &TradingBotRequest{BotType: TradingBotTypeDollarCostAveraging}, err: errTradingBotIDRequired},
		{name: "a bot ID that is not a number", req: &TradingBotRequest{BotID: "bot-1", BotType: TradingBotTypeDollarCostAveraging}, err: errInvalidTradingBotID},
		{name: "no bot type", req: &TradingBotRequest{BotID: "4611686018431234561"}, err: errInvalidTradingBotType},
		{name: "an unknown bot type", req: &TradingBotRequest{BotID: "4611686018431234561", BotType: "MARTINGALE"}, err: errInvalidTradingBotType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newTradingBotParams(tc.req)
			require.ErrorIsf(t, err, tc.err, "newTradingBotParams must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newTradingBotParams should return the params of %s", tc.name)
		})
	}
}

func TestTradingBotIDParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		botID string
		exp   uint64
		err   error
		cause error
	}{
		{botID: "4611686018431234561", exp: 4611686018431234561},
		{botID: "1", exp: 1},
		{botID: "18446744073709551615", exp: math.MaxUint64},
		{botID: "", err: errTradingBotIDRequired},
		{botID: "0", err: errInvalidTradingBotID},
		{botID: "bot-1", err: errInvalidTradingBotID, cause: strconv.ErrSyntax},
		{botID: "-1", err: errInvalidTradingBotID, cause: strconv.ErrSyntax},
		{botID: "1.5", err: errInvalidTradingBotID, cause: strconv.ErrSyntax},
		{botID: " 1", err: errInvalidTradingBotID, cause: strconv.ErrSyntax},
		{botID: "18446744073709551616", err: errInvalidTradingBotID, cause: strconv.ErrRange},
	} {
		t.Run(subtestName(tc.botID), func(t *testing.T) {
			t.Parallel()
			got, err := tradingBotIDParam(tc.botID)
			require.ErrorIsf(t, err, tc.err, "tradingBotIDParam must return the expected error for %q", tc.botID)
			if tc.cause != nil {
				assert.ErrorIsf(t, err, tc.cause, "tradingBotIDParam should keep why %q does not parse", tc.botID)
			}
			assert.Equalf(t, tc.exp, got, "tradingBotIDParam should give %q as an integer", tc.botID)
		})
	}
}

func TestInvestmentFrequencyParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		frequency time.Duration
		exp       uint64
		err       error
	}{
		{},
		{frequency: time.Millisecond, exp: 1},
		{frequency: 1500 * time.Microsecond, exp: 1},
		{frequency: 5 * time.Minute, exp: 300000},
		{frequency: -time.Millisecond, err: errInvalidTradingBotInvestmentFrequency},
		{frequency: -time.Hour, err: errInvalidTradingBotInvestmentFrequency},
		{frequency: -time.Nanosecond, err: errInvalidTradingBotInvestmentFrequency},
		{frequency: -999 * time.Microsecond, err: errInvalidTradingBotInvestmentFrequency},
	} {
		t.Run(tc.frequency.String(), func(t *testing.T) {
			t.Parallel()
			got, err := investmentFrequencyParam(tc.frequency)
			require.ErrorIsf(t, err, tc.err, "investmentFrequencyParam must return the expected error for %s", tc.frequency)
			assert.Equalf(t, tc.exp, got, "investmentFrequencyParam should give %s in whole milliseconds", tc.frequency)
		})
	}
}

func TestTradingBotDollarCostAveragingSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeDollarCostAveraging, (*TradingBotDollarCostAveragingSettings)(nil).BotType(), "BotType should name a DCA bot without reading its settings")
	assert.Equal(t, TradingBotTypeDollarCostAveraging, dcaBasketBotSettings().BotType(), "BotType should name a basket a DCA bot")
}

func TestTradingBotDollarCostAveragingSettingsMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings *TradingBotDollarCostAveragingSettings
		exp      string
		err      error
	}{
		{name: "the docs' DCA bot", settings: dcaBotSettings(), exp: dcaBotSettingsJSON},
		{name: "every setting", settings: dcaBotEverySetting(), exp: dcaBotEverySettingJSON},
		{name: "a basket", settings: dcaBasketBotSettings(), exp: dcaBasketBotSettingsJSON},
		{name: "no settings", settings: &TradingBotDollarCostAveragingSettings{}, exp: `{"is_basket":false,"allocations":null,"investment_currency":"","auto_stake":false,"investment_frequency_mode":""}`},
		{name: "a first investment before the Unix epoch", settings: &TradingBotDollarCostAveragingSettings{FirstInvestmentTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
		{name: "a negative frequency", settings: &TradingBotDollarCostAveragingSettings{InvestmentFrequency: -time.Hour}, err: errInvalidTradingBotInvestmentFrequency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.settings.MarshalJSON()
			require.ErrorIsf(t, err, tc.err, "MarshalJSON must return the expected error for %s", tc.name)
			if tc.err != nil {
				assert.Nilf(t, got, "MarshalJSON should return no JSON for %s", tc.name)
				return
			}
			assert.JSONEqf(t, tc.exp, string(got), "MarshalJSON should encode %s as Crypto.com takes it", tc.name)
			var decoded TradingBotDollarCostAveragingSettings
			require.NoErrorf(t, decoded.UnmarshalJSON(got), "UnmarshalJSON must decode %s as MarshalJSON encodes it", tc.name)
			assert.Equalf(t, *tc.settings, decoded, "UnmarshalJSON should decode %s back to the settings MarshalJSON encoded", tc.name)
		})
	}
	t.Run("investment currency in upper case", func(t *testing.T) {
		t.Parallel()
		settings := dcaBotSettings()
		settings.InvestmentCurrency = currency.NewCode("usdt")
		got, err := settings.MarshalJSON()
		require.NoError(t, err, "MarshalJSON must not error")
		assert.JSONEq(t, dcaBotSettingsJSON, string(got), "MarshalJSON should send the investment currency in upper case")
		assert.Equal(t, "usdt", settings.InvestmentCurrency.String(), "MarshalJSON should leave the settings' currency as it was")
	})
}

func TestTradingBotDollarCostAveragingSettingsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		data      string
		exp       TradingBotDollarCostAveragingSettings
		malformed bool
	}{
		{name: "the docs' DCA bot", data: dcaBotSettingsJSON, exp: *dcaBotSettings()},
		{name: "every setting", data: dcaBotEverySettingJSON, exp: *dcaBotEverySetting()},
		{name: "a basket", data: dcaBasketBotSettingsJSON, exp: *dcaBasketBotSettings()},
		{
			name: "an unset frequency and first investment time",
			data: `{"investment_frequency_mode":"MONTHLY","investment_frequency":0,"first_investment_time":0}`,
			exp:  TradingBotDollarCostAveragingSettings{InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeMonthly},
		},
		{
			name: "a frequency and first investment time too large to hold",
			data: `{"investment_frequency":18446744073709551615,"first_investment_time":18446744073709551615}`,
			exp:  TradingBotDollarCostAveragingSettings{InvestmentFrequency: math.MaxInt64},
		},
		{name: "no settings", data: `{}`},
		{name: "a malformed first investment time", data: `{"first_investment_time":"soon"}`, malformed: true},
		{name: "a malformed allocated percentage", data: `{"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":true}]}`, malformed: true},
		{name: "truncated settings", data: `{"is_basket":`, malformed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := dcaBotEverySetting()
			err := got.UnmarshalJSON([]byte(tc.data))
			if tc.malformed {
				assert.Errorf(t, err, "UnmarshalJSON should error for %s", tc.name)
				return
			}
			require.NoErrorf(t, err, "UnmarshalJSON must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, *got, "UnmarshalJSON should decode %s, keeping nothing from the settings it decodes into", tc.name)
		})
	}
	t.Run("null leaves the settings", func(t *testing.T) {
		t.Parallel()
		got := dcaBotEverySetting()
		require.NoError(t, got.UnmarshalJSON([]byte("null")), "UnmarshalJSON must not error for null")
		assert.Equal(t, dcaBotEverySetting(), got, "UnmarshalJSON should leave the settings as they were for null")
	})
	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		var settings TradingBotDollarCostAveragingSettings
		require.NoError(t, settings.UnmarshalJSON([]byte(dcaBotEverySettingJSON)), "UnmarshalJSON must not error for every setting")
		encoded, err := settings.MarshalJSON()
		require.NoError(t, err, "MarshalJSON must not error for the settings UnmarshalJSON decoded")
		assert.JSONEq(t, dcaBotEverySettingJSON, string(encoded), "MarshalJSON should encode the settings UnmarshalJSON decoded as they arrived")
	})
}

func TestTradingBotTimeWeightedAveragePriceSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeTimeWeightedAveragePrice, (*TradingBotTimeWeightedAveragePriceSettings)(nil).BotType(), "BotType should name a TWAP bot without reading its settings")
	assert.Equal(t, TradingBotTypeTimeWeightedAveragePrice, twapLimitBotSettings().BotType(), "BotType should name a TWAP bot")
}

func TestTradingBotTimeWeightedAveragePriceSettingsMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings *TradingBotTimeWeightedAveragePriceSettings
		exp      string
		err      error
	}{
		{name: "the docs' TWAP bot", settings: twapBotSettings(), exp: twapBotSettingsJSON},
		{name: "a limit order", settings: twapLimitBotSettings(), exp: twapLimitBotSettingsJSON},
		{name: "no settings", settings: &TradingBotTimeWeightedAveragePriceSettings{}, exp: `{"instrument_name":"","side":"","type":"","start_time_millis":0,"end_time_millis":0}`},
		{name: "a start before the Unix epoch", settings: &TradingBotTimeWeightedAveragePriceSettings{StartTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
		{name: "an end before the Unix epoch", settings: &TradingBotTimeWeightedAveragePriceSettings{EndTime: time.UnixMilli(-1)}, err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.settings.MarshalJSON()
			require.ErrorIsf(t, err, tc.err, "MarshalJSON must return the expected error for %s", tc.name)
			if tc.err != nil {
				assert.Nilf(t, got, "MarshalJSON should return no JSON for %s", tc.name)
				return
			}
			assert.JSONEqf(t, tc.exp, string(got), "MarshalJSON should encode %s as Crypto.com takes it", tc.name)
			var decoded TradingBotTimeWeightedAveragePriceSettings
			require.NoErrorf(t, decoded.UnmarshalJSON(got), "UnmarshalJSON must decode %s as MarshalJSON encodes it", tc.name)
			assert.Equalf(t, *tc.settings, decoded, "UnmarshalJSON should decode %s back to the settings MarshalJSON encoded", tc.name)
		})
	}
}

func TestTradingBotTimeWeightedAveragePriceSettingsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		data      string
		exp       TradingBotTimeWeightedAveragePriceSettings
		malformed bool
	}{
		{name: "the docs' TWAP bot", data: twapBotSettingsJSON, exp: *twapBotSettings()},
		{name: "a limit order", data: twapLimitBotSettingsJSON, exp: *twapLimitBotSettings()},
		{
			name: "unset times",
			data: `{"instrument_name":"BTC_USDT","side":"BUY","type":"MARKET","quantity":"1","start_time_millis":0,"end_time_millis":0}`,
			exp:  TradingBotTimeWeightedAveragePriceSettings{InstrumentName: "BTC_USDT", Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 1},
		},
		{name: "times too late to hold", data: `{"start_time_millis":9223372036854775808,"end_time_millis":18446744073709551615}`},
		{name: "no settings", data: `{}`},
		{name: "a malformed start time", data: `{"start_time_millis":"soon"}`, malformed: true},
		{name: "a malformed quantity", data: `{"quantity":true}`, malformed: true},
		{name: "truncated settings", data: `{"instrument_name":`, malformed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := twapLimitBotSettings()
			err := got.UnmarshalJSON([]byte(tc.data))
			if tc.malformed {
				assert.Errorf(t, err, "UnmarshalJSON should error for %s", tc.name)
				return
			}
			require.NoErrorf(t, err, "UnmarshalJSON must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, *got, "UnmarshalJSON should decode %s, keeping nothing from the settings it decodes into", tc.name)
		})
	}
	t.Run("null leaves the settings", func(t *testing.T) {
		t.Parallel()
		got := twapLimitBotSettings()
		require.NoError(t, got.UnmarshalJSON([]byte("null")), "UnmarshalJSON must not error for null")
		assert.Equal(t, twapLimitBotSettings(), got, "UnmarshalJSON should leave the settings as they were for null")
	})
	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		var settings TradingBotTimeWeightedAveragePriceSettings
		require.NoError(t, settings.UnmarshalJSON([]byte(twapLimitBotSettingsJSON)), "UnmarshalJSON must not error for a limit order")
		encoded, err := settings.MarshalJSON()
		require.NoError(t, err, "MarshalJSON must not error for the settings UnmarshalJSON decoded")
		assert.JSONEq(t, twapLimitBotSettingsJSON, string(encoded), "MarshalJSON should encode the settings UnmarshalJSON decoded as they arrived")
	})
}

func TestTradingBotGridSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeGrid, (*TradingBotGridSettings)(nil).BotType(), "BotType should name a grid bot without reading its settings")
	assert.Equal(t, TradingBotTypeGrid, gridBotEverySetting().BotType(), "BotType should name a grid bot")
}

func TestTradingBotFundingArbitrageSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeFundingArbitrage, (*TradingBotFundingArbitrageSettings)(nil).BotType(), "BotType should name a funding arbitrage bot without reading its settings")
	assert.Equal(t, TradingBotTypeFundingArbitrage, fundingArbitrageBotSettings().BotType(), "BotType should name a funding arbitrage bot")
}

func TestTradingBotFundingArbitrageSettingsMarshalJSON(t *testing.T) {
	t.Parallel()
	settings := fundingArbitrageBotSettings()
	settings.InvestmentCurrency = currency.NewCode("usdt")
	got, err := settings.MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.JSONEq(t, `{"total_investment":"5000","investment_currency":"USDT","spot_leg":{"instrument_name":"BTC_USDT","side":"BUY"},"perp_leg":{"instrument_name":"BTCUSD-PERP","side":"SELL"}}`, string(got), "MarshalJSON should send the settings with the investment currency in upper case")
	assert.Equal(t, "usdt", settings.InvestmentCurrency.String(), "MarshalJSON should leave the settings' currency as it was")
}

func TestTradingBotDollarCostAveragingUpdateSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeDollarCostAveraging, (*TradingBotDollarCostAveragingUpdateSettings)(nil).BotType(), "BotType should name a DCA bot without reading its update")
	assert.Equal(t, TradingBotTypeDollarCostAveraging, (&TradingBotDollarCostAveragingUpdateSettings{StopPrice: 20000}).BotType(), "BotType should name a DCA bot")
}

func TestTradingBotGridUpdateSettingsBotType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, TradingBotTypeGrid, (*TradingBotGridUpdateSettings)(nil).BotType(), "BotType should name a grid bot without reading its update")
	assert.Equal(t, TradingBotTypeGrid, (&TradingBotGridUpdateSettings{StopPrice: 24000}).BotType(), "BotType should name a grid bot")
}

func TestTradingBotResponseUnmarshalJSON(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name     string
		botType  string
		settings string
		exp      TradingBotSettings
	}{
		{
			name:     "a DCA bot",
			botType:  TradingBotTypeDollarCostAveraging,
			settings: `{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"BUY","type":"MARKET","auto_stake":false,"quantity":"0.01","investment_frequency_mode":"FIXED_DURATION","investment_frequency":300000,"max_order":"10","first_investment_time":1691510460000}`,
			exp:      dcaBotSettings(),
		},
		{
			name:     "a TWAP bot",
			botType:  TradingBotTypeTimeWeightedAveragePrice,
			settings: `{"instrument_name":"BTC_USDT","side":"BUY","type":"MARKET","quantity":"1","start_time_millis":1691510460000,"end_time_millis":1691513460000}`,
			exp:      twapBotSettings(),
		},
		{
			name:     "a grid bot",
			botType:  TradingBotTypeGrid,
			settings: `{"instrument_name":"BTC_USDT","total_investment":"1000","lower_price":"25000","upper_price":"35000","grid_mode":"ARITHMETIC","grid_no":10,"margin_enabled":false,"trailing_up_enabled":false}`,
			exp:      gridBotSettings(),
		},
		{
			name:     "a funding arbitrage bot",
			botType:  TradingBotTypeFundingArbitrage,
			settings: `{"total_investment":"5000","investment_currency":"USDT","spot_leg":{"instrument_name":"BTC_USDT","side":"BUY"},"perp_leg":{"instrument_name":"BTCUSD-PERP","side":"SELL"}}`,
			exp:      fundingArbitrageBotSettings(),
		},
		{name: "an unknown bot type", botType: "MARTINGALE", settings: `{"martingale_multiplier":"2"}`},
		{name: "a grid bot with null settings", botType: TradingBotTypeGrid, settings: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var bot TradingBotResponse
			payload := `{"bot_id":"9","bot_type":"` + tc.botType + `","state":"RUNNING","settings":` + tc.settings + `}`
			require.NoErrorf(t, json.Unmarshal([]byte(payload), &bot), "Unmarshal must not error for a %s bot", tc.botType)
			exp := TradingBotResponse{BotID: "9", BotType: tc.botType, State: TradingBotStateRunning, Settings: tc.exp}
			assert.Equalf(t, exp, bot, "Unmarshal should decode a %s bot's settings by its type", tc.botType)
		})
	}

	var bot TradingBotResponse
	require.NoError(t, json.Unmarshal([]byte(`{"bot_id":"9","bot_type":"GRID","settings":{"grid_no":10}}`), &bot), "Unmarshal must not error for a grid bot")
	require.NoError(t, json.Unmarshal([]byte(`{"bot_id":"10","bot_type":"MARTINGALE","settings":{"grid_no":10}}`), &bot), "Unmarshal must not error for an unknown bot type")
	assert.Nil(t, bot.Settings, "Unmarshal should clear the settings a reused bot held when its new type is unknown")
	require.NoError(t, json.Unmarshal([]byte(`{"bot_id":"11","bot_type":"GRID"}`), &bot), "Unmarshal must not error without settings")
	assert.Nil(t, bot.Settings, "Unmarshal should leave a bot without settings without them")

	assert.Error(t, json.Unmarshal([]byte(`{"bot_type":"GRID","settings":{"grid_no":"ten"}}`), &bot), "Unmarshal should error for malformed settings of a known bot type")
	assert.Error(t, json.Unmarshal([]byte(`{"bot_type":"DCA","settings":{"first_investment_time":"soon"}}`), &bot), "Unmarshal should error for a malformed DCA bot's settings")
	assert.Error(t, json.Unmarshal([]byte(`{"bot_type":"TWAP","settings":{"start_time_millis":"soon"}}`), &bot), "Unmarshal should error for a malformed TWAP bot's settings")
	assert.Error(t, json.Unmarshal([]byte(`{"bot_id":9}`), &bot), "Unmarshal should error for a malformed bot")
}

func TestTradingBotResponseDecodeSettings(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		botType  string
		settings string
		exp      TradingBotSettings
		err      string
	}{
		{name: "a DCA bot's settings", botType: TradingBotTypeDollarCostAveraging, settings: dcaBotSettingsJSON, exp: dcaBotSettings()},
		{name: "a TWAP bot's settings", botType: TradingBotTypeTimeWeightedAveragePrice, settings: twapBotSettingsJSON, exp: twapBotSettings()},
		{
			name:     "a grid bot's settings",
			botType:  TradingBotTypeGrid,
			settings: `{"instrument_name":"BTC_USDT","total_investment":"1000","lower_price":"25000","upper_price":"35000","grid_mode":"ARITHMETIC","grid_no":10,"margin_enabled":false,"trailing_up_enabled":false}`,
			exp:      gridBotSettings(),
		},
		{
			name:     "a funding arbitrage bot's settings",
			botType:  TradingBotTypeFundingArbitrage,
			settings: `{"total_investment":"5000","investment_currency":"USDT","spot_leg":{"instrument_name":"BTC_USDT","side":"BUY"},"perp_leg":{"instrument_name":"BTCUSD-PERP","side":"SELL"}}`,
			exp:      fundingArbitrageBotSettings(),
		},
		{name: "a grid bot's empty settings", botType: TradingBotTypeGrid, settings: `{}`, exp: &TradingBotGridSettings{}},
		{name: "an unknown bot type's settings", botType: "MARTINGALE", settings: `{"martingale_multiplier":"2"}`},
		{name: "settings of a bot without a type", settings: `{"grid_no":10}`},
		{name: "no settings", botType: TradingBotTypeGrid},
		{name: "null settings", botType: TradingBotTypeGrid, settings: `null`},
		{name: "a DCA bot's malformed settings", botType: TradingBotTypeDollarCostAveraging, settings: `{"first_investment_time":"soon"}`, err: "DCA bot settings: "},
		{name: "a TWAP bot's malformed settings", botType: TradingBotTypeTimeWeightedAveragePrice, settings: `{"start_time_millis":"soon"}`, err: "TWAP bot settings: "},
		{name: "a grid bot's malformed settings", botType: TradingBotTypeGrid, settings: `{"grid_no":"ten"}`, err: "GRID bot settings: "},
		{name: "a funding arbitrage bot's malformed settings", botType: TradingBotTypeFundingArbitrage, settings: `{"spot_leg":[]}`, err: "FUNDING_ARBITRAGE bot settings: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bot := TradingBotResponse{BotID: "9", BotType: tc.botType, Settings: &TradingBotGridSettings{InstrumentName: "ETH_USDT"}}
			err := bot.decodeSettings(json.RawMessage(tc.settings))
			if tc.err != "" {
				assert.ErrorContainsf(t, err, tc.err, "decodeSettings should refuse %s, naming the bot type", tc.name)
				return
			}
			require.NoErrorf(t, err, "decodeSettings must not error for %s", tc.name)
			assert.Equalf(t, TradingBotResponse{BotID: "9", BotType: tc.botType, Settings: tc.exp}, bot, "decodeSettings should replace the bot's settings with %s, decoded by its type", tc.name)
		})
	}
}

func TestMillisecondsDuration(t *testing.T) {
	t.Parallel()
	assert.Zero(t, millisecondsDuration(0), "millisecondsDuration should give no duration for zero")
	assert.Equal(t, 5*time.Minute, millisecondsDuration(300000), "millisecondsDuration should give the duration of the milliseconds")
	assert.Equal(t, time.Duration(math.MaxInt64), millisecondsDuration(math.MaxUint64), "millisecondsDuration should cap a duration too long to hold")
}

func TestUnixMilliTime(t *testing.T) {
	t.Parallel()
	assert.True(t, unixMilliTime(0).IsZero(), "unixMilliTime should give the zero time for zero")
	assert.Equal(t, time.UnixMilli(1691510460000), unixMilliTime(1691510460000), "unixMilliTime should give the time of the milliseconds")
	assert.True(t, unixMilliTime(math.MaxUint64).IsZero(), "unixMilliTime should give the zero time for milliseconds too large to hold")
}
