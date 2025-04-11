package gateio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g *Gateio

func TestMain(m *testing.M) {
	g = new(Gateio)
	if err := testexch.Setup(g); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		g.API.AuthenticatedSupport = true
		g.API.AuthenticatedWebsocketSupport = true
		g.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, g)
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelAllOrders(t.Context(), nil)
	if !errors.Is(err, order.ErrCancelOrderIsNil) {
		t.Error(err)
	}
	orderCancellation := &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          getPair(t, asset.Options),
		AssetType:     asset.Options,
	}
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.Spot
	orderCancellation.Pair = getPair(t, asset.Spot)
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.Margin
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = getPair(t, asset.Margin)
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.CrossMargin
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = getPair(t, asset.CrossMargin)
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.Futures
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = getPair(t, asset.Futures)
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.DeliveryFutures
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = getPair(t, asset.DeliveryFutures)
	_, err = g.CancelAllOrders(t.Context(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.UpdateAccountInfo(t.Context(), asset.Spot)
	if err != nil {
		t.Error("GetAccountInfo() error", err)
	}
	if _, err := g.UpdateAccountInfo(t.Context(), asset.Margin); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(t.Context(), asset.CrossMargin); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(t.Context(), asset.Options); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(t.Context(), asset.Futures); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(t.Context(), asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	cryptocurrencyChains, err := g.GetAvailableTransferChains(t.Context(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	} else if len(cryptocurrencyChains) == 0 {
		t.Fatal("no crypto currency chain available")
	}
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    g.Name,
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
			Chain:   cryptocurrencyChains[0],
		},
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err = g.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest); err != nil {
		t.Errorf("%s WithdrawCryptocurrencyFunds() error: %v", g.Name, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetOrderInfo(t.Context(),
		"917591554", getPair(t, asset.Spot), asset.Spot)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(t.Context(), "917591554", getPair(t, asset.Options), asset.Options)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(t.Context(), "917591554", getPair(t, asset.Margin), asset.Margin)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(t.Context(), "917591554", getPair(t, asset.CrossMargin), asset.CrossMargin)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(t.Context(), "917591554", getPair(t, asset.Futures), asset.Futures)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(t.Context(), "917591554", getPair(t, asset.DeliveryFutures), asset.DeliveryFutures)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.UpdateTicker(t.Context(), getPair(t, a), a)
		assert.NoError(t, err, "UpdateTicker should not error for %s", a)
	}
}

func TestListSpotCurrencies(t *testing.T) {
	t.Parallel()
	if _, err := g.ListSpotCurrencies(t.Context()); err != nil {
		t.Errorf("%s ListAllCurrencies() error %v", g.Name, err)
	}
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCurrencyDetail(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetCurrencyDetail() error %v", g.Name, err)
	}
}

func TestListAllCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, err := g.ListSpotCurrencyPairs(t.Context()); err != nil {
		t.Errorf("%s ListAllCurrencyPairs() error %v", g.Name, err)
	}
}

func TestGetCurrencyPairDetal(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCurrencyPairDetail(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}.String()); err != nil {
		t.Errorf("%s GetCurrencyPairDetal() error %v", g.Name, err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetTickers(t.Context(), "BTC_USDT", ""); err != nil {
		t.Errorf("%s GetTickers() error %v", g.Name, err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	if _, err := g.GetTicker(t.Context(), currency.Pair{Base: currency.BTC, Delimiter: currency.UnderscoreDelimiter, Quote: currency.USDT}.String(), utc8TimeZone); err != nil {
		t.Errorf("%s GetTicker() error %v", g.Name, err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook(t.Context(), getPair(t, asset.Spot).String(), "0.1", 10, false)
	assert.NoError(t, err, "GetOrderbook should not error")
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	assert.NoError(t, err, "getSettlementFromCurrency should not error")
	_, err = g.GetFuturesOrderbook(t.Context(), settle, getPair(t, asset.Futures).String(), "", 10, false)
	assert.NoError(t, err, "GetFuturesOrderbook should not error")
	settle, err = getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "getSettlementFromCurrency should not error")
	_, err = g.GetDeliveryOrderbook(t.Context(), settle, "0.1", getPair(t, asset.DeliveryFutures), 10, false)
	assert.NoError(t, err, "GetDeliveryOrderbook should not error")
	_, err = g.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 10, false)
	assert.NoError(t, err, "GetOptionsOrderbook should not error")
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	if _, err := g.GetMarketTrades(t.Context(), getPair(t, asset.Spot), 0, "", true, time.Time{}, time.Time{}, 1); err != nil {
		t.Errorf("%s GetMarketTrades() error %v", g.Name, err)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCandlesticks(t.Context(), getPair(t, asset.Spot), 0, time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Errorf("%s GetCandlesticks() error %v", g.Name, err)
	}
}

func TestGetTradingFeeRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetTradingFeeRatio(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}); err != nil {
		t.Errorf("%s GetTradingFeeRatio() error %v", g.Name, err)
	}
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSpotAccounts(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetSpotAccounts() error %v", g.Name, err)
	}
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CreateBatchOrders(t.Context(), []CreateOrderRequest{
		{
			CurrencyPair: getPair(t, asset.Spot),
			Side:         "sell",
			Amount:       0.001,
			Price:        12349,
			Account:      g.assetTypeToString(asset.Spot),
			Type:         "limit",
		},
		{
			CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter},
			Side:         "buy",
			Amount:       1,
			Price:        1234567789,
			Account:      g.assetTypeToString(asset.Spot),
			Type:         "limit",
		},
	})
	assert.NoError(t, err, "CreateBatchOrders should not error")
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSpotOpenOrders(t.Context(), 0, 0, false); err != nil {
		t.Errorf("%s GetSpotOpenOrders() error %v", g.Name, err)
	}
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.SpotClosePositionWhenCrossCurrencyDisabled(t.Context(), &ClosePositionRequestParam{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: getPair(t, asset.Spot),
	}); err != nil {
		t.Errorf("%s SpotClosePositionWhenCrossCurrencyDisabled() error %v", g.Name, err)
	}
}

func TestCreateSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.PlaceSpotOrder(t.Context(), &CreateOrderRequest{
		CurrencyPair: getPair(t, asset.Spot),
		Side:         "buy",
		Amount:       1,
		Price:        900000,
		Account:      g.assetTypeToString(asset.Spot),
		Type:         "limit",
	})
	assert.NoError(t, err, "PlaceSpotOrder should not error")
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSpotOrders(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, statusOpen, 0, 0)
	assert.NoError(t, err, "GetSpotOrders should not error")
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelAllOpenOrdersSpecifiedCurrencyPair(t.Context(), getPair(t, asset.Spot), order.Sell, asset.Empty); err != nil {
		t.Errorf("%s CancelAllOpenOrdersSpecifiedCurrencyPair() error %v", g.Name, err)
	}
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelBatchOrdersWithIDList(t.Context(), []CancelOrderByIDParam{
		{
			CurrencyPair: getPair(t, asset.Spot),
			ID:           "1234567",
		},
		{
			CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter},
			ID:           "something",
		},
	}); err != nil {
		t.Errorf("%s CancelBatchOrderWithIDList() error %v", g.Name, err)
	}
}

func TestGetSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSpotOrder(t.Context(), "1234", currency.Pair{
		Base:      currency.BTC,
		Delimiter: currency.UnderscoreDelimiter,
		Quote:     currency.USDT,
	}, asset.Spot); err != nil {
		t.Errorf("%s GetSpotOrder() error %v", g.Name, err)
	}
}

func TestAmendSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := g.AmendSpotOrder(t.Context(), "", getPair(t, asset.Spot), false, &PriceAndAmount{
		Price: 1000,
	})
	if !errors.Is(err, errInvalidOrderID) {
		t.Errorf("expecting %v, but found %v", errInvalidOrderID, err)
	}
	_, err = g.AmendSpotOrder(t.Context(), "123", currency.EMPTYPAIR, false, &PriceAndAmount{
		Price: 1000,
	})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("expecting %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err = g.AmendSpotOrder(t.Context(), "123", getPair(t, asset.Spot), false, &PriceAndAmount{
		Price: 1000,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelSingleSpotOrder(t.Context(), "1234",
		getPair(t, asset.Spot).String(), false); err != nil {
		t.Errorf("%s CancelSingleSpotOrder() error %v", g.Name, err)
	}
}

func TestGetMySpotTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetMySpotTradingHistory(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, "", 0, 0, false, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	if _, err := g.GetServerTime(t.Context(), asset.Spot); err != nil {
		t.Errorf("%s GetServerTime() error %v", g.Name, err)
	}
}

func TestCountdownCancelorder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CountdownCancelorders(t.Context(), CountdownCancelOrderParam{
		Timeout:      10,
		CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.ETH, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		t.Errorf("%s CountdownCancelorder() error %v", g.Name, err)
	}
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreatePriceTriggeredOrder(t.Context(), &PriceTriggeredOrderParam{
		Trigger: TriggerPriceInfo{
			Price:      123,
			Rule:       ">=",
			Expiration: 3600,
		},
		Put: PutOrderData{
			Type:        "limit",
			Side:        "sell",
			Price:       2312312,
			Amount:      30,
			TimeInForce: "gtc",
		},
		Market: currency.Pair{Base: currency.GT, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		t.Errorf("%s CreatePriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestGetPriceTriggeredOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetPriceTriggeredOrderList(t.Context(), statusOpen, currency.EMPTYPAIR, asset.Empty, 0, 0)
	assert.NoError(t, err, "GetPriceTriggeredOrderList should not error")
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelMultipleSpotOpenOrders(t.Context(), currency.EMPTYPAIR, asset.CrossMargin); err != nil {
		t.Errorf("%s CancelAllOpenOrders() error %v", g.Name, err)
	}
}

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSinglePriceTriggeredOrder(t.Context(), "1234"); err != nil {
		t.Errorf("%s GetSinglePriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.CancelPriceTriggeredOrder(t.Context(), "1234"); err != nil {
		t.Errorf("%s CancelPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestGetMarginAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMarginAccountList(t.Context(), currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMarginAccountList() error %v", g.Name, err)
	}
}

func TestListMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListMarginAccountBalanceChangeHistory(t.Context(), currency.BTC, currency.Pair{
		Base:      currency.BTC,
		Delimiter: currency.UnderscoreDelimiter,
		Quote:     currency.USDT,
	}, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s ListMarginAccountBalanceChangeHistory() error %v", g.Name, err)
	}
}

func TestGetMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMarginFundingAccountList(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetMarginFundingAccountList %v", g.Name, err)
	}
}

func TestMarginLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.MarginLoan(t.Context(), &MarginLoanRequestParam{
		Side:         "borrow",
		Amount:       1,
		Currency:     currency.BTC,
		CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter},
		Days:         10,
		Rate:         0.0002,
	}); err != nil {
		t.Errorf("%s MarginLoan() error %v", g.Name, err)
	}
}

func TestGetMarginAllLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetMarginAllLoans(t.Context(), statusOpen, "lend", "", currency.BTC, currency.Pair{Base: currency.BTC, Delimiter: currency.UnderscoreDelimiter, Quote: currency.USDT}, false, 0, 0)
	assert.NoError(t, err, "GetMarginAllLoans should not error")
}

func TestMergeMultipleLendingLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.MergeMultipleLendingLoans(t.Context(), currency.USDT, []string{"123", "23423"}); err != nil {
		t.Errorf("%s MergeMultipleLendingLoans() error %v", g.Name, err)
	}
}

func TestRetriveOneSingleLoanDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.RetriveOneSingleLoanDetail(t.Context(), "borrow", "123"); err != nil {
		t.Errorf("%s RetriveOneSingleLoanDetail() error %v", g.Name, err)
	}
}

func TestModifyALoan(t *testing.T) {
	t.Parallel()
	if _, err := g.ModifyALoan(t.Context(), "1234", &ModifyLoanRequestParam{
		Currency:  currency.BTC,
		Side:      "borrow",
		AutoRenew: false,
	}); !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("%s ModifyALoan() error %v", g.Name, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.ModifyALoan(t.Context(), "1234", &ModifyLoanRequestParam{
		Currency:     currency.BTC,
		Side:         "borrow",
		AutoRenew:    false,
		CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		t.Errorf("%s ModifyALoan() error %v", g.Name, err)
	}
}

func TestCancelLendingLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.CancelLendingLoan(t.Context(), currency.BTC, "1234"); err != nil {
		t.Errorf("%s CancelLendingLoan() error %v", g.Name, err)
	}
}

func TestRepayALoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.RepayALoan(t.Context(), "1234", &RepayLoanRequestParam{
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		Currency:     currency.BTC,
		Mode:         "all",
	}); err != nil {
		t.Errorf("%s RepayALoan() error %v", g.Name, err)
	}
}

func TestListLoanRepaymentRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListLoanRepaymentRecords(t.Context(), "1234"); err != nil {
		t.Errorf("%s LoanRepaymentRecord() error %v", g.Name, err)
	}
}

func TestListRepaymentRecordsOfSpecificLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListRepaymentRecordsOfSpecificLoan(t.Context(), "1234", "", 0, 0); err != nil {
		t.Errorf("%s error while ListRepaymentRecordsOfSpecificLoan() %v", g.Name, err)
	}
}

func TestGetOneSingleloanRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOneSingleLoanRecord(t.Context(), "1234", "123"); err != nil {
		t.Errorf("%s error while GetOneSingleloanRecord() %v", g.Name, err)
	}
}

func TestModifyALoanRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ModifyALoanRecord(t.Context(), "1234", &ModifyLoanRequestParam{
		Currency:     currency.USDT,
		CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		Side:         "lend",
		AutoRenew:    true,
		LoanID:       "1234",
	}); err != nil {
		t.Errorf("%s ModifyALoanRecord() error %v", g.Name, err)
	}
}

func TestUpdateUsersAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.UpdateUsersAutoRepaymentSetting(t.Context(), true); err != nil {
		t.Errorf("%s UpdateUsersAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUserAutoRepaymentSetting(t.Context()); err != nil {
		t.Errorf("%s GetUserAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxTransferableAmountForSpecificMarginCurrency(t.Context(), currency.BTC, currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxBorrowableAmountForSpecificMarginCurrency(t.Context(), currency.BTC, currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.CurrencySupportedByCrossMargin(t.Context()); err != nil {
		t.Errorf("%s CurrencySupportedByCrossMargin() error %v", g.Name, err)
	}
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginSupportedCurrencyDetail(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetCrossMarginSupportedCurrencyDetail() error %v", g.Name, err)
	}
}

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginAccounts(t.Context()); err != nil {
		t.Errorf("%s GetCrossMarginAccounts() error %v", g.Name, err)
	}
}

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginAccountChangeHistory(t.Context(), currency.BTC, time.Time{}, time.Time{}, 0, 6, "in"); err != nil {
		t.Errorf("%s GetCrossMarginAccountChangeHistory() error %v", g.Name, err)
	}
}

var createCrossMarginBorrowLoanJSON = `{"id": "17",	"create_time": 1620381696159,	"update_time": 1620381696159,	"currency": "EOS",	"amount": "110.553635",	"text": "web",	"status": 2,	"repaid": "110.506649705159",	"repaid_interest": "0.046985294841",	"unpaid_interest": "0.0000074393366667"}`

func TestCreateCrossMarginBorrowLoan(t *testing.T) {
	t.Parallel()
	var response CrossMarginLoanResponse
	if err := json.Unmarshal([]byte(createCrossMarginBorrowLoanJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to CrossMarginBorrowLoanResponse %v", g.Name, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateCrossMarginBorrowLoan(t.Context(), CrossMarginBorrowLoanParams{
		Currency: currency.BTC,
		Amount:   3,
	}); err != nil {
		t.Errorf("%s CreateCrossMarginBorrowLoan() error %v", g.Name, err)
	}
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginBorrowHistory(t.Context(), 1, currency.BTC, 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginBorrowHistory() error %v", g.Name, err)
	}
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleBorrowLoanDetail(t.Context(), "1234"); err != nil {
		t.Errorf("%s GetSingleBorrowLoanDetail() error %v", g.Name, err)
	}
}

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.ExecuteRepayment(t.Context(), CurrencyAndAmount{
		Currency: currency.USD,
		Amount:   1234.55,
	}); err != nil {
		t.Errorf("%s ExecuteRepayment() error %v", g.Name, err)
	}
}

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginRepayments(t.Context(), currency.BTC, "123", 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginRepayments() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxTransferableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestListCurrencyChain(t *testing.T) {
	t.Parallel()
	if _, err := g.ListCurrencyChain(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s ListCurrencyChain() error %v", g.Name, err)
	}
}

func TestGenerateCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GenerateCurrencyDepositAddress(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GenerateCurrencyDepositAddress() error %v", g.Name, err)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetWithdrawalRecords(t.Context(), currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetWithdrawalRecords() error %v", g.Name, err)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDepositRecords(t.Context(), currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetDepositRecords() error %v", g.Name, err)
	}
}

func TestTransferCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.TransferCurrency(t.Context(), &TransferCurrencyParam{
		Currency:     currency.BTC,
		From:         g.assetTypeToString(asset.Spot),
		To:           g.assetTypeToString(asset.Margin),
		Amount:       1202.000,
		CurrencyPair: getPair(t, asset.Spot),
	}); err != nil {
		t.Errorf("%s TransferCurrency() error %v", g.Name, err)
	}
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.SubAccountTransfer(t.Context(), SubAccountTransferParam{
		Currency:   currency.BTC,
		SubAccount: "12222",
		Direction:  "to",
		Amount:     1,
	}); err != nil {
		t.Errorf("%s SubAccountTransfer() error %v", g.Name, err)
	}
}

func TestGetSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.GetSubAccountTransferHistory(t.Context(), "", time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetSubAccountTransferHistory() error %v", g.Name, err)
	}
}

func TestSubAccountTransferToSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.SubAccountTransferToSubAccount(t.Context(), &InterSubAccountTransferParams{
		Currency:                currency.BTC,
		SubAccountFromUserID:    "1234",
		SubAccountFromAssetType: asset.Spot,
		SubAccountToUserID:      "4567",
		SubAccountToAssetType:   asset.Spot,
		Amount:                  1234,
	}); err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetWithdrawalStatus(t.Context(), currency.NewCode("")); err != nil {
		t.Errorf("%s GetWithdrawalStatus() error %v", g.Name, err)
	}
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountBalances(t.Context(), ""); err != nil {
		t.Errorf("%s GetSubAccountBalances() error %v", g.Name, err)
	}
}

func TestGetSubAccountMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountMarginBalances(t.Context(), ""); err != nil {
		t.Errorf("%s GetSubAccountMarginBalances() error %v", g.Name, err)
	}
}

func TestGetSubAccountFuturesBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSubAccountFuturesBalances(t.Context(), "", currency.EMPTYCODE)
	assert.Error(t, err, "GetSubAccountFuturesBalances should not error")
}

func TestGetSubAccountCrossMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountCrossMarginBalances(t.Context(), ""); err != nil {
		t.Errorf("%s GetSubAccountCrossMarginBalances() error %v", g.Name, err)
	}
}

func TestGetSavedAddresses(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSavedAddresses(t.Context(), currency.BTC, "", 0); err != nil {
		t.Errorf("%s GetSavedAddresses() error %v", g.Name, err)
	}
}

func TestGetPersonalTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetPersonalTradingFee(t.Context(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, currency.EMPTYCODE)
	assert.NoError(t, err, "GetPersonalTradingFee should not error")
}

func TestGetUsersTotalBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersTotalBalance(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetUsersTotalBalance() error %v", g.Name, err)
	}
}

func TestGetMarginSupportedCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, err := g.GetMarginSupportedCurrencyPairs(t.Context()); err != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, err)
	}
}

func TestGetMarginSupportedCurrencyPair(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSingleMarginSupportedCurrencyPair(t.Context(), getPair(t, asset.Margin)); err != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, err)
	}
}

func TestGetOrderbookOfLendingLoans(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOrderbookOfLendingLoans(t.Context(), currency.BTC); err != nil {
		t.Errorf("%s GetOrderbookOfLendingLoans() error %v", g.Name, err)
	}
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()
	for _, settlementCurrency := range settlementCurrencies {
		if _, err := g.GetAllFutureContracts(t.Context(), settlementCurrency); err != nil {
			assert.Errorf(t, err, "GetAllFutureContracts %s should not error", settlementCurrency)
		}
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFuturesContract(t.Context(), settle, getPair(t, asset.Futures).String())
	assert.NoError(t, err, "GetFuturesContract should not error")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFuturesOrderbook(t.Context(), settle, getPair(t, asset.Futures).String(), "", 0, false)
	assert.NoError(t, err, "GetFuturesOrderbook should not error")
}

func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFuturesTradingHistory(t.Context(), settle, getPair(t, asset.Futures), 0, 0, "", time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesTradingHistory should not error")
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFuturesCandlesticks(t.Context(), settle, getPair(t, asset.Futures).String(), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "GetFuturesCandlesticks should not error")
}

func TestPremiumIndexKLine(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.PremiumIndexKLine(t.Context(), settle, getPair(t, asset.Futures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "PremiumIndexKLine should not error")
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFuturesTickers(t.Context(), settle, getPair(t, asset.Futures))
	assert.NoError(t, err, "GetFutureTickers should not error")
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFutureFundingRates(t.Context(), settle, getPair(t, asset.Futures), 0)
	assert.NoError(t, err, "GetFutureFundingRates should not error")
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesInsuranceBalanceHistory(t.Context(), currency.USDT, 0)
	assert.NoError(t, err, "GetFuturesInsuranceBalanceHistory should not error")
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetFutureStats(t.Context(), settle, getPair(t, asset.Futures), time.Time{}, 0, 0)
	assert.NoError(t, err, "GetFutureStats should not error")
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	_, err := g.GetIndexConstituent(t.Context(), currency.USDT, currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}.String())
	assert.NoError(t, err, "GetIndexConstituent should not error")
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetLiquidationHistory(t.Context(), settle, getPair(t, asset.Futures), time.Time{}, time.Time{}, 0)
	assert.NoError(t, err, "GetLiquidationHistory should not error")
}

func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.QueryFuturesAccount(t.Context(), currency.USDT)
	assert.NoError(t, err, "QueryFuturesAccount should not error")
}

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesAccountBooks(t.Context(), currency.USDT, 0, time.Time{}, time.Time{}, "dnw")
	assert.NoError(t, err, "GetFuturesAccountBooks should not error")
}

func TestGetAllFuturesPositionsOfUsers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetAllFuturesPositionsOfUsers(t.Context(), currency.USDT, true)
	assert.NoError(t, err, "GetAllPositionsOfUsers should not error")
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSinglePosition(t.Context(), currency.USDT, currency.Pair{Quote: currency.BTC, Base: currency.USDT})
	assert.NoError(t, err, "GetSinglePosition should not error")
}

func TestUpdateFuturesPositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdateFuturesPositionMargin(t.Context(), settle, 0.01, getPair(t, asset.Futures))
	assert.NoError(t, err, "UpdateFuturesPositionMargin should not error")
}

func TestUpdateFuturesPositionLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdateFuturesPositionLeverage(t.Context(), settle, getPair(t, asset.Futures), 1, 0)
	assert.NoError(t, err, "UpdateFuturesPositionLeverage should not error")
}

func TestUpdateFuturesPositionRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdateFuturesPositionRiskLimit(t.Context(), settle, getPair(t, asset.Futures), 10)
	assert.NoError(t, err, "UpdateFuturesPositionRiskLimit should not error")
}

func TestPlaceDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.PlaceDeliveryOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.DeliveryFutures),
		Size:        6024,
		Iceberg:     0,
		Price:       "3765",
		Text:        "t-my-custom-id",
		Settle:      settle,
		TimeInForce: gtcTIF,
	})
	assert.NoError(t, err, "CreateDeliveryOrder should not error")
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), statusOpen, settle, "", 0, 0, 1)
	assert.NoError(t, err, "GetDeliveryOrders should not error")
}

func TestCancelMultipleDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.CancelMultipleDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), "ask", settle)
	assert.NoError(t, err, "CancelMultipleDeliveryOrders should not error")
}

func TestGetSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	_, err := g.GetSingleDeliveryOrder(t.Context(), currency.EMPTYCODE, "123456")
	assert.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency, "GetSingleDeliveryOrder should return errEmptyOrInvalidSettlementCurrency")
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err = g.GetSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	assert.NoError(t, err, "GetSingleDeliveryOrder should not error")
}

func TestCancelSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelSingleDeliveryOrder(t.Context(), currency.USDT, "123456")
	assert.NoError(t, err, "CancelSingleDeliveryOrder should not error")
}

func TestGetMyDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetMyDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, 0, 1, "")
	assert.NoError(t, err, "GetMyDeliveryTradingHistory should not error")
}

func TestGetDeliveryPositionCloseHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryPositionCloseHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetDeliveryPositionCloseHistory should not error")
}

func TestGetDeliveryLiquidationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, time.Now())
	assert.NoError(t, err, "GetDeliveryLiquidationHistory should not error")
}

func TestGetDeliverySettlementHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliverySettlementHistory(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0, time.Now())
	assert.NoError(t, err, "GetDeliverySettlementHistory should not error")
}

func TestGetDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     12,
			Contract: getPair(t, asset.DeliveryFutures),
		},
		Trigger: FuturesTrigger{
			Rule:      1,
			OrderType: "close-short-position",
			Price:     123400,
		},
	})
	assert.NoError(t, err, "GetDeliveryPriceTriggeredOrder should not error")
}

func TestGetDeliveryAllAutoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryAllAutoOrder(t.Context(), statusOpen, currency.USDT, getPair(t, asset.DeliveryFutures), 0, 1)
	assert.NoError(t, err, "GetDeliveryAllAutoOrder should not error")
}

func TestCancelAllDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.CancelAllDeliveryPriceTriggeredOrder(t.Context(), settle, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "CancelAllDeliveryPriceTriggeredOrder should not error")
}

func TestGetSingleDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSingleDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err, "GetSingleDeliveryPriceTriggeredOrder should not error")
}

func TestCancelDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err, "CancelDeliveryPriceTriggeredOrder should not error")
}

func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.EnableOrDisableDualMode(t.Context(), currency.BTC, true)
	assert.NoError(t, err, "EnableOrDisableDualMode should not error")
}

func TestRetrivePositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.RetrivePositionDetailInDualMode(t.Context(), settle, getPair(t, asset.Futures))
	assert.NoError(t, err, "RetrivePositionDetailInDualMode should not error")
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdatePositionMarginInDualMode(t.Context(), settle, getPair(t, asset.Futures), 0.001, "dual_long")
	assert.NoError(t, err, "UpdatePositionMarginInDualMode should not error")
}

func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdatePositionLeverageInDualMode(t.Context(), settle, getPair(t, asset.Futures), 0.001, 0.001)
	assert.NoError(t, err, "UpdatePositionLeverageInDualMode should not error")
}

func TestUpdatePositionRiskLimitInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdatePositionRiskLimitInDualMode(t.Context(), settle, getPair(t, asset.Futures), 10)
	assert.NoError(t, err, "UpdatePositionRiskLimitInDualMode should not error")
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.PlaceFuturesOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.Futures),
		Size:        6024,
		Iceberg:     0,
		Price:       "3765",
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      settle,
	})
	assert.NoError(t, err, "PlaceFuturesOrder should not error")
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesOrders(t.Context(), currency.NewPair(currency.BTC, currency.USD), statusOpen, "", currency.BTC, 0, 0, 1)
	assert.NoError(t, err, "GetFuturesOrders should not error")
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelMultipleFuturesOpenOrders(t.Context(), getPair(t, asset.Futures), "ask", currency.USDT)
	assert.NoError(t, err, "CancelMultipleFuturesOpenOrders should not error")
}

func TestGetSingleFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSingleFuturesPriceTriggeredOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err, "GetSingleFuturesPriceTriggeredOrder should not error")
}

func TestCancelFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelFuturesPriceTriggeredOrder(t.Context(), currency.USDT, "12345")
	assert.NoError(t, err, "CancelFuturesPriceTriggeredOrder should not error")
}

func TestPlaceBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.PlaceBatchFuturesOrders(t.Context(), currency.BTC, []ContractOrderCreateParams{
		{
			Contract:    getPair(t, asset.Futures),
			Size:        6024,
			Iceberg:     0,
			Price:       "3765",
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      settle,
		},
		{
			Contract:    currency.NewPair(currency.BTC, currency.USDT),
			Size:        232,
			Iceberg:     0,
			Price:       "376225",
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
	})
	assert.NoError(t, err, "PlaceBatchFuturesOrders should not error")
}

func TestGetSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err, "GetSingleFuturesOrder should not error")
}

func TestCancelSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelSingleFuturesOrder(t.Context(), currency.BTC, "12345")
	assert.NoError(t, err, "CancelSingleFuturesOrder should not error")
}

func TestAmendFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.AmendFuturesOrder(t.Context(), currency.BTC, "1234", AmendFuturesOrderParam{
		Price: 12345.990,
	})
	assert.NoError(t, err, "AmendFuturesOrder should not error")
}

func TestGetMyFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetMyFuturesTradingHistory(t.Context(), currency.BTC, "", "", getPair(t, asset.Futures), 0, 0, 0)
	assert.NoError(t, err, "GetMyFuturesTradingHistory should not error")
}

func TestGetFuturesPositionCloseHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.Futures), 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesPositionCloseHistory should not error")
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.Futures), 0, time.Time{})
	assert.NoError(t, err, "GetFuturesLiquidationHistory should not error")
}

func TestCountdownCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CountdownCancelOrders(t.Context(), currency.BTC, CountdownParams{
		Timeout: 8,
	})
	assert.NoError(t, err, "CountdownCancelOrders should not error")
}

func TestCreatePriceTriggeredFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.CreatePriceTriggeredFuturesOrder(t.Context(), settle, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     2,
			Contract: getPair(t, asset.Futures),
		},
		Trigger: FuturesTrigger{
			Rule:      1,
			OrderType: "close-short-position",
		},
	})
	assert.NoError(t, err, "CreatePriceTriggeredFuturesOrder should not error")
	_, err = g.CreatePriceTriggeredFuturesOrder(t.Context(), settle, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     1,
			Contract: getPair(t, asset.Futures),
		},
		Trigger: FuturesTrigger{
			Rule: 1,
		},
	})
	assert.NoError(t, err, "CreatePriceTriggeredFuturesOrder should not error")
}

func TestListAllFuturesAutoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.ListAllFuturesAutoOrders(t.Context(), statusOpen, currency.BTC, currency.EMPTYPAIR, 0, 0)
	assert.NoError(t, err, "ListAllFuturesAutoOrders should not error")
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.Futures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.CancelAllFuturesOpenOrders(t.Context(), settle, getPair(t, asset.Futures))
	assert.NoError(t, err, "CancelAllFuturesOpenOrders should not error")
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	r, err := g.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	assert.NotEmpty(t, r, "GetAllDeliveryContracts should return data")
	r, err = g.GetAllDeliveryContracts(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	// The test below will fail if support for BTC settlement is added. This is intentional, as it ensures we are alerted when it's time to reintroduce support
	if !assert.Empty(t, r, "GetAllDeliveryContracts should not return any data with unsupported settlement currency BTC") {
		t.Error("BTC settlement for delivery futures appears to be supported again by the API. Please raise an issue to reintroduce BTC support for this exchange")
	}
}

func TestGetDeliveryContract(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetDeliveryContract(t.Context(), settle, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "GetDeliveryContract should not error")
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", getPair(t, asset.DeliveryFutures), 0, false)
	assert.NoError(t, err, "GetDeliveryOrderbook should not error")
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetDeliveryTradingHistory(t.Context(), settle, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetDeliveryTradingHistory should not error")
}

func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetDeliveryFuturesCandlesticks(t.Context(), settle, getPair(t, asset.DeliveryFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "GetDeliveryFuturesCandlesticks should not error")
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.GetDeliveryFutureTickers(t.Context(), settle, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "GetDeliveryFutureTickers should not error")
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryInsuranceBalanceHistory(t.Context(), currency.BTC, 0)
	assert.NoError(t, err, "GetDeliveryInsuranceBalanceHistory should not error")
}

func TestQueryDeliveryFuturesAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryFuturesAccounts(t.Context(), currency.USDT)
	assert.NoError(t, err, "GetDeliveryFuturesAccounts should not error")
}

func TestGetDeliveryAccountBooks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryAccountBooks(t.Context(), currency.USDT, 0, time.Time{}, time.Now(), "dnw")
	assert.NoError(t, err, "GetDeliveryAccountBooks should not error")
}

func TestGetAllDeliveryPositionsOfUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetAllDeliveryPositionsOfUser(t.Context(), currency.USDT)
	assert.NoError(t, err, "GetAllDeliveryPositionsOfUser should not error")
}

func TestGetSingleDeliveryPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSingleDeliveryPosition(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "GetSingleDeliveryPosition should not error")
}

func TestUpdateDeliveryPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateDeliveryPositionMargin(t.Context(), currency.EMPTYCODE, 0.001, currency.Pair{})
	assert.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := getSettlementFromCurrency(getPair(t, asset.DeliveryFutures))
	require.NoError(t, err, "getSettlementFromCurrency must not error")
	_, err = g.UpdateDeliveryPositionMargin(t.Context(), settle, 0.001, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "UpdateDeliveryPositionMargin should not error")
}

func TestUpdateDeliveryPositionLeverage(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateDeliveryPositionLeverage(t.Context(), currency.EMPTYCODE, currency.Pair{}, 0.001)
	assert.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err = g.UpdateDeliveryPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 0.001)
	assert.NoError(t, err, "UpdateDeliveryPositionLeverage should not error")
}

func TestUpdateDeliveryPositionRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateDeliveryPositionRiskLimit(t.Context(), currency.EMPTYCODE, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errEmptyOrInvalidSettlementCurrency)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err = g.UpdateDeliveryPositionRiskLimit(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), 30)
	assert.NoError(t, err, "UpdateDeliveryPositionRiskLimit should not error")
}

func TestGetAllOptionsUnderlyings(t *testing.T) {
	t.Parallel()
	if _, err := g.GetAllOptionsUnderlyings(t.Context()); err != nil {
		t.Errorf("%s GetAllOptionsUnderlyings() error %v", g.Name, err)
	}
}

func TestGetExpirationTime(t *testing.T) {
	t.Parallel()
	if _, err := g.GetExpirationTime(t.Context(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetExpirationTime() error %v", g.Name, err)
	}
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	if _, err := g.GetAllContractOfUnderlyingWithinExpiryDate(t.Context(), "BTC_USDT", time.Time{}); err != nil {
		t.Errorf("%s GetAllContractOfUnderlyingWithinExpiryDate() error %v", g.Name, err)
	}
}

func TestGetOptionsSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsSpecifiedContractDetail(t.Context(), getPair(t, asset.Options)); err != nil {
		t.Errorf("%s GetOptionsSpecifiedContractDetail() error %v", g.Name, err)
	}
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSettlementHistory(t.Context(), "BTC_USDT", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetSettlementHistory() error %v", g.Name, err)
	}
}

func TestGetOptionsSpecifiedSettlementHistory(t *testing.T) {
	t.Parallel()
	underlying := "BTC_USDT"
	optionsSettlement, err := g.GetSettlementHistory(t.Context(), underlying, 0, 1, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	cp, err := currency.NewPairFromString(optionsSettlement[0].Contract)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetOptionsSpecifiedContractsSettlement(t.Context(), cp, underlying, optionsSettlement[0].Timestamp.Time().Unix()); err != nil {
		t.Errorf("%s GetOptionsSpecifiedContractsSettlement() error %s", g.Name, err)
	}
}

func TestGetSupportedFlashSwapCurrencies(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSupportedFlashSwapCurrencies(t.Context()); err != nil {
		t.Errorf("%s GetSupportedFlashSwapCurrencies() error %v", g.Name, err)
	}
}

const flashSwapOrderResponseJSON = `{"id": 54646,  "create_time": 1651116876378,  "update_time": 1651116876378,  "user_id": 11135567,  "sell_currency": "BTC",  "sell_amount": "0.01",  "buy_currency": "USDT",  "buy_amount": "10",  "price": "100",  "status": 1}`

func TestCreateFlashSwapOrder(t *testing.T) {
	t.Parallel()
	var response FlashSwapOrderResponse
	if err := json.Unmarshal([]byte(flashSwapOrderResponseJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to FlashSwapOrderResponse %v", g.Name, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateFlashSwapOrder(t.Context(), FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		BuyAmount:    34234,
		SellAmount:   34234,
	}); err != nil {
		t.Errorf("%s CreateFlashSwapOrder() error %v", g.Name, err)
	}
}

func TestGetAllFlashSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetAllFlashSwapOrders(t.Context(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0); err != nil {
		t.Errorf("%s GetAllFlashSwapOrders() error %v", g.Name, err)
	}
}

func TestGetSingleFlashSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleFlashSwapOrder(t.Context(), "1234"); err != nil {
		t.Errorf("%s GetSingleFlashSwapOrder() error %v", g.Name, err)
	}
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.InitiateFlashSwapOrderReview(t.Context(), FlashSwapOrderParams{
		PreviewID:    "1234",
		SellCurrency: currency.USDT,
		BuyCurrency:  currency.BTC,
		SellAmount:   100,
	}); err != nil {
		t.Errorf("%s InitiateFlashSwapOrderReview() error %v", g.Name, err)
	}
}

func TestGetMyOptionsSettlements(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMyOptionsSettlements(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}); err != nil {
		t.Errorf("%s GetMyOptionsSettlements() error %v", g.Name, err)
	}
}

func TestGetOptionAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOptionAccounts(t.Context()); err != nil {
		t.Errorf("%s GetOptionAccounts() error %v", g.Name, err)
	}
}

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetAccountChangingHistory(t.Context(), 0, 0, time.Time{}, time.Time{}, ""); err != nil {
		t.Errorf("%s GetAccountChangingHistory() error %v", g.Name, err)
	}
}

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersPositionSpecifiedUnderlying(t.Context(), ""); err != nil {
		t.Errorf("%s GetUsersPositionSpecifiedUnderlying() error %v", g.Name, err)
	}
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSpecifiedContractPosition(t.Context(), currency.EMPTYPAIR)
	if err != nil && !errors.Is(err, errInvalidOrMissingContractParam) {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, err)
	}
	_, err = g.GetSpecifiedContractPosition(t.Context(), getPair(t, asset.Options))
	if err != nil {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, err)
	}
}

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersLiquidationHistoryForSpecifiedUnderlying(t.Context(), "BTC_USDT", currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetUsersLiquidationHistoryForSpecifiedUnderlying() error %v", g.Name, err)
	}
}

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.PlaceOptionOrder(t.Context(), &OptionOrderParam{
		Contract:    getPair(t, asset.Options).String(),
		OrderSize:   -1,
		Iceberg:     0,
		Text:        "-",
		TimeInForce: "gtc",
		Price:       100,
	})
	if err != nil {
		t.Errorf("%s PlaceOptionOrder() error %v", g.Name, err)
	}
}

func TestGetOptionFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOptionFuturesOrders(t.Context(), currency.EMPTYPAIR, "", "", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetOptionFuturesOrders() error %v", g.Name, err)
	}
}

func TestCancelOptionOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelMultipleOptionOpenOrders(t.Context(), getPair(t, asset.Options), "", ""); err != nil {
		t.Errorf("%s CancelOptionOpenOrders() error %v", g.Name, err)
	}
}

func TestGetSingleOptionOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleOptionOrder(t.Context(), ""); err != nil && !errors.Is(errInvalidOrderID, err) {
		t.Errorf("%s GetSingleOptionorder() expecting %v, but found %v", g.Name, errInvalidOrderID, err)
	}
	if _, err := g.GetSingleOptionOrder(t.Context(), "1234"); err != nil {
		t.Errorf("%s GetSingleOptionOrder() error %v", g.Name, err)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelOptionSingleOrder(t.Context(), "1234"); err != nil {
		t.Errorf("%s CancelSingleOrder() error %v", g.Name, err)
	}
}

func TestGetMyOptionsTradingHistory(t *testing.T) {
	t.Parallel()

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetMyOptionsTradingHistory(t.Context(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{})
	require.NoError(t, err)
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.WithdrawCurrency(t.Context(), WithdrawalRequestParam{})
	if err != nil && !errors.Is(err, errInvalidAmount) {
		t.Errorf("%s WithdrawCurrency() expecting error %v, but found %v", g.Name, errInvalidAmount, err)
	}
	_, err = g.WithdrawCurrency(t.Context(), WithdrawalRequestParam{
		Currency: currency.BTC,
		Amount:   0.00000001,
		Chain:    "BTC",
		Address:  core.BitcoinDonationAddress,
	})
	if err != nil {
		t.Errorf("%s WithdrawCurrency() expecting error %v, but found %v", g.Name, errInvalidAmount, err)
	}
}

func TestCancelWithdrawalWithSpecifiedID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelWithdrawalWithSpecifiedID(t.Context(), "1234567"); err != nil {
		t.Errorf("%s CancelWithdrawalWithSpecifiedID() error %v", g.Name, err)
	}
}

func TestGetOptionsOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 9, true); err != nil {
		t.Errorf("%s GetOptionsFuturesOrderbooks() error %v", g.Name, err)
	}
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsTickers(t.Context(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetOptionsTickers() error %v", g.Name, err)
	}
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionUnderlyingTickers(t.Context(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetOptionUnderlyingTickers() error %v", g.Name, err)
	}
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionFuturesCandlesticks(t.Context(), getPair(t, asset.Options), 0, time.Now().Add(-time.Hour*10), time.Time{}, kline.ThirtyMin); err != nil {
		t.Error(err)
	}
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionFuturesMarkPriceCandlesticks(t.Context(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth); err != nil {
		t.Errorf("%s GetOptionFuturesMarkPriceCandlesticks() error %v", g.Name, err)
	}
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsTradeHistory(t.Context(), getPair(t, asset.Options), "C", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetOptionsTradeHistory() error %v", g.Name, err)
	}
}

// Sub-account endpoints

func TestCreateNewSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateNewSubAccount(t.Context(), SubAccountParams{
		LoginName: "Sub_Account_for_testing",
	}); err != nil {
		t.Errorf("%s CreateNewSubAccount() error %v", g.Name, err)
	}
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccounts(t.Context()); err != nil {
		t.Errorf("%s GetSubAccounts() error %v", g.Name, err)
	}
}

func TestGetSingleSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleSubAccount(t.Context(), "123423"); err != nil {
		t.Errorf("%s GetSingleSubAccount() error %v", g.Name, err)
	}
}

// Wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := g.FetchTradablePairs(t.Context(), asset.DeliveryFutures)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	if _, err = g.FetchTradablePairs(t.Context(), asset.Options); err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(t.Context(), asset.Futures)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	if _, err = g.FetchTradablePairs(t.Context(), asset.Margin); err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(t.Context(), asset.CrossMargin)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if err := g.UpdateTickers(t.Context(), asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(t.Context(), asset.Futures); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(t.Context(), asset.Spot); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(t.Context(), asset.Options); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(t.Context(), asset.CrossMargin); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(t.Context(), asset.Margin); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateOrderbook(t.Context(), getPair(t, asset.Spot), asset.Spot)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(t.Context(), getPair(t, asset.Margin), asset.Margin)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(t.Context(), getPair(t, asset.CrossMargin), asset.CrossMargin)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(t.Context(), getPair(t, asset.Futures), asset.Futures)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	if _, err = g.UpdateOrderbook(t.Context(), getPair(t, asset.DeliveryFutures), asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	if _, err = g.UpdateOrderbook(t.Context(), getPair(t, asset.Options), asset.Options); err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty); err != nil {
		t.Errorf("%s GetWithdrawalsHistory() error %v", g.Name, err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := g.GetRecentTrades(t.Context(), getPair(t, asset.Spot), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(t.Context(), getPair(t, asset.Margin), asset.Margin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(t.Context(), getPair(t, asset.CrossMargin), asset.CrossMargin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(t.Context(), getPair(t, asset.DeliveryFutures), asset.DeliveryFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(t.Context(), getPair(t, asset.Futures), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(t.Context(), getPair(t, asset.Options), asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.CrossMargin),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.CrossMargin,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.Spot),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.Options),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Options,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.DeliveryFutures),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.DeliveryFutures,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.Futures),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  g.Name,
		Pair:      getPair(t, asset.Margin),
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Margin,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	orderCancellation := &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	err := g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Margin
	err = g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.CrossMargin
	err = g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Options
	err = g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Futures
	err = g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.DeliveryFutures
	err = g.CancelOrder(t.Context(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Spot),
			AssetType:     asset.Spot,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Spot),
			AssetType:     asset.Spot,
		},
	})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Futures),
			AssetType:     asset.Futures,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Futures),
			AssetType:     asset.Futures,
		},
	})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.DeliveryFutures),
			AssetType:     asset.DeliveryFutures,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.DeliveryFutures),
			AssetType:     asset.DeliveryFutures,
		},
	})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Options),
			AssetType:     asset.Options,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Options),
			AssetType:     asset.Options,
		},
	})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Margin),
			AssetType:     asset.Margin,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          getPair(t, asset.Margin),
			AssetType:     asset.Margin,
		},
	})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	chains, err := g.GetAvailableTransferChains(t.Context(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
	for i := range chains {
		_, err = g.GetDepositAddress(t.Context(), currency.BTC, "", chains[i])
		if err != nil {
			t.Error("Test Fail - GetDepositAddress error", err)
		}
	}
}

func TestGetActiveOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	enabledPairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     enabledPairs[:2],
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	cp, err := currency.NewPairFromString("BTC_USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     []currency.Pair{cp},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     enabledPairs[:2],
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Margin,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     enabledPairs[:2],
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.CrossMargin,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{getPair(t, asset.Futures)},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{getPair(t, asset.DeliveryFutures)},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.DeliveryFutures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{getPair(t, asset.Options)},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Options,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	if _, err = g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.DeliveryFutures,
	}); !errors.Is(err, currency.ErrCurrencyPairsEmpty) {
		t.Errorf("%s GetActiveOrders() expecting error %v, but found %v", g.Name, currency.ErrCurrencyPairsEmpty, err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	multiOrderRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	enabledPairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	multiOrderRequest.Pairs = enabledPairs[:3]
	_, err = g.GetOrderHistory(t.Context(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.Futures
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	multiOrderRequest.Pairs = multiOrderRequest.Pairs[len(multiOrderRequest.Pairs)-4:]
	_, err = g.GetOrderHistory(t.Context(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.DeliveryFutures
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.DeliveryFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetOrderHistory(t.Context(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.Options
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetOrderHistory(t.Context(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 10)
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.Spot), asset.Spot, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.Margin), asset.Margin, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.CrossMargin), asset.CrossMargin, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.Futures), asset.Futures, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.DeliveryFutures), asset.DeliveryFutures, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.Options), asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandles() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
	if _, err := g.GetHistoricCandles(t.Context(), getPair(t, asset.Options), asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandles() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 5)
	_, err := g.GetHistoricCandlesExtended(t.Context(),
		getPair(t, asset.Spot), asset.Spot, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricCandlesExtended(t.Context(),
		getPair(t, asset.Margin), asset.Margin, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricCandlesExtended(t.Context(),
		getPair(t, asset.DeliveryFutures), asset.DeliveryFutures, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Futures), asset.Futures, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetHistoricCandlesExtended(t.Context(),
		getPair(t, asset.CrossMargin), asset.CrossMargin, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = g.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Options), asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandlesExtended() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
}

func TestGetAvailableTransferTrains(t *testing.T) {
	t.Parallel()
	_, err := g.GetAvailableTransferChains(t.Context(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestGetUnderlyingFromCurrencyPair(t *testing.T) {
	t.Parallel()
	if uly, err := g.GetUnderlyingFromCurrencyPair(currency.Pair{Delimiter: currency.UnderscoreDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDT_LLK")}); err != nil {
		t.Error(err)
	} else if !uly.Equal(currency.NewPair(currency.BTC, currency.USDT)) {
		t.Error("unexpected underlying")
	}
}

const wsTickerPushDataJSON = `{"time": 1606291803,	"channel": "spot.tickers",	"event": "update",	"result": {	  "currency_pair": "BTC_USDT",	  "last": "19106.55",	  "lowest_ask": "19108.71",	  "highest_bid": "19106.55",	  "change_percentage": "3.66",	  "base_volume": "2811.3042155865",	  "quote_volume": "53441606.52411221454674732293",	  "high_24h": "19417.74",	  "low_24h": "18434.21"	}}`

func TestWsTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket ticker push data error: %v", g.Name, err)
	}
}

const wsTradePushDataJSON = `{	"time": 1606292218,	"channel": "spot.trades",	"event": "update",	"result": {	  "id": 309143071,	  "create_time": 1606292218,	  "create_time_ms": "1606292218213.4578",	  "side": "sell",	  "currency_pair": "BTC_USDT",	  "amount": "16.4700000000",	  "price": "0.4705000000"}}`

func TestWsTradePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsTradePushDataJSON)); err != nil {
		t.Errorf("%s websocket trade push data error: %v", g.Name, err)
	}
}

const wsCandlestickPushDataJSON = `{"time": 1606292600,	"channel": "spot.candlesticks",	"event": "update",	"result": {	  "t": "1606292580",	  "v": "2362.32035",	  "c": "19128.1",	  "h": "19128.1",	  "l": "19128.1",	  "o": "19128.1","n": "1m_BTC_USDT"}}`

func TestWsCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsCandlestickPushDataJSON)); err != nil {
		t.Errorf("%s websocket candlestick push data error: %v", g.Name, err)
	}
}

const wsOrderbookTickerJSON = `{"time": 1606293275,	"channel": "spot.book_ticker",	"event": "update",	"result": {	  "t": 1606293275123,	  "u": 48733182,	  "s": "BTC_USDT",	  "b": "19177.79",	  "B": "0.0003341504",	  "a": "19179.38",	  "A": "0.09"	}}`

func TestWsOrderbookTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsOrderbookTickerJSON)); err != nil {
		t.Errorf("%s websocket orderbook push data error: %v", g.Name, err)
	}
}

const (
	wsOrderbookUpdatePushDataJSON   = `{"time": 1606294781,	"channel": "spot.order_book_update",	"event": "update",	"result": {	  "t": 1606294781123,	  "e": "depthUpdate",	  "E": 1606294781,"s": "BTC_USDT","U": 48776301,"u": 48776306,"b": [["19137.74","0.0001"],["19088.37","0"]],"a": [["19137.75","0.6135"]]	}}`
	wsOrderbookSnapshotPushDataJSON = `{"time":1606295412,"channel": "spot.order_book",	"event": "update",	"result": {	  "t": 1606295412123,	  "lastUpdateId": 48791820,	  "s": "BTC_USDT",	  "bids": [		[		  "19079.55",		  "0.0195"		],		[		  "19079.07",		  "0.7341"],["19076.23",		  "0.00011808"		],		[		  "19073.9",		  "0.105"		],		[		  "19068.83",		  "0.1009"		]	  ],	  "asks": [		[		  "19080.24",		  "0.1638"		],		[		  "19080.91","0.1366"],["19080.92","0.01"],["19081.29","0.01"],["19083.8","0.097"]]}}`
)

func TestWsOrderbookSnapshotPushData(t *testing.T) {
	t.Parallel()
	err := g.WsHandleSpotData(t.Context(), []byte(wsOrderbookSnapshotPushDataJSON))
	if err != nil {
		t.Errorf("%s websocket orderbook snapshot push data error: %v", g.Name, err)
	}
	if err = g.WsHandleSpotData(t.Context(), []byte(wsOrderbookUpdatePushDataJSON)); err != nil {
		t.Errorf("%s websocket orderbook update push data error: %v", g.Name, err)
	}
}

const wsSpotOrderPushDataJSON = `{"time": 1605175506,	"channel": "spot.orders",	"event": "update",	"result": [	  {		"id": "30784435",		"user": 123456,		"text": "t-abc",		"create_time": "1605175506",		"create_time_ms": "1605175506123",		"update_time": "1605175506",		"update_time_ms": "1605175506123",		"event": "put",		"currency_pair": "BTC_USDT",		"type": "limit",		"account": "spot",		"side": "sell",		"amount": "1",		"price": "10001",		"time_in_force": "gtc",		"left": "1",		"filled_total": "0",		"fee": "0",		"fee_currency": "USDT",		"point_fee": "0",		"gt_fee": "0",		"gt_discount": true,		"rebated_fee": "0",		"rebated_fee_currency": "USDT"}	]}`

func TestWsPushOrders(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsSpotOrderPushDataJSON)); err != nil {
		t.Errorf("%s websocket orders push data error: %v", g.Name, err)
	}
}

const wsUserTradePushDataJSON = `{"time": 1605176741,	"channel": "spot.usertrades",	"event": "update",	"result": [	  {		"id": 5736713,		"user_id": 1000001,		"order_id": "30784428",		"currency_pair": "BTC_USDT",		"create_time": 1605176741,		"create_time_ms": "1605176741123.456",		"side": "sell",		"amount": "1.00000000",		"role": "taker",		"price": "10000.00000000",		"fee": "0.00200000000000",		"point_fee": "0",		"gt_fee": "0",		"text": "apiv4"	  }	]}`

func TestWsUserTradesPushDataJSON(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsUserTradePushDataJSON)); err != nil {
		t.Errorf("%s websocket users trade push data error: %v", g.Name, err)
	}
}

const wsBalancesPushDataJSON = `{"time": 1605248616,	"channel": "spot.balances",	"event": "update",	"result": [	  {		"timestamp": "1605248616",		"timestamp_ms": "1605248616123",		"user": "1000001",		"currency": "USDT",		"change": "100",		"total": "1032951.325075926",		"available": "1022943.325075926"}	]}`

func TestBalancesPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsBalancesPushDataJSON)); err != nil {
		t.Errorf("%s websocket balances push data error: %v", g.Name, err)
	}
}

const wsMarginBalancePushDataJSON = `{"time": 1605248616,	"channel": "spot.funding_balances",	"event": "update",	"result": [	  {"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT","change": "100","freeze": "100","lent": "0"}	]}`

func TestMarginBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsMarginBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket margin balance push data error: %v", g.Name, err)
	}
}

const wsCrossMarginBalancePushDataJSON = `{"time": 1605248616,"channel": "spot.cross_balances","event": "update",	"result": [{"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT",	"change": "100","total": "1032951.325075926","available": "1022943.325075926"}]}`

func TestCrossMarginBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsCrossMarginBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket cross margin balance push data error: %v", g.Name, err)
	}
}

const wsCrossMarginBalanceLoan = `{	"time":1658289372,	"channel":"spot.cross_loan",	"event":"update",	"result":{	  "timestamp":1658289372338,	  "user":"1000001",	  "currency":"BTC",	  "change":"0.01",	  "total":"4.992341029566",	  "available":"0.078054772536",	  "borrowed":"0.01",	  "interest":"0.00001375"	}}`

func TestCrossMarginBalanceLoan(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleSpotData(t.Context(), []byte(wsCrossMarginBalanceLoan)); err != nil {
		t.Errorf("%s websocket cross margin loan push data error: %v", g.Name, err)
	}
}

const wsFuturesTickerPushDataJSON = `{"time": 1541659086,	"channel": "futures.tickers","event": "update",	"error": null,	"result": [	  {		"contract": "BTC_USD","last": "118.4","change_percentage": "0.77","funding_rate": "-0.000114","funding_rate_indicative": "0.01875","mark_price": "118.35","index_price": "118.36","total_size": "73648","volume_24h": "745487577","volume_24h_btc": "117",		"volume_24h_usd": "419950",		"quanto_base_rate": "",		"volume_24h_quote": "1665006","volume_24h_settle": "178","volume_24h_base": "5526","low_24h": "99.2","high_24h": "132.5"}	]}`

func TestFuturesTicker(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesTickerPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket push data error: %v", g.Name, err)
	}
}

const wsFuturesTradesPushDataJSON = `{"channel": "futures.trades","event": "update",	"time": 1541503698,	"result": [{"size": -108,"id": 27753479,"create_time": 1545136464,"create_time_ms": 1545136464123,"price": "96.4","contract": "BTC_USD"}]}`

func TestFuturesTrades(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesTradesPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket push data error: %v", g.Name, err)
	}
}

const (
	wsFuturesOrderbookTickerJSON = `{	"time": 1615366379,	"channel": "futures.book_ticker",	"event": "update",	"error": null,	"result": {	  "t": 1615366379123,	  "u": 2517661076,	  "s": "BTC_USD",	  "b": "54696.6",	  "B": 37000,	  "a": "54696.7",	  "A": 47061	}}`
)

func TestOrderbookData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesOrderbookTickerJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket orderbook ticker push data error: %v", g.Name, err)
	}
}

const wsFuturesOrderPushDataJSON = `{	"channel": "futures.orders",	"event": "update",	"time": 1541505434,	"result": [	  {		"contract": "BTC_USD",		"create_time": 1628736847,		"create_time_ms": 1628736847325,		"fill_price": 40000.4,		"finish_as": "filled",		"finish_time": 1628736848,		"finish_time_ms": 1628736848321,		"iceberg": 0,		"id": 4872460,		"is_close": false,		"is_liq": false,		"is_reduce_only": false,		"left": 0,		"mkfr": -0.00025,		"price": 40000.4,		"refr": 0,		"refu": 0,		"size": 1,		"status": "finished",		"text": "-",		"tif": "gtc",		"tkfr": 0.0005,		"user": "110xxxxx"	  }	]}`

func TestFuturesOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesOrderPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures order push data error: %v", g.Name, err)
	}
}

const wsFuturesUsertradesPushDataJSON = `{"time": 1543205083,	"channel": "futures.usertrades","event": "update",	"error": null,	"result": [{"id": "3335259","create_time": 1628736848,"create_time_ms": 1628736848321,"contract": "BTC_USD","order_id": "4872460","size": 1,"price": "40000.4","role": "maker","text": "api","fee": 0.0009290592,"point_fee": 0}]}`

func TestFuturesUserTrades(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesUsertradesPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures user trades push data error: %v", g.Name, err)
	}
}

const wsFuturesLiquidationPushDataJSON = `{"channel": "futures.liquidates",	"event": "update",	"time": 1541505434,	"result": [{"entry_price": 209,"fill_price": 215.1,"left": 0,"leverage": 0.0,"liq_price": 213,"margin": 0.007816722941,"mark_price": 213,"order_id": 4093362,"order_price": 215.1,"size": -124,"time": 1541486601,"time_ms": 1541486601123,"contract": "BTC_USD","user": "1040xxxx"}	]}`

func TestFuturesLiquidationPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesLiquidationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures liquidation push data error: %v", g.Name, err)
	}
}

const wsFuturesAutoDelevergesNotification = `{"channel": "futures.auto_deleverages",	"event": "update",	"time": 1541505434,	"result": [{"entry_price": 209,"fill_price": 215.1,"position_size": 10,"trade_size": 10,"time": 1541486601,"time_ms": 1541486601123,"contract": "BTC_USD","user": "1040"}	]}`

func TestFuturesAutoDeleverges(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesAutoDelevergesNotification), asset.Futures); err != nil {
		t.Errorf("%s websocket futures auto deleverge push data error: %v", g.Name, err)
	}
}

const wsFuturesPositionClosePushDataJSON = ` {"channel": "futures.position_closes",	"event": "update",	"time": 1541505434,	"result": [	  {		"contract": "BTC_USD",		"pnl": -0.000624354791,		"side": "long",		"text": "web",		"time": 1547198562,		"time_ms": 1547198562123,		"user": "211xxxx"	  }	]}`

func TestPositionClosePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesPositionClosePushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures position close push data error: %v", g.Name, err)
	}
}

const wsFuturesBalanceNotificationPushDataJSON = `{"channel": "futures.balances",	"event": "update",	"time": 1541505434,	"result": [	  {		"balance": 9.998739899488,		"change": -0.000002074115,		"text": "BTC_USD:3914424",		"time": 1547199246,		"time_ms": 1547199246123,		"type": "fee",		"user": "211xxx"	  }	]}`

func TestFuturesBalanceNotification(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesBalanceNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures balance notification push data error: %v", g.Name, err)
	}
}

const wsFuturesReduceRiskLimitNotificationPushDataJSON = `{"time": 1551858330,	"channel": "futures.reduce_risk_limits",	"event": "update",	"error": null,	"result": [	  {		"cancel_orders": 0,		"contract": "ETH_USD",		"leverage_max": 10,		"liq_price": 136.53,		"maintenance_rate": 0.09,		"risk_limit": 450,		"time": 1551858330,		"time_ms": 1551858330123,		"user": "20011"	  }	]}`

func TestFuturesReduceRiskLimitPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesReduceRiskLimitNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures reduce risk limit notification push data error: %v", g.Name, err)
	}
}

const wsFuturesPositionsNotificationPushDataJSON = `{"time": 1588212926,"channel": "futures.positions",	"event": "update",	"error": null,	"result": [	  {		"contract": "BTC_USD",		"cross_leverage_limit": 0,		"entry_price": 40000.36666661111,		"history_pnl": -0.000108569505,		"history_point": 0,		"last_close_pnl": -0.000050123368,"leverage": 0,"leverage_max": 100,"liq_price": 0.1,"maintenance_rate": 0.005,"margin": 49.999890611186,"mode": "single","realised_pnl": -1.25e-8,"realised_point": 0,"risk_limit": 100,"size": 3,"time": 1628736848,"time_ms": 1628736848321,"user": "110xxxxx"}]}`

func TestFuturesPositionsNotification(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesPositionsNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures positions change notification push data error: %v", g.Name, err)
	}
}

const wsFuturesAutoOrdersPushDataJSON = `{"time": 1596798126,"channel": "futures.autoorders",	"event": "update",	"error": null,	"result": [	  {		"user": 123456,		"trigger": {		  "strategy_type": 0,		  "price_type": 0,		  "price": "10000",		  "rule": 2,		  "expiration": 86400		},		"initial": {		  "contract": "BTC_USDT",		  "size": 10,		  "price": "10000",		  "tif": "gtc",		  "text": "web",		  "iceberg": 0,		  "is_close": false,		  "is_reduce_only": false		},		"id": 9256,		"trade_id": 0,		"status": "open",		"reason": "",		"create_time": 1596798126,		"name": "price_autoorders",		"is_stop_order": false,		"stop_trigger": {		  "rule": 0,		  "trigger_price": "",		  "order_price": ""		}	  }	]}`

func TestFuturesAutoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleFuturesData(t.Context(), []byte(wsFuturesAutoOrdersPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures auto orders push data error: %v", g.Name, err)
	}
}

// ******************************************** Options web-socket unit test funcs ********************

const optionsContractTickerPushDataJSON = `{"time": 1630576352,	"channel": "options.contract_tickers",	"event": "update",	"result": {    "name": "BTC_USDT-20211231-59800-P",    "last_price": "11349.5",    "mark_price": "11170.19",    "index_price": "",    "position_size": 993,    "bid1_price": "10611.7",    "bid1_size": 100,    "ask1_price": "11728.7",    "ask1_size": 100,    "vega": "34.8731",    "theta": "-72.80588",    "rho": "-28.53331",    "gamma": "0.00003",    "delta": "-0.78311",    "mark_iv": "0.86695",    "bid_iv": "0.65481",    "ask_iv": "0.88145",    "leverage": "3.5541112718136"	}}`

func TestOptionsContractTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsContractTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contract ticker push data failed with error %v", g.Name, err)
	}
}

const optionsUnderlyingTickerPushDataJSON = `{"time": 1630576352,	"channel": "options.ul_tickers",	"event": "update",	"result": {	   "trade_put": 800,	   "trade_call": 41700,	   "index_price": "50695.43",	   "name": "BTC_USDT"	}}`

func TestOptionsUnderlyingTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsUnderlyingTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket options underlying ticker push data error: %v", g.Name, err)
	}
}

const optionsContractTradesPushDataJSON = `{"time": 1630576356,	"channel": "options.trades",	"event": "update",	"result": [    {        "contract": "BTC_USDT-20211231-59800-C",        "create_time": 1639144526,        "id": 12279,        "price": 997.8,        "size": -100,        "create_time_ms": 1639144526597,        "underlying": "BTC_USDT"    }	]}`

func TestOptionsContractTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsContractTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket contract trades push data error: %v", g.Name, err)
	}
}

const optionsUnderlyingTradesPushDataJSON = `{"time": 1630576356,	"channel": "options.ul_trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C","create_time": 1639144526,"id": 12279,"price": 997.8,"size": -100,"create_time_ms": 1639144526597,"underlying": "BTC_USDT","is_call": true}	]}`

func TestOptionsUnderlyingTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsUnderlyingTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket underlying trades push data error: %v", g.Name, err)
	}
}

const optionsUnderlyingPricePushDataJSON = `{	"time": 1630576356,	"channel": "options.ul_price",	"event": "update",	"result": {	   "underlying": "BTC_USDT",	   "price": 49653.24,"time": 1639143988,"time_ms": 1639143988931}}`

func TestOptionsUnderlyingPricePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsUnderlyingPricePushDataJSON)); err != nil {
		t.Errorf("%s websocket underlying price push data error: %v", g.Name, err)
	}
}

const optionsMarkPricePushDataJSON = `{	"time": 1630576356,	"channel": "options.mark_price",	"event": "update",	"result": {    "contract": "BTC_USDT-20211231-59800-P",    "price": 11021.27,    "time": 1639143401,    "time_ms": 1639143401676}}`

func TestOptionsMarkPricePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsMarkPricePushDataJSON)); err != nil {
		t.Errorf("%s websocket mark price push data error: %v", g.Name, err)
	}
}

const optionsSettlementsPushDataJSON = `{	"time": 1630576356,	"channel": "options.settlements",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-55000-P",	   "orderbook_id": 2,	   "position_size": 1,	   "profit": 0.5,	   "settle_price": 70000,	   "strike_price": 65000,	   "tag": "WEEK",	   "trade_id": 1,	   "trade_size": 1,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`

func TestSettlementsPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsSettlementsPushDataJSON)); err != nil {
		t.Errorf("%s websocket options settlements push data error: %v", g.Name, err)
	}
}

const optionsContractPushDataJSON = `{"time": 1630576356,	"channel": "options.contracts",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-50000-P",	   "create_time": 1637917026,	   "expiration_time": 1638230400,	   "init_margin_high": 0.15,	   "init_margin_low": 0.1,	   "is_call": false,	   "maint_margin_base": 0.075,	   "maker_fee_rate": 0.0004,	   "mark_price_round": 0.1,	   "min_balance_short": 0.5,	   "min_order_margin": 0.1,	   "multiplier": 0.0001,	   "order_price_deviate": 0,	   "order_price_round": 0.1,	   "order_size_max": 1,	   "order_size_min": 10,	   "orders_limit": 100000,	   "ref_discount_rate": 0.1,	   "ref_rebate_rate": 0,	   "strike_price": 50000,	   "tag": "WEEK",	   "taker_fee_rate": 0.0004,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000}}`

func TestOptionsContractPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsContractPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contracts push data error: %v", g.Name, err)
	}
}

const (
	optionsContractCandlesticksPushDataJSON   = `{	"time": 1630650451,	"channel": "options.contract_candlesticks",	"event": "update",	"result": [   {       "t": 1639039260,       "v": 100,       "c": "1041.4",       "h": "1041.4",       "l": "1041.4",       "o": "1041.4",       "a": "0",       "n": "10s_BTC_USDT-20211231-59800-C"   }	]}`
	optionsUnderlyingCandlesticksPushDataJSON = `{	"time": 1630650451,	"channel": "options.ul_candlesticks",	"event": "update",	"result": [    {        "t": 1639039260,        "v": 100,        "c": "1041.4",        "h": "1041.4",        "l": "1041.4",        "o": "1041.4",        "a": "0",        "n": "10s_BTC_USDT"    }	]}`
)

func TestOptionsCandlesticksPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsContractCandlesticksPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contracts candlestick push data error: %v", g.Name, err)
	}
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsUnderlyingCandlesticksPushDataJSON)); err != nil {
		t.Errorf("%s websocket options underlying candlestick push data error: %v", g.Name, err)
	}
}

const (
	optionsOrderbookTickerPushDataJSON              = `{	"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {    "t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`
	optionsOrderbookUpdatePushDataJSON              = `{	"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {    "t": 1615366381417,    "s": "%s",    "U": 2517661101,    "u": 2517661113,    "b": [        {            "p": "54672.1",            "s": 95        },        {            "p": "54664.5",            "s": 58794        }    ],    "a": [        {            "p": "54743.6",            "s": 95        },        {            "p": "54742",            "s": 95        }    ]	}}`
	optionsOrderbookSnapshotPushDataJSON            = `{	"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {    "t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [        {            "p": "97.1",            "s": 2245        },		{            "p": "97.2",            "s": 2245        }    ],    "bids": [		{            "p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`
	optionsOrderbookSnapshotUpdateEventPushDataJSON = `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [	  {		"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`
)

func TestOptionsOrderbookPushData(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, g)
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), []byte(optionsOrderbookTickerPushDataJSON)))
	avail, err := g.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), fmt.Appendf(nil, optionsOrderbookUpdatePushDataJSON, avail[0].Upper().String())))
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), []byte(optionsOrderbookSnapshotPushDataJSON)))
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), []byte(optionsOrderbookSnapshotUpdateEventPushDataJSON)))
}

const optionsOrderPushDataJSON = `{"time": 1630654851,"channel": "options.orders",	"event": "update",	"result": [	   {		  "contract": "BTC_USDT-20211130-65000-C",		  "create_time": 1637897000,		  "fill_price": 0,		  "finish_as": "cancelled",		  "iceberg": 0,		  "id": 106,		  "is_close": false,		  "is_liq": false,		  "is_reduce_only": false,		  "left": -10,		  "mkfr": 0.0004,		  "price": 15000,		  "refr": 0,		  "refu": 0,		  "size": -10,		  "status": "finished",		  "text": "web",		  "tif": "gtc",		  "tkfr": 0.0004,		  "underlying": "BTC_USDT",		  "user": "9xxx",		  "time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsOrderPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orders push data error: %v", g.Name, err)
	}
}

const optionsUsersTradesPushDataJSON = `{	"time": 1639144214,	"channel": "options.usertrades",	"event": "update",	"result": [{"id": "1","underlying": "BTC_USDT","order": "557940","contract": "BTC_USDT-20211216-44800-C","create_time": 1639144214,"create_time_ms": 1639144214583,"price": "4999","role": "taker","size": -1}]}`

func TestOptionUserTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsUsersTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orders push data error: %v", g.Name, err)
	}
}

const optionsLiquidatesPushDataJSON = `{	"channel": "options.liquidates",	"event": "update",	"time": 1630654851,	"result": [	   {		  "user": "1xxxx",		  "init_margin": 1190,		  "maint_margin": 1042.5,		  "order_margin": 0,		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`

func TestOptionsLiquidatesPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsLiquidatesPushDataJSON)); err != nil {
		t.Errorf("%s websocket options liquidates push data error: %v", g.Name, err)
	}
}

const optionsSettlementPushDataJSON = `{	"channel": "options.user_settlements",	"event": "update",	"time": 1639051907,	"result": [{"contract": "BTC_USDT-20211130-65000-C","realised_pnl": -13.028,"settle_price": 70000,"settle_profit": 5,"size": 10,"strike_price": 65000,"underlying": "BTC_USDT","user": "9xxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsSettlementPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsSettlementPushDataJSON)); err != nil {
		t.Errorf("%s websocket options settlement push data error: %v", g.Name, err)
	}
}

const optionsPositionClosePushDataJSON = `{"channel": "options.position_closes",	"event": "update",	"time": 1630654851,	"result": [{"contract": "BTC_USDT-20211130-50000-C","pnl": -0.0056,"settle_size": 0,"side": "long","text": "web","underlying": "BTC_USDT","user": "11xxxxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsPositionClosePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsPositionClosePushDataJSON)); err != nil {
		t.Errorf("%s websocket options position close push data error: %v", g.Name, err)
	}
}

const optionsBalancePushDataJSON = `{	"channel": "options.balances",	"event": "update",	"time": 1630654851,	"result": [	   {		  "balance": 60.79009,"change": -0.5,"text": "BTC_USDT-20211130-55000-P","type": "set","user": "11xxxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket options balance push data error: %v", g.Name, err)
	}
}

const optionsPositionPushDataJSON = `{"time": 1630654851,	"channel": "options.positions",	"event": "update",	"error": null,	"result": [	   {		  "entry_price": 0,		  "realised_pnl": -13.028,		  "size": 0,		  "contract": "BTC_USDT-20211130-65000-C",		  "user": "9010",		  "time": 1639051907,		  "time_ms": 1639051907000}	]}`

func TestOptionsPositionPushData(t *testing.T) {
	t.Parallel()
	if err := g.WsHandleOptionsData(t.Context(), []byte(optionsPositionPushDataJSON)); err != nil {
		t.Errorf("%s websocket options position push data error: %v", g.Name, err)
	}
}

const (
	futuresOrderbookPushData       = `{"time": 1678468497, "time_ms": 1678468497232, "channel": "futures.order_book", "event": "all", "result": { "t": 1678468497168, "id": 4010394406, "contract": "BTC_USD", "asks": [ { "p": "19909", "s": 3100 }, { "p": "19909.1", "s": 5000 }, { "p": "19910", "s": 3100 }, { "p": "19914.4", "s": 4400 }, { "p": "19916.6", "s": 5000 }, { "p": "19917.2", "s": 8255 }, { "p": "19919.2", "s": 5000 }, { "p": "19920.3", "s": 11967 }, { "p": "19922.2", "s": 5000 }, { "p": "19924.2", "s": 5000 }, { "p": "19927.1", "s": 17129 }, { "p": "19927.2", "s": 5000 }, { "p": "19929", "s": 20864 }, { "p": "19929.3", "s": 5000 }, { "p": "19929.7", "s": 24683 }, { "p": "19930.3", "s": 750 }, { "p": "19931.4", "s": 5000 }, { "p": "19931.5", "s": 1 }, { "p": "19934.2", "s": 5000 }, { "p": "19935.4", "s": 1 } ], "bids": [ { "p": "19901.2", "s": 5000 }, { "p": "19900.3", "s": 3100 }, { "p": "19900.2", "s": 5000 }, { "p": "19899.3", "s": 2983 }, { "p": "19899.2", "s": 6035 }, { "p": "19897.2", "s": 5000 }, { "p": "19895.7", "s": 5984 }, { "p": "19895", "s": 5000 }, { "p": "19892.9", "s": 195 }, { "p": "19892.8", "s": 5000 }, { "p": "19889.4", "s": 5000 }, { "p": "19889", "s": 8800 }, { "p": "19888.5", "s": 11968 }, { "p": "19887.1", "s": 5000 }, { "p": "19886.4", "s": 24683 }, { "p": "19885.7", "s": 1 }, { "p": "19883.8", "s": 5000 }, { "p": "19880.2", "s": 5000 }, { "p": "19878.2", "s": 5000 }, { "p": "19876.8", "s": 1 } ] } }`
	futuresOrderbookUpdatePushData = `{"time": 1678469222, "time_ms": 1678469222982, "channel": "futures.order_book_update", "event": "update", "result": { "t": 1678469222617, "s": "BTC_USD", "U": 4010424331, "u": 4010424361, "b": [ { "p": "19860.7", "s": 5984 }, { "p": "19858.6", "s": 5000 }, { "p": "19845.4", "s": 20864 }, { "p": "19859.1", "s": 0 }, { "p": "19862.5", "s": 0 }, { "p": "19358", "s": 0 }, { "p": "19864.5", "s": 5000 }, { "p": "19840.7", "s": 0 }, { "p": "19863.6", "s": 3100 }, { "p": "19839.3", "s": 0 }, { "p": "19851.5", "s": 8800 }, { "p": "19720", "s": 0 }, { "p": "19333", "s": 0 }, { "p": "19852.7", "s": 5000 }, { "p": "19861.5", "s": 0 }, { "p": "19860.6", "s": 3100 }, { "p": "19833.6", "s": 0 }, { "p": "19360", "s": 0 }, { "p": "19863.5", "s": 5000 }, { "p": "19736.9", "s": 0 }, { "p": "19838.5", "s": 0 }, { "p": "19841.3", "s": 0 }, { "p": "19858.1", "s": 3100 }, { "p": "19710.9", "s": 0 }, { "p": "19342", "s": 0 }, { "p": "19852.1", "s": 11967 }, { "p": "19343", "s": 0 }, { "p": "19705", "s": 0 }, { "p": "19836.5", "s": 0 }, { "p": "19862.6", "s": 3100 }, { "p": "19729.6", "s": 0 }, { "p": "19849.9", "s": 5000 } ], "a": [ { "p": "19900.5", "s": 0 }, { "p": "19883.1", "s": 11967 }, { "p": "19910.9", "s": 0 }, { "p": "19897.7", "s": 5000 }, { "p": "19875.9", "s": 5984 }, { "p": "19899.6", "s": 0 }, { "p": "19878", "s": 4400 }, { "p": "19877.6", "s": 0 }, { "p": "19889.5", "s": 5000 }, { "p": "19875.5", "s": 3100 }, { "p": "19875.3", "s": 0 }, { "p": "19878.5", "s": 0 }, { "p": "19895.2", "s": 0 }, { "p": "20284.6", "s": 0 }, { "p": "19880.7", "s": 5000 }, { "p": "19875.4", "s": 0 }, { "p": "19985.8", "s": 0 }, { "p": "19887.1", "s": 5000 }, { "p": "19896", "s": 1 }, { "p": "19869.3", "s": 0 }, { "p": "19900", "s": 0 }, { "p": "19875.6", "s": 5000 }, { "p": "19980.6", "s": 0 }, { "p": "19885.1", "s": 5000 }, { "p": "19877.7", "s": 5000 }, { "p": "20000", "s": 0 }, { "p": "19892.2", "s": 8255 }, { "p": "19886.8", "s": 0 }, { "p": "20257.4", "s": 0 }, { "p": "20280", "s": 0 }, { "p": "20002.5", "s": 0 }, { "p": "20263.1", "s": 0 }, { "p": "19900.2", "s": 0 } ] } }`
)

func TestFuturesOrderbookPushData(t *testing.T) {
	t.Parallel()
	err := g.WsHandleFuturesData(t.Context(), []byte(futuresOrderbookPushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	err = g.WsHandleFuturesData(t.Context(), []byte(futuresOrderbookUpdatePushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

const futuresCandlesticksPushData = `{"time": 1678469467, "time_ms": 1678469467981, "channel": "futures.candlesticks", "event": "update", "result": [ { "t": 1678469460, "v": 0, "c": "19896", "h": "19896", "l": "19896", "o": "19896", "n": "1m_BTC_USD" } ] }`

func TestFuturesCandlestickPushData(t *testing.T) {
	t.Parallel()
	err := g.WsHandleFuturesData(t.Context(), []byte(futuresCandlesticksPushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateSubscriptionsSpot(t *testing.T) {
	t.Parallel()

	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Test instance Setup must not error")

	g.Websocket.SetCanUseAuthenticatedEndpoints(true)
	g.Features.Subscriptions = append(g.Features.Subscriptions, &subscription.Subscription{
		Enabled: true, Channel: spotOrderbookChannel, Asset: asset.Spot, Interval: kline.ThousandMilliseconds, Levels: 5,
	})
	subs, err := g.generateSubscriptionsSpot()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	assets := slices.DeleteFunc(g.GetAssetTypes(true), func(a asset.Item) bool { return !g.IsAssetWebsocketSupported(a) })
	for _, s := range g.Features.Subscriptions {
		for _, a := range assets {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := g.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs %s must not error", a)
			pairs = common.SortStrings(pairs).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"})
			s := s.Clone() //nolint:govet // Intentional lexical scope shadow
			s.Asset = a
			if singleSymbolChannel(channelName(s)) {
				for i := range pairs {
					s := s.Clone() //nolint:govet // Intentional lexical scope shadow
					switch s.Channel {
					case subscription.CandlesChannel:
						s.QualifiedChannel = "5m," + pairs[i].String()
					case subscription.OrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",100ms"
					case spotOrderbookChannel:
						s.QualifiedChannel = pairs[i].String() + ",5,1000ms"
					}
					s.Pairs = pairs[i : i+1]
					exp = append(exp, s)
				}
			} else {
				s.Pairs = pairs
				s.QualifiedChannel = pairs.Join()
				exp = append(exp, s)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	subs, err := g.Features.Subscriptions.ExpandTemplates(g)
	require.NoError(t, err, "ExpandTemplates must not error")
	g.Features.Subscriptions = subscription.List{}
	err = g.Subscribe(t.Context(), &DummyConnection{}, subs)
	require.NoError(t, err, "Subscribe must not error")
}

func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := g.GenerateDeliveryFuturesDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
}

func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	subs, err := g.GenerateFuturesDefaultSubscriptions(currency.USDT)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	subs, err = g.GenerateFuturesDefaultSubscriptions(currency.BTC)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	_, err = g.GenerateFuturesDefaultSubscriptions(currency.TABOO)
	require.Error(t, err)
}

func TestGenerateOptionsDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := g.GenerateOptionsDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
}

func TestCreateAPIKeysOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateAPIKeysOfSubAccount(t.Context(), CreateAPIKeySubAccountParams{
		SubAccountUserID: 12345,
		Body: &SubAccountKey{
			APIKeyName: "12312mnfsndfsfjsdklfjsdlkfj",
			Permissions: []APIV4KeyPerm{
				{
					PermissionName: "wallet",
					ReadOnly:       false,
				},
				{
					PermissionName: "spot",
					ReadOnly:       false,
				},
				{
					PermissionName: "futures",
					ReadOnly:       false,
				},
				{
					PermissionName: "delivery",
					ReadOnly:       false,
				},
				{
					PermissionName: "earn",
					ReadOnly:       false,
				},
				{
					PermissionName: "options",
					ReadOnly:       false,
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}
}

func TestListAllAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetAllAPIKeyOfSubAccount(t.Context(), 1234)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.UpdateAPIKeyOfSubAccount(t.Context(), apiKey, CreateAPIKeySubAccountParams{
		SubAccountUserID: 12345,
		Body: &SubAccountKey{
			APIKeyName: "12312mnfsndfsfjsdklfjsdlkfj",
			Permissions: []APIV4KeyPerm{
				{
					PermissionName: "wallet",
					ReadOnly:       false,
				},
				{
					PermissionName: "spot",
					ReadOnly:       false,
				},
				{
					PermissionName: "futures",
					ReadOnly:       false,
				},
				{
					PermissionName: "delivery",
					ReadOnly:       false,
				},
				{
					PermissionName: "earn",
					ReadOnly:       false,
				},
				{
					PermissionName: "options",
					ReadOnly:       false,
				},
			},
		},
	}); err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetAPIKeyOfSubAccount(t.Context(), 1234, "target_api_key")
	if err != nil {
		t.Error(err)
	}
}

func TestLockSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if err := g.LockSubAccount(t.Context(), 1234); err != nil {
		t.Error(err)
	}
}

func TestUnlockSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if err := g.UnlockSubAccount(t.Context(), 1234); err != nil {
		t.Error(err)
	}
}

func TestGetSettlementFromCurrency(t *testing.T) {
	t.Parallel()
	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Setup must not error")
	for _, assetType := range []asset.Item{asset.Futures, asset.DeliveryFutures, asset.Options} {
		availPairs, err := g.GetAvailablePairs(assetType)
		require.NoErrorf(t, err, "GetAvailablePairs for asset %s must not error", assetType)
		for i, pair := range availPairs {
			t.Run(strconv.Itoa(i)+":"+assetType.String(), func(t *testing.T) {
				t.Parallel()
				_, err := getSettlementFromCurrency(pair)
				assert.NoErrorf(t, err, "getSettlementFromCurrency should not error for pair %s and asset %s", pair, assetType)
			})
		}
	}
}

func TestParseGateioMilliSecTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731098
	timeWhenTestingString := `"1684981731098"` // Normal string
	integerJSON := `{"number": 1684981731098}`
	float64JSON := `{"number": 1684981731.098}`

	time := time.UnixMilli(timeWhenTesting)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	if err != nil {
		t.Fatal(err)
	}
	if !in.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", in.Time(), time)
	}
	inInteger := struct {
		Number types.Time `json:"number"`
	}{}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	if err != nil {
		t.Fatal(err)
	}
	if !inInteger.Number.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", inInteger.Number.Time(), time)
	}

	inFloat64 := struct {
		Number types.Time `json:"number"`
	}{}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	if err != nil {
		t.Fatal(err)
	}
	if !inFloat64.Number.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", inFloat64.Number.Time(), time)
	}
}

func TestParseTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731
	timeWhenTestingString := `"1684981731"`
	integerJSON := `{"number": 1684981731}`
	float64JSON := `{"number": 1684981731.234}`
	timeWhenTestingStringMicroSecond := `"1691122380942.173000"`

	whenTime := time.Unix(timeWhenTesting, 0)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	if err != nil {
		t.Fatal(err)
	}
	if !in.Time().Equal(whenTime) {
		t.Fatalf("found %v, but expected %v", in.Time(), whenTime)
	}
	inInteger := struct {
		Number types.Time `json:"number"`
	}{}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	if err != nil {
		t.Fatal(err)
	}
	if !inInteger.Number.Time().Equal(whenTime) {
		t.Fatalf("found %v, but expected %v", inInteger.Number.Time(), whenTime)
	}

	inFloat64 := struct {
		Number types.Time `json:"number"`
	}{}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	if err != nil {
		t.Fatal(err)
	}
	msTime := time.UnixMilli(1684981731234)
	if !inFloat64.Number.Time().Equal(time.UnixMilli(1684981731234)) {
		t.Fatalf("found %v, but expected %v", inFloat64.Number.Time(), msTime)
	}

	var microSeconds types.Time
	err = json.Unmarshal([]byte(timeWhenTestingStringMicroSecond), &microSeconds)
	if err != nil {
		t.Fatal(err)
	}
	if !microSeconds.Time().Equal(time.UnixMicro(1691122380942173)) {
		t.Fatalf("found %v, but expected %v", microSeconds.Time(), time.UnixMicro(1691122380942173))
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, g)

	err := g.UpdateOrderExecutionLimits(t.Context(), 1336)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received %v, expected %v", err, asset.ErrNotSupported)
	}

	err = g.UpdateOrderExecutionLimits(t.Context(), asset.Options)
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received %v, expected %v", err, common.ErrNotYetImplemented)
	}

	err = g.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	avail, err := g.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	for i := range avail {
		mm, err := g.GetOrderExecutionLimits(asset.Spot, avail[i])
		if err != nil {
			t.Fatal(err)
		}

		if mm == (order.MinMaxLevel{}) {
			t.Fatal("expected a value")
		}

		if mm.MinimumBaseAmount <= 0 {
			t.Fatalf("MinimumBaseAmount expected 0 but received %v for %v", mm.MinimumBaseAmount, avail[i])
		}

		// 1INCH_TRY no minimum quote or base values are returned.

		if mm.QuoteStepIncrementSize <= 0 {
			t.Fatalf("QuoteStepIncrementSize expected 0 but received %v for %v", mm.QuoteStepIncrementSize, avail[i])
		}

		if mm.AmountStepIncrementSize <= 0 {
			t.Fatalf("AmountStepIncrementSize expected 0 but received %v for %v", mm.AmountStepIncrementSize, avail[i])
		}
	}
}

func TestForceFileStandard(t *testing.T) {
	t.Parallel()
	err := sharedtestvalues.ForceFileStandard(t, sharedtestvalues.EmptyStringPotentialPattern)
	if err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.Fatal("Please use types.Number type instead of `float64` and remove `,string` as strings can be empty in unmarshal process. Then call the Float64() method.")
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	_, err = g.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	exp, err := g.GetAllDeliveryContracts(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAllDeliveryContracts must not error")
	c, err := g.GetFuturesContractDetails(t.Context(), asset.DeliveryFutures)
	require.NoError(t, err, "GetFuturesContractDetails must not error for DeliveryFutures")
	assert.Equal(t, len(exp), len(c), "GetFuturesContractDetails should return same number of Delivery contracts as exist")

	c, err = g.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error for DeliveryFutures")
	assert.NotEmpty(t, c, "GetFuturesContractDetails should return same number of Future contracts as exist")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT),
		IncludePredictedRate: true,
	})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
	_, err = g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.BTC, currency.USD),
	})
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetHistoricalFundingRates(t.Context(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: %v, expected: %v", err, common.ErrNilPointer)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, expected: %v", err, asset.ErrNotSupported)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.Futures,
	})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v, expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.ENJ, currency.USDT),
	})
	if !errors.Is(err, fundingrate.ErrPaymentCurrencyCannotBeEmpty) {
		t.Fatalf("received: %v, expected: %v", err, fundingrate.ErrPaymentCurrencyCannotBeEmpty)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.Futures,
		Pair:                 currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency:      currency.USDT,
		IncludePayments:      true,
		IncludePredictedRate: true,
	})
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.Futures,
		Pair:                 currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency:      currency.USDT,
		IncludePredictedRate: true,
	})
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.Futures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(time.Hour * 16),
		EndDate:         time.Now(),
	})
	if !errors.Is(err, common.ErrStartAfterEnd) {
		t.Fatalf("received: %v, expected: %v", err, common.ErrStartAfterEnd)
	}

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.Futures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 8008),
		EndDate:         time.Now(),
	})
	if !errors.Is(err, fundingrate.ErrFundingRateOutsideLimits) {
		t.Fatalf("received: %v, expected: %v", err, fundingrate.ErrFundingRateOutsideLimits)
	}

	history, err := g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.Futures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	assert.NotEmpty(t, history, "should return values")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := g.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest should error correctly")

	var resp []futures.OpenInterest
	for _, a := range []asset.Item{asset.Futures, asset.DeliveryFutures} {
		p := getPair(t, a)
		resp, err = g.GetOpenInterest(t.Context(), key.PairAsset{
			Base:  p.Base.Item,
			Quote: p.Quote.Item,
			Asset: a,
		})
		assert.NoErrorf(t, err, "GetOpenInterest should not error for %s asset", a)
		assert.Lenf(t, resp, 1, "GetOpenInterest should return 1 item for %s asset", a)
	}

	resp, err = g.GetOpenInterest(t.Context())
	assert.NoError(t, err, "GetOpenInterest should not error")
	assert.NotEmpty(t, resp, "GetOpenInterest should return some items")
}

var pairMap = map[asset.Item]currency.Pair{
	asset.Spot: currency.NewPairWithDelimiter("BTC", "USDT", "_"),
}

var pairsGuard sync.RWMutex

func getPair(tb testing.TB, a asset.Item) currency.Pair {
	tb.Helper()
	pairsGuard.RLock()
	p, ok := pairMap[a]
	pairsGuard.RUnlock()
	if ok {
		return p
	}
	pairsGuard.Lock()
	defer pairsGuard.Unlock()
	p, ok = pairMap[a] // Protect Race if we blocked on Lock and another RW populated
	if ok {
		return p
	}

	testexch.UpdatePairsOnce(tb, g)
	enabledPairs, err := g.GetEnabledPairs(a)
	assert.NoErrorf(tb, err, "%s GetEnabledPairs should not error", a)
	if !assert.NotEmpty(tb, enabledPairs, "%s GetEnabledPairs should not be empty", a) {
		tb.Fatalf("No pair available for asset %s", a)
		return currency.EMPTYPAIR
	}
	pairMap[a] = enabledPairs[0]

	return pairMap[a]
}

func TestGetClientOrderIDFromText(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getClientOrderIDFromText("api"), "should not return anything")
	assert.Equal(t, "t-123", getClientOrderIDFromText("t-123"), "should return t-123")
}

func TestGetTypeFromTimeInForce(t *testing.T) {
	t.Parallel()
	typeResp, postOnly := getTypeFromTimeInForce("gtc")
	assert.Equal(t, order.Limit, typeResp, "should be a limit order")
	assert.False(t, postOnly, "should return false")

	typeResp, postOnly = getTypeFromTimeInForce("ioc")
	assert.Equal(t, order.Market, typeResp, "should be market order")
	assert.False(t, postOnly, "should return false")

	typeResp, postOnly = getTypeFromTimeInForce("poc")
	assert.Equal(t, order.Limit, typeResp, "should be limit order")
	assert.True(t, postOnly, "should return true")

	typeResp, postOnly = getTypeFromTimeInForce("fok")
	assert.Equal(t, order.Market, typeResp, "should be market order")
	assert.False(t, postOnly, "should return false")
}

func TestGetSideAndAmountFromSize(t *testing.T) {
	t.Parallel()
	side, amount, remaining := getSideAndAmountFromSize(1, 1)
	assert.Equal(t, order.Long, side, "should be a buy order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")

	side, amount, remaining = getSideAndAmountFromSize(-1, -1)
	assert.Equal(t, order.Short, side, "should be a sell order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")
}

func TestGetFutureOrderSize(t *testing.T) {
	t.Parallel()
	_, err := getFutureOrderSize(&order.Submit{Side: order.CouldNotCloseShort, Amount: 1})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	ret, err := getFutureOrderSize(&order.Submit{Side: order.Buy, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, 1.0, ret)

	ret, err = getFutureOrderSize(&order.Submit{Side: order.Sell, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, -1.0, ret)
}

func TestGetTimeInForce(t *testing.T) {
	t.Parallel()

	_, err := getTimeInForce(&order.Submit{Type: order.Market, PostOnly: true})
	assert.ErrorIs(t, err, errPostOnlyOrderTypeUnsupported)

	ret, err := getTimeInForce(&order.Submit{Type: order.Market})
	require.NoError(t, err)
	assert.Equal(t, "ioc", ret)

	ret, err = getTimeInForce(&order.Submit{Type: order.Limit, PostOnly: true})
	require.NoError(t, err)
	assert.Equal(t, "poc", ret)

	ret, err = getTimeInForce(&order.Submit{Type: order.Limit})
	require.NoError(t, err)
	assert.Equal(t, "gtc", ret)

	ret, err = getTimeInForce(&order.Submit{Type: order.Market, FillOrKill: true})
	require.NoError(t, err)
	assert.Equal(t, "fok", ret)
}

func TestProcessFuturesOrdersPushData(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		incoming string
		status   order.Status
	}{
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"open","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Open},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"filled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Filled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"cancelled","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"liquidated","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Liquidated},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"ioc","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"auto_deleveraged","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.AutoDeleverage},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"reduce_only","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Cancelled},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"position_closed","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.Closed},
		{`{"channel":"futures.orders","event":"update","time":1541505434,"time_ms":1541505434123,"result":[{"contract":"BTC_USD","create_time":1628736847,"create_time_ms":1628736847325,"fill_price":40000.4,"finish_as":"stp","finish_time":1628736848,"finish_time_ms":1628736848321,"iceberg":0,"id":4872460,"is_close":false,"is_liq":false,"is_reduce_only":false,"left":0,"mkfr":-0.00025,"price":40000.4,"refr":0,"refu":0,"size":1,"status":"finished","text":"-","tif":"gtc","tkfr":0.0005,"user":"110xxxxx"}]}`, order.STP},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			processed, err := g.processFuturesOrdersPushData([]byte(tc.incoming), asset.Futures)
			require.NoError(t, err)
			require.NotNil(t, processed)
			for i := range processed {
				assert.Equal(t, tc.status.String(), processed[i].Status.String())
			}
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, g)
	for _, a := range g.GetAssetTypes(false) {
		pairs, err := g.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := g.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		if a == asset.Options {
			require.ErrorIs(t, err, asset.ErrNotSupported)
		} else {
			require.NoError(t, err)
			assert.NotEmpty(t, resp)
		}
	}
}

func TestGetUnifiedAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	// Requires unified account to be enabled for this to function.
	payload, err := g.GetUnifiedAccount(t.Context(), currency.EMPTYCODE)
	require.NoError(t, err)
	require.NotEmpty(t, payload)
}

func TestGenerateWebsocketMessageID(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, g.GenerateWebsocketMessageID(false))
}

type DummyConnection struct{ websocket.Connection }

func (d *DummyConnection) GenerateMessageID(bool) int64 { return 1337 }
func (d *DummyConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return []byte(`{"time":1726121320,"time_ms":1726121320745,"id":1,"conn_id":"f903779a148987ca","trace_id":"d8ee37cd14347e4ed298d44e69aedaa7","channel":"spot.tickers","event":"subscribe","payload":["BRETT_USDT"],"result":{"status":"success"},"requestId":"d8ee37cd14347e4ed298d44e69aedaa7"}`), nil
}

func TestHandleSubscriptions(t *testing.T) {
	t.Parallel()

	subs := subscription.List{{Channel: subscription.OrderbookChannel}}

	err := g.handleSubscription(t.Context(), &DummyConnection{}, subscribeEvent, subs, func(context.Context, websocket.Connection, string, subscription.List) ([]WsInput, error) {
		return []WsInput{{}}, nil
	})
	require.NoError(t, err)

	err = g.handleSubscription(t.Context(), &DummyConnection{}, unsubscribeEvent, subs, func(context.Context, websocket.Connection, string, subscription.List) ([]WsInput, error) {
		return []WsInput{{}}, nil
	})
	require.NoError(t, err)
}

func TestParseWSHeader(t *testing.T) {
	in := []string{
		`{"time":1726121320,"time_ms":1726121320745,"id":1,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time_ms":1726121320746,"id":2,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
		`{"time":1726121321,"id":3,"channel":"spot.tickers","event":"subscribe","result":{"status":"success"},"request_id":"a4"}`,
	}
	for _, i := range in {
		h, err := parseWSHeader([]byte(i))
		require.NoError(t, err)
		require.NotEmpty(t, h.ID)
		assert.Equal(t, "a4", h.RequestID)
		assert.Equal(t, "spot.tickers", h.Channel)
		assert.Equal(t, "subscribe", h.Event)
		assert.NotEmpty(t, h.Result)
		switch h.ID {
		case 1:
			assert.Equal(t, int64(1726121320745), h.Time.UnixMilli())
		case 2:
			assert.Equal(t, int64(1726121320746), h.Time.UnixMilli())
		case 3:
			assert.Equal(t, int64(1726121321), h.Time.Unix())
		}
	}
}

func TestDeriveSpotWebsocketOrderResponse(t *testing.T) {
	t.Parallel()

	var resp *WebsocketOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`), &resp), "unmarshal must not error")

	got, err := g.deriveSpotWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             g.Name,
		OrderID:              "766075454481",
		AssetType:            asset.Spot,
		Pair:                 currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1735720637181634009",
		Date:                 time.UnixMilli(1735720637188),
		LastUpdated:          time.UnixMilli(1735720637188),
		Amount:               0.0001,
		AverageExecutedPrice: 93503.3,
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		ImmediateOrCancel:    true,
		Cost:                 0.0001,
		Purchased:            9.35033,
	}, got)
}

func TestDeriveSpotWebsocketOrderResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of spot orders",
			orders: [][]byte{
				[]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`),
				[]byte(`{"left":"0.000008","update_time":"1735720637","amount":"9.99152","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"HNS_USDT","type":"market","account":"spot","side":"buy","amend_text":"-","text":"t-1735720637126962151","status":"closed","iceberg":"0","avg_deal_price":"0.01224","filled_total":"9.991512","id":"766075454188","fill_price":"9.991512","update_time_ms":1735720637142,"create_time_ms":1735720637142}`),
				[]byte(`{"left":"0","update_time":"1735778597","amount":"200","create_time":"1735778597","price":"0.03673","finish_as":"filled","time_in_force":"fok","currency_pair":"REX_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1364","status":"closed","iceberg":"0","avg_deal_price":"0.03673","filled_total":"7.346","id":"766488882062","fill_price":"7.346","update_time_ms":1735778597363,"create_time_ms":1735778597363}`),
				[]byte(`{"left":"0.0003","update_time":"1735780321","amount":"0.0003","create_time":"1735780321","price":"20000","finish_as":"open","time_in_force":"poc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1735780321603944400","status":"open","iceberg":"0","filled_total":"0","id":"766504537761","fill_price":"0","update_time_ms":1735780321729,"create_time_ms":1735780321729}`),
				[]byte(`{"left":"1","update_time":"1735784755","amount":"1","create_time":"1735784755","price":"100","finish_as":"open","time_in_force":"gtc","currency_pair":"GT_USDT","type":"limit","account":"spot","side":"sell","amend_text":"-","text":"t-1735784754905434100","status":"open","iceberg":"0","filled_total":"0","id":"766536556747","fill_price":"0","update_time_ms":1735784755068,"create_time_ms":1735784755068}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             g.Name,
					OrderID:              "766075454481",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637181634009",
					Date:                 time.UnixMilli(1735720637188),
					LastUpdated:          time.UnixMilli(1735720637188),
					Amount:               0.0001,
					AverageExecutedPrice: 93503.3,
					Type:                 order.Market,
					Side:                 order.Sell,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
					Cost:                 0.0001,
					Purchased:            9.35033,
				},
				{
					Exchange:             g.Name,
					OrderID:              "766075454188",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.HNS, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637126962151",
					Date:                 time.UnixMilli(1735720637142),
					LastUpdated:          time.UnixMilli(1735720637142),
					RemainingAmount:      0.000008,
					Amount:               9.99152,
					AverageExecutedPrice: 0.01224,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
					Cost:                 9.991512,
					Purchased:            816.3,
				},
				{
					Exchange:             g.Name,
					OrderID:              "766488882062",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1364",
					Date:                 time.UnixMilli(1735778597363),
					LastUpdated:          time.UnixMilli(1735778597363),
					Amount:               200,
					Price:                0.03673,
					AverageExecutedPrice: 0.03673,
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.Filled,
					FillOrKill:           true,
					Cost:                 7.346,
					Purchased:            200,
				},
				{
					Exchange:        g.Name,
					OrderID:         "766504537761",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735780321603944400",
					Date:            time.UnixMilli(1735780321729),
					LastUpdated:     time.UnixMilli(1735780321729),
					RemainingAmount: 0.0003,
					Amount:          0.0003,
					Price:           20000,
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.Open,
					PostOnly:        true,
				},
				{
					Exchange:        g.Name,
					OrderID:         "766536556747",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.NewCode("GT"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735784754905434100",
					Date:            time.UnixMilli(1735784755068),
					LastUpdated:     time.UnixMilli(1735784755068),
					RemainingAmount: 1,
					Amount:          1,
					Price:           100,
					Type:            order.Limit,
					Side:            order.Sell,
					Status:          order.Open,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := g.deriveSpotWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

func TestDeriveFuturesWebsocketOrderResponse(t *testing.T) {
	t.Parallel()

	var resp *WebsocketFuturesOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`), &resp), "unmarshal must not error")

	got, err := g.deriveFuturesWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             g.Name,
		OrderID:              "596729318437",
		AssetType:            asset.Futures,
		Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1337",
		Date:                 time.UnixMilli(1735787107449),
		LastUpdated:          time.UnixMilli(1735787107450),
		Amount:               2,
		AverageExecutedPrice: 0.0000002625,
		Type:                 order.Market,
		Side:                 order.Long,
		Status:               order.Filled,
		ImmediateOrCancel:    true,
		ReduceOnly:           true,
	}, got)
}

func TestDeriveFuturesWebsocketOrderResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of futures orders",
			orders: [][]byte{
				[]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`),
				[]byte(`{"text":"t-1336","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"REX_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.03654","id":596662040388,"create_time":1735778597.374,"size":-2,"finish_time":1735778597.374,"update_time":1735778597.374,"left":0,"user":12870774}`),
				[]byte(`{"text":"apiv4-ws","price":"40000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596746193678,"create_time":1735789790.476,"size":1,"update_time":1735789790.476,"left":1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"200000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596748780649,"create_time":1735790222.185,"size":-1,"update_time":1735790222.185,"left":-1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98172.9","id":36028797827161124,"create_time":1740108860.761,"size":1,"finish_time":1740108860.761,"update_time":1740108860.761,"left":0,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98113.1","id":36028797827225781,"create_time":1740109172.06,"size":-1,"finish_time":1740109172.06,"update_time":1740109172.06,"left":0,"user":2365748,"is_reduce_only":true}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             g.Name,
					OrderID:              "596729318437",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1337",
					Date:                 time.UnixMilli(1735787107449),
					LastUpdated:          time.UnixMilli(1735787107450),
					Amount:               2,
					AverageExecutedPrice: 0.0000002625,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
					ReduceOnly:           true,
				},
				{
					Exchange:             g.Name,
					OrderID:              "596662040388",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1336",
					Date:                 time.UnixMilli(1735778597374),
					LastUpdated:          time.UnixMilli(1735778597374),
					Amount:               2,
					AverageExecutedPrice: 0.03654,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
				},
				{
					Exchange:        g.Name,
					OrderID:         "596746193678",
					AssetType:       asset.Futures,
					Pair:            currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735789790476),
					LastUpdated:     time.UnixMilli(1735789790476),
					RemainingAmount: 1,
					Amount:          1,
					Price:           40000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
				},
				{
					Exchange:        g.Name,
					OrderID:         "596748780649",
					AssetType:       asset.Futures,
					Pair:            currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735790222185),
					LastUpdated:     time.UnixMilli(1735790222185),
					RemainingAmount: 1,
					Amount:          1,
					Price:           200000,
					Type:            order.Limit,
					Side:            order.Short,
					Status:          order.Open,
				},
				{
					Exchange:             g.Name,
					OrderID:              "36028797827161124",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740108860761),
					LastUpdated:          time.UnixMilli(1740108860761),
					Amount:               1,
					AverageExecutedPrice: 98172.9,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
				},
				{
					Exchange:             g.Name,
					OrderID:              "36028797827225781",
					AssetType:            asset.Futures,
					Pair:                 currency.NewPair(currency.BTC, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740109172060),
					LastUpdated:          time.UnixMilli(1740109172060),
					Amount:               1,
					AverageExecutedPrice: 98113.1,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					ImmediateOrCancel:    true,
					ReduceOnly:           true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketFuturesOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := g.deriveFuturesWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}
