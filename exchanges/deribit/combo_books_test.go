package deribit

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// testUnknownCombo is a combo Deribit has never listed
const testUnknownCombo = "BTC-FS-01JAN20_PERP"

// listedComboDatedFuture returns the first open BTC future, other than the perpetual, that expires at least a day from
// now, which live tests spread against the perpetual
var listedComboDatedFuture = sync.OnceValues(func() (string, error) {
	instruments, err := e.GetInstruments(context.Background(), &InstrumentsRequest{Currency: currency.BTC, Kind: KindFuture})
	if err != nil {
		return "", err
	}
	for i := range instruments {
		if instruments[i].State == "open" && instruments[i].SettlementPeriod != "perpetual" && time.Until(instruments[i].ExpirationTimestamp.Time()) > 24*time.Hour {
			return instruments[i].InstrumentName, nil
		}
	}
	return "", errNoListedMarketDataInstrument
})

// testComboTrades returns the trades of Deribit's documented create combo example, or when testing live a spread of a
// listed future against the perpetual, which Deribit lists as a combo already
func testComboTrades(t *testing.T) []ComboTrade {
	t.Helper()
	if mockTests {
		return []ComboTrade{
			{InstrumentName: "BTC-29APR22-37500-C", Amount: 1, Direction: DirectionBuy},
			{InstrumentName: "BTC-29APR22-37500-P", Amount: 1, Direction: DirectionSell},
		}
	}
	future, err := listedComboDatedFuture()
	require.NoError(t, err, "listedComboDatedFuture must not error")
	return []ComboTrade{
		{InstrumentName: future, Amount: 10, Direction: DirectionBuy},
		{InstrumentName: marketDataPerpetual, Amount: 10, Direction: DirectionSell},
	}
}

// testCreatedCombo is Deribit's documented create combo response
func testCreatedCombo() *Combo {
	return &Combo{
		ID:                "BTC-REV-29APR22-37500",
		InstrumentID:      52,
		State:             "active",
		StateTimestamp:    marketDataMilli(1650960943922),
		CreationTimestamp: marketDataMilli(1650960943000),
		Legs: []ComboLeg{
			{InstrumentName: "BTC-29APR22-37500-C", Amount: 1},
			{InstrumentName: "BTC-29APR22-37500-P", Amount: -1},
		},
	}
}

func TestCreateCombo(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCombo(t.Context(), nil)
	require.ErrorIs(t, err, errComboTradesRequired, "CreateCombo must reject an empty trade list")
	_, err = e.CreateCombo(t.Context(), []ComboTrade{{Amount: 1, Direction: DirectionBuy}})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateCombo must reject a trade without an instrument")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateCombo(t.Context(), testComboTrades(t))
	require.NoError(t, err, "CreateCombo must not error")
	if mockTests {
		assert.Equal(t, testCreatedCombo(), result, "CreateCombo should decode every field")
	} else {
		assert.Len(t, result.Legs, 2, "CreateCombo should return the combo's legs")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).CreateCombo(t.Context(), testComboTrades(t))
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateCombo should reject a null result")
	_, err = newUnavailableServerExchange(t).CreateCombo(t.Context(), testComboTrades(t))
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateCombo should return the request failure")
}

// testLegPricesRequest returns Deribit's documented leg prices example, or when testing live a spread of a listed
// future against the perpetual
func testLegPricesRequest(t *testing.T) *LegPricesRequest {
	t.Helper()
	if mockTests {
		return &LegPricesRequest{
			Legs: []ComboTrade{
				{InstrumentName: "BTC-1NOV24-67000-C", Amount: 2, Direction: DirectionBuy},
				{InstrumentName: "BTC-1NOV24-66000-C", Amount: 2, Direction: DirectionSell},
			},
			Price: 0.6,
		}
	}
	return &LegPricesRequest{Legs: testComboTrades(t), Price: 100}
}

// testLegPrices is Deribit's documented leg prices response
func testLegPrices() *LegPricesResponse {
	return &LegPricesResponse{
		Legs: []LegPrice{
			{
				InstrumentName: "BTC-1NOV24-67000-C",
				Direction:      "buy",
				Price:          0.6001,
				Ratio:          1,
			},
			{
				InstrumentName: "BTC-1NOV24-66000-C",
				Direction:      "sell",
				Price:          0.0001,
				Ratio:          1,
			},
		},
		Amount: 2,
	}
}

func TestGetLegPrices(t *testing.T) {
	t.Parallel()
	_, err := e.GetLegPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegPrices must reject a nil request")
	_, err = e.GetLegPrices(t.Context(), &LegPricesRequest{Price: 0.6})
	require.ErrorIs(t, err, errComboTradesRequired, "GetLegPrices must reject a request without legs")
	_, err = e.GetLegPrices(t.Context(), &LegPricesRequest{Legs: []ComboTrade{{InstrumentName: mockMarketDataOption, Direction: DirectionBuy}}, Price: 0.6})
	require.ErrorIs(t, err, errInvalidAmount, "GetLegPrices must reject a leg without an amount")

	skipLiveWithoutCredentials(t)
	result, err := e.GetLegPrices(t.Context(), testLegPricesRequest(t))
	require.NoError(t, err, "GetLegPrices must not error")
	if mockTests {
		assert.Equal(t, testLegPrices(), result, "GetLegPrices should decode every field")
	} else {
		assert.Len(t, result.Legs, 2, "GetLegPrices should return each leg's price")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetLegPrices(t.Context(), testLegPricesRequest(t))
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLegPrices should reject a null result")
	_, err = newUnavailableServerExchange(t).GetLegPrices(t.Context(), testLegPricesRequest(t))
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLegPrices should return the request failure")
}

func TestValidateComboTrades(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		trades []ComboTrade
		err    error
	}{
		{name: "no trades", err: errComboTradesRequired},
		{name: "a trade without an instrument", trades: []ComboTrade{{Amount: 1, Direction: DirectionBuy}}, err: errInstrumentNameRequired},
		{name: "a trade without an amount", trades: []ComboTrade{{InstrumentName: marketDataPerpetual, Direction: DirectionBuy}}, err: errInvalidAmount},
		{name: "a trade with a negative amount", trades: []ComboTrade{{InstrumentName: marketDataPerpetual, Amount: -10, Direction: DirectionSell}}, err: errInvalidAmount},
		{name: "a trade without a direction", trades: []ComboTrade{{InstrumentName: marketDataPerpetual, Amount: 10}}, err: errInvalidDirection},
		{name: "an invalid second trade", trades: []ComboTrade{{InstrumentName: marketDataPerpetual, Amount: 10, Direction: DirectionBuy}, {InstrumentName: marketDataPerpetual, Amount: 10, Direction: "long"}}, err: errInvalidDirection},
		{name: "valid trades", trades: []ComboTrade{{InstrumentName: mockMarketDataOption, Amount: 1, Direction: DirectionBuy}, {InstrumentName: marketDataPerpetual, Amount: 10, Direction: DirectionSell}}},
	} {
		assert.ErrorIsf(t, validateComboTrades(tc.trades), tc.err, "validateComboTrades should check %s", tc.name)
	}
}

// testFutureComboDetails is the mock combo of mockMarketDataFutureCombo
func testFutureComboDetails() *Combo {
	return &Combo{
		ID:                "BTC-FS-27NOV26_PERP",
		InstrumentID:      683926,
		State:             "active",
		StateTimestamp:    marketDataMilli(1791884521419),
		CreationTimestamp: marketDataMilli(1787818001000),
		Legs: []ComboLeg{
			{InstrumentName: "BTC-PERPETUAL", Amount: -1},
			{InstrumentName: "BTC-27NOV26", Amount: 1},
		},
	}
}

func TestGetComboDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errComboIDRequired, "GetComboDetails must reject an empty combo ID")

	comboID := marketDataFutureCombo(t)
	result, err := e.GetComboDetails(t.Context(), comboID)
	require.NoError(t, err, "GetComboDetails must not error")
	if mockTests {
		assert.Equal(t, testFutureComboDetails(), result, "GetComboDetails should decode every field")
	} else {
		assert.Equal(t, comboID, result.ID, "GetComboDetails should return the combo requested")
		assert.Len(t, result.Legs, 2, "GetComboDetails should return a future spread's legs")
	}

	_, err = e.GetComboDetails(t.Context(), testUnknownCombo)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "GetComboDetails must return Deribit's error for an unknown combo")
	if mockTests {
		exp := &APIError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"param":"combo_id","reason":"combo not found"}`)}
		assert.Equal(t, exp, apiErr, "GetComboDetails should decode Deribit's error")
	} else {
		assert.Equal(t, int64(-32602), apiErr.Code, "GetComboDetails should return Deribit's invalid params error")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetComboDetails should reject a null result")
	_, err = newUnavailableServerExchange(t).GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetComboDetails should return the request failure")
}

// testActiveComboIDs are the mock IDs of BTC's active combos
func testActiveComboIDs() []string {
	return []string{"BTC-CCAL-23OCT26_16OCT26-78000", "BTC-CCAL-30OCT26_16OCT26-82000", "BTC-CDIAG-25DEC26_30OCT26-80000_76000", "BTC-CS-16OCT26-80000_83000", "BTC-CS-23OCT26-79000_82000"}
}

func TestGetComboIDs(t *testing.T) {
	t.Parallel()
	_, err := e.GetComboIDs(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetComboIDs must reject a nil request")
	_, err = e.GetComboIDs(t.Context(), &ComboIDsRequest{State: ComboStateActive})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetComboIDs must reject an empty currency")
	_, err = e.GetComboIDs(t.Context(), &ComboIDsRequest{Currency: currency.BTC, State: "closed"})
	require.ErrorIs(t, err, errInvalidComboState, "GetComboIDs must reject a state public clients cannot select")

	arg := &ComboIDsRequest{Currency: currency.BTC, State: ComboStateActive}
	result, err := e.GetComboIDs(t.Context(), arg)
	require.NoError(t, err, "GetComboIDs must not error")
	if mockTests {
		assert.Equal(t, testActiveComboIDs(), result, "GetComboIDs should decode every ID")
	} else {
		assert.NotEmpty(t, result, "GetComboIDs should return BTC's active combos")
	}

	_, err = newUnavailableServerExchange(t).GetComboIDs(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetComboIDs should return the request failure")
}

// testCombos are the mock active combos of BTC
func testCombos() []Combo {
	return []Combo{
		{
			ID:                "BTC-CCAL-23OCT26_16OCT26-78000",
			InstrumentID:      715532,
			State:             "active",
			StateTimestamp:    marketDataMilli(1792105100094),
			CreationTimestamp: marketDataMilli(1792105100000),
			Legs: []ComboLeg{
				{InstrumentName: "BTC-16OCT26-78000-C", Amount: -1},
				{InstrumentName: "BTC-23OCT26-78000-C", Amount: 1},
			},
		},
		{
			ID:                "BTC-CCAL-30OCT26_16OCT26-82000",
			InstrumentID:      714741,
			State:             "active",
			StateTimestamp:    marketDataMilli(1792102467402),
			CreationTimestamp: marketDataMilli(1791989628000),
			Legs: []ComboLeg{
				{InstrumentName: "BTC-16OCT26-82000-C", Amount: -1},
				{InstrumentName: "BTC-30OCT26-82000-C", Amount: 1},
			},
		},
		{
			ID:                "BTC-CDIAG-25DEC26_30OCT26-80000_76000",
			InstrumentID:      715531,
			State:             "active",
			StateTimestamp:    marketDataMilli(1792104416652),
			CreationTimestamp: marketDataMilli(1792104416000),
			Legs: []ComboLeg{
				{InstrumentName: "BTC-30OCT26-76000-C", Amount: -1},
				{InstrumentName: "BTC-25DEC26-80000-C", Amount: 1},
			},
		},
		{
			ID:                "BTC-CS-16OCT26-80000_83000",
			InstrumentID:      713334,
			State:             "active",
			StateTimestamp:    marketDataMilli(1792107189850),
			CreationTimestamp: marketDataMilli(1791772323000),
			Legs: []ComboLeg{
				{InstrumentName: "BTC-16OCT26-80000-C", Amount: 1},
				{InstrumentName: "BTC-16OCT26-83000-C", Amount: -1},
			},
		},
		{
			ID:                "BTC-CS-23OCT26-79000_82000",
			InstrumentID:      715538,
			State:             "active",
			StateTimestamp:    marketDataMilli(1792107966706),
			CreationTimestamp: marketDataMilli(1792107966000),
			Legs: []ComboLeg{
				{InstrumentName: "BTC-23OCT26-79000-C", Amount: 1},
				{InstrumentName: "BTC-23OCT26-82000-C", Amount: -1},
			},
		},
	}
}

func TestGetCombos(t *testing.T) {
	t.Parallel()
	_, err := e.GetCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCombos must reject an empty currency")

	result, err := e.GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCombos must not error")
	if mockTests {
		assert.Equal(t, testCombos(), result, "GetCombos should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCombos should return BTC's active combos")
		result, err = e.GetCombos(t.Context(), currency.ANY)
		require.NoError(t, err, "GetCombos must not error for every currency")
		assert.NotEmpty(t, result, "GetCombos should return every currency's active combos")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"jsonrpc":"2.0","method":"public/get_combos","params":{"currency":"any"}}`, string(body), "GetCombos should send currency.ANY in lowercase, as Deribit rejects ANY")
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":[]}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetCombos(t.Context(), currency.ANY)
	require.NoError(t, err, "GetCombos must not error for every currency")
	_, err = newUnavailableServerExchange(t).GetCombos(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCombos should return the request failure")
}
