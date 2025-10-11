package coinbaseinternational

import (
	"context"
	"fmt"
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
	e                   *Exchange
	spotTP, perpetualTP currency.Pair
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	e.Enabled = true
	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	e.Websocket = sharedtestvalues.NewTestWebsocket()
	if err := e.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	setupWs()
	os.Exit(m.Run())
}

func setupWs() {
	if !e.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := e.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func (e *Exchange) populateTradablePairs() error {
	err := e.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w: no enabled currency pair found", currency.ErrCurrencyPairsEmpty)
	}
	spotTP = tradablePairs[0]
	tradablePairs, err = e.GetEnabledPairs(asset.PerpetualContract)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w: no enabled currency pair found", currency.ErrCurrencyPairsEmpty)
	}
	perpetualTP = tradablePairs[0]
	return nil
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	result, err := e.ListAssets(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetAssetDetails(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	result, err := e.GetAssetDetails(t.Context(), currency.EMPTYCODE, "", "207597618027560960")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetSupportedNetworksPerAsset(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	result, err := e.GetSupportedNetworksPerAsset(t.Context(), currency.USDC, "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexComposition(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexComposition(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexComposition(t.Context(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCompositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexCompositionHistory(t.Context(), "", time.Time{}, 0, 100)
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexCompositionHistory(t.Context(), "COIN50", time.Time{}, 0, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPrice(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexPrice(t.Context(), "COIN50")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexCandles(t.Context(), "", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errIndexNameRequired)
	_, err = e.GetIndexCandles(t.Context(), "COIN50", "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errGranularityRequired)
	_, err = e.GetIndexCandles(t.Context(), "COIN50", "ONE_DAY", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetIndexCandles(t.Context(), "COIN50", "ONE_DAY", time.Now().Add(-time.Hour*50), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := e.GetInstruments(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentDetails(t.Context(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := e.GetInstrumentDetails(t.Context(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuotePerInstrument(t.Context(), "", "", "")
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := e.GetQuotePerInstrument(t.Context(), "BTC-PERP", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDailyTradingVolumes(t *testing.T) {
	t.Parallel()
	_, err := e.GetDailyTradingVolumes(t.Context(), []string{}, 10, 10, time.Now().Add(-time.Hour*100), true)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	_, err = e.GetDailyTradingVolumes(t.Context(), []string{"BTC-PERP", ""}, 10, 10, time.Now().Add(-time.Hour*100), true)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := e.GetDailyTradingVolumes(t.Context(), []string{"BTC-PERP"}, 10, 1, time.Now().Add(-time.Hour*100), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedCandlesDataPerInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetAggregatedCandlesDataPerInstrument(t.Context(), "", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInstrumentIDRequired)
	_, err = e.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.FiveMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errStartTimeRequired)
	_, err = e.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.TenMin, time.Now().Add(-time.Hour*100), time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)

	result, err := e.GetAggregatedCandlesDataPerInstrument(t.Context(), "BTC-PERP", kline.FifteenMin, time.Now().Add(-time.Hour*100), time.Time{})
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
	_, err := e.GetHistoricalFundingRate(t.Context(), "", 0, 10)
	require.ErrorIs(t, err, errInstrumentIDRequired)

	result, err := e.GetHistoricalFundingRate(t.Context(), "BTC-PERP", 0, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionOffsets(t *testing.T) {
	t.Parallel()
	result, err := e.GetPositionOffsets(t.Context())
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
	_, err = e.CreateOrder(t.Context(), &OrderRequestParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderRequestParams{PostOnly: true}
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "BUY"
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.BaseSize = 1
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.OrderType = orderType
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder)

	arg.Price = 12345.67
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123442"
	_, err = e.CreateOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateOrder(t.Context(), &OrderRequestParams{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenOrders(t.Context(), "", "", "BTC-PERP", "PERPETUAL_FUTURE", "", "", "LIMIT", time.Time{}, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrders(t.Context(), "", "", "")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelOrders(t.Context(), "1234", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOpenOrder(t.Context(), "1234", &ModifyOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ModifyOpenOrder(t.Context(), "", &ModifyOrderParam{Portfolio: "1234"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyOpenOrder(t.Context(), "1234", &ModifyOrderParam{
		Price:     1234,
		StopPrice: 1239,
		Size:      1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetail(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderDetail(t.Context(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTradeOrder(t.Context(), "", "", "", "")
	require.ErrorIsf(t, err, order.ErrOrderIDNotSet, "expected %v, got %v", order.ErrOrderIDNotSet, err)
	_, err = e.CancelTradeOrder(t.Context(), "order-id", "", "", "")
	require.ErrorIsf(t, err, errMissingPortfolioID, "expected %v, got %v", errMissingPortfolioID, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelTradeOrder(t.Context(), "1234", "", "12344232", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAllUserPortfolios(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.CreatePortfolio(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreatePortfolio(t.Context(), "altman")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserPortfolio(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserPortfolio(t.Context(), "4thr7ft-1-0")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPatchPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.PatchPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.PatchPortfolio(t.Context(), "", "", &PatchPortfolioParams{AutoMarginEnabled: true})
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.PatchPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", &PatchPortfolioParams{AutoMarginEnabled: true, PortfolioName: "new-portfolio"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdatePortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.UpdatePortfolio(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.UpdatePortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.ErrorIs(t, err, errMissingPortfolioName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.UpdatePortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "main-portfolio")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioDetails(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioDetails(t.Context(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioMarginCallStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCallStatus(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioMarginCallStatus(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioSummary(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioSummary(t.Context(), "", "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioBalances(t *testing.T) {
	t.Parallel()
	_, err := e.ListPortfolioBalances(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListPortfolioBalances(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioAssetBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioAssetBalance(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.GetPortfolioAssetBalance(t.Context(), "", "", "BTC")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioAssetBalance(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "BTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundTransferLimitBetweenPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundTransferLimitBetweenPortfolio(t.Context(), "", "BTC")
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.GetFundTransferLimitBetweenPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundTransferLimitBetweenPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "BTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveLoansForPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveLoansForPortfolio(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveLoansForPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLoanInfoForPortfolioAsset(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanInfoForPortfolioAsset(t.Context(), "", "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = e.GetLoanInfoForPortfolioAsset(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLoanInfoForPortfolioAsset(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "BTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAcquireRepayLoan(t *testing.T) {
	t.Parallel()
	_, err := e.AcquireRepayLoan(t.Context(), "", "", "BTC", &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.AcquireRepayLoan(t.Context(), "", "", "BTC", &LoanActionAmountParam{Amount: 0.1})
	require.ErrorIs(t, err, errLoanActionMissing)
	_, err = e.AcquireRepayLoan(t.Context(), "", "", "BTC", &LoanActionAmountParam{Action: "ACQUIRE"})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	_, err = e.AcquireRepayLoan(t.Context(), "", "", "BTC", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = e.AcquireRepayLoan(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AcquireRepayLoan(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "BTC", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPreviewLoanUpdate(t *testing.T) {
	t.Parallel()
	_, err := e.PreviewLoanUpdate(t.Context(), "", "", "BTC", &LoanActionAmountParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PreviewLoanUpdate(t.Context(), "", "", "BTC", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = e.PreviewLoanUpdate(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PreviewLoanUpdate(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "1482439423963469", &LoanActionAmountParam{Action: "ACQUIRE", Amount: 0.1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestViewMaxLoanAvailability(t *testing.T) {
	t.Parallel()
	_, err := e.ViewMaxLoanAvailability(t.Context(), "", "", "BTC")
	require.ErrorIs(t, err, errMissingPortfolioID)
	_, err = e.ViewMaxLoanAvailability(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ViewMaxLoanAvailability(t.Context(), "", "892e8c7c-e979-4cad-b61b-55a197932cf1", "BTC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPortfolioPosition(t *testing.T) {
	t.Parallel()
	_, err := e.ListPortfolioPositions(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListPortfolioPositions(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioInstrumentPosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioInstrumentPosition(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetPortfolioInstrumentPosition(t.Context(), "", "", perpetualTP)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioInstrumentPosition(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", perpetualTP)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenPositionLimitsForPortfolioInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenPositionLimitsForPortfolioInstrument(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetOpenPositionLimitsForPortfolioInstrument(t.Context(), "", "", perpetualTP)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenPositionLimitsForPortfolioInstrument(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", perpetualTP)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenPositionLimitsForAllInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenPositionLimitsForAllInstruments(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOpenPositionLimitsForAllInstruments(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTotalOpenPositionLimitPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTotalOpenPositionLimitPortfolio(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTotalOpenPositionLimitPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFillsByPortfolio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFillsByPortfolio(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "", "abcdefg", 10, 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListPortfolioFills(t *testing.T) {
	t.Parallel()
	_, err := e.ListPortfolioFills(t.Context(), "", "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListPortfolioFills(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioCrossCollateral(t *testing.T) {
	t.Parallel()
	_, err := e.EnableDisablePortfolioCrossCollateral(t.Context(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableDisablePortfolioCrossCollateral(t.Context(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEnableDisablePortfolioAutoMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.EnableDisablePortfolioAutoMarginMode(t.Context(), "", "", false)
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EnableDisablePortfolioAutoMarginMode(t.Context(), "", "f67de785-60a7-45ea-b87a-07e83eae7c12", true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPortfolioMarginOverride(t *testing.T) {
	t.Parallel()
	_, err := e.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverride{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverride{MarginOverride: .2})
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverride{PortfolioID: "f67de785-60a7-45ea-b87"})
	require.ErrorIs(t, err, errMarginOverrideValueMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetPortfolioMarginOverride(t.Context(), &PortfolioMarginOverride{MarginOverride: .5, PortfolioID: "f67de785-60a7-45ea-b87"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferFundsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := e.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{From: "", To: "5189861793641175"})
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175"})
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	_, err = e.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", AssetID: "BTC"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TransferFundsBetweenPortfolios(t.Context(), &TransferFundsBetweenPortfoliosParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", AssetID: "BTC", Amount: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferPositionsBetweenPortfolios(t *testing.T) {
	t.Parallel()
	_, err := e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "", To: "5189861793641175"})
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175"})
	require.ErrorIs(t, err, errInstrumentIDRequired)

	_, err = e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Instrument: "BTC-PERP"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	_, err = e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Instrument: "BTC-PERP", Quantity: 123})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TransferPositionsBetweenPortfolios(t.Context(), &TransferPortfolioParams{From: "892e8c7c-e979-4cad-b61b-55a197932cf1", To: "5189861793641175", Instrument: "BTC-PERP", Quantity: 123, Side: "BUY"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPortfolioFeeRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPortfolioFeeRates(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetYourRanking(t *testing.T) {
	t.Parallel()
	_, err := e.GetYourRanking(t.Context(), "", "THIS_MONTH", []string{})
	require.ErrorIs(t, err, errInstrumentTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetYourRanking(t.Context(), "SPOT", "THIS_MONTH", []string{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCounterPartyID(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCounterpartyID(t.Context(), "")
	require.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateCounterpartyID(t.Context(), "5189861793641175")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := e.ValidateCounterpartyID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ValidateCounterpartyID(t.Context(), "CBTQDGENHE")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCounterpartyID(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{Portfolio: "", CounterpartyID: "CBTQDGENHE", Asset: "BTC", Amount: 2})
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{Portfolio: "5189861793641175", Asset: "BTC", Amount: 2})
	require.ErrorIs(t, err, errMissingCounterpartyID)

	_, err = e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{Portfolio: "5189861793641175", CounterpartyID: "CBTQDGENHE", Amount: 2})
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	_, err = e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{Portfolio: "5189861793641175", CounterpartyID: "CBTQDGENHE", Asset: "BTC"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawToCounterpartyID(t.Context(), &AssetCounterpartyWithdrawalResponse{
		Portfolio:      "5189861793641175",
		CounterpartyID: "CBTQDGENHE",
		Asset:          "BTC",
		Amount:         2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCounterpartyWithdrawalLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetCounterpartyWithdrawalLimit(t.Context(), "", "291efb0f-2396-4d41-ad03-db3b2311cb2c")
	require.ErrorIs(t, err, errMissingPortfolioID)

	_, err = e.GetCounterpartyWithdrawalLimit(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "")
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCounterpartyWithdrawalLimit(t.Context(), "892e8c7c-e979-4cad-b61b-55a197932cf1", "291efb0f-2396-4d41-ad03-db3b2311cb2c")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestListMatchingTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ListMatchingTransfers(t.Context(), []string{}, "", "ALL", 10, 0, time.Now().Add(-time.Hour*24*10), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransfer(t.Context(), "")
	require.ErrorIs(t, err, errMissingTransferID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransfer(t.Context(), "12345")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawToCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawToCryptoAddress(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &WithdrawCryptoParams{Nonce: 1234}
	_, err = e.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errMissingPortfolioID)

	arg.Portfolio = "892e8c7c-e979-4cad-b61b-55a197932cf1"
	_, err = e.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errMissingNetworkArnID)

	arg.NetworkArnID = "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a"
	_, err = e.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errAddressIsRequired)

	arg.Address = "1234HGJHGHGHGJ"
	_, err = e.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	arg.Amount = 1200
	_, err = e.WithdrawToCryptoAddress(t.Context(), arg)
	require.ErrorIs(t, err, errAssetIdentifierRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawToCryptoAddress(t.Context(), &WithdrawCryptoParams{
		Portfolio:    "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetID:      "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID: "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
		Amount:       1200,
		Address:      "1234HGJHGHGHGJ",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateCryptoAddress(t *testing.T) {
	t.Parallel()
	_, err := e.CreateCryptoAddress(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CryptoAddressParam{
		NetworkArnID: "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	}
	_, err = e.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errAssetIdentifierRequired)

	arg.AssetID = "291efb0f-2396-4d41-ad03-db3b2311cb2c"
	_, err = e.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errMissingPortfolioID)

	arg.Portfolio = "892e8c7c-e979-4cad-b61b-55a197932cf1"
	arg.NetworkArnID = ""
	_, err = e.CreateCryptoAddress(t.Context(), arg)
	assert.ErrorIs(t, err, errMissingNetworkArnID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateCryptoAddress(t.Context(), &CryptoAddressParam{
		Portfolio:    "892e8c7c-e979-4cad-b61b-55a197932cf1",
		AssetID:      "291efb0f-2396-4d41-ad03-db3b2311cb2c",
		NetworkArnID: "networks/ethereum-mainnet/assets/313ef8a9-ae5a-5f2f-8a56-572c0e2a4d5a",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := e.UpdateTradablePairs(t.Context(), true)
	assert.NoError(t, err)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := e.UpdateTicker(t.Context(), perpetualTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = e.UpdateTickers(t.Context(), asset.PerpetualContract)
	assert.NoError(t, err)

	err = e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGenerateSubscriptionPayload(t *testing.T) {
	t.Parallel()
	_, err := e.GenerateSubscriptionPayload(subscription.List{}, "SUBSCRIBE")
	require.ErrorIs(t, err, common.ErrEmptyParams)

	payload, err := e.GenerateSubscriptionPayload(subscription.List{
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
	_, err := e.FetchOrderbook(t.Context(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.FetchOrderbook(t.Context(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchOrderbook(t.Context(), perpetualTP, asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), spotTP, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.UpdateOrderbook(t.Context(), spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateOrderbook(t.Context(), perpetualTP, asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountInfo(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.UpdateAccountInfo(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := e.FetchAccountInfo(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.FetchAccountInfo(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.FetchAccountInfo(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    perpetualTP,
		FeeType: exchange.CryptocurrencyTradeFee,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		IsMaker: true,
		Pair:    perpetualTP,
		FeeType: exchange.CryptocurrencyWithdrawalFee,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	result, err := e.GetAvailableTransferChains(t.Context(), currency.USDC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      e.Name,
		Pair:          perpetualTP,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ModifyOrder(t.Context(), &order.Modify{
		Exchange:  "CoinbaseInternational",
		OrderID:   "1337",
		Price:     10000,
		Amount:    10,
		Side:      order.Sell,
		Pair:      perpetualTP,
		AssetType: asset.PerpetualContract,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err := e.CancelOrder(t.Context(), &order.Cancel{
		Exchange:  "CoinbaseInternational",
		AssetType: asset.PerpetualContract,
		Pair:      perpetualTP,
		OrderID:   "1234",
		AccountID: "Someones SubAccount",
	})
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errMissingPortfolioID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(t.Context(), &order.Cancel{
		Exchange:  "CoinbaseInternational",
		AssetType: asset.Spot,
		AccountID: "Sam",
		Pair:      spotTP,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderInfo(t.Context(), "12234", spotTP, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:    e.Name,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	require.NoError(t, err)

	pairs, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	for y := range pairs {
		lim, err := e.GetOrderExecutionLimits(asset.Spot, pairs[y])
		require.NoErrorf(t, err, "%v %s %v", err, pairs[y], asset.Spot)
		require.NotEmpty(t, lim, "limit cannot be empty")
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.GetCurrencyTradeURL(t.Context(), asset.PerpetualContract, perpetualTP)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	pairs, err := e.CurrencyPairs.GetPairs(asset.Spot, false)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	resp, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, spotTP)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFeeRateTiers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFeeRateTiers(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Pair:  perpetualTP,
		Asset: asset.PerpetualContract,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 24 * 3)
	end := time.Now().Add(-time.Hour * 1)
	_, err := e.GetHistoricCandlesExtended(t.Context(), perpetualTP, asset.Options, kline.FifteenMin, startTime, end)
	require.ErrorIs(t, err, currency.ErrAssetNotFound)

	result, err := e.GetHistoricCandlesExtended(t.Context(), spotTP, asset.Spot, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandlesExtended(t.Context(), perpetualTP, asset.PerpetualContract, kline.OneMin, startTime, end)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandles(t.Context(), perpetualTP, asset.Options, kline.FifteenMin, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrAssetNotFound)

	result, err := e.GetHistoricCandles(t.Context(), spotTP, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetHistoricCandles(t.Context(), perpetualTP, asset.PerpetualContract, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}
