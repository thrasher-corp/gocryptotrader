package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGateAPIResponseAsError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&gateAPIResponse[*OTCQuoteResponse]{Message: "success"}).AsError(), "a zero code should not error")
	err := (&gateAPIResponse[*OTCQuoteResponse]{Code: 1001, Message: "quote expired"}).AsError()
	require.Error(t, err, "a non-zero code must error")
	assert.Equal(t, "error code: 1001 message: quote expired", err.Error(), "error should carry the code and message")
}

func TestNumericStringUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		payload string
		exp     NumericString
	}{
		{payload: `"110xxxxx"`, exp: "110xxxxx"},
		{payload: `"94294117233225616"`, exp: "94294117233225616"},
		{payload: `94294117233225616`, exp: "94294117233225616"},
		{payload: `-12`, exp: "-12"},
		{payload: `null`, exp: ""},
	} {
		var n NumericString
		require.NoErrorf(t, json.Unmarshal([]byte(tc.payload), &n), "Unmarshal must not error for %s", tc.payload)
		assert.Equalf(t, tc.exp, n, "NumericString should keep the text of %s", tc.payload)
		assert.Equalf(t, string(tc.exp), n.String(), "NumericString.String should return the text of %s", tc.payload)
	}
	for _, payload := range []string{`1.5`, `true`} {
		var n NumericString
		assert.ErrorIsf(t, json.Unmarshal([]byte(payload), &n), errInvalidNumericString, "NumericString should reject %s", payload)
	}
}

func TestFuturesPriceTriggeredOrderParamValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*FuturesPriceTriggeredOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should reject a nil order")

	valid := FuturesPriceTriggeredOrderRequest{
		Initial: FuturesInitial{Price: 1234, Size: 2, Contract: currency.NewBTCUSDT()},
		Trigger: FuturesTrigger{Rule: 1, Price: 3000},
	}
	for _, tc := range []struct {
		name   string
		mutate func(*FuturesPriceTriggeredOrderRequest)
		err    error
	}{
		{"zero price for a market order", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = 0 }, nil},
		{"gtc time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = gtcTIF }, nil},
		{"ioc time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = iocTIF }, nil},
		{"price gap strategy", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 1 }, nil},
		{"mark price trigger", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 1 }, nil},
		{"index price trigger", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 2 }, nil},
		{"at or below rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 2 }, nil},
		{"read only close long order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-order" }, errInvalidAutoOrderType},
		{"read only close short order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-short-order" }, errInvalidAutoOrderType},
		{"close long position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-long-position" }, nil},
		{"close short position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-short-position" }, nil},
		{"plan close long position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "plan-close-long-position" }, nil},
		{"plan close short position type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "plan-close-short-position" }, nil},
		{"empty contract", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Contract = currency.EMPTYPAIR }, currency.ErrCurrencyPairEmpty},
		{"negative price", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.Price = -1 }, limits.ErrPriceBelowMin},
		{"fok time in force", func(p *FuturesPriceTriggeredOrderRequest) { p.Initial.TimeInForce = fokTIF }, order.ErrInvalidTimeInForce},
		{"unknown strategy type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.StrategyType = 2 }, errInvalidTriggerStrategyType},
		{"unknown price type", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.PriceType = 3 }, errInvalidTriggerPriceType},
		{"missing trigger price", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Price = 0 }, limits.ErrPriceBelowMin},
		{"unknown rule", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.Rule = 3 }, errInvalidTriggerRule},
		{"whole second trigger expiry", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.ExpireAfter = types.DurationSeconds(time.Minute) }, nil},
		{"fractional trigger expiry", func(p *FuturesPriceTriggeredOrderRequest) { p.Trigger.ExpireAfter = fractionalSeconds }, errNotWholeSeconds},
		{"unknown order type", func(p *FuturesPriceTriggeredOrderRequest) { p.OrderType = "close-all" }, errInvalidAutoOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := valid
			tc.mutate(&p)
			assert.ErrorIs(t, p.validate(), tc.err, "validate should accept only documented values")
		})
	}
}

func TestFuturesInitialMarshalJSON(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(&FuturesInitial{Contract: currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter), Amount: 0.5, TimeInForce: iocTIF})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"contract":"BTC_USDT","size":0,"amount":"0.5","price":"0","tif":"ioc"}`, string(payload), "FuturesInitial should send a zero size and price for a market order")
}

func TestDisplayNumber(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		exp   DisplayNumber
	}{
		{input: `"$0.04"`, exp: DisplayNumber{Text: "$0.04", Value: 0.04}},
		{input: `"$0"`, exp: DisplayNumber{Text: "$0"}},
		{input: `"50%"`, exp: DisplayNumber{Text: "50%", Value: 50}},
		{input: `"0.1%"`, exp: DisplayNumber{Text: "0.1%", Value: 0.1}},
		{input: `"<$0.01"`, exp: DisplayNumber{Text: "<$0.01", Value: 0.01}},
		{input: `"$1,234.56"`, exp: DisplayNumber{Text: "$1,234.56", Value: 1234.56}},
		{input: `"-$0.04"`, exp: DisplayNumber{Text: "-$0.04", Value: -0.04}},
		{input: `"$-1.5"`, exp: DisplayNumber{Text: "$-1.5", Value: -1.5}},
		{input: `"0.04 USDT"`, exp: DisplayNumber{Text: "0.04 USDT", Value: 0.04}},
		{input: `".5%"`, exp: DisplayNumber{Text: ".5%", Value: 0.5}},
		{input: `"--"`, exp: DisplayNumber{Text: "--"}},
		{input: `"1.2.3%"`, exp: DisplayNumber{Text: "1.2.3%"}},
		{input: `"\u00240.04"`, exp: DisplayNumber{Text: "$0.04", Value: 0.04}},
		{input: `12.5`, exp: DisplayNumber{Text: "12.5", Value: 12.5}},
		{input: `""`, exp: DisplayNumber{}},
		{input: `null`, exp: DisplayNumber{}},
	} {
		got := DisplayNumber{Text: "stale", Value: 1}
		require.NoErrorf(t, got.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error for %s", tc.input)
		assert.Equalf(t, tc.exp, got, "UnmarshalJSON should decode %s", tc.input)
		assert.Equalf(t, tc.exp.Value, got.Float64(), "Float64 should return the number without its signs for %s", tc.input)
		payload, err := json.Marshal(got)
		require.NoErrorf(t, err, "Marshal must not error for %s", tc.input)
		var roundTrip string
		require.NoErrorf(t, json.Unmarshal(payload, &roundTrip), "Unmarshal must read back the marshalled %s", tc.input)
		assert.Equalf(t, tc.exp.Text, roundTrip, "Marshal should write %s as Gate sent it", tc.input)
	}
	for _, input := range []string{`true`, `{}`, `[1]`, `"unterminated`, `1.2.3`} {
		var got DisplayNumber
		assert.ErrorIsf(t, got.UnmarshalJSON([]byte(input)), errInvalidDisplayNumber, "UnmarshalJSON should error for %s", input)
	}
	assert.Equal(t, "0.04", DisplayNumber{Value: 0.04}.String(), "String should write the bare number when there is no text")
}

func TestDate(t *testing.T) {
	t.Parallel()
	for input, exp := range map[string]time.Time{
		`"2025/11/19"`: time.Date(2025, 11, 19, 0, 0, 0, 0, time.UTC),
		`"2026-10-21"`: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		`""`:           {},
		`null`:         {},
	} {
		got := Date(time.Now())
		require.NoErrorf(t, got.UnmarshalJSON([]byte(input)), "UnmarshalJSON must not error for %s", input)
		assert.Truef(t, exp.Equal(got.Time()), "UnmarshalJSON should decode %s as %s, not %s", input, exp, got.Time())
		assert.Equalf(t, time.UTC, got.Time().Location(), "UnmarshalJSON should hold %s in UTC", input)
	}
	for _, input := range []string{`"19/11/2025"`, `"2025-11-19 10:00:00"`, `"today"`, `20251119`, `true`} {
		var got Date
		assert.ErrorIsf(t, got.UnmarshalJSON([]byte(input)), errInvalidDate, "UnmarshalJSON should error for %s", input)
	}
	payload, err := json.Marshal(Date(time.Date(2025, 11, 19, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, err, "Marshal must not error for a date")
	assert.Equal(t, `"2025-11-19"`, string(payload), "Marshal should write the date as 2006-01-02")
	payload, err = json.Marshal(Date{})
	require.NoError(t, err, "Marshal must not error for an unset date")
	assert.Equal(t, `""`, string(payload), "Marshal should write an unset date as an empty string")
}
