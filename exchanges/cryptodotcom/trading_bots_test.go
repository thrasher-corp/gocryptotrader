package cryptodotcom

import (
	"math"
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
	skipLiveUnlessCanManipulate(t)

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
			result, err := e.CreateTradingBot(t.Context(), &CreateTradingBotRequest{BotType: tc.settings.BotType(), NotifyOnBotChange: true, Settings: tc.settings})
			require.NoErrorf(t, err, "CreateTradingBot must not error for the docs' %s bot", tc.settings.BotType())
			if mockTests {
				assert.Equalf(t, &CreateTradingBotResponse{BotID: tc.botID}, result, "CreateTradingBot should return the %s bot's ID", tc.settings.BotType())
			} else {
				assert.NotEmptyf(t, result.BotID, "CreateTradingBot should return the %s bot's ID", tc.settings.BotType())
			}
		})
	}

	autoStake := &TradingBotDollarCostAveragingSettings{
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
	basket := &TradingBotDollarCostAveragingSettings{
		IsBasket:                true,
		Allocations:             []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}},
		InvestmentCurrency:      currency.USDT,
		TotalInvestment:         1000,
		InvestmentFrequencyMode: TradingBotInvestmentFrequencyModeMonthly,
		InvestmentDuration:      12,
	}
	limitTWAP := &TradingBotTimeWeightedAveragePriceSettings{InstrumentName: "BTC_USDT", Side: SideSell, OrderType: OrderTypeLimit, Price: 30000, Quantity: 0.5, StartTime: time.UnixMilli(1792890000000), EndTime: time.UnixMilli(1792893600000)}
	everyGrid := &TradingBotGridSettings{
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
	// The cases run in order, using the settings above, before the calls of CreateTradingBot below, which parallel
	// cases would follow
	for _, tc := range []struct {
		name string
		req  *CreateTradingBotRequest
		exp  string
	}{
		{
			name: "a selling DCA bot with every setting",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging, NotifyOnBotChange: true, DisplayName: "Selling DCA", Settings: autoStake},
			exp:  `{"bot_type":"DCA","notify_on_bot_change":true,"display_name":"Selling DCA","settings":{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"SELL","type":"LIMIT","auto_stake":true,"price":"30000","quantity":"0.01","total_investment":"300","investment_frequency_mode":"FIXED_DURATION","investment_frequency":3600000,"first_investment_time":1792890000000,"investment_duration":86400000,"max_order":"24","stop_price":"20000","take_profit_price":"40000","simplified_mode":true}}`,
		},
		{
			name: "a basket DCA bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging, Settings: basket},
			exp:  `{"bot_type":"DCA","notify_on_bot_change":false,"settings":{"is_basket":true,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"60"},{"instrument_name":"ETH_USDT","allocated_percentage":"40"}],"investment_currency":"USDT","auto_stake":false,"total_investment":"1000","investment_frequency_mode":"MONTHLY","investment_duration":12}}`,
		},
		{
			name: "a limit TWAP bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeTimeWeightedAveragePrice, Settings: limitTWAP},
			exp:  `{"bot_type":"TWAP","notify_on_bot_change":false,"settings":{"instrument_name":"BTC_USDT","side":"SELL","type":"LIMIT","price":"30000","quantity":"0.5","start_time_millis":1792890000000,"end_time_millis":1792893600000}}`,
		},
		{
			name: "the docs' grid bot",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeGrid, Settings: gridBotSettings()},
			exp:  `{"bot_type":"GRID","notify_on_bot_change":false,"settings":{"instrument_name":"BTC_USDT","total_investment":"1000","lower_price":"25000","upper_price":"35000","grid_mode":"ARITHMETIC","grid_no":10,"margin_enabled":false,"trailing_up_enabled":false}}`,
		},
		{
			name: "a trailing grid bot with every setting",
			req:  &CreateTradingBotRequest{BotType: TradingBotTypeGrid, NotifyOnBotChange: true, DisplayName: "Trailing Grid", Settings: everyGrid},
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
	skipLiveUnlessCanManipulate(t)

	// The cases run in order on e, which every test shares, before the calls of UpdateTradingBot below
	for _, req := range []*UpdateTradingBotRequest{
		{
			BotID:             "4611686018431234561",
			BotType:           TradingBotTypeDollarCostAveraging,
			NotifyOnBotChange: true,
			Settings: &TradingBotDollarCostAveragingUpdateSettings{
				Allocations: []TradingBotAllocation{{InstrumentName: "BTC_USDT", AllocatedPercentage: 60}, {InstrumentName: "ETH_USDT", AllocatedPercentage: 40}},
				StopPrice:   20000,
			},
		},
		{BotID: "4611686018431234562", BotType: TradingBotTypeTimeWeightedAveragePrice, DisplayName: "My TWAP Bot"},
		{BotID: "4611686018431234563", BotType: TradingBotTypeGrid, NotifyOnBotChange: true, Settings: &TradingBotGridUpdateSettings{StopPrice: 24000}},
		{BotID: "4611686018431234564", BotType: TradingBotTypeFundingArbitrage, NotifyOnBotChange: true, DisplayName: "Funding Arb Bot"},
	} {
		t.Run(req.BotType, func(t *testing.T) {
			require.NoErrorf(t, e.UpdateTradingBot(t.Context(), req), "UpdateTradingBot must not error for the docs' %s bot", req.BotType)
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

func TestTerminateTradingBot(t *testing.T) { //nolint:tparallel // the cases run in order before the later call of TerminateTradingBot
	t.Parallel()
	require.ErrorIs(t, e.TerminateTradingBot(t.Context(), nil), common.ErrNilPointer, "TerminateTradingBot must error for a nil request")
	requireTradingBotIdentityValidation(t, "TerminateTradingBot", func(botID, botType string) error {
		return e.TerminateTradingBot(t.Context(), &TerminateTradingBotRequest{BotID: botID, BotType: botType})
	})
	skipLiveUnlessCanManipulate(t)

	require.NoError(t, e.TerminateTradingBot(t.Context(), &TerminateTradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}), "TerminateTradingBot must not error")

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
	skipLiveUnlessCanManipulate(t)

	req := &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}
	require.NoError(t, e.PauseTradingBot(t.Context(), req), "PauseTradingBot must not error")

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
	skipLiveUnlessCanManipulate(t)

	req := &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging}
	require.NoError(t, e.ResumeTradingBot(t.Context(), req), "ResumeTradingBot must not error")

	ex := newSignedParamsServer(t, "private/bot/resume-trading-bot", `{"bot_id":4611686018431234561,"bot_type":"DCA"}`, `{"id":1,"method":"private/bot/resume-trading-bot","code":0}`)
	require.NoError(t, ex.ResumeTradingBot(t.Context(), req), "ResumeTradingBot must not error against the test server")
	assert.ErrorIs(t, newUnavailableServerExchange(t).ResumeTradingBot(t.Context(), req), request.ErrBadStatus, "ResumeTradingBot should return a request failure")
}

func TestGetTradingBots(t *testing.T) { //nolint:tparallel // the cases run in order on e and before the later calls of GetTradingBots
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

func TestGetTradingBotExecutions(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of GetTradingBotExecutions
	t.Parallel()
	_, err := e.GetTradingBotExecutions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingBotExecutions must error for a nil request")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{})
	require.ErrorIs(t, err, errTradingBotIDRequired, "GetTradingBotExecutions must error without a bot ID")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "bot-1"})
	require.ErrorIs(t, err, errInvalidTradingBotID, "GetTradingBotExecutions must error for a bot ID that is not a number")
	_, err = e.GetTradingBotExecutions(t.Context(), &TradingBotExecutionsRequest{BotID: "1", PageSize: tradingBotMaximumPageSize + 1})
	require.ErrorIs(t, err, errInvalidCount, "GetTradingBotExecutions must error for a page size above the maximum")
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

func TestTradingBotSettingsMarshalJSON(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name     string
		settings any
		exp      string
	}{
		{
			name:     "DCA settings",
			settings: dcaBotSettings(),
			exp:      `{"is_basket":false,"allocations":[{"instrument_name":"BTC_USDT","allocated_percentage":"100"}],"investment_currency":"USDT","side":"BUY","type":"MARKET","auto_stake":false,"quantity":"0.01","investment_frequency_mode":"FIXED_DURATION","investment_frequency":300000,"first_investment_time":1691510460000,"max_order":"10"}`,
		},
		{
			name:     "TWAP settings",
			settings: twapBotSettings(),
			exp:      `{"instrument_name":"BTC_USDT","side":"BUY","type":"MARKET","quantity":"1","start_time_millis":1691510460000,"end_time_millis":1691513460000}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.settings)
			require.NoErrorf(t, err, "Marshal must not error for %T", tc.settings)
			assert.JSONEqf(t, tc.exp, string(encoded), "Marshal should encode %T as Crypto.com takes it", tc.settings)
		})
	}
	for name, settings := range map[string]any{
		"DCA settings first investing before 1970": &TradingBotDollarCostAveragingSettings{FirstInvestmentTime: time.UnixMilli(-1)},
		"DCA settings with a negative frequency":   &TradingBotDollarCostAveragingSettings{InvestmentFrequency: -time.Hour},
		"TWAP settings starting before 1970":       &TradingBotTimeWeightedAveragePriceSettings{StartTime: time.UnixMilli(-1)},
		"TWAP settings ending before 1970":         &TradingBotTimeWeightedAveragePriceSettings{EndTime: time.UnixMilli(-1)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := json.Marshal(settings)
			assert.Errorf(t, err, "Marshal should error for %+v, which Crypto.com cannot take", settings)
		})
	}
}

func TestTradingBotUpdateSettingsParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		settings TradingBotUpdateSettings
		exp      string
	}{
		{name: "empty DCA settings", settings: &TradingBotDollarCostAveragingUpdateSettings{}, exp: `{}`},
		{name: "a DCA frequency and maximum orders", settings: &TradingBotDollarCostAveragingUpdateSettings{InvestmentFrequency: 90 * time.Second, MaximumOrders: 3}, exp: `{"max_order":"3","investment_frequency":90000}`},
		{name: "empty grid settings", settings: &TradingBotGridUpdateSettings{}, exp: `{}`},
		{name: "a grid take profit price", settings: &TradingBotGridUpdateSettings{TakeProfitPrice: 36000}, exp: `{"take_profit_price":"36000"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.settings.updateSettingsParams()
			require.NoErrorf(t, err, "updateSettingsParams must not error for %+v", tc.settings)
			encoded, err := json.Marshal(params)
			require.NoErrorf(t, err, "Marshal must not error for the params of %+v", tc.settings)
			assert.JSONEqf(t, tc.exp, string(encoded), "updateSettingsParams should give %+v as Crypto.com takes it", tc.settings)
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

func TestTradingBotSettingsBotType(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for exp, settings := range map[string]interface{ BotType() string }{
		TradingBotTypeDollarCostAveraging:      (*TradingBotDollarCostAveragingSettings)(nil),
		TradingBotTypeTimeWeightedAveragePrice: (*TradingBotTimeWeightedAveragePriceSettings)(nil),
		TradingBotTypeGrid:                     (*TradingBotGridSettings)(nil),
		TradingBotTypeFundingArbitrage:         (*TradingBotFundingArbitrageSettings)(nil),
	} {
		t.Run(exp, func(t *testing.T) {
			assert.Equalf(t, exp, settings.BotType(), "%T should be a %s bot's settings", settings, exp)
		})
	}
	assert.Equal(t, TradingBotTypeDollarCostAveraging, (*TradingBotDollarCostAveragingUpdateSettings)(nil).BotType(), "TradingBotDollarCostAveragingUpdateSettings should be a DCA bot's settings")
	assert.Equal(t, TradingBotTypeGrid, (*TradingBotGridUpdateSettings)(nil).BotType(), "TradingBotGridUpdateSettings should be a grid bot's settings")
}
