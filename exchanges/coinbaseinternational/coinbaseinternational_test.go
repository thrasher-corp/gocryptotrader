package coinbaseinternational

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var (
	co      = &CoinbaseInternational{}
	btcPerp = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.PERP}
	spotTP  = currency.NewPairWithDelimiter("BTC", "USDC", currency.DashDelimiter)
)

func TestMain(m *testing.M) {
	co = new(CoinbaseInternational)
	if err := testexch.Setup(co); err != nil {
		log.Fatal(err)
	}

	co.Enabled = true
	if apiKey != "" && apiSecret != "" {
		co.API.AuthenticatedSupport = true
		co.API.AuthenticatedWebsocketSupport = true
		co.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")
		co.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	co.Websocket = sharedtestvalues.NewTestWebsocket()
	if err := co.UpdateTradablePairs(context.Background(), true); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	result, err := co.ListAssets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	result, err := co.GetAssetDetails(t.Context(), currency.EMPTYCODE, "", "207597618027560960")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	co.Verbose = true
	result, err := co.GetSupportedNetworksPerAsset(t.Context(), currency.USDC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexComposition(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexComposition(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexComposition(t.Context(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCompositionHistory(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexCompositionHistory(t.Context(), "", time.Time{}, 0, 100)
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexCompositionHistory(t.Context(), "COIN50", time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexPrice(t.Context(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCandles(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexCandles(t.Context(), "", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errIndexNameRequired)
	_, err = co.GetIndexCandles(t.Context(), "COIN50", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errGranularityRequired)
	_, err = co.GetIndexCandles(t.Context(), "COIN50", "ONE_DAY", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexCandles(t.Context(), "COIN50", "ONE_DAY", time.Now().Add(-time.Hour*50), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := co.GetInstruments(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(t.Context(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetInstrumentDetails(t.Context(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(t.Context(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetQuotePerInstrument(t.Context(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyTradingVolumes(t *testing.T) {
	t.Parallel()
	_, err := co.GetDailyTradingVolumes(t.Context(), []string{}, 10, 10, time.Now().Add(-time.Hour*100), true)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetDailyTradingVolumes(t.Context(), []string{"BTC-PERP"}, 10, 1, time.Now().Add(-time.Hour*100), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedCandlesDataPerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetAggregatedCandlesDataPerInstrument(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInstrumentIDRequired)
	_, err = co.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)
	_, err = co.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.TenMin, time.Now().Add(-time.Hour*100), time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := co.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.FifteenMin, time.Now().Add(-time.Hour*100), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStringFromInterval(t *testing.T) {
	t.Parallel()
	_, err := stringFromInterval(kline.HundredMilliseconds)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	intervalString, err := stringFromInterval(kline.FiveMin)
	require.NoError(t, err)
	assert.NotEmpty(t, intervalString)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := co.GetHistoricalFundingRate(t.Context(), "", 0, 10)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetHistoricalFundingRate(t.Context(), "BTC-PERP", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionOffsets(t *testing.T) {
	t.Parallel()
	result, err := co.GetPositionOffsets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	orderTypeMap := map[order.Type]struct {
		String string
		Error  error
	}{
		order.Limit:       {"LIMIT", nil},
		order.Market:      {"MARKET", nil},
		order.Stop:        {"STOP", nil},
		order.StopLimit:   {"STOP_LIMIT", nil},
		order.UnknownType: {"", order.ErrUnsupportedOrderType},
	}
	for k, v := range orderTypeMap {
		result, err := OrderTypeString(k)
		require.ErrorIs(t, err, v.Error)
		assert.Equal(t, result, v.String)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	orderType, err := OrderTypeString(order.Limit)
	require.NoError(t, err)
	_, err = co.CreateOrder(t.Context(), &OrderRequestParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderRequestParams{PostOnly: true}
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.BaseSize = 1
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 12345.67
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType = orderType
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123442"
	_, err = co.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateOrder(t.Context(), &OrderRequestParams{
		ClientOrderID: "123442",
		Side:          "BUY",
		BaseSize:      1,
		Instrument:    "BTC-PERP",
		OrderType:     orderType,
		Price:         12345.67,
		ExpireTime:    "",
		PostOnly:      true,
		TimeInForce:   "GTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOpenOrders(t.Context(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := co.CancelOrders(t.Context(), "", "", "")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelOrders(t.Context(), "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := co.ModifyOpenOrder(t.Context(), "1234", &ModifyOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = co.ModifyOpenOrder(t.Context(), "", &ModifyOrderParam{Portfolio: "1234"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ModifyOpenOrder(t.Context(), "1234", &ModifyOrderParam{
		Price:     1234,
		StopPrice: 1239,
		Size:      1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOrderDetail(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := co.CancelTradeOrder(t.Context(), "", "", "", "")
	require.ErrorIsf(t, err, order.ErrOrderIDNotSet, "expected %v, got %v", order.ErrOrderIDNotSet, err)
	_, err = co.CancelTradeOrder(t.Context(), "order-id", "", "", "")
	require.ErrorIsf(t, err, errMissingPortfolioID, "expected %v, got %v", errMissingPortfolioID, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelTradeOrder(t.Context(), "1234", "", "12344232", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAllUserPortfolios(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.CreatePortfolio(t.Context(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetUserPortfolio(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetUserPortfolio(t.Context(), "4thr7ft-1-0")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPatchPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.PatchPortfolio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.PatchPortfolio(t.Context(), &PatchPortfolioParams{AutoMarginEnabled: true, PortfolioName: "new-portfolio"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.UpdatePortfolio(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.UpdatePortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioDetails(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioDetails(t.Context(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioSummary(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioSummary(t.Context(), "", "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioBalances(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioBalances(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioAssetBalance(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = co.GetPortfolioAssetBalance(t.Context(), "", "", currency.BTC)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioAssetBalance(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveLoansForPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetActiveLoansForPortfolio(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetActiveLoansForPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLoanInfoForPortfolioAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetLoanInfoForPortfolioAsset(t.Context(), "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.GetLoanInfoForPortfolioAsset(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetLoanInfoForPortfolioAsset(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquireRepayLoan(t *testing.T) {
	t.Parallel()
	_, err := co.AcquireRepayLoan(t.Context(), "", "", currency.BTC, &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = co.AcquireRepayLoan(t.Context(), "", "", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.AcquireRepayLoan(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.AcquireRepayLoan(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPreviewLoanUpdate(t *testing.T) {
	t.Parallel()
	_, err := co.PreviewLoanUpdate(t.Context(), "", "", currency.BTC, &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = co.PreviewLoanUpdate(t.Context(), "", "", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.PreviewLoanUpdate(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.PreviewLoanUpdate(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestViewMaxLoanAvailability(t *testing.T) {
	t.Parallel()
	_, err := co.ViewMaxLoanAvailability(t.Context(), "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.ViewMaxLoanAvailability(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ViewMaxLoanAvailability(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioPositions(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioPositions(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioInstrumentPosition(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = co.GetPortfolioInstrumentPosition(t.Context(), "", "", btcPerp)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioInstrumentPosition(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", btcPerp)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTotalOpenPositionLimitPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetTotalOpenPositionLimitPortfolio(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetTotalOpenPositionLimitPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFillsByPortfolio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFillsByPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "abcdefg", 10, 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioFills(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioFills(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioCrossCollateral(t *testing.T) {
	t.Parallel()
	_, err := co.EnableDisablePortfolioCrossCollateral(t.Context(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.EnableDisablePortfolioCrossCollateral(t.Context(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioAutoMarginMode(t *testing.T) {
	t.Parallel()
	_, err := co.EnableDisablePortfolioAutoMarginMode(t.Context(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.EnableDisablePortfolioAutoMarginMode(t.Context(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPortfolioMarginOverride(t *testing.T) {
	t.Parallel()
	_, err := co.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverrideParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverrideParams{MarginOverride: .5, PortfolioID: "f67de785-60a7-45ea-b87"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferFundsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Asset: currency.BTC})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferPositionsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Instrument: "BTC-PERP", Quantity: 123, Side: "BUY"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioFeeRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioFeeRates(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYourRanking(t *testing.T) {
	t.Parallel()
	_, err := co.GetYourRanking(t.Context(), "", "THIS_MONTH", []string{})
	require.ErrorIs(t, err, errInstrumentTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetYourRanking(t.Context(), "SPOT", "THIS_MONTH", []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCounterPartyID(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCounterPartyID(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateCounterPartyID(t.Context(), "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := co.ValidateCounterpartyID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ValidateCounterpartyID(t.Context(), "CBTQDGENHE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{
		Portfolio:      "5189861793641175",
		CounterpartyID: "CBTQDGENHE",
		Asset:          "BTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCounterpartyWithdrawalLimit(t *testing.T) {
	t.Parallel()
	_, err := co.GetCounterpartyWithdrawalLimit(context.Background(), "", "291efb0f-2396-4d41-ad03-db3b2311cb2c")
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = co.GetCounterpartyWithdrawalLimit(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetCounterpartyWithdrawalLimit(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "291efb0f-2396-4d41-ad03-db3b2311cb2c")
	require.ErrorIs(t, err, nil)
	assert.NotNil(t, result)
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListMatchingTransfers(t.Context(), "", "", "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	_, err := co.GetTransfer(t.Context(), "")
	require.ErrorIs(t, err, errMissingTransferID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetTransfer(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCryptoAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &WithdrawCryptoParams{Nonce: "1234"}
	_, err = co.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errAddressIsRequired)

	arg.Address = "1234HGJHGHGHGJ"
	_, err = co.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = 1200
	_, err = co.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawToCryptoAddress(t.Context(), &WithdrawCryptoParams{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		Amount:          1200,
		Address:         "1234HGJHGHGHGJ",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCryptoAddress(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CryptoAddressParam{
		NetworkArnID: "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	}
	_, err = co.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errAssetIdentifierRequired)

	arg.AssetIdentifier = "291efb0f-2396-4d41-ad03-db3b2311cb2c"
	_, err = co.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errMissingPortfolioID)

	arg.Portfolio = "892e8c7c-e979-4cad-b61b-55a197932cf1"
	arg.NetworkArnID = ""
	_, err = co.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errNetworkArnID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateCryptoAddress(t.Context(), &CryptoAddressParam{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID:    "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := co.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := co.UpdateTradablePairs(t.Context(), true)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := co.UpdateTicker(t.Context(), btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := co.UpdateTickers(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = co.UpdateTickers(t.Context(), asset.Futures)
	assert.NoError(t, err)

	err = co.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGenerateSubscriptionPayload(t *testing.T) {
	t.Parallel()
	_, err := co.GenerateSubscriptionPayload(subscription.List{}, "SUBSCRIBE")
	require.ErrorIs(t, err, common.ErrEmptyParams)

	payload, err := co.GenerateSubscriptionPayload(subscription.List{
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlFunding, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlInstruments, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
		{Channel: cnlInstruments, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDC}}},
		{Channel: cnlMatch, Pairs: currency.Pairs{{Base: currency.BTC, Delimiter: "-", Quote: currency.USDT}}},
	}, "SUBSCRIBE")
	require.NoError(t, err)
	assert.Len(t, payload, 2)
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := co.FetchOrderbook(t.Context(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.FetchOrderbook(t.Context(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchOrderbook(t.Context(), btcPerp, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateOrderbook(t.Context(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.UpdateOrderbook(t.Context(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.UpdateOrderbook(t.Context(), btcPerp, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateAccountInfo(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.UpdateAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.UpdateAccountInfo(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.FetchAccountInfo(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.FetchAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchAccountInfo(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := co.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := co.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyTradeFee,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = co.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	result, err := co.GetAvailableTransferChains(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      co.Name,
		Pair:          btcPerp,
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         0.0001,
		Amount:        10,
		ClientID:      "newOrder",
		ClientOrderID: "my-new-order-id",
		AssetType:     asset.Spot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ModifyOrder(t.Context(), &order.Modify{
		Exchange:  "CoinbaseInternational",
		OrderID:   "1337",
		Price:     10000,
		Amount:    10,
		Side:      order.Sell,
		Pair:      btcPerp,
		AssetType: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	err := co.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  "CoinbaseInternational",
		AssetType: asset.Spot,
		Pair:      btcPerp,
		OrderID:   "1234",
		AccountID: "Someones SubAccount",
	})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := co.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelAllOrders(t.Context(), &order.Cancel{
		Exchange:  "CoinbaseInternational",
		AssetType: asset.Spot,
		AccountID: "Sub-account Samuael",
		Pair:      btcPerp,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOrderInfo(t.Context(), "12234", btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:    co.Name,
		Amount:      10,
		Currency:    currency.LTC,
		PortfolioID: "1234564",
		Crypto: withdraw.CryptoRequest{
			Chain:      "TON",
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := co.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err)

	pairs, err := co.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	for y := range pairs {
		lim, err := co.GetOrderExecutionLimits(asset.Spot, pairs[y])
		require.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		require.NotEmpty(t, lim, "limit cannot be empty")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := co.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = co.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	require.ErrorIs(t, err, asset.ErrNotSupported)

	pairs, err := co.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	resp, err := co.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFeeRateTiers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFeeRateTiers(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-PERP")
	require.NoError(t, err)

	result, err := co.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Pair:  cp,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = co.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)
	_, err := co.GetHistoricCandlesExtended(t.Context(), btcPerp, asset.Options, kline.FifteenMin, startTime, end)
	require.ErrorIs(t, err, asset.ErrNotEnabled)

	result, err := co.GetHistoricCandlesExtended(t.Context(), currency.NewPair(currency.BTC, currency.USDC), asset.Spot, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetHistoricCandlesExtended(t.Context(), btcPerp, asset.Futures, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := co.GetHistoricCandles(t.Context(), btcPerp, asset.Options, kline.FifteenMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, asset.ErrNotEnabled)

	co.Verbose = true
	result, err := co.GetHistoricCandles(t.Context(), currency.NewPair(currency.BTC, currency.USDC), asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetHistoricCandles(t.Context(), btcPerp, asset.Futures, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}
