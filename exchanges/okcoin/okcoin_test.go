package okcoin

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var (
	o = &Okcoin{}

	spotTradablePair currency.Pair
)

func TestMain(m *testing.M) {
	o.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Okcoin load config error", err)
	}
	okcoinConfig, err := cfg.GetExchangeConfig(o.Name)
	if err != nil {
		log.Fatalf("%v Setup() init error", o.Name)
	}

	okcoinConfig.API.AuthenticatedSupport = true
	okcoinConfig.API.AuthenticatedWebsocketSupport = true
	okcoinConfig.API.Credentials.Key = apiKey
	okcoinConfig.API.Credentials.Secret = apiSecret
	okcoinConfig.API.Credentials.ClientID = passphrase
	o.Websocket = sharedtestvalues.NewTestWebsocket()
	err = o.Setup(okcoinConfig)
	if err != nil {
		log.Fatal("Okcoin setup error", err)
	}
	err = o.populateTradablePairs(context.Background())
	if err != nil {
		log.Fatalf("%s populateTradablePairs error %v", o.Name, err)
	}
	setupWS()
	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := o.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = o.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestFetchTradablePair(t *testing.T) {
	t.Parallel()
	_, err := o.GetInstruments(context.Background(), "", "")
	if !errors.Is(err, errInstrumentTypeMissing) {
		t.Errorf("expected: %v, got: %v", errInstrumentTypeMissing, err)
	}
	_, err = o.GetInstruments(context.Background(), "SPOT", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	// allowed state value: ongoing, scheduled, processing, pre_open, completed, canceled
	_, err := o.GetSystemStatus(context.Background(), "scheduled")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	_, err := o.GetSystemTime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := o.GetTickers(context.Background(), "")
	if !errors.Is(err, errInstrumentTypeMissing) {
		t.Errorf("expected: %v, got: %v", errInstrumentTypeMissing, err)
	}
	_, err = o.GetTickers(context.Background(), "SPOT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := o.GetTicker(context.Background(), "USDT-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbooks(t *testing.T) {
	t.Parallel()
	_, err := o.GetOrderbook(context.Background(), spotTradablePair.String(), 200)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlesticks(context.Background(), spotTradablePair.String(), kline.FiveMin, time.Now(), time.Now().Add(-time.Hour*30), 0, false)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestickHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlestickHistory(context.Background(), spotTradablePair.String(), time.Now().Add(-time.Minute*30), time.Now(), kline.FiveMin, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := o.GetTrades(context.Background(), "BTC-USD", 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetTradeHistory(context.Background(), spotTradablePair.String(), "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGet24HourTradingVolume(t *testing.T) {
	t.Parallel()
	_, err := o.Get24HourTradingVolume(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, err := o.GetOracle(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := o.GetExchangeRate(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := o.GenerateDefaultSubscriptions()
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetCurrencies(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetBalance(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAccountAssetValuation(context.Background(), currency.EMPTYCODE)
	if err != nil {
		t.Error(err)
	}
}

func TestFundsTransfer(t *testing.T) {
	t.Parallel()
	_, err := o.FundsTransfer(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.FundsTransfer(context.Background(), &FundingTransferRequest{
		Currency: currency.EMPTYCODE,
	})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("found %v, but expected %v", err, currency.ErrCurrencyCodeEmpty)
	}
	_, err = o.FundsTransfer(context.Background(), &FundingTransferRequest{
		Currency: currency.BTC,
	})
	if !errors.Is(err, errInvalidAmount) { // "From" address
		t.Errorf("found %v, but expected %v", err, errInvalidAmount)
	}
	_, err = o.FundsTransfer(context.Background(), &FundingTransferRequest{
		Currency: currency.BTC,
		Amount:   1,
		From:     "abcde",
	})
	if !errors.Is(err, errAddressMustNotBeEmptyString) { // 'To' address
		t.Errorf("found %v, but expected %v", err, errAddressMustNotBeEmptyString)
	}
	_, err = o.FundsTransfer(context.Background(), &FundingTransferRequest{
		Currency:     currency.BTC,
		Amount:       1,
		From:         "abcdef",
		To:           "ghijklmnopqrstu",
		TransferType: 2,
	})
	if !errors.Is(err, errSubAccountNameRequired) {
		t.Errorf("found %v, but expected %v", err, errSubAccountNameRequired)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err = o.FundsTransfer(context.Background(), &FundingTransferRequest{
		Currency: currency.BTC,
		Amount:   1,
		From:     "1",
		To:       "6",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetFundsTransferState(context.Background(), "", "", "")
	if !errors.Is(err, errTransferIDOrClientIDRequired) {
		t.Error(err)
	}
	_, err = o.GetFundsTransferState(context.Background(), "1", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetBillType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAssetBillsDetail(context.Background(), currency.BTC, "2", "", time.Now().Add(-time.Minute), time.Now().Add(-time.Hour), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetLightningDeposits(context.Background(), currency.BTC, 0.001, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyDepositAddresses(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetCurrencyDepositAddresses(context.Background(), currency.EMPTYCODE)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("found %v, expected %v", err, currency.ErrCurrencyCodeEmpty)
	}
	_, err = o.GetCurrencyDepositAddresses(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetDepositHistory(context.Background(), currency.BTC, "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := o.Withdrawal(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Fatalf("found %v, expected %v", err, errNilArgument)
	}
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("found %v, expected %v", err, currency.ErrCurrencyCodeEmpty)
	}
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{
		Ccy: currency.BTC,
	})
	if !errors.Is(err, errInvalidAmount) {
		t.Fatalf("found %v, expected %v", err, errInvalidAmount)
	}
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{
		Ccy:    currency.BTC,
		Amount: 1,
	})
	if !errors.Is(err, errInvalidWithdrawalMethod) {
		t.Fatalf("found %v, expected %v", err, errInvalidWithdrawalMethod)
	}
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{Amount: 1, Ccy: currency.BTC, WithdrawalMethod: "1"})
	if !errors.Is(err, errAddressMustNotBeEmptyString) {
		t.Fatalf("found %v, expected %v", err, errInvalidWithdrawalMethod)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{Amount: 1, Ccy: currency.BTC, WithdrawalMethod: "1", ToAddress: "abcdefg"})
	if !errors.Is(err, errInvalidTransactionFeeValue) {
		t.Errorf("found %v, expected %v", err, errAddressMustNotBeEmptyString)
	}
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{Amount: 1, Ccy: currency.BTC, WithdrawalMethod: "1", ToAddress: "abcdefg", TransactionFee: 0.004})
	if err != nil {
		t.Error(err)
	}
}

func TestLightningWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.SubmitLightningWithdrawals(context.Background(), &LightningWithdrawalsRequest{
		Ccy:     currency.BTC,
		Invoice: "something",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelWithdrawal(context.Background(), &WithdrawalCancellation{
		WithdrawalID: "1123456",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetWithdrawalHistory(context.Background(), currency.BTC, "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAccountBalance(context.Background(), currency.BTC, currency.USDT)
	if err != nil {
		t.Error(err)
	}
}

func TestGetBillsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetBillsDetails(context.Background(), currency.BTC, "", "", "", "", "", time.Now().Add(-time.Hour*30), time.Now(), 0)
	if err != nil {
		t.Error(err)
	}
}
func TestGetBillsDetailsFor3Months(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetBillsDetailsFor3Months(context.Background(), currency.BTC, "", "", "", "", "", time.Now().Add(-time.Hour*30), time.Now(), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountConfigurations(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAccountConfigurations(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMaximumBuySellOrOpenAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetMaximumBuySellOrOpenAmount(context.Background(), "BTC-USD", "cash", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetMaximumAvailableTradableAmount(context.Background(), "cash", "BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFeeRates(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetFeeRates(context.Background(), "SPOT", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetMaximumWithdrawals(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAvailableRFQPairs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAvailableRFQPairs(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestRequestQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.RequestQuote(context.Background(), &QuoteRequestArg{
		BaseCurrency:  currency.BTC,
		QuoteCurrency: currency.USD,
		Side:          "sell",
		RfqSize:       1000,
		RfqSzCurrency: currency.USD,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceRFQOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.PlaceRFQOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, but found %v", errNilArgument, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{})
	if !errors.Is(err, errClientRequestIDRequired) {
		t.Errorf("expected %v, but found %v", errClientRequestIDRequired, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
	})
	if !errors.Is(err, errQuoteIDRequired) {
		t.Errorf("expected %v, but found %v", errQuoteIDRequired, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
		QuoteID:                     "1234"})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, but found %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
		QuoteID:                     "1234",
		BaseCurrency:                currency.BTC,
	})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, but found %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
		QuoteID:                     "1234",
		BaseCurrency:                currency.BTC,
		QuoteCurrency:               currency.USD})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("expected %v, but found %v", order.ErrSideIsInvalid, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
		QuoteID:                     "1234",
		BaseCurrency:                currency.BTC,
		QuoteCurrency:               currency.USD,
		Side:                        "buy",
	})
	if !errors.Is(err, errInvalidAmount) {
		t.Errorf("expected %v, but found %v", errInvalidAmount, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "1234",
		QuoteID:                     "1234",
		BaseCurrency:                currency.BTC,
		QuoteCurrency:               currency.USD,
		Side:                        "buy",
		Size:                        22,
	})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("expected %v, but found %v", currency.ErrCurrencyCodeEmpty, err)
	}
	_, err = o.PlaceRFQOrder(context.Background(), &PlaceRFQOrderRequest{
		ClientDefinedTradeRequestID: "5111",
		QuoteID:                     "12638308",
		BaseCurrency:                currency.BTC,
		QuoteCurrency:               currency.USD,
		Side:                        "buy",
		Size:                        22,
		SizeCurrency:                currency.BTC,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetRFQOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetRFQOrderDetails(context.Background(), "", "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetRFQOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetRFQOrderHistory(context.Background(), time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestDeposit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.Deposit(context.Background(), &FiatDepositRequestArg{
		ChannelID:         "28",
		BankAccountNumber: "1000221891299",
		Amount:            100,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelFiatDeposit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelFiatDeposit(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFiatDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetFiatDepositHistory(context.Background(), currency.BTC, "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestFiatWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.FiatWithdrawal(context.Background(), &FiatWithdrawalParam{
		ChannelID:      "3",
		BankAcctNumber: "100221891299",
		Amount:         12,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestFiatCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.FiatCancelWithdrawal(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFiatWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetFiatWithdrawalHistory(context.Background(), currency.BTC, "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetChannelInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetChannelInfo(context.Background(), "27")
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.PlaceOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("found %v, but expected %v", err, currency.ErrCurrencyPairEmpty)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{InstrumentID: spotTradablePair})
	if !errors.Is(err, errTradeModeIsRequired) {
		t.Errorf("found %v, but expected %v", err, errTradeModeIsRequired)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrSideIsInvalid)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
		Side:      "buy",
	})
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrTypeIsInvalid)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
		Side:      "buy",
		OrderType: "limit",
	})
	if !errors.Is(err, errInvalidAmount) {
		t.Errorf("found %v, but expected %v", err, errInvalidAmount)
	}
	_, err = o.PlaceOrder(context.Background(), &PlaceTradeOrderParam{
		InstrumentID:  spotTradablePair,
		TradeMode:     "cash",
		ClientOrderID: "12345",
		Side:          "buy",
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.PlaceMultipleOrder(context.Background(), []PlaceTradeOrderParam{
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "1",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          2,
		},
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "12",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          1.5,
		},
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "123",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          1,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelTradeOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.CancelTradeOrder(context.Background(), &CancelTradeOrderRequest{})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, but expected %v", err, errMissingInstrumentID)
	}
	_, err = o.CancelTradeOrder(context.Background(), &CancelTradeOrderRequest{
		InstrumentID: "BTC-USD",
	})
	if !errors.Is(err, errOrderIDOrClientOrderIDRequired) {
		t.Errorf("found %v, but expected %v", err, errOrderIDOrClientOrderIDRequired)
	}
	_, err = o.CancelTradeOrder(context.Background(), &CancelTradeOrderRequest{
		InstrumentID:  "BTC-USD",
		ClientOrderID: "123",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelMultipleOrders(context.Background(), []CancelTradeOrderRequest{
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "123",
		},
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "abcdefg",
		},
		{
			InstrumentID:  "ETH-USD",
			ClientOrderID: "1234",
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.AmendOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.AmendOrder(context.Background(), &AmendTradeOrderRequestParam{})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, expected %v", err, errMissingInstrumentID)
	}
	_, err = o.AmendOrder(context.Background(), &AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD"})
	if !errors.Is(err, errOrderIDOrClientOrderIDRequired) {
		t.Errorf("found %v, expected %v", err, errOrderIDOrClientOrderIDRequired)
	}
	_, err = o.AmendOrder(context.Background(), &AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD",
		OrderID:      "1234"})
	if !errors.Is(err, errSizeOrPriceRequired) {
		t.Errorf("found %v, expected %v", err, errSizeOrPriceRequired)
	}
	_, err = o.AmendOrder(context.Background(), &AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD",
		OrderID:      "1234",
		NewSize:      5,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAmendMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.AmendMultipleOrder(context.Background(), []AmendTradeOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
			OrderID:      "1234",
			NewSize:      5,
		},
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "abe",
			NewPrice:      100,
		},
		{
			InstrumentID:    "BTC-USD",
			OrderID:         "3452",
			ClientRequestID: "9879",
			NewSize:         2,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetPersonalOrderDetail(context.Background(), "BTC-USD", "1243", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPersonalOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetPersonalOrderList(context.Background(), "SPOT", "BTC-USD", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetOrderHistory7Days(context.Background(), "SPOT", "BTC-USD", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory3MonthsDays(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetOrderHistory3Months(context.Background(), "SPOT", "BTC-USD", "", "", time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTransactionDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetRecentTransactionDetail(context.Background(), "SPOT", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}
func TestGetTransactionDetails3Months(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetTransactionDetails3Months(context.Background(), "SPOT", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.PlaceAlgoOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Error(err)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, but expected %v", err, errMissingInstrumentID)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
	})
	if !errors.Is(err, errTradeModeIsRequired) {
		t.Errorf("found %v, but expected %v", err, errTradeModeIsRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrSideIsInvalid)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
	})
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrTypeIsInvalid)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		OrderType:    "oco",
	})
	if !errors.Is(err, errInvalidAmount) {
		t.Errorf("found %v, but expected %v", err, errInvalidAmount)
	}

	// Stop loss
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "conditional",
	})
	if !errors.Is(err, errStopLossOrTakeProfitOrderPriceRequired) {
		t.Errorf("found %v, but expected %v", err, errStopLossOrTakeProfitOrderPriceRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "conditional",
	})
	if !errors.Is(err, errStopLossOrTakeProfitOrderPriceRequired) {
		t.Errorf("found %v, but expected %v", err, errStopLossOrTakeProfitOrderPriceRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "conditional",
		TpOrderPrice: 1,
	})
	if !errors.Is(err, errTakeProfitOrderPriceRequired) {
		t.Errorf("found %v, but expected %v", err, errTakeProfitOrderPriceRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:   "BTC-USD",
		TradeMode:      "cash",
		Side:           "buy",
		Size:           2,
		OrderType:      "conditional",
		TpOrderPrice:   1,
		TpTriggerPrice: 1,
	})
	if !errors.Is(err, errTpTriggerOrderPriceTypeRequired) {
		t.Errorf("found %v, but expected %v", err, errTpTriggerOrderPriceTypeRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:            "BTC-USD",
		TradeMode:               "cash",
		Side:                    "buy",
		Size:                    2,
		OrderType:               "conditional",
		TpOrderPrice:            1,
		TpTriggerOrderPriceType: "last",
		TpTriggerPrice:          1,
	})
	if err != nil {
		t.Error(err)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:       "BTC-USD",
		TradeMode:          "cash",
		Side:               "buy",
		Size:               2,
		OrderType:          "conditional",
		StopLossOrderPrice: 1,
	})
	if !errors.Is(err, errStopLossTriggerPriceRequired) {
		t.Errorf("found %v, but expected %v", err, errStopLossTriggerPriceRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:         "BTC-USD",
		TradeMode:            "cash",
		Side:                 "buy",
		Size:                 2,
		OrderType:            "conditional",
		StopLossOrderPrice:   1,
		StopLossTriggerPrice: 2,
	})
	if !errors.Is(err, errStopLossTriggerPriceTypeRequired) {
		t.Errorf("found %v, but expected %v", err, errStopLossTriggerPriceTypeRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:             "BTC-USD",
		TradeMode:                "cash",
		Side:                     "buy",
		Size:                     2,
		OrderType:                "conditional",
		StopLossOrderPrice:       5000,
		StopLossTriggerPrice:     2,
		StopLossTriggerPriceType: "last",
	})
	if err != nil {
		t.Error(err)
	}
	//  Trigger order
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "trigger",
		TriggerPrice: 123,
	})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("found %v, but expected %v", err, errInvalidPrice)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "trigger",
		TriggerPrice: 123,
	})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("found %v, but expected %v", err, errInvalidPrice)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "trigger",
		TriggerPrice: 123,
		OrderPrice:   234,
	})
	if err != nil {
		t.Error(err)
	}

	// OCO Orders
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "oco",
	})
	if !errors.Is(err, errTakeProfitOrderPriceRequired) {
		t.Errorf("found %v, but expected %v", err, errTakeProfitOrderPriceRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "oco",
		TpOrderPrice: 123,
	})
	if !errors.Is(err, errTpTriggerOrderPriceTypeRequired) {
		t.Errorf("found %v, but expected %v", err, errTpTriggerOrderPriceTypeRequired)
	}

	// Iceberg order
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "move_order_stop",
	})
	if !errors.Is(err, errCallbackRatioOrCallbackSpeedRequired) {
		t.Errorf("found %v, but expected %v", err, errCallbackRatioOrCallbackSpeedRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID:  "BTC-USD",
		TradeMode:     "cash",
		Side:          "buy",
		Size:          2,
		OrderType:     "move_order_stop",
		CallbackRatio: 0.002,
	})
	if err != nil {
		t.Error(err)
	}
	// Twap Order
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
	})
	if !errors.Is(err, errPriceRatioOrPriceSpreadRequired) {
		t.Errorf("found %v, but expected %v", err, errPriceRatioOrPriceSpreadRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
		PriceRatio:   0.02,
	})
	if !errors.Is(err, errSizeLimitRequired) {
		t.Errorf("found %v, but expected %v", err, errSizeLimitRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
		PriceRatio:   0.02,
		PriceLimit:   1234,
	})
	if !errors.Is(err, errSizeLimitRequired) {
		t.Errorf("found %v, but expected %v", err, errSizeLimitRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
		PriceRatio:   0.02,
		SizeLimit:    1234,
		TimeInterval: "1m",
	})
	if !errors.Is(err, errPriceLimitRequired) {
		t.Errorf("found %v, but expected %v", err, errPriceLimitRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
		PriceRatio:   0.02,
		PriceLimit:   1234,
		SizeLimit:    1234,
	})
	if !errors.Is(err, errTimeIntervalInformationRequired) {
		t.Errorf("found %v, but expected %v", err, errTimeIntervalInformationRequired)
	}
	_, err = o.PlaceAlgoOrder(context.Background(), &AlgoOrderRequestParam{
		InstrumentID: "BTC-USD",
		TradeMode:    "cash",
		Side:         "buy",
		Size:         2,
		OrderType:    "twap",
		PriceRatio:   0.02,
		PriceLimit:   1234,
		SizeLimit:    1234,
		TimeInterval: "1m",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{})
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.CancelAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
		},
	})
	if !errors.Is(err, errAlgoIDRequired) {
		t.Errorf("found %v, but expected %v", err, errAlgoIDRequired)
	}
	_, err = o.CancelAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			AlgoOrderID: "1234",
		},
	})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, but expected %v", err, errMissingInstrumentID)
	}
	_, err = o.CancelAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
			AlgoOrderID:  "2234",
		},
	})
	if err != nil {
		t.Error(err)
	}
}
func TestCancelAdvancedAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelAdvancedAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{})
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.CancelAdvancedAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
		},
	})
	if !errors.Is(err, errAlgoIDRequired) {
		t.Errorf("found %v, but expected %v", err, errAlgoIDRequired)
	}
	_, err = o.CancelAdvancedAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			AlgoOrderID: "1234",
		},
	})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, but expected %v", err, errMissingInstrumentID)
	}
	_, err = o.CancelAdvancedAlgoOrder(context.Background(), []CancelAlgoOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
			AlgoOrderID:  "2234",
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := o.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := o.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := o.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := o.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := o.FetchTicker(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := o.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}
func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := o.FetchOrderbook(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := o.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := o.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-5*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, time.Now().Add(-5*time.Hour), time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-kline.ThreeMonth.Duration() * 3).Truncate(kline.ThreeMonth.Duration())
	end := start.Add(kline.ThreeMonth.Duration())
	_, err := o.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.ThreeMonth, start, end)
	if err != nil {
		t.Errorf("%s GetHistoricCandlesExtended() error: %v", o.Name, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := o.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USD), asset.Spot, time.Now().Add(-time.Hour*4), time.Now().Add(-time.Minute*2)); err != nil {
		t.Errorf("%s GetHistoricTrades() error %v", o.Name, err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Pairs:     currency.Pairs{spotTradablePair},
	}
	_, err := o.GetActiveOrders(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := o.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAccountFundingHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	var orderSubmission = &order.Submit{
		Pair:      spotTradablePair,
		Exchange:  o.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		TradeMode: "cash",
		Amount:    1000000000,
		ClientID:  "yeneOrder",
		AssetType: asset.Spot,
	}
	_, err := o.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.ModifyOrder(context.Background(),
		&order.Modify{
			AssetType: asset.Spot,
			Pair:      spotTradablePair,
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          spotTradablePair,
		AssetType:     asset.Spot,
	}
	if err := o.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetOrderInfo(context.Background(), "123", spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	if _, err := o.GetDepositAddress(context.Background(), currency.BTC, "", ""); err != nil {
		t.Error(err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    o.Name,
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address:   "emailaddress@company.com",
			FeeAmount: 0.01,
		},
		ClientOrderID: "1234",
	}
	// fetching currency detail to extract the chain information.
	_, err := o.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	if err != nil {
		t.Error(err)
	}
	currencyInfo, err := o.GetCurrencies(context.Background(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
	withdrawCryptoRequest = withdraw.Request{
		Exchange:    o.Name,
		Amount:      1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Chain:     currencyInfo[0].Chain,
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 0.01,
		},
		ClientOrderID: "1234",
	}
	_, err = o.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	if _, err := o.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot); err != nil {
		t.Error(err)
	}
}
func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	if _, err := o.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                spotTradablePair,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}); err != nil {
		t.Errorf("%s GetFeeByType() error %v", o.Name, err)
	}
}

func setupWS() {
	if !o.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(o) {
		o.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	websocket, err := o.GetWebsocket()
	if err != nil {
		log.Fatal(err)
	}
	err = websocket.Connect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsPlaceOrder(nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("found %v, but expected %v", err, currency.ErrCurrencyPairEmpty)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{InstrumentID: spotTradablePair})
	if !errors.Is(err, errTradeModeIsRequired) {
		t.Errorf("found %v, but expected %v", err, errTradeModeIsRequired)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrSideIsInvalid)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
		Side:      "buy",
	})
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Errorf("found %v, but expected %v", err, order.ErrTypeIsInvalid)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{InstrumentID: spotTradablePair,
		TradeMode: "cash",
		Side:      "buy",
		OrderType: "limit",
	})
	if !errors.Is(err, errInvalidAmount) {
		t.Errorf("found %v, but expected %v", err, errInvalidAmount)
	}
	_, err = o.WsPlaceOrder(&PlaceTradeOrderParam{
		Side:         "buy",
		InstrumentID: spotTradablePair,
		TradeMode:    "cash",
		OrderType:    "market",
		Size:         100,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsPlaceMultipleOrder([]PlaceTradeOrderParam{
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "1",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          2,
		},
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "12",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          1.5,
		},
		{
			InstrumentID:  spotTradablePair,
			TradeMode:     "cash",
			ClientOrderID: "123",
			Side:          "buy",
			OrderType:     "limit",
			Price:         2.15,
			Size:          1,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelTradeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsCancelTradeOrder(nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.WsCancelTradeOrder(&CancelTradeOrderRequest{})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, but expected %v", err, errMissingInstrumentID)
	}
	_, err = o.WsCancelTradeOrder(&CancelTradeOrderRequest{
		InstrumentID: "BTC-USD",
	})
	if !errors.Is(err, errOrderIDOrClientOrderIDRequired) {
		t.Errorf("found %v, but expected %v", err, errOrderIDOrClientOrderIDRequired)
	}
	_, err = o.WsCancelTradeOrder(&CancelTradeOrderRequest{
		InstrumentID:  "BTC-USD",
		ClientOrderID: "123",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsCancelMultipleOrders([]CancelTradeOrderRequest{
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "123",
		},
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "abcdefg",
		},
		{
			InstrumentID:  "ETH-USD",
			ClientOrderID: "1234",
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsAmendOrder(nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("found %v, but expected %v", err, errNilArgument)
	}
	_, err = o.WsAmendOrder(&AmendTradeOrderRequestParam{})
	if !errors.Is(err, errMissingInstrumentID) {
		t.Errorf("found %v, expected %v", err, errMissingInstrumentID)
	}
	_, err = o.WsAmendOrder(&AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD"})
	if !errors.Is(err, errOrderIDOrClientOrderIDRequired) {
		t.Errorf("found %v, expected %v", err, errOrderIDOrClientOrderIDRequired)
	}
	_, err = o.WsAmendOrder(&AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD",
		OrderID:      "1234"})
	if !errors.Is(err, errSizeOrPriceRequired) {
		t.Errorf("found %v, expected %v", err, errSizeOrPriceRequired)
	}
	_, err = o.WsAmendOrder(&AmendTradeOrderRequestParam{
		InstrumentID: "BTC-USD",
		OrderID:      "1234",
		NewSize:      5,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestWsAmendMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.WsAmendMultipleOrder([]AmendTradeOrderRequestParam{
		{
			InstrumentID: "BTC-USD",
			OrderID:      "1234",
			NewSize:      5,
			NewPrice:     100,
		},
		{
			InstrumentID:  "BTC-USD",
			ClientOrderID: "abe",
			NewPrice:      100,
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := o.GetServerTime(context.Background(), asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func (o *Okcoin) populateTradablePairs(ctx context.Context) error {
	err := o.UpdateTradablePairs(ctx, true)
	if err != nil {
		return err
	}
	enabledPairs, err := o.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(enabledPairs) == 0 {
		return errors.New("No enabled pairs found")
	}
	spotTradablePair = enabledPairs[0]
	return nil
}

func TestOKCOINNumberUnmarshal(t *testing.T) {
	type testNumberHolder struct {
		Numb okcoinNumber `json:"numb"`
	}
	var val testNumberHolder
	data1 := `{ "numb":"12345.65" }`
	err := json.Unmarshal([]byte(data1), &val)
	if err != nil {
		t.Error(err)
	} else if val.Numb.Float64() != 12345.65 {
		t.Errorf("found %.2f, but found %.2f", val.Numb.Float64(), 12345.65)
	}
	data2 := `{ "numb":"" }`
	err = json.Unmarshal([]byte(data2), &val)
	if err != nil {
		t.Error(err)
	} else if val.Numb.Float64() != 0 {
		t.Errorf("found %.2f, but found %d", val.Numb.Float64(), 0)
	}
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetSubAccounts(context.Background(), true, "", time.Time{}, time.Now(), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAPIKeyOfSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAPIKeyOfSubAccount(context.Background(), "", "")
	if !errors.Is(err, errSubAccountNameRequired) {
		t.Errorf("expected %v, got %v", errSubAccountNameRequired, err)
	}
	_, err = o.GetAPIKeyOfSubAccount(context.Background(), "Sub-Account-Name-1", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountTradingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetSubAccountTradingBalance(context.Background(), "")
	if !errors.Is(err, errSubAccountNameRequired) {
		t.Errorf("expected %v, got %v", errSubAccountNameRequired, err)
	}
	_, err = o.GetSubAccountTradingBalance(context.Background(), "Sami")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountFundingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetSubAccountFundingBalance(context.Background(), "")
	if !errors.Is(err, errSubAccountNameRequired) {
		t.Errorf("expected %v, got %v", errSubAccountNameRequired, err)
	}
	_, err = o.GetSubAccountFundingBalance(context.Background(), "Sami", "BTC", "USD")
	if err != nil {
		t.Error(err)
	}
}

func TestSubAccountTransferHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.SubAccountTransferHistory(context.Background(), "Sami", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestAccountBalanceTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.AccountBalanceTransfer(context.Background(), &IntraAccountTransferParam{
		Ccy:            "BTC",
		Amount:         1234.0,
		From:           "6",
		To:             "6",
		FromSubAccount: "test-1",
		ToSubAccount:   "Sami",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetAlgoOrderhistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAlgoOrderHistory(context.Background(), "", "effective", "", "SPOT", "", "", "", 0)
	if !errors.Is(err, errOrderTypeRequired) {
		t.Errorf("expected %v, got %v", errOrderTypeRequired, err)
	}
	_, err = o.GetAlgoOrderHistory(context.Background(), "conditional", "effective", "", "SPOT", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.GetAlgoOrderList(context.Background(), "", "", "", "", "", "", "", 0)
	if !errors.Is(err, errOrderTypeRequired) {
		t.Errorf("expected %v, got %v", errOrderTypeRequired, err)
	}
	_, err = o.GetAlgoOrderList(context.Background(), "oco", "", "", "", "", "", "", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	var orderCancellationParams = []order.Cancel{
		{
			OrderID:   "1",
			Pair:      spotTradablePair,
			AssetType: asset.Spot,
		},
		{
			OrderID:   "2",
			Pair:      spotTradablePair,
			AssetType: asset.Spot,
		},
	}
	_, err := o.CancelBatchOrders(context.Background(), orderCancellationParams)
	if err != nil {
		t.Error(err)
	}
}

const (
	// private
	orderPushData                    = `{ "arg": { "channel": "orders", "instType": "SPOT", "instId": "ETH-USD", "uid": "1300041872592896" }, "data": [ { "accFillSz": "0", "amendResult": "", "avgPx": "0", "cTime": "1672898607313", "category": "normal", "ccy": "", "clOrdId": "", "code": "0", "execType": "", "fee": "0", "feeCcy": "ETH", "fillFee": "0", "fillFeeCcy": "", "fillNotionalUsd": "", "fillPx": "", "fillSz": "0", "fillTime": "", "instId": "ETH-USD", "instType": "SPOT", "lever": "0", "msg": "", "notionalUsd": "0.25", "ordId": "531110485559349248", "ordType": "limit", "pnl": "0", "posSide": "", "px": "250", "rebate": "0", "rebateCcy": "USD", "reduceOnly": "false", "reqId": "", "side": "buy", "slOrdPx": "", "slTriggerPx": "", "slTriggerPxType": "last", "source": "", "state": "live", "sz": "0.001", "tag": "", "tdMode": "cash", "tgtCcy": "", "tpOrdPx": "", "tpTriggerPx": "", "tpTriggerPxType": "last", "tradeId": "", "uTime": "1672898607313"}]}`
	accountChannelPushData           = `{"event": "subscribe", "arg": { "channel": "account" }}`
	algoOrdersChannelPushData        = `{	"event": "subscribe", "arg": { "channel": "orders-algo", "instType": "SPOT", "instId": "BTC-USD" } } `
	advancedAlgoOrderChannelPushData = `{ "event": "subscribe", "arg": { "channel": "algo-advance", "instType": "SPOT", "instId": "BTC-USD" } }`

	// public
	instrumentsChannelPushData     = `{"event": "subscribe","arg":{"channel":"instruments","instType":"SPOT"}}`
	instrumentsDataChannelPushData = `{ "arg": { "channel": "instruments", "instType": "SPOT" }, "data": [ { "alias": "", "baseCcy": "BTC", "category": "1", "ctMult": "", "ctType": "", "ctVal": "", "ctValCcy": "", "expTime": "", "instFamily": "", "instId": "BTC-USD", "instType": "SPOT", "lever": "", "listTime": "", "lotSz": "0.0001", "maxIcebergSz": "", "maxLmtSz": "99999999999999", "maxMktSz": "1000000", "maxStopSz": "1000000", "maxTriggerSz": "", "maxTwapSz": "", "minSz": "0.0001", "optType": "", "quoteCcy": "USD", "settleCcy": "", "state": "live", "stk": "", "tickSz": "0.01", "uly": "" } ] }`
	tickersChannelPushData         = `{"arg": { "channel": "tickers", "instId": "BTC-USD" }, "data": [ { "instType": "SPOT", "instId": "BTC-USD", "last": "16838.75", "lastSz": "0.0027", "askPx": "16838.75", "askSz": "0.0275", "bidPx": "16836.5", "bidSz": "0.0404", "open24h": "16762.13", "high24h": "16943.44", "low24h": "16629.04", "sodUtc0": "16688.74", "sodUtc8": "16700.35", "volCcy24h": "3016898.9552", "vol24h": "179.6477", "ts": "1672842446928"}]}`
	candlestickChannelPushData     = `{ "arg": { "channel": "candle1m", "instId": "BTC-USD" }, "data": [ [ "1672913160000", "16837.25", "16837.25", "16837.25", "16837.25", "0.0438", "737.4716", "737.4716", "1" ] ] }`
	tradesPushData                 = `{ "arg": { "channel": "trades", "instId": "BTC-USD" }, "data": [ { "instId": "BTC-USD", "tradeId": "130639474", "px": "42219.9", "sz": "0.12060306", "side": "buy", "ts": "1630048897897"}]}`
	statusChannelPushData          = `{"arg": { "channel": "status" }, "data": [{ "title": "Spot System Upgrade", "state": "scheduled", "begin": "1610019546", "href": "", "end": "1610019546", "serviceType": "1", "system": "classic", "scheDesc": "", "ts": "1597026383085"}]}`
)

func TestSubscriptionPushData(t *testing.T) {
	t.Parallel()
	err := o.WsHandleData([]byte(orderPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(accountChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(algoOrdersChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(advancedAlgoOrderChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(instrumentsChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(instrumentsDataChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(tickersChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(candlestickChannelPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(tradesPushData))
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData([]byte(statusChannelPushData))
	if err != nil {
		t.Error(err)
	}
}

var (
	orderbookSnapshot4000 = []byte(` {"arg":{"channel":"books","instId":"BTC-USD"},"action":"snapshot","data":[{"asks":[["30382.6","0.1504","0","1"],["30382.61","0.6866","0","1"],["30472.95","0.8929","0","1"],["30563.28","0.6827","0","1"],["30648.44","1.8055","0","1"],["30653.62","0.8868","0","1"],["30743.95","0.8222","0","1"],["31000","0.0117","0","1"],["31200","0.0032","0","1"],["31216.26","0.0014","0","1"],["31312.95","0.0054","0","1"],["31450","0.0002","0","1"],["31500","0.0043","0","1"],["31594.77","0.0227","0","1"],["31800","0.0142","0","2"],["31879","0.001","0","1"],["31879.13","0.0266","0","1"],["31980","0.0063","0","1"],["32100","0.0036","0","1"],["32150","0.0041","0","1"],["32166.05","0.0192","0","1"],["32200","0.0053","0","2"],["32434","0.0028","0","1"],["32450","0.0057","0","1"],["32455.55","0.0214","0","1"],["32500","0.0766","0","2"],["32747.65","0.0032","0","1"],["33000","0.1747","0","3"],["33042.38","0.0208","0","1"],["33339.77","0.0081","0","1"],["33639.83","0.0101","0","1"],["33664.27","0.0031","0","1"],["33700.39","0.0101","0","1"],["33942.59","0.0129","0","1"],["34000","5","0","1"],["34200","0.0057","0","1"],["34248.08","0.0145","0","1"],["34556.32","0.0005","0","1"],["34867.33","0.0274","0","1"],["35000","0.0006","0","1"],["35050","3.7276","0","1"],["35181.14","0.0061","0","1"],["35497.78","0.015","0","1"],["35817.27","0.023","0","1"],["36139.63","0.0011","0","1"],["36249.98","2.2015","0","1"],["36400","0.005","0","1"],["36464.89","0.0255","0","1"],["36793.08","0.023","0","1"],["37124.22","0.0054","0","1"],["37500","0.0151","0","1"],["38500","0.0151","0","1"],["39000","0.0404","0","1"],["40000","0.0151","0","1"],["44350","0.0006","0","1"],["45000","0.0055","0","2"],["45600","0.0011","0","1"],["46000","0.0017","0","1"],["46670","0.0003","0","1"],["47300","0.0011","0","1"],["47500","0.0008","0","1"],["48000","0.0012","0","1"],["48500","0.0003","0","1"],["48902.89","0.0002","0","1"],["49000","0.005","0","3"],["49300","0.0011","0","1"],["50000","0.0145","0","3"],["51700","0.0001","0","1"],["52000","0.0006","0","1"],["52902.89","0.0001","0","1"],["53700","0.0001","0","1"],["54655.72","0.0094","0","1"],["54902.89","0.0001","0","1"],["56700","0.0001","0","1"],["56902.89","0.0001","0","1"],["57111.52","0.0005","0","1"],["58700","0.0001","0","1"],["59000","0.0971","0","1"],["59529","0.0008","0","1"],["59996","0.0502","0","1"],["60000","0.0131","0","3"],["60250","0.0166","0","1"],["60666","0.0002","0","1"],["61000","0.024","0","1"],["61900","0.0015","0","1"],["62000","0.01","0","1"],["62234","0.0132","0","1"],["62700","0.0001","0","1"],["62721.57","0.0008","0","1"],["63000","0.0029","0","2"],["63200","0.0017","0","1"],["65000","0.002","0","1"],["65700","0.0001","0","1"],["66000","0.01","0","1"],["67050.52","0.0008","0","1"],["67495","1.8198","0","1"],["67500","0.0042","0","1"],["68000","0.01","0","1"],["68040.94","0.0002","0","1"],["68320","0.0001","0","1"],["68495","1","0","1"],["68978.1","0.0007","0","1"],["69000","0.1981","0","3"],["69940","0.0001","0","1"],["70000","0.0051","0","5"],["71000","2.0836","0","4"],["72000","0.0021","0","1"],["75000","0.0038","0","4"],["75500.9","0.0002","0","1"],["77111.52","0.0006","0","1"],["77777.77","0.012","0","1"],["80000","0.0858","0","5"],["84848","0.0203","0","1"],["88000","0.0001","0","1"],["90000","0.0035","0","3"],["99000","0.0015","0","1"],["100000","0.0017","0","2"],["110000","0.0012","0","2"],["114880","0.0019","0","1"],["180000","0.0001","0","1"],["229700","0.001","0","1"],["409000","0.0003","0","1"],["507610","0.0408","0","1"],["999999","0.0001","0","1"],["1000000","0.0002","0","1"],["1989877","0.015","0","1"],["2000000","0.0064","0","1"]],"bids":[["30080.49","0.0225","0","1"],["30079.19","0.1","0","1"],["30077.48","0.0249","0","1"],["30074.47","0.0238","0","1"],["30073.17","0.1","0","1"],["30071.46","0.0252","0","1"],["30068.45","0.0256","0","1"],["30067.16","0.1","0","1"],["30023.66","0.0799","0","1"],["29993.54","0.0815","0","1"],["29963.43","0.0963","0","1"],["29933.31","0.0801","0","1"],["29903.2","0.0902","0","1"],["29840.5","0.7798","0","1"],["29750.16","0.6772","0","1"],["29659.83","0.7226","0","1"],["29569.49","0.7928","0","1"],["29555","0.0034","0","1"],["29479.16","0.7869","0","1"],["28904.16","0.0075","0","1"],["28800","0.0004","0","2"],["28750","0.0003","0","2"],["28644.02","0.0241","0","1"],["28516","0.0024","0","1"],["28386.22","0.0074","0","1"],["28130.74","0.0039","0","1"],["27877.56","0.0294","0","1"],["27626.66","0.0209","0","1"],["27400","0.001","0","1"],["27378.02","0.0008","0","1"],["27131.61","0.0357","0","1"],["27000","0.0022","0","1"],["26887.42","0.0208","0","1"],["26645.43","0.0358","0","1"],["26600","0.2622","0","1"],["26516","0.003","0","1"],["26405.62","0.0299","0","1"],["26167.96","0.017","0","1"],["25932.44","0.0041","0","1"],["25699.04","0.0283","0","1"],["25467.74","0.0349","0","1"],["25238.53","0.0234","0","1"],["25104.9","4.6891","0","1"],["25011.38","0.0214","0","1"],["24800","0.0093","0","2"],["24786.27","0.0032","0","1"],["24650","0.0088","0","1"],["24563.19","0.0014","0","1"],["24516","0.004","0","1"],["24350","0.0063","0","1"],["24342.12","0.0208","0","1"],["23990","0.4368","0","1"],["23516","0.0042","0","1"],["23216","0.1157","0","1"],["22516","0.0044","0","1"],["21516","0.0046","0","1"],["21000","0.384","0","1"],["20000","0.0871","0","2"],["19000","0.01","0","1"],["18500","0.0001","0","1"],["18000","0.0001","0","1"],["17600","0.0049","0","1"],["17300","0.0029","0","1"],["17010","0.1282","0","1"],["16000","0.078","0","1"],["15250","0.0163","0","1"],["15000","0.0233","0","2"],["14750","0.0169","0","1"],["14700","0.0206","0","1"],["14500","0.0172","0","1"],["14299","0.028","0","1"],["14250","0.0175","0","1"],["14111","0.1","0","1"],["14050","0.0001","0","1"],["14000","0.0178","0","1"],["13750","0.0181","0","1"],["13568","0.04","0","1"],["13500","0.0333","0","2"],["13321","0.8","0","1"],["13264.11","0.0262","0","1"],["13250","0.0188","0","1"],["13010","0.031","0","1"],["13000","0.0574","0","3"],["12800","0.05","0","1"],["12750","0.0196","0","1"],["12651","0.1185","0","1"],["12525","0.02","0","1"],["12505","0.0001","0","1"],["12500","0.02","0","1"],["12498","0.85","0","1"],["12490","0.04","0","1"],["12000","0.0616","0","2"],["11782.83","0.04","0","1"],["11629","0.07","0","1"],["11000","0.0001","0","1"],["10650","0.0004","0","1"],["10000","0.2059","0","4"],["9782.83","0.05","0","1"],["9769.14","0.001","0","1"],["9723.89","0.0019","0","1"],["9589","0.1015","0","1"],["9307","0.0238","0","1"],["8964.64","0.05","0","1"],["7964.64","0.1","0","1"],["7570","0.0006","0","1"],["7000","0.0178","0","1"],["6964.64","0.1","0","1"],["5690","0.4168","0","1"],["5000","1.6422","0","10"],["4980","0.0836","0","1"],["4960","5.6643","0","1"],["4846","0.0001","0","1"],["4490","0.0011","0","1"],["4049.92","1","0","1"],["4000","0.0014","0","1"],["3575","0.2","0","1"],["2857.6","0.9265","0","1"],["2500","0.0422","0","1"],["2310.91","0.8","0","1"],["2250.91","0.8821","0","1"],["2000","0.1001","0","2"],["1950","0.99","0","1"],["1900","0.0001","0","1"],["1810.91","1.1344","0","1"],["1800","0.9","0","1"],["1500","0.0008","0","1"],["1310.91","1.567","0","1"],["1300","0.022","0","1"],["1200","0.0215","0","1"],["1099","0.0033","0","1"],["1000","1.6149","0","5"],["900","1.432","0","1"],["816","1.105","0","1"],["800","0.9141","0","2"],["766","1.359","0","1"],["653","0.05","0","1"],["640.86","0.0011","0","1"],["583","0.09","0","1"],["577","0.13","0","1"],["573","0.17","0","1"],["567","0.21","0","1"],["563","0.19","0","1"],["500","2.721","0","7"],["499","0.0001","0","1"],["449","1.0021","0","1"],["400","0.0001","0","1"],["388","0.01","0","1"],["380","0.0001","0","1"],["379","0.0001","0","1"],["350","0.0056","0","1"],["340","0.0001","0","1"],["305.75","0.01","0","1"],["303.75","0.01","0","1"],["301.75","0.015","0","1"],["300","0.0092","0","3"],["287","1","0","1"],["212","0.01","0","1"],["200","2.0206","0","4"],["195.1","0.0037","0","1"],["180","0.9","0","1"],["160","0.0543","0","6"],["156","0.9838","0","2"],["150","1","0","1"],["140","0.0002","0","2"],["120","0.0053","0","2"],["100","6.1461","0","19"],["99.75","0.0021","0","1"],["93","1","0","1"],["91.78","0.0023","0","1"],["90","0.2665","0","1"],["70","0.0001","0","1"],["63.28","0.0126","0","1"],["56.35","0.0003","0","1"],["55","0.0009","0","1"],["52","1","0","1"],["50","1.1309","0","6"],["49.99","0.0014","0","1"],["49","0.0009","0","1"],["47","0.001","0","1"],["42","0.0009","0","1"],["38","1.0223","0","1"],["35","0.0308","0","1"],["32","0.1","0","1"],["30","0.0004","0","2"],["25.46","0.432","0","1"],["25.23","0.6341","0","1"],["25.04","1.5175","0","1"],["25.01","0.0004","0","1"],["25","6.655","0","10"],["24","0.12","0","2"],["23.01","0.0055","0","2"],["23","0.0013","0","2"],["20","0.1679","0","4"],["19.06","0.001","0","2"],["18","0.9","0","1"],["17.13","0.0002","0","1"],["15","1.0003","0","2"],["13.32","0.5255","0","1"],["13","1","0","1"],["11.34","0.97","0","1"],["11.24","1.6903","0","1"],["11.22","2.5846","0","1"],["11","30.6076","0","1"],["10.56","1","0","1"],["10.24","1.8554","0","1"],["10.13","3.8499","0","1"],["10","5.7326","0","10"],["9.92","1.0009","0","1"],["8","0.0001","0","1"],["6.53","0.0382","0","1"],["6.12","1.0003","0","2"],["5","1.0602","0","6"],["4.79","0.9","0","1"],["4","0.0009","0","1"],["3.2","0.1271","0","4"],["3","2.0002","0","2"],["2.65","0.001","0","1"],["2.5","1","0","1"],["2.48","0.0044","0","1"],["2.23","6.278","0","1"],["2","4.1978","0","6"],["1.99","1","0","1"],["1.7","0.007","0","1"],["1.64","4.233","0","1"],["1.6","21.9051","0","31"],["1.53","1.002","0","1"],["1.5","3","0","1"],["1.48","1","0","1"],["1.39","2","0","1"],["1.38","116.2888","0","1"],["1.18","0.0008","0","1"],["1.11","1.111","0","1"],["1","2589.1261","0","34"],["0.99","0.0015","0","1"],["0.9","0.0053","0","1"],["0.87","0.0018","0","2"],["0.8","0.0015","0","1"],["0.78","10.0001","0","2"],["0.75","0.0001","0","1"],["0.7","0.0014","0","1"],["0.69","0.0008","0","1"],["0.63","0.3959","0","1"],["0.62","0.9968","0","1"],["0.54","3.1526","0","1"],["0.5","23.0221","0","2"],["0.48","1","0","1"],["0.45","0.45","0","1"],["0.42","10","0","1"],["0.3","10","0","1"],["0.22","20","0","1"],["0.2","68.5607","0","10"],["0.18","1.0092","0","1"],["0.16","20","0","1"],["0.13","100","0","1"],["0.12","20","0","1"],["0.11","0.5989","0","2"],["0.1","10521.5651","0","14"],["0.09","33.1","0","4"],["0.08","26","0","2"],["0.07","29","0","2"],["0.06","35.0637","0","3"],["0.05","262.1125","0","7"],["0.04","2626.8527","0","4"],["0.03","405.1061","0","4"],["0.02","633.8765","0","13"],["0.01","15617.3359","0","65"]],"ts":"1688967955142","checksum":-872896843}]}`)
	orderbookUpdate4000   = []byte(`{"arg":{"channel":"books","instId":"BTC-USD"},"action":"update","data":[{"asks":[["30382.6","0","0","0"]],"bids":[],"ts":"1688967974408","checksum":1109161389}]}`)
)

func TestEvaluateChecksumCalculation(t *testing.T) {
	t.Parallel()
	err := o.WsHandleData(orderbookSnapshot4000)
	if err != nil {
		t.Error(err)
	}
	err = o.WsHandleData(orderbookUpdate4000)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	if err := o.UpdateOrderExecutionLimits(context.Background(), asset.Spot); err != nil {
		t.Errorf("Error fetching %s pairs for test: %v", asset.Spot, err)
	}
	instrumentInfo, err := o.GetInstruments(context.Background(), "SPOT", spotTradablePair.String())
	if err != nil {
		t.Error(err)
	}
	if len(instrumentInfo) != 1 {
		t.Fatal("invalid instrument information found")
	}
	limits, err := o.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
	if err != nil {
		t.Errorf("Okcoin GetOrderExecutionLimits() error during TestExecutionLimits; Asset: %s Pair: %s Err: %v", asset.Spot, spotTradablePair, err)
	}
	if got := limits.PriceStepIncrementSize; got != instrumentInfo[0].TickSize.Float64() {
		t.Errorf("Okcoin UpdateOrderExecutionLimits wrong PriceStepIncrementSize; Asset: %s Pair: %s Expected: %v Got: %v", asset.Spot, spotTradablePair, instrumentInfo[0].TickSize.Float64(), got)
	}

	if got := limits.MinimumBaseAmount; got != instrumentInfo[0].MinSize.Float64() {
		t.Errorf("Okcoin UpdateOrderExecutionLimits wrong MinAmount; Pair: %s Expected: %v Got: %v", spotTradablePair, instrumentInfo[0].MinSize.Float64(), got)
	}
	if got := limits.MaxIcebergParts; got != instrumentInfo[0].MaxIcebergSz.Int64() {
		t.Errorf("Okcoin UpdateOrderExecutionLimits MaxIcebergSize; Pair: %s Expected: %v Got: %v", spotTradablePair, instrumentInfo[0].MaxIcebergSz.Int64(), got)
	}
	if got := limits.MarketMaxQty; got != instrumentInfo[0].MaxMarketSize.Float64() {
		t.Errorf("Okcoin UpdateOrderExecutionLimits MaxMarketSizize; Pair: %s Expected: %v Got: %v", spotTradablePair, instrumentInfo[0].MaxMarketSize.Float64(), got)
	}
}

func TestReSubscribeSpecificOrderbook(t *testing.T) {
	t.Parallel()
	err := o.ReSubscribeSpecificOrderbook(wsOrderbooks, spotTradablePair)
	if err != nil {
		t.Error(err)
	}
}
