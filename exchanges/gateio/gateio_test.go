package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var g = &Gateio{}
var spotTradablePair, marginTradablePair, crossMarginTradablePair, futuresTradablePair, optionsTradablePair, deliveryFuturesTradablePair currency.Pair

func TestMain(m *testing.M) {
	g.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("GateIO load config error", err)
	}
	gConf, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		log.Fatal("GateIO Setup() init error")
	}
	gConf.API.AuthenticatedSupport = true
	gConf.API.AuthenticatedWebsocketSupport = true
	gConf.API.Credentials.Key = apiKey
	gConf.API.Credentials.Secret = apiSecret
	g.Websocket = sharedtestvalues.NewTestWebsocket()
	gConf.Features.Enabled.FillsFeed = true
	gConf.Features.Enabled.TradeFeed = true
	err = g.Setup(gConf)
	if err != nil {
		log.Fatal("GateIO setup error", err)
	}
	request.MaxRequestJobs = 200
	g.Run(context.Background())
	getFirstTradablePairOfAssets()
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	err := g.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = g.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelAllOrders(context.Background(), nil)
	if !errors.Is(err, order.ErrCancelOrderIsNil) {
		t.Error(err)
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          optionsTradablePair,
		AssetType:     asset.Options,
	}
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.AssetType = asset.Spot
	orderCancellation.Pair = spotTradablePair
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.Margin
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = marginTradablePair
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.CrossMargin
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = crossMarginTradablePair
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.Futures
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = futuresTradablePair
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = currency.EMPTYPAIR
	orderCancellation.AssetType = asset.DeliveryFutures
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}
	orderCancellation.Pair = deliveryFuturesTradablePair
	_, err = g.CancelAllOrders(context.Background(), orderCancellation)
	if err != nil {
		t.Error(err)
	}
}
func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error("GetAccountInfo() error", err)
	}
	if _, err := g.UpdateAccountInfo(context.Background(), asset.Margin); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(context.Background(), asset.CrossMargin); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(context.Background(), asset.Options); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(context.Background(), asset.Futures); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
	if _, err := g.UpdateAccountInfo(context.Background(), asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	cryptocurrencyChains, err := g.GetAvailableTransferChains(context.Background(), currency.BTC)
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
	if _, err = g.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest); err != nil {
		t.Errorf("%s WithdrawCryptocurrencyFunds() error: %v", g.Name, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetOrderInfo(context.Background(),
		"917591554", spotTradablePair, asset.Spot)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(context.Background(), "917591554", optionsTradablePair, asset.Options)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(context.Background(), "917591554", marginTradablePair, asset.Margin)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(context.Background(), "917591554", crossMarginTradablePair, asset.CrossMargin)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(context.Background(), "917591554", futuresTradablePair, asset.Futures)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
	_, err = g.GetOrderInfo(context.Background(), "917591554", deliveryFuturesTradablePair, asset.DeliveryFutures)
	if err != nil {
		t.Errorf("GetOrderInfo() %v", err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateTicker(context.Background(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
	_, err = g.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = g.UpdateTicker(context.Background(), marginTradablePair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.UpdateTicker(context.Background(), crossMarginTradablePair, asset.CrossMargin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = g.UpdateTicker(context.Background(), deliveryFuturesTradablePair, asset.DeliveryFutures)
	if err != nil {
		t.Error(err)
	}
}

func TestListSpotCurrencies(t *testing.T) {
	t.Parallel()
	if _, err := g.ListSpotCurrencies(context.Background()); err != nil {
		t.Errorf("%s ListAllCurrencies() error %v", g.Name, err)
	}
}

func TestGetCurrencyDetail(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCurrencyDetail(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetCurrencyDetail() error %v", g.Name, err)
	}
}

func TestListAllCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, err := g.ListSpotCurrencyPairs(context.Background()); err != nil {
		t.Errorf("%s ListAllCurrencyPairs() error %v", g.Name, err)
	}
}

func TestGetCurrencyPairDetal(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCurrencyPairDetail(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}.String()); err != nil {
		t.Errorf("%s GetCurrencyPairDetal() error %v", g.Name, err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetTickers(context.Background(), "BTC_USDT", ""); err != nil {
		t.Errorf("%s GetTickers() error %v", g.Name, err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	if _, err := g.GetTicker(context.Background(), currency.Pair{Base: currency.BTC, Delimiter: currency.UnderscoreDelimiter, Quote: currency.USDT}.String(), utc8TimeZone); err != nil {
		t.Errorf("%s GetTicker() error %v", g.Name, err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook(context.Background(), spotTradablePair.String(), "0.1", 10, false)
	if err != nil {
		t.Errorf("%s GetOrderbook() error %v", g.Name, err)
	}
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.GetFuturesOrderbook(context.Background(), settle, futuresTradablePair.String(), "", 10, false); err != nil {
		t.Errorf("%s GetOrderbook() error %v", g.Name, err)
	}
	settle, err = g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.GetDeliveryOrderbook(context.Background(), settle, "0.1", deliveryFuturesTradablePair, 10, false); err != nil {
		t.Errorf("%s GetOrderbook() error %v", g.Name, err)
	}
	if _, err = g.GetOptionsOrderbook(context.Background(), optionsTradablePair, "0.1", 10, false); err != nil {
		t.Errorf("%s GetOrderbook() error %v", g.Name, err)
	}
}

func TestGetMarketTrades(t *testing.T) {
	t.Parallel()
	if _, err := g.GetMarketTrades(context.Background(), spotTradablePair, 0, "", true, time.Time{}, time.Time{}, 1); err != nil {
		t.Errorf("%s GetMarketTrades() error %v", g.Name, err)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetCandlesticks(context.Background(), spotTradablePair, 0, time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Errorf("%s GetCandlesticks() error %v", g.Name, err)
	}
}
func TestGetTradingFeeRatio(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetTradingFeeRatio(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}); err != nil {
		t.Errorf("%s GetTradingFeeRatio() error %v", g.Name, err)
	}
}

func TestGetSpotAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSpotAccounts(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetSpotAccounts() error %v", g.Name, err)
	}
}

func TestCreateBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateBatchOrders(context.Background(), []CreateOrderRequestData{
		{
			CurrencyPair: spotTradablePair,
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
	}); err != nil {
		t.Errorf("%s CreateBatchOrders() error %v", g.Name, err)
	}
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GateioSpotOpenOrders(context.Background(), 0, 0, false); err != nil {
		t.Errorf("%s GetSpotOpenOrders() error %v", g.Name, err)
	}
}

func TestSpotClosePositionWhenCrossCurrencyDisabled(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.SpotClosePositionWhenCrossCurrencyDisabled(context.Background(), &ClosePositionRequestParam{
		Amount:       0.1,
		Price:        1234567384,
		CurrencyPair: spotTradablePair,
	}); err != nil {
		t.Errorf("%s SpotClosePositionWhenCrossCurrencyDisabled() error %v", g.Name, err)
	}
}

func TestCreateSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.PlaceSpotOrder(context.Background(), &CreateOrderRequestData{
		CurrencyPair: spotTradablePair,
		Side:         "buy",
		Amount:       1,
		Price:        900000,
		Account:      g.assetTypeToString(asset.Spot),
		Type:         "limit",
	}); err != nil {
		t.Errorf("%s CreateSpotOrder() error %v", g.Name, err)
	}
}

func TestGetSpotOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSpotOrders(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, "open", 0, 0); err != nil {
		t.Errorf("%s GetSpotOrders() error %v", g.Name, err)
	}
}

func TestCancelAllOpenOrdersSpecifiedCurrencyPair(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelAllOpenOrdersSpecifiedCurrencyPair(context.Background(), spotTradablePair, order.Sell, asset.Empty); err != nil {
		t.Errorf("%s CancelAllOpenOrdersSpecifiedCurrencyPair() error %v", g.Name, err)
	}
}

func TestCancelBatchOrdersWithIDList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelBatchOrdersWithIDList(context.Background(), []CancelOrderByIDParam{
		{
			CurrencyPair: spotTradablePair,
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
	if _, err := g.GetSpotOrder(context.Background(), "1234", currency.Pair{
		Base:      currency.BTC,
		Delimiter: currency.UnderscoreDelimiter,
		Quote:     currency.USDT}, asset.Spot); err != nil {
		t.Errorf("%s GetSpotOrder() error %v", g.Name, err)
	}
}
func TestAmendSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := g.AmendSpotOrder(context.Background(), "", spotTradablePair, false, &PriceAndAmount{
		Price: 1000,
	})
	if !errors.Is(err, errInvalidOrderID) {
		t.Errorf("expecting %v, but found %v", errInvalidOrderID, err)
	}
	_, err = g.AmendSpotOrder(context.Background(), "123", currency.EMPTYPAIR, false, &PriceAndAmount{
		Price: 1000,
	})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("expecting %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err = g.AmendSpotOrder(context.Background(), "123", spotTradablePair, false, &PriceAndAmount{
		Price: 1000,
	})
	if err != nil {
		t.Error(err)
	}
}
func TestCancelSingleSpotOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelSingleSpotOrder(context.Background(), "1234",
		spotTradablePair.String(), false); err != nil {
		t.Errorf("%s CancelSingleSpotOrder() error %v", g.Name, err)
	}
}

func TestGetPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GateIOGetPersonalTradingHistory(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, "", 0, 0, false, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetPersonalTradingHistory() error %v", g.Name, err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	if _, err := g.GetServerTime(context.Background(), asset.Spot); err != nil {
		t.Errorf("%s GetServerTime() error %v", g.Name, err)
	}
}

func TestCountdownCancelorder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CountdownCancelorders(context.Background(), CountdownCancelOrderParam{
		Timeout:      10,
		CurrencyPair: currency.Pair{Base: currency.BTC, Quote: currency.ETH, Delimiter: currency.UnderscoreDelimiter},
	}); err != nil {
		t.Errorf("%s CountdownCancelorder() error %v", g.Name, err)
	}
}

func TestCreatePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreatePriceTriggeredOrder(context.Background(), &PriceTriggeredOrderParam{
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
	if _, err := g.GetPriceTriggeredOrderList(context.Background(), "open", currency.EMPTYPAIR, asset.Empty, 0, 0); err != nil {
		t.Errorf("%s GetPriceTriggeredOrderList() error %v", g.Name, err)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelMultipleSpotOpenOrders(context.Background(), currency.EMPTYPAIR, asset.CrossMargin); err != nil {
		t.Errorf("%s CancelAllOpenOrders() error %v", g.Name, err)
	}
}

func TestGetSinglePriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSinglePriceTriggeredOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSinglePriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCancelPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.CancelPriceTriggeredOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s CancelPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestGetMarginAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMarginAccountList(context.Background(), currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMarginAccountList() error %v", g.Name, err)
	}
}

func TestListMarginAccountBalanceChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListMarginAccountBalanceChangeHistory(context.Background(), currency.BTC, currency.Pair{
		Base:      currency.BTC,
		Delimiter: currency.UnderscoreDelimiter,
		Quote:     currency.USDT}, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s ListMarginAccountBalanceChangeHistory() error %v", g.Name, err)
	}
}

func TestGetMarginFundingAccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMarginFundingAccountList(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMarginFundingAccountList %v", g.Name, err)
	}
}

func TestMarginLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.MarginLoan(context.Background(), &MarginLoanRequestParam{
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
	if _, err := g.GetMarginAllLoans(context.Background(), "open", "lend", "", currency.BTC, currency.Pair{Base: currency.BTC, Delimiter: currency.UnderscoreDelimiter, Quote: currency.USDT}, false, 0, 0); err != nil {
		t.Errorf("%s GetMarginAllLoans() error %v", g.Name, err)
	}
}

func TestMergeMultipleLendingLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.MergeMultipleLendingLoans(context.Background(), currency.USDT, []string{"123", "23423"}); err != nil {
		t.Errorf("%s MergeMultipleLendingLoans() error %v", g.Name, err)
	}
}

func TestRetriveOneSingleLoanDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.RetriveOneSingleLoanDetail(context.Background(), "borrow", "123"); err != nil {
		t.Errorf("%s RetriveOneSingleLoanDetail() error %v", g.Name, err)
	}
}

func TestModifyALoan(t *testing.T) {
	t.Parallel()
	if _, err := g.ModifyALoan(context.Background(), "1234", &ModifyLoanRequestParam{
		Currency:  currency.BTC,
		Side:      "borrow",
		AutoRenew: false,
	}); !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("%s ModifyALoan() error %v", g.Name, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.ModifyALoan(context.Background(), "1234", &ModifyLoanRequestParam{
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
	if _, err := g.CancelLendingLoan(context.Background(), currency.BTC, "1234"); err != nil {
		t.Errorf("%s CancelLendingLoan() error %v", g.Name, err)
	}
}

func TestRepayALoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.RepayALoan(context.Background(), "1234", &RepayLoanRequestParam{
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
	if _, err := g.ListLoanRepaymentRecords(context.Background(), "1234"); err != nil {
		t.Errorf("%s LoanRepaymentRecord() error %v", g.Name, err)
	}
}

func TestListRepaymentRecordsOfSpecificLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListRepaymentRecordsOfSpecificLoan(context.Background(), "1234", "", 0, 0); err != nil {
		t.Errorf("%s error while ListRepaymentRecordsOfSpecificLoan() %v", g.Name, err)
	}
}

func TestGetOneSingleloanRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOneSingleLoanRecord(context.Background(), "1234", "123"); err != nil {
		t.Errorf("%s error while GetOneSingleloanRecord() %v", g.Name, err)
	}
}

func TestModifyALoanRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ModifyALoanRecord(context.Background(), "1234", &ModifyLoanRequestParam{
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
	if _, err := g.UpdateUsersAutoRepaymentSetting(context.Background(), true); err != nil {
		t.Errorf("%s UpdateUsersAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetUserAutoRepaymentSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUserAutoRepaymentSetting(context.Background()); err != nil {
		t.Errorf("%s GetUserAutoRepaymentSetting() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxTransferableAmountForSpecificMarginCurrency(context.Background(), currency.BTC, currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxBorrowableAmountForSpecificMarginCurrency(context.Background(), currency.BTC, currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificMarginCurrency() error %v", g.Name, err)
	}
}

func TestCurrencySupportedByCrossMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.CurrencySupportedByCrossMargin(context.Background()); err != nil {
		t.Errorf("%s CurrencySupportedByCrossMargin() error %v", g.Name, err)
	}
}

func TestGetCrossMarginSupportedCurrencyDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginSupportedCurrencyDetail(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetCrossMarginSupportedCurrencyDetail() error %v", g.Name, err)
	}
}

func TestGetCrossMarginAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginAccounts(context.Background()); err != nil {
		t.Errorf("%s GetCrossMarginAccounts() error %v", g.Name, err)
	}
}

func TestGetCrossMarginAccountChangeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginAccountChangeHistory(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 6, "in"); err != nil {
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
	if _, err := g.CreateCrossMarginBorrowLoan(context.Background(), CrossMarginBorrowLoanParams{
		Currency: currency.BTC,
		Amount:   3,
	}); err != nil {
		t.Errorf("%s CreateCrossMarginBorrowLoan() error %v", g.Name, err)
	}
}

func TestGetCrossMarginBorrowHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginBorrowHistory(context.Background(), 1, currency.BTC, 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginBorrowHistory() error %v", g.Name, err)
	}
}

func TestGetSingleBorrowLoanDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleBorrowLoanDetail(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSingleBorrowLoanDetail() error %v", g.Name, err)
	}
}

func TestExecuteRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.ExecuteRepayment(context.Background(), CurrencyAndAmount{
		Currency: currency.USD,
		Amount:   1234.55,
	}); err != nil {
		t.Errorf("%s ExecuteRepayment() error %v", g.Name, err)
	}
}

func TestGetCrossMarginRepayments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetCrossMarginRepayments(context.Background(), currency.BTC, "123", 0, 0, false); err != nil {
		t.Errorf("%s GetCrossMarginRepayments() error %v", g.Name, err)
	}
}

func TestGetMaxTransferableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxTransferableAmountForSpecificCrossMarginCurrency(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxTransferableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestGetMaxBorrowableAmountForSpecificCrossMarginCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMaxBorrowableAmountForSpecificCrossMarginCurrency(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetMaxBorrowableAmountForSpecificCrossMarginCurrency() error %v", g.Name, err)
	}
}

func TestListCurrencyChain(t *testing.T) {
	t.Parallel()
	if _, err := g.ListCurrencyChain(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s ListCurrencyChain() error %v", g.Name, err)
	}
}

func TestGenerateCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GenerateCurrencyDepositAddress(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GenerateCurrencyDepositAddress() error %v", g.Name, err)
	}
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetWithdrawalRecords(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetWithdrawalRecords() error %v", g.Name, err)
	}
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDepositRecords(context.Background(), currency.BTC, time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetDepositRecords() error %v", g.Name, err)
	}
}

func TestTransferCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.TransferCurrency(context.Background(), &TransferCurrencyParam{
		Currency:     currency.BTC,
		From:         g.assetTypeToString(asset.Spot),
		To:           g.assetTypeToString(asset.Margin),
		Amount:       1202.000,
		CurrencyPair: spotTradablePair,
	}); err != nil {
		t.Errorf("%s TransferCurrency() error %v", g.Name, err)
	}
}

func TestSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.SubAccountTransfer(context.Background(), SubAccountTransferParam{
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
	if _, err := g.GetSubAccountTransferHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 0); err != nil {
		t.Errorf("%s GetSubAccountTransferHistory() error %v", g.Name, err)
	}
}

func TestSubAccountTransferToSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.SubAccountTransferToSubAccount(context.Background(), &InterSubAccountTransferParams{
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
	if _, err := g.GetWithdrawalStatus(context.Background(), currency.NewCode("")); err != nil {
		t.Errorf("%s GetWithdrawalStatus() error %v", g.Name, err)
	}
}

func TestGetSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountBalances(context.Background(), ""); err != nil {
		t.Errorf("%s GetSubAccountBalances() error %v", g.Name, err)
	}
}

func TestGetSubAccountMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountMarginBalances(context.Background(), ""); err != nil {
		t.Errorf("%s GetSubAccountMarginBalances() error %v", g.Name, err)
	}
}

func TestGetSubAccountFuturesBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountFuturesBalances(context.Background(), "", ""); err != nil {
		t.Errorf("%s GetSubAccountFuturesBalance() error %v", g.Name, err)
	}
}

func TestGetSubAccountCrossMarginBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccountCrossMarginBalances(context.Background(), ""); err != nil {
		t.Errorf("%s GetSubAccountCrossMarginBalances() error %v", g.Name, err)
	}
}

func TestGetSavedAddresses(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSavedAddresses(context.Background(), currency.BTC, "", 0); err != nil {
		t.Errorf("%s GetSavedAddresses() error %v", g.Name, err)
	}
}

func TestGetPersonalTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetPersonalTradingFee(context.Background(), currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}, ""); err != nil {
		t.Errorf("%s GetPersonalTradingFee() error %v", g.Name, err)
	}
}

func TestGetUsersTotalBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersTotalBalance(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetUsersTotalBalance() error %v", g.Name, err)
	}
}

func TestGetMarginSupportedCurrencyPairs(t *testing.T) {
	t.Parallel()
	if _, err := g.GetMarginSupportedCurrencyPairs(context.Background()); err != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, err)
	}
}

func TestGetMarginSupportedCurrencyPair(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSingleMarginSupportedCurrencyPair(context.Background(), marginTradablePair); err != nil {
		t.Errorf("%s GetMarginSupportedCurrencyPair() error %v", g.Name, err)
	}
}

func TestGetOrderbookOfLendingLoans(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOrderbookOfLendingLoans(context.Background(), currency.BTC); err != nil {
		t.Errorf("%s GetOrderbookOfLendingLoans() error %v", g.Name, err)
	}
}

func TestGetAllFutureContracts(t *testing.T) {
	t.Parallel()
	_, err := g.GetAllFutureContracts(context.Background(), settleUSD)
	if err != nil {
		t.Errorf("%s GetAllFutureContracts() error %v", g.Name, err)
	}
	if _, err = g.GetAllFutureContracts(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s GetAllFutureContracts() error %v", g.Name, err)
	}
	if _, err = g.GetAllFutureContracts(context.Background(), settleBTC); err != nil {
		t.Errorf("%s GetAllFutureContracts() error %v", g.Name, err)
	}
}
func TestGetSingleContract(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.GetSingleContract(context.Background(), settle, futuresTradablePair.String()); err != nil {
		t.Errorf("%s GetSingleContract() error %s", g.Name, err)
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetFuturesOrderbook(context.Background(), settle, futuresTradablePair.String(), "", 0, false); err != nil {
		t.Errorf("%s GetFuturesOrderbook() error %v", g.Name, err)
	}
}
func TestGetFuturesTradingHistory(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetFuturesTradingHistory(context.Background(), settle, futuresTradablePair, 0, 0, "", time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetFuturesTradingHistory() error %v", g.Name, err)
	}
}

func TestGetFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetFuturesCandlesticks(context.Background(), settle, futuresTradablePair.String(), time.Time{}, time.Time{}, 0, kline.OneWeek); err != nil {
		t.Errorf("%s GetFuturesCandlesticks() error %v", g.Name, err)
	}
}

func TestPremiumIndexKLine(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Error(err)
	}
	if _, err := g.PremiumIndexKLine(context.Background(), settle, futuresTradablePair, time.Time{}, time.Time{}, 0, kline.OneWeek); err != nil {
		t.Errorf("%s PremiumIndexKLine() error %v", g.Name, err)
	}
}

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetFuturesTickers(context.Background(), settle, futuresTradablePair); err != nil {
		t.Errorf("%s GetFuturesTickers() error %v", g.Name, err)
	}
}

func TestGetFutureFundingRates(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetFutureFundingRates(context.Background(), settle, futuresTradablePair, 0); err != nil {
		t.Errorf("%s GetFutureFundingRates() error %v", g.Name, err)
	}
}

func TestGetFuturesInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetFuturesInsuranceBalanceHistory(context.Background(), settleUSDT, 0); err != nil {
		t.Errorf("%s GetFuturesInsuranceBalanceHistory() error %v", g.Name, err)
	}
}

func TestGetFutureStats(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Error(err)
	}
	if _, err := g.GetFutureStats(context.Background(), settle, futuresTradablePair, time.Time{}, kline.OneHour, 0); err != nil {
		t.Errorf("%s GetFutureStats() error %v", g.Name, err)
	}
}

func TestGetIndexConstituent(t *testing.T) {
	t.Parallel()
	if _, err := g.GetIndexConstituent(context.Background(), settleUSDT, currency.Pair{Base: currency.BTC, Quote: currency.USDT, Delimiter: currency.UnderscoreDelimiter}.String()); err != nil {
		t.Errorf("%s GetIndexConstituent() error %v", g.Name, err)
	}
}

func TestGetLiquidationHistory(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Error(err)
	}
	if _, err := g.GetLiquidationHistory(context.Background(), settle, futuresTradablePair, time.Time{}, time.Time{}, 0); err != nil {
		t.Errorf("%s GetLiquidationHistory() error %v", g.Name, err)
	}
}
func TestQueryFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.QueryFuturesAccount(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s QueryFuturesAccount() error %v", g.Name, err)
	}
}

func TestGetFuturesAccountBooks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetFuturesAccountBooks(context.Background(), settleUSDT, 0, time.Time{}, time.Time{}, "dnw"); err != nil {
		t.Errorf("%s GetFuturesAccountBooks() error %v", g.Name, err)
	}
}

func TestGetAllPositionsOfUsers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetAllFuturesPositionsOfUsers(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s GetAllPositionsOfUsers() error %v", g.Name, err)
	}
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSinglePosition(context.Background(), settleUSDT, currency.Pair{Quote: currency.BTC, Base: currency.USDT}); err != nil {
		t.Errorf("%s GetSinglePosition() error %v", g.Name, err)
	}
}

func TestUpdatePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.UpdateFuturesPositionMargin(context.Background(), settle, 0.01, futuresTradablePair); err != nil {
		t.Errorf("%s UpdatePositionMargin() error %v", g.Name, err)
	}
}

func TestUpdatePositionLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.UpdateFuturesPositionLeverage(context.Background(), settle, futuresTradablePair, 1, 0); err != nil {
		t.Errorf("%s UpdatePositionLeverage() error %v", g.Name, err)
	}
}

func TestUpdatePositionRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.UpdateFuturesPositionRiskLimit(context.Background(), settle, futuresTradablePair, 10); err != nil {
		t.Errorf("%s UpdatePositionRiskLimit() error %v", g.Name, err)
	}
}

func TestCreateDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.PlaceDeliveryOrder(context.Background(), &OrderCreateParams{
		Contract:    deliveryFuturesTradablePair,
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		Text:        "t-my-custom-id",
		Settle:      settle,
		TimeInForce: gtcTIF,
	}); err != nil {
		t.Errorf("%s CreateDeliveryOrder() error %v", g.Name, err)
	}
}

func TestGetDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetDeliveryOrders(context.Background(), deliveryFuturesTradablePair, "open", settle, "", 0, 0, 1); err != nil {
		t.Errorf("%s GetDeliveryOrders() error %v", g.Name, err)
	}
}

func TestCancelAllDeliveryOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.CancelMultipleDeliveryOrders(context.Background(), deliveryFuturesTradablePair, "ask", settle); err != nil {
		t.Errorf("%s CancelAllDeliveryOrders() error %v", g.Name, err)
	}
}

func TestGetSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleDeliveryOrder(context.Background(), settleUSDT, "123456"); err != nil {
		t.Errorf("%s GetSingleDeliveryOrder() error %v", g.Name, err)
	}
	if _, err := g.GetSingleDeliveryOrder(context.Background(), settleBTC, "123456"); err != nil {
		t.Errorf("%s GetSingleDeliveryOrder() error %v", g.Name, err)
	}
	if _, err := g.GetSingleDeliveryOrder(context.Background(), settleUSD, "123456"); !errors.Is(err, errEmptySettlementCurrency) {
		t.Errorf("%s GetSingleDeliveryOrder() expected %v, but found %v", g.Name, errEmptySettlementCurrency, err)
	}
}

func TestCancelSingleDeliveryOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelSingleDeliveryOrder(context.Background(), settleUSDT, "123456"); err != nil {
		t.Errorf("%s CancelSingleDeliveryOrder() error %v", g.Name, err)
	}
	if _, err := g.CancelSingleDeliveryOrder(context.Background(), settleBTC, "123456"); err != nil {
		t.Errorf("%s CancelSingleDeliveryOrder() error %v", g.Name, err)
	}
}

func TestGetDeliveryPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryPersonalTradingHistory(context.Background(), settleUSDT, "", deliveryFuturesTradablePair, 0, 0, 1, ""); err != nil {
		t.Errorf("%s GetDeliveryPersonalTradingHistory() error %v", g.Name, err)
	}
}

func TestGetDeliveryPositionCloseHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryPositionCloseHistory(context.Background(), settleUSDT, deliveryFuturesTradablePair, 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetDeliveryPositionCloseHistory() error %v", g.Name, err)
	}
}

func TestGetDeliveryLiquidationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryLiquidationHistory(context.Background(), settleUSDT, deliveryFuturesTradablePair, 0, time.Now()); err != nil {
		t.Errorf("%s GetDeliveryLiquidationHistory() error %v", g.Name, err)
	}
}

func TestGetDeliverySettlementHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliverySettlementHistory(context.Background(), settleUSDT, deliveryFuturesTradablePair, 0, time.Now()); err != nil {
		t.Errorf("%s GetDeliverySettlementHistory() error %v", g.Name, err)
	}
}

func TestGetDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryPriceTriggeredOrder(context.Background(), settleUSDT, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     12,
			Contract: deliveryFuturesTradablePair,
		},
		Trigger: FuturesTrigger{
			Rule:      1,
			OrderType: "close-short-position",
			Price:     123400,
		},
	}); err != nil {
		t.Errorf("%s GetDeliveryPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestGetDeliveryAllAutoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryAllAutoOrder(context.Background(), "open", settleUSDT, deliveryFuturesTradablePair, 0, 1); err != nil {
		t.Errorf("%s GetDeliveryAllAutoOrder() error %v", g.Name, err)
	}
}

func TestCancelAllDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.CancelAllDeliveryPriceTriggeredOrder(context.Background(), settle, deliveryFuturesTradablePair); err != nil {
		t.Errorf("%s CancelAllDeliveryPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestGetSingleDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleDeliveryPriceTriggeredOrder(context.Background(), settleBTC, "12345"); err != nil {
		t.Errorf("%s GetSingleDeliveryPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCancelDeliveryPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelDeliveryPriceTriggeredOrder(context.Background(), settleUSDT, "12345"); err != nil {
		t.Errorf("%s CancelDeliveryPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestEnableOrDisableDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.EnableOrDisableDualMode(context.Background(), settleBTC, true); err != nil {
		t.Errorf("%s EnableOrDisableDualMode() error %v", g.Name, err)
	}
}

func TestRetrivePositionDetailInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.RetrivePositionDetailInDualMode(context.Background(), settle, futuresTradablePair); err != nil {
		t.Errorf("%s RetrivePositionDetailInDualMode() error %v", g.Name, err)
	}
}

func TestUpdatePositionMarginInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.UpdatePositionMarginInDualMode(context.Background(), settle, futuresTradablePair, 0.001, "dual_long"); err != nil {
		t.Errorf("%s UpdatePositionMarginInDualMode() error %v", g.Name, err)
	}
}
func TestUpdatePositionLeverageInDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.UpdatePositionLeverageInDualMode(context.Background(), settle, futuresTradablePair, 0.001, 0.001); err != nil {
		t.Errorf("%s UpdatePositionLeverageInDualMode() error %v", g.Name, err)
	}
}

func TestUpdatePositionRiskLimitinDualMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.UpdatePositionRiskLimitInDualMode(context.Background(), settle, futuresTradablePair, 10); err != nil {
		t.Errorf("%s UpdatePositionRiskLimitinDualMode() error %v", g.Name, err)
	}
}

func TestCreateFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.PlaceFuturesOrder(context.Background(), &OrderCreateParams{
		Contract:    futuresTradablePair,
		Size:        6024,
		Iceberg:     0,
		Price:       3765,
		TimeInForce: "gtc",
		Text:        "t-my-custom-id",
		Settle:      settle,
	}); err != nil {
		t.Errorf("%s CreateFuturesOrder() error %v", g.Name, err)
	}
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetFuturesOrders(context.Background(), currency.NewPair(currency.BTC, currency.USD), "open", "", settleBTC, 0, 0, 1); err != nil {
		t.Errorf("%s GetFuturesOrders() error %v", g.Name, err)
	}
}

func TestCancelMultipleFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelMultipleFuturesOpenOrders(context.Background(), futuresTradablePair, "ask", settleUSDT); err != nil {
		t.Errorf("%s CancelAllOpenOrdersMatched() error %v", g.Name, err)
	}
}

func TestGetSingleFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleFuturesPriceTriggeredOrder(context.Background(), settleBTC, "12345"); err != nil {
		t.Errorf("%s GetSingleFuturesPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCancelFuturesPriceTriggeredOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelFuturesPriceTriggeredOrder(context.Background(), settleUSDT, "12345"); err != nil {
		t.Errorf("%s CancelFuturesPriceTriggeredOrder() error %v", g.Name, err)
	}
}

func TestCreateBatchFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.PlaceBatchFuturesOrders(context.Background(), settleBTC, []OrderCreateParams{
		{
			Contract:    futuresTradablePair,
			Size:        6024,
			Iceberg:     0,
			Price:       3765,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      settle,
		},
		{
			Contract:    currency.NewPair(currency.BTC, currency.USDT),
			Size:        232,
			Iceberg:     0,
			Price:       376225,
			TimeInForce: "gtc",
			Text:        "t-my-custom-id",
			Settle:      settleBTC,
		},
	}); err != nil {
		t.Errorf("%s CreateBatchFuturesOrders() error %v", g.Name, err)
	}
}

func TestGetSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleFuturesOrder(context.Background(), settleBTC, "12345"); err != nil {
		t.Errorf("%s GetSingleFuturesOrder() error %v", g.Name, err)
	}
}
func TestCancelSingleFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelSingleFuturesOrder(context.Background(), settleBTC, "12345"); err != nil {
		t.Errorf("%s CancelSingleFuturesOrder() error %v", g.Name, err)
	}
}
func TestAmendFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.AmendFuturesOrder(context.Background(), settleBTC, "1234", AmendFuturesOrderParam{
		Price: 12345.990,
	}); err != nil {
		t.Errorf("%s AmendFuturesOrder() error %v", g.Name, err)
	}
}

func TestGetMyPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetMyPersonalTradingHistory(context.Background(), settleBTC, "", "", futuresTradablePair, 0, 0, 0); err != nil {
		t.Errorf("%s GetMyPersonalTradingHistory() error %v", g.Name, err)
	}
}

func TestGetPositionCloseHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetFuturesPositionCloseHistory(context.Background(), settleBTC, futuresTradablePair, 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetPositionCloseHistory() error %v", g.Name, err)
	}
}

func TestGetFuturesLiquidationHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetFuturesLiquidationHistory(context.Background(), settleBTC, futuresTradablePair, 0, time.Time{}); err != nil {
		t.Errorf("%s GetFuturesLiquidationHistory() error %v", g.Name, err)
	}
}

func TestCountdownCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CountdownCancelOrders(context.Background(), settleBTC, CountdownParams{
		Timeout: 8,
	}); err != nil {
		t.Errorf("%s CountdownCancelOrders() error %v", g.Name, err)
	}
}

func TestCreatePriceTriggeredFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.CreatePriceTriggeredFuturesOrder(context.Background(), settle, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     2,
			Contract: futuresTradablePair,
		},
		Trigger: FuturesTrigger{
			Rule:      1,
			OrderType: "close-short-position",
		},
	}); err != nil {
		t.Errorf("%s CreatePriceTriggeredFuturesOrder() error %v", g.Name, err)
	}
	if _, err := g.CreatePriceTriggeredFuturesOrder(context.Background(), settle, &FuturesPriceTriggeredOrderParam{
		Initial: FuturesInitial{
			Price:    1234.,
			Size:     1,
			Contract: futuresTradablePair,
		},
		Trigger: FuturesTrigger{
			Rule: 1,
		},
	}); err != nil {
		t.Errorf("%s CreatePriceTriggeredFuturesOrder() error %v", g.Name, err)
	}
}

func TestListAllFuturesAutoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.ListAllFuturesAutoOrders(context.Background(), "open", settleBTC, currency.EMPTYPAIR, 0, 0); err != nil {
		t.Errorf("%s ListAllFuturesAutoOrders() error %v", g.Name, err)
	}
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(futuresTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.CancelAllFuturesOpenOrders(context.Background(), settle, futuresTradablePair); err != nil {
		t.Errorf("%s CancelAllFuturesOpenOrders() error %v", g.Name, err)
	}
}

func TestGetAllDeliveryContracts(t *testing.T) {
	t.Parallel()
	if _, err := g.GetAllDeliveryContracts(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s GetAllDeliveryContracts() error %v", g.Name, err)
	}
}

func TestGetSingleDeliveryContracts(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Skip(err)
	}
	if _, err := g.GetSingleDeliveryContracts(context.Background(), settle, deliveryFuturesTradablePair); err != nil {
		t.Errorf("%s GetSingleDeliveryContracts() error %v", g.Name, err)
	}
}

func TestGetDeliveryOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := g.GetDeliveryOrderbook(context.Background(), settleUSDT, "0", deliveryFuturesTradablePair, 0, false); err != nil {
		t.Errorf("%s GetDeliveryOrderbook() error %v", g.Name, err)
	}
}

func TestGetDeliveryTradingHistory(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Skip(err)
	}
	if _, err := g.GetDeliveryTradingHistory(context.Background(), settle, "", deliveryFuturesTradablePair, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetDeliveryTradingHistory() error %v", g.Name, err)
	}
}
func TestGetDeliveryFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Skip(err)
	}
	if _, err := g.GetDeliveryFuturesCandlesticks(context.Background(), settle, deliveryFuturesTradablePair, time.Time{}, time.Time{}, 0, kline.OneWeek); err != nil {
		t.Errorf("%s GetFuturesCandlesticks() error %v", g.Name, err)
	}
}

func TestGetDeliveryFutureTickers(t *testing.T) {
	t.Parallel()
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Skip(err)
	}
	if _, err := g.GetDeliveryFutureTickers(context.Background(), settle, deliveryFuturesTradablePair); err != nil {
		t.Errorf("%s GetDeliveryFutureTickers() error %v", g.Name, err)
	}
}

func TestGetDeliveryInsuranceBalanceHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetDeliveryInsuranceBalanceHistory(context.Background(), settleBTC, 0); err != nil {
		t.Errorf("%s GetDeliveryInsuranceBalanceHistory() error %v", g.Name, err)
	}
}

func TestQueryDeliveryFuturesAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryFuturesAccounts(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s QueryDeliveryFuturesAccounts() error %v", g.Name, err)
	}
}
func TestGetDeliveryAccountBooks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetDeliveryAccountBooks(context.Background(), settleUSDT, 0, time.Time{}, time.Now(), "dnw"); err != nil {
		t.Errorf("%s GetDeliveryAccountBooks() error %v", g.Name, err)
	}
}

func TestGetAllDeliveryPositionsOfUser(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetAllDeliveryPositionsOfUser(context.Background(), settleUSDT); err != nil {
		t.Errorf("%s GetAllDeliveryPositionsOfUser() error %v", g.Name, err)
	}
}

func TestGetSingleDeliveryPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleDeliveryPosition(context.Background(), settleUSDT, deliveryFuturesTradablePair); err != nil {
		t.Errorf("%s GetSingleDeliveryPosition() error %v", g.Name, err)
	}
}

func TestUpdateDeliveryPositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	settle, err := g.getSettlementFromCurrency(deliveryFuturesTradablePair, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.UpdateDeliveryPositionMargin(context.Background(), settle, 0.001, deliveryFuturesTradablePair); err != nil {
		t.Errorf("%s UpdateDeliveryPositionMargin() error %v", g.Name, err)
	}
}

func TestUpdateDeliveryPositionLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.UpdateDeliveryPositionLeverage(context.Background(), "usdt", deliveryFuturesTradablePair, 0.001); err != nil {
		t.Errorf("%s UpdateDeliveryPositionLeverage() error %v", g.Name, err)
	}
}

func TestUpdateDeliveryPositionRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.UpdateDeliveryPositionRiskLimit(context.Background(), "usdt", deliveryFuturesTradablePair, 30); err != nil {
		t.Errorf("%s UpdateDeliveryPositionRiskLimit() error %v", g.Name, err)
	}
}

func TestGetAllOptionsUnderlyings(t *testing.T) {
	t.Parallel()
	if _, err := g.GetAllOptionsUnderlyings(context.Background()); err != nil {
		t.Errorf("%s GetAllOptionsUnderlyings() error %v", g.Name, err)
	}
}

func TestGetExpirationTime(t *testing.T) {
	t.Parallel()
	if _, err := g.GetExpirationTime(context.Background(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetExpirationTime() error %v", g.Name, err)
	}
}

func TestGetAllContractOfUnderlyingWithinExpiryDate(t *testing.T) {
	t.Parallel()
	if _, err := g.GetAllContractOfUnderlyingWithinExpiryDate(context.Background(), "BTC_USDT", time.Time{}); err != nil {
		t.Errorf("%s GetAllContractOfUnderlyingWithinExpiryDate() error %v", g.Name, err)
	}
}

func TestGetOptionsSpecifiedContractDetail(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsSpecifiedContractDetail(context.Background(), optionsTradablePair); err != nil {
		t.Errorf("%s GetOptionsSpecifiedContractDetail() error %v", g.Name, err)
	}
}

func TestGetSettlementHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSettlementHistory(context.Background(), "BTC_USDT", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetSettlementHistory() error %v", g.Name, err)
	}
}

func TestGetOptionsSpecifiedSettlementHistory(t *testing.T) {
	t.Parallel()
	underlying := "BTC_USDT"
	optionsSettlement, err := g.GetSettlementHistory(context.Background(), underlying, 0, 1, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	cp, err := currency.NewPairFromString(optionsSettlement[0].Contract)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.GetOptionsSpecifiedContractsSettlement(context.Background(), cp, underlying, optionsSettlement[0].Timestamp.Time().Unix()); err != nil {
		t.Errorf("%s GetOptionsSpecifiedContractsSettlement() error %s", g.Name, err)
	}
}

func TestGetSupportedFlashSwapCurrencies(t *testing.T) {
	t.Parallel()
	if _, err := g.GetSupportedFlashSwapCurrencies(context.Background()); err != nil {
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
	if _, err := g.CreateFlashSwapOrder(context.Background(), FlashSwapOrderParams{
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
	if _, err := g.GetAllFlashSwapOrders(context.Background(), 1, currency.EMPTYCODE, currency.EMPTYCODE, true, 0, 0); err != nil {
		t.Errorf("%s GetAllFlashSwapOrders() error %v", g.Name, err)
	}
}

func TestGetSingleFlashSwapOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleFlashSwapOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSingleFlashSwapOrder() error %v", g.Name, err)
	}
}

func TestInitiateFlashSwapOrderReview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.InitiateFlashSwapOrderReview(context.Background(), FlashSwapOrderParams{
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
	if _, err := g.GetMyOptionsSettlements(context.Background(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}); err != nil {
		t.Errorf("%s GetMyOptionsSettlements() error %v", g.Name, err)
	}
}

func TestGetOptionAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOptionAccounts(context.Background()); err != nil {
		t.Errorf("%s GetOptionAccounts() error %v", g.Name, err)
	}
}

func TestGetAccountChangingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetAccountChangingHistory(context.Background(), 0, 0, time.Time{}, time.Time{}, ""); err != nil {
		t.Errorf("%s GetAccountChangingHistory() error %v", g.Name, err)
	}
}

func TestGetUsersPositionSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersPositionSpecifiedUnderlying(context.Background(), ""); err != nil {
		t.Errorf("%s GetUsersPositionSpecifiedUnderlying() error %v", g.Name, err)
	}
}

func TestGetSpecifiedContractPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	_, err := g.GetSpecifiedContractPosition(context.Background(), currency.EMPTYPAIR)
	if err != nil && !errors.Is(err, errInvalidOrMissingContractParam) {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, err)
	}
	_, err = g.GetSpecifiedContractPosition(context.Background(), optionsTradablePair)
	if err != nil {
		t.Errorf("%s GetSpecifiedContractPosition() error expecting %v, but found %v", g.Name, errInvalidOrMissingContractParam, err)
	}
}

func TestGetUsersLiquidationHistoryForSpecifiedUnderlying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetUsersLiquidationHistoryForSpecifiedUnderlying(context.Background(), "BTC_USDT", currency.EMPTYPAIR); err != nil {
		t.Errorf("%s GetUsersLiquidationHistoryForSpecifiedUnderlying() error %v", g.Name, err)
	}
}

func TestPlaceOptionOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.PlaceOptionOrder(context.Background(), OptionOrderParam{
		Contract:    optionsTradablePair.String(),
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
	if _, err := g.GetOptionFuturesOrders(context.Background(), currency.EMPTYPAIR, "", "", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetOptionFuturesOrders() error %v", g.Name, err)
	}
}

func TestCancelOptionOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelMultipleOptionOpenOrders(context.Background(), optionsTradablePair, "", ""); err != nil {
		t.Errorf("%s CancelOptionOpenOrders() error %v", g.Name, err)
	}
}
func TestGetSingleOptionOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleOptionOrder(context.Background(), ""); err != nil && !errors.Is(errInvalidOrderID, err) {
		t.Errorf("%s GetSingleOptionorder() expecting %v, but found %v", g.Name, errInvalidOrderID, err)
	}
	if _, err := g.GetSingleOptionOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s GetSingleOptionOrder() error %v", g.Name, err)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CancelOptionSingleOrder(context.Background(), "1234"); err != nil {
		t.Errorf("%s CancelSingleOrder() error %v", g.Name, err)
	}
}

func TestGetOptionsPersonalTradingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetOptionsPersonalTradingHistory(context.Background(), "BTC_USDT", currency.EMPTYPAIR, 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetOptionPersonalTradingHistory() error %v", g.Name, err)
	}
}

func TestWithdrawCurrency(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.WithdrawCurrency(context.Background(), WithdrawalRequestParam{})
	if err != nil && !errors.Is(err, errInvalidAmount) {
		t.Errorf("%s WithdrawCurrency() expecting error %v, but found %v", g.Name, errInvalidAmount, err)
	}
	_, err = g.WithdrawCurrency(context.Background(), WithdrawalRequestParam{
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
	if _, err := g.CancelWithdrawalWithSpecifiedID(context.Background(), "1234567"); err != nil {
		t.Errorf("%s CancelWithdrawalWithSpecifiedID() error %v", g.Name, err)
	}
}

func TestGetOptionsOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsOrderbook(context.Background(), optionsTradablePair, "0.1", 9, true); err != nil {
		t.Errorf("%s GetOptionsFuturesOrderbooks() error %v", g.Name, err)
	}
}

func TestGetOptionsTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsTickers(context.Background(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetOptionsTickers() error %v", g.Name, err)
	}
}

func TestGetOptionUnderlyingTickers(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionUnderlyingTickers(context.Background(), "BTC_USDT"); err != nil {
		t.Errorf("%s GetOptionUnderlyingTickers() error %v", g.Name, err)
	}
}

func TestGetOptionFuturesCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionFuturesCandlesticks(context.Background(), optionsTradablePair, 0, time.Now().Add(-time.Hour*10), time.Time{}, kline.ThirtyMin); err != nil {
		t.Error(err)
	}
}

func TestGetOptionFuturesMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionFuturesMarkPriceCandlesticks(context.Background(), "BTC_USDT", 0, time.Time{}, time.Time{}, kline.OneMonth); err != nil {
		t.Errorf("%s GetOptionFuturesMarkPriceCandlesticks() error %v", g.Name, err)
	}
}

func TestGetOptionsTradeHistory(t *testing.T) {
	t.Parallel()
	if _, err := g.GetOptionsTradeHistory(context.Background(), optionsTradablePair, "C", 0, 0, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetOptionsTradeHistory() error %v", g.Name, err)
	}
}

// Sub-account endpoints

func TestCreateNewSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if _, err := g.CreateNewSubAccount(context.Background(), SubAccountParams{
		LoginName: "Sub_Account_for_testing",
	}); err != nil {
		t.Errorf("%s CreateNewSubAccount() error %v", g.Name, err)
	}
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSubAccounts(context.Background()); err != nil {
		t.Errorf("%s GetSubAccounts() error %v", g.Name, err)
	}
}

func TestGetSingleSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetSingleSubAccount(context.Background(), "123423"); err != nil {
		t.Errorf("%s GetSingleSubAccount() error %v", g.Name, err)
	}
}

// Wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := g.FetchTradablePairs(context.Background(), asset.DeliveryFutures)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	if _, err = g.FetchTradablePairs(context.Background(), asset.Options); err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	if _, err = g.FetchTradablePairs(context.Background(), asset.Margin); err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(context.Background(), asset.CrossMargin)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
	_, err = g.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Errorf("%s FetchTradablePairs() error %v", g.Name, err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if err := g.UpdateTradablePairs(context.Background(), true); err != nil {
		t.Errorf("%s UpdateTradablePairs() error %v", g.Name, err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if err := g.UpdateTickers(context.Background(), asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(context.Background(), asset.Futures); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(context.Background(), asset.Spot); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(context.Background(), asset.Options); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(context.Background(), asset.CrossMargin); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
	if err := g.UpdateTickers(context.Background(), asset.Margin); err != nil {
		t.Errorf("%s UpdateTickers() error %v", g.Name, err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(context.Background(), marginTradablePair, asset.Margin)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(context.Background(), crossMarginTradablePair, asset.CrossMargin)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	_, err = g.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	if _, err = g.UpdateOrderbook(context.Background(), deliveryFuturesTradablePair, asset.DeliveryFutures); err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
	if _, err = g.UpdateOrderbook(context.Background(), optionsTradablePair, asset.Options); err != nil {
		t.Errorf("%s UpdateOrderbook() error %v", g.Name, err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if _, err := g.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Empty); err != nil {
		t.Errorf("%s GetWithdrawalsHistory() error %v", g.Name, err)
	}
}
func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := g.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(context.Background(), marginTradablePair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(context.Background(), crossMarginTradablePair, asset.CrossMargin)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(context.Background(), deliveryFuturesTradablePair, asset.DeliveryFutures)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetRecentTrades(context.Background(), optionsTradablePair, asset.Options)
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      crossMarginTradablePair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.CrossMargin,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      spotTradablePair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      optionsTradablePair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Options,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      deliveryFuturesTradablePair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.DeliveryFutures,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      futuresTradablePair,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf("Order failed to be placed: %v", err)
	}
	_, err = g.SubmitOrder(context.Background(), &order.Submit{
		Exchange:  g.Name,
		Pair:      marginTradablePair,
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
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	err := g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Margin
	err = g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.CrossMargin
	err = g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Options
	err = g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.Futures
	err = g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	orderCancellation.AssetType = asset.DeliveryFutures
	err = g.CancelOrder(context.Background(), orderCancellation)
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	_, err := g.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          spotTradablePair,
			AssetType:     asset.Spot,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          spotTradablePair,
			AssetType:     asset.Spot,
		}})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          futuresTradablePair,
			AssetType:     asset.Futures,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          futuresTradablePair,
			AssetType:     asset.Futures,
		}})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          deliveryFuturesTradablePair,
			AssetType:     asset.DeliveryFutures,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          deliveryFuturesTradablePair,
			AssetType:     asset.DeliveryFutures,
		}})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          optionsTradablePair,
			AssetType:     asset.Options,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          optionsTradablePair,
			AssetType:     asset.Options,
		}})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
	_, err = g.CancelBatchOrders(context.Background(), []order.Cancel{
		{
			OrderID:       "1",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          marginTradablePair,
			AssetType:     asset.Margin,
		}, {
			OrderID:       "2",
			WalletAddress: core.BitcoinDonationAddress,
			AccountID:     "1",
			Pair:          marginTradablePair,
			AssetType:     asset.Margin,
		}})
	if err != nil {
		t.Errorf("%s CancelOrder error: %v", g.Name, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	chains, err := g.GetAvailableTransferChains(context.Background(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
	for i := range chains {
		_, err = g.GetDepositAddress(context.Background(), currency.BTC, "", chains[i])
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
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
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
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     []currency.Pair{cp},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     enabledPairs[:2],
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Margin,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     enabledPairs[:2],
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.CrossMargin,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{futuresTradablePair},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Futures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{deliveryFuturesTradablePair},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.DeliveryFutures,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	_, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
		Pairs:     currency.Pairs{optionsTradablePair},
		Type:      order.AnyType,
		Side:      order.AnySide,
		AssetType: asset.Options,
	})
	if err != nil {
		t.Errorf(" %s GetActiveOrders() error: %v", g.Name, err)
	}
	if _, err = g.GetActiveOrders(context.Background(), &order.MultiOrderRequest{
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
	var multiOrderRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	enabledPairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	multiOrderRequest.Pairs = enabledPairs[:3]
	_, err = g.GetOrderHistory(context.Background(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.Futures
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	multiOrderRequest.Pairs = multiOrderRequest.Pairs[len(multiOrderRequest.Pairs)-4:]
	_, err = g.GetOrderHistory(context.Background(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.DeliveryFutures
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.DeliveryFutures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetOrderHistory(context.Background(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
	multiOrderRequest.AssetType = asset.Options
	multiOrderRequest.Pairs, err = g.GetEnabledPairs(asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetOrderHistory(context.Background(), &multiOrderRequest)
	if err != nil {
		t.Errorf("%s GetOrderhistory() error: %v", g.Name, err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 10)
	if _, err := g.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), marginTradablePair, asset.Margin, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), crossMarginTradablePair, asset.CrossMargin, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), deliveryFuturesTradablePair, asset.DeliveryFutures, kline.OneDay, startTime, time.Now()); err != nil {
		t.Errorf("%s GetHistoricCandles() error: %v", g.Name, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandles() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
	if _, err := g.GetHistoricCandles(context.Background(), optionsTradablePair, asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandles() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime := time.Now().Add(-time.Hour * 5)
	_, err := g.GetHistoricCandlesExtended(context.Background(),
		spotTradablePair, asset.Spot, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricCandlesExtended(context.Background(),
		marginTradablePair, asset.Margin, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetHistoricCandlesExtended(context.Background(),
		deliveryFuturesTradablePair, asset.DeliveryFutures, kline.OneMin, time.Now().Add(-time.Hour*5), time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetHistoricCandlesExtended(context.Background(),
		crossMarginTradablePair, asset.CrossMargin, kline.OneMin, startTime, time.Now())
	if err != nil {
		t.Error(err)
	}
	if _, err = g.GetHistoricCandlesExtended(context.Background(), optionsTradablePair, asset.Options, kline.OneDay, startTime, time.Now()); !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("%s GetHistoricCandlesExtended() expecting: %v, but found %v", g.Name, asset.ErrNotSupported, err)
	}
}
func TestGetAvailableTransferTrains(t *testing.T) {
	t.Parallel()
	_, err := g.GetAvailableTransferChains(context.Background(), currency.USDT)
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
	if err := g.wsHandleData([]byte(wsTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket ticker push data error: %v", g.Name, err)
	}
}

const wsTradePushDataJSON = `{	"time": 1606292218,	"channel": "spot.trades",	"event": "update",	"result": {	  "id": 309143071,	  "create_time": 1606292218,	  "create_time_ms": "1606292218213.4578",	  "side": "sell",	  "currency_pair": "BTC_USDT",	  "amount": "16.4700000000",	  "price": "0.4705000000"}}`

func TestWsTradePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsTradePushDataJSON)); err != nil {
		t.Errorf("%s websocket trade push data error: %v", g.Name, err)
	}
}

const wsCandlestickPushDataJSON = `{"time": 1606292600,	"channel": "spot.candlesticks",	"event": "update",	"result": {	  "t": "1606292580",	  "v": "2362.32035",	  "c": "19128.1",	  "h": "19128.1",	  "l": "19128.1",	  "o": "19128.1","n": "1m_BTC_USDT"}}`

func TestWsCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsCandlestickPushDataJSON)); err != nil {
		t.Errorf("%s websocket candlestick push data error: %v", g.Name, err)
	}
}

const wsOrderbookTickerJSON = `{"time": 1606293275,	"channel": "spot.book_ticker",	"event": "update",	"result": {	  "t": 1606293275123,	  "u": 48733182,	  "s": "BTC_USDT",	  "b": "19177.79",	  "B": "0.0003341504",	  "a": "19179.38",	  "A": "0.09"	}}`

func TestWsOrderbookTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsOrderbookTickerJSON)); err != nil {
		t.Errorf("%s websocket orderbook push data error: %v", g.Name, err)
	}
}

const (
	wsOrderbookUpdatePushDataJSON   = `{"time": 1606294781,	"channel": "spot.order_book_update",	"event": "update",	"result": {	  "t": 1606294781123,	  "e": "depthUpdate",	  "E": 1606294781,"s": "BTC_USDT","U": 48776301,"u": 48776306,"b": [["19137.74","0.0001"],["19088.37","0"]],"a": [["19137.75","0.6135"]]	}}`
	wsOrderbookSnapshotPushDataJSON = `{"time":1606295412,"channel": "spot.order_book",	"event": "update",	"result": {	  "t": 1606295412123,	  "lastUpdateId": 48791820,	  "s": "BTC_USDT",	  "bids": [		[		  "19079.55",		  "0.0195"		],		[		  "19079.07",		  "0.7341"],["19076.23",		  "0.00011808"		],		[		  "19073.9",		  "0.105"		],		[		  "19068.83",		  "0.1009"		]	  ],	  "asks": [		[		  "19080.24",		  "0.1638"		],		[		  "19080.91","0.1366"],["19080.92","0.01"],["19081.29","0.01"],["19083.8","0.097"]]}}`
)

func TestWsOrderbookSnapshotPushData(t *testing.T) {
	t.Parallel()
	err := g.wsHandleData([]byte(wsOrderbookSnapshotPushDataJSON))
	if err != nil {
		t.Errorf("%s websocket orderbook snapshot push data error: %v", g.Name, err)
	}
	if err = g.wsHandleData([]byte(wsOrderbookUpdatePushDataJSON)); err != nil {
		t.Errorf("%s websocket orderbook update push data error: %v", g.Name, err)
	}
}

const wsSpotOrderPushDataJSON = `{"time": 1605175506,	"channel": "spot.orders",	"event": "update",	"result": [	  {		"id": "30784435",		"user": 123456,		"text": "t-abc",		"create_time": "1605175506",		"create_time_ms": "1605175506123",		"update_time": "1605175506",		"update_time_ms": "1605175506123",		"event": "put",		"currency_pair": "BTC_USDT",		"type": "limit",		"account": "spot",		"side": "sell",		"amount": "1",		"price": "10001",		"time_in_force": "gtc",		"left": "1",		"filled_total": "0",		"fee": "0",		"fee_currency": "USDT",		"point_fee": "0",		"gt_fee": "0",		"gt_discount": true,		"rebated_fee": "0",		"rebated_fee_currency": "USDT"}	]}`

func TestWsPushOrders(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsSpotOrderPushDataJSON)); err != nil {
		t.Errorf("%s websocket orders push data error: %v", g.Name, err)
	}
}

const wsUserTradePushDataJSON = `{"time": 1605176741,	"channel": "spot.usertrades",	"event": "update",	"result": [	  {		"id": 5736713,		"user_id": 1000001,		"order_id": "30784428",		"currency_pair": "BTC_USDT",		"create_time": 1605176741,		"create_time_ms": "1605176741123.456",		"side": "sell",		"amount": "1.00000000",		"role": "taker",		"price": "10000.00000000",		"fee": "0.00200000000000",		"point_fee": "0",		"gt_fee": "0",		"text": "apiv4"	  }	]}`

func TestWsUserTradesPushDataJSON(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsUserTradePushDataJSON)); err != nil {
		t.Errorf("%s websocket users trade push data error: %v", g.Name, err)
	}
}

const wsBalancesPushDataJSON = `{"time": 1605248616,	"channel": "spot.balances",	"event": "update",	"result": [	  {		"timestamp": "1605248616",		"timestamp_ms": "1605248616123",		"user": "1000001",		"currency": "USDT",		"change": "100",		"total": "1032951.325075926",		"available": "1022943.325075926"}	]}`

func TestBalancesPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsBalancesPushDataJSON)); err != nil {
		t.Errorf("%s websocket balances push data error: %v", g.Name, err)
	}
}

const wsMarginBalancePushDataJSON = `{"time": 1605248616,	"channel": "spot.funding_balances",	"event": "update",	"result": [	  {"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT","change": "100","freeze": "100","lent": "0"}	]}`

func TestMarginBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsMarginBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket margin balance push data error: %v", g.Name, err)
	}
}

const wsCrossMarginBalancePushDataJSON = `{"time": 1605248616,"channel": "spot.cross_balances","event": "update",	"result": [{"timestamp": "1605248616","timestamp_ms": "1605248616123","user": "1000001","currency": "USDT",	"change": "100","total": "1032951.325075926","available": "1022943.325075926"}]}`

func TestCrossMarginBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsCrossMarginBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket cross margin balance push data error: %v", g.Name, err)
	}
}

const wsCrossMarginBalanceLoan = `{	"time":1658289372,	"channel":"spot.cross_loan",	"event":"update",	"result":{	  "timestamp":1658289372338,	  "user":"1000001",	  "currency":"BTC",	  "change":"0.01",	  "total":"4.992341029566",	  "available":"0.078054772536",	  "borrowed":"0.01",	  "interest":"0.00001375"	}}`

func TestCrossMarginBalanceLoan(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleData([]byte(wsCrossMarginBalanceLoan)); err != nil {
		t.Errorf("%s websocket cross margin loan push data error: %v", g.Name, err)
	}
}

const wsFuturesTickerPushDataJSON = `{"time": 1541659086,	"channel": "futures.tickers","event": "update",	"error": null,	"result": [	  {		"contract": "BTC_USD","last": "118.4","change_percentage": "0.77","funding_rate": "-0.000114","funding_rate_indicative": "0.01875","mark_price": "118.35","index_price": "118.36","total_size": "73648","volume_24h": "745487577","volume_24h_btc": "117",		"volume_24h_usd": "419950",		"quanto_base_rate": "",		"volume_24h_quote": "1665006","volume_24h_settle": "178","volume_24h_base": "5526","low_24h": "99.2","high_24h": "132.5"}	]}`

func TestFuturesTicker(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesTickerPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket push data error: %v", g.Name, err)
	}
}

const wsFuturesTradesPushDataJSON = `{"channel": "futures.trades","event": "update",	"time": 1541503698,	"result": [{"size": -108,"id": 27753479,"create_time": 1545136464,"create_time_ms": 1545136464123,"price": "96.4","contract": "BTC_USD"}]}`

func TestFuturesTrades(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesTradesPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket push data error: %v", g.Name, err)
	}
}

const (
	wsFuturesOrderbookTickerJSON = `{	"time": 1615366379,	"channel": "futures.book_ticker",	"event": "update",	"error": null,	"result": {	  "t": 1615366379123,	  "u": 2517661076,	  "s": "BTC_USD",	  "b": "54696.6",	  "B": 37000,	  "a": "54696.7",	  "A": 47061	}}`
)

func TestOrderbookData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesOrderbookTickerJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket orderbook ticker push data error: %v", g.Name, err)
	}
}

const wsFuturesOrderPushDataJSON = `{	"channel": "futures.orders",	"event": "update",	"time": 1541505434,	"result": [	  {		"contract": "BTC_USD",		"create_time": 1628736847,		"create_time_ms": 1628736847325,		"fill_price": 40000.4,		"finish_as": "filled",		"finish_time": 1628736848,		"finish_time_ms": 1628736848321,		"iceberg": 0,		"id": 4872460,		"is_close": false,		"is_liq": false,		"is_reduce_only": false,		"left": 0,		"mkfr": -0.00025,		"price": 40000.4,		"refr": 0,		"refu": 0,		"size": 1,		"status": "finished",		"text": "-",		"tif": "gtc",		"tkfr": 0.0005,		"user": "110xxxxx"	  }	]}`

func TestFuturesOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesOrderPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures order push data error: %v", g.Name, err)
	}
}

const wsFuturesUsertradesPushDataJSON = `{"time": 1543205083,	"channel": "futures.usertrades","event": "update",	"error": null,	"result": [{"id": "3335259","create_time": 1628736848,"create_time_ms": 1628736848321,"contract": "BTC_USD","order_id": "4872460","size": 1,"price": "40000.4","role": "maker","text": "api","fee": 0.0009290592,"point_fee": 0}]}`

func TestFuturesUserTrades(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesUsertradesPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures user trades push data error: %v", g.Name, err)
	}
}

const wsFuturesLiquidationPushDataJSON = `{"channel": "futures.liquidates",	"event": "update",	"time": 1541505434,	"result": [{"entry_price": 209,"fill_price": 215.1,"left": 0,"leverage": 0.0,"liq_price": 213,"margin": 0.007816722941,"mark_price": 213,"order_id": 4093362,"order_price": 215.1,"size": -124,"time": 1541486601,"time_ms": 1541486601123,"contract": "BTC_USD","user": "1040xxxx"}	]}`

func TestFuturesLiquidationPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesLiquidationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures liquidation push data error: %v", g.Name, err)
	}
}

const wsFuturesAutoDelevergesNotification = `{"channel": "futures.auto_deleverages",	"event": "update",	"time": 1541505434,	"result": [{"entry_price": 209,"fill_price": 215.1,"position_size": 10,"trade_size": 10,"time": 1541486601,"time_ms": 1541486601123,"contract": "BTC_USD","user": "1040"}	]}`

func TestFuturesAutoDeleverges(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesAutoDelevergesNotification), asset.Futures); err != nil {
		t.Errorf("%s websocket futures auto deleverge push data error: %v", g.Name, err)
	}
}

const wsFuturesPositionClosePushDataJSON = ` {"channel": "futures.position_closes",	"event": "update",	"time": 1541505434,	"result": [	  {		"contract": "BTC_USD",		"pnl": -0.000624354791,		"side": "long",		"text": "web",		"time": 1547198562,		"time_ms": 1547198562123,		"user": "211xxxx"	  }	]}`

func TestPositionClosePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesPositionClosePushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures position close push data error: %v", g.Name, err)
	}
}

const wsFuturesBalanceNotificationPushDataJSON = `{"channel": "futures.balances",	"event": "update",	"time": 1541505434,	"result": [	  {		"balance": 9.998739899488,		"change": -0.000002074115,		"text": "BTC_USD:3914424",		"time": 1547199246,		"time_ms": 1547199246123,		"type": "fee",		"user": "211xxx"	  }	]}`

func TestFuturesBalanceNotification(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesBalanceNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures balance notification push data error: %v", g.Name, err)
	}
}

const wsFuturesReduceRiskLimitNotificationPushDataJSON = `{"time": 1551858330,	"channel": "futures.reduce_risk_limits",	"event": "update",	"error": null,	"result": [	  {		"cancel_orders": 0,		"contract": "ETH_USD",		"leverage_max": 10,		"liq_price": 136.53,		"maintenance_rate": 0.09,		"risk_limit": 450,		"time": 1551858330,		"time_ms": 1551858330123,		"user": "20011"	  }	]}`

func TestFuturesReduceRiskLimitPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesReduceRiskLimitNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures reduce risk limit notification push data error: %v", g.Name, err)
	}
}

const wsFuturesPositionsNotificationPushDataJSON = `{"time": 1588212926,"channel": "futures.positions",	"event": "update",	"error": null,	"result": [	  {		"contract": "BTC_USD",		"cross_leverage_limit": 0,		"entry_price": 40000.36666661111,		"history_pnl": -0.000108569505,		"history_point": 0,		"last_close_pnl": -0.000050123368,"leverage": 0,"leverage_max": 100,"liq_price": 0.1,"maintenance_rate": 0.005,"margin": 49.999890611186,"mode": "single","realised_pnl": -1.25e-8,"realised_point": 0,"risk_limit": 100,"size": 3,"time": 1628736848,"time_ms": 1628736848321,"user": "110xxxxx"}]}`

func TestFuturesPositionsNotification(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesPositionsNotificationPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures positions change notification push data error: %v", g.Name, err)
	}
}

const wsFuturesAutoOrdersPushDataJSON = `{"time": 1596798126,"channel": "futures.autoorders",	"event": "update",	"error": null,	"result": [	  {		"user": 123456,		"trigger": {		  "strategy_type": 0,		  "price_type": 0,		  "price": "10000",		  "rule": 2,		  "expiration": 86400		},		"initial": {		  "contract": "BTC_USDT",		  "size": 10,		  "price": "10000",		  "tif": "gtc",		  "text": "web",		  "iceberg": 0,		  "is_close": false,		  "is_reduce_only": false		},		"id": 9256,		"trade_id": 0,		"status": "open",		"reason": "",		"create_time": 1596798126,		"name": "price_autoorders",		"is_stop_order": false,		"stop_trigger": {		  "rule": 0,		  "trigger_price": "",		  "order_price": ""		}	  }	]}`

func TestFuturesAutoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleFuturesData([]byte(wsFuturesAutoOrdersPushDataJSON), asset.Futures); err != nil {
		t.Errorf("%s websocket futures auto orders push data error: %v", g.Name, err)
	}
}

// ******************************************** Options web-socket unit test funcs ********************

const optionsContractTickerPushDataJSON = `{"time": 1630576352,	"channel": "options.contract_tickers",	"event": "update",	"result": {    "name": "BTC_USDT-20211231-59800-P",    "last_price": "11349.5",    "mark_price": "11170.19",    "index_price": "",    "position_size": 993,    "bid1_price": "10611.7",    "bid1_size": 100,    "ask1_price": "11728.7",    "ask1_size": 100,    "vega": "34.8731",    "theta": "-72.80588",    "rho": "-28.53331",    "gamma": "0.00003",    "delta": "-0.78311",    "mark_iv": "0.86695",    "bid_iv": "0.65481",    "ask_iv": "0.88145",    "leverage": "3.5541112718136"	}}`

func TestOptionsContractTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsContractTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contract ticker push data failed with error %v", g.Name, err)
	}
}

const optionsUnderlyingTickerPushDataJSON = `{"time": 1630576352,	"channel": "options.ul_tickers",	"event": "update",	"result": {	   "trade_put": 800,	   "trade_call": 41700,	   "index_price": "50695.43",	   "name": "BTC_USDT"	}}`

func TestOptionsUnderlyingTickerPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsUnderlyingTickerPushDataJSON)); err != nil {
		t.Errorf("%s websocket options underlying ticker push data error: %v", g.Name, err)
	}
}

const optionsContractTradesPushDataJSON = `{"time": 1630576356,	"channel": "options.trades",	"event": "update",	"result": [    {        "contract": "BTC_USDT-20211231-59800-C",        "create_time": 1639144526,        "id": 12279,        "price": 997.8,        "size": -100,        "create_time_ms": 1639144526597,        "underlying": "BTC_USDT"    }	]}`

func TestOptionsContractTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsContractTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket contract trades push data error: %v", g.Name, err)
	}
}

const optionsUnderlyingTradesPushDataJSON = `{"time": 1630576356,	"channel": "options.ul_trades",	"event": "update",	"result": [{"contract": "BTC_USDT-20211231-59800-C","create_time": 1639144526,"id": 12279,"price": 997.8,"size": -100,"create_time_ms": 1639144526597,"underlying": "BTC_USDT","is_call": true}	]}`

func TestOptionsUnderlyingTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsUnderlyingTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket underlying trades push data error: %v", g.Name, err)
	}
}

const optionsUnderlyingPricePushDataJSON = `{	"time": 1630576356,	"channel": "options.ul_price",	"event": "update",	"result": {	   "underlying": "BTC_USDT",	   "price": 49653.24,"time": 1639143988,"time_ms": 1639143988931	}}`

func TestOptionsUnderlyingPricePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsUnderlyingPricePushDataJSON)); err != nil {
		t.Errorf("%s websocket underlying price push data error: %v", g.Name, err)
	}
}

const optionsMarkPricePushDataJSON = `{	"time": 1630576356,	"channel": "options.mark_price",	"event": "update",	"result": {    "contract": "BTC_USDT-20211231-59800-P",    "price": 11021.27,    "time": 1639143401,    "time_ms": 1639143401676	}}`

func TestOptionsMarkPricePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsMarkPricePushDataJSON)); err != nil {
		t.Errorf("%s websocket mark price push data error: %v", g.Name, err)
	}
}

const optionsSettlementsPushDataJSON = `{	"time": 1630576356,	"channel": "options.settlements",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-55000-P",	   "orderbook_id": 2,	   "position_size": 1,	   "profit": 0.5,	   "settle_price": 70000,	   "strike_price": 65000,	   "tag": "WEEK",	   "trade_id": 1,	   "trade_size": 1,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000	}}`

func TestSettlementsPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsSettlementsPushDataJSON)); err != nil {
		t.Errorf("%s websocket options settlements push data error: %v", g.Name, err)
	}
}

const optionsContractPushDataJSON = `{"time": 1630576356,	"channel": "options.contracts",	"event": "update",	"result": {	   "contract": "BTC_USDT-20211130-50000-P",	   "create_time": 1637917026,	   "expiration_time": 1638230400,	   "init_margin_high": 0.15,	   "init_margin_low": 0.1,	   "is_call": false,	   "maint_margin_base": 0.075,	   "maker_fee_rate": 0.0004,	   "mark_price_round": 0.1,	   "min_balance_short": 0.5,	   "min_order_margin": 0.1,	   "multiplier": 0.0001,	   "order_price_deviate": 0,	   "order_price_round": 0.1,	   "order_size_max": 1,	   "order_size_min": 10,	   "orders_limit": 100000,	   "ref_discount_rate": 0.1,	   "ref_rebate_rate": 0,	   "strike_price": 50000,	   "tag": "WEEK",	   "taker_fee_rate": 0.0004,	   "underlying": "BTC_USDT",	   "time": 1639051907,	   "time_ms": 1639051907000	}}`

func TestOptionsContractPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsContractPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contracts push data error: %v", g.Name, err)
	}
}

const (
	optionsContractCandlesticksPushDataJSON   = `{	"time": 1630650451,	"channel": "options.contract_candlesticks",	"event": "update",	"result": [   {       "t": 1639039260,       "v": 100,       "c": "1041.4",       "h": "1041.4",       "l": "1041.4",       "o": "1041.4",       "a": "0",       "n": "10s_BTC_USDT-20211231-59800-C"   }	]}`
	optionsUnderlyingCandlesticksPushDataJSON = `{	"time": 1630650451,	"channel": "options.ul_candlesticks",	"event": "update",	"result": [    {        "t": 1639039260,        "v": 100,        "c": "1041.4",        "h": "1041.4",        "l": "1041.4",        "o": "1041.4",        "a": "0",        "n": "10s_BTC_USDT"    }	]}`
)

func TestOptionsCandlesticksPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsContractCandlesticksPushDataJSON)); err != nil {
		t.Errorf("%s websocket options contracts candlestick push data error: %v", g.Name, err)
	}
	if err := g.wsHandleOptionsData([]byte(optionsUnderlyingCandlesticksPushDataJSON)); err != nil {
		t.Errorf("%s websocket options underlying candlestick push data error: %v", g.Name, err)
	}
}

const (
	optionsOrderbookTickerPushDataJSON              = `{	"time": 1630650452,	"channel": "options.book_ticker",	"event": "update",	"result": {    "t": 1615366379123,    "u": 2517661076,    "s": "BTC_USDT-20211130-50000-C",    "b": "54696.6",    "B": 37000,    "a": "54696.7",    "A": 47061	}}`
	optionsOrderbookUpdatePushDataJSON              = `{	"time": 1630650445,	"channel": "options.order_book_update",	"event": "update",	"result": {    "t": 1615366381417,    "s": "BTC_USDT-20211130-50000-C",    "U": 2517661101,    "u": 2517661113,    "b": [        {            "p": "54672.1",            "s": 95        },        {            "p": "54664.5",            "s": 58794        }    ],    "a": [        {            "p": "54743.6",            "s": 95        },        {            "p": "54742",            "s": 95        }    ]	}}`
	optionsOrderbookSnapshotPushDataJSON            = `{	"time": 1630650445,	"channel": "options.order_book",	"event": "all",	"result": {    "t": 1541500161123,    "contract": "BTC_USDT-20211130-50000-C",    "id": 93973511,    "asks": [        {            "p": "97.1",            "s": 2245        },		{            "p": "97.2",            "s": 2245        }    ],    "bids": [		{            "p": "97.2",            "s": 2245        },        {            "p": "97.1",            "s": 2245        }    ]	}}`
	optionsOrderbookSnapshotUpdateEventPushDataJSON = `{"channel": "options.order_book",	"event": "update",	"time": 1630650445,	"result": [	  {		"p": "49525.6",		"s": 7726,		"c": "BTC_USDT-20211130-50000-C",		"id": 93973511	  }	]}`
)

func TestOptionsOrderbookPushData(t *testing.T) {
	t.Parallel()
	err := g.wsHandleOptionsData([]byte(optionsOrderbookTickerPushDataJSON))
	if err != nil {
		t.Errorf("%s websocket options orderbook ticker push data error: %v", g.Name, err)
	}
	if err = g.wsHandleOptionsData([]byte(optionsOrderbookSnapshotPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orderbook snapshot push data error: %v", g.Name, err)
	}
	if err = g.wsHandleOptionsData([]byte(optionsOrderbookUpdatePushDataJSON)); err != nil {
		t.Errorf("%s websocket options orderbook update push data error: %v", g.Name, err)
	}
	if err = g.wsHandleOptionsData([]byte(optionsOrderbookSnapshotUpdateEventPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orderbook snapshot update event push data error: %v", g.Name, err)
	}
}

const optionsOrderPushDataJSON = `{"time": 1630654851,"channel": "options.orders",	"event": "update",	"result": [	   {		  "contract": "BTC_USDT-20211130-65000-C",		  "create_time": 1637897000,		  "fill_price": 0,		  "finish_as": "cancelled",		  "iceberg": 0,		  "id": 106,		  "is_close": false,		  "is_liq": false,		  "is_reduce_only": false,		  "left": -10,		  "mkfr": 0.0004,		  "price": 15000,		  "refr": 0,		  "refu": 0,		  "size": -10,		  "status": "finished",		  "text": "web",		  "tif": "gtc",		  "tkfr": 0.0004,		  "underlying": "BTC_USDT",		  "user": "9xxx",		  "time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsOrderPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsOrderPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orders push data error: %v", g.Name, err)
	}
}

const optionsUsersTradesPushDataJSON = `{	"time": 1639144214,	"channel": "options.usertrades",	"event": "update",	"result": [{"id": "1","underlying": "BTC_USDT","order": "557940","contract": "BTC_USDT-20211216-44800-C","create_time": 1639144214,"create_time_ms": 1639144214583,"price": "4999","role": "taker","size": -1}]}`

func TestOptionUserTradesPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsUsersTradesPushDataJSON)); err != nil {
		t.Errorf("%s websocket options orders push data error: %v", g.Name, err)
	}
}

const optionsLiquidatesPushDataJSON = `{	"channel": "options.liquidates",	"event": "update",	"time": 1630654851,	"result": [	   {		  "user": "1xxxx",		  "init_margin": 1190,		  "maint_margin": 1042.5,		  "order_margin": 0,		  "time": 1639051907,		  "time_ms": 1639051907000	   }	]}`

func TestOptionsLiquidatesPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsLiquidatesPushDataJSON)); err != nil {
		t.Errorf("%s websocket options liquidates push data error: %v", g.Name, err)
	}
}

const optionsSettlementPushDataJSON = `{	"channel": "options.user_settlements",	"event": "update",	"time": 1639051907,	"result": [{"contract": "BTC_USDT-20211130-65000-C","realised_pnl": -13.028,"settle_price": 70000,"settle_profit": 5,"size": 10,"strike_price": 65000,"underlying": "BTC_USDT","user": "9xxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsSettlementPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsSettlementPushDataJSON)); err != nil {
		t.Errorf("%s websocket options settlement push data error: %v", g.Name, err)
	}
}

const optionsPositionClosePushDataJSON = `{"channel": "options.position_closes",	"event": "update",	"time": 1630654851,	"result": [{"contract": "BTC_USDT-20211130-50000-C","pnl": -0.0056,"settle_size": 0,"side": "long","text": "web","underlying": "BTC_USDT","user": "11xxxxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsPositionClosePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsPositionClosePushDataJSON)); err != nil {
		t.Errorf("%s websocket options position close push data error: %v", g.Name, err)
	}
}

const optionsBalancePushDataJSON = `{	"channel": "options.balances",	"event": "update",	"time": 1630654851,	"result": [	   {		  "balance": 60.79009,"change": -0.5,"text": "BTC_USDT-20211130-55000-P","type": "set","user": "11xxxx","time": 1639051907,"time_ms": 1639051907000}]}`

func TestOptionsBalancePushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsBalancePushDataJSON)); err != nil {
		t.Errorf("%s websocket options balance push data error: %v", g.Name, err)
	}
}

const optionsPositionPushDataJSON = `{"time": 1630654851,	"channel": "options.positions",	"event": "update",	"error": null,	"result": [	   {		  "entry_price": 0,		  "realised_pnl": -13.028,		  "size": 0,		  "contract": "BTC_USDT-20211130-65000-C",		  "user": "9010",		  "time": 1639051907,		  "time_ms": 1639051907000	   }	]}`

func TestOptionsPositionPushData(t *testing.T) {
	t.Parallel()
	if err := g.wsHandleOptionsData([]byte(optionsPositionPushDataJSON)); err != nil {
		t.Errorf("%s websocket options position push data error: %v", g.Name, err)
	}
}

const (
	futuresOrderbookPushData       = `{"time": 1678468497, "time_ms": 1678468497232, "channel": "futures.order_book", "event": "all", "result": { "t": 1678468497168, "id": 4010394406, "contract": "BTC_USD", "asks": [ { "p": "19909", "s": 3100 }, { "p": "19909.1", "s": 5000 }, { "p": "19910", "s": 3100 }, { "p": "19914.4", "s": 4400 }, { "p": "19916.6", "s": 5000 }, { "p": "19917.2", "s": 8255 }, { "p": "19919.2", "s": 5000 }, { "p": "19920.3", "s": 11967 }, { "p": "19922.2", "s": 5000 }, { "p": "19924.2", "s": 5000 }, { "p": "19927.1", "s": 17129 }, { "p": "19927.2", "s": 5000 }, { "p": "19929", "s": 20864 }, { "p": "19929.3", "s": 5000 }, { "p": "19929.7", "s": 24683 }, { "p": "19930.3", "s": 750 }, { "p": "19931.4", "s": 5000 }, { "p": "19931.5", "s": 1 }, { "p": "19934.2", "s": 5000 }, { "p": "19935.4", "s": 1 } ], "bids": [ { "p": "19901.2", "s": 5000 }, { "p": "19900.3", "s": 3100 }, { "p": "19900.2", "s": 5000 }, { "p": "19899.3", "s": 2983 }, { "p": "19899.2", "s": 6035 }, { "p": "19897.2", "s": 5000 }, { "p": "19895.7", "s": 5984 }, { "p": "19895", "s": 5000 }, { "p": "19892.9", "s": 195 }, { "p": "19892.8", "s": 5000 }, { "p": "19889.4", "s": 5000 }, { "p": "19889", "s": 8800 }, { "p": "19888.5", "s": 11968 }, { "p": "19887.1", "s": 5000 }, { "p": "19886.4", "s": 24683 }, { "p": "19885.7", "s": 1 }, { "p": "19883.8", "s": 5000 }, { "p": "19880.2", "s": 5000 }, { "p": "19878.2", "s": 5000 }, { "p": "19876.8", "s": 1 } ] } }`
	futuresOrderbookUpdatePushData = `{"time": 1678469222, "time_ms": 1678469222982, "channel": "futures.order_book_update", "event": "update", "result": { "t": 1678469222617, "s": "BTC_USD", "U": 4010424331, "u": 4010424361, "b": [ { "p": "19860.7", "s": 5984 }, { "p": "19858.6", "s": 5000 }, { "p": "19845.4", "s": 20864 }, { "p": "19859.1", "s": 0 }, { "p": "19862.5", "s": 0 }, { "p": "19358", "s": 0 }, { "p": "19864.5", "s": 5000 }, { "p": "19840.7", "s": 0 }, { "p": "19863.6", "s": 3100 }, { "p": "19839.3", "s": 0 }, { "p": "19851.5", "s": 8800 }, { "p": "19720", "s": 0 }, { "p": "19333", "s": 0 }, { "p": "19852.7", "s": 5000 }, { "p": "19861.5", "s": 0 }, { "p": "19860.6", "s": 3100 }, { "p": "19833.6", "s": 0 }, { "p": "19360", "s": 0 }, { "p": "19863.5", "s": 5000 }, { "p": "19736.9", "s": 0 }, { "p": "19838.5", "s": 0 }, { "p": "19841.3", "s": 0 }, { "p": "19858.1", "s": 3100 }, { "p": "19710.9", "s": 0 }, { "p": "19342", "s": 0 }, { "p": "19852.1", "s": 11967 }, { "p": "19343", "s": 0 }, { "p": "19705", "s": 0 }, { "p": "19836.5", "s": 0 }, { "p": "19862.6", "s": 3100 }, { "p": "19729.6", "s": 0 }, { "p": "19849.9", "s": 5000 } ], "a": [ { "p": "19900.5", "s": 0 }, { "p": "19883.1", "s": 11967 }, { "p": "19910.9", "s": 0 }, { "p": "19897.7", "s": 5000 }, { "p": "19875.9", "s": 5984 }, { "p": "19899.6", "s": 0 }, { "p": "19878", "s": 4400 }, { "p": "19877.6", "s": 0 }, { "p": "19889.5", "s": 5000 }, { "p": "19875.5", "s": 3100 }, { "p": "19875.3", "s": 0 }, { "p": "19878.5", "s": 0 }, { "p": "19895.2", "s": 0 }, { "p": "20284.6", "s": 0 }, { "p": "19880.7", "s": 5000 }, { "p": "19875.4", "s": 0 }, { "p": "19985.8", "s": 0 }, { "p": "19887.1", "s": 5000 }, { "p": "19896", "s": 1 }, { "p": "19869.3", "s": 0 }, { "p": "19900", "s": 0 }, { "p": "19875.6", "s": 5000 }, { "p": "19980.6", "s": 0 }, { "p": "19885.1", "s": 5000 }, { "p": "19877.7", "s": 5000 }, { "p": "20000", "s": 0 }, { "p": "19892.2", "s": 8255 }, { "p": "19886.8", "s": 0 }, { "p": "20257.4", "s": 0 }, { "p": "20280", "s": 0 }, { "p": "20002.5", "s": 0 }, { "p": "20263.1", "s": 0 }, { "p": "19900.2", "s": 0 } ] } }`
)

func TestFuturesOrderbookPushData(t *testing.T) {
	t.Parallel()
	err := g.wsHandleFuturesData([]byte(futuresOrderbookPushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
	err = g.wsHandleFuturesData([]byte(futuresOrderbookUpdatePushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

const futuresCandlesticksPushData = `{"time": 1678469467, "time_ms": 1678469467981, "channel": "futures.candlesticks", "event": "update", "result": [ { "t": 1678469460, "v": 0, "c": "19896", "h": "19896", "l": "19896", "o": "19896", "n": "1m_BTC_USD" } ] }`

func TestFuturesCandlestickPushData(t *testing.T) {
	t.Parallel()
	err := g.wsHandleFuturesData([]byte(futuresCandlesticksPushData), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := g.GenerateDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
}
func TestGenerateDeliveryFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := g.GenerateDeliveryFuturesDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
}
func TestGenerateFuturesDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := g.GenerateFuturesDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
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
	if _, err := g.CreateAPIKeysOfSubAccount(context.Background(), CreateAPIKeySubAccountParams{
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
	_, err := g.GetAllAPIKeyOfSubAccount(context.Background(), 1234)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g, canManipulateRealOrders)
	if err := g.UpdateAPIKeyOfSubAccount(context.Background(), apiKey, CreateAPIKeySubAccountParams{
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
	_, err := g.GetAPIKeyOfSubAccount(context.Background(), 1234, "target_api_key")
	if err != nil {
		t.Error(err)
	}
}

func TestLockSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if err := g.LockSubAccount(context.Background(), 1234); err != nil {
		t.Error(err)
	}
}

func TestUnlockSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
	if err := g.UnlockSubAccount(context.Background(), 1234); err != nil {
		t.Error(err)
	}
}

func getFirstTradablePairOfAssets() {
	enabledPairs, err := g.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.Spot)
	}
	spotTradablePair = enabledPairs[0]
	enabledPairs, err = g.GetEnabledPairs(asset.Margin)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.Margin)
	}
	marginTradablePair = enabledPairs[0]
	enabledPairs, err = g.GetEnabledPairs(asset.CrossMargin)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.CrossMargin)
	}
	crossMarginTradablePair = enabledPairs[0]
	enabledPairs, err = g.GetEnabledPairs(asset.Futures)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.Futures)
	}
	futuresTradablePair = enabledPairs[len(enabledPairs)-1]
	enabledPairs, err = g.GetEnabledPairs(asset.Options)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.Options)
	}
	optionsTradablePair = enabledPairs[0]
	enabledPairs, err = g.GetEnabledPairs(asset.DeliveryFutures)
	if err != nil {
		log.Fatalf("GateIO %v, trying to get %v enabled pairs error", err, asset.DeliveryFutures)
	}
	deliveryFuturesTradablePair = enabledPairs[0]
}

func TestSettlement(t *testing.T) {
	availablePairs, err := g.GetAvailablePairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	for x := range availablePairs {
		t.Run(strconv.Itoa(x), func(t *testing.T) {
			_, err = g.getSettlementFromCurrency(availablePairs[x], true)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	availablePairs, err = g.GetAvailablePairs(asset.DeliveryFutures)
	if err != nil {
		t.Fatal(err)
	}
	for x := range availablePairs {
		t.Run(strconv.Itoa(x), func(t *testing.T) {
			_, err = g.getSettlementFromCurrency(availablePairs[x], false)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	availablePairs, err = g.GetAvailablePairs(asset.Options)
	if err != nil {
		t.Fatal(err)
	}
	for x := range availablePairs {
		t.Run(strconv.Itoa(x), func(t *testing.T) {
			_, err := g.getSettlementFromCurrency(availablePairs[x], false)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestParseGateioMilliSecTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731098
	timeWhenTestingString := `"1684981731098"` // Normal string
	integerJSON := `{"number": 1684981731098}`
	float64JSON := `{"number": 1684981731.098}`

	time := time.UnixMilli(timeWhenTesting)
	var in gateioTime
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	if err != nil {
		t.Fatal(err)
	}
	if !in.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", in.Time(), time)
	}
	inInteger := struct {
		Number gateioTime `json:"number"`
	}{}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	if err != nil {
		t.Fatal(err)
	}
	if !inInteger.Number.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", inInteger.Number.Time(), time)
	}

	inFloat64 := struct {
		Number gateioTime `json:"number"`
	}{}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	if err != nil {
		t.Fatal(err)
	}
	if !inFloat64.Number.Time().Equal(time) {
		t.Fatalf("found %v, but expected %v", inFloat64.Number.Time(), time)
	}
}

func TestParseGateioTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731
	timeWhenTestingString := `"1684981731"`
	integerJSON := `{"number": 1684981731}`
	float64JSON := `{"number": 1684981731.234}`
	timeWhenTestingStringMicroSecond := `"1691122380942.173000"`

	whenTime := time.Unix(timeWhenTesting, 0)
	var in gateioTime
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	if err != nil {
		t.Fatal(err)
	}
	if !in.Time().Equal(whenTime) {
		t.Fatalf("found %v, but expected %v", in.Time(), whenTime)
	}
	inInteger := struct {
		Number gateioTime `json:"number"`
	}{}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	if err != nil {
		t.Fatal(err)
	}
	if !inInteger.Number.Time().Equal(whenTime) {
		t.Fatalf("found %v, but expected %v", inInteger.Number.Time(), whenTime)
	}

	inFloat64 := struct {
		Number gateioTime `json:"number"`
	}{}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	if err != nil {
		t.Fatal(err)
	}
	msTime := time.UnixMilli(1684981731234)
	if !inFloat64.Number.Time().Equal(time.UnixMilli(1684981731234)) {
		t.Fatalf("found %v, but expected %v", inFloat64.Number.Time(), msTime)
	}

	var microSeconds gateioTime
	err = json.Unmarshal([]byte(timeWhenTestingStringMicroSecond), &microSeconds)
	if err != nil {
		t.Fatal(err)
	}
	if !microSeconds.Time().Equal(time.UnixMicro(1691122380942173)) {
		t.Fatalf("found %v, but expected %v", microSeconds.Time(), time.UnixMicro(1691122380942173))
	}
}

func TestGateioNumericalValue(t *testing.T) {
	t.Parallel()
	in := &struct {
		Number gateioNumericalValue `json:"number"`
	}{}

	numberJSON := `{"number":123442.231}`
	err := json.Unmarshal([]byte(numberJSON), in)
	if err != nil {
		t.Fatal(err)
	} else if in.Number != 123442.231 {
		t.Fatalf("found %f, but expected %f", in.Number, 123442.231)
	}

	numberJSON = `{"number":"123442.231"}`
	err = json.Unmarshal([]byte(numberJSON), in)
	if err != nil {
		t.Fatal(err)
	} else if in.Number != 123442.231 {
		t.Fatalf("found %f, but expected %s", in.Number, "123442.231")
	}

	numberJSON = `{"number":""}`
	err = json.Unmarshal([]byte(numberJSON), in)
	if err != nil {
		t.Fatal(err)
	} else if in.Number != 0 {
		t.Fatalf("found %f, but expected %d", in.Number, 0)
	}

	numberJSON = `{"number":0}`
	err = json.Unmarshal([]byte(numberJSON), in)
	if err != nil {
		t.Fatal(err)
	} else if in.Number != 0 {
		t.Fatalf("found %f, but expected %d", in.Number, 0)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	err := g.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}

	err = g.UpdateOrderExecutionLimits(context.Background(), 1336)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received %v, expected %v", err, asset.ErrNotSupported)
	}

	err = g.UpdateOrderExecutionLimits(context.Background(), asset.Options)
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received %v, expected %v", err, common.ErrNotYetImplemented)
	}

	err = g.UpdateOrderExecutionLimits(context.Background(), asset.Spot)
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
