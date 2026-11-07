package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIChangeMarginModel(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := testDerivativesChangeMarginModelRequest()
	if !mockTests {
		arg = liveDerivativesChangeMarginModelRequest()
	}
	resp, err := ex.WebsocketAPI().ChangeMarginModel(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.ChangeMarginModel must not error")
	if mockTests {
		assert.Equal(t, testDerivativesMarginModelChanges(), resp, "WebsocketAPI.ChangeMarginModel should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.ChangeMarginModel should return the change's effect on each currency")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().ChangeMarginModel(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.ChangeMarginModel should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetAccessLog(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{Count: testDerivativesAccessLogCount})
	require.NoError(t, err, "WebsocketAPI.GetAccessLog must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAccessLog(), resp, "WebsocketAPI.GetAccessLog should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Entries), testDerivativesAccessLogCount, "WebsocketAPI.GetAccessLog should return at most the entries asked for")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetAccessLog(t.Context(), &DerivativesGetAccessLogRequest{})
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetAccessLog should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetAccountSummaries(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetAccountSummaries(t.Context(), &DerivativesGetAccountSummariesRequest{Extended: true, IncludeIsolated: true})
	require.NoError(t, err, "WebsocketAPI.GetAccountSummaries must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAccountSummaries(), resp, "WebsocketAPI.GetAccountSummaries should decode every field")
	} else {
		assert.NotEmpty(t, resp.Summaries, "WebsocketAPI.GetAccountSummaries should return a summary for each currency")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetAccountSummaries(t.Context(), &DerivativesGetAccountSummariesRequest{})
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetAccountSummaries should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetAccountSummary(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesGetAccountSummaryRequest{Currency: currency.BTC, Extended: true}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := ex.WebsocketAPI().GetAccountSummary(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetAccountSummary must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAccountSummary(), resp, "WebsocketAPI.GetAccountSummary should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(currency.USDC), "WebsocketAPI.GetAccountSummary should return the currency asked for")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetAccountSummary(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetAccountSummary should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetLeverage(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesGetLeverageRequest{InstrumentName: testDerivativesAccountInstrument}
	if !mockTests {
		arg.InstrumentName = "BTC_USDC-PERPETUAL"
	}
	resp, err := ex.WebsocketAPI().GetLeverage(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetLeverage must not error")
	if mockTests {
		assert.Equal(t, testDerivativesLeverages(), resp, "WebsocketAPI.GetLeverage should decode every field")
	} else {
		assert.Len(t, resp, 1, "WebsocketAPI.GetLeverage should return the instrument's leverage")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetLeverage(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetLeverage should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetPMEParams(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	ccy := currency.BTC
	if !mockTests {
		ccy = currency.USDC
	}
	resp, err := ex.WebsocketAPI().GetPMEParams(t.Context(), ccy)
	require.NoError(t, err, "WebsocketAPI.GetPMEParams must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPMEParams(), resp, "WebsocketAPI.GetPMEParams should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetPMEParams(t.Context(), ccy)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetPMEParams should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetPosition(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesGetPositionRequest{InstrumentName: testDerivativesAccountInstrument}
	if !mockTests {
		arg.InstrumentName = "BTC_USDC-PERPETUAL"
	}
	resp, err := ex.WebsocketAPI().GetPosition(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetPosition must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPosition(), resp, "WebsocketAPI.GetPosition should decode every field")
	} else {
		assert.Equal(t, arg.InstrumentName, resp.InstrumentName, "WebsocketAPI.GetPosition should return the instrument's position")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetPosition(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetPosition should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetPositions(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesGetPositionsRequest{Currency: currency.NewCode(DerivativesCurrencyAny), IncludeIsolated: true}
	resp, err := ex.WebsocketAPI().GetPositions(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetPositions must not error")
	if mockTests {
		assert.Equal(t, testDerivativesIsolatedPositions(), resp, "WebsocketAPI.GetPositions should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetPositions(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetPositions should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetRiskProfile(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetRiskProfile(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetRiskProfile must not error")
	if mockTests {
		exp := &DerivativesRiskProfileResponse{
			PortfolioMarginingAllowed: true,
			ShortOptionAllowed:        true,
		}
		assert.Equal(t, exp, resp, "WebsocketAPI.GetRiskProfile should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetRiskProfile(t.Context())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetRiskProfile should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetSubaccounts(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetSubaccounts(t.Context(), &DerivativesGetSubaccountsRequest{WithPortfolio: true})
	require.NoError(t, err, "WebsocketAPI.GetSubaccounts must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSubaccounts(), resp, "WebsocketAPI.GetSubaccounts should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetSubaccounts should return the main account")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetSubaccounts(t.Context(), &DerivativesGetSubaccountsRequest{})
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetSubaccounts should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetSubaccountsDetails(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesGetSubaccountsDetailsRequest{Currency: currency.BTC, WithOpenOrders: true}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := ex.WebsocketAPI().GetSubaccountsDetails(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetSubaccountsDetails must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSubaccountsDetails(), resp, "WebsocketAPI.GetSubaccountsDetails should decode every field")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetSubaccountsDetails(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetSubaccountsDetails should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetTradingLimits(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	ccy := currency.BTC
	if !mockTests {
		ccy = currency.USDC
	}
	resp, err := ex.WebsocketAPI().GetTradingLimits(t.Context(), ccy)
	require.NoError(t, err, "WebsocketAPI.GetTradingLimits must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTradingLimits(), resp, "WebsocketAPI.GetTradingLimits should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(ccy), "WebsocketAPI.GetTradingLimits should return the currency's limits")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetTradingLimits(t.Context(), ccy)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetTradingLimits should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetTransactionLog(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := testDerivativesTransactionLogRequest()
	if !mockTests {
		arg = liveDerivativesTransactionLogRequest()
	}
	resp, err := ex.WebsocketAPI().GetTransactionLog(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetTransactionLog must not error")
	if mockTests {
		assert.Equal(t, testDerivativesTransactionLog(), resp, "WebsocketAPI.GetTransactionLog should decode every field")
	} else {
		assert.LessOrEqual(t, len(resp.Logs), testDerivativesTransactionLogCount, "WebsocketAPI.GetTransactionLog should return at most the entries asked for")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetTransactionLog(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetTransactionLog should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetUserLocks(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetUserLocks(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetUserLocks must not error")
	if mockTests {
		assert.Equal(t, testDerivativesUserLocks(), resp, "WebsocketAPI.GetUserLocks should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetUserLocks should return each currency's lock")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetUserLocks(t.Context())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetUserLocks should refuse an unauthenticated connection")
}

func TestWebsocketAPIPMESimulate(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesSimulatePortfolioRequest{Currency: currency.BTC}
	if !mockTests {
		arg.Currency = currency.USDC
	}
	resp, err := ex.WebsocketAPI().PMESimulate(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.PMESimulate must not error")
	if mockTests {
		assert.Equal(t, testDerivativesPMESimulation(), resp, "WebsocketAPI.PMESimulate should decode every field")
	} else {
		assert.NotEmpty(t, resp.Margins, "WebsocketAPI.PMESimulate should return the simulated margins")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().PMESimulate(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.PMESimulate should refuse an unauthenticated connection")
}

func TestWebsocketAPISimulatePortfolio(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := testDerivativesSimulatePortfolioRequest()
	if !mockTests {
		arg = &DerivativesSimulatePortfolioRequest{Currency: currency.USDC, SimulatedPositions: map[string]float64{"BTC_USDC-PERPETUAL": 0.0001}}
	}
	resp, err := ex.WebsocketAPI().SimulatePortfolio(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.SimulatePortfolio must not error")
	if mockTests {
		assert.Equal(t, testDerivativesSimulatedPortfolio(), resp, "WebsocketAPI.SimulatePortfolio should decode every field")
	} else {
		assert.True(t, resp.Currency.Equal(currency.USDC), "WebsocketAPI.SimulatePortfolio should return the simulated currency")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().SimulatePortfolio(t.Context(), arg)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.SimulatePortfolio should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetAnnouncements(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetAnnouncements(t.Context(), testDerivativesAnnouncementsRequest())
	require.NoError(t, err, "WebsocketAPI.GetAnnouncements must not error")
	if mockTests {
		assert.Equal(t, testDerivativesAnnouncements(), resp, "WebsocketAPI.GetAnnouncements should decode every field")
	} else {
		require.Len(t, resp, testDerivativesAnnouncementsCount, "WebsocketAPI.GetAnnouncements must return the announcements asked for")
		assert.False(t, resp[0].PublicationTimestamp.Time().After(time.UnixMilli(testDerivativesAnnouncementsStart)), "WebsocketAPI.GetAnnouncements should return announcements published up to the start time")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetAnnouncements(t.Context(), testDerivativesAnnouncementsRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetAnnouncements should error without a websocket connection")
}
