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
	result, err := co.ListAssets(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	result, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	result, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.USDC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexComposition(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexComposition(context.Background(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexComposition(context.Background(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCompositionHistory(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexCompositionHistory(context.Background(), "", time.Time{}, 0, 100)
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexCompositionHistory(context.Background(), "COIN50", time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexPrice(context.Background(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexPrice(context.Background(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCandles(t *testing.T) {
	t.Parallel()
	_, err := co.GetIndexCandles(context.Background(), "", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errIndexNameRequired)
	_, err = co.GetIndexCandles(context.Background(), "COIN50", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errGranularityRequired)
	_, err = co.GetIndexCandles(context.Background(), "COIN50", "ONE_DAY", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetIndexCandles(context.Background(), "COIN50", "ONE_DAY", time.Now().Add(-time.Hour*50), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := co.GetInstruments(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(context.Background(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(context.Background(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyTradingVolumes(t *testing.T) {
	t.Parallel()
	_, err := co.GetDailyTradingVolumes(context.Background(), []string{}, 10, 10, time.Now().Add(-time.Hour*100), true)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetDailyTradingVolumes(context.Background(), []string{"BTC-PERP"}, 10, 1, time.Now().Add(-time.Hour*100), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedCandlesDataPerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetAggregatedCandlesDataPerInstrument(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInstrumentIDRequired)
	_, err = co.GetAggregatedCandlesDataPerInstrument(context.Background(), "BTC-PERP", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)
	_, err = co.GetAggregatedCandlesDataPerInstrument(context.Background(), "BTC-PERP", kline.TenMin, time.Now().Add(-time.Hour*100), time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := co.GetAggregatedCandlesDataPerInstrument(context.Background(), "BTC-PERP", kline.FifteenMin, time.Now().Add(-time.Hour*100), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := co.GetHistoricalFundingRate(context.Background(), "", 0, 10)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := co.GetHistoricalFundingRate(context.Background(), "BTC-PERP", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionOffsets(t *testing.T) {
	t.Parallel()
	result, err := co.GetPositionOffsets(context.Background())
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
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderRequestParams{PostOnly: true}
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.BaseSize = 1
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 12345.67
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType = orderType
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123442"
	_, err = co.CreateOrder(context.Background(), arg)
	require.ErrorIs(t, err, errTimeInForceRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateOrder(context.Background(), &OrderRequestParams{
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
	result, err := co.GetOpenOrders(context.Background(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := co.CancelOrders(context.Background(), "", "", "")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelOrders(context.Background(), "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = co.ModifyOpenOrder(context.Background(), "", &ModifyOrderParam{Portfolio: "1234"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ModifyOpenOrder(context.Background(), "1234", &ModifyOrderParam{
		Price:     1234,
		StopPrice: 1239,
		Size:      1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetOrderDetail(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetOrderDetail(context.Background(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := co.CancelTradeOrder(context.Background(), "", "", "", "")
	require.ErrorIsf(t, err, order.ErrOrderIDNotSet, "expected %v, got %v", order.ErrOrderIDNotSet, err)
	_, err = co.CancelTradeOrder(context.Background(), "order-id", "", "", "")
	require.ErrorIsf(t, err, errMissingPortfolioID, "expected %v, got %v", errMissingPortfolioID, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelTradeOrder(context.Background(), "1234", "", "12344232", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAllUserPortfolios(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.CreatePortfolio(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetUserPortfolio(context.Background(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetUserPortfolio(context.Background(), "4thr7ft-1-0")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPatchPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.PatchPortfolio(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.PatchPortfolio(context.Background(), &PatchPortfolioParams{AutoMarginEnabled: true, PortfolioName: "new-portfolio"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.UpdatePortfolio(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.UpdatePortfolio(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioDetails(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioDetails(context.Background(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioSummary(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioSummary(context.Background(), "", "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioBalances(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioBalances(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioAssetBalance(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = co.GetPortfolioAssetBalance(context.Background(), "", "", currency.BTC)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioAssetBalance(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveLoansForPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetActiveLoansForPortfolio(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetActiveLoansForPortfolio(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLoanInfoForPortfolioAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetLoanInfoForPortfolioAsset(context.Background(), "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.GetLoanInfoForPortfolioAsset(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetLoanInfoForPortfolioAsset(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquireRepayLoan(t *testing.T) {
	t.Parallel()
	_, err := co.AcquireRepayLoan(context.Background(), "", "", currency.BTC, &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = co.AcquireRepayLoan(context.Background(), "", "", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.AcquireRepayLoan(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.AcquireRepayLoan(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPreviewLoanUpdate(t *testing.T) {
	t.Parallel()
	_, err := co.PreviewLoanUpdate(context.Background(), "", "", currency.BTC, &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = co.PreviewLoanUpdate(context.Background(), "", "", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.PreviewLoanUpdate(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.PreviewLoanUpdate(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC, &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestViewMaxLoanAvailability(t *testing.T) {
	t.Parallel()
	_, err := co.ViewMaxLoanAvailability(context.Background(), "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = co.ViewMaxLoanAvailability(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ViewMaxLoanAvailability(context.Background(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioPositions(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioPositions(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioInstrumentPosition(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = co.GetPortfolioInstrumentPosition(context.Background(), "", "", btcPerp)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioInstrumentPosition(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", btcPerp)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTotalOpenPositionLimitPortfolio(t *testing.T) {
	t.Parallel()
	_, err := co.GetTotalOpenPositionLimitPortfolio(context.Background(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetTotalOpenPositionLimitPortfolio(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFillsByPortfolio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFillsByPortfolio(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "abcdefg", 10, 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	_, err := co.ListPortfolioFills(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListPortfolioFills(context.Background(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioCrossCollateral(t *testing.T) {
	t.Parallel()
	_, err := co.EnableDisablePortfolioCrossCollateral(context.Background(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.EnableDisablePortfolioCrossCollateral(context.Background(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioAutoMarginMode(t *testing.T) {
	t.Parallel()
	_, err := co.EnableDisablePortfolioAutoMarginMode(context.Background(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.EnableDisablePortfolioAutoMarginMode(context.Background(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPortfolioMarginOverride(t *testing.T) {
	t.Parallel()
	_, err := co.SetPortfolioMarginOverride(context.Background(), &PortfolioMarginOverrideParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.SetPortfolioMarginOverride(context.Background(), &PortfolioMarginOverrideParams{MarginOverride: .5, PortfolioID: "f67de785-60a7-45ea-b87"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferFundsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.TransferFundsBetweenPortfolios(context.Background(), &TransferFundsBetweenPortfoliosParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.TransferFundsBetweenPortfolios(context.Background(), &TransferFundsBetweenPortfoliosParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Asset: currency.BTC})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferPositionsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.TransferPositionsBetweenPortfolios(context.Background(), &TransferPortfolioParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.TransferPositionsBetweenPortfolios(context.Background(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Instrument: "BTC-PERP", Quantity: 123, Side: "BUY"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioFeeRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetPortfolioFeeRates(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYourRanking(t *testing.T) {
	t.Parallel()
	_, err := co.GetYourRanking(context.Background(), "", "THIS_MONTH", []string{})
	require.ErrorIs(t, err, errInstrumentTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetYourRanking(context.Background(), "SPOT", "THIS_MONTH", []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCounterPartyID(t *testing.T) {
	t.Parallel()
	_, err := co.CreateCounterPartyID(context.Background(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateCounterPartyID(context.Background(), "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := co.ValidateCounterpartyID(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.ValidateCounterpartyID(context.Background(), "CBTQDGENHE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCounterpartyID(context.Background(), &AssetCounterpartyWithdrawalResponse{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawToCounterpartyID(context.Background(), &AssetCounterpartyWithdrawalResponse{
		Portfolio:      "5189861793641175",
		CounterpartyID: "CBTQDGENHE",
		Asset:          "BTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.ListMatchingTransfers(context.Background(), "", "", "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	_, err := co.GetTransfer(context.Background(), "")
	require.ErrorIs(t, err, errMissingTransferID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetTransfer(context.Background(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := co.WithdrawToCryptoAddress(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &WithdrawCryptoParams{Nonce: "1234"}
	_, err = co.WithdrawToCryptoAddress(context.Background(), arg)
	require.ErrorIs(t, err, errAddressIsRequired)

	arg.Address = "1234HGJHGHGHGJ"
	_, err = co.WithdrawToCryptoAddress(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = 1200
	_, err = co.WithdrawToCryptoAddress(context.Background(), arg)
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawToCryptoAddress(context.Background(), &WithdrawCryptoParams{
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
	_, err := co.CreateCryptoAddress(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CryptoAddressParam{
		NetworkArnID: "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	}
	_, err = co.CreateCryptoAddress(context.Background(), arg)
	assert.ErrorIs(t, err, errAssetIdentifierRequired)

	arg.AssetIdentifier = "291efb0f-2396-4d41-ad03-db3b2311cb2c"
	_, err = co.CreateCryptoAddress(context.Background(), arg)
	assert.ErrorIs(t, err, errMissingPortfolioID)

	arg.Portfolio = "892e8c7c-e979-4cad-b61b-55a197932cf1"
	arg.NetworkArnID = ""
	_, err = co.CreateCryptoAddress(context.Background(), arg)
	assert.ErrorIs(t, err, errNetworkArnID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CreateCryptoAddress(context.Background(), &CryptoAddressParam{
		Portfolio:       "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetIdentifier: "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID:    "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := co.FetchTradablePairs(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchTradablePairs(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := co.UpdateTradablePairs(context.Background(), true)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := co.UpdateTicker(context.Background(), btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := co.UpdateTickers(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = co.UpdateTickers(context.Background(), asset.Futures)
	assert.NoError(t, err)

	err = co.UpdateTickers(context.Background(), asset.Spot)
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
	_, err := co.FetchOrderbook(context.Background(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.FetchOrderbook(context.Background(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchOrderbook(context.Background(), btcPerp, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateOrderbook(context.Background(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.UpdateOrderbook(context.Background(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.UpdateOrderbook(context.Background(), btcPerp, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.UpdateAccountInfo(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.UpdateAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.UpdateAccountInfo(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := co.FetchAccountInfo(context.Background(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.FetchAccountInfo(context.Background(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.FetchAccountInfo(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetAccountFundingHistory(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := co.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := co.GetFeeByType(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyTradeFee,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = co.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    btcPerp,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	result, err := co.GetAvailableTransferChains(context.Background(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.SubmitOrder(context.Background(), &order.Submit{
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
	result, err := co.ModifyOrder(context.Background(), &order.Modify{
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
	err := co.CancelOrder(context.Background(), &order.Cancel{
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
	_, err := co.CancelAllOrders(context.Background(), &order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.CancelAllOrders(context.Background(),
		&order.Cancel{
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
	result, err := co.GetOrderInfo(context.Background(), "12234", btcPerp, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co, canManipulateRealOrders)
	result, err := co.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Exchange:    co.Name,
		Amount:      10,
		Currency:    currency.LTC,
		PortfolioID: "1234564",
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.LTC.String(),
			Address:    "3CDJNfdWX8m2NwuGUV3nhXHXEeLygMXoAj",
			AddressTag: "",
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetActiveOrders(context.Background(), &order.MultiOrderRequest{AssetType: asset.Spot})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := co.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
	require.NoError(t, err)

	pairs, err := co.FetchTradablePairs(context.Background(), asset.Spot)
	require.NoError(t, err)
	for y := range pairs {
		lim, err := co.GetOrderExecutionLimits(asset.Spot, pairs[y])
		require.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		require.NotEmpty(t, lim, "limit cannot be empty")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := co.GetCurrencyTradeURL(context.Background(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = co.GetCurrencyTradeURL(context.Background(), asset.Futures, currency.NewPair(currency.BTC, currency.USDC))
	require.ErrorIs(t, err, asset.ErrNotSupported)

	pairs, err := co.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	resp, err := co.GetCurrencyTradeURL(context.Background(), asset.Spot, currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFeeRateTiers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, co)
	result, err := co.GetFeeRateTiers(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-PERP")
	require.NoError(t, err)

	result, err := co.GetLatestFundingRates(context.Background(), &fundingrate.LatestRateRequest{
		Pair:  cp,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = co.GetFuturesContractDetails(context.Background(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := co.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)
	_, err := co.GetHistoricCandlesExtended(context.Background(), btcPerp, asset.Options, kline.FifteenMin, startTime, end)
	require.ErrorIs(t, err, asset.ErrNotEnabled)

	result, err := co.GetHistoricCandlesExtended(context.Background(), currency.NewPair(currency.BTC, currency.USDC), asset.Spot, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetHistoricCandlesExtended(context.Background(), btcPerp, asset.Futures, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := co.GetHistoricCandles(context.Background(), btcPerp, asset.Options, kline.FifteenMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, asset.ErrNotEnabled)

	co.Verbose = true
	result, err := co.GetHistoricCandles(context.Background(), currency.NewPair(currency.BTC, currency.USDC), asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = co.GetHistoricCandles(context.Background(), btcPerp, asset.Futures, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}
