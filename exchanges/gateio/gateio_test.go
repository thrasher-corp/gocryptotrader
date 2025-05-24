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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	r := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
	}

	for _, a := range g.GetAssetTypes(false) {
		r.AssetType = a
		r.Pair = currency.EMPTYPAIR
		_, err = g.CancelAllOrders(t.Context(), r)
		assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

		r.Pair = getPair(t, a)
		_, err = g.CancelAllOrders(t.Context(), r)
		require.NoError(t, err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.UpdateAccountInfo(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateAccountInfo should not error for asset %s", a)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	cryptocurrencyChains, err := g.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	require.NotEmpty(t, cryptocurrencyChains, "GetAvailableTransferChains must return some chains")
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
	_, err = g.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.GetOrderInfo(t.Context(), "917591554", getPair(t, a), a)
		require.NoErrorf(t, err, "GetOrderInfo must not error for asset %s", a)
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
		CurrencyPair: currency.NewBTCUSDT(),
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
		CurrencyPair: currency.NewBTCUSDT(),
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
	ctx := t.Context()
	req := SubAccountTransferParam{SubAccountType: "index"}
	require.ErrorIs(t, g.SubAccountTransfer(ctx, req), currency.ErrCurrencyCodeEmpty)
	req.Currency = currency.BTC
	require.ErrorIs(t, g.SubAccountTransfer(ctx, req), errInvalidSubAccount)
	req.SubAccount = "1337"
	require.ErrorIs(t, g.SubAccountTransfer(ctx, req), errInvalidTransferDirection)
	req.Direction = "to"
	require.ErrorIs(t, g.SubAccountTransfer(ctx, req), errInvalidAmount)
	req.Amount = 1.337
	require.ErrorIs(t, g.SubAccountTransfer(ctx, req), asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	req.SubAccountType = "spot"
	require.NoError(t, g.SubAccountTransfer(ctx, req))
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

	for _, c := range []currency.Code{currency.BTC, currency.USDT} {
		_, err := g.GetAllFutureContracts(t.Context(), c)
		assert.NoErrorf(t, err, "GetAllFutureContracts %s should not error", c)
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesContract(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String())
	assert.NoError(t, err, "GetFuturesContract should not error")
	_, err = g.GetFuturesContract(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String())
	assert.NoError(t, err, "GetFuturesContract should not error")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesOrderbook(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String(), "", 10, false)
	assert.NoError(t, err, "GetFuturesOrderbook should not error for CoinMarginedFutures")
	_, err = g.GetFuturesOrderbook(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String(), "", 10, false)
	assert.NoError(t, err, "GetFuturesOrderbook should not error for USDTMarginedFutures")
}

func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesTradingHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesTradingHistory should not error for CoinMarginedFutures")
	_, err = g.GetFuturesTradingHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0, 0, "", time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesTradingHistory should not error for USDTMarginedFutures")
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesCandlesticks(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures).String(), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "GetFuturesCandlesticks should not error for CoinMarginedFutures")
	_, err = g.GetFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures).String(), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "GetFuturesCandlesticks should not error for USDTMarginedFutures")
}

func TestPremiumIndexKLine(t *testing.T) {
	t.Parallel()
	_, err := g.PremiumIndexKLine(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "PremiumIndexKLine should not error for CoinMarginedFutures")
	_, err = g.PremiumIndexKLine(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "PremiumIndexKLine should not error for USDTMarginedFutures")
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesTickers(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err, "GetFutureTickers should not error for CoinMarginedFutures")
	_, err = g.GetFuturesTickers(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err, "GetFutureTickers should not error for USDTMarginedFutures")
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetFutureFundingRates(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0)
	assert.NoError(t, err, "GetFutureFundingRates should not error for CoinMarginedFutures")
	_, err = g.GetFutureFundingRates(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0)
	assert.NoError(t, err, "GetFutureFundingRates should not error for USDTMarginedFutures")
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetFuturesInsuranceBalanceHistory(t.Context(), currency.USDT, 0)
	assert.NoError(t, err, "GetFuturesInsuranceBalanceHistory should not error")
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	_, err := g.GetFutureStats(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, 0, 0)
	assert.NoError(t, err, "GetFutureStats should not error for CoinMarginedFutures")
	_, err = g.GetFutureStats(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, 0, 0)
	assert.NoError(t, err, "GetFutureStats should not error for USDTMarginedFutures")
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	_, err := g.GetIndexConstituent(t.Context(), currency.USDT, currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}.String())
	assert.NoError(t, err, "GetIndexConstituent should not error")
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), time.Time{}, time.Time{}, 0)
	assert.NoError(t, err, "GetLiquidationHistory should not error for CoinMarginedFutures")
	_, err = g.GetLiquidationHistory(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), time.Time{}, time.Time{}, 0)
	assert.NoError(t, err, "GetLiquidationHistory should not error for USDTMarginedFutures")
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
	_, err := g.UpdateFuturesPositionMargin(t.Context(), currency.BTC, 0.01, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err, "UpdateFuturesPositionMargin should not error for CoinMarginedFutures")
	_, err = g.UpdateFuturesPositionMargin(t.Context(), currency.USDT, 0.01, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err, "UpdateFuturesPositionMargin should not error for USDTMarginedFutures")
}

func TestUpdateFuturesPositionLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.UpdateFuturesPositionLeverage(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 1, 0)
	assert.NoError(t, err, "UpdateFuturesPositionLeverage should not error for CoinMarginedFutures")
	_, err = g.UpdateFuturesPositionLeverage(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 1, 0)
	assert.NoError(t, err, "UpdateFuturesPositionLeverage should not error for USDTMarginedFutures")
}

func TestUpdateFuturesPositionRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.UpdateFuturesPositionRiskLimit(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 10)
	assert.NoError(t, err, "UpdateFuturesPositionRiskLimit should not error for CoinMarginedFutures")
	_, err = g.UpdateFuturesPositionRiskLimit(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10)
	assert.NoError(t, err, "UpdateFuturesPositionRiskLimit should not error for USDTMarginedFutures")
}

func TestPlaceDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.PlaceDeliveryOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.DeliveryFutures),
		Size:        6024,
		Iceberg:     0,
		Price:       "3765",
		Text:        "t-my-custom-id",
		Settle:      currency.USDT,
		TimeInForce: gtcTIF,
	})
	assert.NoError(t, err, "CreateDeliveryOrder should not error")
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), statusOpen, currency.USDT, "", 0, 0, 1)
	assert.NoError(t, err, "GetDeliveryOrders should not error")
}

func TestCancelMultipleDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelMultipleDeliveryOrders(t.Context(), getPair(t, asset.DeliveryFutures), "ask", currency.USDT)
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
	_, err := g.CancelAllDeliveryPriceTriggeredOrder(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
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
	_, err := g.RetrivePositionDetailInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err, "RetrivePositionDetailInDualMode should not error for CoinMarginedFutures")
	_, err = g.RetrivePositionDetailInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err, "RetrivePositionDetailInDualMode should not error for USDTMarginedFutures")
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.UpdatePositionMarginInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, "dual_long")
	assert.NoError(t, err, "UpdatePositionMarginInDualMode should not error for CoinMarginedFutures")
	_, err = g.UpdatePositionMarginInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0.001, "dual_long")
	assert.NoError(t, err, "UpdatePositionMarginInDualMode should not error for USDTMarginedFutures")
}

func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.UpdatePositionLeverageInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0.001, 0.001)
	assert.NoError(t, err, "UpdatePositionLeverageInDualMode should not error for CoinMarginedFutures")
	_, err = g.UpdatePositionLeverageInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 0.001, 0.001)
	assert.NoError(t, err, "UpdatePositionLeverageInDualMode should not error for USDTMarginedFutures")
}

func TestUpdatePositionRiskLimitInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.UpdatePositionRiskLimitInDualMode(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 10)
	assert.NoError(t, err, "UpdatePositionRiskLimitInDualMode should not error for CoinMarginedFutures")
	_, err = g.UpdatePositionRiskLimitInDualMode(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures), 10)
	assert.NoError(t, err, "UpdatePositionRiskLimitInDualMode should not error for USDTMarginedFutures")
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.PlaceFuturesOrder(t.Context(), &ContractOrderCreateParams{
		Contract:    getPair(t, asset.CoinMarginedFutures),
		Size:        6024,
		Iceberg:     0,
		Price:       "3765",
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      currency.BTC,
	})
	assert.NoError(t, err, "PlaceFuturesOrder should not error")
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesOrders(t.Context(), currency.NewBTCUSD(), statusOpen, "", currency.BTC, 0, 0, 1)
	assert.NoError(t, err, "GetFuturesOrders should not error")
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelMultipleFuturesOpenOrders(t.Context(), getPair(t, asset.USDTMarginedFutures), "ask", currency.USDT)
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
	_, err := g.PlaceBatchFuturesOrders(t.Context(), currency.BTC, []ContractOrderCreateParams{
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
			Size:        6024,
			Iceberg:     0,
			Price:       "3765",
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      currency.BTC,
		},
		{
			Contract:    getPair(t, asset.CoinMarginedFutures),
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
	_, err := g.GetMyFuturesTradingHistory(t.Context(), currency.BTC, "", "", getPair(t, asset.CoinMarginedFutures), 0, 0, 0)
	assert.NoError(t, err, "GetMyFuturesTradingHistory should not error")
}

func TestGetFuturesPositionCloseHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesPositionCloseHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetFuturesPositionCloseHistory should not error")
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetFuturesLiquidationHistory(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures), 0, time.Time{})
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
	for _, tc := range []struct {
		c currency.Code
		a asset.Item
	}{
		{currency.BTC, asset.CoinMarginedFutures},
		{currency.USDT, asset.USDTMarginedFutures},
	} {
		_, err := g.CreatePriceTriggeredFuturesOrder(t.Context(), tc.c, &FuturesPriceTriggeredOrderParam{
			Initial: FuturesInitial{
				Price:    1234.,
				Size:     2,
				Contract: getPair(t, tc.a),
			},
			Trigger: FuturesTrigger{
				Rule:      1,
				OrderType: "close-short-position",
			},
		})
		assert.NoErrorf(t, err, "CreatePriceTriggeredFuturesOrder should not error for settlement currency: %s, asset: %s", tc.c, tc.a)
	}
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
	_, err := g.CancelAllFuturesOpenOrders(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	assert.NoError(t, err, "CancelAllFuturesOpenOrders should not error for CoinMarginedFutures")
	_, err = g.CancelAllFuturesOpenOrders(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	assert.NoError(t, err, "CancelAllFuturesOpenOrders should not error for USDTMarginedFutures")
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
	_, err := g.GetDeliveryContract(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
	assert.NoError(t, err, "GetDeliveryContract should not error")
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryOrderbook(t.Context(), currency.USDT, "0", getPair(t, asset.DeliveryFutures), 0, false)
	assert.NoError(t, err, "GetDeliveryOrderbook should not error")
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryTradingHistory(t.Context(), currency.USDT, "", getPair(t, asset.DeliveryFutures), 0, time.Time{}, time.Time{})
	assert.NoError(t, err, "GetDeliveryTradingHistory should not error")
}

func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryFuturesCandlesticks(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures), time.Time{}, time.Time{}, 0, kline.OneWeek)
	assert.NoError(t, err, "GetDeliveryFuturesCandlesticks should not error")
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetDeliveryFutureTickers(t.Context(), currency.USDT, getPair(t, asset.DeliveryFutures))
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
	_, err = g.UpdateDeliveryPositionMargin(t.Context(), currency.USDT, 0.001, getPair(t, asset.DeliveryFutures))
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
	_, err := g.GetOptionsOrderbook(t.Context(), getPair(t, asset.Options), "0.1", 9, true)
	assert.NoError(t, err, "GetOptionsOrderbook should not error")
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
	for _, a := range g.GetAssetTypes(false) {
		pairs, err := g.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "FetchTradablePairs must return some pairs for %s", a)
		if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
			for _, p := range pairs {
				_, err := getSettlementCurrency(p, a)
				require.NoErrorf(t, err, "Fetched pair %s %s must not error on getSettlementCurrency", a, p)
			}
		}
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	for _, a := range g.GetAssetTypes(false) {
		err := g.UpdateTickers(t.Context(), a)
		assert.NoErrorf(t, err, "UpdateTickers should not error for %s", a)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.UpdateOrderbook(t.Context(), getPair(t, a), a)
		assert.NoErrorf(t, err, "UpdateOrderbook should not error for %s", a)
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
	for _, a := range g.GetAssetTypes(false) {
		if a != asset.CoinMarginedFutures {
			_, err := g.GetRecentTrades(t.Context(), getPair(t, a), a)
			assert.NoErrorf(t, err, "GetRecentTrades should not error for %s", a)
		}
	}
}

func TestSubmitOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.SubmitOrder(t.Context(), &order.Submit{
			Exchange:    g.Name,
			Pair:        getPair(t, a),
			Side:        order.Buy,
			Type:        order.Limit,
			Price:       1,
			Amount:      1,
			AssetType:   a,
			TimeInForce: order.GoodTillCancel,
		})
		assert.NoErrorf(t, err, "SubmitOrder should not error for %s", a)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	for _, a := range g.GetAssetTypes(false) {
		orderCancellation := &order.Cancel{
			OrderID:   "1",
			AccountID: "1",
			Pair:      getPair(t, a),
			AssetType: a,
		}
		err := g.CancelOrder(t.Context(), orderCancellation)
		assert.NoErrorf(t, err, "CancelOrder should not error for %s", a)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.CancelBatchOrders(t.Context(), []order.Cancel{
			{
				OrderID:   "1",
				AccountID: "1",
				Pair:      getPair(t, a),
				AssetType: a,
			}, {
				OrderID:   "2",
				AccountID: "1",
				Pair:      getPair(t, a),
				AssetType: a,
			},
		})
		assert.NoErrorf(t, err, "CancelBatchOrders should not error for %s", a)
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
	for _, a := range g.GetAssetTypes(false) {
		enabledPairs := getPairs(t, a)
		if len(enabledPairs) > 2 {
			enabledPairs = enabledPairs[:2]
		}
		_, err := g.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			Pairs:     enabledPairs,
			Type:      order.AnyType,
			Side:      order.AnySide,
			AssetType: a,
		})
		assert.NoErrorf(t, err, "GetActiveOrders should not error for %s", a)
	}
}

func TestGetOrderHistory(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	for _, a := range g.GetAssetTypes(false) {
		enabledPairs := getPairs(t, a)
		if len(enabledPairs) > 4 {
			enabledPairs = enabledPairs[:4]
		}
		multiOrderRequest := order.MultiOrderRequest{
			Type:      order.AnyType,
			Side:      order.Buy,
			Pairs:     enabledPairs,
			AssetType: a,
		}
		_, err := g.GetOrderHistory(t.Context(), &multiOrderRequest)
		assert.NoErrorf(t, err, "GetOrderHistory should not error for %s", a)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 10)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.GetHistoricCandles(t.Context(), getPair(t, a), a, kline.OneDay, startTime, time.Now())
		if a == asset.Options {
			assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles should error correctly for options")
		} else {
			assert.NoErrorf(t, err, "GetHistoricCandles should not error for %s", a)
		}
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 5)
	for _, a := range g.GetAssetTypes(false) {
		_, err := g.GetHistoricCandlesExtended(t.Context(), getPair(t, a), a, kline.OneMin, startTime, time.Now())
		if a == asset.Options {
			assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended should error correctly for options")
		} else {
			assert.NoErrorf(t, err, "GetHistoricCandlesExtended should not error for %s", a)
		}
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
	} else if !uly.Equal(currency.NewBTCUSDT()) {
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
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: "test"})
	if err := g.WsHandleSpotData(ctx, []byte(wsBalancesPushDataJSON)); err != nil {
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
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: "test"})
	if err := g.WsHandleSpotData(ctx, []byte(wsCrossMarginBalancePushDataJSON)); err != nil {
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

// TestFuturesDataHandler ensures that messages from various futures channels do not error
func TestFuturesDataHandler(t *testing.T) {
	t.Parallel()
	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsFutures.json", func(m []byte) error {
		ctx := t.Context()
		if strings.Contains(string(m), "futures.balances") {
			ctx = account.DeployCredentialsToContext(ctx, &account.Credentials{Key: "test", Secret: "test"})
		}
		return g.WsHandleFuturesData(ctx, m, asset.CoinMarginedFutures)
	})
	close(g.Websocket.DataHandler)
	assert.Len(t, g.Websocket.DataHandler, 14, "Should see the correct number of messages")
	for resp := range g.Websocket.DataHandler {
		if err, isErr := resp.(error); isErr {
			assert.NoError(t, err, "Should not get any errors down the data handler")
		}
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
	p := getPair(t, asset.Options)
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), []byte(optionsOrderbookTickerPushDataJSON)))
	assert.NoError(t, g.WsHandleOptionsData(t.Context(), fmt.Appendf(nil, optionsOrderbookUpdatePushDataJSON, p.Upper().String())))
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
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "test", Secret: "test"})
	if err := g.WsHandleOptionsData(ctx, []byte(optionsBalancePushDataJSON)); err != nil {
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

func TestGenerateSubscriptionsSpot(t *testing.T) {
	t.Parallel()

	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Test instance Setup must not error")

	g.Websocket.SetCanUseAuthenticatedEndpoints(true)
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
	g := new(Gateio) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(g), "Test instance Setup must not error")
	subs, err := g.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	subs, err = g.GenerateFuturesDefaultSubscriptions(asset.CoinMarginedFutures)
	require.NoError(t, err)
	require.NotEmpty(t, subs)
	require.NoError(t, g.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = g.GenerateFuturesDefaultSubscriptions(asset.USDTMarginedFutures)
	require.NoError(t, err, "Disabled asset must not error")
	require.Empty(t, subs, "Disabled asset must return no pairs")
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

	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		c, err = g.GetFuturesContractDetails(t.Context(), a)
		require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
		assert.NotEmptyf(t, c, "GetFuturesContractDetails should return some contracts for %s", a)
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	assert.NoError(t, err)

	_, err = g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset: asset.CoinMarginedFutures,
		Pair:  currency.NewBTCUSD(),
	})
	assert.NoError(t, err)

	_, err = g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures})
	assert.NoError(t, err)
	_, err = g.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures})
	assert.NoError(t, err)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := g.GetHistoricalFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  currency.NewPair(currency.ENJ, currency.USDT),
	})
	assert.ErrorIs(t, err, fundingrate.ErrPaymentCurrencyCannotBeEmpty)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		IncludePayments: true,
	})
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency:      currency.USDT,
		IncludePredictedRate: true,
	})
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(time.Hour * 16),
		EndDate:         time.Now(),
	})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)

	_, err = g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 8008),
		EndDate:         time.Now(),
	})
	assert.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits)

	history, err := g.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            currency.NewPair(currency.ENJ, currency.USDT),
		PaymentCurrency: currency.USDT,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, history)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := g.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("GOLDFISH").Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should error correctly")

	var resp []futures.OpenInterest
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures} {
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

func TestGetClientOrderIDFromText(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getClientOrderIDFromText("api"), "should not return anything")
	assert.Equal(t, "t-123", getClientOrderIDFromText("t-123"), "should return t-123")
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
			processed, err := g.processFuturesOrdersPushData([]byte(tc.incoming), asset.CoinMarginedFutures)
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

func TestGetSettlementCurrency(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		p   currency.Pair
		exp currency.Code
		err error
	}{
		{asset.Futures, currency.EMPTYPAIR, currency.EMPTYCODE, asset.ErrNotSupported},
		{asset.DeliveryFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.USDTMarginedFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.CoinMarginedFutures), currency.EMPTYCODE, errInvalidSettlementQuote},
		{asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.BTC, nil},
		{asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), currency.BTC, nil},
		{asset.CoinMarginedFutures, getPair(t, asset.USDTMarginedFutures), currency.EMPTYCODE, errInvalidSettlementBase},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.ETH, Quote: currency.USD}, currency.EMPTYCODE, errInvalidSettlementBase},
		{asset.CoinMarginedFutures, currency.NewBTCUSDT(), currency.EMPTYCODE, errInvalidSettlementQuote},
	} {
		c, err := getSettlementCurrency(tt.p, tt.a)
		if tt.err == nil {
			require.NoErrorf(t, err, "getSettlementCurrency must not error for %s %s", tt.a, tt.p)
		} else {
			assert.ErrorIsf(t, err, tt.err, "getSettlementCurrency should return correct error for %s %s", tt.a, tt.p)
		}
		assert.Equalf(t, tt.exp, c, "getSettlementCurrency should return correct settlement currency for %s %s", tt.a, tt.p)
	}
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
		Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1735720637181634009",
		Date:                 time.UnixMilli(1735720637188),
		LastUpdated:          time.UnixMilli(1735720637188),
		Amount:               0.0001,
		AverageExecutedPrice: 93503.3,
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
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
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637181634009",
					Date:                 time.UnixMilli(1735720637188),
					LastUpdated:          time.UnixMilli(1735720637188),
					Amount:               0.0001,
					AverageExecutedPrice: 93503.3,
					Type:                 order.Market,
					Side:                 order.Sell,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
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
					TimeInForce:          order.ImmediateOrCancel,
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
					TimeInForce:          order.FillOrKill,
					Cost:                 7.346,
					Purchased:            200,
				},
				{
					Exchange:        g.Name,
					OrderID:         "766504537761",
					AssetType:       asset.Spot,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735780321603944400",
					Date:            time.UnixMilli(1735780321729),
					LastUpdated:     time.UnixMilli(1735780321729),
					RemainingAmount: 0.0003,
					Amount:          0.0003,
					Price:           20000,
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.Open,
					TimeInForce:     order.PostOnly,
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
					TimeInForce:     order.GoodTillCancel,
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
		TimeInForce:          order.ImmediateOrCancel,
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
					TimeInForce:          order.ImmediateOrCancel,
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
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:        g.Name,
					OrderID:         "596746193678",
					AssetType:       asset.Futures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735789790476),
					LastUpdated:     time.UnixMilli(1735789790476),
					RemainingAmount: 1,
					Amount:          1,
					Price:           40000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        g.Name,
					OrderID:         "596748780649",
					AssetType:       asset.Futures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735790222185),
					LastUpdated:     time.UnixMilli(1735790222185),
					RemainingAmount: 1,
					Amount:          1,
					Price:           200000,
					Type:            order.Limit,
					Side:            order.Short,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:             g.Name,
					OrderID:              "36028797827161124",
					AssetType:            asset.Futures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740108860761),
					LastUpdated:          time.UnixMilli(1740108860761),
					Amount:               1,
					AverageExecutedPrice: 98172.9,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:             g.Name,
					OrderID:              "36028797827225781",
					AssetType:            asset.Futures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740109172060),
					LastUpdated:          time.UnixMilli(1740109172060),
					Amount:               1,
					AverageExecutedPrice: 98113.1,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
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

func TestConvertSmallBalances(t *testing.T) {
	t.Parallel()
	err := g.ConvertSmallBalances(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)

	err = g.ConvertSmallBalances(t.Context(), currency.F16)
	require.NoError(t, err)
}

func TestGetAccountDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	got, err := g.GetAccountDetails(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

func TestGetUserTransactionRateLimitInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	got, err := g.GetUserTransactionRateLimitInfo(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, got)
}

var pairMap = map[asset.Item]currency.Pairs{}

var pairsGuard sync.RWMutex

func getPair(tb testing.TB, a asset.Item) currency.Pair {
	tb.Helper()
	if p := getPairs(tb, a); len(p) != 0 {
		return p[0]
	}
	return currency.EMPTYPAIR
}

func getPairs(tb testing.TB, a asset.Item) currency.Pairs {
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
	if !assert.NotEmptyf(tb, enabledPairs, "%s GetEnabledPairs should not be empty", a) {
		tb.Fatalf("No pair available for asset %s", a)
		return nil
	}
	pairMap[a] = enabledPairs

	return enabledPairs
}

func BenchmarkTimeInForceFromString(b *testing.B) {
	for b.Loop() {
		for _, tifString := range []string{gtcTIF, iocTIF, pocTIF, fokTIF} {
			if _, err := timeInForceFromString(tifString); err != nil {
				b.Fatal(tifString)
			}
		}
	}
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	_, err := timeInForceFromString("abcdef")
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	for k, v := range map[string]order.TimeInForce{gtcTIF: order.GoodTillCancel, iocTIF: order.ImmediateOrCancel, pocTIF: order.PostOnly, fokTIF: order.FillOrKill} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			tif, err := timeInForceFromString(k)
			require.NoError(t, err)
			assert.Equal(t, v, tif)
		})
	}
}

func TestGetTypeFromTimeInForce(t *testing.T) {
	t.Parallel()
	typeResp := getTypeFromTimeInForce("gtc", 0)
	assert.Equal(t, order.Limit, typeResp)

	typeResp = getTypeFromTimeInForce("ioc", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")

	typeResp = getTypeFromTimeInForce("poc", 123)
	assert.Equal(t, order.Limit, typeResp, "should be limit order")

	typeResp = getTypeFromTimeInForce("fok", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	assert.Empty(t, timeInForceString(order.UnknownTIF))
	for _, valid := range validTimesInForce {
		assert.Equal(t, valid.String, timeInForceString(valid.TimeInForce))
	}
}

func TestIsSingleOrderbookChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel  string
		expected bool
	}{
		{channel: spotOrderbookUpdateChannel, expected: true},
		{channel: spotOrderbookChannel, expected: true},
		{channel: spotOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookChannel, expected: true},
		{channel: futuresOrderbookTickerChannel, expected: true},
		{channel: futuresOrderbookUpdateChannel, expected: true},
		{channel: optionsOrderbookChannel, expected: true},
		{channel: optionsOrderbookTickerChannel, expected: true},
		{channel: optionsOrderbookUpdateChannel, expected: true},
		{channel: spotTickerChannel, expected: false},
		{channel: "sad", expected: false},
	} {
		assert.Equal(t, tc.expected, isSingleOrderbookChannel(tc.channel))
	}
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	require.NoError(t, g.ValidateSubscriptions(nil))
	require.NoError(t, g.ValidateSubscriptions([]*subscription.Subscription{{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}}}))
	require.NoError(t, g.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}))
	require.NoError(t, g.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD(), currency.NewBTCUSDT()}},
	}))
	require.NoError(t, g.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSDT()}},
	}))
	require.ErrorIs(t, g.ValidateSubscriptions([]*subscription.Subscription{
		{Channel: spotTickerChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookUpdateChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
		{Channel: spotOrderbookChannel, Pairs: []currency.Pair{currency.NewBTCUSD()}},
	}), subscription.ErrExclusiveSubscription)
}

func TestCandlesChannelIntervals(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: 0}
	_, err := candlesChannelInterval(s)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "candlestickChannelInterval must error correctly with a 0 interval")
	s.Interval = kline.ThousandMilliseconds
	i, err := candlesChannelInterval(s)
	require.NoError(t, err)
	assert.Equal(t, "1000ms", i)
}

func TestOrderbookChannelIntervals(t *testing.T) {
	t.Parallel()

	s := &subscription.Subscription{Channel: futuresOrderbookUpdateChannel, Interval: kline.TwentyMilliseconds, Levels: 100}
	_, err := orderbookChannelInterval(s, asset.Futures)
	require.ErrorIs(t, err, subscription.ErrInvalidInterval)
	require.ErrorContains(t, err, "20ms only valid with Levels 20")
	s.Levels = 20
	i, err := orderbookChannelInterval(s, asset.Futures)
	require.NoError(t, err)
	assert.Equal(t, "20ms", i)

	for s, exp := range map[*subscription.Subscription]error{
		{Asset: asset.Binary, Channel: "unknown_channel", Interval: kline.OneYear}:                                   nil,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: kline.OneDay}:                             subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookTickerChannel, Interval: 0}:                                        nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.OneDay}:                                   subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.HundredMilliseconds}:                      nil,
		{Asset: asset.Spot, Channel: spotOrderbookChannel, Interval: kline.ThousandMilliseconds}:                     nil,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.OneDay}:                             subscription.ErrInvalidInterval,
		{Asset: asset.Spot, Channel: spotOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:                nil,
		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:              subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                                  nil,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:                    subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookChannel, Interval: 0}:                                        nil,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:                       subscription.ErrInvalidInterval,
		{Asset: asset.Futures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:          nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: kline.TenMilliseconds}:      subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookTickerChannel, Interval: 0}:                          nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: kline.TenMilliseconds}:            subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookChannel, Interval: 0}:                                nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.OneDay}:               subscription.ErrInvalidInterval,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:  nil,
		{Asset: asset.DeliveryFutures, Channel: futuresOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}: nil,

		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: kline.TenMilliseconds}:          subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookTickerChannel, Interval: 0}:                              nil,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: kline.TwoHundredAndFiftyMilliseconds}: subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookChannel, Interval: 0}:                                    nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.OneDay}:                   subscription.ErrInvalidInterval,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.HundredMilliseconds}:      nil,
		{Asset: asset.Options, Channel: optionsOrderbookUpdateChannel, Interval: kline.ThousandMilliseconds}:     nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+s.Interval.Short(), func(t *testing.T) {
			t.Parallel()
			i, err := orderbookChannelInterval(s, s.Asset)
			if exp != nil {
				require.ErrorIs(t, err, exp)
			} else {
				switch {
				case s.Channel == "unknown_channel":
					assert.Empty(t, i, "orderbookChannelInterval should return empty for unknown channels")
				case strings.HasSuffix(s.Channel, "_ticker"):
					assert.Empty(t, i)
				case s.Interval == 0:
					assert.Equal(t, "0", i)
				default:
					exp, err2 := getIntervalString(s.Interval)
					require.NoError(t, err2, "getIntervalString must not error for validating expected value")
					require.Equal(t, exp, i)
				}
			}
		})
	}
}

func TestChannelLevels(t *testing.T) {
	t.Parallel()

	for s, exp := range map[*subscription.Subscription]error{
		{Channel: "unknown_channel", Asset: asset.Binary}:                                   nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookTickerChannel, Asset: asset.Spot, Levels: 1}:                 subscription.ErrInvalidLevel,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot}:                            nil,
		{Channel: spotOrderbookUpdateChannel, Asset: asset.Spot, Levels: 100}:               subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot}:                                  subscription.ErrInvalidLevel,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 5}:                       nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 10}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 20}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 50}:                      nil,
		{Channel: spotOrderbookChannel, Asset: asset.Spot, Levels: 100}:                     nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures}:                            subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 1}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 5}:                 nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 10}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 20}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 50}:                nil,
		{Channel: futuresOrderbookChannel, Asset: asset.Futures, Levels: 100}:               nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures}:                      nil,
		{Channel: futuresOrderbookTickerChannel, Asset: asset.Futures, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures}:                      subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 20}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.Futures, Levels: 50}:          nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures}:              subscription.ErrInvalidLevel,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 5}:   nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 10}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 20}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 50}:  nil,
		{Channel: futuresOrderbookUpdateChannel, Asset: asset.DeliveryFutures, Levels: 100}: nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options}:                      nil,
		{Channel: optionsOrderbookTickerChannel, Asset: asset.Options, Levels: 1}:           subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options}:                      subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 5}:           nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 10}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 20}:          nil,
		{Channel: optionsOrderbookUpdateChannel, Asset: asset.Options, Levels: 50}:          nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options}:                            subscription.ErrInvalidLevel,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 5}:                 nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 10}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 20}:                nil,
		{Channel: optionsOrderbookChannel, Asset: asset.Options, Levels: 50}:                nil,
	} {
		t.Run(s.Asset.String()+"/"+s.Channel+"/"+strconv.Itoa(s.Levels), func(t *testing.T) {
			t.Parallel()
			l, err := channelLevels(s, s.Asset)
			switch {
			case exp != nil:
				require.ErrorIs(t, err, exp)
			case s.Levels == 0:
				assert.Empty(t, l)
			default:
				require.NoError(t, err)
				require.NotEmpty(t, l)
			}
		})
	}
}

func TestGetIntervalString(t *testing.T) {
	t.Parallel()
	for k, exp := range map[kline.Interval]string{
		kline.TenMilliseconds:                "10ms",
		kline.TwentyMilliseconds:             "20ms",
		kline.HundredMilliseconds:            "100ms",
		kline.TwoHundredAndFiftyMilliseconds: "250ms",
		kline.ThousandMilliseconds:           "1000ms",
		kline.TenSecond:                      "10s",
		kline.ThirtySecond:                   "30s",
		kline.OneMin:                         "1m",
		kline.FiveMin:                        "5m",
		kline.FifteenMin:                     "15m",
		kline.ThirtyMin:                      "30m",
		kline.OneHour:                        "1h",
		kline.TwoHour:                        "2h",
		kline.FourHour:                       "4h",
		kline.EightHour:                      "8h",
		kline.TwelveHour:                     "12h",
		kline.OneDay:                         "1d",
		kline.SevenDay:                       "7d",
		kline.OneMonth:                       "30d",
	} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			s, err := getIntervalString(k)
			require.NoError(t, err)
			assert.Equal(t, exp, s)
		})
	}
	_, err := getIntervalString(0)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "0 should be an invalid interval")
	_, err = getIntervalString(kline.FiveDay)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Any other random interval should also be invalid")
}
