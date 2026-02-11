package okx

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
	e *Exchange

	leadTraderUniqueID string
	loadLeadTraderOnce sync.Once
	errSyncLeadTrader  error

	mainPair          = currency.NewPairWithDelimiter("BTC", "USDT", "-") // Is used for spot, margin symbols and underlying contracts
	optionsPair       = currency.NewPairWithDelimiter("BTC", "USD", "-")
	perpetualSwapPair = currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-")
	spreadPair        = currency.NewPairWithDelimiter("BTC-USDT", "BTC-USDT-SWAP", "_")
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Okx Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" && passphrase != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, passphrase, "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	os.Exit(m.Run())
}

func syncLeadTraderUniqueID(t *testing.T) error {
	t.Helper()

	if useTestNet {
		t.Skip("Testnet does not support lead trader API")
	}

	loadLeadTraderOnce.Do(func() {
		result, err := e.GetLeadTradersRanks(contextGenerate(), &LeadTraderRanksRequest{
			InstrumentType: instTypeSwap,
			SortType:       "pnl_ratio",
			HasVacancy:     true,
			Limit:          10,
		})
		if err != nil {
			errSyncLeadTrader = fmt.Errorf("GetLeadTradersRanks failed: %s", err)
			return
		}
		if len(result) == 0 {
			errSyncLeadTrader = errors.New("no lead trader found")
			return
		}
		if len(result[0].Ranks) == 0 {
			errSyncLeadTrader = errors.New("could not load lead traders ranks")
			return
		}

		leadTraderUniqueID = result[0].Ranks[0].UniqueCode
	})

	return errSyncLeadTrader
}

// contextGenerate sends an optional value to allow test requests
// named this way, so it shows up in auto-complete and reminds you to use it
func contextGenerate() context.Context {
	return context.WithValue(context.Background(), testNetVal, useTestNet)
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	testexch.UpdatePairsOnce(t, e)
	pairs, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, pairs, "GetAvailablePairs must not return empty pairs")

	instFamily, err := e.instrumentFamilyFromInstID(instTypeOption, pairs[0].String())
	require.NoError(t, err, "instrumentFamilyFromInstID must not error")

	_, err = e.GetTickers(contextGenerate(), "", "", instFamily)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetTickers(contextGenerate(), instTypeOption, "", instFamily)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexTickers(contextGenerate(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	result, err := e.GetIndexTickers(contextGenerate(), currency.USDT, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetTicker(contextGenerate(), perpetualSwapPair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPremiumHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumHistory(contextGenerate(), "", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetPremiumHistory(contextGenerate(), perpetualSwapPair.String(), time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBookDepth(contextGenerate(), "", 400)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetOrderBookDepth(contextGenerate(), mainPair.String(), 400)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticks(contextGenerate(), "", kline.OneHour, time.Now().Add(-time.Minute*2), time.Now(), 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetCandlesticks(contextGenerate(), mainPair.String(), kline.OneHour, time.Now().Add(-time.Hour), time.Now(), 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticksHistory(contextGenerate(), "", kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetCandlesticksHistory(contextGenerate(), mainPair.String(), kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(contextGenerate(), "", 3)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetTrades(contextGenerate(), mainPair.String(), 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradesHistory(contextGenerate(), "", "", "", 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetTradesHistory(contextGenerate(), mainPair.String(), "", "", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTradesByInstrumentFamily(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTradesByInstrumentFamily(contextGenerate(), "")
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	result, err := e.GetOptionTradesByInstrumentFamily(contextGenerate(), optionsPair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTrades(contextGenerate(), "", "", "C")
	require.ErrorIs(t, err, errInstrumentIDorFamilyRequired)

	result, err := e.GetOptionTrades(contextGenerate(), "", optionsPair.String(), "C")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	result, err := e.Get24HTotalVolume(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping test: The server endpoint has a rate-limiting issue that needs to be fixed.")
	result, err := e.GetOracle(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeRate(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexComponents(contextGenerate(), "")
	require.ErrorIs(t, err, errIndexComponentNotFound)

	result, err := e.GetIndexComponents(contextGenerate(), mainPair.String())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Index, "Index should not be empty")
	assert.NotEmpty(t, result.Components, "Components should not be empty")
}

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockTickers(contextGenerate(), "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetBlockTickers(contextGenerate(), "SWAP", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetBlockTicker(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetBlockTicker(contextGenerate(), mainPair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicBlockTrades(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	trades, err := e.GetPublicBlockTrades(contextGenerate(), mainPair.String())
	require.NoError(t, err)
	if assert.NotEmpty(t, trades, "Should get some block trades") {
		blockTrade := trades[0]
		assert.Equal(t, mainPair.String(), blockTrade.InstrumentID, "InstrumentID should have correct value")
		assert.NotEmpty(t, blockTrade.TradeID, "TradeID should not be empty")
		assert.Positive(t, blockTrade.Price.Float64(), "Price should have a positive value")
		assert.Positive(t, blockTrade.Size.Float64(), "Size should have a positive value")
		assert.Contains(t, []order.Side{order.Buy, order.Sell}, blockTrade.Side, "Side should be a side")
		assert.WithinRange(t, blockTrade.Timestamp.Time(), time.Now().Add(time.Hour*-24*90), time.Now(), "Timestamp should be within last 90 days")
	}

	testexch.UpdatePairsOnce(t, e)

	pairs, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err)
	require.NotEmpty(t, pairs)

	publicTrades, err := e.GetPublicRFQTrades(contextGenerate(), "", "", 100)
	require.NoError(t, err)

	tested := false
LOOP:
	for _, trade := range publicTrades {
		for _, leg := range trade.Legs {
			p, err := e.MatchSymbolWithAvailablePairs(leg.InstrumentID, asset.Options, true)
			if err != nil {
				continue
			}

			trades, err = e.GetPublicBlockTrades(contextGenerate(), p.String())
			require.NoError(t, err, "GetBlockTrades must not error on Options")
			for _, trade := range trades {
				assert.Equal(t, p.String(), trade.InstrumentID, "InstrumentID should have correct value")
				assert.NotEmpty(t, trade.TradeID, "TradeID should not be empty")
				assert.Positive(t, trade.Price.Float64(), "Price should have a positive value")
				assert.Positive(t, trade.Size.Float64(), "Size should have a positive value")
				assert.Contains(t, []order.Side{order.Buy, order.Sell}, trade.Side, "Side should be a side")
				assert.GreaterOrEqual(t, trade.FillVolatility.Float64(), float64(0), "FillVolatility should not be negative")
				assert.Positive(t, trade.ForwardPrice.Float64(), "ForwardPrice should have a positive value")
				assert.Positive(t, trade.IndexPrice.Float64(), "IndexPrice should have a positive value")
				assert.Positive(t, trade.MarkPrice.Float64(), "MarkPrice should have a positive value")
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
	_, err := e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{Underlying: mainPair.String()})
	assert.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: instTypeOption, Underlying: "",
	})
	assert.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	resp, err := e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: instTypeFutures,
		Underlying:     "SOL-USD",
	})
	require.NoError(t, err)
	assert.Empty(t, resp, "Should get back no instruments for SOL-USD futures")

	result, err := e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: instTypeSpot,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	_, err = e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: instTypeSwap,
		Underlying:     mainPair.String(),
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryHistory(contextGenerate(), "", mainPair.String(), "", time.Time{}, time.Time{}, 3)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.GetDeliveryHistory(contextGenerate(), instTypeFutures, "", "", time.Time{}, time.Time{}, 3)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	_, err = e.GetDeliveryHistory(contextGenerate(), instTypeFutures, mainPair.String(), "", time.Time{}, time.Time{}, 345)
	require.ErrorIs(t, err, errLimitValueExceedsMaxOf100)

	result, err := e.GetDeliveryHistory(contextGenerate(), instTypeFutures, mainPair.String(), "", time.Time{}, time.Time{}, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestData(contextGenerate(), "", mainPair.String(), "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.GetOpenInterestData(contextGenerate(), instTypeOption, "", "", "")
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, p, "GetAvailablePairs must not return empty pairs")

	uly, err := e.underlyingFromInstID(instTypeOption, p[0].String())
	require.NoError(t, err)

	result, err := e.GetOpenInterestData(contextGenerate(), instTypeOption, uly, optionsPair.String(), p[0].String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func (e *Exchange) underlyingFromInstID(instrumentType, instID string) (string, error) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	if instrumentType != "" {
		insts, okay := e.instrumentsInfoMap[instrumentType]
		if !okay {
			return "", errInvalidInstrumentType
		}
		for a := range insts {
			if insts[a].InstrumentID.String() == instID {
				return insts[a].Underlying, nil
			}
		}
	} else {
		for _, insts := range e.instrumentsInfoMap {
			for a := range insts {
				if insts[a].InstrumentID.String() == instID {
					return insts[a].Underlying, nil
				}
			}
		}
	}
	return "", fmt.Errorf("underlying not found for instrument %s", instID)
}

func TestGetSingleFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetSingleFundingRate(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetSingleFundingRate(contextGenerate(), perpetualSwapPair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateHistory(contextGenerate(), "", time.Time{}, time.Time{}, 2)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetFundingRateHistory(contextGenerate(), perpetualSwapPair.String(), time.Time{}, time.Time{}, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLimitPrice(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetLimitPrice(contextGenerate(), perpetualSwapPair.String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionMarketData(contextGenerate(), "", "", time.Time{})
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	result, err := e.GetOptionMarketData(contextGenerate(), "BTC-USD", "", time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedDeliveryPrice(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetAvailablePairs(asset.Futures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, p, "GetAvailablePairs must not return empty pairs")

	result, err := e.GetEstimatedDeliveryPrice(contextGenerate(), p[0].String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	result, err := e.GetDiscountRateAndInterestFreeQuota(contextGenerate(), currency.EMPTYCODE, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTime(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Time().IsZero(), "GetSystemTime should not return a zero time")
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()

	result, err := e.GetLiquidationOrders(contextGenerate(), &LiquidationOrderRequestParams{
		InstrumentType: instTypeMargin,
		Underlying:     mainPair.String(),
		Currency:       currency.BTC,
		Limit:          2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPrice(contextGenerate(), "", "", "", mainPair.String())
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetMarkPrice(contextGenerate(), "MARGIN", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionTiers(contextGenerate(), "", "cross", mainPair.String(), "", "", "", currency.ETH)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	_, err = e.GetPositionTiers(contextGenerate(), instTypeFutures, "", mainPair.String(), "", "", "", currency.ETH)
	require.ErrorIs(t, err, errInvalidTradeMode)

	_, err = e.GetPositionTiers(contextGenerate(), instTypeFutures, "cross", "", "", "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	_, err = e.GetPositionTiers(contextGenerate(), instTypeFutures, "cross", mainPair.String(), "", "", "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	result, err := e.GetPositionTiers(contextGenerate(), instTypeFutures, "cross", mainPair.String(), "", "", "", currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	result, err := e.GetInterestRateAndLoanQuota(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRateAndLoanQuotaForVIPLoans(t *testing.T) {
	t.Parallel()
	result, err := e.GetInterestRateAndLoanQuotaForVIPLoans(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicUnderlyings(contextGenerate(), "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetPublicUnderlyings(contextGenerate(), instTypeFutures)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetInsuranceFundInformation(contextGenerate(), &InsuranceFundInformationRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &InsuranceFundInformationRequestParams{Limit: 2}
	_, err = e.GetInsuranceFundInformation(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	arg.InstrumentType = instTypeSwap
	_, err = e.GetInsuranceFundInformation(contextGenerate(), arg)
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	arg.Underlying = mainPair.String()
	r, err := e.GetInsuranceFundInformation(contextGenerate(), arg)
	require.NoError(t, err)
	assert.Positive(t, r.Total, "Total should be positive")
	assert.NotEmpty(t, r.Details, "Should have some details")
	for _, d := range r.Details {
		assert.Positive(t, d.Balance, "Balance should be positive")
		assert.NotEmpty(t, d.InsuranceType, "Type should not be empty")
		assert.Positive(t, d.Timestamp, "Timestamp should be positive")
	}

	r, err = e.GetInsuranceFundInformation(contextGenerate(), &InsuranceFundInformationRequestParams{
		InstrumentType: instTypeFutures,
		Underlying:     mainPair.String(),
		Limit:          2,
	})
	require.NoError(t, err)
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
	_, err := e.CurrencyUnitConvert(contextGenerate(), "", 1, 3500, 1, currency.EMPTYCODE, false)
	require.ErrorIs(t, err, errMissingInstrumentID)

	_, err = e.CurrencyUnitConvert(contextGenerate(), perpetualSwapPair.String(), 0, 3500, 1, currency.EMPTYCODE, false)
	require.ErrorIs(t, err, errMissingQuantity)

	result, err := e.CurrencyUnitConvert(contextGenerate(), perpetualSwapPair.String(), 1, 3500, 1, currency.EMPTYCODE, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetSupportCoins(contextGenerate())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Spot, "SupportedCoins Spot should not be empty")
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetTakerVolume(contextGenerate(), currency.BTC, "", "", time.Time{}, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetTakerVolume(contextGenerate(), currency.BTC, instTypeSpot, "", time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarginLendingRatio(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetLongShortRatio(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	result, err := e.GetContractsOpenInterestAndVolume(contextGenerate(), currency.BTC, time.Time{}, time.Time{}, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	result, err := e.GetOptionsOpenInterestAndVolume(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetPutCallRatio(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	result, err := e.GetOpenInterestAndVolumeExpiry(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestAndVolumeStrike(contextGenerate(), currency.BTC, time.Time{}, kline.OneDay)
	require.ErrorIs(t, err, errMissingExpiryTimeParameter)

	instruments, err := e.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: instTypeOption,
		Underlying:     optionsPair.String(),
	})
	require.NoErrorf(t, err, "GetInstruments for options (underlying: %s) must not error", optionsPair)
	require.NotEmptyf(t, instruments, "GetInstruments for options (underlying: %s) must return at least one instrument", optionsPair)
	var selectedExpTime time.Time
	for _, inst := range instruments {
		if inst.ExpTime.Time().IsZero() {
			continue
		}
		selectedExpTime = inst.ExpTime.Time()
		break
	}
	require.NotZero(t, selectedExpTime, "GetInstruments must return an instrument with a non-zero expiry time")
	result, err := e.GetOpenInterestAndVolumeStrike(contextGenerate(), currency.BTC, selectedExpTime, kline.OneDay)
	require.NoErrorf(t, err, "GetOpenInterestAndVolumeStrike with expiry %s for currency %s must not error", selectedExpTime, currency.BTC)
	assert.NotNilf(t, result, "GetOpenInterestAndVolumeStrike with expiry %s for currency %s should return a non-nil result", selectedExpTime, currency.BTC)
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	result, err := e.GetTakerFlow(contextGenerate(), currency.BTC, kline.OneDay)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Timestamp.Time().IsZero(), "Timestamp should not be zero")
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceOrderRequestParam{
		ReduceOnly: true,
		AssetType:  asset.Margin,
	}
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.Lower()
	arg.TradeMode = "abc"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	arg.TargetCurrency = "abcd"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errCurrencyQuantityTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         order.Buy.String(),
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
		AssetType:    asset.Margin,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
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

	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{})
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	arg := PlaceOrderRequestParam{
		ReduceOnly: true,
	}
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	arg.TradeMode = "abc"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = orderLimit
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequestParam{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceMultipleOrders(contextGenerate(), params)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{OrderID: "12321312312"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.CancelSingleOrder(contextGenerate(), &CancelOrderRequestParam{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelSingleOrder(contextGenerate(),
		&CancelOrderRequestParam{
			InstrumentID: mainPair.String(),
			OrderID:      "2510789768709120",
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := CancelOrderRequestParam{}
	_, err = e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{
		{
			InstrumentID: mainPair.String(),
			OrderID:      "2510789768709120",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &AmendOrderRequestParams{}
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendOrder(contextGenerate(), &AmendOrderRequestParams{
		InstrumentID: mainPair.String(),
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := AmendOrderRequestParams{
		NewPriceInUSD: 1233,
	}
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123212"
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{arg})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{{
		InstrumentID: mainPair.String(),
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{MarginMode: "cross"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{InstrumentID: mainPair.String(), MarginMode: "abc"})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{
		InstrumentID: mainPair.String(),
		MarginMode:   "cross",
		Currency:     "BTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{OrderID: "1234"})
	assert.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{InstrumentID: mainPair.String()})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{InstrumentID: "SUI-USDT", OrderID: "1974857619964870656"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderList(contextGenerate(), &OrderListRequestParams{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderList(contextGenerate(), &OrderListRequestParams{Limit: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGet7DayOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.getOrderHistory(contextGenerate(), &OrderHistoryRequestParams{}, "", request.UnAuth)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.getOrderHistory(contextGenerate(), &OrderHistoryRequestParams{Category: "abc"}, "", request.UnAuth)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequestParams{OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"}})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGet3MonthOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequestParams{OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.getTransactionDetails(contextGenerate(), &TransactionDetailRequestParams{}, "", request.UnAuth)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.getTransactionDetails(contextGenerate(), &TransactionDetailRequestParams{Limit: 10}, "", request.UnAuth)
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransactionDetailsLast3Days(contextGenerate(), &TransactionDetailRequestParams{InstrumentType: "MARGIN", Limit: 1})
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestGetTransactionDetailsLast3Months(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequestParams{InstrumentType: "MARGIN"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAlgoOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PlaceAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
}

func TestStopOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceStopOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	arg.OrderType = "conditional"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.TakeProfitTriggerPrice = 123
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	arg.TakeProfitTriggerPriceType = "last_price"
	arg.AlgoClientOrderID = "12345"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	arg.OrderType = ""
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PlaceStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceStopOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID:          "681096944655273984",
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               mainPair.String(),
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
	_, err := e.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{OrderType: "iceberg"})
	require.ErrorIs(t, err, errMissingSizeLimit)
	_, err = e.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{OrderType: "iceberg", SizeLimit: 123})
	require.ErrorIs(t, err, errInvalidPriceLimit)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		LimitPrice:        100.22,
		SizeLimit:         9999.9,
		PriceSpread:       0.04,
		InstrumentID:      mainPair.String(),
		OrderType:         "iceberg",
		Side:              order.Buy.Lower(),
		TradeMode:         "isolated",
		Size:              6,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{OrderType: "twap"})
	require.ErrorIs(t, err, errMissingSizeLimit)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{SizeLimit: 2, OrderType: "twap"})
	require.ErrorIs(t, err, errInvalidPriceLimit)

	_, err = e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{SizeLimit: 2, OrderType: "twap", LimitPrice: 1234.5})
	require.ErrorIs(t, err, errMissingIntervalValue)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      mainPair.String(),
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

func TestPlaceTakeProfitStopLossOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceTakeProfitStopLossOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &AlgoOrderParams{ReduceOnly: true})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &AlgoOrderParams{OrderType: "conditional"})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)
	_, err = e.PlaceTakeProfitStopLossOrder(contextGenerate(), &AlgoOrderParams{
		OrderType:                "conditional",
		StopLossTriggerPrice:     1234,
		StopLossTriggerPriceType: "abcd",
	})
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceTakeProfitStopLossOrder(contextGenerate(), &AlgoOrderParams{
		OrderType:                "conditional",
		StopLossTriggerPrice:     1234,
		StopLossTriggerPriceType: "last",
		AlgoClientOrderID:        "681096944655273984",
		InstrumentID:             mainPair.String(),
		LimitPrice:               100.22,
		SizeLimit:                9999.9,
		PriceSpread:              0.4,
		TradeMode:                "cross",
		Side:                     order.Sell.Lower(),
		Size:                     6,
		TimeInterval:             kline.ThreeDay,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceChaseAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceChaseAlgoOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &AlgoOrderParams{
		ReduceOnly: true,
	}
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = orderChase
	arg.MaxChaseType = "percentage"
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errPriceTrackingNotSet)

	arg.MaxChaseType = "percentage"
	arg.MaxChaseValue = .5
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Sell.Lower()
	_, err = e.PlaceChaseAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceChaseAlgoOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		InstrumentID:      mainPair.String(),
		LimitPrice:        100.22,
		OrderType:         orderChase,
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
	_, err := e.PlaceTriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{AlgoClientOrderID: "1234"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{AlgoClientOrderID: "1234", OrderType: "trigger"})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	_, err = e.PlaceTriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{AlgoClientOrderID: "1234", OrderType: "trigger", TriggerPrice: 123., TriggerPriceType: "abcd"})
	require.ErrorIs(t, err, order.ErrUnknownPriceType)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceTriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984",
		TriggerPriceType:  "mark",
		TriggerPrice:      1234,
		InstrumentID:      mainPair.String(),
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
	_, err := e.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{Size: 2})
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{Size: 2, OrderType: orderMoveOrderStop})
	assert.ErrorIs(t, err, errPriceTrackingNotSet)

	// Offline error handling unit tests for the base function PlaceAlgoOrder are already covered within unit test TestPlaceAlgoOrder.
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{
		AlgoClientOrderID: "681096944655273984", CallbackRatio: 0.01,
		InstrumentID: mainPair.String(), OrderType: orderMoveOrderStop,
		Side: order.Buy.Lower(), TradeMode: "isolated",
		Size: 2, ActivePrice: 1234,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	arg := AlgoOrderCancelParams{}
	_, err := e.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.AlgoOrderID = "90994943"
	_, err = e.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	arg.AlgoOrderID = ""
	_, err = e.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{
		{
			InstrumentID: mainPair.String(),
			AlgoOrderID:  "90994943",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAdvanceAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{}})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{InstrumentID: "90994943"}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{AlgoOrderID: "90994943"}})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{
		InstrumentID: mainPair.String(),
		AlgoOrderID:  "90994943",
	}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderList(contextGenerate(), "", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlgoOrderList(contextGenerate(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderHistory(contextGenerate(), "", "effective", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)
	_, err = e.GetAlgoOrderHistory(contextGenerate(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1)
	require.ErrorIs(t, err, errMissingEitherAlgoIDOrState)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlgoOrderHistory(contextGenerate(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEasyConvertCurrencyList(contextGenerate(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOneClickRepayCurrencyList(contextGenerate(), "cross")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceEasyConvert(contextGenerate(), PlaceEasyConvertParam{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.PlaceEasyConvert(contextGenerate(), PlaceEasyConvertParam{FromCurrency: []string{"BTC"}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceEasyConvert(contextGenerate(), PlaceEasyConvertParam{FromCurrency: []string{"BTC"}, ToCurrency: "USDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEasyConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOneClickRepayHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	_, err := e.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{DebtCurrency: []string{}, RepayCurrency: "USDT"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{DebtCurrency: []string{"BTC"}, RepayCurrency: ""})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{
		DebtCurrency:  []string{"BTC"},
		RepayCurrency: "USDT",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCounterparties(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const createRFQInputJSON = `{"anonymous": true,"counterparties":["Trader1","Trader2"],"clRfqId":"rfq01","legs":[{"sz":"25","side":"buy","instId":"BTCUSD-221208-100000-C"},{"sz":"150","side":"buy","instId":"BTC-USDT","tgtCcy":"base_ccy"}]}`

func TestCreateRFQ(t *testing.T) {
	t.Parallel()
	var input *CreateRFQInput
	err := json.Unmarshal([]byte(createRFQInputJSON), &input)
	require.NoError(t, err)

	_, err = e.CreateRFQ(contextGenerate(), &CreateRFQInput{CounterParties: []string{}})
	require.ErrorIs(t, err, errInvalidCounterParties)

	_, err = e.CreateRFQ(contextGenerate(), &CreateRFQInput{CounterParties: []string{"Trader1"}})
	require.ErrorIs(t, err, errMissingLegs)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateRFQ(contextGenerate(), input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CancelRFQ(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelRFQ(contextGenerate(), "", "somersdjskfjsdkfjxvxv")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMultipleCancelRFQ(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleRFQs(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.CancelMultipleRFQs(contextGenerate(), &CancelRFQRequestsParam{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	_, err = e.CancelMultipleRFQs(contextGenerate(), &CancelRFQRequestsParam{RFQIDs: make([]string, 100), ClientRFQIDs: make([]string, 100)})
	require.ErrorIs(t, err, errMaxRFQOrdersToCancel)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMultipleRFQs(contextGenerate(), &CancelRFQRequestsParam{ClientRFQIDs: []string{"somersdjskfjsdkfjxvxv"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllRFQs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllRFQs(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	_, err := e.ExecuteQuote(contextGenerate(), "", "")
	assert.ErrorIs(t, err, errMissingRFQIDOrQuoteID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ExecuteQuote(contextGenerate(), "22540", "84073")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuoteProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetQuoteProducts(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{{InstrumentType: "ABC"}})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	arg := SetQuoteProductParam{InstrumentType: "SWAP"}
	_, err = e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errMissingMakerInstrumentSettings)

	data := MakerInstrumentSetting{MaxBlockSize: 10000, MakerPriceBand: 5}
	arg.Data = []MakerInstrumentSetting{data}
	_, err = e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errInvalidUnderlying)

	arg.InstrumentType = "SPOT"
	data = MakerInstrumentSetting{Underlying: "BTC-USD", MaxBlockSize: 10000, MakerPriceBand: 5}
	arg.Data = []MakerInstrumentSetting{data}
	_, err = e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{
		{
			InstrumentType: "SWAP",
			Data: []MakerInstrumentSetting{
				{
					Underlying:     "BTC-USD",
					MaxBlockSize:   10000,
					MakerPriceBand: 5,
				},
				{
					Underlying: mainPair.String(),
				},
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetRFQMMPStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ResetRFQMMPStatus(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CreateQuote(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &CreateQuoteParams{}
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingRFQID)

	arg.RFQID = "123456789"
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.QuoteSide = "sell"
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingLegs)

	subArg := QuoteLeg{}
	arg.Legs = []QuoteLeg{subArg}
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	subArg.InstrumentID = "SOL-USD-220909"
	arg.Legs = []QuoteLeg{subArg}
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingSizeOfQuote)

	subArg.SizeOfQuoteLeg = 2
	arg.Legs = []QuoteLeg{subArg}
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingLegsQuotePrice)

	subArg.Price = 1234
	arg.Legs = []QuoteLeg{subArg}
	_, err = e.CreateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateQuote(contextGenerate(), &CreateQuoteParams{
		RFQID:     "12345",
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
	_, err := e.CancelQuote(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelQuote(contextGenerate(), "1234", "")
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.CancelQuote(contextGenerate(), "", "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{
		QuoteIDs: []string{"1150", "1151", "1152"},
		// Block trades require a minimum of $100,000 in assets in your trading account
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllRFQQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	tt, err := e.CancelAllRFQQuotes(contextGenerate())
	require.NoError(t, err)
	assert.NotEmpty(t, tt)
}

func TestGetRFQs(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQs(contextGenerate(), &RFQRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRFQs(contextGenerate(), &RFQRequestParams{
		Limit: 1,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuotes(contextGenerate(), &QuoteRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetQuotes(contextGenerate(), &QuoteRequestParams{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRFQTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRFQTrades(contextGenerate(), &RFQTradesRequestParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRFQTrades(contextGenerate(), &RFQTradesRequestParams{Limit: 1})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicRFQTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPublicRFQTrades(contextGenerate(), "", "", 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingCurrencies(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBalance(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNonTradableAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetNonTradableAssets(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountAssetValuation(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{})
	assert.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		BeneficiaryAccountType: "6", RemittingAccountType: "18", Currency: currency.BTC,
	})
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount: 12.000, BeneficiaryAccountType: "6",
		RemittingAccountType: "18", Currency: currency.EMPTYCODE,
	})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount: 12.000, BeneficiaryAccountType: "2",
		RemittingAccountType: "3", Currency: currency.BTC,
	})
	assert.ErrorIs(t, err, errAddressRequired)
	_, err = e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount: 12.000, BeneficiaryAccountType: "2",
		RemittingAccountType: "18", Currency: currency.BTC,
	})
	assert.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount:                 12.000,
		BeneficiaryAccountType: "6",
		RemittingAccountType:   "18",
		Currency:               currency.BTC,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundsTransferState(contextGenerate(), "", "", 1)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundsTransferState(contextGenerate(), "754147", "1232", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAssetBillsDetails(contextGenerate(), currency.EMPTYCODE, "", time.Time{}, time.Time{}, 0, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	_, err := e.GetLightningDeposits(contextGenerate(), currency.EMPTYCODE, 1.00, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetLightningDeposits(contextGenerate(), currency.BTC, 0, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLightningDeposits(contextGenerate(), currency.BTC, 1.00, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDepositAddress(contextGenerate(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyDepositAddress(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCurrencyDepositHistory(contextGenerate(), currency.BTC, "", "", "", "271", time.Time{}, time.Time{}, 0, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.Withdrawal(contextGenerate(), &WithdrawalInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.EMPTYCODE, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.Withdrawal(contextGenerate(), &WithdrawalInput{TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.BTC, ToAddress: core.BitcoinDonationAddress})
	require.ErrorIs(t, err, errAddressRequired)
	_, err = e.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4"})
	require.ErrorIs(t, err, errAddressRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: -0.1, TransactionFee: 0.00005, Currency: currency.BTC, WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	require.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.Withdrawal(contextGenerate(), &WithdrawalInput{
		Amount:                -0.1,
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
	_, err := e.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{
		Invoice: "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz", Currency: currency.EMPTYCODE,
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = e.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{Invoice: "", Currency: currency.BTC})
	require.ErrorIs(t, err, errInvoiceTextMissing)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.LightningWithdrawal(contextGenerate(), &LightningWithdrawalRequestInput{
		Currency: currency.BTC,
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingValidWithdrawalID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelWithdrawal(contextGenerate(), "fjasdfkjasdk")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalHistory(contextGenerate(), currency.BTC, "", "", "", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SmallAssetsConvert(contextGenerate(), []string{"BTC", "USDT"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSavingBalance(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	_, err := e.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SavingsPurchaseRedemptionInput{Rate: 1}
	_, err = e.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.BTC
	_, err = e.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 123.4
	_, err = e.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Rate = 0.001
	arg.ActionType = "purchase"
	_, err = e.SavingsPurchaseOrRedemption(contextGenerate(), arg)
	require.ErrorIs(t, err, errRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   currency.BTC,
		Rate:       1,
		ActionType: "purchase",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
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
	_, err := e.SetLendingRate(contextGenerate(), &LendingRate{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.EMPTYCODE, Rate: 2})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.BTC})
	require.ErrorIs(t, err, errRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetLendingRate(contextGenerate(), &LendingRate{Currency: currency.BTC, Rate: 2})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLendingHistory(contextGenerate(), currency.USDT, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicBorrowInfo(contextGenerate(), currency.EMPTYCODE)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetPublicBorrowInfo(contextGenerate(), currency.USDT)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicBorrowHistory(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicBorrowHistory(contextGenerate(), currency.USDT, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := e.GetMonthlyStatement(contextGenerate(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	_, err = e.GetMonthlyStatement(contextGenerate(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMonthlyStatement(contextGenerate(), "Jan")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyForMonthlyStatement(t *testing.T) {
	t.Parallel()
	_, err := e.ApplyForMonthlyStatement(contextGenerate(), "")
	require.ErrorIs(t, err, errMonthNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ApplyForMonthlyStatement(contextGenerate(), "Jan")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetConvertCurrencies(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertCurrencyPair(contextGenerate(), currency.EMPTYCODE, currency.BTC)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetConvertCurrencyPair(contextGenerate(), currency.USDT, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetConvertCurrencyPair(contextGenerate(), currency.USDT, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	_, err := e.EstimateQuote(contextGenerate(), &EstimateQuoteRequestInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &EstimateQuoteRequestInput{Tag: "abcd"}
	_, err = e.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BaseCurrency = currency.BTC
	_, err = e.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	arg.QuoteCurrency = currency.BTC
	_, err = e.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	arg.Side = order.Sell.Lower()
	_, err = e.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	arg.RFQAmount = 30
	_, err = e.EstimateQuote(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.EstimateQuote(contextGenerate(), &EstimateQuoteRequestInput{
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
	_, err := e.ConvertTrade(contextGenerate(), &ConvertTradeInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := &ConvertTradeInput{Tag: "123"}
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.BaseCurrency = "BTC"
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.QuoteCurrency = "USDT"
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.Lower()
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Size = 2
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.SizeCurrency = currency.USDT
	_, err = e.ConvertTrade(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ConvertTrade(contextGenerate(), &ConvertTradeInput{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetNonZeroAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.AccountBalance(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPositions(contextGenerate(), "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPositionsHistory(contextGenerate(), "", "", "", "1234213123", 0, 1, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountAndPositionRisk(contextGenerate(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailQueryParameter{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailQueryParameter{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBillsDetail3Months(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsDetail3Months(contextGenerate(), &BillsDetailQueryParameter{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBillsDetail3Months(contextGenerate(), &BillsDetailQueryParameter{Limit: 3})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyBillDetails(t *testing.T) {
	t.Parallel()
	_, err := e.ApplyBillDetails(contextGenerate(), "", "Q2")
	require.ErrorIs(t, err, errYearRequired)
	_, err = e.ApplyBillDetails(contextGenerate(), "2023", "")
	require.ErrorIs(t, err, errQuarterValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ApplyBillDetails(contextGenerate(), "2023", "Q2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBillsHistoryArchive(t *testing.T) {
	t.Parallel()
	_, err := e.GetBillsHistoryArchive(contextGenerate(), "", "Q2")
	require.ErrorIs(t, err, errYearRequired)
	_, err = e.GetBillsHistoryArchive(contextGenerate(), "2023", "")
	require.ErrorIs(t, err, errQuarterValueRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBillsHistoryArchive(contextGenerate(), "2023", "Q2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountConfiguration(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.SetPositionMode(contextGenerate(), "")
	require.ErrorIs(t, err, errInvalidPositionMode)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetPositionMode(contextGenerate(), "net_mode")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverageRate(t *testing.T) {
	t.Parallel()
	_, err := e.SetLeverageRate(contextGenerate(), &SetLeverageInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetLeverageRate(contextGenerate(), &SetLeverageInput{Leverage: 5, MarginMode: "isolated", AssetType: asset.PerpetualSwap})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired)

	_, err = e.SetLeverageRate(contextGenerate(), &SetLeverageInput{
		Currency:     currency.USDT,
		Leverage:     5,
		MarginMode:   "isolated",
		InstrumentID: perpetualSwapPair.String(),
		AssetType:    asset.PerpetualSwap,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.SetLeverageRate(contextGenerate(), &SetLeverageInput{
		Currency:     currency.USDT,
		Leverage:     5,
		MarginMode:   "cross",
		InstrumentID: perpetualSwapPair.String(),
	})
	assert.Truef(t, err == nil || errors.Is(err, common.ErrNoResponse), "SetLeverageRate should not error: %s", err)
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, "", "cross", "", 5, true)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, mainPair.String(), "", "", 5, true)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), currency.BTC, mainPair.String(), "cross", "", 5, true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, "", "cross", "", true, false, 123)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, mainPair.String(), "", "", true, false, 123)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaximumAvailableTradableAmount(contextGenerate(), currency.BTC, mainPair.String(), "cross", "", true, false, 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	_, err := e.IncreaseDecreaseMargin(contextGenerate(), &IncreaseDecreaseMarginInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &IncreaseDecreaseMarginInput{Currency: "USD"}
	_, err = e.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.MarginBalanceType = "reduce"
	_, err = e.IncreaseDecreaseMargin(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.IncreaseDecreaseMargin(contextGenerate(), &IncreaseDecreaseMarginInput{
		InstrumentID:      mainPair.String(),
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
	_, err := e.GetLeverageRate(contextGenerate(), "", "cross", currency.EMPTYCODE)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetLeverageRate(contextGenerate(), mainPair.String(), "", currency.EMPTYCODE)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeverageRate(contextGenerate(), mainPair.String(), "cross", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumLoanOfInstrument(contextGenerate(), "", "isolated", currency.ZRX)
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetMaximumLoanOfInstrument(contextGenerate(), "ZRX-BTC", "", currency.ZRX)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaximumLoanOfInstrument(contextGenerate(), mainPair.String(), "isolated", currency.ZRX)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeFee(contextGenerate(), "", "", "", "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeFee(contextGenerate(), instTypeSpot, "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInterestAccruedData(contextGenerate(), 0, 1, currency.EMPTYCODE, "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInterestRate(contextGenerate(), currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	_, err := e.SetGreeks(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingValidGreeksType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetGreeks(contextGenerate(), "PA")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	_, err := e.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{IsoMode: "", InstrumentType: "MARGIN"})
	require.ErrorIs(t, err, errMissingIsolatedMarginTradingSetting)
	_, err = e.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{IsoMode: "autonomy", InstrumentType: ""})
	require.ErrorIs(t, err, errInvalidInstrumentType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.IsolatedMarginTradingSettings(contextGenerate(), &IsolatedMode{IsoMode: "autonomy", InstrumentType: "MARGIN"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMaximumWithdrawals(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountRiskState(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	_, err := e.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.EMPTYCODE, Side: "borrow", Amount: 12})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "", Amount: 12})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "borrow", Amount: 0})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.VIPLoansBorrowAndRepay(contextGenerate(), &LoanBorrowAndReplayInput{Currency: currency.BTC, Side: "borrow", Amount: 12})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBorrowAndRepayHistoryForVIPLoans(contextGenerate(), currency.EMPTYCODE, time.Time{}, time.Time{}, 3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBorrowInterestAndLimit(contextGenerate(), 1, currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFixedLoanBorrowLimit(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "", "30D", "123423423", 1, .4)
	require.ErrorIs(t, err, errBorrowTypeRequired)
	_, err = e.GetFixedLoanBorrowQuote(contextGenerate(), currency.EMPTYCODE, "normal", "30D", "123423423", 1, .4)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "normal", "30D", "", 0, .4)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "normal", "30D", "123423423", 1, 0)
	require.ErrorIs(t, err, errMaxRateRequired)
	_, err = e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "normal", "", "123423423", 1, .4)
	require.ErrorIs(t, err, errLendingTermIsRequired)
	_, err = e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "reborrow", "30D", "", 1, .4)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFixedLoanBorrowQuote(contextGenerate(), currency.USDT, "normal", "30D", "123423423", 1, .4)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceFixedLoanBorrowingOrder(contextGenerate(), currency.EMPTYCODE, 1, .3, .2, "30D", false)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.PlaceFixedLoanBorrowingOrder(contextGenerate(), currency.USDT, 0, .3, .2, "30D", false)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.PlaceFixedLoanBorrowingOrder(contextGenerate(), currency.USDT, 1, 0, .2, "30D", false)
	require.ErrorIs(t, err, errMaxRateRequired)
	_, err = e.PlaceFixedLoanBorrowingOrder(contextGenerate(), currency.USDT, 1, .3, .2, "", false)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceFixedLoanBorrowingOrder(contextGenerate(), currency.USDT, 1, .3, .2, "30D", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendFixedLoanBorrowingOrder(contextGenerate(), "", false, .4)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendFixedLoanBorrowingOrder(contextGenerate(), "12312312", false, .4)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualRenewFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ManualRenewFixedLoanBorrowingOrder(contextGenerate(), "", .3)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.ManualRenewFixedLoanBorrowingOrder(contextGenerate(), "12312312", 0)
	require.ErrorIs(t, err, errMaxRateRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ManualRenewFixedLoanBorrowingOrder(contextGenerate(), "12312312", .3)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRepayFixedLoanBorrowingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.RepayFixedLoanBorrowingOrder(contextGenerate(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.RepayFixedLoanBorrowingOrder(contextGenerate(), "12321")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConvertFixedLoanToMarketLoan(t *testing.T) {
	t.Parallel()
	_, err := e.ConvertFixedLoanToMarketLoan(contextGenerate(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ConvertFixedLoanToMarketLoan(contextGenerate(), "12321")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReduceLiabilitiesForFixedLoan(t *testing.T) {
	t.Parallel()
	_, err := e.ReduceLiabilitiesForFixedLoan(contextGenerate(), "", false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ReduceLiabilitiesForFixedLoan(contextGenerate(), "123123", false)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFixedLoanBorrowOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFixedLoanBorrowOrderList(contextGenerate(), currency.USDT, "1231231", "8", "30D", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualBorrowOrRepay(t *testing.T) {
	t.Parallel()
	_, err := e.ManualBorrowOrRepay(contextGenerate(), currency.EMPTYCODE, "borrow", 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.ManualBorrowOrRepay(contextGenerate(), currency.USDT, "", 1)
	require.ErrorIs(t, err, errLendingSideRequired)
	_, err = e.ManualBorrowOrRepay(contextGenerate(), currency.USDT, "borrow", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ManualBorrowOrRepay(contextGenerate(), currency.USDT, "borrow", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAutoRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetAutoRepay(contextGenerate(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBorrowRepayHistory(contextGenerate(), currency.ETH, "auto_borrow", time.Time{}, time.Time{}, 100)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNewPositionBuilder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPositionBuilder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.NewPositionBuilder(contextGenerate(), &PositionBuilderParam{
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
	_, err := e.SetRiskOffsetAmount(contextGenerate(), currency.EMPTYCODE, 123)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.SetRiskOffsetAmount(contextGenerate(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetRiskOffsetAmount(contextGenerate(), currency.USDT, 123)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetGreeks(contextGenerate(), currency.EMPTYCODE)
	assert.NoError(t, err)
}

func TestGetPMLimitation(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMPositionLimitation(contextGenerate(), "", mainPair.String(), "")
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetPMPositionLimitation(contextGenerate(), "SWAP", "", "")
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPMPositionLimitation(contextGenerate(), "SWAP", mainPair.String(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ViewSubAccountList(contextGenerate(), false, "", time.Time{}, time.Time{}, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	_, err := e.ResetSubAccountAPIKey(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{APIKey: apiKey, APIKeyPermission: "trade"})
	require.ErrorIs(t, err, errInvalidSubAccountName)
	_, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{SubAccountName: "sam", APIKey: "", APIKeyPermission: "trade"})
	require.ErrorIs(t, err, errInvalidAPIKey)
	_, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{IP: "1.2.3.", SubAccountName: "sam", APIKeyPermission: "trade", APIKey: "sample-api-key"})
	require.ErrorIs(t, err, errInvalidIPAddress)
	_, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{APIKeyPermission: "abc", APIKey: "sample-api-key", SubAccountName: "sam"})
	require.ErrorIs(t, err, errInvalidAPIKeyPermission)
	_, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		Permissions: []string{"abc"}, SubAccountName: "sam",
		APIKey: "sample-api-key",
	})
	require.ErrorIs(t, err, errInvalidAPIKeyPermission)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		SubAccountName:   "sam",
		APIKey:           apiKey,
		APIKeyPermission: "trade",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	result, err = e.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		SubAccountName: "sam",
		APIKey:         apiKey,
		Permissions:    []string{"trade", "read"},
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubaccountTradingBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountTradingBalance(contextGenerate(), "")
	assert.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubaccountTradingBalance(contextGenerate(), "test1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountFundingBalance(contextGenerate(), "", currency.EMPTYCODE)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubaccountFundingBalance(contextGenerate(), "test1", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSubAccountMaximumWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccountMaximumWithdrawal(contextGenerate(), "", currency.BTC)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountMaximumWithdrawal(contextGenerate(), "test1", currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.HistoryOfSubaccountTransfer(contextGenerate(), currency.EMPTYCODE, "0", "", time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoryOfManagedSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHistoryOfManagedSubAccountTransfer(contextGenerate(), currency.BTC, "", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), &SubAccountAssetTransferParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SubAccountAssetTransferParams{LoanTransfer: true}
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.BTC
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1234
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidSubaccount)

	arg.From = 1
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidSubaccount)

	arg.To = 7
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidSubaccount)

	arg.To = 6
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.FromSubAccount = "sami"
	_, err = e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), &SubAccountAssetTransferParams{Currency: currency.BTC, Amount: 1200, From: 6, To: 6, FromSubAccount: "test1", ToSubAccount: "test2", LoanTransfer: true})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	_, err := e.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{CanTransOut: true})
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetPermissionOfTransferOut(contextGenerate(), &PermissionOfTransfer{SubAcct: "Test1"})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCustodyTradingSubaccountList(contextGenerate(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetSubAccountVIPLoanAllocation(t *testing.T) {
	t.Parallel()
	_, err := e.SetSubAccountVIPLoanAllocation(contextGenerate(), &SubAccountLoanAllocationParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := subAccountVIPLoanAllocationInfo{}
	_, err = e.SetSubAccountVIPLoanAllocation(contextGenerate(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.LoanAlloc = 123
	_, err = e.SetSubAccountVIPLoanAllocation(contextGenerate(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, errInvalidSubAccountName)

	arg.LoanAlloc = -1
	arg.SubAcct = "sams"
	_, err = e.SetSubAccountVIPLoanAllocation(contextGenerate(), &SubAccountLoanAllocationParam{Alloc: []subAccountVIPLoanAllocationInfo{arg}})
	require.ErrorIs(t, err, errInvalidLoanAllocationValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetSubAccountVIPLoanAllocation(contextGenerate(), &SubAccountLoanAllocationParam{
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
	_, err := e.GetSubAccountBorrowInterestAndLimit(contextGenerate(), "", currency.ETH)
	require.ErrorIs(t, err, errInvalidSubAccountName)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSubAccountBorrowInterestAndLimit(contextGenerate(), "123456", currency.ETH)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ETH Staking

func TestGetProductInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetProductInfo(contextGenerate())
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPurcahseETHStaking(t *testing.T) {
	t.Parallel()
	err := e.PurchaseETHStaking(contextGenerate(), 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.PurchaseETHStaking(contextGenerate(), 100)
	assert.NoError(t, err)
}

func TestRedeemETHStaking(t *testing.T) {
	t.Parallel()
	err := e.RedeemETHStaking(contextGenerate(), 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.RedeemETHStaking(contextGenerate(), 100)
	assert.NoError(t, err)
}

func TestGetBETHAssetsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBETHAssetsBalance(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPurchaseAndRedeemHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPurchaseAndRedeemHistory(contextGenerate(), "", "pending", time.Time{}, time.Now(), 10)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetPurchaseAndRedeemHistory(contextGenerate(), "purchase", "pending", time.Time{}, time.Now(), 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAPYHistory(t *testing.T) {
	t.Parallel()
	result, err := e.GetAPYHistory(contextGenerate(), 34)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const gridTradingPlaceOrder = `{"instId": "BTC-USD-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input GridAlgoOrder
	err := json.Unmarshal([]byte(gridTradingPlaceOrder), &input)
	require.NoError(t, err)

	_, err = e.PlaceGridAlgoOrder(contextGenerate(), &GridAlgoOrder{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &GridAlgoOrder{BasePosition: true}
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	arg.AlgoOrdType = "contract_grid"
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.MaxPrice = 1000
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.MinPrice = 1200
	arg.GridQuantity = -1
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidGridQuantity)

	arg.GridQuantity = 123
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)

	arg.Size = 123
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingRequiredArgumentDirection)

	arg.Direction = positionSideLong
	_, err = e.PlaceGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidLeverage)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceGridAlgoOrder(contextGenerate(), &input)
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
	_, err = e.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.TakeProfitTriggerPrice = 1234.5
	_, err = e.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "560472804207104000"
	_, err = e.AmendGridAlgoOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendGridAlgoOrder(contextGenerate(), input)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const stopGridAlgoOrderJSON = `{"algoId":"198273485",	"instId":"BTC-USDT",	"stopType":"1",	"algoOrdType":"grid"}`

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var resp StopGridAlgoOrderRequest
	err := json.Unmarshal([]byte(stopGridAlgoOrderJSON), &resp)
	require.NoError(t, err)

	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := StopGridAlgoOrderRequest{}
	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.StopType = 20
	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "algo_id"
	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	arg.AlgoOrderType = AlgoOrdTypeGrid
	_, err = e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{arg})
	require.ErrorIs(t, err, errMissingValidStopType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{resp})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoOrdersList(contextGenerate(), "abc", "", "", "", "", "", 1)
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetGridAlgoOrdersList(contextGenerate(), "grid", "", "", "", "", "", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoOrderHistory(contextGenerate(), "abc", "", "", "", "", "", 1)
	require.ErrorIs(t, err, errMissingAlgoOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetGridAlgoOrderHistory(contextGenerate(), "contract_grid", "", "", "", "", "", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoOrderDetails(contextGenerate(), "grid", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetGridAlgoOrderDetails(contextGenerate(), "grid", "7878")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetGridAlgoSubOrders(contextGenerate(), "", "", "", "", "", "", 2)
	require.ErrorIs(t, err, errMissingAlgoOrderType)
	_, err = e.GetGridAlgoSubOrders(contextGenerate(), "grid", "", "", "", "", "", 2)
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "", "", "", "", 2)
	require.ErrorIs(t, err, errMissingSubOrderType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "live", "", "", "", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

const spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	var resp AlgoOrderPosition
	err := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp)
	require.NoError(t, err)
	_, err = e.GetGridAlgoOrderPositions(contextGenerate(), "", "")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = e.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", "448965992920907776")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	_, err := e.SpotGridWithdrawProfit(contextGenerate(), "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SpotGridWithdrawProfit(contextGenerate(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestComputeMarginBalance(t *testing.T) {
	t.Parallel()
	_, err := e.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{AlgoID: "123456", AdjustMarginBalanceType: "other"})
	require.ErrorIs(t, err, errInvalidMarginTypeAdjust)

	_, err = e.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{AdjustMarginBalanceType: "other"})
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{
		AlgoID:                  "123456",
		AdjustMarginBalanceType: "reduce",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdjustMarginBalance(t *testing.T) {
	t.Parallel()
	arg := &MarginBalanceParam{}
	_, err := e.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.Amount = 12345
	_, err = e.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, errAlgoIDRequired)

	arg.AlgoID = "1234"
	_, err = e.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidMarginTypeAdjust)

	arg.AdjustMarginBalanceType = "reduce"
	arg.Amount = 0
	_, err = e.AdjustMarginBalance(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AdjustMarginBalance(contextGenerate(), &MarginBalanceParam{
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

	_, err = e.GetGridAIParameter(contextGenerate(), "", mainPair.String(), "", "")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = e.GetGridAIParameter(contextGenerate(), "grid", "", "", "")
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetGridAIParameter(contextGenerate(), "contract_grid", mainPair.String(), "", "")
	require.ErrorIs(t, err, errMissingRequiredArgumentDirection)
	_, err = e.GetGridAIParameter(contextGenerate(), "grid", mainPair.String(), "", "12M")
	require.ErrorIs(t, err, errInvalidDuration)

	result, err := e.GetGridAIParameter(contextGenerate(), "grid", mainPair.String(), "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOffers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOffers(contextGenerate(), "", "", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPurchase(t *testing.T) {
	t.Parallel()
	_, err := e.Purchase(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = e.Purchase(contextGenerate(), &PurchaseRequestParam{Term: 2})
	require.ErrorIs(t, err, errMissingRequiredParameter)
	_, err = e.Purchase(contextGenerate(), &PurchaseRequestParam{ProductID: "1234", Term: 2, InvestData: []PurchaseInvestDataItem{{Amount: 1}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.Purchase(contextGenerate(), &PurchaseRequestParam{ProductID: "1234", Term: 2, InvestData: []PurchaseInvestDataItem{{Currency: currency.USDT}}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.Purchase(contextGenerate(), &PurchaseRequestParam{
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
	_, err := e.Redeem(contextGenerate(), &RedeemRequestParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.Redeem(contextGenerate(), &RedeemRequestParam{AllowEarlyRedeem: true})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.Redeem(contextGenerate(), &RedeemRequestParam{OrderID: "754147"})
	require.ErrorIs(t, err, errInvalidProtocolType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.Redeem(contextGenerate(), &RedeemRequestParam{
		OrderID:          "754147",
		ProtocolType:     "defi",
		AllowEarlyRedeem: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelPurchaseOrRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{ProtocolType: "defi"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{OrderID: "754147"})
	require.ErrorIs(t, err, errInvalidProtocolType)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelPurchaseOrRedemption(contextGenerate(), &CancelFundingParam{
		OrderID:      "754147",
		ProtocolType: "defi",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEarnActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEarnActiveOrders(contextGenerate(), "", "", "", currency.EMPTYCODE)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFundingOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFundingOrderHistory(contextGenerate(), "", "", currency.EMPTYCODE, time.Time{}, time.Time{}, 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	result, err := e.SystemStatusResponse(contextGenerate(), "completed")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

var instrumentTypeToAssetTypeMap = map[string]struct {
	AssetType asset.Item
	Error     error
}{
	instTypeSwap:     {AssetType: asset.PerpetualSwap},
	instTypeContract: {AssetType: asset.PerpetualSwap},
	instTypeSpot:     {AssetType: asset.Spot},
	instTypeMargin:   {AssetType: asset.Margin},
	instTypeFutures:  {AssetType: asset.Futures},
	instTypeOption:   {AssetType: asset.Options},
	"":               {AssetType: asset.Empty},
	"lol":            {AssetType: asset.Empty, Error: asset.ErrNotSupported},
}

func TestAssetTypeFromInstrumentType(t *testing.T) {
	t.Parallel()
	for k, v := range instrumentTypeToAssetTypeMap {
		assetItem, err := assetTypeFromInstrumentType(k)
		require.ErrorIs(t, err, v.Error)
		assert.Equal(t, v.AssetType, assetItem)
	}
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Options, asset.PerpetualSwap, asset.Futures, asset.Spot, asset.Spread} {
		result, err := e.FetchTradablePairs(contextGenerate(), a)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, true)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
		})
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()

	_, err := e.UpdateTicker(contextGenerate(), currency.Pair{}, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		p, err := e.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs for asset %s must not error", a)
		require.NotEmptyf(t, p, "GetAvailablePairs for asset %s must not return empty pairs", a)
		result, err := e.UpdateTicker(contextGenerate(), p[0], a)
		require.NoErrorf(t, err, "UpdateTicker for asset %s and pair %s must not error", a, p[0])
		assert.NotNilf(t, result, "UpdateTicker for asset %s and pair %s should not return nil", a, p[0])
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		err := e.UpdateTickers(contextGenerate(), a)
		require.NoErrorf(t, err, "UpdateTickers for asset %s must not error", a)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		p, err := e.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs for asset %s must not error", a)
		require.NotEmptyf(t, p, "GetAvailablePairs for asset %s must not return empty pairs", a)
		result, err := e.UpdateOrderbook(contextGenerate(), p[0], a)
		require.NoErrorf(t, err, "UpdateOrderbook for asset %s and pair %s must not error", a, p[0])
		assert.NotNilf(t, result, "UpdateOrderbook for asset %s and pair %s should not return nil", a, p[0])
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.UpdateAccountBalances(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(contextGenerate(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetRecentTrades(contextGenerate(), mainPair, asset.PerpetualSwap)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetRecentTrades(contextGenerate(), mainPair, asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var resp []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	require.NoError(t, err)

	arg := &order.Submit{
		Exchange:  e.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		ClientID:  "yeneOrder",
		AssetType: asset.Binary,
	}
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1000000000
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = mainPair
	arg.AssetType = asset.Futures
	arg.Leverage = -1
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  e.Name,
		Side:      order.Sell,
		Type:      order.Limit,
		Price:     120000,
		Amount:    1000000000,
		ClientID:  "yeneOrder",
		AssetType: asset.Spot,
	}
	result, err := e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Trigger
	arg.TriggerPrice = 11999
	arg.TriggerPriceType = order.LastPrice
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.ConditionalStop
	arg.TriggerPrice = 11999
	arg.TriggerPriceType = order.IndexPrice
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Chase
	_, err = e.SubmitOrder(contextGenerate(), arg)
	assert.ErrorIs(t, err, order.ErrUnknownTrackingMode)

	arg.TrackingMode = order.Percentage
	_, err = e.SubmitOrder(contextGenerate(), arg)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.TrackingValue = .5
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.TWAP
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.TrailingStop
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.OCO
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

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
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	cp, err := currency.NewPairFromString("BTC-USDT-230630")
	require.NoError(t, err)

	arg = &order.Submit{
		Pair:       cp,
		Exchange:   e.Name,
		Side:       order.Long,
		Type:       order.Market,
		Amount:     1,
		ClientID:   "hellomoto",
		AssetType:  asset.Futures,
		MarginType: margin.Multi,
	}
	result, err = e.SubmitOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	pair, err := currency.NewPairFromString("BTC-USDT-SWAP_BTC-USDT-250328")
	require.NoError(t, err)

	result, err = e.SubmitOrder(contextGenerate(), &order.Submit{
		Pair:       pair,
		Exchange:   e.Name,
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
	arg := &order.Cancel{
		AccountID: "1",
		AssetType: asset.Binary,
	}
	err := e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	err = e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = mainPair
	err = e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.CancelOrder(contextGenerate(), &order.Cancel{
		OrderID: "1", AccountID: "1", Pair: mainPair, AssetType: asset.Spot,
	})
	assert.NoError(t, err)

	err = e.CancelOrder(contextGenerate(), &order.Cancel{
		Type: order.OCO, OrderID: "1", AccountID: "1", Pair: mainPair, AssetType: asset.Spot,
	})
	assert.NoError(t, err)

	err = e.CancelOrder(contextGenerate(), &order.Cancel{
		OrderID: "1", AccountID: "1", Pair: spreadPair, AssetType: asset.Spread,
	})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(contextGenerate(), make([]order.Cancel, 21))
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.CancelBatchOrders(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	arg := order.Cancel{
		AccountID: "1",
		AssetType: asset.Binary,
	}
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	arg.Pair = mainPair
	arg.Type = order.Liquidation
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Trigger
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.Type = order.Limit
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orderCancellationParams := []order.Cancel{
		{
			OrderID:   "1",
			AccountID: "1",
			Pair:      mainPair,
			AssetType: asset.Spot,
		},
		{
			OrderID:   "1",
			AccountID: "1",
			Pair:      perpetualSwapPair,
			AssetType: asset.PerpetualSwap,
		},
		{
			OrderID:   "1",
			AccountID: "1",
			Type:      order.Trigger,
			Pair:      mainPair,
			AssetType: asset.Spot,
		},
	}
	result, err := e.CancelBatchOrders(contextGenerate(), orderCancellationParams)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Binary})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spread})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Futures})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spot})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil)

	arg := &order.Modify{}
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = mainPair
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Spot
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	arg.Type = order.Liquidation
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	arg.Type = order.Trigger
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Type = order.OCO
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	arg = &order.Modify{
		AssetType: asset.Spot,
		Pair:      mainPair,
		OrderID:   "1234",
		Price:     123456.44,
		Amount:    123,
	}
	result, err := e.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Limit
	result, err = e.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.Trigger
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.TriggerPrice = 12345678
	_, err = e.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	arg.Type = order.OCO
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Price: 12345677},
		StopLoss:   order.RiskManagement{Price: 12345667},
	}
	result, err = e.ModifyOrder(contextGenerate(), arg)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.ModifyOrder(contextGenerate(),
		&order.Modify{
			AssetType: asset.Spread,
			Pair:      spreadPair,
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetOrderInfo(contextGenerate(), "123", perpetualSwapPair, asset.PerpetualSwap)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetOrderInfo(contextGenerate(), "123", spreadPair, asset.Spread)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(contextGenerate(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddress(contextGenerate(), currency.BTC, core.BitcoinDonationAddress, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Exchange: e.Name,
		Amount:   -0.1,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	result, err := e.WithdrawCryptocurrencyFunds(contextGenerate(), &withdrawCryptoRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPairFromInstrumentID(t *testing.T) {
	t.Parallel()
	instruments := []string{
		mainPair.String(),
		perpetualSwapPair.String(),
		"BTC-USDT-ER33234",
	}
	dPair, err := e.GetPairFromInstrumentID(instruments[0])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = e.GetPairFromInstrumentID(instruments[1])
	require.NoError(t, err)
	require.NotNil(t, dPair)
	dPair, err = e.GetPairFromInstrumentID(instruments[2])
	require.NoError(t, err)
	assert.NotNil(t, dPair)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	for _, a := range []asset.Item{asset.Spot, asset.Spread} {
		pairs := []currency.Pair{currency.NewPair(currency.LTC, currency.USDT), mainPair}
		if a == asset.Spread {
			pairs = []currency.Pair{spreadPair}
		}

		result, err := e.GetActiveOrders(contextGenerate(), &order.MultiOrderRequest{
			Type:      order.Limit,
			Pairs:     pairs,
			AssetType: asset.Spot,
			Side:      order.Buy,
		})
		require.NoErrorf(t, err, "GetActiveOrders for asset %s and pair %s must not error", a, pairs[0])
		assert.NotNil(t, result)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err := e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	getOrdersRequest.Pairs = []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)}
	result, err := e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	getOrdersRequest.AssetType = asset.Spread
	getOrdersRequest.Type = order.Market
	result, err = e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFeeByType(contextGenerate(), &exchange.FeeBuilder{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.ValidateAPICredentials(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	_, err := e.GetHistoricCandles(contextGenerate(), currency.Pair{}, asset.Binary, kline.OneDay, time.Now(), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)

	startTime := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 100)
	_, err = e.GetHistoricCandles(contextGenerate(), mainPair, asset.Spot, kline.Interval(time.Hour*4), startTime, endTime)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs for asset %s must not error", a)
		require.NotEmptyf(t, pairs, "GetEnabledPairs for asset %s must not return empty pairs", a)
		result, err := e.GetHistoricCandles(contextGenerate(), pairs[0], a, kline.OneMin, time.Now().Add(-time.Hour), time.Now())
		if (a == asset.Spread || a == asset.Options) && err != nil { // Options and spread candles sometimes returns no data
			continue
		}
		require.NoErrorf(t, err, "GetHistoricCandles for asset %s and pair %s must not error", a, pairs[0])
		assert.NotNilf(t, result, "GetHistoricCandles for asset %s and pair %s should not return nil", a, pairs[0])
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	result, err := e.GetHistoricCandlesExtended(contextGenerate(), mainPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	var orderbookBase orderbook.Book
	err := json.Unmarshal([]byte(calculateOrderbookChecksumUpdateOrderbookJSON), &orderbookBase)
	require.NoError(t, err)
	require.Equal(t, uint32(2832680552), generateOrderbookChecksum(&orderbookBase))
}

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrders.json", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, nil, b) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 4, "Should see 4 orders")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case *order.Detail:
			switch len(e.Websocket.DataHandler.C) {
			case 3:
				assert.Equal(t, "452197707845865472", v.OrderID, "OrderID")
				assert.Equal(t, "HamsterParty14", v.ClientOrderID, "ClientOrderID")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, order.Filled, v.Status, "Status")
				assert.Equal(t, order.Limit, v.Type, "Type")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "Pair")
				assert.Equal(t, 31527.1, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, time.UnixMilli(1654084334977), v.Date, "Date")
				assert.Equal(t, time.UnixMilli(1654084353263), v.CloseTime, "CloseTime")
				assert.Equal(t, 0.001, v.Amount, "Amount")
				assert.Equal(t, 0.001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.000, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 31527.1, v.Price, "Price")
				assert.Equal(t, 0.02522168, v.Fee, "Fee")
				assert.Equal(t, currency.USDT, v.FeeAsset, "FeeAsset")
			case 2:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, order.Market, v.Type, "Type")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, order.Active, v.Status, "Status")
				assert.Equal(t, 0.0, v.Amount, "Amount should be 0 for a market sell")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
			case 1:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				assert.Equal(t, 0.00038127046945832905, v.Amount, "Amount")
				assert.Equal(t, 0.010000249968, v.Fee, "Fee")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, order.PartiallyFilled, v.Status, "Status")
			case 0:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				assert.Equal(t, 0.010000249968, v.Fee, "Fee")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.00038128, v.Amount, "Amount should be derived because order filled")
				assert.Equal(t, order.Filled, v.Status, "Status")
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
	"RFQs":                                  `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
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
	"Balance Save Error":                    `{"arg": {"channel": "balance_and_position","uid": "77982378738415880"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`,
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	for name, msg := range pushDataMap {
		switch name {
		case "Balance And Position":
			e.API.AuthenticatedSupport = true
			e.API.AuthenticatedWebsocketSupport = true
			e.SetCredentials("test", "test", "test", "", "", "")
		default:
			e.API.AuthenticatedSupport = false
			e.API.AuthenticatedWebsocketSupport = false
		}
		err := e.wsHandleData(t.Context(), nil, []byte(msg))
		if name == "Balance Save Error" {
			assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "wsProcessBalanceAndPosition Accounts.Save should error without credentials")
		} else {
			require.NoErrorf(t, err, "%s must not error", name)
		}
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
		err = e.wsHandleData(t.Context(), nil, []byte(dataMap[x]))
		require.NoError(t, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(contextGenerate(), mainPair, asset.Spread, time.Now(), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)

	result, err := e.GetHistoricTrades(contextGenerate(), mainPair, asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWSProcessTrades(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	assets, err := e.getAssetsFromInstrumentID(mainPair.String())
	require.NoError(t, err, "getAssetsFromInstrumentID must not error")

	p := currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter)

	for _, a := range assets {
		err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{
			Asset:   a,
			Pairs:   currency.Pairs{p},
			Channel: subscription.AllTradesChannel,
			Key:     fmt.Sprintf("%s-%s", p, a),
		})
		require.NoError(t, err, "AddSubscriptions must not error")
	}
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, nil, b) })

	exp := []trade.Data{
		{
			Timestamp: time.UnixMilli(1740394561685).UTC(),
			Price:     95634.9,
			Amount:    0.00011186,
			Side:      order.Buy,
			TID:       "674510826",
		},
		{
			Timestamp: time.UnixMilli(1740394561686).UTC(),
			Price:     95635.3,
			Amount:    0.00011194,
			Side:      order.Sell,
			TID:       "674510827",
		},
	}

	total := len(assets) * len(exp)
	require.Len(t, e.Websocket.DataHandler.C, total, "Must see correct number of trades")

	trades := make(map[asset.Item][]trade.Data)

	for len(e.Websocket.DataHandler.C) > 0 {
		resp := <-e.Websocket.DataHandler.C
		switch v := resp.Data.(type) {
		case trade.Data:
			trades[v.AssetType] = append(trades[v.AssetType], v)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}

	for _, assetType := range assets {
		require.Lenf(t, trades[assetType], len(exp), "Must have received %d trades for asset %v", len(exp), assetType)
		slices.SortFunc(trades[assetType], func(a, b trade.Data) int {
			return strings.Compare(a.TID, b.TID)
		})
		for i, tradeData := range trades[assetType] {
			expected := exp[i]
			expected.AssetType = assetType
			expected.Exchange = e.Name
			expected.CurrencyPair = p
			require.Equalf(t, expected, tradeData, "Trade %d (TID: %s) for asset %v must match expected data", i, tradeData.TID, assetType)
		}
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(contextGenerate(), asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAvailableTransferChains(contextGenerate(), currency.BTC)
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
		{Expected: "", AppendUTC: true},
		{Description: "kline.OneMin ", Interval: kline.OneMin, Expected: "1m"},
		{Description: "kline.ThreeMin ", Interval: kline.ThreeMin, Expected: "3m"},
		{Description: "kline.FiveMin ", Interval: kline.FiveMin, Expected: "5m"},
		{Description: "kline.FifteenMin ", Interval: kline.FifteenMin, Expected: "15m"},
		{Description: "kline.ThirtyMin ", Interval: kline.ThirtyMin, Expected: "30m"},
		{Description: "kline.OneHour ", Interval: kline.OneHour, Expected: "1H"},
		{Description: "kline.TwoHour ", Interval: kline.TwoHour, Expected: "2H"},
		{Description: "kline.FourHour ", Interval: kline.FourHour, Expected: "4H"},
		{Description: "kline.SixHour ", Interval: kline.SixHour, Expected: "6H"},
		{Description: "kline.TwelveHour ", Interval: kline.TwelveHour, Expected: "12H"},
		{Description: "kline.OneDay ", Interval: kline.OneDay, Expected: "1D"},
		{Description: "kline.TwoDay ", Interval: kline.TwoDay, Expected: "2D"},
		{Description: "kline.ThreeDay ", Interval: kline.ThreeDay, Expected: "3D"},
		{Description: "kline.OneWeek ", Interval: kline.OneWeek, Expected: "1W"},
		{Description: "kline.FiveDay ", Interval: kline.FiveDay, Expected: "5D"},
		{Description: "kline.OneMonth ", Interval: kline.OneMonth, Expected: "1M"},
		{Description: "kline.ThreeMonth ", Interval: kline.ThreeMonth, Expected: "3M"},
		{Description: "kline.SixMonth ", Interval: kline.SixMonth, Expected: "6M"},
		{Description: "kline.OneYear ", Interval: kline.OneYear, Expected: "1Y"},
		{Description: "kline.SixHour + UTC", Interval: kline.SixHour, Expected: "6Hutc", AppendUTC: true},
		{Description: "kline.TwelveHour + UTC ", Interval: kline.TwelveHour, Expected: "12Hutc", AppendUTC: true},
		{Description: "kline.OneDay + UTC ", Interval: kline.OneDay, Expected: "1Dutc", AppendUTC: true},
		{Description: "kline.TwoDay + UTC ", Interval: kline.TwoDay, Expected: "2Dutc", AppendUTC: true},
		{Description: "kline.ThreeDay + UTC ", Interval: kline.ThreeDay, Expected: "3Dutc", AppendUTC: true},
		{Description: "kline.FiveDay + UTC ", Interval: kline.FiveDay, Expected: "5Dutc", AppendUTC: true},
		{Description: "kline.OneWeek + UTC ", Interval: kline.OneWeek, Expected: "1Wutc", AppendUTC: true},
		{Description: "kline.OneMonth + UTC ", Interval: kline.OneMonth, Expected: "1Mutc", AppendUTC: true},
		{Description: "kline.ThreeMonth + UTC ", Interval: kline.ThreeMonth, Expected: "3Mutc", AppendUTC: true},
		{Description: "kline.SixMonth + UTC ", Interval: kline.SixMonth, Expected: "6Mutc", AppendUTC: true},
		{Description: "kline.OneYear + UTC ", Interval: kline.OneYear, Expected: "1Yutc", AppendUTC: true},
	}

	for _, tt := range tests {
		t.Run(tt.Description, func(t *testing.T) {
			t.Parallel()
			r := IntervalFromString(tt.Interval, tt.AppendUTC)
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
	assert.Empty(t, i.BaseCurrency, "expected empty base currency")
	assert.Equal(t, "1", i.Category, "expected 1 category")
	assert.Equal(t, 1, int(i.ContractMultiplier.Int64()), "expected 1 contract multiplier")
	assert.Equal(t, "linear", i.ContractType, "expected linear contract type")
	assert.Equal(t, 0.0001, i.ContractValue.Float64(), "expected 0.0001 contract value")
	assert.Equal(t, currency.BTC.String(), i.ContractValueCurrency, "expected BTC contract value currency")
	assert.True(t, i.ExpTime.Time().IsZero(), "expected empty expiry time")
	assert.Equal(t, "BTC-USDC", i.InstrumentFamily, "expected BTC-USDC instrument family")
	assert.Equal(t, "BTC-USDC-SWAP", i.InstrumentID.String(), "expected BTC-USDC-SWAP instrument ID")

	swap := GetInstrumentTypeFromAssetItem(asset.PerpetualSwap)
	assert.Equal(t, swap, i.InstrumentType, "expected SWAP instrument type")
	assert.Equal(t, 125, int(i.MaxLeverage), "expected 125 leverage")
	assert.Equal(t, int64(1666076190000), i.ListTime.Time().UnixMilli(), "expected 1666076190000 listing time")
	assert.Equal(t, 1, int(i.LotSize))
	assert.Equal(t, 100000000.0000000000000000, i.MaxSpotIcebergSize.Float64())
	assert.Equal(t, 100000000, int(i.MaxQuantityOfSpotLimitOrder))
	assert.Equal(t, 85000, int(i.MaxQuantityOfMarketLimitOrder))
	assert.Equal(t, 85000, int(i.MaxStopSize))
	assert.Equal(t, 100000000.0000000000000000, i.MaxTriggerSize.Float64())
	assert.Equal(t, 0, int(i.MaxQuantityOfSpotTwapLimitOrder), "expected empty max TWAP size")
	assert.Equal(t, 1, int(i.MinimumOrderSize))
	assert.Empty(t, i.OptionType, "expected empty option type")
	assert.Empty(t, i.QuoteCurrency, "expected empty quote currency")
	assert.Equal(t, currency.USDC.String(), i.SettlementCurrency, "expected USDC settlement currency")
	assert.Equal(t, "live", i.State)
	assert.Empty(t, i.StrikePrice, "expected empty strike price")
	assert.Equal(t, 0.1, i.TickSize.Float64())
	assert.Equal(t, "BTC-USDC", i.Underlying, "expected BTC-USDC underlying")
}

func TestGetLatestFundingRate(t *testing.T) {
	t.Parallel()
	result, err := e.GetLatestFundingRates(contextGenerate(), &fundingrate.LatestRateRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 perpetualSwapPair,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	r := &fundingrate.HistoricalRatesRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 perpetualSwapPair,
		PaymentCurrency:      currency.USDT,
		StartDate:            time.Now().Add(-time.Hour * 24 * 2),
		EndDate:              time.Now(),
		IncludePredictedRate: true,
	}

	r.StartDate = time.Now().Add(-time.Hour * 24 * 120)
	_, err := e.GetHistoricalFundingRates(contextGenerate(), r)
	require.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits)

	if sharedtestvalues.AreAPICredentialsSet(e) {
		r.IncludePayments = true
	}
	r.StartDate = time.Now().Add(-time.Hour * 24 * 12)
	result, err := e.GetHistoricalFundingRates(contextGenerate(), r)
	require.NoError(t, err)
	require.NotNil(t, result)

	r.RespectHistoryLimits = true
	result, err = e.GetHistoricalFundingRates(contextGenerate(), r)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, mainPair)
	require.NoError(t, err)
	require.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.PerpetualSwap, perpetualSwapPair)
	require.NoError(t, err)
	assert.True(t, is, "expected true")
}

func TestGetAssetsFromInstrumentTypeOrID(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	_, err := e.getAssetsFromInstrumentID("")
	assert.ErrorIs(t, err, errMissingInstrumentID)

	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options} {
		assets, err2 := e.getAssetsFromInstrumentID(e.CurrencyPairs.Pairs[a].Enabled[0].String())
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		switch a {
		case asset.Spot, asset.Margin:
			// spot and margin instruments are similar
			require.Len(t, assets, 2)
		default:
			require.Len(t, assets, 1)
		}
		assert.Containsf(t, assets, a, "Should contain asset: %s", a)
	}

	_, err = e.getAssetsFromInstrumentID("test")
	assert.ErrorIs(t, err, currency.ErrCurrencyNotSupported)
	_, err = e.getAssetsFromInstrumentID("test-test")
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	for _, a := range []asset.Item{asset.Margin, asset.Spot} {
		assets, err2 := e.getAssetsFromInstrumentID(e.CurrencyPairs.Pairs[a].Enabled[0].String())
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		assert.Contains(t, assets, a)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := e.SetMarginType(contextGenerate(), asset.Spot, mainPair, margin.Isolated)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ChangePositionMargin(contextGenerate(), &margin.PositionChangeRequest{
		Pair:                    mainPair,
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
	_, err := e.GetCollateralMode(contextGenerate(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCollateralMode(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralMode(contextGenerate(), asset.Spot, collateral.SingleMode)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	pp, err := e.CurrencyPairs.GetPairs(asset.PerpetualSwap, true)
	require.NoError(t, err)
	result, err := e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.PerpetualSwap,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	pp, err = e.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           pp[0],
		UnderlyingPair: mainPair,
	})
	require.ErrorIsf(t, err, futures.ErrNotFuturesAsset, "received '%v', expected '%v'", err, futures.ErrNotFuturesAsset)

	result, err = e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Futures,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	pp, err := e.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = e.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Spot,
		Pairs:     []currency.Pair{pp[0]},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
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
	pp, err := e.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	_, err = e.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.UnknownSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, order.UnknownSide)
	require.NoError(t, err)
	require.NotNil(t, result)
	result, err = e.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Long)
	require.NoError(t, err)
	require.NotNil(t, result)

	result, err = e.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Short)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	pp, err := e.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.UnknownSide)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.CouldNotBuy)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	err = e.SetLeverage(contextGenerate(), asset.Spot, pp[0], margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Long)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Short)
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(contextGenerate(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = e.GetFuturesContractDetails(contextGenerate(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap, asset.Spread} {
		result, err := e.GetFuturesContractDetails(contextGenerate(), a)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
}

func TestWsProcessOrderbook5(t *testing.T) {
	t.Parallel()
	ob5payload := []byte(`{"arg":{"channel":"books5","instId":"OKB-USDT"},"data":[{"asks":[["0.0000007465","2290075956","0","4"],["0.0000007466","1747284705","0","4"],["0.0000007467","1338861655","0","3"],["0.0000007468","1661668387","0","6"],["0.0000007469","2715477116","0","5"]],"bids":[["0.0000007464","15693119","0","1"],["0.0000007463","2330835024","0","4"],["0.0000007462","1182926517","0","2"],["0.0000007461","3818684357","0","4"],["0.000000746","6021641435","0","7"]],"instId":"OKB-USDT","ts":"1695864901807","seqId":4826378794}]}`)
	err := e.wsProcessOrderbook5(ob5payload)
	require.NoError(t, err)

	required := currency.NewPairWithDelimiter("OKB", "USDT", "-")
	got, err := orderbook.Get(e.Name, required, asset.Spot)
	require.NoError(t, err)

	require.Len(t, got.Asks, 5)
	require.Len(t, got.Bids, 5)
	// Book replicated to margin
	got, err = orderbook.Get(e.Name, required, asset.Margin)
	require.NoError(t, err)
	require.Len(t, got.Asks, 5)
	assert.Len(t, got.Bids, 5)
}

func TestGetLeverateEstimatedInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverageEstimatedInfo(contextGenerate(), "", "cross", "1", "", mainPair.String(), currency.BTC)
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetLeverageEstimatedInfo(contextGenerate(), "MARGIN", "", "1", "", mainPair.String(), currency.BTC)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.GetLeverageEstimatedInfo(contextGenerate(), "MARGIN", "cross", "", "", mainPair.String(), currency.BTC)
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = e.GetLeverageEstimatedInfo(contextGenerate(), "MARGIN", "cross", "1", "", mainPair.String(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeverageEstimatedInfo(contextGenerate(), "MARGIN", "cross", "1", "", mainPair.String(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestManualBorrowAndRepayInQuickMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{
		InstrumentID: mainPair.String(),
		LoanCcy:      currency.USDT,
		Side:         "borrow",
	})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: mainPair.String(),
		Side:         "borrow",
	})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: mainPair.String(),
		LoanCcy:      currency.USDT,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{
		Amount:  1,
		LoanCcy: currency.USDT,
		Side:    "borrow",
	})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ManualBorrowAndRepayInQuickMarginMode(contextGenerate(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: mainPair.String(),
		LoanCcy:      currency.USDT,
		Side:         "borrow",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetBorrowAndRepayHistoryInQuickMarginMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetBorrowAndRepayHistoryInQuickMarginMode(contextGenerate(), currency.EMPTYPAIR, currency.BTC, "borrow", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPInterestAccruedData(contextGenerate(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPInterestDeductedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPInterestDeductedData(contextGenerate(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanOrderList(contextGenerate(), "", "1", currency.BTC, time.Time{}, time.Now(), 20)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetVIPLoanOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetVIPLoanOrderDetail(contextGenerate(), "123456", currency.BTC, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetRiskOffsetType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetRiskOffsetType(contextGenerate(), "3")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestActivateOption(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.ActivateOption(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAutoLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetAutoLoan(contextGenerate(), true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetAccountMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetAccountMode(contextGenerate(), "1")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ResetMMPStatus(contextGenerate(), instTypeOption, "")
	require.ErrorIs(t, err, errInstrumentFamilyRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ResetMMPStatus(contextGenerate(), instTypeOption, "BTC-USD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	_, err := e.SetMMP(contextGenerate(), &MMPConfig{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetMMP(contextGenerate(), &MMPConfig{
		TimeInterval: 5000,
	})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.SetMMP(contextGenerate(), &MMPConfig{
		InstrumentFamily: "BTC-USD",
	})
	require.ErrorIs(t, err, errInvalidQuantityLimit)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetMMP(contextGenerate(), &MMPConfig{
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMMPConfig(contextGenerate(), "BTC-USD")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMassCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllMMPOrders(contextGenerate(), "", "BTC-USD", 2000)
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.CancelAllMMPOrders(contextGenerate(), "OPTION", "", 2000)
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.CancelAllMMPOrders(contextGenerate(), "OPTION", "BTC-USD", -1)
	require.ErrorIs(t, err, errMissingIntervalValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllMMPOrders(contextGenerate(), "OPTION", "BTC-USD", 2000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllDelayed(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllDelayed(contextGenerate(), 2, "")
	require.ErrorIs(t, err, errCountdownTimeoutRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllDelayed(contextGenerate(), 60, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTradeAccountRateLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTradeAccountRateLimit(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPreCheckOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PreCheckOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderPreCheckParams{
		ClientOrderID: "b15",
	}
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cash"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PreCheckOrder(contextGenerate(), &OrderPreCheckParams{
		InstrumentID:  mainPair.String(),
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
	_, err := e.AmendAlgoOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderParam{NewSize: 2})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderParam{
		InstrumentID: perpetualSwapPair.String(),
		NewSize:      2,
	})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendAlgoOrder(contextGenerate(), &AmendAlgoOrderParam{
		AlgoID:       "2510789768709120",
		InstrumentID: perpetualSwapPair.String(),
		NewSize:      2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetAlgoOrderDetail(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAlgoOrderDetail(contextGenerate(), "1234231231423", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClosePositionForContractID(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePositionForContractID(contextGenerate(), &ClosePositionParams{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.ClosePositionForContractID(contextGenerate(), &ClosePositionParams{AlgoID: "", MarketCloseAllPositions: true})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.ClosePositionForContractID(contextGenerate(), &ClosePositionParams{AlgoID: "448965992920907776", MarketCloseAllPositions: false})
	require.ErrorIs(t, err, order.ErrAmountMustBeSet)
	_, err = e.ClosePositionForContractID(contextGenerate(), &ClosePositionParams{AlgoID: "448965992920907776", MarketCloseAllPositions: false, Size: 123})
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.ClosePositionForContractID(contextGenerate(), &ClosePositionParams{
		AlgoID:                  "448965992920907776",
		MarketCloseAllPositions: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelClosePositionOrderForContractGrid(t *testing.T) {
	t.Parallel()
	_, err := e.CancelClosePositionOrderForContractGrid(contextGenerate(), &CancelClosePositionOrder{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CancelClosePositionOrderForContractGrid(contextGenerate(), &CancelClosePositionOrder{OrderID: "570627699870375936"})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.CancelClosePositionOrderForContractGrid(contextGenerate(), &CancelClosePositionOrder{AlgoID: "448965992920907776"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelClosePositionOrderForContractGrid(contextGenerate(), &CancelClosePositionOrder{
		AlgoID:  "448965992920907776",
		OrderID: "570627699870375936",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInstantTriggerGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.InstantTriggerGridAlgoOrder(contextGenerate(), "123456789")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestComputeMinInvestment(t *testing.T) {
	t.Parallel()
	arg := &ComputeInvestmentDataParam{
		RunType: "1",
	}
	_, err := e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)
	arg.InstrumentID = mainPair.String()
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	arg.AlgoOrderType = "grid"
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.MaxPrice = 5000
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.MinPrice = 5000
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidGridQuantity)

	arg.GridNumber = 1234
	arg.RunType = ""
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errRunTypeRequired)

	arg.RunType = "1"
	arg.AlgoOrderType = "contract_grid"
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingRequiredArgumentDirection)

	arg.Direction = positionSideLong
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidLeverage)

	arg.Leverage = 5
	arg.InvestmentData = []InvestmentData{{Currency: currency.ETH}}
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.InvestmentData = []InvestmentData{{Amount: 0.01}}
	_, err = e.ComputeMinInvestment(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	result, err := e.ComputeMinInvestment(contextGenerate(), &ComputeInvestmentDataParam{
		InstrumentID:  mainPair.String(),
		AlgoOrderType: "grid",
		GridNumber:    50,
		MaxPrice:      5000,
		MinPrice:      3000,
		RunType:       "1",
		InvestmentData: []InvestmentData{
			{
				Amount:   0.01,
				Currency: currency.BTC,
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
	_, err := e.RSIBackTesting(contextGenerate(), "", "", "", 50, 14, kline.FiveMin)
	require.ErrorIs(t, err, errMissingInstrumentID)
	result, err := e.RSIBackTesting(contextGenerate(), mainPair.String(), "", "", 50, 14, kline.FiveMin)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSignalBotTrading(t *testing.T) {
	t.Parallel()
	_, err := e.GetSignalBotOrderDetail(contextGenerate(), "", "623833708424069120")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = e.GetSignalBotOrderDetail(contextGenerate(), "contract", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.GetSignalBotOrderDetail(contextGenerate(), "contract", "623833708424069120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalOrderPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetSignalOrderPositions(contextGenerate(), "", "623833708424069120")
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = e.GetSignalOrderPositions(contextGenerate(), "contract", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSignalOrderPositions(contextGenerate(), "contract", "623833708424069120")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalBotSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSignalBotSubOrders(contextGenerate(), "", "contract", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.GetSignalBotSubOrders(contextGenerate(), "623833708424069120", "", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, errInvalidAlgoOrderType)
	_, err = e.GetSignalBotSubOrders(contextGenerate(), "623833708424069120", "contract", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSignalBotSubOrders(contextGenerate(), "623833708424069120", "contract", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSignalBotEventHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSignalBotEventHistory(contextGenerate(), "", time.Time{}, time.Now(), 50)
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSignalBotEventHistory(contextGenerate(), "12345", time.Time{}, time.Now(), 50)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceRecurringBuyOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceRecurringBuyOrderParam{
		TimeZone: "3",
	}
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errStrategyNameRequired)

	arg.StrategyName = "BTC|ETH recurring buy monthly"
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.RecurringList = []RecurringListItem{{}}
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.RecurringList = []RecurringListItem{{Currency: currency.BTC}}
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errRecurringDayRequired)

	arg.RecurringDay = "1"
	arg.RecurringTime = -10
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errRecurringBuyTimeRequired)

	arg.RecurringTime = 2
	_, err = e.PlaceRecurringBuyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceRecurringBuyOrder(contextGenerate(), &PlaceRecurringBuyOrderParam{
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
	_, err := e.AmendRecurringBuyOrder(contextGenerate(), &AmendRecurringOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.AmendRecurringBuyOrder(contextGenerate(), &AmendRecurringOrderParam{StrategyName: "stg1"})
	require.ErrorIs(t, err, errAlgoIDRequired)
	_, err = e.AmendRecurringBuyOrder(contextGenerate(), &AmendRecurringOrderParam{AlgoID: "448965992920907776"})
	require.ErrorIs(t, err, errStrategyNameRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendRecurringBuyOrder(contextGenerate(), &AmendRecurringOrderParam{
		AlgoID:       "448965992920907776",
		StrategyName: "stg1",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestStopRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.StopRecurringBuyOrder(contextGenerate(), []StopRecurringBuyOrder{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.StopRecurringBuyOrder(contextGenerate(), []StopRecurringBuyOrder{{}})
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.StopRecurringBuyOrder(contextGenerate(), []StopRecurringBuyOrder{{AlgoID: "1232323434234"}})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringBuyOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRecurringBuyOrderList(contextGenerate(), "", "paused", time.Time{}, time.Time{}, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringBuyOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRecurringBuyOrderHistory(contextGenerate(), "", time.Time{}, time.Time{}, 30)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringOrderDetails(contextGenerate(), "", "")
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRecurringOrderDetails(contextGenerate(), "560473220642766848", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecurringSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecurringSubOrders(contextGenerate(), "", "123422", time.Time{}, time.Now(), 0)
	require.ErrorIs(t, err, errAlgoIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetRecurringSubOrders(contextGenerate(), "560473220642766848", "123422", time.Time{}, time.Now(), 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetExistingLeadingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetExistingLeadingPositions(contextGenerate(), instTypeSpot, mainPair.String(), time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadingPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeadingPositionsHistory(contextGenerate(), "OPTION", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceLeadingStopOrder(t *testing.T) {
	t.Parallel()
	arg := &TPSLOrderParam{}
	_, err := e.PlaceLeadingStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg.Tag = "1235454"
	_, err = e.PlaceLeadingStopOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errSubPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceLeadingStopOrder(contextGenerate(), &TPSLOrderParam{
		SubPositionID:          "1235454",
		TakeProfitTriggerPrice: 123455,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCloseLeadingPosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseLeadingPosition(contextGenerate(), &CloseLeadingPositionParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.CloseLeadingPosition(contextGenerate(), &CloseLeadingPositionParam{Tag: "tag-here"})
	require.ErrorIs(t, err, errSubPositionIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CloseLeadingPosition(contextGenerate(), &CloseLeadingPositionParam{
		SubPositionID: "518541406042591232",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadingInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLeadingInstrument(contextGenerate(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendLeadingInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.AmendLeadingInstruments(contextGenerate(), "", "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendLeadingInstruments(contextGenerate(), perpetualSwapPair.String(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetProfitSharingDetails(contextGenerate(), "", time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTotalProfitSharing(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetTotalProfitSharing(contextGenerate(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUnrealizedProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUnrealizedProfitSharingDetails(contextGenerate(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetFirstCopySettings(t *testing.T) {
	t.Parallel()
	_, err := e.AmendCopySettings(contextGenerate(), &FirstCopySettings{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendCopySettings(contextGenerate(), &FirstCopySettings{
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
	_, err := e.SetFirstCopySettings(contextGenerate(), &FirstCopySettings{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &FirstCopySettings{
		CopyMode: "ratio_copy",
	}
	_, err = e.SetFirstCopySettings(contextGenerate(), arg)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	arg.UniqueCode = "25CD5A80241D6FE6"
	_, err = e.SetFirstCopySettings(contextGenerate(), arg)
	require.ErrorIs(t, err, errCopyInstrumentIDTypeRequired)

	arg.CopyInstrumentIDType = "copy"
	_, err = e.SetFirstCopySettings(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.CopyTotalAmount = 500
	_, err = e.SetFirstCopySettings(contextGenerate(), arg)
	require.ErrorIs(t, err, errSubPositionCloseTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.SetFirstCopySettings(contextGenerate(), &FirstCopySettings{
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
	_, err := e.StopCopying(contextGenerate(), &StopCopyingParameter{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	_, err = e.StopCopying(contextGenerate(), &StopCopyingParameter{
		InstrumentType:       "SWAP",
		SubPositionCloseType: "manual_close",
	})
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.StopCopying(contextGenerate(), &StopCopyingParameter{
		InstrumentType: "SWAP",
		UniqueCode:     "25CD5A80241D6FE6",
	})
	require.ErrorIs(t, err, errSubPositionCloseTypeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.StopCopying(contextGenerate(), &StopCopyingParameter{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		SubPositionCloseType: "manual_close",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCopySettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopySettings(contextGenerate(), "SWAP", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCopySettings(contextGenerate(), "SWAP", "213E8C92DC61EFAC")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMultipleLeverages(t *testing.T) {
	t.Parallel()
	_, err := e.GetMultipleLeverages(contextGenerate(), "", "213E8C92DC61EFAC", "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.GetMultipleLeverages(contextGenerate(), "isolated", "", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMultipleLeverages(contextGenerate(), "isolated", "213E8C92DC61EFAC", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetMultipleLeverages(t *testing.T) {
	t.Parallel()
	_, err := e.SetMultipleLeverages(contextGenerate(), &SetLeveragesParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.SetMultipleLeverages(contextGenerate(), &SetLeveragesParam{Leverage: 5})
	require.ErrorIs(t, err, margin.ErrInvalidMarginType)
	_, err = e.SetMultipleLeverages(contextGenerate(), &SetLeveragesParam{MarginMode: "cross"})
	require.ErrorIs(t, err, errInvalidLeverage)
	_, err = e.SetMultipleLeverages(contextGenerate(), &SetLeveragesParam{
		MarginMode: "cross",
		Leverage:   5,
	})
	require.ErrorIs(t, err, errMissingInstrumentID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.SetMultipleLeverages(contextGenerate(), &SetLeveragesParam{
		MarginMode:   "cross",
		Leverage:     5,
		InstrumentID: mainPair.String(),
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetMyLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetMyLeadTraders(contextGenerate(), "SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoryLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetHistoryLeadTraders(contextGenerate(), "", "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWeeklyTraderProfitAndLoss(t *testing.T) {
	t.Parallel()
	_, err := e.GetWeeklyTraderProfitAndLoss(contextGenerate(), "", "")
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	mainResult, err := e.GetWeeklyTraderProfitAndLoss(contextGenerate(), "", leadTraderUniqueID)
	require.NoError(t, err)
	assert.NotNil(t, mainResult)
}

func TestGetDailyLeadTraderPNL(t *testing.T) {
	t.Parallel()
	_, err := e.GetDailyLeadTraderPNL(contextGenerate(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.GetDailyLeadTraderPNL(contextGenerate(), "SWAP", "WOOF", "")
	require.ErrorIs(t, err, errLastDaysRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	mainResult, err := e.GetDailyLeadTraderPNL(contextGenerate(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, mainResult)
}

func TestGetLeadTraderStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderStats(contextGenerate(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.GetLeadTraderStats(contextGenerate(), "SWAP", "RAWR", "")
	require.ErrorIs(t, err, errLastDaysRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderStats(contextGenerate(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadTraderCurrencyPreferences(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderCurrencyPreferences(contextGenerate(), "SWAP", "", "2")
	require.ErrorIs(t, err, errUniqueCodeRequired)
	_, err = e.GetLeadTraderCurrencyPreferences(contextGenerate(), "SWAP", "MEOW", "")
	require.ErrorIs(t, err, errLastDaysRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderCurrencyPreferences(contextGenerate(), "SWAP", leadTraderUniqueID, "2")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLeadTraderCurrentLeadPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderCurrentLeadPositions(contextGenerate(), instTypeSwap, "", "", "", 10)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	_, err = e.GetLeadTraderCurrentLeadPositions(contextGenerate(), "SWAP", leadTraderUniqueID, "", "", 10)
	require.NoError(t, err)
	// No test validation of positions performed as the lead trader may not have any positions open
}

func TestGetLeadTraderLeadPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeadTraderLeadPositionHistory(contextGenerate(), "SWAP", "", "", "", 10)
	require.ErrorIs(t, err, errUniqueCodeRequired)

	require.NoError(t, syncLeadTraderUniqueID(t), "syncLeadTraderUniqueID must not error")
	result, err := e.GetLeadTraderLeadPositionHistory(contextGenerate(), "SWAP", leadTraderUniqueID, "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpreadOrder(contextGenerate(), &SpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &SpreadOrderParam{Tag: "tag-here"}
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.SpreadID = spreadPair.String()
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Size = 1
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Price = 12345
	_, err = e.PlaceSpreadOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceSpreadOrder(contextGenerate(), &SpreadOrderParam{
		InstrumentID:  spreadPair.String(),
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
	_, err := e.CancelSpreadOrder(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelSpreadOrder(request.WithVerbose(contextGenerate()), "12345", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllSpreadOrders(contextGenerate(), "123456")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	_, err = e.AmendSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{NewSize: 2})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.AmendSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{OrderID: "2510789768709120"})
	require.ErrorIs(t, err, errSizeOrPriceIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendSpreadOrder(contextGenerate(), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadOrderDetails(contextGenerate(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpreadOrderDetails(contextGenerate(), "1234567", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetActiveSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetActiveSpreadOrders(contextGenerate(), "", "post_only", "partially_filled", "", "", 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetCompletedSpreadOrdersLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCompletedSpreadOrdersLast7Days(contextGenerate(), "", "limit", "canceled", "", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadTradesOfLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpreadTradesOfLast7Days(contextGenerate(), "", "", "", "", "", time.Time{}, time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreads(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicSpreads(contextGenerate(), "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadOrderBooks(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicSpreadOrderBooks(contextGenerate(), "", 0)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetPublicSpreadOrderBooks(contextGenerate(), "BTC-USDT_BTC-USDT-SWAP", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicSpreadTickers(contextGenerate(), "")
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetPublicSpreadTickers(contextGenerate(), "BTC-USDT_BTC-USDT-SWAP")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicSpreadTrades(contextGenerate(), "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetSpreadCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadCandlesticks(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetSpreadCandlesticks(contextGenerate(), spreadPair.String(), kline.FiveMin, time.Now().AddDate(0, 0, -1), time.Now(), 10)
	require.NoError(t, err, "GetSpreadCandlesticks must not error")
	assert.NotEmpty(t, result, "GetSpreadCandlesticks should not return an empty result")
}

func TestGetSpreadCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadCandlesticksHistory(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetSpreadCandlesticksHistory(contextGenerate(), spreadPair.String(), kline.FiveMin, time.Now().AddDate(0, 0, -1), time.Now(), 10)
	require.NoError(t, err, "GetSpreadCandlesticksHistory must not error")
	assert.NotEmpty(t, result, "GetSpreadCandlesticksHistory should not return an empty result")
}

func TestGetOptionsTickBands(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionsTickBands(contextGenerate(), "", "")
	require.ErrorIs(t, err, errInvalidInstrumentType)

	result, err := e.GetOptionsTickBands(contextGenerate(), "OPTION", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExtractIndexCandlestick(t *testing.T) {
	t.Parallel()
	data := []byte(`[ [ "1597026383085", "3.721", "3.743", "3.677", "3.708", "1" ], [ "1597026383085", "3.731", "3.799", "3.494", "3.72", "1" ]]`)
	var resp []CandlestickHistoryItem
	err := json.Unmarshal(data, &resp)
	require.NoError(t, err)
	require.Len(t, resp, 2)
	require.Equal(t, 3.743, resp[0].HighestPrice.Float64())
	require.Equal(t, StateCompleted, resp[0].Confirm)
}

func TestGetHistoricIndexAndMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricIndexCandlesticksHistory(contextGenerate(), "", time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetHistoricIndexCandlesticksHistory(contextGenerate(), "BTC-USD", time.Time{}, time.Time{}, kline.FiveMin, 10)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetMarkPriceCandlestickHistory(contextGenerate(), perpetualSwapPair.String(), time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetEconomicCanendarData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetEconomicCalendarData(contextGenerate(), "", "", time.Now(), time.Time{}, 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDepositWithdrawalStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositWithdrawalStatus(contextGenerate(), currency.EMPTYCODE, "", "", "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.GetDepositWithdrawalStatus(contextGenerate(), currency.EMPTYCODE, "", "1244", "", "")
	require.ErrorIs(t, err, errMissingValidWithdrawalID)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositWithdrawalStatus(contextGenerate(), currency.EMPTYCODE, "1244", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetPublicExchangeList(t *testing.T) {
	t.Parallel()
	result, err := e.GetPublicExchangeList(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetInviteesDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetInviteesDetail(contextGenerate(), "")
	require.ErrorIs(t, err, errUserIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetInviteesDetail(contextGenerate(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetUserAffilateRebateInformation(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserAffiliateRebateInformation(contextGenerate(), "")
	require.ErrorIs(t, err, errInvalidAPIKey)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetUserAffiliateRebateInformation(contextGenerate(), "1234")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(contextGenerate(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	usdSwapCode := currency.NewCode("USD-SWAP")
	resp, err := e.GetOpenInterest(contextGenerate(), key.PairAsset{
		Base:  perpetualSwapPair.Base.Item,
		Quote: perpetualSwapPair.Quote.Item,
		Asset: asset.PerpetualSwap,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp1 := currency.NewPair(currency.DOGE, usdSwapCode)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.PerpetualSwap, cp1)
	resp, err = e.GetOpenInterest(contextGenerate(),
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
	resp, err = e.GetOpenInterest(contextGenerate())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, pairs)

		resp, err := e.GetCurrencyTradeURL(contextGenerate(), a, pairs[0])
		assert.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestPlaceLendingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceLendingOrder(contextGenerate(), &LendingOrderParam{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := &LendingOrderParam{AutoRenewal: true}
	_, err = e.PlaceLendingOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	arg.Currency = currency.USDT
	_, err = e.PlaceLendingOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1
	_, err = e.PlaceLendingOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errRateRequired)

	arg.Rate = 0.01
	_, err = e.PlaceLendingOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errLendingTermIsRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.PlaceLendingOrder(contextGenerate(), &LendingOrderParam{
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
	_, err := e.AmendLendingOrder(contextGenerate(), "", 0, 0, false)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.AmendLendingOrder(contextGenerate(), "12312312", 1., 2., true)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLendingOrders(contextGenerate(), "", "pending", currency.ETH, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLendingSubOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingSubOrderList(contextGenerate(), "", "pending", time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetLendingSubOrderList(contextGenerate(), "12345", "", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelAllSpreadOrdersAfterCountdown(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllSpreadOrdersAfterCountdown(contextGenerate(), 2)
	require.ErrorIs(t, err, errCountdownTimeoutRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CancelAllSpreadOrdersAfterCountdown(contextGenerate(), 12)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetContractsOpenInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractsOpenInterestHistory(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetFuturesContractsOpenInterestHistory(contextGenerate(), perpetualSwapPair.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractTakerVolume(contextGenerate(), "", kline.FiveMin, 1, 10, time.Time{}, time.Time{})
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetFuturesContractTakerVolume(contextGenerate(), perpetualSwapPair.String(), kline.FiveMin, 1, 10, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFuturesContractLongShortAccountRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractLongShortAccountRatio(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetFuturesContractLongShortAccountRatio(contextGenerate(), perpetualSwapPair.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTopTradersFuturesContractLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTopTradersFuturesContractLongShortAccountRatio(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetTopTradersFuturesContractLongShortAccountRatio(contextGenerate(), perpetualSwapPair.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetTopTradersFuturesContractLongShortPositionRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetTopTradersFuturesContractLongShortPositionRatio(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetTopTradersFuturesContractLongShortPositionRatio(contextGenerate(), perpetualSwapPair.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAccountInstruments(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountInstruments(contextGenerate(), asset.Empty, "", "", mainPair.String())
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetAccountInstruments(contextGenerate(), asset.Futures, "", "", mainPair.String())
	require.ErrorIs(t, err, errInvalidUnderlying)
	_, err = e.GetAccountInstruments(contextGenerate(), asset.Options, "", "", mainPair.String())
	require.ErrorIs(t, err, errInstrumentFamilyOrUnderlyingRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountInstruments(contextGenerate(), asset.Spot, "", "", mainPair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	result, err = e.GetAccountInstruments(contextGenerate(), asset.PerpetualSwap, "", mainPair.String(), perpetualSwapPair.String())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.NotEmpty(t, p, "GetEnabledPairs must not return empty pairs")

	uly := p[0].Base.String()
	idx := strings.Index(p[0].Quote.String(), "-")
	require.NotEqual(t, -1, idx, "strings.Index must find a hyphen")
	uly += "-" + p[0].Quote.String()[:idx]

	result, err = e.GetAccountInstruments(contextGenerate(), asset.Options, uly, "", p[0].String())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	type OrderTypeWithTIF struct {
		OrderType order.Type
		TIF       order.TimeInForce
	}
	orderTypesToStringMap := map[OrderTypeWithTIF]struct {
		Expected string
		Error    error
	}{
		{OrderType: order.Market, TIF: order.UnknownTIF}:                {Expected: orderMarket},
		{OrderType: order.Limit, TIF: order.UnknownTIF}:                 {Expected: orderLimit},
		{OrderType: order.Limit, TIF: order.PostOnly}:                   {Expected: orderPostOnly},
		{OrderType: order.Market, TIF: order.FillOrKill}:                {Expected: orderFOK},
		{OrderType: order.Market, TIF: order.ImmediateOrCancel}:         {Expected: orderIOC},
		{OrderType: order.OptimalLimit, TIF: order.ImmediateOrCancel}:   {Expected: orderOptimalLimitIOC},
		{OrderType: order.MarketMakerProtection, TIF: order.UnknownTIF}: {Expected: orderMarketMakerProtection},
		{OrderType: order.MarketMakerProtection, TIF: order.PostOnly}:   {Expected: orderMarketMakerProtectionAndPostOnly},
		{OrderType: order.Liquidation, TIF: order.UnknownTIF}:           {Error: order.ErrUnsupportedOrderType},
		{OrderType: order.OCO, TIF: order.UnknownTIF}:                   {Expected: orderOCO},
		{OrderType: order.TrailingStop, TIF: order.UnknownTIF}:          {Expected: orderMoveOrderStop},
		{OrderType: order.Chase, TIF: order.UnknownTIF}:                 {Expected: orderChase},
		{OrderType: order.TWAP, TIF: order.UnknownTIF}:                  {Expected: orderTWAP},
		{OrderType: order.ConditionalStop, TIF: order.UnknownTIF}:       {Expected: orderConditional},
		{OrderType: order.Chase, TIF: order.GoodTillCancel}:             {Expected: orderChase},
		{OrderType: order.TWAP, TIF: order.ImmediateOrCancel}:           {Expected: orderTWAP},
		{OrderType: order.ConditionalStop, TIF: order.GoodTillDay}:      {Expected: orderConditional},
		{OrderType: order.Trigger, TIF: order.UnknownTIF}:               {Expected: orderTrigger},
		{OrderType: order.UnknownType, TIF: order.PostOnly}:             {Expected: orderPostOnly},
		{OrderType: order.UnknownType, TIF: order.FillOrKill}:           {Expected: orderFOK},
		{OrderType: order.UnknownType, TIF: order.ImmediateOrCancel}:    {Expected: orderIOC},
	}
	for tc, val := range orderTypesToStringMap {
		t.Run(tc.OrderType.String()+"/"+tc.TIF.String(), func(t *testing.T) {
			t.Parallel()
			orderTypeString, err := orderTypeString(tc.OrderType, tc.TIF)
			require.ErrorIs(t, err, val.Error)
			assert.Equal(t, val.Expected, orderTypeString)
		})
	}
}

func TestGetMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceCandlesticks(contextGenerate(), "", kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetMarkPriceCandlesticks(contextGenerate(), mainPair.String(), kline.FiveMin, time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetHistoricIndexCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricIndexCandlesticksHistory(contextGenerate(), "", time.Time{}, time.Time{}, kline.TenMin, 10)
	require.ErrorIs(t, err, errMissingInstrumentID)

	result, err := e.GetHistoricIndexCandlesticksHistory(contextGenerate(), mainPair.String(), time.Time{}, time.Time{}, kline.FiveMin, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAssetTypeString(t *testing.T) {
	t.Parallel()
	_, err := assetTypeString(asset.LinearContract)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	assetTypes := e.GetAssetTypes(false)
	for a := range assetTypes {
		if assetTypes[a] == asset.Spread {
			continue
		}
		_, err := assetTypeString(assetTypes[a])
		assert.NoError(t, err)
	}
}

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	result, err := e.GetAnnouncements(contextGenerate(), "", 0)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAnnouncementTypes(t *testing.T) {
	t.Parallel()
	_, err := e.GetAnnouncementTypes(contextGenerate())
	assert.NoError(t, err)
	// No tests of contents of resp because currently in US based github actions announcement-types returns empty
}

func TestGetDepositOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositOrderDetail(contextGenerate(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositOrderDetail(contextGenerate(), "12312312")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatDepositOrderHistory(contextGenerate(), currency.USDT, "TR_BANKS", "failed", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalOrderDetail(contextGenerate(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalOrderDetail(contextGenerate(), "024041201450544699")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatWithdrawalOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatWithdrawalOrderHistory(contextGenerate(), currency.USDT, "SEPA", "failed", time.Time{}, time.Time{}, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCancelWithdrawalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawalOrder(contextGenerate(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.CancelWithdrawalOrder(contextGenerate(), "124041201450544699")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreateWithdrawalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateWithdrawalOrder(contextGenerate(), currency.BTC, "", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, errIDNotSet)
	_, err = e.CreateWithdrawalOrder(contextGenerate(), currency.EMPTYCODE, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CreateWithdrawalOrder(contextGenerate(), currency.BTC, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.CreateWithdrawalOrder(contextGenerate(), currency.BTC, "1231312312", "", "194a6975e98246538faeb0fab0d502df", 1000)
	require.ErrorIs(t, err, errPaymentMethodRequired)
	_, err = e.CreateWithdrawalOrder(contextGenerate(), currency.BTC, "1231312312", "SEPA", "", 1000)
	require.ErrorIs(t, err, errIDNotSet)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	result, err := e.CreateWithdrawalOrder(contextGenerate(), currency.BTC, "1231312312", "SEPA", "194a6975e98246538faeb0fab0d502df", 1000)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatWithdrawalPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatWithdrawalPaymentMethods(contextGenerate(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatWithdrawalPaymentMethods(contextGenerate(), currency.TRY)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetFiatDepositPaymentMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositPaymentMethods(contextGenerate(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFiatDepositPaymentMethods(contextGenerate(), currency.TRY)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func (e *Exchange) instrumentFamilyFromInstID(instrumentType, instID string) (string, error) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	if instrumentType != "" {
		insts, okay := e.instrumentsInfoMap[instrumentType]
		if !okay {
			return "", errInvalidInstrumentType
		}
		for a := range insts {
			if insts[a].InstrumentID.String() == instID {
				return insts[a].InstrumentFamily, nil
			}
		}
	} else {
		for _, insts := range e.instrumentsInfoMap {
			for a := range insts {
				if insts[a].InstrumentID.String() == instID {
					return insts[a].InstrumentFamily, nil
				}
			}
		}
	}
	return "", fmt.Errorf("instrument family not found for instrument %s", instID)
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	public, err := e.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	private, err := e.generateSubscriptions(false)
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{
		{Channel: subscription.MyAccountChannel, QualifiedChannel: `{"channel":"account"}`, Authenticated: true},
	}
	var pairs currency.Pairs
	for _, s := range e.Features.Subscriptions {
		for _, a := range e.GetAssetTypes(true) {
			if a == asset.Spread || (s.Asset != asset.All && s.Asset != a) {
				continue
			}
			pairs, err = e.GetEnabledPairs(a)
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
					s.QualifiedChannel = fmt.Sprintf(`{"channel":%q,"instType":%q}`, name, GetInstrumentTypeFromAssetItem(s.Asset))
				} else {
					s.QualifiedChannel = `{"channel":"` + name + `"}`
				}
				s.Pairs = pairs
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, append(public, private...))

	e.Features.Subscriptions = subscription.List{{Channel: channelGridPositions, Params: map[string]any{"algoId": "42"}}}
	public, err = e.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	private, err = e.generateSubscriptions(false)
	require.NoError(t, err, "generateSubscriptions must not error")
	exp = subscription.List{{Channel: channelGridPositions, Params: map[string]any{"algoId": "42"}, QualifiedChannel: `{"channel":"grid-positions","algoId":"42"}`}}
	testsubs.EqualLists(t, exp, append(public, private...))

	e.Features.Subscriptions = subscription.List{{Channel: channelGridPositions}}
	public, err = e.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")
	private, err = e.generateSubscriptions(false)
	require.NoError(t, err, "generateSubscriptions must not error")
	exp = subscription.List{{Channel: channelGridPositions, QualifiedChannel: `{"channel":"grid-positions"}`}}
	testsubs.EqualLists(t, exp, append(public, private...))
}

// TODO: Implement channel subscriptions for business ws and remove this test
func TestBusinessWSCandleSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	e.Features.Subscriptions = nil // Subscriptions not needed for this test

	finish := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() { // reader routine so nothing blocks
		for {
			select {
			case <-finish:
				return
			case <-e.Websocket.DataHandler.C:
			}
		}
	})

	for _, a := range e.GetAssetTypes(true) { // Disable all assets except spread and spot so only those are tested and data handler isn't polluted
		switch a {
		case asset.Spread:
			enabled, err := e.GetBase().CurrencyPairs.GetPairs(a, true)
			require.NoError(t, err, "GetPairs must not error")
			randomPair, err := enabled.GetRandomPair()
			require.NoError(t, err, "GetRandomPair must not error")
			require.NoError(t, e.GetBase().SetPairs(currency.Pairs{randomPair}, a, true), "SetPairs must not error")
			continue
		case asset.Spot:
		default:
			require.NoError(t, e.GetBase().CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error")
		}
	}

	require.NoError(t, e.Websocket.Connect(t.Context()))

	conn, err := e.Websocket.GetConnection(businessConnection)
	require.NoError(t, err, "GetConnection must not error")

	err = e.BusinessSubscribe(t.Context(), conn, subscription.List{{Channel: channelCandle1D}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	close(finish) // yield so that assertion below gets all data
	wg.Wait()

	p := currency.Pairs{
		mainPair,
		currency.NewPairWithDelimiter("ETH", "USDT", "-"),
		currency.NewPairWithDelimiter("OKB", "USDT", "-"),
	}

	var subs subscription.List
	for i, ch := range []string{channelCandle1D, channelMarkPriceCandle1M, channelIndexCandle1H} {
		subs = append(subs, &subscription.Subscription{Channel: ch, Pairs: p[i : i+1]})
	}

	err = e.BusinessSubscribe(t.Context(), conn, subs)
	require.NoError(t, err, "BusinessSubscribe must not error")

	var got currency.Pairs
	check := func() bool {
		data := <-e.Websocket.DataHandler.C
		switch v := data.Data.(type) {
		case websocket.KlineData:
			got = got.Add(v.Pair)
		case []CandlestickMarkPrice:
			if len(v) > 0 {
				got = got.Add(v[0].Pair)
			}
		default:
		}
		return len(got) == 3
	}
	assert.Eventually(t, check, 5*time.Second, time.Millisecond)
	require.Equal(t, 3, len(got), "must receive candles for all three subscriptions")
	require.NoError(t, got.ContainsAll(p, true), "must receive candles for all subscribed pairs")
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
	err := e.wsProcessSpreadOrderbook([]byte(processSpreadOrderbookJSON))
	assert.NoError(t, err)
}

func TestWsProcessPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	err := e.wsProcessPublicSpreadTrades([]byte(wsProcessPublicSpreadTradesJSON))
	assert.NoError(t, err)
}

func TestWsProcessPublicSpreadTicker(t *testing.T) {
	t.Parallel()
	err := e.wsProcessPublicSpreadTicker(t.Context(), []byte(okxSpreadPublicTickerJSON))
	assert.NoError(t, err)
}

func TestWsProcessSpreadOrders(t *testing.T) {
	t.Parallel()
	err := e.wsProcessSpreadOrders(t.Context(), []byte(wsProcessSpreadOrdersJSON))
	assert.NoError(t, err)
}

func TestWsProcessSpreadTradesJSON(t *testing.T) {
	t.Parallel()
	err := e.wsProcessSpreadTrades([]byte(wsProcessSpreadTradesJSON))
	assert.NoError(t, err)
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	orderTypeStrings := map[string]struct {
		OType order.Type
		TIF   order.TimeInForce
		Error error
	}{
		"market":            {OType: order.Market},
		"LIMIT":             {OType: order.Limit},
		"limit":             {OType: order.Limit},
		"post_only":         {OType: order.Limit, TIF: order.PostOnly},
		"fok":               {OType: order.Limit, TIF: order.FillOrKill},
		"ioc":               {OType: order.Limit, TIF: order.ImmediateOrCancel},
		"optimal_limit_ioc": {OType: order.OptimalLimit, TIF: order.ImmediateOrCancel},
		"mmp":               {OType: order.MarketMakerProtection},
		"mmp_and_post_only": {OType: order.MarketMakerProtection, TIF: order.PostOnly},
		"trigger":           {OType: order.UnknownType, Error: order.ErrTypeIsInvalid},
		"chase":             {OType: order.Chase},
		"move_order_stop":   {OType: order.TrailingStop},
		"twap":              {OType: order.TWAP},
		"abcd":              {OType: order.UnknownType, Error: order.ErrTypeIsInvalid},
	}
	for s, exp := range orderTypeStrings {
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			oType, tif, err := orderTypeFromString(s)
			require.ErrorIs(t, err, exp.Error)
			assert.Equal(t, exp.OType, oType)
			assert.Equal(t, exp.TIF.String(), tif.String())
		})
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder := &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyWithdrawalFee,
		Pair:          mainPair,
		PurchasePrice: 1,
	}
	_, err := e.GetFee(contextGenerate(), feeBuilder)
	require.ErrorIs(t, err, errFeeTypeUnsupported)

	feeBuilder.FeeType = exchange.OfflineTradeFee
	_, err = e.GetFee(contextGenerate(), feeBuilder)
	assert.NoError(t, err)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
	_, err = e.GetFee(contextGenerate(), feeBuilder)
	require.NoError(t, err)
}

func TestPriceTypeString(t *testing.T) {
	t.Parallel()
	priceTypeToStringMap := map[order.PriceType]string{
		order.LastPrice:        "last",
		order.IndexPrice:       "index",
		order.MarkPrice:        "mark",
		order.UnknownPriceType: "",
	}
	var priceTString string
	for x := range priceTypeToStringMap {
		priceTString = priceTypeString(x)
		assert.Equal(t, priceTString, priceTypeToStringMap[x])
	}
}

func TestMarginTypeToString(t *testing.T) {
	t.Parallel()
	marginTypeToStringMap := map[margin.Type]string{
		margin.Isolated:     "isolated",
		margin.Multi:        "cross",
		margin.NoMargin:     "cash",
		margin.SpotIsolated: "spot_isolated",
		margin.Unset:        "",
	}
	var marginTypeString string
	for m := range marginTypeToStringMap {
		marginTypeString = e.marginTypeToString(m)
		assert.Equal(t, marginTypeString, marginTypeToStringMap[m])
	}
}

func TestValidatePlaceOrderRequestParam(t *testing.T) {
	t.Parallel()
	var p *PlaceOrderRequestParam
	require.ErrorIs(t, p.Validate(), common.ErrNilPointer)
	p = &PlaceOrderRequestParam{}
	require.ErrorIs(t, p.Validate(), errMissingInstrumentID)
	p.InstrumentID = mainPair.String()
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid)
	p.Side = order.Buy.String()
	p.TradeMode = "abc"
	require.ErrorIs(t, p.Validate(), errInvalidTradeModeValue)
	p.TradeMode = TradeModeIsolated
	p.AssetType = asset.Futures
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid)
	p.PositionSide = "long"
	require.ErrorIs(t, p.Validate(), order.ErrTypeIsInvalid)
	p.OrderType = order.Market.String()
	require.ErrorIs(t, p.Validate(), limits.ErrAmountBelowMin)
	p.Amount = 1
	p.TargetCurrency = "moo cows"
	require.ErrorIs(t, p.Validate(), errCurrencyQuantityTypeRequired)
	p.TargetCurrency = "base_ccy"
	require.NoError(t, p.Validate())
}

func TestValidateSpreadOrderParam(t *testing.T) {
	t.Parallel()
	var p *SpreadOrderParam
	require.ErrorIs(t, p.Validate(), common.ErrNilPointer)
	p = &SpreadOrderParam{}
	require.ErrorIs(t, p.Validate(), errMissingInstrumentID)
	p.SpreadID = spreadPair.String()
	require.ErrorIs(t, p.Validate(), order.ErrTypeIsInvalid)
	p.OrderType = order.Market.String()
	require.ErrorIs(t, p.Validate(), limits.ErrAmountBelowMin)
	p.Size = 1
	require.ErrorIs(t, p.Validate(), limits.ErrPriceBelowMin)
	p.Price = 1
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid)
	p.Side = order.Buy.String()
	require.NoError(t, p.Validate())
}
