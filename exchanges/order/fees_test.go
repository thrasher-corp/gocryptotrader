package order

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func newTestFees(t *testing.T, amounts ...FeeAmount) Fees {
	t.Helper()
	f, err := NewFees(amounts...)
	require.NoError(t, err, "NewFees must not error")
	return f
}

func detailWithFees(f Fees) Detail {
	var d Detail
	d.SetFees(f)
	return d
}

func fillWithFees(f Fees) TradeHistory {
	var fill TradeHistory
	fill.SetFees(f)
	return fill
}

func TestNewFees(t *testing.T) {
	t.Parallel()

	f, err := NewFees()
	require.NoError(t, err, "NewFees must not error without amounts")
	assert.Equal(t, Fees{reported: true}, f, "NewFees should report no fees without amounts")

	amounts := []FeeAmount{
		{Currency: currency.NewCode("usdt"), Amount: -0.2},
		{Currency: currency.BTC},
		{Currency: currency.NewCode("1inch"), Amount: 0.5},
	}
	f, err = NewFees(amounts...)
	require.NoError(t, err, "NewFees must not error")
	exp := Fees{
		reported: true,
		amounts: []FeeAmount{
			{Currency: currency.NewCode("1INCH"), Amount: 0.5},
			{Currency: currency.BTC},
			{Currency: currency.USDT, Amount: -0.2},
		},
	}
	assert.Equal(t, exp, f, "NewFees should upper-case currencies, keep zero and negative amounts and order by currency")
	amounts[0].Amount = 1
	assert.Equal(t, exp, f, "NewFees should not share its amounts with the caller")

	for _, tc := range []struct {
		name    string
		amounts []FeeAmount
		err     error
	}{
		{name: "empty currency", amounts: []FeeAmount{{Amount: 1}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "NaN", amounts: []FeeAmount{{Currency: currency.BTC, Amount: math.NaN()}}, err: ErrInvalidFeeAmount},
		{name: "positive infinity", amounts: []FeeAmount{{Currency: currency.BTC, Amount: math.Inf(1)}}, err: ErrInvalidFeeAmount},
		{name: "negative infinity", amounts: []FeeAmount{{Currency: currency.BTC, Amount: math.Inf(-1)}}, err: ErrInvalidFeeAmount},
		{name: "duplicate currency", amounts: []FeeAmount{{Currency: currency.BTC, Amount: 1}, {Currency: currency.BTC, Amount: 2}}, err: ErrDuplicateFeeCurrency},
		{name: "duplicate currency in another case", amounts: []FeeAmount{{Currency: currency.USDT, Amount: 1}, {Currency: currency.NewCode("usdt"), Amount: 1}}, err: ErrDuplicateFeeCurrency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := NewFees(tc.amounts...)
			require.ErrorIs(t, err, tc.err, "NewFees must reject the amounts")
			assert.False(t, f.IsReported(), "NewFees should return fees that were not reported on error")
		})
	}
}

func TestFeesIsReported(t *testing.T) {
	t.Parallel()
	assert.False(t, Fees{}.IsReported(), "IsReported should be false for the zero value")
	assert.True(t, newTestFees(t).IsReported(), "IsReported should be true for a snapshot without amounts")
	assert.True(t, newTestFees(t, FeeAmount{Currency: currency.BTC}).IsReported(), "IsReported should be true for a snapshot")
}

func TestFeesAmounts(t *testing.T) {
	t.Parallel()
	assert.Nil(t, Fees{}.Amounts(), "Amounts should be nil for fees that were not reported")
	assert.Nil(t, newTestFees(t).Amounts(), "Amounts should be nil for a snapshot without amounts")

	f := newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: 0.2}, FeeAmount{Currency: currency.BTC, Amount: 0.001})
	exp := []FeeAmount{{Currency: currency.BTC, Amount: 0.001}, {Currency: currency.USDT, Amount: 0.2}}
	amounts := f.Amounts()
	require.Equal(t, exp, amounts, "Amounts must return the amounts ordered by currency")
	amounts[0].Amount = 1
	assert.Equal(t, exp, f.Amounts(), "Amounts should return a copy")
}

func TestFeesMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fees Fees
		exp  string
	}{
		{name: "not reported", fees: Fees{}, exp: `null`},
		{name: "no fees", fees: newTestFees(t), exp: `[]`},
		{
			name: "fees",
			fees: newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: -0.2}, FeeAmount{Currency: currency.BTC}),
			exp:  `[{"Currency":"BTC","Amount":0},{"Currency":"USDT","Amount":-0.2}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := tc.fees.MarshalJSON()
			require.NoError(t, err, "MarshalJSON must not error")
			assert.JSONEq(t, tc.exp, string(b), "MarshalJSON should encode the fees")
		})
	}
}

func TestFeesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  Fees
	}{
		{name: "null", data: `null`, exp: Fees{}},
		{name: "no fees", data: `[]`, exp: Fees{reported: true}},
		{
			name: "fees",
			data: `[{"Currency":"usdt","Amount":-0.2},{"Currency":"BTC","Amount":0}]`,
			exp:  Fees{reported: true, amounts: []FeeAmount{{Currency: currency.BTC}, {Currency: currency.USDT, Amount: -0.2}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newTestFees(t, FeeAmount{Currency: currency.ETH, Amount: 1})
			require.NoError(t, f.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, f, "UnmarshalJSON should replace the fees with the decoded ones")
		})
	}

	exp := newTestFees(t, FeeAmount{Currency: currency.ETH, Amount: 1})
	f := exp
	assert.Error(t, f.UnmarshalJSON([]byte(`{}`)), "UnmarshalJSON should reject an object")
	err := f.UnmarshalJSON([]byte(`[{"Currency":"BTC","Amount":1},{"Currency":"btc","Amount":2}]`))
	assert.ErrorIs(t, err, ErrDuplicateFeeCurrency, "UnmarshalJSON should reject a duplicate currency")
	err = f.UnmarshalJSON([]byte(`[{"Amount":1}]`))
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UnmarshalJSON should reject an empty currency")
	assert.Equal(t, exp, f, "UnmarshalJSON should leave the fees alone on error")
}

// TestFeesJSONRoundTrip checks that fees that were not reported, no fees and a zero fee stay distinct through JSON
func TestFeesJSONRoundTrip(t *testing.T) {
	t.Parallel()
	type container struct {
		Fees Fees
	}
	for _, tc := range []struct {
		name string
		fees Fees
		json string
	}{
		{name: "not reported", fees: Fees{}, json: `{"Fees":null}`},
		{name: "no fees", fees: newTestFees(t), json: `{"Fees":[]}`},
		{name: "zero fee", fees: newTestFees(t, FeeAmount{Currency: currency.BTC}), json: `{"Fees":[{"Currency":"BTC","Amount":0}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(container{Fees: tc.fees})
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.json, string(b), "Marshal should encode the fees")
			var decoded container
			require.NoError(t, json.Unmarshal(b, &decoded), "Unmarshal must not error")
			assert.Equal(t, tc.fees, decoded.Fees, "Unmarshal should restore the fees")
		})
	}
	var decoded container
	require.NoError(t, json.Unmarshal([]byte(`{}`), &decoded), "Unmarshal must not error")
	assert.False(t, decoded.Fees.IsReported(), "Unmarshal should leave omitted fees unreported")
}

func TestFeesEqual(t *testing.T) {
	t.Parallel()
	btc := newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 1})
	assert.True(t, Fees{}.equal(Fees{}), "equal should be true for fees that were not reported")
	assert.True(t, newTestFees(t).equal(newTestFees(t)), "equal should be true for snapshots without amounts")
	assert.True(t, btc.equal(newTestFees(t, FeeAmount{Currency: currency.NewCode("btc"), Amount: 1})), "equal should be true for the same amounts")
	assert.False(t, Fees{}.equal(newTestFees(t)), "equal should tell fees that were not reported from no fees")
	assert.False(t, btc.equal(newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 2})), "equal should compare amounts")
	assert.False(t, btc.equal(newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: 1})), "equal should compare currencies")
	assert.False(t, btc.equal(newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 1}, FeeAmount{Currency: currency.USDT, Amount: 1})), "equal should compare every currency")
}

func TestFeesScalar(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		fees     Fees
		fee      float64
		feeAsset currency.Code
	}{
		{name: "not reported", fees: Fees{}},
		{name: "no fees", fees: newTestFees(t)},
		{name: "one currency", fees: newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: -0.2}), fee: -0.2, feeAsset: currency.USDT},
		{name: "several currencies", fees: newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 0.001}, FeeAmount{Currency: currency.USDT, Amount: 0.2})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fee, feeAsset := tc.fees.scalar()
			assert.Equal(t, tc.fee, fee, "scalar should return the only amount, or zero")
			assert.Equal(t, tc.feeAsset, feeAsset, "scalar should return the only currency, or an empty one")
		})
	}
}

// TestSetFees checks the compatibility policy for Fee and FeeAsset: reported fees in one currency keep them as before,
// while no fees or fees in several currencies leave no single amount standing for part of the fees
func TestSetFees(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		fees     Fees
		fee      float64
		feeAsset currency.Code
	}{
		{name: "not reported", fees: Fees{}, fee: 1, feeAsset: currency.ETH},
		{name: "no fees", fees: newTestFees(t)},
		{name: "one currency", fees: newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: -0.2}), fee: -0.2, feeAsset: currency.USDT},
		{name: "several currencies", fees: newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 0.001}, FeeAmount{Currency: currency.USDT, Amount: 0.2})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &Detail{Fee: 1, FeeAsset: currency.ETH}
			d.SetFees(tc.fees)
			assert.Equal(t, &Detail{Fees: tc.fees, Fee: tc.fee, FeeAsset: tc.feeAsset}, d, "Detail.SetFees should set the fees and their single fee")

			s := &SubmitResponse{Fee: 1, FeeAsset: currency.ETH}
			s.SetFees(tc.fees)
			assert.Equal(t, &SubmitResponse{Fees: tc.fees, Fee: tc.fee, FeeAsset: tc.feeAsset}, s, "SubmitResponse.SetFees should set the fees and their single fee")

			fill := &TradeHistory{Fee: 1, FeeAsset: currency.ETH.String()}
			fill.SetFees(tc.fees)
			assert.Equal(t, &TradeHistory{Fees: tc.fees, Fee: tc.fee, FeeAsset: tc.feeAsset.String()}, fill, "TradeHistory.SetFees should set the fees and their single fee")
		})
	}
}

func TestDetailMergeFees(t *testing.T) {
	t.Parallel()
	btc := newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 0.001})
	btcUSDT := newTestFees(t, FeeAmount{Currency: currency.BTC, Amount: 0.001}, FeeAmount{Currency: currency.USDT, Amount: 0.2})
	for _, tc := range []struct {
		name    string
		stored  Detail
		update  Detail
		exp     Detail
		changed bool
	}{
		{name: "snapshot replaces a single fee", stored: Detail{Fee: 0.5, FeeAsset: currency.ETH}, update: detailWithFees(btc), exp: detailWithFees(btc), changed: true},
		{name: "second currency clears the single fee", stored: detailWithFees(btc), update: detailWithFees(btcUSDT), exp: Detail{Fees: btcUSDT}, changed: true},
		{name: "repeated snapshot", stored: detailWithFees(btcUSDT), update: detailWithFees(btcUSDT), exp: Detail{Fees: btcUSDT}},
		{name: "empty snapshot clears the fees", stored: detailWithFees(btcUSDT), update: detailWithFees(newTestFees(t)), exp: Detail{Fees: newTestFees(t)}, changed: true},
		{name: "omitted fees keep reported fees", stored: detailWithFees(btcUSDT), update: Detail{}, exp: Detail{Fees: btcUSDT}},
		{name: "single fee cannot change reported fees", stored: detailWithFees(btcUSDT), update: Detail{Fee: 0.3, FeeAsset: currency.USDT}, exp: Detail{Fees: btcUSDT}},
		{name: "snapshot corrects a single fee set apart", stored: Detail{Fees: btc, Fee: 9}, update: detailWithFees(btc), exp: detailWithFees(btc), changed: true},
		{name: "positive single fee replaces a single fee", stored: Detail{Fee: 0.1}, update: Detail{Fee: 0.2}, exp: Detail{Fee: 0.2}, changed: true},
		{name: "zero single fee keeps a single fee", stored: Detail{Fee: 0.1}, update: Detail{}, exp: Detail{Fee: 0.1}},
		{name: "negative single fee keeps a single fee", stored: Detail{Fee: 0.1}, update: Detail{Fee: -0.05}, exp: Detail{Fee: 0.1}},
		{name: "same single fee", stored: Detail{Fee: 0.1}, update: Detail{Fee: 0.1}, exp: Detail{Fee: 0.1}},
		{name: "fee asset replaces a single fee asset", stored: Detail{Fee: 0.1, FeeAsset: currency.BTC}, update: Detail{FeeAsset: currency.USDT}, exp: Detail{Fee: 0.1, FeeAsset: currency.USDT}, changed: true},
		{name: "single fee and asset replace both", stored: Detail{Fee: 0.1, FeeAsset: currency.BTC}, update: Detail{Fee: 0.2, FeeAsset: currency.USDT}, exp: Detail{Fee: 0.2, FeeAsset: currency.USDT}, changed: true},
		{name: "same fee asset", stored: Detail{Fee: 0.1, FeeAsset: currency.USDT}, update: Detail{FeeAsset: currency.USDT}, exp: Detail{Fee: 0.1, FeeAsset: currency.USDT}},
		{name: "empty fee asset keeps a single fee asset", stored: Detail{FeeAsset: currency.USDT}, update: Detail{Fee: 0.2}, exp: Detail{Fee: 0.2, FeeAsset: currency.USDT}, changed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			changed := tc.stored.mergeFees(&tc.update)
			assert.Equal(t, tc.changed, changed, "mergeFees should report whether the order changed")
			assert.Equal(t, tc.exp, tc.stored, "mergeFees should merge the fees")
		})
	}
}

func TestTradeHistoryMergeFees(t *testing.T) {
	t.Parallel()
	usdt := newTestFees(t, FeeAmount{Currency: currency.USDT, Amount: 0.1})
	usdtGT := newTestFees(t, FeeAmount{Currency: currency.NewCode("GT"), Amount: 0.01}, FeeAmount{Currency: currency.USDT, Amount: 0.1})
	for _, tc := range []struct {
		name    string
		stored  TradeHistory
		update  TradeHistory
		exp     TradeHistory
		changed bool
	}{
		{name: "snapshot replaces a single fee", stored: TradeHistory{Fee: 0.5, FeeAsset: "ETH"}, update: fillWithFees(usdt), exp: fillWithFees(usdt), changed: true},
		{name: "second currency clears the single fee", stored: fillWithFees(usdt), update: fillWithFees(usdtGT), exp: TradeHistory{Fees: usdtGT}, changed: true},
		{name: "repeated snapshot", stored: fillWithFees(usdtGT), update: fillWithFees(usdtGT), exp: TradeHistory{Fees: usdtGT}},
		{name: "omitted fees keep reported fees", stored: fillWithFees(usdt), update: TradeHistory{}, exp: fillWithFees(usdt)},
		{name: "single fee cannot change reported fees", stored: fillWithFees(usdt), update: TradeHistory{Fee: 0.3}, exp: fillWithFees(usdt)},
		{name: "snapshot corrects a single fee set apart", stored: TradeHistory{Fees: usdt, FeeAsset: "ETH"}, update: fillWithFees(usdt), exp: fillWithFees(usdt), changed: true},
		{name: "single fee replaces a single fee", stored: TradeHistory{Fee: 0.1}, update: TradeHistory{Fee: 0.2}, exp: TradeHistory{Fee: 0.2}, changed: true},
		{name: "zero single fee replaces a single fee", stored: TradeHistory{Fee: 0.1}, update: TradeHistory{}, exp: TradeHistory{}, changed: true},
		{name: "same single fee", stored: TradeHistory{Fee: 0.1}, update: TradeHistory{Fee: 0.1}, exp: TradeHistory{Fee: 0.1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			changed := tc.stored.mergeFees(&tc.update)
			assert.Equal(t, tc.changed, changed, "mergeFees should report whether the fill changed")
			assert.Equal(t, tc.exp, tc.stored, "mergeFees should merge the fees")
		})
	}
}

// TestFeesIsolation checks that fees stored in or returned from an order cannot be changed through any other copy
func TestFeesIsolation(t *testing.T) {
	t.Parallel()
	amounts := []FeeAmount{{Currency: currency.BTC, Amount: 0.001}, {Currency: currency.USDT, Amount: 0.2}}
	exp := []FeeAmount{{Currency: currency.BTC, Amount: 0.001}, {Currency: currency.USDT, Amount: 0.2}}
	fees := newTestFees(t, amounts...)
	update := &Detail{OrderID: "1", Trades: []TradeHistory{{TID: "1"}}}
	update.SetFees(fees)
	update.Trades[0].SetFees(fees)
	stored := &Detail{OrderID: "1"}
	require.NoError(t, stored.UpdateOrderFromDetail(update), "UpdateOrderFromDetail must not error")
	require.Len(t, stored.Trades, 1, "UpdateOrderFromDetail must store the fill")

	other := newTestFees(t, FeeAmount{Currency: currency.ETH, Amount: 1})
	amounts[0].Amount = 1
	returned := stored.Fees.Amounts()
	require.Len(t, returned, 2, "Amounts must return both amounts")
	returned[1].Amount = 1
	update.SetFees(other)
	update.Trades[0].SetFees(other)
	c := stored.Copy()
	c.SetFees(other)
	c.Trades[0].SetFees(other)
	p := stored.CopyToPointer()
	p.SetFees(other)
	p.Trades[0].SetFees(other)

	assert.Equal(t, exp, stored.Fees.Amounts(), "the stored order's fees should not change through another copy")
	assert.Equal(t, exp, stored.Trades[0].Fees.Amounts(), "the stored fill's fees should not change through another copy")
}
