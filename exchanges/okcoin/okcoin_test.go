package okcoin

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var (
	o                    = &OKCoin{}
	spotCurrency         = currency.NewPairWithDelimiter(currency.BTC.String(), currency.USD.String(), "-")
	spotCurrencyLowerStr = spotCurrency.Lower().String()
	spotCurrencyUpperStr = spotCurrency.Upper().String()
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
		log.Fatal("OKCoin setup error", err)
	}
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
	o.Verbose = true
	_, err := o.GetInstruments(context.Background(), "SPOT", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemStatus(t *testing.T) {
	t.Parallel()
	o.Verbose = true
	// allowed state value: ongoing, scheduled, processing, pre_open, completed, canceled
	_, err := o.GetSystemStatus(context.Background(), "scheduled")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "ongoing")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "processing")
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.GetSystemStatus(context.Background(), "pre_open")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	systemTime, err := o.GetSystemTime(context.Background())
	if err != nil {
		t.Fatal(err)
	} else {
		println(systemTime.String())
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := o.GetTickers(context.Background(), "SPOT")
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
	_, err := o.GetOrderbook(context.Background(), "BTC-USD", 200)
	if err != nil {
		t.Error(err)
	}
}

func TestGetLiteOrderbook(t *testing.T) {
	t.Parallel()
	_, err := o.GetOrderbookLitebook(context.Background(), "BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlesticks(context.Background(), "BTC-USD", kline.FiveMin, time.Now().Add(-time.Hour*3), time.Now(), 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestickHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetCandlestickHistory(context.Background(), "BTC-USD", time.Now().Add(-time.Minute*30), time.Now(), kline.FiveMin, 0)
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
	_, err := o.GetTradeHistory(context.Background(), "BTC-USD", "", time.Time{}, time.Time{}, 0)
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
	o.Verbose = true
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

func TestWsConnect(t *testing.T) {
	err := o.WsConnect()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 25)
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
	if !errors.Is(err, errTransferIDOrClientIDRequred) {
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
	_, err := o.GetAssetBilsDetail(context.Background(), currency.BTC, "2", "", time.Now().Add(-time.Minute), time.Now().Add(-time.Hour), 0)
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
	_, err = o.Withdrawal(context.Background(), &WithdrawalRequest{Amount: 1, Ccy: currency.BTC, WithdrawalMethod: "1", ToAddress: "abcdefg"})
	if !errors.Is(err, errInvalidTrasactionFeeValue) {
		t.Fatalf("found %v, expected %v", err, errAddressMustNotBeEmptyString)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
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
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, o, canManipulateRealOrders)
	_, err := o.CancelWithdrawal(context.Background(), &WithdrawalCancelation{
		WithdrawalID: "1123456",
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
	_, err := o.PlaceRFQOrder(context.Background(), nil)
	if !errors.Is(err, errNilArgument) {
		t.Errorf("expected %v, but found %v", errNilArgument, err)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, o)
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
	_, err := o.GetChannelInfo(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}
