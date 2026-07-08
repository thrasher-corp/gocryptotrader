package htx

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetLinearSwapMarkets(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetLinearSwapMarkets(t.Context(), btcusdtPair, "cross", "swap", "futures")
	require.Error(t, err, "GetLinearSwapMarkets must return transport error")
}

func TestGetLinearSwapMarketDepth(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetLinearSwapMarketDepth(t.Context(), btcusdtPair, "step0")
	require.Error(t, err, "GetLinearSwapMarketDepth must return transport error")
}

func TestGetLinearSwapMarketOverview(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetLinearSwapMarketOverview(t.Context(), btcusdtPair)
	require.Error(t, err, "GetLinearSwapMarketOverview must return transport error")
}

func TestGetLinearSwapFundingRate(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetLinearSwapFundingRate(t.Context(), btcusdtPair)
	require.Error(t, err, "GetLinearSwapFundingRate must return transport error")
}

func TestGetLinearSwapFundingRates(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetLinearSwapFundingRates(t.Context())
	require.Error(t, err, "GetLinearSwapFundingRates must return transport error")
}

func TestGetV5OpenInterest(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "http://127.0.0.1:1"), "USDT-margined endpoint must be set")
	_, err := h.GetV5OpenInterest(t.Context(), btcusdtPair)
	require.Error(t, err, "GetV5OpenInterest must return transport error")
}

func TestGetV5AccountBalance(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetV5AccountBalance(t.Context())
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetV5AccountBalance must return credentials error")
}

func TestPlaceV5Order(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.PlaceV5Order(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceV5Order must reject nil request")
	_, err = h.PlaceV5Order(t.Context(), &V5OrderRequest{ContractCode: "BTC-USDT", MarginMode: "cross", Side: "buy", Type: "limit", Volume: types.Number(1)})
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "PlaceV5Order must return credentials error")
}

func TestCancelV5Order(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.CancelV5Order(t.Context(), btcusdtPair, "1", "")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelV5Order must return credentials error")
}

func TestCancelAllV5Orders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.CancelAllV5Orders(t.Context(), btcusdtPair, "buy", "long")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelAllV5Orders must return credentials error")
}

func TestGetV5Order(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetV5Order(t.Context(), btcusdtPair, "cross", "1", "")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetV5Order must return credentials error")
}

func TestGetV5OpenOrders(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	_, err := h.GetV5OpenOrders(t.Context(), btcusdtPair, "cross", "", "", 1, 10, "next")
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetV5OpenOrders must return credentials error")
}
