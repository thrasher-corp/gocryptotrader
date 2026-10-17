package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

	for _, data := range []string{
		`["84010"]`,
		`["84010","0.01190","1","2"]`,
		`["wibble","0.01190"]`,
		`["84010","wibble"]`,
		`["84010","0.01190","wibble"]`,
	} {
		var l OrderbookLevel
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &l), errMalformedOrderbookLevel, "Unmarshal should error on %s", data)
	}
	var l OrderbookLevel
	assert.Error(t, json.Unmarshal([]byte(`{"price":"84010"}`), &l), "Unmarshal should error on an object")
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

	for _, data := range []string{
		`{"tid":"wibble"}`,
		`{"price":"wibble"}`,
		`{"price":null}`,
		`{"fee":"wibble"}`,
		`{"datetime":"wibble"}`,
		`{"type":"wibble"}`,
	} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &tx), errMalformedTransaction, "Unmarshal should error on %s", data)
	}
	assert.Error(t, json.Unmarshal([]byte(`[]`), &tx), "Unmarshal should error on an array")
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

	for _, data := range []string{
		`{"id":"wibble"}`,
		`{"datetime":"wibble"}`,
		`{"type":256}`,
		`{"fee":"wibble"}`,
		`{"order_id":"wibble"}`,
		`{"self_trade":"wibble"}`,
		`{"self_trade_order_id":"wibble"}`,
	} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &tx), errMalformedTransaction, "Unmarshal should error on %s", data)
	}
	assert.Error(t, json.Unmarshal([]byte(`[]`), &tx), "Unmarshal should error on an array")
}

func TestUnconfirmedDepositsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	deposit := UnconfirmedDepositResponse{Amount: 0.1, Address: "3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF", Confirmations: 2, TransferID: 1}
	for _, data := range []string{
		` {"amount":"0.1","address":"3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF","confirmations":2,"transfer_id":1}`,
		`[{"amount":"0.1","address":"3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF","confirmations":2,"transfer_id":1}]`,
	} {
		var deposits unconfirmedDeposits
		require.NoErrorf(t, json.Unmarshal([]byte(data), &deposits), "Unmarshal must not error for %s", data)
		assert.Equalf(t, unconfirmedDeposits{deposit}, deposits, "Unmarshal should decode %s", data)
	}
	var deposits unconfirmedDeposits
	assert.Error(t, json.Unmarshal([]byte(`{"amount":[]}`), &deposits), "Unmarshal should error on a malformed deposit")
	assert.Error(t, json.Unmarshal([]byte(`[{"amount":[]}]`), &deposits), "Unmarshal should error on malformed deposits")
}

func TestParseNumber(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]float64{`"1.5"`: 1.5, `2`: 2, `""`: 0} {
		n, err := parseNumber(json.RawMessage(data))
		require.NoErrorf(t, err, "parseNumber must not error for %s", data)
		assert.Equalf(t, exp, n, "parseNumber should decode %s", data)
	}
	_, err := parseNumber(json.RawMessage(`null`))
	assert.ErrorIs(t, err, errNullNumber, "parseNumber should error on null")
	_, err = parseNumber(json.RawMessage(`"wibble"`))
	assert.Error(t, err, "parseNumber should error on a non-numeric string")
}

func TestIsAlphanumeric(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]bool{"btc": true, "USD": true, "1INCH": true, "": false, "trade_uti": false, "usd-perp": false, "ü": false} {
		assert.Equalf(t, exp, isAlphanumeric(s), "isAlphanumeric should be correct for %q", s)
	}
}

func TestUnmarshalTransactionType(t *testing.T) {
	t.Parallel()
	for _, data := range []string{`2`, `"2"`} {
		var typ TransactionType
		require.NoErrorf(t, unmarshalTransactionType(json.RawMessage(data), &typ), "unmarshalTransactionType must not error for %s", data)
		assert.Equalf(t, TransactionTypeMarketTrade, typ, "unmarshalTransactionType should decode %s", data)
	}
	var typ TransactionType
	assert.Error(t, unmarshalTransactionType(json.RawMessage(`256`), &typ), "unmarshalTransactionType should error on an out of range type")
	assert.Error(t, unmarshalTransactionType(json.RawMessage(`"wibble"`), &typ), "unmarshalTransactionType should error on a non-numeric type")
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]time.Time{
		`"2026-05-14T10:00:00Z"`:             time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		`"2026-05-14T20:00:00.5+10:00"`:      time.Date(2026, 5, 14, 10, 0, 0, 500000000, time.UTC),
		`"2026-05-14T10:00:00"`:              time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		`"2026-05-14 10:00:00"`:              time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		`"2026-05-14 10:00:00.123456"`:       time.Date(2026, 5, 14, 10, 0, 0, 123456000, time.UTC),
		`"2026-05-14 20:00:00+10:00"`:        time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		`"2026-05-14 05:00:00.250000-05:00"`: time.Date(2026, 5, 14, 10, 0, 0, 250000000, time.UTC),
		`""`:                                 {},
		`null`:                               {},
	} {
		var d DateTime
		require.NoErrorf(t, json.Unmarshal([]byte(data), &d), "Unmarshal must not error for %s", data)
		assert.Truef(t, exp.Equal(d.Time()), "Unmarshal should decode %s as %s, not %s", data, exp, d.Time())
	}
	for _, data := range []string{`""`, `null`} {
		reused := DateTime(time.Unix(1, 0))
		require.NoErrorf(t, reused.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error for %s", data)
		assert.Truef(t, reused.Time().IsZero(), "UnmarshalJSON should clear a reused time for %s", data)
	}
	for _, data := range []string{`"wibble"`, `"2026-05-14"`, `1778752800`, `[]`} {
		var d DateTime
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &d), errInvalidDateTime, "Unmarshal should error on %s", data)
	}
}

func TestRetailInfoJSON(t *testing.T) {
	t.Parallel()
	info := &RetailInfo{
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
	data, err := json.Marshal(&CustomerInfo{RetailInfo: info})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"retail_info":{"first_name":"Satoshi","last_name":"Nakamoto","date_of_birth":"1975-04-05","place_of_birth":"Tokyo","id_type":"passport","id_number":"A1234567","street_and_house_number":"1 Main Street","city":"Sydney","zip":"2000","country":"AU"}}`, string(data), "Marshal should send the documented fields with a date only date of birth")

	var decoded CustomerInfo
	require.NoError(t, json.Unmarshal(data, &decoded), "Unmarshal must not error")
	assert.Equal(t, info, decoded.RetailInfo, "Unmarshal should decode what Marshal encodes")

	data, err = json.Marshal(&RetailInfo{FirstName: "Satoshi", LastName: "Nakamoto"})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"first_name":"Satoshi","last_name":"Nakamoto"}`, string(data), "Marshal should omit unset details")

	var r RetailInfo
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"date_of_birth":"05/04/1975"}`), &r), errInvalidDateTime, "Unmarshal should error on a malformed date of birth")
	assert.Error(t, json.Unmarshal([]byte(`[]`), &r), "Unmarshal should error on an array")
}
