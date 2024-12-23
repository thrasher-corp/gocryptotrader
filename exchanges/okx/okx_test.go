package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
	useTestNet              = false
)

var (
	ok = &Okx{}

	leadTraderUniqueID string
	loadLeadTraderOnce sync.Once

	spotTP, marginTP, futuresTP, perpetualSwapTP, optionsTP, spreadTP currency.Pair
)

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("Okx")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	ok.SetDefaults()

	if apiKey != "" && apiSecret != "" && passphrase != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
		ok.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		ok.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")
		ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if !useTestNet {
		ok.Websocket = sharedtestvalues.NewTestWebsocket()
	}
	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	err = ok.UpdateTradablePairs(contextGenerate(), true)
	if err != nil {
		log.Fatal(err)
	}
	if !useTestNet {
		ok.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
		ok.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
		setupWS()
	}
	err = populateTradablePairs()
	if err != nil {
		log.Fatal(err)
	}
	syncLeadTraderUniqueID()
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	errNoEnabledPair := errors.New("no enabled pair found")
	err := ok.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		return err
	}

	assetToTradablePairMap := map[asset.Item]currency.Pair{
		asset.Spot:          spotTP,
		asset.Margin:        marginTP,
		asset.Futures:       futuresTP,
		asset.Options:       optionsTP,
		asset.PerpetualSwap: perpetualSwapTP,
		asset.Spread:        spreadTP,
	}
	for a := range assetToTradablePairMap {
		tradablePairs, err := ok.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		if len(tradablePairs) == 0 {
			return fmt.Errorf("%w %v", errNoEnabledPair, a)
		}
		switch a {
		case asset.Spot:
			spotTP = tradablePairs[0]
		case asset.Margin:
			marginTP = tradablePairs[0]
		case asset.Futures:
			futuresTP = tradablePairs[0]
		case asset.Options:
			optionsTP = tradablePairs[0]
		case asset.PerpetualSwap:
			perpetualSwapTP = tradablePairs[0]
		case asset.Spread:
			spreadTP = tradablePairs[0]
		}
	}
	return nil
}

func syncLeadTraderUniqueID() {
	loadLeadTraderOnce.Do(func() {
		result, err := ok.GetLeadTradersRanks(context.Background(), "SWAP", "pnl_ratio", "1", "", "", "", "", "", "", "", 10)
		if err != nil {
			log.Fatal(err)
		}
		if len(result) == 0 {
			log.Fatal("No lead trader found")
		}
		if len(result[0].Ranks) == 0 {
			log.Fatal("could not load lead traders ranks")
		}
		leadTraderUniqueID = result[0].Ranks[0].UniqueCode
	})
}

// contextGenerate sends an optional value to allow test requests
// named this way, so it shows up in auto-complete and reminds you to use it
func contextGenerate() context.Context {
	ctx := context.Background()
	if useTestNet {
		ctx = context.WithValue(ctx, testNetKey("testnet"), useTestNet)
	}
	return ctx
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	instFamily, err := ok.instrumentFamilyFromInstID(okxInstTypeOption, optionsTP.String())
	require.NoError(t, err)

	_, err = ok.GetTickers(contextGenerate(), "", "", instFamily)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetTickers(contextGenerate(), okxInstTypeOption, "", instFamily)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexTickers(contextGenerate(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	result, err := ok.GetIndexTickers(contextGenerate(), currency.USDT, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTicker(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetTicker(contextGenerate(), perpetualSwapTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPremiumHistory(context.Background(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetPremiumHistory(context.Background(), perpetualSwapTP.String(), time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderBookDepth(contextGenerate(), "", 400)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetOrderBookDepth(contextGenerate(), spotTP.String(), 400)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticks(contextGenerate(), "", kline.OneHour, time.Now().Add(-time.Minute*2), time.Now(), 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetCandlesticks(contextGenerate(), spotTP.String(), kline.OneHour, time.Now().Add(-time.Minute*2), time.Now(), 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticksHistory(contextGenerate(), "", kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetCandlesticksHistory(contextGenerate(), spotTP.String(), kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTrades(contextGenerate(), "", 3)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetTrades(contextGenerate(), spotTP.String(), 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTradesHistory(contextGenerate(), "", "", "", 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetTradesHistory(contextGenerate(), spotTP.String(), "", "", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetoptionTradesByInstrumentFamily(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionTradesByInstrumentFamily(context.Background(), "")
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	instFamily, err := ok.instrumentFamilyFromInstID(okxInstTypeOption, optionsTP.String())
	require.NoError(t, err)

	result, err := ok.GetOptionTradesByInstrumentFamily(context.Background(), instFamily)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionTrades(context.Background(), "", "", "C")
	require.ErrorIs(t, err, errInstrumentIDorFamilyRequired)

	instFamily, err := ok.instrumentFamilyFromInstID(okxInstTypeOption, optionsTP.String())
	require.NoError(t, err)

	result, err := ok.GetOptionTrades(context.Background(), "", instFamily, "C")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	result, err := ok.Get24HTotalVolume(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	result, err := ok.GetOracle(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	result, err := ok.GetExchangeRate(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexComponents(contextGenerate(), "")
	require.ErrorIs(t, err, errIndexComponentNotFound)

	result, err := ok.GetIndexComponents(contextGenerate(), "ETH-USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBlockTickers(contextGenerate(), "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetBlockTickers(contextGenerate(), "SWAP", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBlockTicker(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetBlockTicker(contextGenerate(), "BTC-USDT")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	trades, err := ok.GetPublicBlockTrades(contextGenerate(), "BTC-USDT")
	require.NoError(t, err)
	if assert.NotEmpty(t, trades, "Should get some block trades") {
		trade := trades[0]
		assert.Equal(t, "BTC-USDT", trade.InstrumentID, "InstrumentID should have correct value")
		assert.NotEmpty(t, trade.TradeID, "TradeID should not be empty")
		assert.Positive(t, trade.Price, "Price should have a positive value")
		assert.Positive(t, trade.Size, "Size should have a positive value")
		assert.Contains(t, []order.Side{order.Buy, order.Sell}, trade.Side, "Side should be a side")
		assert.WithinRange(t, trade.Timestamp.Time(), time.Now().Add(time.Hour*-24*90), time.Now(), "Timestamp should be within last 90 days")
	}

	testexch.UpdatePairsOnce(t, ok)

	pairs, err := ok.GetAvailablePairs(asset.Options)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	publicTrades, err := ok.GetPublicRFQTrades(contextGenerate(), "", "", 100)
	require.NoError(t, err)

	tested := false
LOOP:
	for _, trade := range publicTrades {
		for _, leg := range trade.Legs {
			p, err := ok.MatchSymbolWithAvailablePairs(leg.InstrumentID, asset.Options, true)
			if err != nil {
				continue
			}

			trades, err = ok.GetPublicBlockTrades(contextGenerate(), p.String())
			require.NoError(t, err, "GetBlockTrades should not error on Options")
			for _, trade := range trades {
				assert.Equal(t, p.String(), trade.InstrumentID, "InstrumentID should have correct value")
				assert.NotEmpty(t, trade.TradeID, "TradeID should not be empty")
				assert.Positive(t, trade.Price, "Price should have a positive value")
				assert.Positive(t, trade.Size, "Size should have a positive value")
				assert.Contains(t, []order.Side{order.Buy, order.Sell}, trade.Side, "Side should be a side")
				assert.Positive(t, trade.FillVolatility, "FillVolatility should have a positive value")
				assert.Positive(t, trade.ForwardPrice, "ForwardPrice should have a positive value")
				assert.Positive(t, trade.IndexPrice, "IndexPrice should have a positive value")
				assert.Positive(t, trade.MarkPrice, "MarkPrice should have a positive value")
				assert.NotEmpty(t, trade.Timestamp, "Timestamp should not be empty")
				tested = true
				break LOOP
			}
		}
	}
	assert.True(t, tested, "Should find at least one BlockTrade somewhere")
}

func TestGetInstrument(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{Underlying: "SOL-USD"})
	assert.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: okxInstTypeOption, Underlying: ""})
	assert.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	result, err := ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: okxInstTypeFutures,
		Underlying:     "SOL-USD",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: okxInstTypeSpot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: okxInstTypeSwap,
		Underlying:     "SOL-USD",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDeliveryHistory(contextGenerate(), "", "BTC-USDT", "", time.Time{}, time.Time{}, 3)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.GetDeliveryHistory(contextGenerate(), okxInstTypeFutures, "", "", time.Time{}, time.Time{}, 3)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	_, err = ok.GetDeliveryHistory(contextGenerate(), okxInstTypeFutures, "BTC-USDT", "", time.Time{}, time.Time{}, 345)
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	result, err := ok.GetDeliveryHistory(contextGenerate(), okxInstTypeFutures, "BTC-USDT", "", time.Time{}, time.Time{}, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterestData(contextGenerate(), "", "BTC-USDT", "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.GetOpenInterestData(contextGenerate(), okxInstTypeOption, "", "", "")
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	uly, err := ok.underlyingFromInstID(okxInstTypeFutures, futuresTP.String())
	require.NoError(t, err)

	result, err := ok.GetOpenInterestData(contextGenerate(), okxInstTypeFutures, uly, "", futuresTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSingleFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSingleFundingRate(context.Background(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetSingleFundingRate(context.Background(), "BTC-USD-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFundingRateHistory(contextGenerate(), "", time.Time{}, time.Time{}, 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetFundingRateHistory(contextGenerate(), "BTC-USD-SWAP", time.Time{}, time.Time{}, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLimitPrice(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetLimitPrice(contextGenerate(), "BTC-USD-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionMarketData(contextGenerate(), "", "", time.Time{})
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	result, err := ok.GetOptionMarketData(contextGenerate(), "BTC-USD", "", time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := ok.GetEstimatedDeliveryPrice(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	r, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	require.NoError(t, err)

	result, err := ok.GetEstimatedDeliveryPrice(contextGenerate(), r[0].String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	result, err := ok.GetDiscountRateAndInterestFreeQuota(contextGenerate(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := ok.GetSystemTime(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	insts, err := ok.FetchTradablePairs(contextGenerate(), asset.Margin)
	require.NoError(t, err)

	result, err := ok.GetLiquidationOrders(contextGenerate(), &LiquidationOrderRequestParams{
		InstrumentType: okxInstTypeMargin,
		Underlying:     insts[0].String(),
		Currency:       currency.BTC,
		Limit:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMarkPrice(contextGenerate(), "", "", "", "BTC-USDT")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetMarkPrice(contextGenerate(), "MARGIN", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPositionTiers(contextGenerate(), "", "cross", "BTC-USDT", "", "", "", currency.ETH)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.GetPositionTiers(contextGenerate(), okxInstTypeFutures, "", "BTC-USDT", "", "", "", currency.ETH)
	require.ErrorIs(t, err, errInvalidTradeMode)

	_, err = ok.GetPositionTiers(contextGenerate(), okxInstTypeFutures, "cross", "", "", "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	_, err = ok.GetPositionTiers(contextGenerate(), okxInstTypeFutures, "cross", "BTC-USDT", "", "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	result, err := ok.GetPositionTiers(contextGenerate(), okxInstTypeFutures, "cross", "BTC-USDT", "", "", "", currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	result, err := ok.GetInterestRateAndLoanQuota(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRateAndLoanQuotaForVIPLoans(t *testing.T) {
	t.Parallel()
	result, err := ok.GetInterestRateAndLoanQuotaForVIPLoans(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicUnderlyings(contextGenerate(), "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetPublicUnderlyings(contextGenerate(), okxInstTypeFutures)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	arg := &InsuranceFundInformationRequestParams{
		Limit: 2,
	}
	_, err := ok.GetInsuranceFundInformation(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	arg.InstrumentType = okxInstTypeSwap
	_, err = ok.GetInsuranceFundInformation(contextGenerate(), arg)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	arg.Underlying = "BTC-USDT"
	r, err := ok.GetInsuranceFundInformation(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.Positive(t, r.Total, "Total should be positive")
	assert.NotEmpty(t, r.Details, "Should have some details")
	for _, d := range r.Details {
		assert.Positive(t, d.Balance, "Balance should be positive")
		assert.NotEmpty(t, d.InsuranceType, "Type should not be empty")
		assert.Positive(t, d.Timestamp, "Timestamp should be positive")
	}

	r, err = ok.GetInsuranceFundInformation(contextGenerate(), &InsuranceFundInformationRequestParams{
		InstrumentType: okxInstTypeFutures,
		Underlying:     "BTC-USDT",
		Limit:          2,
	})
	assert.NoError(t, err)
	assert.Positive(t, r.Total, "Total should be positive")
	assert.NotEmpty(t, r.Details, "Should have some details")
	for _, d := range r.Details {
		assert.Positive(t, d.Balance, "Balance should be positive")
		assert.NotEmpty(t, d.InsuranceType, "Type should not be empty")
		assert.Positive(t, d.Timestamp, "Timestamp should be positive")
	}
}

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	_, err := ok.CurrencyUnitConvert(contextGenerate(), "", 1, 3500, 1, currency.EMPTYCODE, false)
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.CurrencyUnitConvert(contextGenerate(), "BTC-USD-SWAP", 0, 3500, 1, currency.EMPTYCODE, false)
	require.ErrorIs(t, err, errMissingQuantity)

	result, err := ok.CurrencyUnitConvert(contextGenerate(), perpetualSwapTP.String(), 1, 3500, 1, currency.EMPTYCODE, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// Trading related endpoints test functions.
func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	result, err := ok.GetSupportCoins(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTakerVolume(contextGenerate(), currency.BTC, "", "", time.Time{}, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetTakerVolume(contextGenerate(), currency.BTC, okxInstTypeSpot, "", time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	result, err := ok.GetMarginLendingRatio(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	result, err := ok.GetLongShortRatio(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	result, err := ok.GetContractsOpenInterestAndVolume(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	result, err := ok.GetOptionsOpenInterestAndVolume(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPutCallRatio(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	result, err := ok.GetOpenInterestAndVolumeExpiry(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterestAndVolumeStrike(contextGenerate(), currency.BTC, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, errMissingExpiryTimeParameter)

	result, err := ok.GetOpenInterestAndVolumeStrike(contextGenerate(), currency.BTC, time.Now(), kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	result, err := ok.GetTakerFlow(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &PlaceOrderRequestParam{
		ReduceOnly: true,
		AssetType:  asset.Margin,
	}
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.Lower()
	arg.TradeMode = "abc"
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1
	arg.QuantityType = "abcd"
	_, err = ok.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errCurrencyQuantitTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         order.Buy.Lower(),
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
		AssetType:    asset.Margin,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         order.Buy.Lower(),
		PositionSide: "long",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
		AssetType:    asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const (
	instrumentJSON                                = `{"alias":"","baseCcy":"","category":"1","ctMult":"1","ctType":"linear","ctVal":"0.0001","ctValCcy":"BTC","expTime":"","instFamily":"BTC-USDC","instId":"BTC-USDC-SWAP","instType":"SWAP","lever":"125","listTime":"1666076190000","lotSz":"1","maxIcebergSz":"100000000.0000000000000000","maxLmtSz":"100000000","maxMktSz":"85000","maxStopSz":"85000","maxTriggerSz":"100000000.0000000000000000","maxTwapSz":"","minSz":"1","optType":"","quoteCcy":"","settleCcy":"USDC","state":"live","stk":"","tickSz":"0.1","uly":"BTC-USDC"}`
	placeOrderArgs                                = `[{"side": "buy","instId": "BTC-USDT","tdMode": "cash","ordType": "market","sz": "100"},{"side": "buy","instId": "LTC-USDT","tdMode": "cash","ordType": "market","sz": "1"}]`
	calculateOrderbookChecksumUpdateOrderbookJSON = `{"Bids":[{"Amount":56,"Price":0.07014,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":608,"Price":0.07011,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07009,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.07006,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2347,"Price":0.07004,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":279,"Price":0.07003,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":52,"Price":0.07001,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":91,"Price":0.06997,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4242,"Price":0.06996,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":486,"Price":0.06995,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.06992,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.06991,"ID":0,"Period":0,"LiquidationOrders":0,
	"OrderCount":0},{"Amount":7518,"Price":0.06988,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.06976,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":71,"Price":0.06975,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1086,"Price":0.06973,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":513,"Price":0.06961,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4603,"Price":0.06959,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.0695,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3043,"Price":0.06946,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":103,"Price":0.06939,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5053,"Price":0.0693,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5039,"Price":0.06909,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5037,"Price":0.06888,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06886,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5008,"Price":0.06867,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5065,"Price":0.06846,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1572,"Price":0.06826,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1565,"Price":0.06801,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":67,"Price":0.06748,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":111,"Price":0.0674,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10038,"Price":0.0672,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.06652,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06625,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10924,"Price":0.06619,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05986,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05387,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04848,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04363,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Asks":[{"Amount":5,"Price":0.07026,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":765,"Price":0.07027,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07028,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.0703,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":280,"Price":0.07034,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2255,"Price":0.07035,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":28,"Price":0.07036,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.07037,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":137,"Price":0.07039,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":48,"Price":0.0704,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":32,"Price":0.07041,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3985,"Price":0.07043,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":257,"Price":0.07057,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":7870,"Price":0.07058,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.07059,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4539,"Price":0.07061,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1438,"Price":0.07068,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3162,"Price":0.07088,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":99,"Price":0.07104,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5018,"Price":0.07108,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1540,"Price":0.07115,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5080,"Price":0.07129,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1512,"Price":0.07145,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5016,"Price":0.0715,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5026,"Price":0.07171,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5062,"Price":0.07192,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1517,"Price":0.07197,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1511,"Price":0.0726,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10376,"Price":0.07314,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.07354,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10277,"Price":0.07466,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07626,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07636,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.0809,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.08899,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.09789,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.10768,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Exchange":"Okx","Pair":"BTC-USDT","Asset":"spot","LastUpdated":"0001-01-01T00:00:00Z","LastUpdateID":0,"PriceDuplication":false,"IsFundingRate":false,"RestSnapshot":false,"IDAlignment":false}`
	placeMultipleOrderParamsJSON = `[{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b159","side":"buy","ordType":"limit","px":"2.15","sz":"2"},{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b15","side":"buy","ordType":"limit","px":"2.15","sz":"2"}]`
)

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	var params []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeMultipleOrderParamsJSON), &params)
	assert.NoError(t, err)

	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{})
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	arg := PlaceOrderRequestParam{
		ReduceOnly: true,
	}
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	arg.TradeMode = "abc"
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = OkxOrderLimit
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = ok.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceMultipleOrders(contextGenerate(), params)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{OrderID: "12321312312"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{InstrumentID: "BTC-USDT"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelSingleOrder(contextGenerate(),
		&CancelOrderRequestParam{
			InstrumentID: spotTP.String(),
			OrderID:      "2510789768709120",
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := CancelOrderRequestParam{}
	_, err = ok.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{
		{
			InstrumentID: spotTP.String(),
			OrderID:      "2510789768709120",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &AmendOrderRequestParams{}
	_, err = ok.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	_, err = ok.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendOrder(contextGenerate(), &AmendOrderRequestParams{
		InstrumentID: spotTP.String(),
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := AmendOrderRequestParams{
		NewPriceInUSD: 1233,
	}
	_, err = ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123212"
	_, err = ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{{
		InstrumentID: spotTP.String(),
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	_, err := ok.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{MarginMode: "cross"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{InstrumentID: "BTC-USDT", MarginMode: "abc"})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{
		InstrumentID: spotTP.String(),
		MarginMode:   "cross",
		Currency:     "BTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{OrderID: "1234"})
	assert.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{InstrumentID: "BTC-USDT"})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{InstrumentID: "SUI-USDT", OrderID: "1974857619964870656"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderList(contextGenerate(), &OrderListRequestParams{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOrderList(contextGenerate(), &OrderListRequestParams{Limit: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet7And3MonthDayOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.getOrderHistory(context.Background(), &OrderHistoryRequestParams{}, "", request.UnAuth)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.getOrderHistory(context.Background(), &OrderHistoryRequestParams{Category: "abc"}, "", request.UnAuth)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequestParams{OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"}})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGet3MonthOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequestParams{OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.getTransactionDetails(context.Background(), &TransactionDetailRequestParams{}, "", request.UnAuth)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.getTransactionDetails(context.Background(), &TransactionDetailRequestParams{Limit: 10}, "", request.UnAuth)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetTransactionDetailsLast3Days(contextGenerate(), &TransactionDetailRequestParams{InstrumentType: "MARGIN", Limit: 1})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = ok.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequestParams{InstrumentType: "MARGIN"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceAlgoOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = ok.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = ok.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = ok.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = ok.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
}

func TestStopOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceStopOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.TakeProfitTriggerPrice = 123
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	arg.TakeProfitTriggerPriceType = "last_price"
	arg.AlgoClientOrderID = "12345"
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = ok.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceStopOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID:          "681096944655273984",
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "conditional",
		Side:                       order.Sell.Lower(),
		TradeMode:                  "isolated",
		Size:                       12,
		TakeProfitTriggerPrice:     12335,
		TakeProfitOrderPrice:       1234,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceIcebergOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{OrderType: "iceberg"})
	require.ErrorIs(t, err, errMissingSizeLimit)
	_, err = ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{OrderType: "iceberg", SizeLimit: 123})
	require.ErrorIs(t, err, errInvalidPriceLimit)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		LimitPrice:        100.22, SizeLimit: 9999.9,
		PriceSpread: 0.04, InstrumentID: "BTC-USDT",
		OrderType: "iceberg", Side: order.Buy.Lower(),
		TradeMode: "isolated", Size: 6,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{OrderType: "twap"})
	require.ErrorIs(t, err, errMissingSizeLimit)

	_, err = ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{SizeLimit: 2, OrderType: "twap"})
	require.ErrorIs(t, err, errInvalidPriceLimit)

	_, err = ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{SizeLimit: 2, OrderType: "twap", LimitPrice: 1234.5})
	require.ErrorIs(t, err, errMissingIntervalValue)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      "BTC-USDT",
		LimitPrice:        100.22,
		SizeLimit:         9999.9,
		OrderType:         "twap",
		PriceSpread:       0.4,
		TradeMode:         "cross",
		Side:              order.Sell.Lower(),
		Size:              6,
		TimeInterval:      kline.ThreeDay,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceChaseAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceChaseAlgoOrder(context.Background(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	_, err = ok.PlaceTWAPOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "chase"
	arg.MaxChaseType = "percentage"
	_, err = ok.PlaceChaseAlgoOrder(context.Background(), arg)
	require.ErrorIs(t, err, errPriceTrackingNotSet)

	arg.MaxChaseType = "percentage"
	arg.MaxChaseValue = .5
	_, err = ok.PlaceChaseAlgoOrder(context.Background(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "BTC-USDT"
	_, err = ok.PlaceChaseAlgoOrder(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = ok.PlaceChaseAlgoOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	_, err = ok.PlaceChaseAlgoOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceChaseAlgoOrder(context.Background(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      "BTC-USDT",
		LimitPrice:        100.22,
		OrderType:         "chase",
		TradeMode:         "cross",
		Side:              order.Sell.Lower(),
		MaxChaseType:      "distance",
		MaxChaseValue:     .5,
		Size:              6,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTriggerAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.TriggerAlgoOrder(context.Background(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.TriggerAlgoOrder(context.Background(), &AlgoOrderParams{AlgoClientOrderID: "1234"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = ok.TriggerAlgoOrder(context.Background(), &AlgoOrderParams{AlgoClientOrderID: "1234", OrderType: "trigger"})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	_, err = ok.TriggerAlgoOrder(context.Background(), &AlgoOrderParams{AlgoClientOrderID: "1234", OrderType: "trigger", TriggerPrice: 123., TriggerPriceType: "abcd"})
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.TriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		TriggerPriceType:  "mark",
		TriggerPrice:      1234,
		InstrumentID:      "BTC-USDT",
		OrderType:         "trigger",
		Side:              order.Buy.Lower(),
		TradeMode:         "cross",
		Size:              5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceTrailingStopOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{Size: 2})
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = ok.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{Size: 2, OrderType: "move_order_stop"})
	assert.ErrorIs(t, err, errPriceTrackingNotSet)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984", CallbackRatio: 0.01,
		InstrumentID: "BTC-USDT", OrderType: "move_order_stop",
		Side: order.Buy.Lower(), TradeMode: "isolated",
		Size: 2, ActivePrice: 1234})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	arg := AlgoOrderCancelParams{}
	_, err := ok.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.AlgoOrderID = "90994943"
	_, err = ok.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "BTC-USDT"
	arg.AlgoOrderID = ""
	_, err = ok.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{
		{
			InstrumentID: "BTC-USDT",
			AlgoOrderID:  "90994943",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelAdvanceAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{InstrumentID: "90994943"}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{AlgoOrderID: "90994943"}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{
		InstrumentID: "BTC-USDT",
		AlgoOrderID:  "90994943",
	}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAlgoOrderList(contextGenerate(), "", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAlgoOrderList(contextGenerate(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAlgoOrderHistory(contextGenerate(), "", "effective", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = ok.GetAlgoOrderHistory(contextGenerate(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, errMissingEitherAlgoIDOrState)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAlgoOrderHistory(contextGenerate(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetEasyConvertCurrencyList(contextGenerate(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOneClickRepayCurrencyList(contextGenerate(), "cross")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceEasyConvert(contextGenerate(),
		PlaceEasyConvertParam{
			FromCurrency: []string{"BTC"},
			ToCurrency:   "USDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetEasyConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOneClickRepayHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	_, err := ok.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{DebtCurrency: []string{}, RepayCurrency: "USDT"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{DebtCurrency: []string{"BTC"}, RepayCurrency: ""})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{
		DebtCurrency:  []string{"BTC"},
		RepayCurrency: "USDT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCounterparties(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const createRfqInputJSON = `{"anonymous": true,"counterparties":["Trader1","Trader2"],"clRfqId":"rfq01","legs":[{"sz":"25","side":"buy","instId":"BTCUSD-221208-100000-C"},{"sz":"150","side":"buy","instId":"ETH-USDT","tgtCcy":"base_ccy"}]}`

func TestCreateRfq(t *testing.T) {
	t.Parallel()
	var input CreateRfqInput
	err := json.Unmarshal([]byte(createRfqInputJSON), &input)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CreateRfq(contextGenerate(), input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelRfq(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelRfq(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelRfq(context.Background(), "", "somersdjskfjsdkfjxvxv")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMultipleCancelRfq(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelMultipleRfqs(contextGenerate(), &CancelRfqRequestsParam{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelMultipleRfqs(contextGenerate(), &CancelRfqRequestsParam{ClientRfqIDs: []string{"somersdjskfjsdkfjxvxv"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllRfqs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAllRfqs(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.ExecuteQuote(contextGenerate(), "", "")
	assert.ErrorIs(t, err, errMissingRfqIDOrQuoteID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ExecuteQuote(contextGenerate(), "22540", "84073")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuoteProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetQuoteProducts(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	_, err := ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{{InstrumentType: "ABC"}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	arg := SetQuoteProductParam{InstrumentType: "SWAP"}
	_, err = ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errMissingMakerInstrumentSettings)

	data := MakerInstrumentSetting{MaxBlockSize: 10000, MakerPriceBand: 5}
	arg.Data = []MakerInstrumentSetting{data}
	_, err = ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errInvalidUnderlying)

	arg.InstrumentType = "SPOT"
	data = MakerInstrumentSetting{Underlying: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5}
	arg.Data = []MakerInstrumentSetting{data}
	_, err = ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{
		{
			InstrumentType: "SWAP",
			Data: []MakerInstrumentSetting{
				{
					Underlying:     "BTC-USD",
					MaxBlockSize:   10000,
					MakerPriceBand: 5,
				},
				{
					Underlying: "ETH-USDT",
				},
			},
		}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetRFQMMPStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ResetRFQMMPStatus(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.CreateQuote(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CreateQuoteParams{}
	_, err = ok.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingRfqID)

	arg.RfqID = "123456789"
	_, err = ok.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.QuoteSide = "sell"
	_, err = ok.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingLegs)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CreateQuote(contextGenerate(), &CreateQuoteParams{
		RfqID:     "12345",
		QuoteSide: order.Buy.Lower(),
		Legs: []QuoteLeg{
			{
				Price:          1234,
				SizeOfQuoteLeg: 2,
				InstrumentID:   "SOL-USD-220909",
				Side:           order.Sell,
			},
			{
				Price:          1234,
				SizeOfQuoteLeg: 1,
				InstrumentID:   "SOL-USD-220909",
				Side:           order.Buy,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelQuote(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelQuote(contextGenerate(), "1234", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = ok.CancelQuote(contextGenerate(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{
		QuoteIDs: []string{"1150", "1151", "1152"},
		// Block trades require a minimum of $100,000 in assets in your trading account
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	tt, err := ok.CancelAllQuotes(contextGenerate())
	require.NoError(t, err)
	assert.NotEmpty(t, tt)
}

func TestGetRfqs(t *testing.T) {
	t.Parallel()
	_, err := ok.GetRfqs(contextGenerate(), &RfqRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRfqs(contextGenerate(), &RfqRequestParams{
		Limit: 1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	_, err := ok.GetQuotes(contextGenerate(), &QuoteRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetQuotes(contextGenerate(), &QuoteRequestParams{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRfqTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetRfqTrades(contextGenerate(), &RfqTradesRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRfqTrades(contextGenerate(), &RfqTradesRequestParams{Limit: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicRFQTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetPublicRFQTrades(contextGenerate(), "", "", 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFundingCurrencies(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBalance(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNonTradableAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetNonTradableAssets(context.Background(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountAssetValuation(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	_, err := ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		FundingRecipientAddress: "6", FundingSourceAddress: "18", Currency: currency.BTC})
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount: 12.000, FundingRecipientAddress: "6",
		FundingSourceAddress: "18", Currency: currency.EMPTYCODE})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount: 12.000, FundingRecipientAddress: "6",
		FundingSourceAddress: "", Currency: currency.BTC})
	assert.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount:                  12.000,
		FundingRecipientAddress: "6",
		FundingSourceAddress:    "18",
		Currency:                currency.BTC,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFundsTransferState(contextGenerate(), "", "", 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFundsTransferState(contextGenerate(), "754147", "1232", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAssetBillsDetails(contextGenerate(), currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLightningDeposits(contextGenerate(), currency.EMPTYCODE, 1.00, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.GetLightningDeposits(contextGenerate(), currency.BTC, 0, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLightningDeposits(contextGenerate(), currency.BTC, 1.00, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCurrencyDepositAddress(contextGenerate(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCurrencyDepositAddress(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCurrencyDepositHistory(contextGenerate(), currency.BTC, "", "", "", "271", time.Time{}, time.Time{}, 0, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := ok.Withdrawal(contextGenerate(), &WithdrawalInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.EMPTYCODE, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.Withdrawal(contextGenerate(), &WithdrawalInput{TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.BTC, ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, errAddressRequired)
	_, err = ok.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.Withdrawal(contextGenerate(), &WithdrawalInput{
		Amount:                0.1,
		WithdrawalDestination: "4",
		TransactionFee:        0.00005,
		Currency:              currency.BTC,
		ChainName:             "BTC-Bitcoin",
		ToAddress:             core.BitcoinDonationAddress,
		RecipientInformation: &WithdrawalRecipientInformation{
			WalletType:        "exchange",
			ExchangeID:        "did:ethr:0xfeb4f99829a9acdf52979abee87e83addf22a7e1",
			ReceiverFirstName: "Bruce",
			ReceiverLastName:  "Wayne",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestLightningWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := ok.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{
		Invoice: "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz", Currency: currency.EMPTYCODE})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{
		Currency: currency.BTC,
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelWithdrawal(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingValidWithdrawalID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelWithdrawal(contextGenerate(), "fjasdfkjasdk")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetWithdrawalHistory(contextGenerate(), currency.BTC, "", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SmallAssetsConvert(contextGenerate(), []string{"BTC", "USDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSavingBalance(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	_, err := ok.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SavingsPurchaseRedemptionInput{Rate: 1}
	_, err = ok.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.BTC
	_, err = ok.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 123.4
	_, err = ok.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   currency.BTC,
		Rate:       1,
		ActionType: "purchase",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = ok.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   currency.BTC,
		Rate:       1,
		ActionType: "redempt",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	_, err := ok.SetLendingRate(contextGenerate(), &LendingRate{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.EMPTYCODE, Rate: 2})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.BTC})
	require.ErrorIs(t, err, errLendingRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.BTC, Rate: 2})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLendingHistory(contextGenerate(), currency.USDT, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPublicBorrowInfo(contextGenerate(), currency.EMPTYCODE)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetPublicBorrowInfo(context.Background(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicBorrowHistory(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPublicBorrowHistory(context.Background(), currency.USDT, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMonthlyStatement(context.Background(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	_, err = ok.GetMonthlyStatement(context.Background(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMonthlyStatement(context.Background(), "Jan")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyForMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := ok.ApplyForMonthlyStatement(context.Background(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ApplyForMonthlyStatement(context.Background(), "Jan")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetConvertCurrencies(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := ok.GetConvertCurrencyPair(contextGenerate(), currency.EMPTYCODE, currency.BTC)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.GetConvertCurrencyPair(contextGenerate(), currency.USDT, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetConvertCurrencyPair(contextGenerate(), currency.USDT, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.EstimateQuote(contextGenerate(), &EstimateQuoteRequestInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &EstimateQuoteRequestInput{Tag: "abcd"}
	_, err = ok.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BaseCurrency = currency.BTC
	_, err = ok.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	arg.QuoteCurrency = currency.BTC
	_, err = ok.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Sell.Lower()
	_, err = ok.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	arg.RFQAmount = 30
	_, err = ok.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.EstimateQuote(contextGenerate(), &EstimateQuoteRequestInput{
		BaseCurrency:  currency.BTC,
		QuoteCurrency: currency.USDT,
		Side:          order.Sell.Lower(),
		RFQAmount:     30,
		RFQSzCurrency: "USDT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	_, err := ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{Tag: "123"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          order.Buy.Lower(),
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          order.Buy.Lower(),
		Size:          2,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          order.Buy.Lower(),
		Size:          2,
		SizeCurrency:  currency.USDT,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          order.Buy.Lower(),
		Size:          2,
		SizeCurrency:  currency.USDT,
		QuoteID:       "16461885104612381",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNonZeroAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.AccountBalance(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetPositions(contextGenerate(), "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetPositionsHistory(contextGenerate(), "", "", "", "1234213123", 0, 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountAndPositionRisk(contextGenerate(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailQueryParameter{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailQueryParameter{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyBillDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.ApplyBillDetails(context.Background(), "", "Q2")
	require.ErrorIs(t, err, errYearRequired)
	_, err = ok.ApplyBillDetails(context.Background(), "2023", "")
	require.ErrorIs(t, err, errQuarterValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.ApplyBillDetails(context.Background(), "2023", "Q2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBillsHistoryArchive(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBillsHistoryArchive(context.Background(), "", "Q2")
	require.ErrorIs(t, err, errYearRequired)
	_, err = ok.GetBillsHistoryArchive(context.Background(), "2023", "")
	require.ErrorIs(t, err, errQuarterValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBillsHistoryArchive(context.Background(), "2023", "Q2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountConfiguration(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	_, err := ok.SetPositionMode(contextGenerate(), "")
	require.ErrorIs(t, err, errInvalidPositionMode)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.SetPositionMode(contextGenerate(), "net_mode")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverageRate(t *testing.T) {
	t.Parallel()
	_, err := ok.SetLeverageRate(contextGenerate(), &SetLeverageInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.SetLeverageRate(contextGenerate(), &SetLeverageInput{Leverage: 5, MarginMode: "isolated", AssetType: asset.Futures})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	_, err = ok.SetLeverageRate(contextGenerate(), &SetLeverageInput{
		Currency:     currency.USDT,
		Leverage:     5,
		MarginMode:   "isolated",
		InstrumentID: "BTC-USDT",
		AssetType:    asset.Futures,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err = ok.SetLeverageRate(contextGenerate(), &SetLeverageInput{
		Currency:     currency.USDT,
		Leverage:     5,
		MarginMode:   "cross",
		InstrumentID: "BTC-USDT",
	})
	assert.True(t, err == nil || errors.Is(err, common.ErrNoResponse))
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, "", "cross", "", 5, true)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, "BTC-USDT", "", "", 5, true)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, "BTC-USDT", "cross", "", 5, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, "", "cross", "", true, false, 123)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, "BTC-USDT", "", "", true, false, 123)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, "BTC-USDT", "cross", "", true, false, 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	arg := &IncreaseDecreaseMarginInput{Currency: "USD"}
	_, err := ok.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "BTC-USDT"
	_, err = ok.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = ok.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.MarginBalanceType = "reduce"
	_, err = ok.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.IncreaseDecreaseMargin(contextGenerate(), &IncreaseDecreaseMarginInput{
		InstrumentID:      "BTC-USDT",
		PositionSide:      "long",
		MarginBalanceType: "add",
		Amount:            1000,
		Currency:          "USD",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeverageRate(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeverageRate(contextGenerate(), "", "cross", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetLeverageRate(contextGenerate(), "BTC-USDT", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLeverageRate(contextGenerate(), "BTC-USDT", "cross", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMaximumLoanOfInstrument(contextGenerate(), "", "isolated", currency.ZRX)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetMaximumLoanOfInstrument(contextGenerate(), "ZRX-BTC", "", currency.ZRX)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMaximumLoanOfInstrument(contextGenerate(), spotTP.String(), "isolated", currency.ZRX)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTradeFee(contextGenerate(), "", "", "", "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetTradeFee(contextGenerate(), okxInstTypeSpot, "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetInterestAccruedData(contextGenerate(), 0, 1, currency.EMPTYCODE, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetInterestRate(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	_, err := ok.SetGreeks(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingValidGreeksType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetGreeks(contextGenerate(), "PA")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	_, err := ok.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{IsoMode: "", InstrumentType: "MARGIN"})
	require.ErrorIs(t, err, errMissingIsolatedMarginTradingSetting)
	_, err = ok.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{IsoMode: "autonomy", InstrumentType: ""})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{
		IsoMode:        "autonomy",
		InstrumentType: "MARGIN",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMaximumWithdrawals(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountRiskState(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	_, err := ok.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.EMPTYCODE, Side: "borrow", Amount: 12})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "", Amount: 12})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ok.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "borrow", Amount: 0})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "borrow", Amount: 12})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBorrowAndRepayHistoryForVIPLoans(contextGenerate(), currency.EMPTYCODE, time.Time{}, time.Time{}, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBorrowInterestAndLimit(contextGenerate(), 1, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFixedLoanBorrowLimit(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowQuote(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "", "30D", "123423423", 1, .4)
	require.ErrorIs(t, err, errBorrowTypeRequired)
	_, err = ok.GetFixedLoanBorrowQuote(context.Background(), currency.EMPTYCODE, "normal", "30D", "123423423", 1, .4)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "normal", "30D", "", 0, .4)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "normal", "30D", "123423423", 1, 0)
	require.ErrorIs(t, err, errMaxRateRequired)
	_, err = ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "normal", "", "123423423", 1, .4)
	require.ErrorIs(t, err, errLendingTermIsRequired)
	_, err = ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "reborrow", "30D", "", 1, .4)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFixedLoanBorrowQuote(context.Background(), currency.USDT, "normal", "30D", "123423423", 1, .4)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceFixedLoanBorrowingOrder(context.Background(), currency.EMPTYCODE, 1, .3, .2, "30D", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.PlaceFixedLoanBorrowingOrder(context.Background(), currency.USDT, 0, .3, .2, "30D", false)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.PlaceFixedLoanBorrowingOrder(context.Background(), currency.USDT, 1, 0, .2, "30D", false)
	require.ErrorIs(t, err, errMaxRateRequired)
	_, err = ok.PlaceFixedLoanBorrowingOrder(context.Background(), currency.USDT, 1, .3, .2, "", false)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceFixedLoanBorrowingOrder(context.Background(), currency.USDT, 1, .3, .2, "30D", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendFixedLoanBorrowingOrder(context.Background(), "", false, .4)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendFixedLoanBorrowingOrder(context.Background(), "12312312", false, .4)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualRenewFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.ManualRenewFixedLoanBorrowingOrder(context.Background(), "", .3)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.ManualRenewFixedLoanBorrowingOrder(context.Background(), "12312312", 0)
	require.ErrorIs(t, err, errMaxRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ManualRenewFixedLoanBorrowingOrder(context.Background(), "12312312", .3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.RepayFixedLoanBorrowingOrder(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.RepayFixedLoanBorrowingOrder(context.Background(), "12321")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertFixedLoanToMarketLoan(t *testing.T) {
	t.Parallel()
	_, err := ok.ConvertFixedLoanToMarketLoan(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ConvertFixedLoanToMarketLoan(context.Background(), "12321")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReduceLiabilitiesForFixedLoan(t *testing.T) {
	t.Parallel()
	_, err := ok.ReduceLiabilitiesForFixedLoan(context.Background(), "", false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ReduceLiabilitiesForFixedLoan(context.Background(), "123123", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFixedLoanBorrowOrderList(context.Background(), currency.USDT, "1231231", "8", "30D", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := ok.ManualBorrowOrRepay(context.Background(), currency.EMPTYCODE, "borrow", 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.ManualBorrowOrRepay(context.Background(), currency.USDT, "", 1)
	require.ErrorIs(t, err, errLendingSideRequired)
	_, err = ok.ManualBorrowOrRepay(context.Background(), currency.USDT, "borrow", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ManualBorrowOrRepay(context.Background(), currency.USDT, "borrow", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAutoRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetAutoRepay(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBorrowRepayHistory(context.Background(), currency.ETH, "auto_borrow", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewPositionBuilder(t *testing.T) {
	t.Parallel()
	_, err := ok.NewPositionBuilder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.NewPositionBuilder(context.Background(), &PositionBuilderParam{
		InclRealPosAndEq: false,
		SimPos: []SimulatedPosition{
			{
				Position:     "-10",
				InstrumentID: "BTC-USDT-SWAP",
			},
			{
				Position:     "10",
				InstrumentID: "LTC-USDT-SWAP",
			},
		},
		SimAsset: []SimulatedAsset{
			{
				Currency: "USDT",
				Amount:   100,
			},
		},
		SpotOffsetType: "1",
		GreeksType:     "CASH",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetRiskOffsetAmount(t *testing.T) {
	t.Parallel()
	_, err := ok.SetRiskOffsetAmount(context.Background(), currency.EMPTYCODE, 123)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.SetRiskOffsetAmount(context.Background(), currency.USDT, 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.SetRiskOffsetAmount(context.Background(), currency.USDT, 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetGreeks(contextGenerate(), currency.EMPTYCODE)
	assert.False(t, err != nil && !strings.Contains(err.Error(), "Unsupported operation"), err)
}

func TestGetPMLimitation(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPMPositionLimitation(contextGenerate(), "", "BTC-USDT", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = ok.GetPMPositionLimitation(contextGenerate(), "SWAP", "", "")
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetPMPositionLimitation(contextGenerate(), "SWAP", "BTC-USDT", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.ViewSubAccountList(contextGenerate(), false, "", time.Time{}, time.Time{}, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := ok.ResetSubAccountAPIKey(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{APIKey: apiKey, APIKeyPermission: "trade"})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{SubAccountName: "sam", APIKey: "", APIKeyPermission: "trade"})
	require.ErrorIs(t, err, errInvalidAPIKey)
	_, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{IP: "1.2.3.", SubAccountName: "sam", APIKeyPermission: "trade", APIKey: "sample-api-key"})
	require.ErrorIs(t, err, errInvalidIPAddress)
	_, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{APIKeyPermission: "abc", APIKey: "sample-api-key", SubAccountName: "sam"})
	require.ErrorIs(t, err, errInvalidAPIKeyPermission)
	_, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		Permissions: []string{"abc"}, SubAccountName: "sam",
		APIKey: "sample-api-key"})
	require.ErrorIs(t, err, errInvalidAPIKeyPermission)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		SubAccountName:   "sam",
		APIKey:           apiKey,
		APIKeyPermission: "trade",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		SubAccountName: "sam",
		APIKey:         apiKey,
		Permissions:    []string{"trade", "read"},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubaccountTradingBalance(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSubaccountTradingBalance(contextGenerate(), "")
	assert.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSubaccountTradingBalance(contextGenerate(), "test1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSubaccountFundingBalance(contextGenerate(), "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSubaccountFundingBalance(contextGenerate(), "test1", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetSubAccountMaximumWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSubAccountMaximumWithdrawal(context.Background(), "", currency.BTC)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSubAccountMaximumWithdrawal(context.Background(), "test1", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.HistoryOfSubaccountTransfer(contextGenerate(), currency.EMPTYCODE, "0", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoryOfManagedSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetHistoryOfManagedSubAccountTransfer(context.Background(), currency.BTC, "", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), &SubAccountAssetTransferParams{Currency: currency.BTC, Amount: 1200, From: 9, To: 9, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true})
	require.ErrorIs(t, err, errInvalidSubaccount)
	_, err = ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), &SubAccountAssetTransferParams{Currency: currency.BTC, Amount: 1200, From: 8, To: 8, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true})
	require.ErrorIs(t, err, errInvalidSubaccount)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), &SubAccountAssetTransferParams{Currency: currency.BTC, Amount: 1200, From: 6, To: 6, FromSubAccount: "test1", ToSubAccount: "test2", LoanTransfer: true})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	_, err := ok.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{CanTransOut: true})
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{SubAcct: "Test1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCustodyTradingSubaccountList(contextGenerate(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetSubAccountVIPLoanAllocation(t *testing.T) {
	t.Parallel()
	_, err := ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := subAccountVIPLoanAllocationInfo{}
	_, err = ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.LoanAlloc = 123
	_, err = ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.LoanAlloc = -1
	arg.SubAcct = "sams"
	_, err = ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, errInvalidLoanAllocationValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{
		Enable: true,
		Alloc: []subAccountVIPLoanAllocationInfo{
			{
				SubAcct:   "subAcct1",
				LoanAlloc: 20.01,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSubAccountBorrowInterestAndLimit(context.Background(), "", currency.ETH)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSubAccountBorrowInterestAndLimit(context.Background(), "123456", currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ETH Staking

func TestGetProductInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetProductInfo(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPurcahseETHStaking(t *testing.T) {
	t.Parallel()
	err := ok.PurchaseETHStaking(context.Background(), 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err = ok.PurchaseETHStaking(context.Background(), 100)
	assert.NoError(t, err)
}

// RedeemETHStaking
func TestRedeemETHStaking(t *testing.T) {
	t.Parallel()
	err := ok.RedeemETHStaking(context.Background(), 0)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err = ok.RedeemETHStaking(context.Background(), 100)
	assert.NoError(t, err)
}

func TestGetBETHAssetsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBETHAssetsBalance(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPurchaseAndRedeemHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPurchaseAndRedeemHistory(context.Background(), "", "pending", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetPurchaseAndRedeemHistory(context.Background(), "purchase", "pending", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAPYHistory(t *testing.T) {
	t.Parallel()
	result, err := ok.GetAPYHistory(context.Background(), 34)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const gridTradingPlaceOrder = `{"instId": "BTC-USD-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input GridAlgoOrder
	err := json.Unmarshal([]byte(gridTradingPlaceOrder), &input)
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceGridAlgoOrder(contextGenerate(), &input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const gridOrderAmendAlgo = `{
    "algoId":"448965992920907776",
    "instId":"BTC-USDT",
    "slTriggerPx":"1200",
    "tpTriggerPx":""
}`

func TestAmendGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input *GridAlgoOrderAmend
	err := json.Unmarshal([]byte(gridOrderAmendAlgo), &input)
	require.NoError(t, err)

	arg := &GridAlgoOrderAmend{}
	_, err = ok.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.TakeProfitTriggerPrice = 1234.5
	_, err = ok.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "560472804207104000"
	_, err = ok.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendGridAlgoOrder(contextGenerate(), input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const stopGridAlgoOrderJSON = `{"algoId":"198273485",	"instId":"BTC-USDT",	"stopType":"1",	"algoOrdType":"grid"}`

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var resp StopGridAlgoOrderRequest
	err := json.Unmarshal([]byte(stopGridAlgoOrderJSON), &resp)
	require.NoError(t, err)

	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := StopGridAlgoOrderRequest{}
	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.StopType = 20
	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "algo_id"
	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	arg.AlgoOrderType = AlgoOrdTypeGrid
	_, err = ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingValidStopType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{resp})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	_, err := ok.GetGridAlgoOrdersList(contextGenerate(), "abc", "", "", "", "", "", 1)
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetGridAlgoOrdersList(contextGenerate(), "grid", "", "", "", "", "", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetGridAlgoOrderHistory(contextGenerate(), "abc", "", "", "", "", "", 1)
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetGridAlgoOrderHistory(contextGenerate(), "contract_grid", "", "", "", "", "", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.GetGridAlgoOrderDetails(contextGenerate(), "grid", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetGridAlgoOrderDetails(contextGenerate(), "grid", "7878")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.GetGridAlgoSubOrders(contextGenerate(), "", "", "", "", "", "", 2)
	require.ErrorIs(t, err, errMissingAlgoOrderType)
	_, err = ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "", "", "", "", "", 2)
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "", "", "", "", 2)
	require.ErrorIs(t, err, errMissingSubOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "live", "", "", "", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	var resp AlgoOrderPosition
	err := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp)
	require.NoError(t, err)
	_, err = ok.GetGridAlgoOrderPositions(contextGenerate(), "", "")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = ok.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", "448965992920907776")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	_, err := ok.SpotGridWithdrawProfit(contextGenerate(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SpotGridWithdrawProfit(contextGenerate(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestComputeMarginBalance(t *testing.T) {
	t.Parallel()
	_, err := ok.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{AlgoID: "123456", AdjustMarginBalanceType: "other"})
	require.ErrorIs(t, err, errInvalidMarginTypeAdjust)

	_, err = ok.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{AdjustMarginBalanceType: "other"})
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{
		AlgoID:                  "123456",
		AdjustMarginBalanceType: "reduce",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginBalance(t *testing.T) {
	t.Parallel()
	arg := &MarginBalanceParam{}
	_, err := ok.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.Amount = 12345
	_, err = ok.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "1234"
	_, err = ok.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidMarginTypeAdjust)

	arg.AdjustMarginBalanceType = "reduce"
	arg.Amount = 0
	_, err = ok.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AdjustMarginBalance(contextGenerate(), &MarginBalanceParam{
		AlgoID:                  "1234",
		AdjustMarginBalanceType: "reduce",
		Amount:                  12345,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const gridAIParamJSON = `{"algoOrdType": "grid","annualizedRate": "1.5849","ccy": "USDT","direction": "",	"duration": "7D","gridNum": "5","instId": "BTC-USDT","lever": "0","maxPx": "21373.3","minInvestment": "0.89557758",	"minPx": "15544.2",	"perMaxProfitRate": "0.0733865364573281","perMinProfitRate": "0.0561101403446263","runType": "1"}`

func TestGetGridAIParameter(t *testing.T) {
	t.Parallel()
	var response GridAIParameterResponse
	err := json.Unmarshal([]byte(gridAIParamJSON), &response)
	require.NoError(t, err)

	_, err = ok.GetGridAIParameter(contextGenerate(), "", "BTC-USDT", "", "")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = ok.GetGridAIParameter(contextGenerate(), "grid", "", "", "")
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.GetGridAIParameter(contextGenerate(), "contract_grid", "BTC-USDT", "", "")
	require.ErrorIs(t, err, errMissingRequiredArgumentDirection)
	_, err = ok.GetGridAIParameter(contextGenerate(), "grid", "BTC-USDT", "", "12M")
	require.ErrorIs(t, err, errInvalidDuration)

	result, err := ok.GetGridAIParameter(contextGenerate(), "grid", "BTC-USDT", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetOffers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOffers(contextGenerate(), "", "", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPurchase(t *testing.T) {
	t.Parallel()
	_, err := ok.Purchase(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.Purchase(contextGenerate(), &PurchaseRequestParam{Term: 2})
	require.ErrorIs(t, err, errMissingRequiredParameter)
	_, err = ok.Purchase(contextGenerate(), &PurchaseRequestParam{ProductID: "1234", Term: 2, InvestData: []PurchaseInvestDataItem{{Amount: 1}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.Purchase(contextGenerate(), &PurchaseRequestParam{ProductID: "1234", Term: 2, InvestData: []PurchaseInvestDataItem{{Currency: currency.USDT}}})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.Purchase(contextGenerate(), &PurchaseRequestParam{
		ProductID: "1234",
		InvestData: []PurchaseInvestDataItem{
			{
				Currency: currency.BTC,
				Amount:   100,
			},
			{
				Currency: currency.ETH,
				Amount:   100,
			},
		},
		Term: 30,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRedeem(t *testing.T) {
	t.Parallel()
	_, err := ok.Redeem(contextGenerate(), &RedeemRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.Redeem(contextGenerate(), &RedeemRequestParam{AllowEarlyRedeem: true})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.Redeem(contextGenerate(), &RedeemRequestParam{OrderID: "754147"})
	require.ErrorIs(t, err, errInvalidProtocolType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.Redeem(contextGenerate(), &RedeemRequestParam{
		OrderID:          "754147",
		ProtocolType:     "defi",
		AllowEarlyRedeem: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelPurchaseOrRedemption(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{ProtocolType: "defi"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{OrderID: "754147"})
	require.ErrorIs(t, err, errInvalidProtocolType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{
		OrderID:      "754147",
		ProtocolType: "defi",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetEarnActiveOrders(contextGenerate(), "", "", "", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFundingOrderHistory(contextGenerate(), "", "", currency.EMPTYCODE, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	result, err := ok.SystemStatusResponse(contextGenerate(), "completed")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	result, err := ok.FetchTradablePairs(contextGenerate(), asset.Options)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.FetchTradablePairs(contextGenerate(), asset.PerpetualSwap)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.FetchTradablePairs(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.FetchTradablePairs(contextGenerate(), asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, ok)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	tests := map[asset.Item][]currency.Pair{
		asset.Spot: {
			currency.NewPair(currency.ETH, currency.USDT),
			currency.NewPair(currency.BTC, currency.USDT),
		},
		asset.Margin: {
			currency.NewPair(currency.ETH, currency.USDT),
			currency.NewPair(currency.ETH, currency.BTC),
		},
	}

	for _, a := range []asset.Item{asset.PerpetualSwap, asset.Futures, asset.Options} {
		pairs, err := ok.FetchTradablePairs(context.Background(), a)
		if assert.NoErrorf(t, err, "FetchTradablePairs should not error for %s", a) {
			tests[a] = []currency.Pair{pairs[0]}
		}
	}

	var err error
	for _, a := range ok.GetAssetTypes(false) {
		err = ok.UpdateOrderExecutionLimits(context.Background(), a)
		if !assert.NoError(t, err) {
			continue
		}

		for _, p := range tests[a] {
			limits, err := ok.GetOrderExecutionLimits(a, p)
			if assert.NoError(t, err, "GetOrderExecutionLimits should not error") {
				require.Positivef(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
				require.Positivef(t, limits.MinimumBaseAmount, "PriceStepIncrementSize should be positive for %s", p)
			}
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	result, err := ok.UpdateTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := ok.UpdateTickers(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	err = ok.UpdateTickers(contextGenerate(), asset.Spread)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	result, err := ok.FetchTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.PerpetualSwap)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.FetchTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	result, err := ok.FetchOrderbook(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	result, err := ok.UpdateOrderbook(contextGenerate(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.Spot)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = ok.UpdateOrderbook(contextGenerate(), spreadTP, asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.UpdateAccountInfo(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.FetchAccountInfo(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountFundingHistory(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetWithdrawalsHistory(contextGenerate(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := ok.GetRecentTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.GetRecentTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var resp []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	require.NoError(t, err)

	arg := &order.Submit{
		Exchange:  ok.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		ClientID:  "yeneOrder",
		AssetType: asset.Binary,
	}
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1000000000
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = spotTP
	arg.AssetType = asset.Futures
	arg.Leverage = -1
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	arg = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  ok.Name,
		Side:      order.Sell,
		Type:      order.Limit,
		Price:     120000,
		Amount:    1000000000,
		ClientID:  "yeneOrder",
		AssetType: asset.Spot,
	}
	result, err := ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Trigger
	arg.TriggerPrice = 11999
	arg.TriggerPriceType = order.LastPrice
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.ConditionalStop
	arg.TriggerPrice = 11999
	arg.TriggerPriceType = order.IndexPrice
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Chase
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.ErrorIs(t, err, order.ErrUnknownTrackingMode)

	arg.TrackingMode = order.Percentage
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.TrackingValue = .5
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.TWAP
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.TrailingStop
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.OCO
	_, err = ok.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{
			Price:      11999,
			LimitPrice: 12000,
		},
		StopLoss: order.RiskManagement{
			Price:      10999,
			LimitPrice: 11000,
		},
	}
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	cp, err := currency.NewPairFromString("BTC-USDT-230630")
	require.NoError(t, err)

	arg = &order.Submit{
		Pair:       cp,
		Exchange:   ok.Name,
		Side:       order.Long,
		Type:       order.Market,
		Amount:     1,
		ClientID:   "hellomoto",
		AssetType:  asset.Futures,
		MarginType: margin.Multi,
	}
	result, err = ok.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	pair, err := currency.NewPairFromString("BTC-USDT-SWAP_BTC-USDT-250328")
	require.NoError(t, err)

	result, err = ok.SubmitOrder(contextGenerate(), &order.Submit{
		Pair:       pair,
		Exchange:   ok.Name,
		Side:       order.Sell,
		Type:       order.Limit,
		Price:      120000,
		Amount:     1,
		ClientID:   "hellomoto",
		AssetType:  asset.Spread,
		MarginType: margin.Multi,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	var arg = &order.Cancel{
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		AssetType:     asset.Binary,
	}
	err := ok.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	err = ok.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = spotTP
	err = ok.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err = ok.CancelOrder(contextGenerate(), &order.Cancel{
		OrderID: "1", WalletAddress: core.BitcoinDonationAddress,
		AccountID: "1", Pair: spotTP, AssetType: asset.Spot})
	assert.NoError(t, err)

	err = ok.CancelOrder(contextGenerate(), &order.Cancel{
		Type:    order.OCO,
		OrderID: "1", WalletAddress: core.BitcoinDonationAddress,
		AccountID: "1", Pair: spotTP, AssetType: asset.Spot})
	assert.NoError(t, err)

	err = ok.CancelOrder(contextGenerate(), &order.Cancel{OrderID: "1",
		WalletAddress: core.BitcoinDonationAddress, AccountID: "1",
		Pair: spreadTP, AssetType: asset.Spread})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelBatchOrders(contextGenerate(), make([]order.Cancel, 21))
	require.ErrorIs(t, err, errExceedLimit)
	_, err = ok.CancelBatchOrders(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	arg := order.Cancel{
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		AssetType:     asset.Binary,
	}
	_, err = ok.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = ok.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	arg.Pair = spotTP
	arg.Type = order.Liquidation
	_, err = ok.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Trigger
	_, err = ok.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.Type = order.Limit
	_, err = ok.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	var orderCancellationParams = []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          spotTP,
			AssetType:     asset.Spot,
		},
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          perpetualSwapTP,
			AssetType:     asset.PerpetualSwap,
		},
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Type:          order.Trigger,
			Pair:          spotTP,
			AssetType:     asset.Spot,
		},
	}
	result, err := ok.CancelBatchOrders(contextGenerate(), orderCancellationParams)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Binary})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spread})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Futures})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spot})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.ModifyOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil)

	arg := &order.Modify{}
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = spotTP
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Spot
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	arg.Type = order.Liquidation
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	arg.Type = order.Trigger
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Type = order.OCO
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	arg = &order.Modify{
		AssetType: asset.Spot,
		Pair:      spotTP,
		OrderID:   "1234",
		Price:     123456.44,
		Amount:    123,
	}
	result, err := ok.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Limit
	result, err = ok.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Trigger
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.TriggerPrice = 12345678
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.OCO
	_, err = ok.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Price: 12345677},
		StopLoss:   order.RiskManagement{Price: 12345667},
	}
	result, err = ok.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.ModifyOrder(contextGenerate(),
		&order.Modify{
			AssetType: asset.Spread,
			Pair:      spotTP,
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	if len(enabled) == 0 {
		t.SkipNow()
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetOrderInfo(contextGenerate(), "123", enabled[0], asset.Futures)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetOrderInfo(contextGenerate(), "123", enabled[0], asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDepositAddress(contextGenerate(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetDepositAddress(contextGenerate(), currency.BTC, core.BitcoinDonationAddress, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Exchange: ok.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	result, err := ok.WithdrawCryptocurrencyFunds(contextGenerate(), &withdrawCryptoRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPairFromInstrumentID(t *testing.T) {
	t.Parallel()
	instruments := []string{
		"BTC-USDT",
		"BTC-USDT-SWAP",
		"BTC-USDT-ER33234",
	}
	dPair, err := ok.GetPairFromInstrumentID(instruments[0])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = ok.GetPairFromInstrumentID(instruments[1])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = ok.GetPairFromInstrumentID(instruments[2])
	require.NoError(t, err)
	assert.NotNil(t, dPair)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USD")
	require.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetActiveOrders(contextGenerate(), &order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{pair, spotTP, currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
		Side:      order.Buy,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetActiveOrders(contextGenerate(), &order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{pair, spotTP, currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spread,
		Side:      order.Buy,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err := ok.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	getOrdersRequest.Pairs = []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)}
	result, err := ok.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.AssetType = asset.Spread
	getOrdersRequest.Type = order.Market
	result, err = ok.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFeeByType(contextGenerate(), &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.ValidateAPICredentials(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 100)
	_, err := ok.GetHistoricCandles(contextGenerate(), pair, asset.Spot, kline.Interval(time.Hour*4), startTime, endTime)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	result, err := ok.GetHistoricCandles(contextGenerate(), pair, asset.Spot, kline.OneDay, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	currencyPair := currency.NewPair(currency.BTC, currency.USDT)
	result, err := ok.GetHistoricCandlesExtended(contextGenerate(), currencyPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	var orderbookBase orderbook.Base
	err := json.Unmarshal([]byte(calculateOrderbookChecksumUpdateOrderbookJSON), &orderbookBase)
	require.NoError(t, err)

	err = ok.CalculateUpdateOrderbookChecksum(&orderbookBase, 2832680552)
	assert.NoError(t, err)
}

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	ok := new(Okx) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(ok), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrders.json", ok.WsHandleData)
	close(ok.Websocket.DataHandler)
	require.Len(t, ok.Websocket.DataHandler, 4, "Should see 4 orders")
	for resp := range ok.Websocket.DataHandler {
		switch v := resp.(type) {
		case *order.Detail:
			switch len(ok.Websocket.DataHandler) {
			case 3:
				require.Equal(t, "452197707845865472", v.OrderID, "OrderID")
				require.Equal(t, "HamsterParty14", v.ClientOrderID, "ClientOrderID")
				require.Equal(t, asset.Spot, v.AssetType, "AssetType")
				require.Equal(t, order.Sell, v.Side, "Side")
				require.Equal(t, order.Filled, v.Status, "Status")
				require.Equal(t, order.Limit, v.Type, "Type")
				require.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "Pair")
				require.Equal(t, 31527.1, v.AverageExecutedPrice, "AverageExecutedPrice")
				require.Equal(t, time.UnixMilli(1654084334977), v.Date, "Date")
				require.Equal(t, time.UnixMilli(1654084353263), v.CloseTime, "CloseTime")
				require.Equal(t, 0.001, v.Amount, "Amount")
				require.Equal(t, 0.001, v.ExecutedAmount, "ExecutedAmount")
				require.Equal(t, 0.000, v.RemainingAmount, "RemainingAmount")
				require.Equal(t, 31527.1, v.Price, "Price")
				require.Equal(t, 0.02522168, v.Fee, "Fee")
				require.Equal(t, currency.USDT, v.FeeAsset, "FeeAsset")
			case 2:
				require.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				require.Equal(t, asset.Spot, v.AssetType, "AssetType")
				require.Equal(t, order.Market, v.Type, "Type")
				require.Equal(t, order.Sell, v.Side, "Side")
				require.Equal(t, order.Active, v.Status, "Status")
				require.Equal(t, 0.0, v.Amount, "Amount should be 0 for a market sell")
				require.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
			case 1:
				require.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				require.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				require.Equal(t, 0.00038127046945832905, v.Amount, "Amount")
				require.Equal(t, 0.010000249968, v.Fee, "Fee")
				require.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				require.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				require.Equal(t, order.PartiallyFilled, v.Status, "Status")
			case 0:
				require.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				require.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				require.Equal(t, 0.010000249968, v.Fee, "Fee")
				require.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				require.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				require.Equal(t, 0.00038128, v.Amount, "Amount should be derived because order filled")
				require.Equal(t, order.Filled, v.Status, "Status")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Got unexpected data: %T %v", v, v)
		}
	}
}

var pushDataMap = map[string]string{
	"Algo Orders":                           `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`,
	"Advanced Algo Order":                   `{"arg": {"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`,
	"Position Risk":                         `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Account Greeks":                        `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"Rfqs":                                  `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"Accounts":                              `{"arg": {"channel": "account","ccy": "BTC","uid": "77982378738415879"},	"data": [{"uTime": "1597026383085","totalEq": "41624.32","isoEq": "3624.32","adjEq": "41624.32","ordFroz": "0","imr": "4162.33","mmr": "4","notionalUsd": "","mgnRatio": "41624.32","details": [{"availBal": "","availEq": "1","ccy": "BTC","cashBal": "1","uTime": "1617279471503","disEq": "50559.01","eq": "1","eqUsd": "45078.3790756226851775","frozenBal": "0","interest": "0","isoEq": "0","liab": "0","maxLoan": "","mgnRatio": "","notionalLever": "0.0022195262185864","ordFrozen": "0","upl": "0","uplLiab": "0","crossLiab": "0","isoLiab": "0","coinUsdPrice": "60000","stgyEq":"0","spotInUseAmt":"","isoUpl":""}]}]}`,
	"Quotes":                                `{"arg": {"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`,
	"Structure Block Trades":                `{"arg": {"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`,
	"Spot Grid Algo Orders":                 `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`,
	"Contract Grid Algo Orders":             `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`,
	"Grid Positions":                        `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`,
	"Grid Sub Orders":                       `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`,
	"Instrument":                            `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`,
	"Open Interest":                         `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`,
	"Trade":                                 `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`,
	"Estimated Delivery And Exercise Price": `{"arg": {"args": "estimated-price","instType": "FUTURES","uly": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`,
	"Mark Price":                            `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`,
	"Mark Price Candlestick":                `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`,
	"Price Limit":                           `{"arg": {"channel": "price-limit","instId": "LTC-USD-190628"},"data": [{"instId": "LTC-USD-190628","buyLmt": "200","sellLmt": "300","ts": "1597026383085"}]}`,
	"Test Snapshot Orderbook":               `{"arg": {"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
	"Options Trades":                        `{"arg": {"channel": "option-trades", "instType": "OPTION", "instFamily": "BTC-USD" }, "data": [ { "fillVol": "0.5066007836914062", "fwdPx": "16469.69928595038", "idxPx": "16537.2", "instFamily": "BTC-USD", "instId": "BTC-USD-230224-18000-C", "markPx": "0.04690107010619562", "optType": "C", "px": "0.045", "side": "sell", "sz": "2", "tradeId": "38", "ts": "1672286551080" } ] }`,
	"Public Block Trades":                   `{"arg": {"channel":"public-block-trades", "instId":"BTC-USD-231020-5000-P" }, "data":[ { "fillVol":"5", "fwdPx":"26808.16", "idxPx":"27222.5", "instId":"BTC-USD-231020-5000-P", "markPx":"0.0022406326071111", "px":"0.0048", "side":"buy", "sz":"1", "tradeId":"633971452580106242", "ts":"1697422572972"}]}`,
	"Option Summary":                        `{"arg": {"channel": "opt-summary","uly": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`,
	"Funding Rate":                          `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`,
	"Index Candlestick":                     `{"arg": {"channel": "index-candle30m","instId": "BTC-USDT"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`,
	"Index Ticker":                          `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`,
	"Status":                                `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`,
	"Public Struct Block Trades":            `{"arg": {"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`,
	"Block Ticker":                          `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Account":                               `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Position":                              `{"arg": {"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Position Data With Underlying":         `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Balance And Position":                  `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`,
	"Deposit Info Details":                  `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Withdrawal Info Details":               `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Recurring Buy Order":                   `{"arg": {"channel": "algo-recurring-buy", "instType": "SPOT", "uid": "447*******584" }, "data": [{ "algoClOrdId": "", "algoId": "644497312047435776", "algoOrdType": "recurring", "amt": "100", "cTime": "1699932133373", "cycles": "0", "instType": "SPOT", "investmentAmt": "0", "investmentCcy": "USDC", "mktCap": "0", "nextInvestTime": "1699934415300", "pTime": "1699933314691", "period": "hourly", "pnlRatio": "0", "recurringDay": "", "recurringHour": "1", "recurringList": [{ "avgPx": "0", "ccy": "BTC", "profit": "0", "px": "36482", "ratio": "0.2", "totalAmt": "0" }, { "avgPx": "0", "ccy": "ETH", "profit": "0", "px": "2057.54", "ratio": "0.8", "totalAmt": "0" }], "recurringTime": "12", "state": "running", "stgyName": "stg1", "tag": "", "timeZone": "8", "totalAnnRate": "0", "totalPnl": "0", "uTime": "1699932136249" }] }`,
	"Liquidation Orders":                    `{"arg": {"channel": "liquidation-orders", "instType": "SWAP" }, "data": [ { "details": [ { "bkLoss": "0", "bkPx": "0.007831", "ccy": "", "posSide": "short", "side": "buy", "sz": "13", "ts": "1692266434010" } ], "instFamily": "IOST-USDT", "instId": "IOST-USDT-SWAP", "instType": "SWAP", "uly": "IOST-USDT"}]}`,
	"Economic Calendar":                     `{"arg": {"channel": "economic-calendar" }, "data": [ { "calendarId": "319275", "date": "1597026383085", "region": "United States", "category": "Manufacturing PMI", "event": "S&P Global Manufacturing PMI Final", "refDate": "1597026383085", "actual": "49.2", "previous": "47.3", "forecast": "49.3", "importance": "2", "prevInitial": "", "ccy": "", "unit": "", "ts": "1698648096590" } ] }`,
	"Failure":                               `{ "event": "error", "code": "60012", "msg": "Invalid request: {\"op\": \"subscribe\", \"args\":[{ \"channel\" : \"block-tickers\", \"instId\" : \"LTC-USD-200327\"}]}", "connId": "a4d3ae55" }`,
}

func TestPushData(t *testing.T) {
	t.Parallel()
	var err error
	for x := range pushDataMap {
		err = ok.WsHandleData([]byte(pushDataMap[x]))
		require.NoErrorf(t, err, "Okx %s error %v", x, err)
	}
}

func TestPushDataDynamic(t *testing.T) {
	t.Parallel()
	dataMap := map[string]string{
		"Ticker":             `{"arg": {"channel": "tickers","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`,
		"Candlesticks":       `{"arg": {"channel": "candle1D","instId": "BTC-USD-SWAP"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`,
		"Snapshot OrderBook": `{"arg":{"channel":"books","instId":"BTC-USD-SWAP"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
	}
	var err error
	for x := range dataMap {
		err = ok.WsHandleData([]byte(dataMap[x]))
		require.NoError(t, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	result, err := ok.GetHistoricTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func setupWS() {
	if !ok.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(ok) {
		ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := ok.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

// ************************** Public Channel Subscriptions *****************************

func TestInstrumentsSubscription(t *testing.T) {
	t.Parallel()
	err := ok.InstrumentsSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestTickersSubscription(t *testing.T) {
	t.Parallel()
	err := ok.TickersSubscription(context.Background(), "subscribe", asset.Margin, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	err = ok.TickersSubscription(context.Background(), "unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}
func TestOpenInterestSubscription(t *testing.T) {
	t.Parallel()
	err := ok.OpenInterestSubscription(context.Background(), "subscribe", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}
func TestCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.PerpetualSwap)
	require.NoError(t, err)
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.CandlesticksSubscription(context.Background(), "subscribe", okxChannelCandle1m, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.TradesSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestEstimatedDeliveryExercisePriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	require.NoErrorf(t, err, "%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	err = ok.EstimatedDeliveryExercisePriceSubscription(context.Background(), "subscribe", asset.Futures, futuresPairs[0])
	assert.NoError(t, err)
}

func TestMarkPriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	require.NoErrorf(t, err, "%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	err = ok.MarkPriceSubscription(context.Background(), "subscribe", asset.Futures, futuresPairs[0])
	assert.NoError(t, err)
}

func TestMarkPriceCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.MarkPriceCandlesticksSubscription(context.Background(), "subscribe", okxChannelMarkPriceCandle1Y, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestPriceLimitSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PriceLimitSubscription(context.Background(), "subscribe", asset.PerpetualSwap, currency.NewPairWithDelimiter("BTC", "USD-SWAP", currency.DashDelimiter))
	assert.NoError(t, err)
}

func TestOrderBooksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.OrderBooksSubscription(context.Background(), "subscribe", okxChannelOrderBooks, asset.Futures, enabled[0])
	require.NoError(t, err)
	err = ok.OrderBooksSubscription(context.Background(), "unsubscribe", okxChannelOrderBooks, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestOptionSummarySubscription(t *testing.T) {
	t.Parallel()
	err := ok.OptionSummarySubscription(context.Background(), "subscribe", currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.OptionSummarySubscription(context.Background(), "unsubscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestFundingRateSubscription(t *testing.T) {
	t.Parallel()
	err := ok.FundingRateSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")))
	require.NoError(t, err)
	err = ok.FundingRateSubscription(context.Background(), "unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")))
	assert.NoError(t, err)
}

func TestIndexCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	err := ok.IndexCandlesticksSubscription(context.Background(), "subscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.IndexCandlesticksSubscription(context.Background(), "unsubscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}
func TestIndexTickerChannelIndexTickerChannel(t *testing.T) {
	t.Parallel()
	err := ok.IndexTickerChannel(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.IndexTickerChannel(context.Background(), "unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestStatusSubscription(t *testing.T) {
	t.Parallel()
	err := ok.StatusSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.StatusSubscription(context.Background(), "unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestPublicStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PublicStructureBlockTradesSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.PublicStructureBlockTradesSubscription(context.Background(), "unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}
func TestBlockTickerSubscription(t *testing.T) {
	t.Parallel()
	err := ok.BlockTickerSubscription(context.Background(), "subscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	err = ok.BlockTickerSubscription(context.Background(), "unsubscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestPublicBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PublicBlockTradesSubscription(context.Background(), "subscribe", asset.Options, currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"))
	require.NoError(t, err)
	err = ok.PublicBlockTradesSubscription(context.Background(), "unsubscribe", asset.Options, currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"))
	assert.NoError(t, err)
}

// ************ Authenticated Websocket endpoints Test **********************************************

func TestWsAccountSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.WsAccountSubscription(context.Background(), "subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsPlaceOrder(contextGenerate(), &PlaceOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceOrderRequestParam{
		ReduceOnly: true,
		AssetType:  asset.Margin,
	}
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "Buy"
	arg.TradeMode = "abc"
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = ok.WsPlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsPlaceOrder(context.Background(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         order.Buy.Lower(),
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.WsPlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         order.Buy.Lower(),
		PositionSide: "long",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
		AssetType:    asset.Futures,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	var resp []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	require.NoError(t, err)

	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{})
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	arg := PlaceOrderRequestParam{
		ReduceOnly: true,
	}
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	arg.TradeMode = "abc"
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = ok.WsPlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err = ok.WsPlaceMultipleOrders(context.Background(), resp)
	assert.False(t, (err != nil && !errors.Is(err, errWebsocketStreamNotAuthenticated)), err)
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsCancelOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = ok.WsCancelOrder(context.Background(), &CancelOrderRequestParam{OrderID: "1234"})
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = ok.WsCancelOrder(context.Background(), &CancelOrderRequestParam{InstrumentID: "BTC-USD-190927"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsCancelOrder(context.Background(), &CancelOrderRequestParam{
		InstrumentID: "BTC-USD-190927",
		OrderID:      "2510789768709120",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()
	arg := CancelOrderRequestParam{
		OrderID: "2510789768709120",
	}
	_, err := ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "DCR-BTC"
	arg.OrderID = ""
	_, err = ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsCancelMultipleOrder(context.Background(), []CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsAmendOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &AmendOrderRequestParams{}
	_, err = ok.WsAmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = spotTP.String()
	_, err = ok.WsAmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	_, err = ok.WsAmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsAmendOrder(context.Background(), &AmendOrderRequestParams{
		InstrumentID: spotTP.String(),
		OrderID:      "2510789768709120",
		NewQuantity:  1234,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	arg := AmendOrderRequestParams{
		CancelOnFail: true,
	}
	_, err := ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "DCR-BTC"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "2510789768709120"
	_, err = ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsAmendMultipleOrders(context.Background(), []AmendOrderRequestParams{
		{
			InstrumentID: "DCR-BTC",
			OrderID:      "2510789768709120",
			NewPrice:     1233324.332,
			NewQuantity:  1234,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsMassCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.WsMassCancelOrders(context.Background(), []CancelMassReqParam{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.WsMassCancelOrders(context.Background(), []CancelMassReqParam{{InstrumentFamily: "BTC-USD"}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = ok.WsMassCancelOrders(context.Background(), []CancelMassReqParam{{InstrumentType: "OPTION"}})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsMassCancelOrders(context.Background(), []CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: "BTC-USD",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPositionChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.WsPositionChannel(context.Background(), "subscribe", asset.Options, currency.NewPair(currency.USD, currency.BTC))
	assert.NoError(t, err)
}

func TestBalanceAndPositionSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.BalanceAndPositionSubscription(context.Background(), "subscribe", "1234")
	require.NoError(t, err)
	err = ok.BalanceAndPositionSubscription(context.Background(), "unsubscribe", "1234")
	assert.NoError(t, err)
}

func TestWsOrderChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.WsOrderChannel(context.Background(), "subscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), "")
	require.NoError(t, err)
	err = ok.WsOrderChannel(context.Background(), "unsubscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), "")
	assert.NoError(t, err)
}

func TestAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.AlgoOrdersSubscription(context.Background(), "subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	require.NoError(t, err)
	err = ok.AlgoOrdersSubscription(context.Background(), "unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}

func TestAdvanceAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.AdvanceAlgoOrdersSubscription(context.Background(), "subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), "")
	require.NoError(t, err)
	err = ok.AdvanceAlgoOrdersSubscription(context.Background(), "unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), "")
	assert.NoError(t, err)
}

func TestPositionRiskWarningSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.PositionRiskWarningSubscription(context.Background(), "subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	require.NoError(t, err)
	err = ok.PositionRiskWarningSubscription(context.Background(), "unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}

func TestAccountGreeksSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.AccountGreeksSubscription(context.Background(), "subscribe", currency.NewPair(currency.SOL, currency.USD))
	require.NoError(t, err)
	err = ok.AccountGreeksSubscription(context.Background(), "unsubscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestRfqSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.RfqSubscription(context.Background(), "subscribe", "")
	require.NoError(t, err)
	err = ok.RfqSubscription(context.Background(), "unsubscribe", "")
	assert.NoError(t, err)
}

func TestQuotesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.QuotesSubscription(context.Background(), "subscribe")
	require.NoError(t, err)
	err = ok.QuotesSubscription(context.Background(), "unsubscribe")
	assert.NoError(t, err)
}

func TestStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.StructureBlockTradesSubscription(context.Background(), "subscribe")
	require.NoError(t, err)
	err = ok.StructureBlockTradesSubscription(context.Background(), "unsubscribe")
	assert.NoError(t, err)
}

func TestSpotGridAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.SpotGridAlgoOrdersSubscription(context.Background(), "subscribe", asset.Empty, currency.EMPTYPAIR, "")
	require.NoError(t, err)
	err = ok.SpotGridAlgoOrdersSubscription(context.Background(), "unsubscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
}

func TestContractGridAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.ContractGridAlgoOrders(context.Background(), "subscribe", asset.Empty, currency.EMPTYPAIR, "")
	require.NoError(t, err)
	err = ok.ContractGridAlgoOrders(context.Background(), "unsubscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
}

func TestGridPositionsSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.GridPositionsSubscription(context.Background(), "subscribe", "1234")
	require.NoError(t, err)
	err = ok.GridPositionsSubscription(context.Background(), "unsubscribe", "1234")
	assert.NoError(t, err)
}

func TestGridSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.GridSubOrders(context.Background(), "subscribe", "")
	require.NoError(t, err)
	err = ok.GridSubOrders(context.Background(), "unsubscribe", "")
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := ok.GetServerTime(contextGenerate(), asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAvailableTransferChains(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIntervalEnum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Description string
		Interval    kline.Interval
		Expected    string
		AppendUTC   bool
	}{
		{Description: "4hr with UTC", Interval: kline.FourHour, Expected: "4H", AppendUTC: true},
		{Description: "6H without UTC", Interval: kline.SixHour, Expected: "6H"},
		{Description: "6H with UTC", Interval: kline.SixHour, Expected: "6Hutc", AppendUTC: true},
		{Description: "Unsupported interval with UTC", Expected: "", AppendUTC: true},
	}

	for _, tt := range tests {
		t.Run(tt.Description, func(t *testing.T) {
			t.Parallel()
			r := ok.GetIntervalEnum(tt.Interval, tt.AppendUTC)
			require.Equalf(t, tt.Expected, r, "%s: received: %s but expected: %s", tt.Description, r, tt.Expected)
		})
	}
}

func TestInstrument(t *testing.T) {
	t.Parallel()
	var i Instrument
	err := json.Unmarshal([]byte(instrumentJSON), &i)
	require.NoError(t, err)

	require.Empty(t, i.Alias, "expected empty alias")
	require.Empty(t, i.BaseCurrency, "expected empty base currency")
	require.Equal(t, "1", i.Category, "expected 1 category")
	require.Equal(t, 1, int(i.ContractMultiplier.Int64()), "expected 1 contract multiplier")
	require.Equal(t, "linear", i.ContractType, "expected linear contract type")
	require.Equal(t, 0.0001, i.ContractValue.Float64(), "expected 0.0001 contract value")
	require.Equal(t, currency.BTC.String(), i.ContractValueCurrency, "expected BTC contract value currency")
	require.True(t, i.ExpTime.Time().IsZero(), "expected empty expiry time")
	require.Equal(t, "BTC-USDC", i.InstrumentFamily, "expected BTC-USDC instrument family")
	require.Equal(t, "BTC-USDC-SWAP", i.InstrumentID, "expected BTC-USDC-SWAP instrument ID")
	swap := ok.GetInstrumentTypeFromAssetItem(asset.PerpetualSwap)
	require.Equal(t, swap, i.InstrumentType, "expected SWAP instrument type")
	require.Equal(t, 125, int(i.MaxLeverage), "expected 125 leverage")
	require.Equal(t, int64(1666076190000), i.ListTime.Time().UnixMilli(), "expected 1666076190000 listing time")
	require.Equal(t, 1, int(i.LotSize))
	require.Equal(t, 100000000.0000000000000000, i.MaxSpotIcebergSize.Float64())
	require.Equal(t, 100000000, int(i.MaxQuantityOfSpotLimitOrder))
	require.Equal(t, 85000, int(i.MaxQuantityOfMarketLimitOrder))
	require.Equal(t, 85000, int(i.MaxStopSize))
	require.Equal(t, 100000000.0000000000000000, i.MaxTriggerSize.Float64())
	require.Equal(t, 0, int(i.MaxQuantityOfSpotTwapLimitOrder), "expected empty max TWAP size")
	require.Equal(t, 1, int(i.MinimumOrderSize))
	require.Empty(t, i.OptionType, "expected empty option type")
	require.Empty(t, i.QuoteCurrency, "expected empty quote currency")
	require.Equal(t, currency.USDC.String(), i.SettlementCurrency, "expected USDC settlement currency")
	require.Equal(t, "live", i.State)
	require.Empty(t, i.StrikePrice, "expected empty strike price")
	require.Equal(t, 0.1, i.TickSize.Float64())
	assert.Equal(t, "BTC-USDC", i.Underlying, "expected BTC-USDC underlying")
}

func TestGetLatestFundingRate(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	require.NoError(t, err)
	result, err := ok.GetLatestFundingRates(contextGenerate(), &fundingrate.LatestRateRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	require.NoError(t, err)
	r := &fundingrate.HistoricalRatesRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 cp,
		PaymentCurrency:      currency.USDT,
		StartDate:            time.Now().Add(-time.Hour * 24 * 2),
		EndDate:              time.Now(),
		IncludePredictedRate: true,
	}

	r.StartDate = time.Now().Add(-time.Hour * 24 * 120)
	_, err = ok.GetHistoricalFundingRates(contextGenerate(), r)
	require.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits)

	if sharedtestvalues.AreAPICredentialsSet(ok) {
		r.IncludePayments = true
	}
	r.StartDate = time.Now().Add(-time.Hour * 24 * 12)
	result, err := ok.GetHistoricalFundingRates(contextGenerate(), r)
	require.NoError(t, err)
	require.NotNil(t, result)

	r.RespectHistoryLimits = true
	result, err = ok.GetHistoricalFundingRates(contextGenerate(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := ok.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	require.NoError(t, err)
	require.False(t, is, "expected false")

	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	require.NoError(t, err)
	is, err = ok.IsPerpetualFutureCurrency(asset.PerpetualSwap, cp)
	require.NoError(t, err)
	assert.True(t, is, "expected true")
}

func TestGetAssetsFromInstrumentTypeOrID(t *testing.T) {
	t.Parallel()

	ok := new(Okx) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(ok), "Setup must not error")

	_, err := ok.GetAssetsFromInstrumentTypeOrID("", "")
	assert.ErrorIs(t, err, errMissingInstrumentID)

	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options} {
		symbol := ""
		if a != asset.Spot {
			symbol = ok.CurrencyPairs.Pairs[a].Enabled[0].String()
		}
		assets, err2 := ok.GetAssetsFromInstrumentTypeOrID(a.String(), symbol)
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		require.Len(t, assets, 1)
		assert.Equalf(t, a, assets[0], "Should contain asset: %s", a)
	}

	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test")
	assert.ErrorIs(t, err, currency.ErrCurrencyNotSupported)
	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test-test")
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	for _, a := range []asset.Item{asset.Margin, asset.Spot} {
		assets, err2 := ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[a].Enabled[0].String())
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		assert.Contains(t, assets, a)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := ok.SetMarginType(contextGenerate(), asset.Spot, currency.NewBTCUSDT(), margin.Isolated)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	cp, err := currency.NewPairFromString("eth/btc")
	require.NoError(t, err)
	result, err := ok.ChangePositionMargin(contextGenerate(), &margin.PositionChangeRequest{
		Pair:                    cp,
		Asset:                   asset.Margin,
		MarginType:              margin.Isolated,
		OriginalAllocatedMargin: 4.0695,
		NewAllocatedMargin:      5,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCollateralMode(contextGenerate(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCollateralMode(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	_, err = ok.GetCollateralMode(contextGenerate(), asset.Futures)
	assert.True(t, errors.Is(err, nil) || errors.Is(err, asset.ErrNotSupported))
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := ok.SetCollateralMode(contextGenerate(), asset.Spot, collateral.SingleMode)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	pp, err := ok.CurrencyPairs.GetPairs(asset.PerpetualSwap, true)
	require.NoError(t, err)
	result, err := ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.PerpetualSwap,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	pp, err = ok.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           pp[0],
		UnderlyingPair: currency.NewBTCUSDT(),
	})
	require.ErrorIsf(t, err, futures.ErrNotFuturesAsset, "received '%v', expected '%v'", err, futures.ErrNotFuturesAsset)

	result, err = ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Futures,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = ok.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Spot,
		Pairs:     []currency.Pair{pp[0]},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Futures,
		Pairs:     []currency.Pair{pp[0]},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
		EndDate:   time.Now(),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.UnknownSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, order.UnknownSide)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Long)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Short)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.UnknownSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.CouldNotBuy)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = ok.SetLeverage(contextGenerate(), asset.Spot, pp[0], margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Long)
	require.NoError(t, err)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Short)
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFuturesContractDetails(context.Background(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = ok.GetFuturesContractDetails(context.Background(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := ok.GetFuturesContractDetails(context.Background(), asset.Futures)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = ok.GetFuturesContractDetails(context.Background(), asset.PerpetualSwap)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsProcessOrderbook5(t *testing.T) {
	t.Parallel()
	var ob5payload = []byte(`{"arg":{"channel":"books5","instId":"OKB-USDT"},"data":[{"asks":[["0.0000007465","2290075956","0","4"],["0.0000007466","1747284705","0","4"],["0.0000007467","1338861655","0","3"],["0.0000007468","1661668387","0","6"],["0.0000007469","2715477116","0","5"]],"bids":[["0.0000007464","15693119","0","1"],["0.0000007463","2330835024","0","4"],["0.0000007462","1182926517","0","2"],["0.0000007461","3818684357","0","4"],["0.000000746","6021641435","0","7"]],"instId":"OKB-USDT","ts":"1695864901807","seqId":4826378794}]}`)
	err := ok.wsProcessOrderbook5(ob5payload)
	require.NoError(t, err)

	required := currency.NewPairWithDelimiter("OKB", "USDT", "-")
	got, err := orderbook.Get("okx", required, asset.Spot)
	require.NoError(t, err)

	require.Len(t, got.Asks, 5)
	require.Len(t, got.Bids, 5)
	// Book replicated to margin
	got, err = orderbook.Get("okx", required, asset.Margin)
	require.NoError(t, err)
	require.Len(t, got.Asks, 5)
	assert.Len(t, got.Bids, 5)
}

func TestGetLeverateEstimatedInfo(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeverageEstimatedInfo(context.Background(), "", "cross", "1", "", "BTC-USDT", currency.BTC)
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = ok.GetLeverageEstimatedInfo(context.Background(), "MARGIN", "", "1", "", "BTC-USDT", currency.BTC)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = ok.GetLeverageEstimatedInfo(context.Background(), "MARGIN", "cross", "", "", "BTC-USDT", currency.BTC)
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = ok.GetLeverageEstimatedInfo(context.Background(), "MARGIN", "cross", "1", "", "BTC-USDT", currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLeverageEstimatedInfo(context.Background(), "MARGIN", "cross", "1", "", "BTC-USDT", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualBorrowAndRepayInQuickMarginMode(t *testing.T) {
	t.Parallel()
	_, err := ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		InstrumentID: "BTC-USDT",
		LoanCcy:      currency.USDT,
		Side:         "borrow",
	})
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: "BTC-USDT",
		Side:         "borrow",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: "BTC-USDT",
		LoanCcy:      currency.USDT,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		Amount:  1,
		LoanCcy: currency.USDT,
		Side:    "borrow",
	})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: "BTC-USDT",
		LoanCcy:      currency.USDT,
		Side:         "borrow",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowAndRepayHistoryInQuickMarginMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetBorrowAndRepayHistoryInQuickMarginMode(context.Background(), currency.EMPTYPAIR, currency.BTC, "borrow", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetVIPInterestAccruedData(context.Background(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestGetVIPInterestDeductedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetVIPInterestDeductedData(context.Background(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetVIPLoanOrderList(context.Background(), "", "1", currency.BTC, time.Time{}, time.Now(), 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetVIPLoanOrderDetail(context.Background(), "123456", currency.BTC, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
func TestSetRiskOffsetType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetRiskOffsetType(context.Background(), "3")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestActivateOption(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.ActivateOption(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAutoLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.SetAutoLoan(context.Background(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAccountMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetAccountMode(context.Background(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	_, err := ok.ResetMMPStatus(contextGenerate(), okxInstTypeOption, "")
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ResetMMPStatus(contextGenerate(), okxInstTypeOption, "BTC-USD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	_, err := ok.SetMMP(context.Background(), &MMPConfig{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.SetMMP(context.Background(), &MMPConfig{
		TimeInterval: 5000,
	})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = ok.SetMMP(context.Background(), &MMPConfig{
		InstrumentFamily: "BTC-USD",
	})
	require.ErrorIs(t, err, errInvalidQuantityLimit)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetMMP(context.Background(), &MMPConfig{
		InstrumentFamily: "BTC-USD",
		TimeInterval:     5000,
		FrozenInterval:   2000,
		QuantityLimit:    100,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMMPConfig(context.Background(), "BTC-USD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMassCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.MassCancelOrder(context.Background(), "", "BTC-USD", 2000)
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = ok.MassCancelOrder(context.Background(), "OPTION", "", 2000)
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = ok.MassCancelOrder(context.Background(), "OPTION", "BTC-USD", -1)
	require.ErrorIs(t, err, errMissingIntervalValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.MassCancelOrder(context.Background(), "OPTION", "BTC-USD", 2000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllMMPOrdersAfterCountdown(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelAllMMPOrdersAfterCountdown(context.Background(), 2, "")
	require.ErrorIs(t, err, errCountdownTimeoutRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAllMMPOrdersAfterCountdown(context.Background(), 60, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeAccountRateLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetTradeAccountRateLimit(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderPreCheck(t *testing.T) {
	t.Parallel()
	_, err := ok.OrderPreCheck(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderPreCheckParams{
		ClientOrderID: "b15",
	}
	_, err = ok.OrderPreCheck(context.Background(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = "BTC-USDT"
	_, err = ok.OrderPreCheck(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cash"
	_, err = ok.OrderPreCheck(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = ok.OrderPreCheck(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = ok.OrderPreCheck(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.OrderPreCheck(context.Background(), &OrderPreCheckParams{
		InstrumentID:  "BTC-USDT",
		TradeMode:     "cash",
		ClientOrderID: "b15",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendAlgoOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.AmendAlgoOrder(context.Background(), &AmendAlgoOrderParam{NewSize: 2})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.AmendAlgoOrder(context.Background(), &AmendAlgoOrderParam{
		InstrumentID: perpetualSwapTP.String(),
		NewSize:      2,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendAlgoOrder(context.Background(), &AmendAlgoOrderParam{
		AlgoID:       "2510789768709120",
		InstrumentID: perpetualSwapTP.String(),
		NewSize:      2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAlgoOrderDetail(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAlgoOrderDetail(context.Background(), "1234231231423", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClosePositionForContractID(t *testing.T) {
	t.Parallel()
	_, err := ok.ClosePositionForContractID(context.Background(), &ClosePositionParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.ClosePositionForContractID(context.Background(), &ClosePositionParams{AlgoID: "", MarketCloseAllPositions: true})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = ok.ClosePositionForContractID(context.Background(), &ClosePositionParams{AlgoID: "448965992920907776", MarketCloseAllPositions: false})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	_, err = ok.ClosePositionForContractID(context.Background(), &ClosePositionParams{AlgoID: "448965992920907776", MarketCloseAllPositions: false, Size: 123})
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.ClosePositionForContractID(context.Background(), &ClosePositionParams{
		AlgoID:                  "448965992920907776",
		MarketCloseAllPositions: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelClosePositionOrderForContractGrid(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelClosePositionOrderForContractGrid(context.Background(), &CancelClosePositionOrder{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CancelClosePositionOrderForContractGrid(context.Background(), &CancelClosePositionOrder{OrderID: "570627699870375936"})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = ok.CancelClosePositionOrderForContractGrid(context.Background(), &CancelClosePositionOrder{AlgoID: "448965992920907776"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelClosePositionOrderForContractGrid(context.Background(), &CancelClosePositionOrder{
		AlgoID:  "448965992920907776",
		OrderID: "570627699870375936",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInstantTriggerGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.InstantTriggerGridAlgoOrder(context.Background(), "123456789")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestComputeMinInvestment(t *testing.T) {
	t.Parallel()
	arg := &ComputeInvestmentDataParam{
		RunType: "1",
	}
	_, err := ok.ComputeMinInvestment(context.Background(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)
	arg.InstrumentID = "ETH-USDT"
	_, err = ok.ComputeMinInvestment(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	arg.AlgoOrderType = "grid"
	_, err = ok.ComputeMinInvestment(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.MaxPrice = 5000
	_, err = ok.ComputeMinInvestment(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)
	arg.MinPrice = 5000
	_, err = ok.ComputeMinInvestment(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidGridQuantity)

	result, err := ok.ComputeMinInvestment(context.Background(), &ComputeInvestmentDataParam{
		InstrumentID:  "ETH-USDT",
		AlgoOrderType: "grid",
		GridNumber:    50,
		MaxPrice:      5000,
		MinPrice:      3000,
		RunType:       "1",
		InvestmentData: []InvestmentData{
			{
				Amount:   0.01,
				Currency: currency.ETH,
			},
			{
				Amount:   100,
				Currency: currency.USDT,
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRSIBackTesting(t *testing.T) {
	t.Parallel()
	_, err := ok.RSIBackTesting(context.Background(), "", "", "", 50, 14, kline.FiveMin)
	require.ErrorIs(t, err, errMissingInstrumentID)
	result, err := ok.RSIBackTesting(context.Background(), "BTC-USDT", "", "", 50, 14, kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSignalBotTrading(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSignalBotOrderDetail(context.Background(), "", "623833708424069120")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = ok.GetSignalBotOrderDetail(context.Background(), "contract", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.GetSignalBotOrderDetail(context.Background(), "contract", "623833708424069120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalOrderPositions(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSignalOrderPositions(context.Background(), "", "623833708424069120")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = ok.GetSignalOrderPositions(context.Background(), "contract", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSignalOrderPositions(context.Background(), "contract", "623833708424069120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalBotSubOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSignalBotSubOrders(context.Background(), "", "contract", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = ok.GetSignalBotSubOrders(context.Background(), "623833708424069120", "", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = ok.GetSignalBotSubOrders(context.Background(), "623833708424069120", "contract", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSignalBotSubOrders(context.Background(), "623833708424069120", "contract", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalBotEventHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSignalBotEventHistory(context.Background(), "", time.Time{}, time.Now(), 50)
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSignalBotEventHistory(context.Background(), "12345", time.Time{}, time.Now(), 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceRecurringBuyOrder(context.Background(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceRecurringBuyOrderParam{
		TimeZone: "3",
	}
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, errStrategyNameRequired)

	arg.StrategyName = "BTC|ETH recurring buy monthly"
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.RecurringList = []RecurringListItem{{}}
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.RecurringList = []RecurringListItem{{Currency: currency.BTC}}
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, errRecurringDayRequired)

	arg.RecurringDay = "1"
	arg.RecurringTime = -10
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, errRecurringBuyTimeRequired)

	arg.RecurringTime = 2
	_, err = ok.PlaceRecurringBuyOrder(context.Background(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceRecurringBuyOrder(context.Background(), &PlaceRecurringBuyOrderParam{
		StrategyName: "BTC|ETH recurring buy monthly",
		Amount:       100,
		RecurringList: []RecurringListItem{
			{
				Currency: currency.BTC,
				Ratio:    0.2,
			},
			{
				Currency: currency.ETH,
				Ratio:    0.8,
			},
		},
		Period:             "monthly",
		RecurringDay:       "1",
		RecurringTime:      0,
		TimeZone:           "8", // UTC +8
		TradeMode:          "cross",
		InvestmentCurrency: "USDT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendRecurringBuyOrder(context.Background(), &AmendRecurringOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.AmendRecurringBuyOrder(context.Background(), &AmendRecurringOrderParam{StrategyName: "stg1"})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = ok.AmendRecurringBuyOrder(context.Background(), &AmendRecurringOrderParam{AlgoID: "448965992920907776"})
	require.ErrorIs(t, err, errStrategyNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendRecurringBuyOrder(context.Background(), &AmendRecurringOrderParam{
		AlgoID:       "448965992920907776",
		StrategyName: "stg1",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStopRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.StopRecurringBuyOrder(context.Background(), []StopRecurringBuyOrder{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.StopRecurringBuyOrder(context.Background(), []StopRecurringBuyOrder{{}})
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.StopRecurringBuyOrder(context.Background(), []StopRecurringBuyOrder{{AlgoID: "1232323434234"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringBuyOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRecurringBuyOrderList(context.Background(), "", "paused", time.Time{}, time.Time{}, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringBuyOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRecurringBuyOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.GetRecurringOrderDetails(context.Background(), "", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRecurringOrderDetails(context.Background(), "560473220642766848", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringSubOrders(t *testing.T) {
	t.Parallel()
	_, err := ok.GetRecurringSubOrders(context.Background(), "", "123422", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetRecurringSubOrders(context.Background(), "560473220642766848", "123422", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExistingLeadingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetExistingLeadingPositions(context.Background(), okxInstTypeSpot, "BTC-USDT", time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadingPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLeadingPositionsHistory(context.Background(), "OPTION", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceLeadingStopOrder(t *testing.T) {
	t.Parallel()
	arg := &TPSLOrderParam{}
	_, err := ok.PlaceLeadingStopOrder(context.Background(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.Tag = "1235454"
	_, err = ok.PlaceLeadingStopOrder(context.Background(), arg)
	require.ErrorIs(t, err, errSubPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceLeadingStopOrder(context.Background(), &TPSLOrderParam{
		SubPositionID:          "1235454",
		TakeProfitTriggerPrice: 123455,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloseLeadingPosition(t *testing.T) {
	t.Parallel()
	_, err := ok.CloseLeadingPosition(context.Background(), &CloseLeadingPositionParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.CloseLeadingPosition(context.Background(), &CloseLeadingPositionParam{Tag: "tag-here"})
	require.ErrorIs(t, err, errSubPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CloseLeadingPosition(context.Background(), &CloseLeadingPositionParam{
		SubPositionID: "518541406042591232",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadingInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLeadingInstrument(context.Background(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendLeadingInstruments(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendLeadingInstruments(context.Background(), "", "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendLeadingInstruments(context.Background(), "BTC-USDT-SWAP", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetProfitSharingDetails(context.Background(), "", time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTotalProfitSharing(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetTotalProfitSharing(context.Background(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnrealizedProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetUnrealizedProfitSharingDetails(context.Background(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFirstCopySettings(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendCopySettings(context.Background(), &FirstCopySettings{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendCopySettings(context.Background(), &FirstCopySettings{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMarginMode:       "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             "ratio_copy",
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPosCloseType:      "copy_close",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendCopySettings(t *testing.T) {
	t.Parallel()
	_, err := ok.SetFirstCopySettings(context.Background(), &FirstCopySettings{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &FirstCopySettings{
		CopyMode: "ratio_copy",
	}
	_, err = ok.SetFirstCopySettings(context.Background(), arg)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	arg.UniqueCode = "25CD5A80241D6FE6"
	_, err = ok.SetFirstCopySettings(context.Background(), arg)
	require.ErrorIs(t, err, errCopyInstrumentIDTypeRequired)

	arg.CopyInstrumentIDType = "copy"
	_, err = ok.SetFirstCopySettings(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.CopyTotalAmount = 500
	_, err = ok.SetFirstCopySettings(context.Background(), arg)
	require.ErrorIs(t, err, errSubPositionCloseTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.SetFirstCopySettings(context.Background(), &FirstCopySettings{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMarginMode:       "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             "ratio_copy",
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPosCloseType:      "copy_close",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStopCopying(t *testing.T) {
	t.Parallel()
	_, err := ok.StopCopying(context.Background(), &StopCopyingParameter{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = ok.StopCopying(context.Background(), &StopCopyingParameter{
		InstrumentType:       "SWAP",
		SubPositionCloseType: "manual_close",
	})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = ok.StopCopying(context.Background(), &StopCopyingParameter{InstrumentType: "SWAP",
		UniqueCode: "25CD5A80241D6FE6"})
	require.ErrorIs(t, err, errSubPositionCloseTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.StopCopying(context.Background(), &StopCopyingParameter{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		SubPositionCloseType: "manual_close",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCopySettings(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCopySettings(context.Background(), "SWAP", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCopySettings(context.Background(), "SWAP", "213E8C92DC61EFAC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultipleLeverages(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMultipleLeverages(context.Background(), "", "213E8C92DC61EFAC", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = ok.GetMultipleLeverages(context.Background(), "isolated", "", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMultipleLeverages(context.Background(), "isolated", "213E8C92DC61EFAC", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMultipleLeverages(t *testing.T) {
	t.Parallel()
	_, err := ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{Leverage: 5})
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{MarginMode: "cross"})
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{
		MarginMode: "cross",
		Leverage:   5,
	})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{
		MarginMode:   "cross",
		Leverage:     5,
		InstrumentID: "BTC-USDT-SWAP",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMyLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetMyLeadTraders(context.Background(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoryLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetHistoryLeadTraders(context.Background(), "", "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWeeklyTraderProfitAndLoss(t *testing.T) {
	t.Parallel()
	_, err := ok.GetWeeklyTraderProfitAndLoss(context.Background(), "", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	mainResult, err := ok.GetWeeklyTraderProfitAndLoss(context.Background(), "", leadTraderUniqueID)
	require.NoError(t, err)
	assert.NotNil(t, mainResult)
}

func TestGetDailyLeadTraderPNL(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDailyLeadTraderPNL(context.Background(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = ok.GetDailyLeadTraderPNL(context.Background(), "SWAP", leadTraderUniqueID, "")
	require.ErrorIs(t, err, errLastDaysRequired)

	mainResult, err := ok.GetDailyLeadTraderPNL(context.Background(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, mainResult)
}

func TestGetLeadTraderStats(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderStats(context.Background(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = ok.GetLeadTraderStats(context.Background(), "SWAP", leadTraderUniqueID, "")
	require.ErrorIs(t, err, errLastDaysRequired)

	result, err := ok.GetLeadTraderStats(context.Background(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadTraderCurrencyPreferences(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderCurrencyPreferences(context.Background(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = ok.GetLeadTraderCurrencyPreferences(context.Background(), "SWAP", leadTraderUniqueID, "")
	require.ErrorIs(t, err, errLastDaysRequired)

	result, err := ok.GetLeadTraderCurrencyPreferences(context.Background(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadTraderCurrentLeadPositions(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderCurrentLeadPositions(context.Background(), okxInstTypeSwap, "", "", "", 10)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	result, err := ok.GetLeadTraderCurrentLeadPositions(context.Background(), "SWAP", leadTraderUniqueID, "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadTraderLeadPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderLeadPositionHistory(context.Background(), "SWAP", "", "", "", 10)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	result, err := ok.GetLeadTraderLeadPositionHistory(context.Background(), "SWAP", leadTraderUniqueID, "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceSpreadOrder(context.Background(), &SpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SpreadOrderParam{Tag: "tag-here"}
	_, err = ok.PlaceSpreadOrder(context.Background(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.SpreadID = spreadTP.String()
	_, err = ok.PlaceSpreadOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = ok.PlaceSpreadOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Size = 1
	_, err = ok.PlaceSpreadOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrPriceBelowMin)

	arg.Price = 12345
	_, err = ok.PlaceSpreadOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceSpreadOrder(context.Background(), &SpreadOrderParam{
		InstrumentID:  spreadTP.String(),
		SpreadID:      "1234",
		ClientOrderID: "12354123523",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Size:          1,
		Price:         12345,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelSpreadOrder(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelSpreadOrder(context.Background(), "12345", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsCancelSpreadOrder(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsCancelSpreadOrder(context.Background(), "1234", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAllSpreadOrders(context.Background(), "123456")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsCancelAllSpreadOrders(context.Background(), "BTC-USDT_BTC-USDT-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendSpreadOrder(context.Background(), &AmendSpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.AmendSpreadOrder(context.Background(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.AmendSpreadOrder(context.Background(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendSpreadOrder(context.Background(), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsAmandSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsAmandSpreadOrder(context.Background(), &AmendSpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = ok.WsAmandSpreadOrder(context.Background(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.WsAmandSpreadOrder(context.Background(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsAmandSpreadOrder(context.Background(), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSpreadOrderDetails(context.Background(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSpreadOrderDetails(context.Background(), "1234567", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetActiveSpreadOrders(context.Background(), "", "post_only", "partially_filled", "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCompletedSpreadOrdersLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetCompletedSpreadOrdersLast7Days(context.Background(), "", "limit", "canceled", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadTradesOfLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetSpreadTradesOfLast7Days(context.Background(), "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreads(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPublicSpreads(context.Background(), "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadOrderBooks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreadOrderBooks(context.Background(), "", 0)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetPublicSpreadOrderBooks(context.Background(), "BTC-USDT_BTC-USDT-SWAP", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreadTickers(context.Background(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetPublicSpreadTickers(context.Background(), "BTC-USDT_BTC-USDT-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPublicSpreadTrades(context.Background(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsTickBands(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionsTickBands(context.Background(), "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := ok.GetOptionsTickBands(context.Background(), "OPTION", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExtractIndexCandlestick(t *testing.T) {
	t.Parallel()
	data := `[ [ "1597026383085", "3.721", "3.743", "3.677", "3.708", "1" ], [ "1597026383085", "3.731", "3.799", "3.494", "3.72", "1" ]]`
	var resp []CandlestickHistoryItem
	err := json.Unmarshal([]byte(data), &resp)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	require.Equal(t, 3.743, resp[0].HighestPrice.Float64())
	require.Equal(t, StateCompleted, resp[0].Confirm)
}

func TestGetHistoricIndexAndMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetHistoricIndexCandlesticksHistory(context.Background(), "", time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetHistoricIndexCandlesticksHistory(context.Background(), "BTC-USD", time.Time{}, time.Time{}, kline.FiveMin, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetMarkPriceCandlestickHistory(context.Background(), "BTC-USD-SWAP", time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEconomicCanendarData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetEconomicCalendarData(context.Background(), "", "", time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositWithdrawalStatus(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDepositWithdrawalStatus(context.Background(), currency.EMPTYCODE, "", "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = ok.GetDepositWithdrawalStatus(context.Background(), currency.EMPTYCODE, "", "1244", "", "")
	require.ErrorIs(t, err, errMissingValidWithdrawalID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetDepositWithdrawalStatus(context.Background(), currency.EMPTYCODE, "1244", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicExchangeList(t *testing.T) {
	t.Parallel()
	result, err := ok.GetPublicExchangeList(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWsPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.WsPlaceSpreadOrder(context.Background(), &SpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.WsPlaceSpreadOrder(context.Background(), &SpreadOrderParam{
		SpreadID:      "BTC-USDT_BTC-USDT-SWAP",
		ClientOrderID: "b15",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInviteesDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInviteesDetail(context.Background(), "")
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetInviteesDetail(context.Background(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAffilateRebateInformation(t *testing.T) {
	t.Parallel()
	_, err := ok.GetUserAffiliateRebateInformation(context.Background(), "")
	require.ErrorIs(t, err, errInvalidAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetUserAffiliateRebateInformation(context.Background(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	usdSwapCode := currency.NewCode("USD-SWAP")
	resp, err := ok.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  perpetualSwapTP.Base.Item,
		Quote: perpetualSwapTP.Quote.Item,
		Asset: asset.PerpetualSwap,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp1 := currency.NewPair(currency.DOGE, usdSwapCode)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ok, asset.PerpetualSwap, cp1)
	resp, err = ok.GetOpenInterest(context.Background(),
		key.PairAsset{
			Base:  currency.BTC.Item,
			Quote: usdSwapCode.Item,
			Asset: asset.PerpetualSwap,
		},
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.PerpetualSwap,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = ok.GetOpenInterest(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, ok)
	for _, a := range ok.GetAssetTypes(false) {
		pairs, err := ok.CurrencyPairs.GetPairs(a, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, pairs)

		resp, err := ok.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestPlaceLendingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.PlaceLendingOrder(context.Background(), &LendingOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &LendingOrderParam{AutoRenewal: true}
	_, err = ok.PlaceLendingOrder(context.Background(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.USDT
	_, err = ok.PlaceLendingOrder(context.Background(), arg)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = ok.PlaceLendingOrder(context.Background(), arg)
	require.ErrorIs(t, err, errLendingRateRequired)

	arg.Rate = 0.01
	_, err = ok.PlaceLendingOrder(context.Background(), arg)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.PlaceLendingOrder(context.Background(), &LendingOrderParam{
		Currency:    currency.USDT,
		Amount:      1,
		Rate:        0.01,
		Term:        "30D",
		AutoRenewal: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendLendingOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.AmendLendingOrder(context.Background(), "", 0, 0, false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.AmendLendingOrder(context.Background(), "12312312", 1., 2., true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLendingOrders(context.Background(), "", "pending", currency.ETH, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingSubOrderList(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLendingSubOrderList(context.Background(), "", "pending", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetLendingSubOrderList(context.Background(), "12345", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSpreadOrdersAfterCountdown(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelAllSpreadOrdersAfterCountdown(context.Background(), 2)
	require.ErrorIs(t, err, errCountdownTimeoutRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CancelAllSpreadOrdersAfterCountdown(context.Background(), 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsOpenInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFuturesContractsOpenInterestHistory(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetFuturesContractsOpenInterestHistory(context.Background(), futuresTP.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFuturesContractTakerVolume(context.Background(), "", kline.FiveMin, 1, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetFuturesContractTakerVolume(context.Background(), futuresTP.String(), kline.FiveMin, 1, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractLongShortAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFuturesContractLongShortAccountRatio(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetFuturesContractLongShortAccountRatio(context.Background(), futuresTP.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTopTradersFuturesContractLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTopTradersFuturesContractLongShortAccountRatio(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetTopTradersFuturesContractLongShortAccountRatio(context.Background(), futuresTP.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTopTradersFuturesContractLongShortPositionRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTopTradersFuturesContractLongShortPositionRatio(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetTopTradersFuturesContractLongShortPositionRatio(context.Background(), futuresTP.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInstruments(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAccountInstruments(context.Background(), asset.Empty, "", "", spotTP.String())
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = ok.GetAccountInstruments(context.Background(), asset.Futures, "", "", spotTP.String())
	require.ErrorIs(t, err, errInvalidUnderlying)
	_, err = ok.GetAccountInstruments(context.Background(), asset.Options, "", "", spotTP.String())
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetAccountInstruments(context.Background(), asset.Spot, "", "", spotTP.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetAccountInstruments(context.Background(), asset.Options, "", "BTC-USD", optionsTP.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = ok.GetAccountInstruments(context.Background(), asset.Futures, "BTC-USD", "", futuresTP.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	var orderTypesToStringMap = map[order.Type]struct {
		Expected string
		Error    error
	}{
		order.Market:                           {Expected: OkxOrderMarket},
		order.Limit:                            {Expected: OkxOrderLimit},
		order.PostOnly:                         {Expected: OkxOrderPostOnly},
		order.FillOrKill:                       {Expected: OkxOrderFOK},
		order.ImmediateOrCancel:                {Expected: OkxOrderIOC},
		order.OptimalLimitIOC:                  {Expected: OkxOrderOptimalLimitIOC},
		order.MarketMakerProtection:            {Expected: "mmp"},
		order.MarketMakerProtectionAndPostOnly: {Expected: "mmp_and_post_only"},
		order.Liquidation:                      {Error: order.ErrUnsupportedOrderType},
		order.OCO:                              {Expected: "oco"},
		order.TrailingStop:                     {Expected: "move_order_stop"},
		order.Chase:                            {Expected: "chase"},
		order.TWAP:                             {Expected: "twap"},
		order.ConditionalStop:                  {Expected: "conditional"},
		order.Trigger:                          {Expected: "trigger"},
	}
	for oType, val := range orderTypesToStringMap {
		orderTypeString, err := ok.OrderTypeString(oType)
		require.ErrorIs(t, err, val.Error)
		assert.Equal(t, val.Expected, orderTypeString)
	}
}

func TestGetMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMarkPriceCandlesticks(context.Background(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetMarkPriceCandlesticks(context.Background(), spotTP.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricIndexCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetHistoricIndexCandlesticksHistory(context.Background(), "", time.Time{}, time.Time{}, kline.TenMin, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := ok.GetHistoricIndexCandlesticksHistory(context.Background(), spotTP.String(), time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAssetTypeString(t *testing.T) {
	t.Parallel()
	assetTypes := ok.GetAssetTypes(false)
	for a := range assetTypes {
		_, err := AssetTypeString(assetTypes[a])
		assert.NoError(t, err)
	}
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := ok.GetAnnouncements(context.Background(), "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAnnouncementTypes(t *testing.T) {
	t.Parallel()
	results, err := ok.GetAnnouncementTypes(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestGetDepositOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDepositOrderDetail(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetDepositOrderDetail(context.Background(), "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFiatDepositOrderHistory(context.Background(), currency.USDT, "TR_BANKS", "failed", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := ok.GetWithdrawalOrderDetail(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetWithdrawalOrderDetail(context.Background(), "024041201450544699")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatWithdrawalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFiatWithdrawalOrderHistory(context.Background(), currency.USDT, "SEPA", "failed", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawalOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.CancelWithdrawalOrder(context.Background(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.CancelWithdrawalOrder(context.Background(), "124041201450544699")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateWithdrawalOrder(t *testing.T) {
	t.Parallel()
	_, err := ok.CreateWithdrawalOrder(context.Background(), currency.BTC, "", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, errIDNotSet)
	_, err = ok.CreateWithdrawalOrder(context.Background(), currency.EMPTYCODE, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = ok.CreateWithdrawalOrder(context.Background(), currency.BTC, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 0)
	require.ErrorIs(t, err, order.ErrAmountBelowMin)
	_, err = ok.CreateWithdrawalOrder(context.Background(), currency.BTC, "1231312312", "", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, errPaymentMethodRequired)
	_, err = ok.CreateWithdrawalOrder(context.Background(), currency.BTC, "1231312312", "SEPA", "", 1000)
	require.ErrorIs(t, err, errIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	result, err := ok.CreateWithdrawalOrder(context.Background(), currency.BTC, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatWithdrawalPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFiatWithdrawalPaymentMethods(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFiatWithdrawalPaymentMethods(context.Background(), currency.TRY)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFiatDepositPaymentMethods(context.Background(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	result, err := ok.GetFiatDepositPaymentMethods(context.Background(), currency.TRY)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func (ok *Okx) instrumentFamilyFromInstID(instrumentType, instID string) (string, error) {
	ok.instrumentsInfoMapLock.Lock()
	defer ok.instrumentsInfoMapLock.Unlock()
	if instrumentType != "" {
		insts, okay := ok.instrumentsInfoMap[instrumentType]
		if !okay {
			return "", errInvalidInstrumentType
		}
		for a := range insts {
			if insts[a].InstrumentID == instID {
				return insts[a].InstrumentFamily, nil
			}
		}
	} else {
		for _, insts := range ok.instrumentsInfoMap {
			for a := range insts {
				if insts[a].InstrumentID == instID {
					return insts[a].InstrumentFamily, nil
				}
			}
		}
	}
	return "", fmt.Errorf("instrument family not found for instrument %s", instID)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ok.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ok.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{
		{Channel: subscription.MyAccountChannel, QualifiedChannel: `{"channel":"account"}`, Authenticated: true},
	}
	var pairs currency.Pairs
	for _, s := range ok.Features.Subscriptions {
		for _, a := range ok.GetAssetTypes(true) {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err = ok.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: "-"})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			name := channelName(s)
			if isSymbolChannel(s) {
				for i, p := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					s.QualifiedChannel = fmt.Sprintf(`{"channel":%q,"instID":%q}`, name, p)
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s := s.Clone() //nolint:govet // Intentional lexical scope shadow
				if isAssetChannel(s) {
					s.QualifiedChannel = fmt.Sprintf(`{"channel":%q,"instType":%q}`, name, ok.GetInstrumentTypeFromAssetItem(s.Asset))
				} else {
					s.QualifiedChannel = `{"channel":"` + name + `"}`
				}
				s.Pairs = pairs
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)

	ok.Features.Subscriptions = subscription.List{{Channel: okxChannelGridPositions, Params: map[string]any{"algoId": "42"}}}
	subs, err = ok.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp = subscription.List{{Channel: okxChannelGridPositions, Params: map[string]any{"algoId": "42"}, QualifiedChannel: `{"channel":"grid-positions","algoId":"42"}`}}
	testsubs.EqualLists(t, exp, subs)

	ok.Features.Subscriptions = subscription.List{{Channel: okxChannelGridPositions}}
	subs, err = ok.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp = subscription.List{{Channel: okxChannelGridPositions, QualifiedChannel: `{"channel":"grid-positions"}`}}
	testsubs.EqualLists(t, exp, subs)
}

const (
	processSpreadOrderbookJSON      = `{"arg":{"channel":"sprd-books5", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "asks": [ ["111.06","55154","2"], ["111.07","53276","2"], ["111.08","72435","2"], ["111.09","70312","2"], ["111.1","67272","2"]], "bids": [ ["111.05","57745","2"], ["111.04","57109","2"], ["111.03","69563","2"], ["111.02","71248","2"], ["111.01","65090","2"]], "ts": "1670324386802"}]}`
	wsProcessPublicSpreadTradesJSON = `{"arg":{"channel":"sprd-public-trades", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "sprdId": "BTC-USDT_BTC-USDT-SWAP", "tradeId": "2499206329160695808", "px": "-10", "sz": "0.001", "side": "sell", "ts": "1726801105519"}]}`
	okxSpreadPublicTickerJSON       = `{"arg":{"channel":"sprd-tickers", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "sprdId": "BTC-USDT_BTC-USDT-SWAP", "last": "4", "lastSz": "0.01", "askPx": "19.7", "askSz": "5.79", "bidPx": "5.9", "bidSz": "5.79", "open24h": "-7", "high24h": "19.6", "low24h": "-7", "vol24h": "9.87", "ts": "1715247061026"}]}`
	wsProcessSpreadOrdersJSON       = `{"arg":{"channel":"sprd-orders","sprdId": "BTC-USDT_BTC-USDT-SWAP", "uid": "614488474791936"}, "data": [{"sprdId": "BTC-USDT_BTC-UST-SWAP", "ordId": "312269865356374016", "clOrdId": "b1", "tag": "", "px": "999", "sz": "3", "ordType": "limit", "side": "buy", "fillSz": "0", "fillPx": "", "tradeId": "", "accFillSz": "0", "pendingFillSz": "2", "pendingSettleSz": "1", "canceledSz": "1", "state": "live", "avgPx": "0", "cancelSource": "", "uTime": "1597026383085", "cTime": "1597026383085", "code": "0", "msg": "", "reqId": "", "amendResult": ""}]}`
	wsProcessSpreadTradesJSON       = `{"arg":{"channel":"sprd-trades", "sprdId": "BTC-USDT_BTC-USDT-SWAP", "uid": "614488474791936" }, "data":[ { "sprdId":"BTC-USDT-SWAP_BTC-USDT-200329", "tradeId":"123", "ordId":"123445", "clOrdId": "b16", "tag":"", "fillPx":"999", "fillSz":"3", "state": "filled", "side":"buy", "execType":"M", "ts":"1597026383085", "legs": [ { "instId": "BTC-USDT-SWAP", "px": "20000", "sz": "3", "szCont": "0.03", "side": "buy", "fee": "", "feeCcy": "", "tradeId": "1232342342" }, { "instId": "BTC-USDT-200329", "px": "21000", "sz": "3", "szCont": "0.03", "side": "sell", "fee": "", "feeCcy": "", "tradeId": "5345646634" } ], "code": "", "msg":""}]}`
)

func TestWsProcessSpreadOrderbook(t *testing.T) {
	t.Parallel()
	err := ok.wsProcessSpreadOrderbook([]byte(processSpreadOrderbookJSON))
	assert.NoError(t, err)
}

func TestWsProcessPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	err := ok.wsProcessPublicSpreadTrades([]byte(wsProcessPublicSpreadTradesJSON))
	assert.NoError(t, err)
}

func TestWsProcessPublicSpreadTicker(t *testing.T) {
	t.Parallel()
	err := ok.wsProcessPublicSpreadTicker([]byte(okxSpreadPublicTickerJSON))
	assert.NoError(t, err)
}

func TestWsProcessSpreadOrders(t *testing.T) {
	t.Parallel()
	err := ok.wsProcessSpreadOrders([]byte(wsProcessSpreadOrdersJSON))
	assert.NoError(t, err)
}

func TestWsProcessSpreadTradesJSON(t *testing.T) {
	t.Parallel()
	err := ok.wsProcessSpreadTrades([]byte(wsProcessSpreadTradesJSON))
	assert.NoError(t, err)
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	orderTypeStrings := map[string]struct {
		OType order.Type
		Error error
	}{
		"market":            {OType: order.Market, Error: nil},
		"LIMIT":             {OType: order.Limit, Error: nil},
		"limit":             {OType: order.Limit, Error: nil},
		"post_only":         {OType: order.PostOnly, Error: nil},
		"fok":               {OType: order.FillOrKill, Error: nil},
		"ioc":               {OType: order.ImmediateOrCancel, Error: nil},
		"optimal_limit_ioc": {OType: order.OptimalLimitIOC, Error: nil},
		"mmp":               {OType: order.MarketMakerProtection, Error: nil},
		"mmp_and_post_only": {OType: order.MarketMakerProtectionAndPostOnly, Error: nil},
		"trigger":           {OType: order.UnknownType, Error: order.ErrTypeIsInvalid},
	}
	for a := range orderTypeStrings {
		oType, err := ok.OrderTypeFromString(a)
		require.ErrorIs(t, err, orderTypeStrings[a].Error)
		assert.Equal(t, oType, orderTypeStrings[a].OType)
	}
}
