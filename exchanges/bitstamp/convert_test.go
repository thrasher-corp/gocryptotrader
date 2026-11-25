package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
)

func TestOrderSideUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data string
		exp  order.Side
		err  error
	}{
		// Bare numbers arrive on the websocket order feed, quoted ones on the REST ticker
		{data: `0`, exp: order.Buy},
		{data: `1`, exp: order.Sell},
		{data: `"0"`, exp: order.Buy},
		{data: `"1"`, exp: order.Sell},

		// Bad data bings
		{data: `null`, err: order.ErrSideIsInvalid},
		{data: `1.2`, err: order.ErrSideIsInvalid},
		{data: `-0`, err: order.ErrSideIsInvalid},
		{data: `-1`, err: order.ErrSideIsInvalid},
		{data: `""`, err: order.ErrSideIsInvalid},
		{data: `"buy"`, err: order.ErrSideIsInvalid},
		{data: `true`, err: order.ErrSideIsInvalid},
		{data: `1e0`, err: order.ErrSideIsInvalid},
		{data: `"-0"`, err: order.ErrSideIsInvalid},
		{data: `"2"`, err: order.ErrSideIsInvalid},
		{data: `"1e0"`, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.data, func(t *testing.T) {
			t.Parallel()
			var s OrderSide
			err := json.Unmarshal([]byte(tc.data), &s)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "Unmarshal should reject a non-binary side")
				return
			}
			require.NoError(t, err, "Unmarshal must not error")
			assert.Equal(t, tc.exp, s.Side(), "Side should decode")
		})
	}
}

func TestOrderSideSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		side OrderSide
		exp  order.Side
	}{
		{name: "buy", side: OrderSide(order.Buy), exp: order.Buy},
		{name: "sell", side: OrderSide(order.Sell), exp: order.Sell},
		{name: "zero value", exp: order.UnknownSide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.side.Side(), "Side should return the order side")
		})
	}
}

func TestOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var levels []OrderbookLevel
	require.NoError(t, json.Unmarshal([]byte(`[["84010","0.01190"],["84006","0.33643","1458532827766784"]]`), &levels), "Unmarshal must not error")
	exp := []OrderbookLevel{
		{Price: 84010, Amount: 0.0119},
		{Price: 84006, Amount: 0.33643, OrderID: 1458532827766784},
	}
	assert.Equal(t, exp, levels, "OrderbookLevel should unmarshal grouped and ungrouped levels")

	reused := OrderbookLevel{Price: 1, Amount: 1, OrderID: 1}
	require.NoError(t, reused.UnmarshalJSON([]byte(`["84010","0.01190"]`)), "UnmarshalJSON must not error")
	assert.Equal(t, OrderbookLevel{Price: 84010, Amount: 0.0119}, reused, "UnmarshalJSON should clear the order ID of a reused level")

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "price only", data: `["84010"]`},
		{name: "four fields", data: `["84010","0.01190","1","2"]`},
		{name: "malformed price", data: `["wibble","0.01190"]`},
		{name: "malformed amount", data: `["84010","wibble"]`},
		{name: "malformed order ID", data: `["84010","0.01190","wibble"]`},
	} {
		// Serial and in order, as the object below goes to the same function
		t.Run(tc.name, func(t *testing.T) {
			var l OrderbookLevel
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &l), errMalformedOrderbookLevel, "Unmarshal should error on %s", tc.data)
		})
	}
	t.Run("object", func(t *testing.T) {
		t.Parallel()
		var l OrderbookLevel
		assert.Error(t, json.Unmarshal([]byte(`{"price":"84010"}`), &l), "Unmarshal should error on an object")
	})
}

func TestOrderTransactionUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const data = `{"tid":209895701,"price":"20000.00","fee":8,"datetime":"2022-01-31 14:45:15.322000","type":2,"btc":"0.10000000","usd":-2000,"trade_uti":"BSTP-20220131-209895701","note":"wibble"}`
	var tx OrderTransaction
	require.NoError(t, json.Unmarshal([]byte(data), &tx), "Unmarshal must not error")
	exp := OrderTransaction{
		TradeID:  209895701,
		Price:    20000,
		Fee:      8,
		DateTime: time.Date(2022, 1, 31, 14, 45, 15, 322000000, time.UTC),
		Type:     TransactionTypeMarketTrade,
		Amounts:  map[currency.Code]float64{currency.BTC: 0.1, currency.USD: -2000},
	}
	assert.Equal(t, exp, tx, "OrderTransaction should unmarshal the currency amounts and ignore other fields")

	require.NoError(t, json.Unmarshal([]byte(`{"fee":null,"datetime":null}`), &tx), "Unmarshal must not error on a null fee or date")
	assert.Equal(t, OrderTransaction{Amounts: map[currency.Code]float64{}}, tx, "a null fee and date should decode as zero")

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "malformed trade ID", data: `{"tid":"wibble"}`},
		{name: "malformed price", data: `{"price":"wibble"}`},
		{name: "null price", data: `{"price":null}`},
		{name: "malformed fee", data: `{"fee":"wibble"}`},
		{name: "malformed date", data: `{"datetime":"wibble"}`},
		{name: "malformed type", data: `{"type":"wibble"}`},
	} {
		// Serial and in order, as the cases and the array below decode into the same transaction
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &tx), errMalformedTransaction, "Unmarshal should error on %s", tc.data)
		})
	}
	t.Run("array", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, json.Unmarshal([]byte(`[]`), &tx), "Unmarshal should error on an array")
	})
}

func TestUserTransactionResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const data = `{"id":1,"datetime":"2022-03-01 10:54:53.849000","type":"2","fee":"0.40","order_id":1458532827766784,"self_trade":true,"self_trade_order_id":1458532827766785,"btc":"0.1","usd":-2000,"eur":0.0,"usdc":null,"btc_usd":20000,"eur_usd":"1.08","trade-uti":"1","note":"wibble","fee_currency":"USD"}`
	var tx UserTransactionResponse
	require.NoError(t, json.Unmarshal([]byte(data), &tx), "Unmarshal must not error")
	exp := UserTransactionResponse{
		ID:               1,
		DateTime:         time.Date(2022, 3, 1, 10, 54, 53, 849000000, time.UTC),
		Type:             TransactionTypeMarketTrade,
		Fee:              0.4,
		OrderID:          1458532827766784,
		SelfTrade:        true,
		SelfTradeOrderID: 1458532827766785,
		Amounts:          map[currency.Code]float64{currency.BTC: 0.1, currency.USD: -2000, currency.EUR: 0},
		ExchangeRates: map[currency.Pair]float64{
			currency.NewBTCUSD():                         20000,
			currency.NewPair(currency.EUR, currency.USD): 1.08,
		},
	}
	assert.Equal(t, exp, tx, "UserTransactionResponse should unmarshal the currency amounts and exchange rates, and ignore other fields")

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "malformed ID", data: `{"id":"wibble"}`},
		{name: "malformed date", data: `{"datetime":"wibble"}`},
		{name: "type out of range", data: `{"type":256}`},
		{name: "malformed fee", data: `{"fee":"wibble"}`},
		{name: "malformed order ID", data: `{"order_id":"wibble"}`},
		{name: "malformed self trade flag", data: `{"self_trade":"wibble"}`},
		{name: "malformed self trade order ID", data: `{"self_trade_order_id":"wibble"}`},
	} {
		// Serial and in order, as the cases and the array below decode into the same transaction
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &tx), errMalformedTransaction, "Unmarshal should error on %s", tc.data)
		})
	}
	t.Run("array", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, json.Unmarshal([]byte(`[]`), &tx), "Unmarshal should error on an array")
	})
}

func TestUnconfirmedDepositsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	deposit := UnconfirmedDepositResponse{Amount: 0.1, Address: "3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF", Confirmations: 2, TransferID: 1}
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "single deposit", data: ` {"amount":"0.1","address":"3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF","confirmations":2,"transfer_id":1}`},
		{name: "list of deposits", data: `[{"amount":"0.1","address":"3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF","confirmations":2,"transfer_id":1}]`},
	} {
		// Serial and in order, as the malformed deposits below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			var deposits unconfirmedDeposits
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &deposits), "Unmarshal must not error for %s", tc.data)
			assert.Equalf(t, unconfirmedDeposits{deposit}, deposits, "Unmarshal should decode %s", tc.data)
		})
	}
	t.Run("malformed deposit", func(t *testing.T) {
		t.Parallel()
		var deposits unconfirmedDeposits
		assert.Error(t, json.Unmarshal([]byte(`{"amount":[]}`), &deposits), "Unmarshal should error on a malformed deposit")
	})
	t.Run("malformed deposits", func(t *testing.T) {
		t.Parallel()
		var deposits unconfirmedDeposits
		assert.Error(t, json.Unmarshal([]byte(`[{"amount":[]}]`), &deposits), "Unmarshal should error on malformed deposits")
	})
}

func TestParseOptionalNumber(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  float64
		err  bool
	}{
		{name: "null", data: `null`},
		{name: "quoted number", data: `"1.5"`, exp: 1.5},
		{name: "bare number", data: `2`, exp: 2},
		{name: "non-numeric string", data: `"wibble"`, err: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			n, err := parseOptionalNumber(json.RawMessage(tc.data))
			if tc.err {
				assert.Error(t, err, "parseOptionalNumber should error on a non-numeric value")
				return
			}
			require.NoError(t, err, "parseOptionalNumber must not error")
			assert.Equal(t, tc.exp, n, "parseOptionalNumber should decode the number")
		})
	}
}

func TestParseOptionalDateTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  time.Time
		err  bool
	}{
		{name: "null", data: `null`},
		{name: "date and time", data: `"2022-01-31 14:45:15"`, exp: time.Date(2022, 1, 31, 14, 45, 15, 0, time.UTC)},
		{name: "fractional seconds", data: `"2022-01-31 14:45:15.322000"`, exp: time.Date(2022, 1, 31, 14, 45, 15, 322000000, time.UTC)},
		{name: "malformed string", data: `"wibble"`, err: true},
		{name: "number", data: `1643640315`, err: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dt, err := parseOptionalDateTime(json.RawMessage(tc.data))
			if tc.err {
				assert.Error(t, err, "parseOptionalDateTime should error on a malformed date and time")
				return
			}
			require.NoError(t, err, "parseOptionalDateTime must not error")
			assert.Equal(t, tc.exp, dt, "parseOptionalDateTime should decode the date and time")
		})
	}
}

func TestParseNumber(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  float64
	}{
		{name: "quoted number", data: `"1.5"`, exp: 1.5},
		{name: "bare number", data: `2`, exp: 2},
		{name: "empty string", data: `""`, exp: 0},
	} {
		// Serial and in order, as the null and non-numeric cases below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			n, err := parseNumber(json.RawMessage(tc.data))
			require.NoErrorf(t, err, "parseNumber must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, n, "parseNumber should decode %s", tc.data)
		})
	}
	t.Run("null", func(t *testing.T) {
		t.Parallel()
		_, err := parseNumber(json.RawMessage(`null`))
		assert.ErrorIs(t, err, errNullNumber, "parseNumber should error on null")
	})
	t.Run("non-numeric string", func(t *testing.T) {
		t.Parallel()
		_, err := parseNumber(json.RawMessage(`"wibble"`))
		assert.Error(t, err, "parseNumber should error on a non-numeric string")
	})
}

func TestIsAlphanumeric(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    string
		exp  bool
	}{
		{name: "lower case", s: "btc", exp: true},
		{name: "upper case", s: "USD", exp: true},
		{name: "leading digit", s: "1INCH", exp: true},
		{name: "empty", s: "", exp: false},
		{name: "underscore", s: "trade_uti", exp: false},
		{name: "hyphen", s: "usd-perp", exp: false},
		{name: "non-ASCII letter", s: "ü", exp: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isAlphanumeric(tc.s), "isAlphanumeric should be correct for %q", tc.s)
		})
	}
}

func TestUnmarshalTransactionType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "bare number", data: `2`},
		{name: "quoted number", data: `"2"`},
	} {
		// Serial and in order, as the malformed types below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			var typ TransactionType
			require.NoErrorf(t, unmarshalTransactionType(json.RawMessage(tc.data), &typ), "unmarshalTransactionType must not error for %s", tc.data)
			assert.Equalf(t, TransactionTypeMarketTrade, typ, "unmarshalTransactionType should decode %s", tc.data)
		})
	}
	t.Run("out of range type", func(t *testing.T) {
		t.Parallel()
		var typ TransactionType
		assert.Error(t, unmarshalTransactionType(json.RawMessage(`256`), &typ), "unmarshalTransactionType should error on an out of range type")
	})
	t.Run("non-numeric type", func(t *testing.T) {
		t.Parallel()
		var typ TransactionType
		assert.Error(t, unmarshalTransactionType(json.RawMessage(`"wibble"`), &typ), "unmarshalTransactionType should error on a non-numeric type")
	})
}

func TestEnabledStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  EnabledStatus
	}{
		{name: "enabled", data: `"Enabled"`, exp: true},
		{name: "disabled", data: `"Disabled"`, exp: false},
		{name: "lower case enabled", data: `"enabled"`, exp: false},
		{name: "other status", data: `"Paused"`, exp: false},
		{name: "empty string", data: `""`, exp: false},
		{name: "null", data: `null`, exp: false},
	} {
		// Serial and in order, as the malformed statuses below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			reused := !tc.exp
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &reused), "Unmarshal must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, reused, "Unmarshal should decode %s", tc.data)
		})
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "boolean", data: `true`},
		{name: "number", data: `1`},
		{name: "array", data: `[]`},
		{name: "object", data: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var s EnabledStatus
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &s), errInvalidEnabledStatus, "Unmarshal should error on %s", tc.data)
		})
	}
}

func TestEnabledStatusString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Enabled", EnabledStatus(true).String(), "String should return Enabled for true")
	assert.Equal(t, "Disabled", EnabledStatus(false).String(), "String should return Disabled for false")
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  time.Time
	}{
		{name: "RFC 3339", data: `"2026-05-14T10:00:00Z"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
		{name: "RFC 3339 with fractional seconds and an offset", data: `"2026-05-14T20:00:00.5+10:00"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 500000000, time.UTC)},
		{name: "RFC 3339 without a zone", data: `"2026-05-14T10:00:00"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
		{name: "date and time", data: `"2026-05-14 10:00:00"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
		{name: "date and time with microseconds", data: `"2026-05-14 10:00:00.123456"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 123456000, time.UTC)},
		{name: "date and time with an offset", data: `"2026-05-14 20:00:00+10:00"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
		{name: "date and time with fractional seconds and a negative offset", data: `"2026-05-14 05:00:00.250000-05:00"`, exp: time.Date(2026, 5, 14, 10, 0, 0, 250000000, time.UTC)},
		{name: "empty string", data: `""`, exp: time.Time{}},
		{name: "null", data: `null`, exp: time.Time{}},
	} {
		// Serial and in order, as the cases below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			var d DateTime
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &d), "Unmarshal must not error for %s", tc.data)
			assert.Truef(t, tc.exp.Equal(d.Time()), "Unmarshal should decode %s as %s, not %s", tc.data, tc.exp, d.Time())
		})
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "empty string clears a reused time", data: `""`},
		{name: "null clears a reused time", data: `null`},
	} {
		// Serial and in order, as the malformed values below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			reused := DateTime(time.Unix(1, 0))
			require.NoErrorf(t, reused.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error for %s", tc.data)
			assert.Truef(t, reused.Time().IsZero(), "UnmarshalJSON should clear a reused time for %s", tc.data)
		})
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "malformed string", data: `"wibble"`},
		{name: "date only", data: `"2026-05-14"`},
		{name: "number", data: `1778752800`},
		{name: "array", data: `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var d DateTime
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &d), errInvalidDateTime, "Unmarshal should error on %s", tc.data)
		})
	}
}

func TestDateTimeTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		exp  time.Time
	}{
		{name: "zero value"},
		{name: "UTC", exp: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
		{name: "offset zone", exp: time.Date(2026, 5, 14, 20, 0, 0, 500000000, time.FixedZone("", 10*60*60))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, DateTime(tc.exp).Time(), "Time should return the time unchanged, zone included")
		})
	}
}

// newRetailInfoWithEveryField returns retail details with every field set
func newRetailInfoWithEveryField() *RetailInfo {
	return &RetailInfo{
		FirstName:            "Satoshi",
		LastName:             "Nakamoto",
		DateOfBirth:          time.Date(1975, 4, 5, 0, 0, 0, 0, time.UTC),
		PlaceOfBirth:         "Tokyo",
		IDType:               IDTypePassport,
		IDNumber:             "A1234567",
		StreetAndHouseNumber: "1 Main Street",
		City:                 "Sydney",
		PostalCode:           "2000",
		Country:              "AU",
	}
}

func TestRetailInfoMarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("every field", func(t *testing.T) {
		t.Parallel()
		data, err := json.Marshal(&CustomerInfo{RetailInfo: newRetailInfoWithEveryField()})
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, `{"retail_info":{"first_name":"Satoshi","last_name":"Nakamoto","date_of_birth":"1975-04-05","place_of_birth":"Tokyo","id_type":"passport","id_number":"A1234567","street_and_house_number":"1 Main Street","city":"Sydney","zip":"2000","country":"AU"}}`, string(data), "Marshal should send the documented fields with a date only date of birth")
	})
	t.Run("unset details", func(t *testing.T) {
		t.Parallel()
		data, err := json.Marshal(&RetailInfo{FirstName: "Satoshi", LastName: "Nakamoto"})
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, `{"first_name":"Satoshi","last_name":"Nakamoto"}`, string(data), "Marshal should omit unset details")
	})
	t.Run("value", func(t *testing.T) {
		t.Parallel()
		data, err := json.Marshal(RetailInfo{FirstName: "Satoshi", LastName: "Nakamoto"})
		require.NoError(t, err, "Marshal must not error for a value")
		assert.JSONEq(t, `{"first_name":"Satoshi","last_name":"Nakamoto"}`, string(data), "Marshal should send the documented fields for a value")
	})
}

func TestRetailInfoUnmarshalJSON(t *testing.T) {
	t.Parallel()
	// Serial, so that the decodes run before the refusals below, which go to the same function
	t.Run("marshalled details", func(t *testing.T) {
		info := newRetailInfoWithEveryField()
		data, err := json.Marshal(&CustomerInfo{RetailInfo: info})
		require.NoError(t, err, "Marshal must not error")
		var decoded CustomerInfo
		require.NoError(t, json.Unmarshal(data, &decoded), "Unmarshal must not error")
		assert.Equal(t, info, decoded.RetailInfo, "Unmarshal should decode what Marshal encodes")
	})
	for _, tc := range []struct {
		name   string
		reused RetailInfo
		data   string
		exp    RetailInfo
	}{
		{name: "without a date of birth", data: `{"first_name":"Satoshi","last_name":"Nakamoto"}`, exp: RetailInfo{FirstName: "Satoshi", LastName: "Nakamoto"}},
		{name: "reused details", reused: RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC), City: "Sydney"}, data: `{"first_name":"Satoshi","last_name":"Nakamoto"}`, exp: RetailInfo{FirstName: "Satoshi", LastName: "Nakamoto"}},
	} {
		// Serial and in order, as the refusals below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			r := tc.reused
			require.NoError(t, r.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, r, "UnmarshalJSON should replace every detail with the decoded ones")
		})
	}
	t.Run("malformed date of birth", func(t *testing.T) {
		t.Parallel()
		var r RetailInfo
		assert.ErrorIs(t, json.Unmarshal([]byte(`{"date_of_birth":"05/04/1975"}`), &r), errInvalidDateTime, "Unmarshal should error on a malformed date of birth")
	})
	t.Run("array", func(t *testing.T) {
		t.Parallel()
		var r RetailInfo
		assert.Error(t, json.Unmarshal([]byte(`[]`), &r), "Unmarshal should error on an array")
	})
}
