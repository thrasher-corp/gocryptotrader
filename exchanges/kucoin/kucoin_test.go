package kucoin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passPhrase              = ""
	canManipulateRealOrders = false
)

var (
	ku                                                        *Kucoin
	spotTradablePair, marginTradablePair, futuresTradablePair currency.Pair
	assertToTradablePairMap                                   map[asset.Item]currency.Pair
)

func TestMain(m *testing.M) {
	ku = new(Kucoin)
	if err := testexch.Setup(ku); err != nil {
		log.Fatalf("Kucoin Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" && passPhrase != "" {
		ku.API.AuthenticatedSupport = true
		ku.API.AuthenticatedWebsocketSupport = true
		ku.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		ku.SetCredentials(apiKey, apiSecret, passPhrase, "", "", "")
		ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	getFirstTradablePairOfAssets(context.Background())
	assertToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:    spotTradablePair,
		asset.Margin:  marginTradablePair,
		asset.Futures: futuresTradablePair,
	}
	fetchedFuturesOrderbook = map[string]bool{}

	os.Exit(m.Run())
}

// Spot asset test cases starts from here
func TestGetSymbols(t *testing.T) {
	t.Parallel()
	symbols, err := ku.GetSymbols(t.Context(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, symbols)
	// Using market string reduces the scope of what is returned.
	symbols, err = ku.GetSymbols(t.Context(), "ETF")
	assert.NoError(t, err)
	assert.NotEmpty(t, symbols, "should return all available ETF symbols")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetTicker(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	result, err := ku.GetTickers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	tickers, err := ku.GetFuturesTickers(t.Context())
	assert.NoError(t, err)
	for i := range tickers {
		assert.Positive(t, tickers[i].Last, "Last should be positive")
		assert.Positive(t, tickers[i].Bid, "Bid should be positive")
		assert.Positive(t, tickers[i].Ask, "Ask should be positive")
		assert.NotEmpty(t, tickers[i].Pair, "Pair should not be empty")
		assert.NotEmpty(t, tickers[i].LastUpdated, "LastUpdated should not be empty")
		assert.Equal(t, ku.Name, tickers[i].ExchangeName)
		assert.Equal(t, asset.Futures, tickers[i].AssetType)
	}
}

func TestGet24hrStats(t *testing.T) {
	t.Parallel()
	_, err := ku.Get24hrStats(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.Get24hrStats(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarketList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarketList(t.Context())
	assert.NoError(t, err)
}

func TestGetPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook20(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetPartOrderbook20(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook100(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetPartOrderbook100(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOrderbook(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetOrderbook(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTradeHistory(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = ku.GetTradeHistory(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
}

func TestKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[["1746645900","96248.3","96060.4","96248.3","95991.1","7.30387554","701787.956631596"],["1746645600","96407.2","96243.5","96420.2","96213.1","6.72799595","648257.95148221"],["1746645300","96382.8","96407.2","96466.1","96227.8","7.31425727","704541.034713515"],["1746645000","96490.5","96382.8","96503","96376.7","5.06147446","488102.261377795"],["1746644700","96424","96490.5","96517.9","96323.4","12.04216802","1160916.511036681"],["1746644400","96593.4","96423.9","96608.6","96403","10.75654084","1037793.471887188"],["1746644100","96200.5","96588.1","96591.6","96200.5","10.12317892","976893.020212471"],["1746643800","96182.2","96191.8","96241.7","95998.6","8.00901063","769988.0586614"],["1746643500","96404.1","96160.1","96477.6","96102.8","10.86244787","1045287.271213675"],["1746643200","96680.1","96395.4","96734.7","96395.3","9.54921963","921978.587594588"],["1746642900","96790.7","96680.1","96851.6","96587.5","11.35501379","1098593.622144195"],["1746642600","96447.7","96760","96868.5","96291.1","16.35392542","1580649.199051741"]]`)
	var target []Kline
	err := json.Unmarshal(data, &target)
	require.NoError(t, err)
	require.Len(t, target, 12)
	assert.Equal(t, Kline{
		StartTime: types.Time(time.Unix(1746645900, 0)),
		Open:      96248.3,
		Close:     96060.4,
		High:      96248.3,
		Low:       95991.1,
		Volume:    7.30387554,
		Amount:    701787.956631596,
	}, target[0])
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := ku.GetKlines(t.Context(), "", "1week", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.GetKlines(t.Context(), spotTradablePair.String(), "invalid-period", time.Time{}, time.Time{})
	require.ErrorIs(t, err, errInvalidPeriod)

	result, err := ku.GetKlines(t.Context(), spotTradablePair.String(), "1week", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetKlines(t.Context(), spotTradablePair.String(), "5min", time.Now().Add(-time.Hour*1), time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrenciesV3(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrenciesV3(t.Context())
	assert.NoError(t, err)
}

func TestGetCurrencyV3(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencyDetailV3(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	result, err := ku.GetCurrencyDetailV3(t.Context(), currency.BTC, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetCurrencyDetailV3(t.Context(), currency.BTC, "ETH")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatPrice(t *testing.T) {
	t.Parallel()
	result, err := ku.GetFiatPrice(t.Context(), "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetFiatPrice(t.Context(), "EUR", "ETH,BTC")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeveragedTokenInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetLeveragedTokenInfo(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarkPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetMarkPrice(t.Context(), marginTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllMarginTradingPairsMarkPrices(t *testing.T) {
	t.Parallel()
	result, err := ku.GetAllMarginTradingPairsMarkPrices(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginConfiguration(t *testing.T) {
	t.Parallel()
	result, err := ku.GetMarginConfiguration(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetMarginAccount(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginRiskLimitCurrencyConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetCrossMarginRiskLimitCurrencyConfig(t.Context(), "", currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginRiskLimitCurrencyConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetIsolatedMarginRiskLimitCurrencyConfig(t.Context(), "", currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostBorrowOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PostMarginBorrowOrder(t.Context(), &MarginBorrowParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ku.PostMarginBorrowOrder(t.Context(), &MarginBorrowParam{IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = ku.PostMarginBorrowOrder(t.Context(), &MarginBorrowParam{IsIsolated: true, Currency: currency.BTC})
	require.ErrorIs(t, err, errTimeInForceRequired)

	_, err = ku.PostMarginBorrowOrder(t.Context(),
		&MarginBorrowParam{
			Currency:    currency.USDT,
			TimeInForce: "FOK",
			Size:        0,
		})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PostMarginBorrowOrder(t.Context(),
		&MarginBorrowParam{
			Currency:    currency.USDT,
			TimeInForce: "IOC",
			Size:        0.05,
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginBorrowingHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginBorrowingHistory(t.Context(), currency.EMPTYCODE, true, marginTradablePair.String(), "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.GetMarginBorrowingHistory(t.Context(), currency.BTC, true, "", "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetMarginBorrowingHistory(t.Context(), currency.BTC, true, marginTradablePair.String(), "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	assert.NoError(t, err)
}

func TestPostRepayment(t *testing.T) {
	t.Parallel()
	_, err := ku.PostRepayment(t.Context(), &RepayParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.PostRepayment(t.Context(), &RepayParam{Size: 0.05})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.PostRepayment(t.Context(), &RepayParam{Currency: currency.ETH})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PostRepayment(t.Context(), &RepayParam{
		Currency: currency.USDT,
		Size:     0.05,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossIsolatedMarginInterestRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetCrossIsolatedMarginInterestRecords(t.Context(), false, "", currency.BTC, time.Now().Add(-time.Hour*50), time.Now(), 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRepaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetRepaymentHistory(t.Context(), currency.EMPTYCODE, true, spotTradablePair.String(), "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetRepaymentHistory(t.Context(), currency.BTC, true, spotTradablePair.String(), "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginPairConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetIsolatedMarginPairConfig(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetIsolatedMarginAccountInfo(t.Context(), "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = ku.GetIsolatedMarginAccountInfo(t.Context(), "USDT")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := ku.GetSingleIsolatedMarginAccountInfo(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetSingleIsolatedMarginAccountInfo(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	result, err := ku.GetCurrentServerTime(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()
	result, err := ku.GetServiceStatus(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostOrder(t *testing.T) {
	t.Parallel()

	// default order type is limit
	_, err := ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: "",
	})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	customID, err := uuid.NewV4()
	assert.NoError(t, err)

	_, err = ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair,
		OrderType: "",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565,
	})

	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(),
		Side:          "buy",
		Symbol:        spotTradablePair,
		OrderType:     "limit",
		Size:          0.005,
		Price:         1000,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostOrderTest(t *testing.T) {
	t.Parallel()

	// default order type is limit
	_, err := ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: "",
	})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	customID, err := uuid.NewV4()
	assert.NoError(t, err)

	_, err = ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair,
		OrderType: "",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565,
	})

	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.PostOrderTest(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(),
		Side:          "buy",
		Symbol:        spotTradablePair,
		OrderType:     "limit",
		Size:          0.005,
		Price:         1000,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHandlePostOrder(t *testing.T) {
	t.Parallel()
	// default order type is limit
	_, err := ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: "",
	}, "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	customID, err := uuid.NewV4()
	assert.NoError(t, err)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair,
		OrderType: "",
	}, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565,
	}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "OCO", Size: 0.1,
	}, "")
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1,
	}, "")
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0, Price: 1000,
	}, "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: .1, Price: 1000, VisibleSize: -1,
	}, "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	_, err = ku.HandlePostOrder(t.Context(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair, Side: "buy",
		OrderType: "market", Price: 234565,
	}, "")
	require.ErrorIs(t, err, errSizeOrFundIsRequired)
}

func TestPostMarginOrder(t *testing.T) {
	t.Parallel()
	// default order type is limit
	_, err := ku.PostMarginOrder(t.Context(), &MarginOrderParam{
		ClientOrderID: "",
	})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.PostMarginOrder(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair,
		OrderType: "",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostMarginOrder(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.PostMarginOrder(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    marginTradablePair,
		OrderType: "limit", Size: 0.1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.PostMarginOrder(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	// default order type is limit and margin mode is cross
	result, err := ku.PostMarginOrder(t.Context(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			Price: 1000, Size: 0.1, PostOnly: true,
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// market isolated order
	result, err = ku.PostMarginOrder(t.Context(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			OrderType: "market", Funds: 1234,
			Remark: "remark", MarginModel: "cross", Price: 1000, PostOnly: true, AutoBorrow: true,
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostMarginOrderTest(t *testing.T) {
	t.Parallel()
	// default order type is limit
	_, err := ku.PostMarginOrderTest(t.Context(), &MarginOrderParam{
		ClientOrderID: "",
	})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.PostMarginOrderTest(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair,
		OrderType: "",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostMarginOrderTest(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.PostMarginOrderTest(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    marginTradablePair,
		OrderType: "limit", Size: 0.1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.PostMarginOrderTest(t.Context(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	// default order type is limit and margin mode is cross
	result, err := ku.PostMarginOrderTest(t.Context(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			Price: 1000, Size: 0.1, PostOnly: true,
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// market isolated order
	result, err = ku.PostMarginOrderTest(t.Context(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			OrderType: "market", Funds: 1234,
			Remark: "remark", MarginModel: "cross", Price: 1000, PostOnly: true, AutoBorrow: true,
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostBulkOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PostBulkOrder(t.Context(), "", []OrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := OrderRequest{
		Size: 0.01,
	}
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{arg})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	arg.ClientOID = "3d07008668054da6b3cb12e432c2b13a"
	arg.Size = 0
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Sell"
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{arg})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 1000
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.PostBulkOrder(t.Context(), spotTradablePair.String(), []OrderRequest{
		{
			ClientOID: "3d07008668054da6b3cb12e432c2b13a",
			Side:      "buy",
			Type:      "limit",
			Price:     1000,
			Size:      0.01,
		},
		{
			ClientOID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:      "buy",
			Type:      "limit",
			Price:     1000,
			Size:      0.01,
		},
	})
	assert.NoError(t, err)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelSingleOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelSingleOrder(t.Context(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByClientOID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelOrderByClientOID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelOrderByClientOID(t.Context(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllOpenOrders(t.Context(), "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const ordersListResponseJSON = `{"currentPage": 1, "pageSize": 1, "totalNum": 153408, "totalPage": 153408, "items": [ { "id": "5c35c02703aa673ceec2a168", "symbol": "BTC-USDT", "opType": "DEAL", "type": "limit", "side": "buy", "price": "10", "size": "2", "funds": "0", "dealFunds": "0.166", "dealSize": "2", "fee": "0", "feeCurrency": "USDT", "stp": "", "stop": "", "stopTriggered": false, "stopPrice": "0", "timeInForce": "GTC", "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": "0", "cancelAfter": 0, "channel": "IOS", "clientOid": "", "remark": "", "tags": "", "isActive": false, "cancelExist": false, "createdAt": 1547026471000, "tradeType": "TRADE" } ] }`

func TestGetOrders(t *testing.T) {
	t.Parallel()
	var resp *OrdersListResponse
	err := json.Unmarshal([]byte(ordersListResponseJSON), &resp)
	assert.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	result, err := ku.ListOrders(t.Context(), "", "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetRecentOrders(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOrderByID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetOrderByID(t.Context(), "5c35c02703aa673ceec2a168")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderByClientOID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOrderByClientSuppliedOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetOrderByClientSuppliedOrderID(t.Context(), "6d539dc614db312")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFills(t.Context(), "", "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetFills(t.Context(), "5c35c02703aa673ceec2a168", spotTradablePair.String(), "buy", "limit", SpotTradeType, time.Now().Add(-time.Hour*12), time.Now())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const limitFillsResponseJSON = `[{ "counterOrderId":"5db7ee769797cf0008e3beea", "createdAt":1572335233000, "fee":"0.946357371456", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.357371456", "liquidity":"taker", "orderId":"5db7ee805d53620008dce1ba", "price":"9466.8", "side":"buy", "size":"0.09996592", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee8054c05c0008069e21", "tradeType":"MARGIN_TRADE", "type":"market" }, { "counterOrderId":"5db7ee4b5d53620008dcde8e", "createdAt":1572335207000, "fee":"0.94625", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.25", "liquidity":"taker", "orderId":"5db7ee675d53620008dce01e", "price":"9462.5", "side":"sell", "size":"0.1", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee6754c05c0008069e03", "tradeType":"MARGIN_TRADE", "type":"market" }]`

func TestGetRecentFills(t *testing.T) {
	t.Parallel()
	var resp []Fill
	err := json.Unmarshal([]byte(limitFillsResponseJSON), &resp)
	assert.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetRecentFills(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostStopOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PostStopOrder(t.Context(), "", "buy", spotTradablePair.String(), "", "", "entry", "CO", SpotTradeType, "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.PostStopOrder(t.Context(), "5bd6e9286d99522a52e458de", "", spotTradablePair.String(), "", "", "entry", "CO", SpotTradeType, "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostStopOrder(t.Context(), "5bd6e9286d99522a52e458de", "buy", "", "", "", "entry", "CO", SpotTradeType, "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PostStopOrder(t.Context(), "5bd6e9286d99522a52e458de", "buy", spotTradablePair.String(), "", "", "entry", "CO", SpotTradeType, "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelStopOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelStopOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelStopOrder(t.Context(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelStopOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelStopOrderByClientOrderID(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelStopOrderByClientOrderID(t.Context(), "5bd6e9286d99522a52e458de", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelStopOrders(t.Context(), "", "", []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const stopOrderResponseJSON = `{"id": "vs8hoo8q2ceshiue003b67c0", "symbol": "KCS-USDT", "userId": "60fe4956c43cbc0006562c2c", "status": "NEW", "type": "limit", "side": "buy", "price": "0.01000000000000000000", "size": "0.01000000000000000000", "funds": null, "stp": null, "timeInForce": "GTC", "cancelAfter": -1, "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": null, "channel": "API", "clientOid": "40e0eb9efe6311eb8e58acde48001122", "remark": null, "tags": null, "orderTime": 1629098781127530345, "domainId": "kucoin", "tradeSource": "USER", "tradeType": "TRADE", "feeCurrency": "USDT", "takerFeeRate": "0.00200000000000000000", "makerFeeRate": "0.00200000000000000000", "createdAt": 1629098781128, "stop": "loss", "stopTriggerTime": null, "stopPrice": "10.00000000000000000000" }`

func TestGetStopOrder(t *testing.T) {
	t.Parallel()
	var resp *StopOrder
	err := json.Unmarshal([]byte(stopOrderResponseJSON), &resp)
	assert.NoError(t, err)
	_, err = ku.GetStopOrder(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetStopOrder(t.Context(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.ListStopOrders(t.Context(), "", "", "", "", []string{}, time.Time{}, time.Time{}, 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetStopOrderByClientID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetStopOrderByClientID(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetStopOrderByClientID(t.Context(), "", "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelStopOrderByClientID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelStopOrderByClientID(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelStopOrderByClientID(t.Context(), "", "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAllAccounts(t.Context(), currency.EMPTYCODE, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	_, err := ku.GetAccountDetail(t.Context(), "")
	require.ErrorIs(t, err, errAccountIDMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAccountDetail(t.Context(), "62fcd1969474ea0001fd20e4")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCrossMarginAccountsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetCrossMarginAccountsDetail(t.Context(), "KCS", "MARGIN_V2")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIsolatedMarginAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetIsolatedMarginAccountDetail(t.Context(), marginTradablePair.String(), "BTC", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesAccountDetail(t.Context(), currency.USDT)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := ku.GetSubAccounts(t.Context(), "", false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetSubAccounts(t.Context(), "5caefba7d9575a0688f83c45", false)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllFuturesSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAllFuturesSubAccountBalances(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const accountLedgerResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 2, "totalPage": 1, "items": [ { "id": "611a1e7c6a053300067a88d9", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629101692950, "context": "{\"borrowerUserId\":\"601ad03e50dc810006d242ea\",\"loanRepayDetailNo\":\"611a1e7cc913d000066cf7ec\"}" }, { "id": "611a18bc6a0533000671e1bf", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629100220843, "context": "{\"borrowerUserId\":\"5e3f4623dbf52d000800292f\",\"loanRepayDetailNo\":\"611a18bc7255c200063ea545\"}" } ] }`

func TestGetAccountLedgers(t *testing.T) {
	t.Parallel()
	var resp *AccountLedgerResponse
	err := json.Unmarshal([]byte(accountLedgerResponseJSON), &resp)
	assert.NoError(t, err)

	_, err = ku.GetAccountLedgers(t.Context(), currency.EMPTYCODE, "", "", time.Now(), time.Now().Add(-time.Hour*24*10))
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAccountLedgers(t.Context(), currency.EMPTYCODE, "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountLedgersHFTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountLedgersHFTrade(t.Context(), currency.BTC, "", "", 0, 10, time.Time{}, time.Now())
	assert.NoError(t, err)
}

func TestGetAccountLedgerHFMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAccountLedgerHFMargin(t.Context(), currency.BTC, "", "", 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountLedgers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesAccountLedgers(t.Context(), currency.BTC, true, time.Time{}, time.Now(), 0, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSubAccountsInfoV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAllSubAccountsInfoV1(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSubAccountsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAllSubAccountsInfoV2(t.Context(), 0, 30)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountSummaryInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAccountSummaryInformation(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAggregatedSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAggregatedSubAccountBalance(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAllSubAccountsBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAllSubAccountsBalanceV2(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPaginatedSubAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetPaginatedSubAccountInformation(t.Context(), 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTransferableBalance(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTransferableBalance(t.Context(), currency.EMPTYCODE, "MAIN", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.GetTransferableBalance(t.Context(), currency.BTC, "", "")
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetTransferableBalance(t.Context(), currency.BTC, "MAIN", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUniversalTransfer(t *testing.T) {
	t.Parallel()
	_, err := ku.GetUniversalTransfer(t.Context(), &UniversalTransferParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &UniversalTransferParam{
		ToAccountTag: "1234",
	}
	_, err = ku.GetUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	arg.ClientSuppliedOrderID = "64ccc0f164781800010d8c09"
	_, err = ku.GetUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = ku.GetUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errAccountTypeMissing)

	arg.FromAccountType = "MAIN"
	_, err = ku.GetUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errTransferTypeMissing)

	arg.TransferType = "INTERNAL"
	_, err = ku.GetUniversalTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.GetUniversalTransfer(t.Context(), &UniversalTransferParam{
		ClientSuppliedOrderID: "64ccc0f164781800010d8c09",
		TransferType:          "INTERNAL",
		Currency:              currency.BTC,
		Amount:                1,
		FromAccountType:       SpotTradeType,
		ToAccountType:         "CONTRACT",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetUniversalTransfer(t.Context(), &UniversalTransferParam{
		ClientSuppliedOrderID: "64ccc0f164781800010d8c09",
		TransferType:          "PARENT_TO_SUB",
		Currency:              currency.BTC,
		Amount:                1,
		FromAccountType:       SpotTradeType,
		ToUserID:              "62f5f5d4d72aaf000122707e",
		ToAccountType:         "CONTRACT",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferMainToSubAccount(t *testing.T) {
	t.Parallel()
	_, err := ku.TransferMainToSubAccount(t.Context(), currency.EMPTYCODE, 1, "62fcd1969474ea0001fd20e4", "OUT", "", "", "5caefba7d9575a0688f83c45")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.TransferMainToSubAccount(t.Context(), currency.BTC, 1, "", "OUT", "", "", "5caefba7d9575a0688f83c45")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.TransferMainToSubAccount(t.Context(), currency.BTC, 0, "62fcd1969474ea0001fd20e4", "OUT", "", "", "5caefba7d9575a0688f83c45")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.TransferMainToSubAccount(t.Context(), currency.BTC, 1, "62fcd1969474ea0001fd20e4", "", "", "", "5caefba7d9575a0688f83c45")
	require.ErrorIs(t, err, errTransferDirectionRequired)
	_, err = ku.TransferMainToSubAccount(t.Context(), currency.BTC, 1, "62fcd1969474ea0001fd20e4", "OUT", "", "", "")
	require.ErrorIs(t, err, errSubUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.TransferMainToSubAccount(t.Context(), currency.BTC, 1, "62fcd1969474ea0001fd20e4", "OUT", "", "", "5caefba7d9575a0688f83c45")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMakeInnerTransfer(t *testing.T) {
	t.Parallel()
	_, err := ku.MakeInnerTransfer(t.Context(), 0, currency.EMPTYCODE, "62fcd1969474ea0001fd20e4", "trade", "main", "1", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.MakeInnerTransfer(t.Context(), 0, currency.USDT, "", "trade", "main", "1", "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.MakeInnerTransfer(t.Context(), 0, currency.USDT, "62fcd1969474ea0001fd20e4", "", "main", "", "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.MakeInnerTransfer(t.Context(), 1, currency.USDT, "62fcd1969474ea0001fd20e4", "", "main", "", "")
	require.ErrorIs(t, err, errAccountTypeMissing)
	_, err = ku.MakeInnerTransfer(t.Context(), 5, currency.USDT, "62fcd1969474ea0001fd20e4", "margin_hf", "", "", "")
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.MakeInnerTransfer(t.Context(), 10, currency.USDT, "62fcd1969474ea0001fd20e4", "main", "trade_hf", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferToMainOrTradeAccount(t *testing.T) {
	t.Parallel()
	_, err := ku.TransferToMainOrTradeAccount(t.Context(), &FundTransferFuturesParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.TransferToMainOrTradeAccount(t.Context(), &FundTransferFuturesParam{RecieveAccountType: "MAIN"})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.TransferToMainOrTradeAccount(t.Context(), &FundTransferFuturesParam{Amount: 1, RecieveAccountType: "MAIN"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.TransferToMainOrTradeAccount(t.Context(), &FundTransferFuturesParam{
		Amount:             1,
		Currency:           currency.USDT,
		RecieveAccountType: SpotTradeType,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferToFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := ku.TransferToFuturesAccount(t.Context(), &FundTransferToFuturesParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.TransferToFuturesAccount(t.Context(), &FundTransferToFuturesParam{PaymentAccountType: "Main"})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.TransferToFuturesAccount(t.Context(), &FundTransferToFuturesParam{PaymentAccountType: "Main", Amount: 12})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.TransferToFuturesAccount(t.Context(), &FundTransferToFuturesParam{
		Amount:             60,
		Currency:           currency.USDT,
		PaymentAccountType: SpotTradeType,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTransferOutRequestRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesTransferOutRequestRecords(t.Context(), time.Time{}, time.Now(), "", "", currency.BTC, 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := ku.CreateDepositAddress(t.Context(), &DepositAddressParams{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CreateDepositAddress(t.Context(), &DepositAddressParams{
		Currency: currency.BTC,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.CreateDepositAddress(t.Context(),
		&DepositAddressParams{
			Currency: currency.USDT,
			Chain:    "TRC20",
		})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressV2(t *testing.T) {
	t.Parallel()
	_, err := ku.GetDepositAddressesV2(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetDepositAddressesV2(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddressesV1(t *testing.T) {
	t.Parallel()
	_, err := ku.GetDepositAddressV1(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetDepositAddressV1(t.Context(), currency.BTC, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const depositResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 1, "totalPage": 1, "items": [ { "currency": "XRP", "chain": "xrp", "status": "SUCCESS", "address": "rNFugeoj3ZN8Wv6xhuLegUBBPXKCyWLRkB", "memo": "1919537769", "isInner": false, "amount": "20.50000000", "fee": "0.00000000", "walletTxId": "2C24A6D5B3E7D5B6AA6534025B9B107AC910309A98825BF5581E25BEC94AD83B@e8902757998fc352e6c9d8890d18a71c", "createdAt": 1666600519000, "updatedAt": 1666600549000, "remark": "Deposit" } ] }`

func TestGetDepositList(t *testing.T) {
	t.Parallel()
	var resp DepositResponse
	err := json.Unmarshal([]byte(depositResponseJSON), &resp)
	assert.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetDepositList(t.Context(), currency.EMPTYCODE, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const historicalDepositResponseJSON = `{"currentPage":1, "pageSize":1, "totalNum":9, "totalPage":9, "items":[ { "currency":"BTC", "createAt":1528536998, "amount":"0.03266638", "walletTxId":"55c643bc2c68d6f17266383ac1be9e454038864b929ae7cee0bc408cc5c869e8@12ffGWmMMD1zA1WbFm7Ho3JZ1w6NYXjpFk@234", "isInner":false, "status":"SUCCESS" } ] }`

func TestGetHistoricalDepositList(t *testing.T) {
	t.Parallel()
	var resp *HistoricalDepositWithdrawalResponse
	err := json.Unmarshal([]byte(historicalDepositResponseJSON), &resp)
	assert.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHistoricalDepositList(t.Context(), currency.EMPTYCODE, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetWithdrawalList(t.Context(), currency.EMPTYCODE, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHistoricalWithdrawalList(t.Context(), currency.EMPTYCODE, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	_, err := ku.GetWithdrawalQuotas(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetWithdrawalQuotas(t.Context(), currency.BTC, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := ku.ApplyWithdrawal(t.Context(), currency.EMPTYCODE, "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.ApplyWithdrawal(t.Context(), currency.ETH, "", "", "", "", "", false, 1)
	require.ErrorIs(t, err, errAddressRequired)
	_, err = ku.ApplyWithdrawal(t.Context(), currency.ETH, "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.ApplyWithdrawal(t.Context(), currency.ETH, "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	err := ku.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err = ku.CancelWithdrawal(t.Context(), "5bffb63303aa675e8bbe18f9")
	assert.NoError(t, err)
}

func TestGetBasicFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetBasicFee(t.Context(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradingFee(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTradingFee(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	avail, err := ku.GetAvailablePairs(asset.Spot)
	assert.NoError(t, err)
	assert.NotEmpty(t, avail)

	pairs := currency.Pairs{avail[0]}
	btcusdTradingFee, err := ku.GetTradingFee(t.Context(), pairs)
	assert.NoErrorf(t, err, "received %v, expected %v", err, nil)
	assert.Len(t, btcusdTradingFee, 1)

	// NOTE: Test below will error out from an external call as this will exceed
	// the allowed pairs. If this does not error then this endpoint will allow
	// more items to be requested.
	pairs = append(pairs, avail[1:10]...)
	result, err := ku.GetTradingFee(t.Context(), pairs)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	got, err := ku.GetTradingFee(t.Context(), pairs[:10])
	assert.NoError(t, err)
	assert.Len(t, got, 10)
}

// futures
func TestGetFuturesOpenContracts(t *testing.T) {
	t.Parallel()
	result, err := ku.GetFuturesOpenContracts(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesContract(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesContract(t.Context(), "XBTUSDTM")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	tick, err := ku.GetFuturesTicker(t.Context(), "XBTUSDTM")
	if assert.NoError(t, err) {
		assert.Positive(t, tick.Sequence, "Sequence should be positive")
		assert.Equal(t, "XBTUSDTM", tick.Symbol)
		assert.Contains(t, []order.Side{order.Buy, order.Sell}, tick.Side, "Side should be a side")
		assert.Positive(t, tick.Size, "Size should be positive")
		assert.Positive(t, tick.Price.Float64(), "Price should be positive")
		assert.Positive(t, tick.BestBidPrice.Float64(), "BestBidPrice should be positive")
		assert.Positive(t, tick.BestBidSize, "BestBidSize should be positive")
		assert.Positive(t, tick.BestAskPrice.Float64(), "BestAskPrice should be positive")
		assert.Positive(t, tick.BestAskSize, "BestAskSize should be positive")
		assert.NotEmpty(t, tick.TradeID, "TradeID should not be empty")
		assert.WithinRange(t, tick.FilledTime.Time(), time.Now().Add(time.Hour*-24), time.Now().Add(time.Hour))
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOrderbook(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesOrderbook(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook20(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesPartOrderbook20(t.Context(), "XBTUSDTM")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook100(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesPartOrderbook100(t.Context(), "XBTUSDTM")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTradeHistory(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesTradeHistory(t.Context(), "XBTUSDTM")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesInterestRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesInterestRate(t.Context(), "", time.Time{}, time.Time{}, false, false, 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	result, err := ku.GetFuturesInterestRate(t.Context(), futuresTradablePair.String(), time.Time{}, time.Time{}, false, false, 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesIndexList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesIndexList(t.Context(), "", time.Time{}, time.Time{}, false, false, 0, 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	result, err := ku.GetFuturesIndexList(t.Context(), futuresTradablePair.String(), time.Time{}, time.Time{}, false, false, 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesCurrentMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentMarkPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesCurrentMarkPrice(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPremiumIndex(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPremiumIndex(t.Context(), "", time.Time{}, time.Time{}, false, false, 0, 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesPremiumIndex(t.Context(), futuresTradablePair.String(), time.Time{}, time.Time{}, false, false, 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HourFuturesTransactionVolume(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	resp, err := ku.Get24HourFuturesTransactionVolume(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentFundingRate(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesCurrentFundingRate(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPublicFundingRate(t.Context(), "", time.Now().Add(-time.Hour*24*30), time.Now().Add(-time.Hour*5))
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetPublicFundingRate(t.Context(), futuresTradablePair.String(), time.Now().Add(-time.Hour*24*30), time.Now().Add(-time.Hour*5))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := ku.GetFuturesServerTime(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesServiceStatus(t *testing.T) {
	t.Parallel()
	result, err := ku.GetFuturesServiceStatus(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFuturesKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	data := []byte(`[1746518400000,1806.48,1806.48,1794.41,1794.41,1560]`)
	var target *FuturesKline
	err := json.Unmarshal(data, &target)
	require.NoError(t, err)
	require.NotNil(t, target)
	assert.Equal(t, FuturesKline{
		StartTime: types.Time(time.UnixMilli(1746518400000)),
		Open:      1806.48,
		High:      1806.48,
		Low:       1794.41,
		Close:     1794.41,
		Volume:    1560,
	}, *target)
}

func TestGetFuturesKline(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesKline(t.Context(), 0, "XBTUSDTM", time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrInvalidInterval)
	_, err = ku.GetFuturesKline(t.Context(), int64(kline.ThirtyMin.Duration().Seconds()), futuresTradablePair.String(), time.Time{}, time.Time{})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	_, err = ku.GetFuturesKline(t.Context(), int64(kline.ThirtyMin.Duration().Minutes()), "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := ku.GetFuturesKline(t.Context(), int64(kline.ThirtyMin.Duration().Minutes()), futuresTradablePair.String(), time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPostFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy"})
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{Side: "buy", Leverage: 1})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Leverage: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Leverage: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	// With Stop order configuration
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, errInvalidStopPriceType)

	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", StopPrice: 123456, TimeInForce: "", Size: 1, Price: 1000, Leverage: 1, VisibleSize: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Limit Orders
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "limit", Remark: "10", Leverage: 1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10", Price: 1000, Leverage: 1, VisibleSize: 0})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	result, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Size: 1, Price: 1000, Leverage: 1, VisibleSize: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Market Orders
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "market", Remark: "10", Leverage: 1,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "market", Remark: "10",
		Size: 1, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	result, err = ku.PostFuturesOrder(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de",
		Side:          "buy",
		Symbol:        futuresTradablePair,
		OrderType:     "limit",
		Remark:        "10",
		Stop:          "",
		StopPriceType: "",
		TimeInForce:   "",
		Size:          1,
		Price:         1000,
		StopPrice:     0,
		Leverage:      1,
		VisibleSize:   0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFillFuturesPostOrderArgumentFilter(t *testing.T) {
	t.Parallel()
	err := ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy"})
	require.ErrorIs(t, err, errInvalidLeverage)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{Side: "buy", Leverage: 1})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Leverage: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Leverage: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	// With Stop order configuration
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, errInvalidStopPriceType)

	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", StopPrice: 123456, TimeInForce: "", Size: 1, Price: 1000, Leverage: 1, VisibleSize: 0,
	})
	assert.NoError(t, err)

	// Limit Orders
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "limit", Remark: "10", Leverage: 1,
	})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10", Price: 1000, Leverage: 1, VisibleSize: 0})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Size: 1, Price: 1000, Leverage: 1, VisibleSize: 0,
	})
	assert.NoError(t, err)

	// Market Orders
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "market", Remark: "10", Leverage: 1,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "market", Remark: "10",
		Size: 0, Leverage: 1, VisibleSize: 0,
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	err = ku.FillFuturesPostOrderArgumentFilter(&FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de",
		Side:          "buy",
		Symbol:        futuresTradablePair,
		OrderType:     "limit",
		Remark:        "10",
		Stop:          "",
		StopPriceType: "",
		TimeInForce:   "",
		Size:          1,
		Price:         1000,
		StopPrice:     0,
		Leverage:      1,
		VisibleSize:   0,
	})
	assert.NoError(t, err)
}

func TestPostFuturesOrderTest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	response, err := ku.PostFuturesOrderTest(t.Context(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de",
		Side:          "buy",
		Symbol:        futuresTradablePair,
		OrderType:     "market",
		Remark:        "10",
		Stop:          "",
		StopPriceType: "",
		TimeInForce:   "",
		Size:          1,
		StopPrice:     0,
		Leverage:      1,
		VisibleSize:   0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestPlaceMultipleFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.PlaceMultipleFuturesOrders(t.Context(), []FuturesOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PlaceMultipleFuturesOrders(t.Context(), []FuturesOrderParam{
		{
			ClientOrderID: "5c52e11203aa677f33e491",
			Side:          "buy",
			Symbol:        futuresTradablePair,
			OrderType:     "limit",
			Price:         2150,
			Size:          2,
			Leverage:      1,
		},
		{
			ClientOrderID: "5c52e11203aa677f33e492",
			Side:          "buy",
			Symbol:        futuresTradablePair,
			OrderType:     "limit",
			Price:         32150,
			Size:          2,
			Leverage:      1,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelFuturesOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelFuturesOrderByOrderID(t.Context(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelFuturesOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelFuturesOrderByClientOrderID(t.Context(), "", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.CancelFuturesOrderByClientOrderID(t.Context(), futuresTradablePair.String(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelFuturesOrderByClientOrderID(t.Context(), futuresTradablePair.String(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelMultipleFuturesLimitOrders(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllFuturesStopOrders(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesOrders(t.Context(), "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUntriggeredFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetUntriggeredFuturesStopOrders(t.Context(), "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRecentCompletedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesRecentCompletedOrders(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOrderDetails(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesOrderDetails(t.Context(), "5cdfc138b21023a909e5ad55", "2212332")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOrderDetailsByClientID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOrderDetailsByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesOrderDetailsByClientOrderID(t.Context(), "eresc138b21023a909e5ad59")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesFills(t.Context(), "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRecentFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesRecentFills(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesOpenOrderStats(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOpenOrderStats(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesOpenOrderStats(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPosition(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPosition(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesPosition(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesPositionList(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAutoDepositMargin(t *testing.T) {
	t.Parallel()
	_, err := ku.SetAutoDepositMargin(t.Context(), "", true)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SetAutoDepositMargin(t.Context(), futuresTradablePair.String(), true)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaxWithdrawMargin(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMaxWithdrawMargin(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetMaxWithdrawMargin(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRemoveMarginManually(t *testing.T) {
	t.Parallel()
	_, err := ku.RemoveMarginManually(t.Context(), &WithdrawMarginResponse{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.RemoveMarginManually(t.Context(), &WithdrawMarginResponse{
		WithdrawAmount: 1,
	})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.RemoveMarginManually(t.Context(), &WithdrawMarginResponse{
		Symbol:         "ADAUSDTM",
		WithdrawAmount: 1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddMargin(t *testing.T) {
	t.Parallel()
	_, err := ku.AddMargin(t.Context(), "", "6200c9b83aecfb000152dasfdee", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.AddMargin(t.Context(), futuresTradablePair.String(), "6200c9b83aecfb000152dasfdee", 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesRiskLimitLevel(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesRiskLimitLevel(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateRiskLmitLevel(t *testing.T) {
	t.Parallel()
	_, err := ku.FuturesUpdateRiskLmitLevel(t.Context(), "", 2)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.FuturesUpdateRiskLmitLevel(t.Context(), futuresTradablePair.String(), 2)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesFundingHistory(t.Context(), "", 0, 0, true, true, time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesFundingHistory(t.Context(), futuresTradablePair.String(), 0, 0, true, true, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesAccountOverview(t.Context(), "BTC")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesTransactionHistory(t.Context(), currency.EMPTYCODE, "", 0, 0, true, time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateFuturesSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := ku.CreateFuturesSubAccountAPIKey(t.Context(), "", "passphrase", "", "", "subAccName")
	require.ErrorIs(t, err, errRemarkIsRequired)
	_, err = ku.CreateFuturesSubAccountAPIKey(t.Context(), "", "passphrase", "", "remark", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ku.CreateFuturesSubAccountAPIKey(t.Context(), "", "", "", "remark", "subAccName")
	require.ErrorIs(t, err, errInvalidPassPhraseInstance)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CreateFuturesSubAccountAPIKey(t.Context(), "", "passphrase", "", "remark", "subAccName")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferFuturesFundsToMainAccount(t *testing.T) {
	t.Parallel()
	var resp *TransferRes
	err := json.Unmarshal([]byte(transferFuturesFundsResponseJSON), &resp)
	assert.NoError(t, err)

	_, err = ku.TransferFuturesFundsToMainAccount(t.Context(), 0, currency.USDT, "MAIN")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.TransferFuturesFundsToMainAccount(t.Context(), 1, currency.EMPTYCODE, "MAIN")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.TransferFuturesFundsToMainAccount(t.Context(), 1, currency.ETH, "")
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.TransferFuturesFundsToMainAccount(t.Context(), 1, currency.USDT, "MAIN")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferFundsToFuturesAccount(t *testing.T) {
	t.Parallel()
	err := ku.TransferFundsToFuturesAccount(t.Context(), 0, currency.USDT, "MAIN")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	err = ku.TransferFundsToFuturesAccount(t.Context(), 1, currency.EMPTYCODE, "MAIN")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	err = ku.TransferFundsToFuturesAccount(t.Context(), 1, currency.USDT, "")
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err = ku.TransferFundsToFuturesAccount(t.Context(), 1, currency.USDT, "MAIN")
	assert.NoError(t, err)
}

func TestGetFuturesTransferOutList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTransferOutList(t.Context(), currency.EMPTYCODE, "", time.Time{}, time.Time{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesTransferOutList(t.Context(), currency.USDT, "", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	assetTypes := ku.GetAssetTypes(true)
	for _, assetType := range assetTypes {
		result, err := ku.FetchTradablePairs(t.Context(), assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	var result *orderbook.Book
	var err error
	for assetType, tp := range assertToTradablePairMap {
		result, err = ku.UpdateOrderbook(t.Context(), tp, assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for _, a := range ku.GetAssetTypes(true) {
		err := ku.UpdateTickers(t.Context(), a)
		assert.NoError(t, err)

		pairs, err := ku.GetEnabledPairs(a)
		assert.NoError(t, err)
		assert.NotEmpty(t, pairs)

		for _, p := range pairs {
			tick, err := ticker.GetTicker(ku.Name, p, a)
			if assert.NoError(t, err) {
				assert.Positivef(t, tick.Last, "%s %s Tick Last should be positive", a, p)
				assert.NotEmptyf(t, tick.Pair, "%s %s Tick Pair should not be empty", a, p)
				assert.Equalf(t, ku.Name, tick.ExchangeName, "ExchangeName should be correct")
				assert.Equalf(t, a, tick.AssetType, "AssetType should be correct")
				assert.NotEmptyf(t, tick.LastUpdated, "%s %s Tick LastUpdated should not be empty", a, p)
			}
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	var result *ticker.Price
	var err error
	for assetType, tp := range assertToTradablePairMap {
		result, err = ku.UpdateTicker(t.Context(), tp, assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48)
	endTime := time.Now().Add(-time.Hour * 3)
	var result *kline.Item
	var err error
	for assetType, tp := range assertToTradablePairMap {
		result, err = ku.GetHistoricCandles(t.Context(), tp, assetType, kline.OneHour, startTime, endTime)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48 * 10)
	endTime := time.Now().Add(-time.Hour * 1)
	result, err := ku.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.FifteenMin, startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.FifteenMin, startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetHistoricCandlesExtended(t.Context(), marginTradablePair, asset.Margin, kline.FifteenMin, startTime, endTime)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.Margin} {
		result, err := ku.GetServerTime(t.Context(), a)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	var result []trade.Data
	var err error
	for assetType, tp := range assertToTradablePairMap {
		result, err = ku.GetRecentTrades(t.Context(), tp, assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     []currency.Pair{futuresTradablePair},
		AssetType: asset.Binary,
		Side:      order.AnySide,
	}
	_, err := ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	getOrdersRequest.AssetType = asset.Futures
	result, err := ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.AssetType = asset.Spot
	getOrdersRequest.Pairs = []currency.Pair{}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{spotTradablePair}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Type = order.OCO
	getOrdersRequest.Pairs = []currency.Pair{}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{spotTradablePair}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.AssetType = asset.Margin
	getOrdersRequest.Type = order.Stop
	getOrdersRequest.Pairs = []currency.Pair{spotTradablePair}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Pairs = []currency.Pair{}
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.Type = order.StopLimit
	getOrdersRequest.MarginType = margin.Multi
	result, err = ku.GetOrderHistory(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest order.MultiOrderRequest

	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err)
	getOrdersRequest = order.MultiOrderRequest{
		Pairs:     enabledPairs,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}

	getOrdersRequest.Type = order.OptimalLimit
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	getOrdersRequest.Type = order.Limit
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.Market
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err)

	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs,
		AssetType: asset.Margin,
		Side:      order.Buy,
	}
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.Market
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.OCO
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.StopMarket
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.Stop
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	assert.NoError(t, err)

	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs,
		AssetType: asset.Futures,
		Side:      order.Buy,
	}
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)

	getOrdersRequest.Type = order.StopLimit
	_, err = ku.GetActiveOrders(t.Context(), &getOrdersRequest)
	assert.NoError(t, err)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	assetTypes := ku.CurrencyPairs.GetAssetTypes(true)
	for _, at := range assetTypes {
		err := ku.ValidateCredentials(t.Context(), at)
		assert.NoError(t, err)
	}
}

func TestGetInstanceServers(t *testing.T) {
	t.Parallel()
	result, err := ku.GetInstanceServers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAuthenticatedServersInstances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAuthenticatedInstanceServers(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPushData(t *testing.T) {
	t.Parallel()
	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	ku.SetCredentials("mock", "test", "test", "", "", "")
	ku.API.AuthenticatedSupport = true
	ku.API.AuthenticatedWebsocketSupport = true
	testexch.FixtureToDataHandler(t, "testdata/wsHandleData.json", ku.wsHandleData)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	// Pairs overlap for spot/margin tests:
	// Only in Spot: BTC-USDT, ETH-USDT
	// In Both: ETH-BTC, LTC-USDT
	// Only in Margin: TRX-BTC, SOL-USDC
	pairs := map[string]currency.Pairs{}
	for a, ss := range map[string][]string{
		"spot":    {"BTC-USDT", "ETH-BTC", "ETH-USDT", "LTC-USDT"},
		"margin":  {"ETH-BTC", "LTC-USDT", "SOL-USDC", "TRX-BTC"},
		"futures": {"ETHUSDCM", "SOLUSDTM", "XBTUSDCM"},
	} {
		for _, s := range ss {
			p, err := currency.NewPairFromString(s)
			require.NoError(t, err, "NewPairFromString must not error")
			pairs[a] = pairs[a].Add(p)
		}
	}
	pairs["both"] = common.SortStrings(pairs["spot"].Add(pairs["margin"]...))

	exp := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: pairs["both"], QualifiedChannel: "/market/ticker:" + pairs["both"].Join()},
		{Channel: subscription.TickerChannel, Asset: asset.Futures, Pairs: pairs["futures"], QualifiedChannel: "/contractMarket/tickerV2:" + pairs["futures"].Join()},
		{
			Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: pairs["both"], QualifiedChannel: "/spotMarket/level2Depth5:" + pairs["both"].Join(),
			Interval: kline.HundredMilliseconds,
		},
		{
			Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: pairs["futures"], QualifiedChannel: "/contractMarket/level2Depth5:" + pairs["futures"].Join(),
			Interval: kline.HundredMilliseconds,
		},
		{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: pairs["both"], QualifiedChannel: "/market/match:" + pairs["both"].Join()},
	}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	ku.Websocket.SetCanUseAuthenticatedEndpoints(true)

	var loanPairs currency.Pairs
	loanCurrs := common.SortStrings(pairs["both"].GetCurrencies())
	for _, c := range loanCurrs {
		loanPairs = append(loanPairs, currency.Pair{Base: c})
	}

	exp = append(exp, subscription.List{
		{Asset: asset.Futures, Channel: futuresTradeOrderChannel, QualifiedChannel: "/contractMarket/tradeOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresStopOrdersLifecycleEventChannel, QualifiedChannel: "/contractMarket/advancedOrders", Pairs: pairs["futures"]},
		{Asset: asset.Futures, Channel: futuresAccountBalanceEventChannel, QualifiedChannel: "/contractAccount/wallet", Pairs: pairs["futures"]},
		{Asset: asset.Margin, Channel: marginPositionChannel, QualifiedChannel: "/margin/position", Pairs: pairs["margin"]},
		{Asset: asset.Margin, Channel: marginLoanChannel, QualifiedChannel: "/margin/loan:" + loanCurrs.Join(), Pairs: loanPairs},
		{Channel: accountBalanceChannel, QualifiedChannel: "/account/balance"},
	}...)

	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)
}

func TestGenerateTickerAllSub(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:11], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot}}
	exp := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, QualifiedChannel: "/market/ticker:all", Pairs: avail[:11]},
	}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions with Auth must not error")
	testsubs.EqualLists(t, exp, subs)
}

// TestGenerateOtherSubscriptions exercises non-default subscriptions
func TestGenerateOtherSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	subs := subscription.List{
		{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.FourHour},
		{Channel: marketSnapshotChannel, Asset: asset.Spot},
	}

	for _, s := range subs {
		ku.Features.Subscriptions = subscription.List{s}
		got, err := ku.generateSubscriptions()
		assert.NoError(t, err, "generateSubscriptions should not error")
		require.Len(t, got, 1, "Must generate just one sub")
		assert.NotEmpty(t, got[0].QualifiedChannel, "Qualified Channel should not be empty")
		if got[0].Channel == subscription.CandlesChannel {
			assert.Equal(t, "/market/candles:BTC-USDT_4hour,ETH-BTC_4hour,ETH-USDT_4hour,LTC-USDT_4hour", got[0].QualifiedChannel, "QualifiedChannel should be correct")
		}
	}
}

// TestGenerateMarginSubscriptions is a regression test for #1755 and ensures margin subscriptions work without spot subs
func TestGenerateMarginSubscriptions(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes

	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error storing spot pairs")
	avail = common.SortStrings(avail)
	err = ku.CurrencyPairs.StorePairs(asset.Margin, avail[:6], true)
	require.NoError(t, err, "StorePairs must not error storing margin pairs")
	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:3], true)
	require.NoError(t, err, "StorePairs must not error storing spot pairs")

	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Margin}}
	subs, err := ku.Features.Subscriptions.ExpandTemplates(ku)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.Len(t, subs, 1, "Must generate just one sub")
	assert.Equal(t, asset.Margin, subs[0].Asset, "Asset should be correct")
	assert.Equal(t, "/market/ticker:"+avail[:6].Join(), subs[0].QualifiedChannel, "QualifiedChannel should be correct")

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, err, "SetAssetEnabled must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.Features.Subscriptions.ExpandTemplates(ku)
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Margin is disabled")
	require.NotEmpty(t, subs, "ExpandTemplates must return some subs")

	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "SetAssetEnabled Margin must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled Spot must not error")
	require.NoError(t, ku.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled Futures must not error")
	ku.Features.Subscriptions = subscription.List{{Channel: subscription.TickerChannel, Asset: asset.All}}
	subs, err = ku.Features.Subscriptions.ExpandTemplates(ku)
	require.NoError(t, err, "mergeMarginPairs must not cause errAssetRecords by adding an empty asset when Spot is disabled")
	require.NotEmpty(t, subs, "ExpandTemplates must return some subs")
}

// TestCheckSubscriptions ensures checkSubscriptions upgrades user config correctly
func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()

	ku := &Kucoin{ //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
		Base: exchange.Base{
			Config: &config.Exchange{
				Features: &config.FeaturesConfig{
					Subscriptions: subscription.List{
						{Enabled: true, Channel: "ticker"},
						{Enabled: true, Channel: "allTrades"},
						{Enabled: true, Channel: "orderbook", Interval: kline.HundredMilliseconds},
						{Enabled: true, Channel: "/contractMarket/tickerV2:%s"},
						{Enabled: true, Channel: "/contractMarket/level2Depth50:%s"},
						{Enabled: true, Channel: "/margin/fundingBook:%s", Authenticated: true},
						{Enabled: true, Channel: "/account/balance", Authenticated: true},
						{Enabled: true, Channel: "/margin/position", Authenticated: true},
						{Enabled: true, Channel: "/margin/loan:%s", Authenticated: true},
						{Enabled: true, Channel: "/contractMarket/tradeOrders", Authenticated: true},
						{Enabled: true, Channel: "/contractMarket/advancedOrders", Authenticated: true},
						{Enabled: true, Channel: "/contractAccount/wallet", Authenticated: true},
					},
				},
			},
			Features: exchange.Features{},
		},
	}

	ku.checkSubscriptions()
	testsubs.EqualLists(t, defaultSubscriptions, ku.Features.Subscriptions)
	testsubs.EqualLists(t, defaultSubscriptions, ku.Config.Features.Subscriptions)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAvailableTransferChains(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = ku.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var result *order.Detail
	var err error
	result, err = ku.GetOrderInfo(t.Context(), "54541241349183409134134133", futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetOrderInfo(t.Context(), "54541241349183409134134133", spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.GetOrderInfo(t.Context(), "54541241349183409134134133", marginTradablePair, asset.Margin)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddress(t.Context(), currency.BTC, "", "")
	assert.Truef(t, err == nil || errors.Is(err, errNoDepositAddress), "GetDepositAddress should not error: %s", err)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := ku.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: ku.Name,
		Amount:   0.00000000001,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	})
	assert.ErrorContains(t, err, withdraw.ErrStrNoCurrencySet)
	_, err = ku.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: ku.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto:   withdraw.CryptoRequest{},
	})
	assert.ErrorContains(t, err, "address cannot be empty")
	_, err = ku.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: ku.Name,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	})
	assert.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: ku.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	orderSubmission := &order.Submit{
		Pair:          futuresTradablePair,
		Exchange:      ku.Name,
		Side:          order.Bid,
		Type:          order.Limit,
		Price:         1,
		Amount:        100000,
		ClientOrderID: "myOrder",
		AssetType:     asset.Options,
	}
	_, err := ku.SubmitOrder(t.Context(), orderSubmission)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	orderSubmission.AssetType = asset.Futures
	orderSubmission.Pair = futuresTradablePair
	result, err := ku.SubmitOrder(t.Context(), orderSubmission)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	orderSubmission.Type = order.OCO
	orderSubmission.TriggerPrice = 50000
	orderSubmission.TriggerPriceType = order.LastPrice
	_, err = ku.SubmitOrder(t.Context(), orderSubmission)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	// Spot order creation tests
	spotOrderSubmission := &order.Submit{
		Side:          order.Buy,
		AssetType:     asset.Spot,
		Pair:          spotTradablePair,
		Type:          order.Limit,
		Price:         1,
		Amount:        100000,
		ClientOrderID: "myOrder",
	}
	result, err = ku.SubmitOrder(t.Context(), spotOrderSubmission)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	spotOrderSubmission.Type = order.StopLimit
	spotOrderSubmission.RiskManagementModes = order.RiskManagementModes{
		StopEntry: order.RiskManagement{
			Enabled:          true,
			Price:            1234,
			TriggerPriceType: order.LastPrice,
		},
	}
	result, err = ku.SubmitOrder(t.Context(), spotOrderSubmission)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	spotOrderSubmission.Type = order.OCO
	spotOrderSubmission.Side = order.Sell
	spotOrderSubmission.RiskManagementModes.TakeProfit = order.RiskManagement{
		Enabled:          true,
		Price:            1334,
		TriggerPriceType: order.LastPrice,
	}
	spotOrderSubmission.RiskManagementModes.StopLoss = order.RiskManagement{
		Enabled:          true,
		Price:            1234,
		TriggerPriceType: order.LastPrice,
	}
	result, err = ku.SubmitOrder(t.Context(), spotOrderSubmission)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	spotOrderSubmission.Type = order.ConditionalStop
	_, err = ku.SubmitOrder(t.Context(), spotOrderSubmission)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	// Margin order creation tests
	marginOrderSubmission := &order.Submit{
		Side:          order.Buy,
		AssetType:     asset.Margin,
		Pair:          marginTradablePair,
		Type:          order.Limit,
		Price:         1,
		Amount:        100000,
		ClientOrderID: "myOrder",
	}
	result, err = ku.SubmitOrder(t.Context(), marginOrderSubmission)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      futuresTradablePair,
		AssetType: asset.Options,
	}
	err := ku.CancelOrder(t.Context(), orderCancellation)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	orderCancellation.AssetType = asset.Futures
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.OrderID = ""
	orderCancellation.ClientOrderID = "12345"
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.Pair = currency.EMPTYPAIR
	err = ku.CancelOrder(t.Context(), orderCancellation)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	orderCancellation.AssetType = asset.Spot
	orderCancellation.Pair = spotTradablePair
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.Type = order.OCO
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.OrderID = "12345"
	orderCancellation.ClientOrderID = ""
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.Type = order.Stop
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.OrderID = ""
	orderCancellation.ClientOrderID = "12345"
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.Type = order.Limit
	orderCancellation.AssetType = asset.Margin
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.ClientOrderID = ""
	orderCancellation.OrderID = "12345"
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.AssetType = asset.Margin
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)

	orderCancellation.ClientOrderID = ""
	orderCancellation.OrderID = "12345"
	orderCancellation.AssetType = asset.Margin
	err = ku.CancelOrder(t.Context(), orderCancellation)
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType:  asset.Futures,
		MarginType: margin.Isolated,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	_, err = ku.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType:  asset.Margin,
		MarginType: margin.Isolated,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	result, err = ku.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType: asset.Spot,
		Type:      order.OCO,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType: asset.Spot,
		Type:      order.Stop,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.CancelAllOrders(t.Context(), &order.Cancel{
		AssetType: asset.Spot,
		Type:      order.StopLimit,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

const (
	subUserResponseJSON              = `{"userId":"635002438793b80001dcc8b3", "uid":62356, "subName":"margin01", "status":2, "type":4, "access":"Margin", "createdAt":1666187844000, "remarks":null }`
	transferFuturesFundsResponseJSON = `{"applyId": "620a0bbefeaa6a000110e833", "bizNo": "620a0bbefeaa6a000110e832", "payAccountType": "CONTRACT", "payTag": "DEFAULT", "remark": "", "recAccountType": "MAIN", "recTag": "DEFAULT", "recRemark": "", "recSystem": "KUCOIN", "status": "PROCESSING", "currency": "USDT", "amount": "0.001", "fee": "0", "sn": 889048787670001, "reason": "", "createdAt": 1644825534000, "updatedAt": 1644825534000}`
	modifySubAccountSpotAPIs         = `{"subName": "AAAAAAAAAA0007", "remark": "remark", "apiKey": "630325e0e750870001829864", "apiSecret": "110f31fc-61c5-4baf-a29f-3f19a62bbf5d", "passphrase": "passphrase", "permission": "General", "ipWhitelist": "", "createdAt": 1661150688000}`
)

func TestCreateSubUser(t *testing.T) {
	t.Parallel()
	var resp *SubAccount
	err := json.Unmarshal([]byte(subUserResponseJSON), &resp)
	assert.NoError(t, err)
	_, err = ku.CreateSubUser(t.Context(), "", "Subaccount-1", "", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ku.CreateSubUser(t.Context(), "Subaccount-2", "", "", "")
	require.ErrorIs(t, err, errInvalidPassPhraseInstance)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CreateSubUser(t.Context(), "Subaccount-2", "Subaccount-1", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountSpotAPIList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetSubAccountSpotAPIList(t.Context(), "", "")
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetSubAccountSpotAPIList(t.Context(), "Sam", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateSpotAPIsForSubAccount(t *testing.T) {
	t.Parallel()
	_, err := ku.CreateSpotAPIsForSubAccount(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "the-remark",
	})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ku.CreateSpotAPIsForSubAccount(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "",
		Remark:         "the-remark",
	})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance)
	_, err = ku.CreateSpotAPIsForSubAccount(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "",
	})
	require.ErrorIs(t, err, errRemarkIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CreateSpotAPIsForSubAccount(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "the-remark",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifySubAccountSpotAPIs(t *testing.T) {
	t.Parallel()
	var resp SpotAPISubAccount
	err := json.Unmarshal([]byte(modifySubAccountSpotAPIs), &resp)
	assert.NoError(t, err)
	_, err = ku.ModifySubAccountSpotAPIs(t.Context(), &SpotAPISubAccountParams{
		APIKey: "65e7f22077172b0001f9ee41", SubAccountName: "", Passphrase: "mysecretPassphrase123",
	})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ku.ModifySubAccountSpotAPIs(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1", Passphrase: "mysecretPassphrase123",
	})
	require.ErrorIs(t, err, errAPIKeyRequired)
	_, err = ku.ModifySubAccountSpotAPIs(t.Context(), &SpotAPISubAccountParams{
		APIKey: "65e7f22077172b0001f9ee41", SubAccountName: "gocryptoTrader1", Passphrase: "",
	})
	require.ErrorIs(t, err, errInvalidPassPhraseInstance)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.ModifySubAccountSpotAPIs(t.Context(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		APIKey:         apiKey,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDeleteSubAccountSpotAPI(t *testing.T) {
	t.Parallel()
	_, err := ku.DeleteSubAccountSpotAPI(t.Context(), "65e7f22077172b0001f9ee41", "", "the-passphrase")
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ku.DeleteSubAccountSpotAPI(t.Context(), "", "gocryptoTrader1", "the-passphrase")
	require.ErrorIs(t, err, errAPIKeyRequired)
	_, err = ku.DeleteSubAccountSpotAPI(t.Context(), "65e7f22077172b0001f9ee41", "gocryptoTrader1", "")
	require.ErrorIs(t, err, errInvalidPassPhraseInstance)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.DeleteSubAccountSpotAPI(t.Context(), apiKey, "gocryptoTrader1", "the-passphrase")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserInfoOfAllSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetUserInfoOfAllSubAccounts(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPaginatedListOfSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetPaginatedListOfSubAccounts(t.Context(), 1, 100)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAccountFundingHistory(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func getFirstTradablePairOfAssets(ctx context.Context) {
	if err := ku.UpdateTradablePairs(ctx, true); err != nil {
		log.Fatalf("Kucoin error while updating tradable pairs. %v", err)
	}
	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Spot)
	}
	spotTradablePair = enabledPairs[0]
	enabledPairs, err = ku.GetEnabledPairs(asset.Margin)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Margin)
	}
	marginTradablePair = enabledPairs[0]
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Futures)
	}
	futuresTradablePair = enabledPairs[0]
	futuresTradablePair.Delimiter = ""
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	assetTypes := ku.GetAssetTypes(true)
	for _, assetType := range assetTypes {
		result, err := ku.UpdateAccountInfo(t.Context(), assetType)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

const (
	orderbookLevel5PushData = `{"type": "message","topic": "/spotMarket/level2Depth50:BTC-USDT","subject": "level2","data": {"asks": [["21621.7","3.03206193"],["21621.8","1.00048239"],["21621.9","0.29558803"],["21622","0.0049653"],["21622.4","0.06177582"],["21622.9","0.39664116"],["21623.7","0.00803466"],["21624.2","0.65405"],["21624.3","0.34661426"],["21624.6","0.00035589"],["21624.9","0.61282048"],["21625.2","0.16421424"],["21625.4","0.90107014"],["21625.5","0.73484442"],["21625.9","0.04"],["21626.2","0.28569324"],["21626.4","0.18403701"],["21627.1","0.06503999"],["21627.2","0.56105832"],["21627.7","0.10649999"],["21628.1","2.66459953"],["21628.2","0.32"],["21628.5","0.27605551"],["21628.6","1.59482596"],["21628.9","0.16"],["21629.8","0.08"],["21630","0.04"],["21631.6","0.1"],["21631.8","0.0920185"],["21633.6","0.00447983"],["21633.7","0.00015044"],["21634.3","0.32193346"],["21634.4","0.00004"],["21634.5","0.1"],["21634.6","0.0002865"],["21635.6","0.12069941"],["21635.8","0.00117158"],["21636","0.00072816"],["21636.5","0.98611492"],["21636.6","0.00007521"],["21637.2","0.00699999"],["21637.6","0.00017129"],["21638","0.00013035"],["21638.1","0.05"],["21638.5","0.92427"],["21639.2","1.84998696"],["21639.3","0.04827233"],["21640","0.56255996"],["21640.9","0.8"],["21641","0.12"]],"bids": [["21621.6","0.40949924"],["21621.5","0.27703279"],["21621.3","0.04"],["21621.1","0.0086"],["21621","0.6653104"],["21620.9","0.35435999"],["21620.8","0.37224309"],["21620.5","0.416184"],["21620.3","0.24"],["21619.6","0.13883999"],["21619.5","0.21053355"],["21618.7","0.2"],["21618.6","0.001"],["21618.5","0.2258151"],["21618.4","0.06503999"],["21618.3","0.00370056"],["21618","0.12067842"],["21617.7","0.34844131"],["21617.6","0.92845495"],["21617.5","0.66460535"],["21617","0.01"],["21616.7","0.0004624"],["21616.4","0.02"],["21615.6","0.04828251"],["21615","0.59065665"],["21614.4","0.00227"],["21614.3","0.1"],["21613","0.32193346"],["21612.9","0.0028638"],["21612.6","0.1"],["21612.5","0.92539"],["21610.7","0.08208616"],["21610.6","0.00967666"],["21610.3","0.12"],["21610.2","0.00611126"],["21609.9","0.00226344"],["21609.8","0.00315812"],["21609.1","0.00547218"],["21608.6","0.09793157"],["21608.5","0.00437793"],["21608.4","1.85013454"],["21608.1","0.00366647"],["21607.9","0.00611595"],["21607.7","0.83263561"],["21607.6","0.00368919"],["21607.5","0.00280702"],["21607.1","0.66610849"],["21606.8","0.00364164"],["21606.2","0.80351642"],["21605.7","0.075"]],"timestamp": 1676319280783}}`
	wsOrderbookData         = `{"changes":{"asks":[["21621.7","3.03206193",""],["21621.8","1.00048239",""],["21621.9","0.29558803",""],["21622","0.0049653",""],["21622.4","0.06177582",""],["21622.9","0.39664116",""],["21623.7","0.00803466",""],["21624.2","0.65405",""],["21624.3","0.34661426",""],["21624.6","0.00035589",""],["21624.9","0.61282048",""],["21625.2","0.16421424",""],["21625.4","0.90107014",""],["21625.5","0.73484442",""],["21625.9","0.04",""],["21626.2","0.28569324",""],["21626.4","0.18403701",""],["21627.1","0.06503999",""],["21627.2","0.56105832",""],["21627.7","0.10649999",""],["21628.1","2.66459953",""],["21628.2","0.32",""],["21628.5","0.27605551",""],["21628.6","1.59482596",""],["21628.9","0.16",""],["21629.8","0.08",""],["21630","0.04",""],["21631.6","0.1",""],["21631.8","0.0920185",""],["21633.6","0.00447983",""],["21633.7","0.00015044",""],["21634.3","0.32193346",""],["21634.4","0.00004",""],["21634.5","0.1",""],["21634.6","0.0002865",""],["21635.6","0.12069941",""],["21635.8","0.00117158",""],["21636","0.00072816",""],["21636.5","0.98611492",""],["21636.6","0.00007521",""],["21637.2","0.00699999",""],["21637.6","0.00017129",""],["21638","0.00013035",""],["21638.1","0.05",""],["21638.5","0.92427",""],["21639.2","1.84998696",""],["21639.3","0.04827233",""],["21640","0.56255996",""],["21640.9","0.8",""],["21641","0.12",""]],"bids":[["21621.6","0.40949924",""],["21621.5","0.27703279",""],["21621.3","0.04",""],["21621.1","0.0086",""],["21621","0.6653104",""],["21620.9","0.35435999",""],["21620.8","0.37224309",""],["21620.5","0.416184",""],["21620.3","0.24",""],["21619.6","0.13883999",""],["21619.5","0.21053355",""],["21618.7","0.2",""],["21618.6","0.001",""],["21618.5","0.2258151",""],["21618.4","0.06503999",""],["21618.3","0.00370056",""],["21618","0.12067842",""],["21617.7","0.34844131",""],["21617.6","0.92845495",""],["21617.5","0.66460535",""],["21617","0.01",""],["21616.7","0.0004624",""],["21616.4","0.02",""],["21615.6","0.04828251",""],["21615","0.59065665",""],["21614.4","0.00227",""],["21614.3","0.1",""],["21613","0.32193346",""],["21612.9","0.0028638",""],["21612.6","0.1",""],["21612.5","0.92539",""],["21610.7","0.08208616",""],["21610.6","0.00967666",""],["21610.3","0.12",""],["21610.2","0.00611126",""],["21609.9","0.00226344",""],["21609.8","0.00315812",""],["21609.1","0.00547218",""],["21608.6","0.09793157",""],["21608.5","0.00437793",""],["21608.4","1.85013454",""],["21608.1","0.00366647",""],["21607.9","0.00611595",""],["21607.7","0.83263561",""],["21607.6","0.00368919",""],["21607.5","0.00280702",""],["21607.1","0.66610849",""],["21606.8","0.00364164",""],["21606.2","0.80351642",""],["21605.7","0.075",""]]},"sequenceEnd":1676319280783,"sequenceStart":0,"symbol":"BTC-USDT","time":1676319280783}`
)

func TestProcessOrderbook(t *testing.T) {
	t.Parallel()
	response := &WsOrderbook{}
	err := json.Unmarshal([]byte(wsOrderbookData), &response)
	assert.NoError(t, err)
	ku.setupOrderbookManager(t.Context())
	result, err := ku.updateLocalBuffer(response, asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	err = ku.processOrderbook([]byte(orderbookLevel5PushData), "BTC-USDT", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	err = ku.wsHandleData(t.Context(), []byte(orderbookLevel5PushData))
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProcessMarketSnapshot(t *testing.T) {
	t.Parallel()
	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	testexch.FixtureToDataHandler(t, "testdata/wsMarketSnapshot.json", ku.wsHandleData)
	close(ku.Websocket.DataHandler)
	assert.Len(t, ku.Websocket.DataHandler, 4, "Should see 4 tickers")
	seenAssetTypes := map[asset.Item]int{}
	for resp := range ku.Websocket.DataHandler {
		switch v := resp.(type) {
		case *ticker.Price:
			switch len(ku.Websocket.DataHandler) {
			case 3:
				assert.Equal(t, asset.Margin, v.AssetType, "AssetType")
				assert.Equal(t, time.UnixMilli(1700555342007), v.LastUpdated, "datetime")
				assert.Equal(t, 0.004445, v.High, "high")
				assert.Equal(t, 0.004415, v.Last, "lastTradedPrice")
				assert.Equal(t, 0.004191, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("TRX", "BTC", "-"), v.Pair, "symbol")
				assert.Equal(t, 13097.3357, v.Volume, "volume")
				assert.Equal(t, 57.44552981, v.QuoteVolume, "volValue")
			case 2, 1:
				assert.Equal(t, time.UnixMilli(1700555340197), v.LastUpdated, "datetime")
				assert.Contains(t, []asset.Item{asset.Spot, asset.Margin}, v.AssetType, "AssetType is Spot or Margin")
				seenAssetTypes[v.AssetType]++
				assert.Equal(t, 1, seenAssetTypes[v.AssetType], "Each Asset Type is sent only once per unique snapshot")
				assert.Equal(t, 0.054846, v.High, "high")
				assert.Equal(t, 0.053778, v.Last, "lastTradedPrice")
				assert.Equal(t, 0.05364, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("ETH", "BTC", "-"), v.Pair, "symbol")
				assert.Equal(t, 2958.3139116, v.Volume, "volume")
				assert.Equal(t, 160.7847672784213, v.QuoteVolume, "volValue")
			case 0:
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, time.UnixMilli(1700555342151), v.LastUpdated, "datetime")
				assert.Equal(t, 37750.0, v.High, "high")
				assert.Equal(t, 37366.8, v.Last, "lastTradedPrice")
				assert.Equal(t, 36700.0, v.Low, "low")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "symbol")
				assert.Equal(t, 2900.37846402, v.Volume, "volume")
				assert.Equal(t, 108210331.34015164, v.QuoteVolume, "volValue")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Got unexpected data: %T %v", v, v)
		}
	}
}

// TestSubscribeBatches ensures that endpoints support batching, contrary to kucoin api docs
func TestSubscribeBatches(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	ku.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, ku)

	ku.Features.Subscriptions = subscription.List{
		{Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Asset: asset.Futures, Channel: subscription.TickerChannel},
		{Asset: asset.Spot, Channel: marketSnapshotChannel},
	}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, len(ku.Features.Subscriptions), "Must generate batched subscriptions")

	err = ku.Subscribe(subs)
	assert.NoError(t, err, "Subscribe to small batches should not error")
}

// TestSubscribeTickerAll ensures that ticker subscriptions switch to using all and it works

// TestSubscribeBatchLimit exercises the kucoin batch limits of 400 per connection
// Ensures batching of 100 pairs and the connection symbol limit is still 400 at Kucoin's end
func TestSubscribeBatchLimit(t *testing.T) {
	t.Parallel()

	const expectedLimit = 400

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	ku.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, ku)

	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")

	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:expectedLimit], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.AllTradesChannel}}
	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 4, "Must get 4 subs")

	err = ku.Subscribe(subs)
	require.NoError(t, err, "Subscribe must not error")

	err = ku.Unsubscribe(subs)
	require.NoError(t, err, "Unsubscribe must not error")

	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:expectedLimit+20], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.AllTradesChannel}}
	subs, err = ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 5, "Must get 5 subs")

	err = ku.Subscribe(subs)
	require.Error(t, err, "Subscribe must error")
	assert.ErrorContains(t, err, "exceed max subscription count limitation of 400 per session", "Subscribe to MarketSnapshot should error above connection symbol limit")
}

func TestSubscribeTickerAll(t *testing.T) {
	t.Parallel()

	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	ku.Features.Subscriptions = subscription.List{}
	testexch.SetupWs(t, ku)

	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")

	err = ku.CurrencyPairs.StorePairs(asset.Spot, avail[:500], true)
	require.NoError(t, err, "StorePairs must not error")

	ku.Features.Subscriptions = subscription.List{{Asset: asset.Spot, Channel: subscription.TickerChannel}}

	subs, err := ku.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "Must generate one subscription")
	assert.Equal(t, "/market/ticker:all", subs[0].QualifiedChannel, "QualifiedChannel should be correct")

	err = ku.Subscribe(subs)
	assert.NoError(t, err, "Subscribe to should not error")
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	err := ku.SeedLocalCache(t.Context(), marginTradablePair, asset.Margin)
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = ku.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := ku.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := ku.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	req := &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewBTCUSD(),
	}
	_, err = ku.GetLatestFundingRates(t.Context(), req)
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req = &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.XBT, currency.USDTM),
	}
	resp, err := ku.GetLatestFundingRates(t.Context(), req)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)

	req = &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.EMPTYPAIR,
	}
	resp, err = ku.GetLatestFundingRates(t.Context(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := ku.IsPerpetualFutureCurrency(asset.Spot, currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.EOS))
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.USDTM))
	assert.NoError(t, err)
	assert.True(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.USDM))
	assert.NoError(t, err)
	assert.True(t, is)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := ku.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	req := &margin.PositionChangeRequest{}
	_, err = ku.ChangePositionMargin(t.Context(), req)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	req.Asset = asset.Futures
	_, err = ku.ChangePositionMargin(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = currency.NewPair(currency.XBT, currency.USDTM)
	_, err = ku.ChangePositionMargin(t.Context(), req)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.MarginType = margin.Isolated
	result, err := ku.ChangePositionMargin(t.Context(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	req.NewAllocatedMargin = 1337
	result, err = ku.ChangePositionMargin(t.Context(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	req := &futures.PositionSummaryRequest{}
	_, err = ku.GetFuturesPositionSummary(t.Context(), req)
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req.Asset = asset.Futures
	_, err = ku.GetFuturesPositionSummary(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.Pair = currency.NewPair(currency.XBT, currency.USDTM)
	result, err := ku.GetFuturesPositionSummary(t.Context(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	req := &futures.PositionsRequest{}
	_, err = ku.GetFuturesPositionOrders(t.Context(), req)
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req.Asset = asset.Futures
	_, err = ku.GetFuturesPositionOrders(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pairs = currency.Pairs{
		currency.NewPair(currency.XBT, currency.USDTM),
	}
	_, err = ku.GetFuturesPositionOrders(t.Context(), req)
	require.ErrorIs(t, err, common.ErrDateUnset)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.EndDate = time.Now()
	req.StartDate = req.EndDate.Add(-time.Hour * 24 * 7)
	_, err = ku.GetFuturesPositionOrders(t.Context(), req)
	assert.NoError(t, err)

	req.StartDate = req.EndDate.Add(-time.Hour * 24 * 30)
	_, err = ku.GetFuturesPositionOrders(t.Context(), req)
	require.ErrorIs(t, err, futures.ErrOrderHistoryTooLarge)

	req.RespectOrderHistoryLimits = true
	result, err := ku.GetFuturesPositionOrders(t.Context(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := ku.UpdateOrderExecutionLimits(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assets := []asset.Item{asset.Spot, asset.Futures, asset.Margin}
	for x := range assets {
		err = ku.UpdateOrderExecutionLimits(t.Context(), assets[x])
		assert.NoError(t, err)

		enabled, err := ku.GetEnabledPairs(assets[x])
		assert.NoError(t, err)

		for y := range enabled {
			lim, err := ku.GetOrderExecutionLimits(assets[x], enabled[y])
			assert.NoErrorf(t, err, "%v %s %v", err, enabled[y], assets[x])
			assert.NotEmptyf(t, lim, "limit cannot be empty")
		}
	}
}

func BenchmarkIntervalToString(b *testing.B) {
	for b.Loop() {
		result, err := IntervalToString(kline.OneWeek)
		assert.NoError(b, err)
		assert.NotNil(b, result)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	ku := testInstance(t) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	_, err := ku.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := ku.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  futuresTradablePair.Base.Item,
		Quote: futuresTradablePair.Quote.Item,
		Asset: asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp1 := currency.NewPair(currency.ETH, currency.USDTM)

	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ku, asset.Futures, cp1)
	resp, err = ku.GetOpenInterest(t.Context(),
		key.PairAsset{
			Base:  futuresTradablePair.Base.Item,
			Quote: futuresTradablePair.Quote.Item,
			Asset: asset.Futures,
		},
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.Futures,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = ku.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestValidatePlaceOrderParams(t *testing.T) {
	t.Parallel()
	arg := &PlaceHFParam{}
	err := arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, common.ErrNilPointer)
	arg.Size = 1
	err = arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	arg.Symbol = spotTradablePair
	err = arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	arg.OrderType = "limit"
	err = arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = "Sell"
	err = arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	arg.Price = 323423423
	arg.Size = 0
	err = arg.ValidatePlaceOrderParams()
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	arg.Size = 1
	err = arg.ValidatePlaceOrderParams()
	assert.NoError(t, err)
}

func TestSpotHFPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.HFSpotPlaceOrder(t.Context(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.HFSpotPlaceOrder(t.Context(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      spotTradablePair,
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpotPlaceHFOrderTest(t *testing.T) {
	t.Parallel()
	_, err := ku.SpotPlaceHFOrderTest(t.Context(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SpotPlaceHFOrderTest(t.Context(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      spotTradablePair,
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSyncPlaceHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.SyncPlaceHFOrder(t.Context(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SyncPlaceHFOrder(t.Context(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PlaceMultipleOrders(t.Context(), []PlaceHFParam{
		{
			TimeInForce: "GTT",
			Symbol:      spotTradablePair,
			OrderType:   "limit",
			Side:        order.Sell.String(),
			Price:       1234,
			Size:        1,
		},
		{
			ClientOrderID: "3d07008668054da6b3cb12e432c2b13a",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
		{
			ClientOrderID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSyncPlaceMultipleHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SyncPlaceMultipleHFOrders(t.Context(), []PlaceHFParam{
		{
			TimeInForce: "GTT",
			Symbol:      spotTradablePair,
			OrderType:   "limit",
			Side:        order.Sell.String(),
			Price:       1234,
			Size:        1,
		},
		{
			ClientOrderID: "3d07008668054da6b3cb12e432c2b13a",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        spotTradablePair,
		},
		{
			ClientOrderID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        spotTradablePair,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.ModifyHFOrder(t.Context(), &ModifyHFOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ku.ModifyHFOrder(t.Context(), &ModifyHFOrderParam{OrderID: "1234"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.ModifyHFOrder(t.Context(), &ModifyHFOrderParam{
		Symbol:        spotTradablePair,
		ClientOrderID: "4314oiu5345u2y554x",
		OrderID:       "4314oiu5345u2y554x",
		NewPrice:      1234,
		NewSize:       2,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelHFOrder(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.CancelHFOrder(t.Context(), "630625dbd9180300014c8d52", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelHFOrder(t.Context(), "630625dbd9180300014c8d52", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSyncCancelHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.SyncCancelHFOrder(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.SyncCancelHFOrder(t.Context(), "12312312", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SyncCancelHFOrder(t.Context(), "641d67ea162d47000160bfb8", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSyncCancelHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.SyncCancelHFOrderByClientOrderID(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.SyncCancelHFOrderByClientOrderID(t.Context(), "cliend-order-id", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SyncCancelHFOrderByClientOrderID(t.Context(), "client-order-id", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelHFOrderByClientOrderID(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)
	_, err = ku.CancelHFOrderByClientOrderID(t.Context(), "cliend-order-id", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelHFOrderByClientOrderID(t.Context(), "client-order-id", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSpecifiedNumberHFOrdersByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelSpecifiedNumberHFOrdersByOrderID(t.Context(), "", spotTradablePair.String(), 10.0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.CancelSpecifiedNumberHFOrdersByOrderID(t.Context(), "1", "", 10.0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.CancelSpecifiedNumberHFOrdersByOrderID(t.Context(), "1", spotTradablePair.String(), 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelSpecifiedNumberHFOrdersByOrderID(t.Context(), "1", spotTradablePair.String(), 10.0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllHFOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelAllHFOrdersBySymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllHFOrdersBySymbol(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllHFOrders(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveHFOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetActiveHFOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	_, err = ku.GetActiveHFOrders(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetActiveHFOrders(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)
}

func TestGetSymbolsWithActiveHFOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSymbolsWithActiveHFOrderList(t.Context())
	assert.NoError(t, err)
}

func TestGetHFCompletedOrderList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetHFCompletedOrderList(t.Context(), "", "sell", "limit", "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHFCompletedOrderList(t.Context(), spotTradablePair.String(), "sell", "limit", "", time.Time{}, time.Now(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHFOrderDetailsByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetHFOrderDetailsByOrderID(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.GetHFOrderDetailsByOrderID(t.Context(), "1234567", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHFOrderDetailsByOrderID(t.Context(), "1234567", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHFOrderDetailsByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetHFOrderDetailsByClientOrderID(t.Context(), "", spotTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.GetHFOrderDetailsByClientOrderID(t.Context(), "1234567", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHFOrderDetailsByClientOrderID(t.Context(), "6d539dc614db312", spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAutoCancelHFOrderSetting(t *testing.T) {
	t.Parallel()
	_, err := ku.AutoCancelHFOrderSetting(t.Context(), 0, []string{})
	require.ErrorIs(t, err, errTimeoutRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.AutoCancelHFOrderSetting(t.Context(), 450, []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAutoCancelHFOrderSettingQuery(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.AutoCancelHFOrderSettingQuery(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHFFilledList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetHFFilledList(t.Context(), "", "", "sell", "market", "", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetHFFilledList(t.Context(), "", spotTradablePair.String(), "sell", "market", "", time.Time{}, time.Now(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PlaceOCOOrder(t.Context(), &OCOOrderParams{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OCOOrderParams{Remark: "oco-new-order"}
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = spotTradablePair
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Sell"
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 1000
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Size = .1
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.StopPrice = .1
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.LimitPrice = .1
	_, err = ku.PlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	cpDetail, err := ku.GetTicker(t.Context(), spotTradablePair.String())
	assert.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.PlaceOCOOrder(t.Context(), &OCOOrderParams{
		Symbol:        spotTradablePair,
		Side:          order.Buy.String(),
		Price:         cpDetail.Price - 3,
		Size:          1,
		StopPrice:     cpDetail.Price - 2,
		LimitPrice:    cpDetail.Price - 1,
		TradeType:     SpotTradeType,
		ClientOrderID: "6572fdd65723280007deb5e0",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelOCOOrderByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelOCOOrderByOrderID(t.Context(), "6572fdd65723280007deb5e0")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelOCOOrderByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelOCOOrderByClientOrderID(t.Context(), "6572fdd65723280007deb5e0")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelOCOMultipleOrders(t.Context(), []string{}, spotTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfoByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOCOOrderInfoByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetOCOOrderInfoByOrderID(t.Context(), "order-id-here")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfoByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOCOOrderInfoByClientOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetOCOOrderInfoByClientOrderID(t.Context(), "client-order-id-here")
	assert.NoError(t, err)
}

func TestGetOrderDetailsByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOCOOrderDetailsByOrderID(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetOCOOrderDetailsByOrderID(t.Context(), "order-id-here")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetOCOOrderList(t.Context(), 9, 0, spotTradablePair.String(), time.Time{}, time.Now(), []string{})
	require.ErrorIs(t, err, errPageSizeRequired)
	_, err = ku.GetOCOOrderList(t.Context(), 10, 0, spotTradablePair.String(), time.Time{}, time.Now(), []string{})
	require.ErrorIs(t, err, errCurrentPageRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetOCOOrderList(t.Context(), 10, 2, spotTradablePair.String(), time.Time{}, time.Now(), []string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSendPlaceMarginHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.SendPlaceMarginHFOrder(t.Context(), &PlaceMarginHFOrderParam{}, "")
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceMarginHFOrderParam{PostOnly: true}
	_, err = ku.SendPlaceMarginHFOrder(t.Context(), arg, "")
	require.ErrorIs(t, err, order.ErrClientOrderIDNotSupported)

	arg.ClientOrderID = "first-order"
	_, err = ku.SendPlaceMarginHFOrder(t.Context(), arg, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Sell"
	_, err = ku.SendPlaceMarginHFOrder(t.Context(), arg, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Symbol = marginTradablePair
	_, err = ku.SendPlaceMarginHFOrder(t.Context(), arg, "")
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 1000
	_, err = ku.SendPlaceMarginHFOrder(t.Context(), arg, "")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
}

func TestPlaceMarginHFOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.PlaceMarginHFOrder(t.Context(), &PlaceMarginHFOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.PlaceMarginHFOrder(t.Context(), &PlaceMarginHFOrderParam{
		ClientOrderID:       "first-order",
		Side:                "sell",
		Symbol:              marginTradablePair,
		OrderType:           "market",
		SelfTradePrevention: "CB",
		Price:               1234,
		Size:                0.0000001,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceMarginHFOrderTest(t *testing.T) {
	t.Parallel()
	_, err := ku.PlaceMarginHFOrderTest(t.Context(), &PlaceMarginHFOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.PlaceMarginHFOrderTest(t.Context(), &PlaceMarginHFOrderParam{
		ClientOrderID:       "first-order",
		Side:                "sell",
		Symbol:              marginTradablePair,
		OrderType:           "market",
		SelfTradePrevention: "CB",
		Price:               1234,
		Size:                0.0000001,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginHFOrderByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelMarginHFOrderByOrderID(t.Context(), "", marginTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.CancelMarginHFOrderByOrderID(t.Context(), "order-id-here", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelMarginHFOrderByOrderID(t.Context(), "order-id-here", marginTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMarginHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelMarginHFOrderByClientOrderID(t.Context(), "", marginTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.CancelMarginHFOrderByClientOrderID(t.Context(), "order-id-here", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelMarginHFOrderByClientOrderID(t.Context(), "order-id-here", marginTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllMarginHFOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := ku.CancelAllMarginHFOrdersBySymbol(t.Context(), "", "MARGIN_TRADE")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.CancelAllMarginHFOrdersBySymbol(t.Context(), marginTradablePair.String(), "")
	require.ErrorIs(t, err, errTradeTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.CancelAllMarginHFOrdersBySymbol(t.Context(), marginTradablePair.String(), "MARGIN_TRADE")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ku.CancelAllMarginHFOrdersBySymbol(t.Context(), marginTradablePair.String(), "MARGIN_ISOLATED_TRADE")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveMarginHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetActiveMarginHFOrders(t.Context(), marginTradablePair.String(), "MARGIN_TRADE")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFilledHFMarginOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFilledHFMarginOrders(t.Context(), spotTradablePair.String(), "", "sell", "limit", time.Time{}, time.Now(), 0, 20)
	require.ErrorIs(t, err, errTradeTypeMissing)
	_, err = ku.GetFilledHFMarginOrders(t.Context(), "", "MARGIN_TRADE", "sell", "limit", time.Time{}, time.Now(), 0, 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetFilledHFMarginOrders(t.Context(), marginTradablePair.String(), "MARGIN_TRADE", "sell", "limit", time.Time{}, time.Now(), 0, 20)
	assert.NoError(t, err)
}

func TestGetMarginHFOrderDetailByOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginHFOrderDetailByOrderID(t.Context(), "", marginTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetMarginHFOrderDetailByOrderID(t.Context(), "243432432423the-order-id", marginTradablePair.String())
	assert.Truef(t, errors.Is(err, order.ErrOrderNotFound) || err == nil, "GetMarginHFOrderDetailByOrderID should not error: %s", err)
}

func TestGetMarginHFOrderDetailByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginHFOrderDetailByClientOrderID(t.Context(), "", marginTradablePair.String())
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetMarginHFOrderDetailByClientOrderID(t.Context(), "the-client-order-id", marginTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginHFTradeFills(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginHFTradeFills(t.Context(), "", marginTradablePair.String(), "", "sell", "", time.Time{}, time.Now(), 0, 30)
	require.ErrorIs(t, err, errTradeTypeMissing)

	_, err = ku.GetMarginHFTradeFills(t.Context(), "", "", "MARGIN_TRADE", "sell", "", time.Time{}, time.Now(), 0, 30)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetMarginHFTradeFills(t.Context(), "12312312", marginTradablePair.String(), "MARGIN_TRADE", "sell", "market", time.Time{}, time.Now(), 0, 30)
	assert.NoError(t, err)
}

func TestGetLendingCurrencyInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetLendingCurrencyInformation(t.Context(), currency.ETH)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetInterestRate(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetInterestRate(t.Context(), currency.ETH)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMarginLendingSubscription(t *testing.T) {
	t.Parallel()
	_, err := ku.MarginLendingSubscription(t.Context(), currency.EMPTYCODE, 1, 0.22)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.MarginLendingSubscription(t.Context(), currency.ETH, 0, 0.22)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.MarginLendingSubscription(t.Context(), currency.ETH, 1, 0)
	require.ErrorIs(t, err, errMissingInterestRate)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.MarginLendingSubscription(t.Context(), currency.ETH, 1, 0.22)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedemption(t *testing.T) {
	t.Parallel()
	_, err := ku.Redemption(t.Context(), currency.EMPTYCODE, 1, "1245")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.Redemption(t.Context(), currency.ETH, 0, "1245")
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.Redemption(t.Context(), currency.ETH, 1, "")
	require.ErrorIs(t, err, errMissingPurchaseOrderNumber)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.Redemption(t.Context(), currency.ETH, 1, "1245")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifySubscriptionOrder(t *testing.T) {
	t.Parallel()
	_, err := ku.ModifySubscriptionOrder(t.Context(), currency.EMPTYCODE, "12345", 1.23)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.ModifySubscriptionOrder(t.Context(), currency.ETH, "12345", 0)
	require.ErrorIs(t, err, errMissingInterestRate)
	_, err = ku.ModifySubscriptionOrder(t.Context(), currency.ETH, "", 1.23)
	require.ErrorIs(t, err, errMissingPurchaseOrderNumber)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.ModifySubscriptionOrder(t.Context(), currency.ETH, "12345", 1.23)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRedemptionOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetRedemptionOrders(t.Context(), currency.EMPTYCODE, "DONE", "2234", 0, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.GetRedemptionOrders(t.Context(), currency.ETH, "", "", 0, 20)
	require.ErrorIs(t, err, errStatusMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetRedemptionOrders(t.Context(), currency.BTC, "2234", "PENDING", 0, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubscriptionOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetSubscriptionOrders(t.Context(), currency.EMPTYCODE, "2234", "DONE", 0, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ku.GetSubscriptionOrders(t.Context(), currency.ETH, "", "", 0, 20)
	require.ErrorIs(t, err, errStatusMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetSubscriptionOrders(t.Context(), currency.BTC, "2234", "DONE", 0, 20)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, ku)
	for _, a := range ku.GetAssetTypes(false) {
		pairs, err := ku.CurrencyPairs.GetPairs(a, false)
		assert.NoErrorf(t, err, "cannot get pairs for %s", a)
		assert.NotEmptyf(t, pairs, "no pairs for %s", a)

		resp, err := ku.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

// testInstance returns a local Kucoin for isolated testing
func testInstance(tb testing.TB) *Kucoin {
	tb.Helper()
	kucoin := new(Kucoin)
	require.NoError(tb, testexch.Setup(kucoin), "Test instance Setup must not error")
	kucoin.obm = &orderbookManager{
		state: make(map[currency.Code]map[currency.Code]map[asset.Item]*update),
		jobs:  make(chan job, maxWSOrderbookJobs),
	}
	return kucoin
}

func TestGetTradingPairActualFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetTradingPairActualFees(t.Context(), []string{spotTradablePair.String()})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesTradingPairsActualFees(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTradingPairsActualFees(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetFuturesTradingPairsActualFees(t.Context(), futuresTradablePair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPositionHistory(t.Context(), futuresTradablePair.String(), time.Now(), time.Now().Add(-time.Hour*5), 0, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetPositionHistory(t.Context(), futuresTradablePair.String(), time.Time{}, time.Time{}, 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumOpenPositionSize(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMaximumOpenPositionSize(t.Context(), "", 1, 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.GetMaximumOpenPositionSize(t.Context(), futuresTradablePair.String(), 0., 1)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	_, err = ku.GetMaximumOpenPositionSize(t.Context(), futuresTradablePair.String(), 1, 0)
	require.ErrorIs(t, err, errInvalidLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetMaximumOpenPositionSize(t.Context(), futuresTradablePair.String(), 1, 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestTickersForAllContracts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetLatestTickersForAllContracts(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubscribeToEarnFixedIncomeProduct(t *testing.T) {
	t.Parallel()
	_, err := ku.SubscribeToEarnFixedIncomeProduct(t.Context(), "", "MAIN", 12.2)
	require.ErrorIs(t, err, errProductIDMissing)
	_, err = ku.SubscribeToEarnFixedIncomeProduct(t.Context(), "1232412", "MAIN", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ku.SubscribeToEarnFixedIncomeProduct(t.Context(), "1232412", "", 12.2)
	require.ErrorIs(t, err, errAccountTypeMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.SubscribeToEarnFixedIncomeProduct(t.Context(), "1232412", "MAIN", 12.2)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeemByEarnHoldingID(t *testing.T) {
	t.Parallel()
	_, err := ku.RedeemByEarnHoldingID(t.Context(), "", SpotTradeType, "1", 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ku.RedeemByEarnHoldingID(t.Context(), "123231", "Main", "1", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	result, err := ku.RedeemByEarnHoldingID(t.Context(), "123231", SpotTradeType, "1", 1)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnRedeemPreviewByHoldingID(t *testing.T) {
	t.Parallel()
	_, err := ku.GetEarnRedeemPreviewByHoldingID(t.Context(), "", "MAIN")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnRedeemPreviewByHoldingID(t.Context(), "12345", "MAIN")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnSavingsProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnSavingsProducts(t.Context(), currency.EMPTYCODE)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnFixedIncomeCurrentHoldings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnFixedIncomeCurrentHoldings(t.Context(), "12312", "", currency.EMPTYCODE, 0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitedTimePromotionProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetLimitedTimePromotionProducts(t.Context(), currency.BTC)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnKCSStakingProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnKCSStakingProducts(t.Context(), currency.EMPTYCODE)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnStakingProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnStakingProducts(t.Context(), currency.EMPTYCODE)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnETHStakingProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetEarnETHStakingProducts(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInformationOnOffExchangeFundingAndLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetInformationOnOffExchangeFundingAndLoans(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInformationOnAccountInvolvedInOffExchangeLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetInformationOnAccountInvolvedInOffExchangeLoans(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAffilateUserRebateInformation(t *testing.T) {
	t.Parallel()
	_, err := ku.GetAffilateUserRebateInformation(t.Context(), time.Time{}, "1234", 0)
	require.ErrorIs(t, err, errQueryDateIsRequired)
	_, err = ku.GetAffilateUserRebateInformation(t.Context(), time.Now(), "", 0)
	require.ErrorIs(t, err, errOffsetIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetAffilateUserRebateInformation(t.Context(), time.Now(), "1234", 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginPairsConfigurations(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginPairsConfigurations(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetMarginPairsConfigurations(t.Context(), marginTradablePair.String())
	assert.NoError(t, err)
}

func TestModifyLeverageMultiplier(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.ModifyLeverageMultiplier(t.Context(), spotTradablePair.String(), 1, true)
	assert.NoError(t, err)
}

func TestGetActiveHFOrderSymbols(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	result, err := ku.GetActiveHFOrderSymbols(t.Context(), "MARGIN_TRADE")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	sideErrInput := []struct {
		Side   order.Side
		Result string
		Err    error
	}{
		{Side: order.Sell, Result: "sell"},
		{Side: order.Buy, Result: "buy"},
		{Side: order.AnySide, Result: ""},
		{Side: order.Bid, Result: "buy"},
		{Side: order.Ask, Result: "sell"},
		{Side: order.CouldNotShort, Result: "", Err: order.ErrSideIsInvalid},
	}
	var err error
	var sideString string
	for a := range sideErrInput {
		sideString, err = ku.OrderSideString(sideErrInput[a].Side)
		require.ErrorIs(t, err, sideErrInput[a].Err)
		assert.Equal(t, sideString, sideErrInput[a].Result)
	}
}

func TestOrderTypeToString(t *testing.T) {
	t.Parallel()
	oTypeErrInputs := []struct {
		OrderType order.Type
		Result    string
		Err       error
	}{
		{OrderType: order.Limit, Result: "limit"},
		{OrderType: order.Market, Result: "market"},
		{OrderType: order.AnyType, Result: ""},
		{OrderType: order.OCO, Result: "", Err: order.ErrUnsupportedOrderType},
	}
	var err error
	var oTypeString string
	for a := range oTypeErrInputs {
		oTypeString, err = OrderTypeToString(oTypeErrInputs[a].OrderType)
		require.ErrorIs(t, err, oTypeErrInputs[a].Err)
		assert.Equal(t, oTypeString, oTypeErrInputs[a].Result)
	}
}

func TestMarginModeToString(t *testing.T) {
	t.Parallel()
	marginModeResults := []struct {
		MarginMode margin.Type
		Result     string
	}{
		{MarginMode: margin.Isolated, Result: "isolated"},
		{MarginMode: margin.Multi, Result: "cross"},
		{MarginMode: margin.Unknown, Result: ""},
	}
	for a := range marginModeResults {
		result := MarginModeToString(marginModeResults[a].MarginMode)
		assert.Equal(t, result, marginModeResults[a].Result)
	}
}

func TestAccountToTradeTypeString(t *testing.T) {
	t.Parallel()
	accountToTradeTypeResults := []struct {
		AccountType asset.Item
		MarginMode  string
		Result      string
	}{
		{AccountType: asset.Margin, MarginMode: "cross", Result: "MARGIN_TRADE"},
		{AccountType: asset.Margin, MarginMode: "isolated", Result: "MARGIN_ISOLATED_TRADE"},
		{AccountType: asset.Spot, MarginMode: "cross", Result: SpotTradeType},
		{AccountType: asset.Spot, MarginMode: "isolated", Result: SpotTradeType},
		{AccountType: asset.Futures, MarginMode: "isolated", Result: ""},
		{AccountType: asset.Futures, MarginMode: "isolated", Result: ""},
	}
	for a := range accountToTradeTypeResults {
		result := ku.AccountToTradeTypeString(accountToTradeTypeResults[a].AccountType, accountToTradeTypeResults[a].MarginMode)
		assert.Equal(t, result, accountToTradeTypeResults[a].Result)
	}
}

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	orderStatusResults := []struct {
		Input  string
		Result order.Status
		HasErr bool
	}{
		{Input: "match", Result: order.Filled},
		{Input: "open", Result: order.Open},
		{Input: "done", Result: order.Closed},
		{Input: "accepted", Result: order.New},
		{Input: "PLaced", Result: order.New},
		{Input: "something", Result: order.UnknownStatus, HasErr: true},
	}
	for a := range orderStatusResults {
		result, err := ku.StringToOrderStatus(orderStatusResults[a].Input)
		if !orderStatusResults[a].HasErr {
			assert.NoError(t, err)
		}
		assert.Equal(t, result, orderStatusResults[a].Result)
	}
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	intervalStringResults := []struct {
		Interval kline.Interval
		Result   string
		Err      error
	}{
		{Interval: kline.OneMin, Result: "1min"},
		{Interval: kline.ThreeMin, Result: "3min"},
		{Interval: kline.FiveMin, Result: "5min"},
		{Interval: kline.TenMin, Result: "", Err: kline.ErrUnsupportedInterval},
	}
	for a := range intervalStringResults {
		intervalString, err := IntervalToString(intervalStringResults[a].Interval)
		require.ErrorIs(t, err, intervalStringResults[a].Err)
		assert.Equal(t, intervalString, intervalStringResults[a].Result)
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	r := &fundingrate.HistoricalRatesRequest{
		Asset:     asset.Spot,
		Pair:      futuresTradablePair,
		StartDate: time.Now().Add(-time.Hour * 24 * 2),
		EndDate:   time.Now(),
	}
	_, err := ku.GetHistoricalFundingRates(t.Context(), r)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	r.Pair = currency.EMPTYPAIR
	_, err = ku.GetHistoricalFundingRates(t.Context(), r)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	r.Asset = asset.Futures
	r.Pair = futuresTradablePair
	result, err := ku.GetHistoricalFundingRates(t.Context(), r)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProcessFuturesKline(t *testing.T) {
	t.Parallel()
	data := fmt.Sprintf(`{"symbol":%q,"candles":["1714964400","63815.1","63890.8","63928.5","63797.8","17553.0","17553"],"time":1714964823722}`, futuresTradablePair.String())
	err := ku.processFuturesKline([]byte(data), "1hour")
	assert.NoError(t, err)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	withdrawFiatRequest := withdraw.Request{}
	_, err := ku.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	assert.ErrorIs(t, common.ErrFunctionNotSupported, err)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := ku.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, common.ErrFunctionNotSupported, err)
}

func TestModifyOrder(t *testing.T) {
	_, err := ku.ModifyOrder(t.Context(), &order.Modify{})
	assert.ErrorIs(t, common.ErrFunctionNotSupported, err)
}

func TestGetHistoricTrades(t *testing.T) {
	_, err := ku.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, common.ErrFunctionNotSupported, err)
}

func TestCancelBatchOrders(t *testing.T) {
	_, err := ku.CancelBatchOrders(t.Context(), nil)
	assert.ErrorIs(t, common.ErrFunctionNotSupported, err)
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		ch  string
		exp string
	}{
		{asset.Futures, futuresOrderbookDepth50Channel, futuresOrderbookDepth50Channel},
		{asset.Futures, subscription.OrderbookChannel, futuresOrderbookDepth5Channel},
		{asset.Futures, subscription.CandlesChannel, marketCandlesChannel},
		{asset.Futures, subscription.TickerChannel, futuresTickerChannel},
		{asset.Spot, subscription.OrderbookChannel, marketOrderbookDepth5Channel},
		{asset.Spot, subscription.AllTradesChannel, marketMatchChannel},
		{asset.Spot, subscription.CandlesChannel, marketCandlesChannel},
		{asset.Spot, subscription.TickerChannel, marketTickerChannel},
	} {
		assert.Equal(t, tt.exp, channelName(&subscription.Subscription{Channel: tt.ch}, tt.a))
	}
}

func TestStringToTimeInForce(t *testing.T) {
	t.Parallel()
	tifMap := []struct {
		String      string
		PostOnly    bool
		TimeInForce order.TimeInForce
	}{
		{"GTC", false, order.GoodTillCancel},
		{"GTC", true, order.GoodTillCancel | order.PostOnly},
		{"GTT", false, order.GoodTillTime},
		{"GTT", true, order.GoodTillTime | order.PostOnly},
		{"IOC", false, order.ImmediateOrCancel},
		{"ioC", false, order.ImmediateOrCancel},
		{"Fok", false, order.FillOrKill},
	}
	for a := range tifMap {
		result := StringToTimeInForce(tifMap[a].String, tifMap[a].PostOnly)
		assert.Equal(t, tifMap[a].TimeInForce, result)
	}
}
