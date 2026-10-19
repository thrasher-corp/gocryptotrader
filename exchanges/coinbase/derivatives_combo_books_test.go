package coinbase

import (
	"context"
	"errors"
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

// testComboBooksUnknownCombo is a combo the gateway has never listed
const testComboBooksUnknownCombo = "BTC-FS-01JAN20_PERP"

// listedComboBooksDatedFuture returns the first open BTC future, other than the perpetual, that expires at least a day
// from now, which live tests spread against the perpetual
var listedComboBooksDatedFuture = sync.OnceValues(func() (string, error) {
	instruments, err := e.GlobalDerivatives().GetInstruments(context.Background(), &DerivativesInstrumentsRequest{Currency: currency.BTC, Kind: DerivativesKindFuture})
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

// mockComboBooksTrades are the trades of the documented create combo example
func mockComboBooksTrades() []DerivativesComboTrade {
	return []DerivativesComboTrade{
		{InstrumentName: "BTC-29APR22-37500-C", Amount: 1, Direction: DerivativesDirectionBuy},
		{InstrumentName: "BTC-29APR22-37500-P", Amount: 1, Direction: DerivativesDirectionSell},
	}
}

// testComboBooksTrades returns the documented create combo example's trades, or when testing live a spread of a listed
// future against the inverse perpetual, which the gateway lists as a combo already
func testComboBooksTrades(t *testing.T) []DerivativesComboTrade {
	t.Helper()
	if mockTests {
		return mockComboBooksTrades()
	}
	future, err := listedComboBooksDatedFuture()
	require.NoError(t, err, "listedComboBooksDatedFuture must not error")
	return []DerivativesComboTrade{
		{InstrumentName: future, Amount: 10, Direction: DerivativesDirectionBuy},
		{InstrumentName: testMarketDataInversePerpetual, Amount: 10, Direction: DerivativesDirectionSell},
	}
}

// mockComboBooksCreatedCombo is the documented create combo response
func mockComboBooksCreatedCombo() *DerivativesCombo {
	return &DerivativesCombo{
		ID:                "BTC-REV-29APR22-37500",
		InstrumentID:      52,
		State:             "active",
		StateTimestamp:    testMarketDataMilli(1650960943922),
		CreationTimestamp: testMarketDataMilli(1650960943000),
		Legs: []DerivativesComboLeg{
			{InstrumentName: "BTC-29APR22-37500-C", Amount: 1},
			{InstrumentName: "BTC-29APR22-37500-P", Amount: -1},
		},
	}
}

func TestGlobalDerivativesCreateCombo(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().CreateCombo(t.Context(), nil)
	require.ErrorIs(t, err, errDerivativesComboTradesRequired, "CreateCombo must require trades")
	_, err = e.GlobalDerivatives().CreateCombo(t.Context(), []DerivativesComboTrade{{Amount: 1, Direction: DerivativesDirectionBuy}})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateCombo must require each trade's instrument")

	skipLiveUnlessCanManipulate(t)
	resp, err := e.GlobalDerivatives().CreateCombo(t.Context(), testComboBooksTrades(t))
	require.NoError(t, err, "CreateCombo must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksCreatedCombo(), resp, "CreateCombo should decode every field")
	} else {
		assert.Len(t, resp.Legs, 2, "CreateCombo should return the combo's legs")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().CreateCombo(t.Context(), mockComboBooksTrades())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateCombo should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().CreateCombo(t.Context(), mockComboBooksTrades())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateCombo should return the request failure")
}

// mockComboBooksLegPricesRequest is the documented leg prices example
func mockComboBooksLegPricesRequest() *DerivativesLegPricesRequest {
	return &DerivativesLegPricesRequest{
		Legs: []DerivativesComboTrade{
			{InstrumentName: "BTC-1NOV24-67000-C", Amount: 2, Direction: DerivativesDirectionBuy},
			{InstrumentName: "BTC-1NOV24-66000-C", Amount: 2, Direction: DerivativesDirectionSell},
		},
		Price: 0.6,
	}
}

// testComboBooksLegPricesRequest returns the documented leg prices example, or when testing live a spread of a listed
// future against the inverse perpetual
func testComboBooksLegPricesRequest(t *testing.T) *DerivativesLegPricesRequest {
	t.Helper()
	if mockTests {
		return mockComboBooksLegPricesRequest()
	}
	return &DerivativesLegPricesRequest{Legs: testComboBooksTrades(t), Price: 100}
}

// mockComboBooksLegPrices is the documented leg prices response
func mockComboBooksLegPrices() *DerivativesLegPricesResponse {
	return &DerivativesLegPricesResponse{
		Legs: []DerivativesLegPrice{
			{InstrumentName: "BTC-1NOV24-67000-C", Direction: "buy", Price: 0.6001, Ratio: 1},
			{InstrumentName: "BTC-1NOV24-66000-C", Direction: "sell", Price: 0.0001, Ratio: 1},
		},
		Amount: 2,
	}
}

func TestGlobalDerivativesGetLegPrices(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetLegPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLegPrices must reject a nil request")
	_, err = e.GlobalDerivatives().GetLegPrices(t.Context(), &DerivativesLegPricesRequest{Price: 0.6})
	require.ErrorIs(t, err, errDerivativesComboTradesRequired, "GetLegPrices must require legs")
	_, err = e.GlobalDerivatives().GetLegPrices(t.Context(), &DerivativesLegPricesRequest{Legs: []DerivativesComboTrade{{InstrumentName: mockMarketDataOption, Direction: DerivativesDirectionBuy}}, Price: 0.6})
	require.ErrorIs(t, err, errInvalidAmount, "GetLegPrices must require each leg's amount")

	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetLegPrices(t.Context(), testComboBooksLegPricesRequest(t))
	require.NoError(t, err, "GetLegPrices must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksLegPrices(), resp, "GetLegPrices should decode every field")
	} else {
		assert.Len(t, resp.Legs, 2, "GetLegPrices should return each leg's price")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.JSONEq(t, `{"legs":[{"instrument_name":"BTC-FS-27NOV26_PERP","amount":1,"direction":"sell"}],"price":0}`, string(req.Params), "GetLegPrices should send a zero price")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"amount":1,"legs":[]}}`)
	}))
	_, err = ex.GlobalDerivatives().GetLegPrices(t.Context(), &DerivativesLegPricesRequest{Legs: []DerivativesComboTrade{{InstrumentName: mockMarketDataFutureCombo, Amount: 1, Direction: DerivativesDirectionSell}}})
	require.NoError(t, err, "GetLegPrices must not error for a zero price")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetLegPrices(t.Context(), mockComboBooksLegPricesRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLegPrices should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetLegPrices(t.Context(), mockComboBooksLegPricesRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLegPrices should return the request failure")
}

func TestValidateDerivativesComboTrades(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		trades []DerivativesComboTrade
		err    error
	}{
		{name: "no trades", err: errDerivativesComboTradesRequired},
		{name: "a trade without an instrument", trades: []DerivativesComboTrade{{Amount: 1, Direction: DerivativesDirectionBuy}}, err: errInstrumentNameRequired},
		{name: "a trade without an amount", trades: []DerivativesComboTrade{{InstrumentName: testMarketDataInversePerpetual, Direction: DerivativesDirectionBuy}}, err: errInvalidAmount},
		{name: "a trade with a negative amount", trades: []DerivativesComboTrade{{InstrumentName: testMarketDataInversePerpetual, Amount: -10, Direction: DerivativesDirectionSell}}, err: errInvalidAmount},
		{name: "a trade without a direction", trades: []DerivativesComboTrade{{InstrumentName: testMarketDataInversePerpetual, Amount: 10}}, err: errDerivativesInvalidComboDirection},
		{name: "an invalid second trade", trades: []DerivativesComboTrade{{InstrumentName: testMarketDataInversePerpetual, Amount: 10, Direction: DerivativesDirectionBuy}, {InstrumentName: mockMarketDataOption, Amount: 1, Direction: "long"}}, err: errDerivativesInvalidComboDirection},
		{name: "valid trades", trades: mockComboBooksTrades()},
	} {
		assert.ErrorIsf(t, validateDerivativesComboTrades(tc.trades), tc.err, "validateDerivativesComboTrades should check %s", tc.name)
	}
}

// mockComboBooksFutureComboDetails is the mock combo of mockMarketDataFutureCombo
func mockComboBooksFutureComboDetails() *DerivativesCombo {
	return &DerivativesCombo{
		ID:                "BTC-FS-27NOV26_PERP",
		InstrumentID:      683926,
		State:             "active",
		StateTimestamp:    testMarketDataMilli(1791884521419),
		CreationTimestamp: testMarketDataMilli(1787818001000),
		Legs: []DerivativesComboLeg{
			{InstrumentName: "BTC-PERPETUAL", Amount: -1},
			{InstrumentName: "BTC-27NOV26", Amount: 1},
		},
	}
}

func TestGlobalDerivativesGetComboDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetComboDetails(t.Context(), "")
	require.ErrorIs(t, err, errDerivativesComboIDRequired, "GetComboDetails must require a combo ID")

	comboID := marketDataFutureCombo(t)
	resp, err := e.GlobalDerivatives().GetComboDetails(t.Context(), comboID)
	require.NoError(t, err, "GetComboDetails must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksFutureComboDetails(), resp, "GetComboDetails should decode every field")
	} else {
		assert.Equal(t, comboID, resp.ID, "GetComboDetails should return the combo requested")
		assert.Len(t, resp.Legs, 2, "GetComboDetails should return a future spread's legs")
	}

	_, err = e.GlobalDerivatives().GetComboDetails(t.Context(), testComboBooksUnknownCombo)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "GetComboDetails must return the gateway's error for an unknown combo")
	if mockTests {
		exp := &DerivativesError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"reason":"combo not found","param":"combo_id"}`)}
		assert.Equal(t, exp, derivativesErr, "GetComboDetails should decode the gateway's error")
	} else {
		assert.Equal(t, int64(-32602), derivativesErr.Code, "GetComboDetails should return the gateway's invalid params error")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetComboDetails should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetComboDetails should return the request failure")
}

// mockComboBooksActiveComboIDs are the first mock IDs of BTC's active combos
func mockComboBooksActiveComboIDs() []string {
	return []string{"BTC-CDIAG-25JUN27_26MAR27-82000_78000", "BTC-CS-27NOV26-85000_90000", "BTC-CS-30OCT26-84000_88000", "BTC-FS-19OCT26_PERP", "BTC-FS-20OCT26_19OCT26"}
}

func TestGlobalDerivativesGetComboIDs(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetComboIDs(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetComboIDs must reject a nil request")
	_, err = e.GlobalDerivatives().GetComboIDs(t.Context(), &DerivativesComboIDsRequest{State: DerivativesComboStateActive})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetComboIDs must require a currency")
	_, err = e.GlobalDerivatives().GetComboIDs(t.Context(), &DerivativesComboIDsRequest{Currency: currency.BTC, State: "closed"})
	require.ErrorIs(t, err, errDerivativesInvalidComboState, "GetComboIDs must refuse an unknown state")

	arg := &DerivativesComboIDsRequest{Currency: currency.BTC, State: DerivativesComboStateActive}
	resp, err := e.GlobalDerivatives().GetComboIDs(t.Context(), arg)
	require.NoError(t, err, "GetComboIDs must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksActiveComboIDs(), resp, "GetComboIDs should decode every ID")
	} else {
		assert.NotEmpty(t, resp, "GetComboIDs should return BTC's active combos")
	}

	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetComboIDs(t.Context(), arg)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetComboIDs should return the request failure")
}

// mockComboBooksCombos are the first mock active combos of BTC
func mockComboBooksCombos() []DerivativesCombo {
	return []DerivativesCombo{
		{
			ID:                "BTC-CDIAG-25JUN27_26MAR27-82000_78000",
			InstrumentID:      715448,
			State:             "active",
			StateTimestamp:    testMarketDataMilli(1792366888541),
			CreationTimestamp: testMarketDataMilli(1792061674000),
			Legs: []DerivativesComboLeg{
				{InstrumentName: "BTC-26MAR27-78000-C", Amount: -1},
				{InstrumentName: "BTC-25JUN27-82000-C", Amount: 1},
			},
		},
		{
			ID:                "BTC-CS-27NOV26-85000_90000",
			InstrumentID:      710021,
			State:             "active",
			StateTimestamp:    testMarketDataMilli(1792368321001),
			CreationTimestamp: testMarketDataMilli(1791385864000),
			Legs: []DerivativesComboLeg{
				{InstrumentName: "BTC-27NOV26-85000-C", Amount: 1},
				{InstrumentName: "BTC-27NOV26-90000-C", Amount: -1},
			},
		},
		{
			ID:                "BTC-CS-30OCT26-84000_88000",
			InstrumentID:      705363,
			State:             "active",
			StateTimestamp:    testMarketDataMilli(1792368319148),
			CreationTimestamp: testMarketDataMilli(1790720975000),
			Legs: []DerivativesComboLeg{
				{InstrumentName: "BTC-30OCT26-84000-C", Amount: 1},
				{InstrumentName: "BTC-30OCT26-88000-C", Amount: -1},
			},
		},
		{
			ID:                "BTC-FS-19OCT26_PERP",
			InstrumentID:      715377,
			State:             "active",
			StateTimestamp:    testMarketDataMilli(1792051602745),
			CreationTimestamp: testMarketDataMilli(1792051602000),
			Legs: []DerivativesComboLeg{
				{InstrumentName: "BTC-PERPETUAL", Amount: -1},
				{InstrumentName: "BTC-19OCT26", Amount: 1},
			},
		},
		{
			ID:                "BTC-FS-20OCT26_19OCT26",
			InstrumentID:      716023,
			State:             "active",
			StateTimestamp:    testMarketDataMilli(1792137999404),
			CreationTimestamp: testMarketDataMilli(1792137999000),
			Legs: []DerivativesComboLeg{
				{InstrumentName: "BTC-19OCT26", Amount: -1},
				{InstrumentName: "BTC-20OCT26", Amount: 1},
			},
		},
	}
}

func TestGlobalDerivativesGetCombos(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().GetCombos(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCombos must require a currency")

	resp, err := e.GlobalDerivatives().GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err, "GetCombos must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksCombos(), resp, "GetCombos should decode every field")
	} else {
		assert.NotEmpty(t, resp, "GetCombos should return BTC's active combos")
		resp, err = e.GlobalDerivatives().GetCombos(t.Context(), currency.ANY)
		require.NoError(t, err, "GetCombos must not error for every currency")
		assert.NotEmpty(t, resp, "GetCombos should return every currency's active combos")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.JSONEq(t, `{"currency":"any"}`, string(req.Params), "GetCombos should send any in lower case, as the gateway refuses ANY")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":[]}`)
	}))
	resp, err = ex.GlobalDerivatives().GetCombos(t.Context(), currency.ANY)
	require.NoError(t, err, "GetCombos must not error for every currency")
	assert.Empty(t, resp, "GetCombos should decode an empty list")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetCombos(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCombos should return the request failure")
}
