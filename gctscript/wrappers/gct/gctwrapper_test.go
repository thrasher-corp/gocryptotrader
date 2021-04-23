package gct

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
)

func TestMain(m *testing.M) {
	settings := engine.Settings{
		ConfigFile:                  filepath.Join("..", "..", "..", "testdata", "configtest.json"),
		EnableDryRun:                true,
		DataDir:                     filepath.Join("..", "..", "..", "testdata", "gocryptotrader"),
		EnableDepositAddressManager: true,
	}
	var err error
	engine.Bot, err = engine.NewFromSettings(&settings, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(exch.Value)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	exch.SetDefaults()
	em.Add(exch)
	engine.Bot.ExchangeManager = em
	engine.Bot.WithdrawManager, err = engine.SetupWithdrawManager(em, nil, true)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	engine.Bot.DepositAddressManager = engine.SetupDepositAddressManager()
	err = engine.Bot.DepositAddressManager.Sync(engine.Bot.GetExchangeCryptocurrencyDepositAddresses())
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	engine.Bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &engine.Bot.ServicesWG, false)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	err = engine.Bot.OrderManager.Start()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	modules.SetModuleWrapper(Setup())
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	x := Setup()
	xType := reflect.TypeOf(x).String()
	if xType != "*gct.Wrapper" {
		t.Fatalf("SetupCommunicationManager() should return pointer to Wrapper instead received: %v", x)
	}
}

var (
	exch = &objects.String{
		Value: "Bitstamp",
	}
	exchError = &objects.String{
		Value: "error",
	}
	currencyPair = &objects.String{
		Value: "BTCUSD",
	}
	delimiter = &objects.String{
		Value: "",
	}
	assetType = &objects.String{
		Value: "spot",
	}
	orderID = &objects.String{
		Value: "1235",
	}

	tv            = objects.TrueValue
	fv            = objects.FalseValue
	errTestFailed = errors.New("test failed")
)

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderbook(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeOrderbook(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = gct.ExchangeOrderbook()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeTicker(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeTicker(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = gct.ExchangeTicker()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangeExchanges(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeExchanges(tv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges(exch)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges(fv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangePairs(exch, tv, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangePairs(exchError, tv, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Fatal(err)
	}

	_, err = gct.ExchangePairs()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeAccountInfo()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
	_, err = gct.ExchangeAccountInfo(exch, assetType)
	if err != nil &&
		!strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
	}
}

func TestExchangeOrderQuery(t *testing.T) {
	t.Parallel()

	_, err := gct.ExchangeOrderQuery()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	_, err = gct.ExchangeOrderQuery(exch, orderID)
	if err != nil && err != common.ErrNotYetImplemented {
		t.Error(err)
	}
}

func TestExchangeOrderCancel(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderCancel()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
	_, err = gct.ExchangeOrderCancel(exch, orderID, currencyPair, assetType)
	if err != nil && err != common.ErrNotYetImplemented {
		t.Error(err)
	}
}

func TestExchangeOrderSubmit(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderSubmit()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	orderSide := &objects.String{Value: "ASK"}
	orderType := &objects.String{Value: "LIMIT"}
	orderPrice := &objects.Float{Value: 1}
	orderAmount := &objects.Float{Value: 1}
	orderAsset := &objects.String{Value: asset.Spot.String()}

	_, err = gct.ExchangeOrderSubmit(exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil &&
		!strings.Contains(err.Error(), "unset/default API keys") {
		t.Error(err)
	}
}

func TestAllModuleNames(t *testing.T) {
	t.Parallel()
	x := gct.AllModuleNames()
	xType := reflect.TypeOf(x).Kind()
	if xType != reflect.Slice {
		t.Errorf("AllModuleNames() should return slice instead received: %v", x)
	}
}

func TestExchangeDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeDepositAddress()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	currCode := &objects.String{Value: "BTC"}
	_, err = gct.ExchangeDepositAddress(exch, currCode)
	if err != nil && err.Error() != "deposit address store is nil" {
		t.Error(err)
	}
}

func TestExchangeWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeWithdrawCrypto()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	currCode := &objects.String{Value: "BTC"}
	desc := &objects.String{Value: "HELLO"}
	address := &objects.String{Value: "0xTHISISALEGITBTCADDRESSS"}
	amount := &objects.Float{Value: 1.0}

	_, err = gct.ExchangeWithdrawCrypto(exch, currCode, address, address, amount, amount, desc)
	if err != nil {
		t.Error(err)
	}
}

func TestExchangeWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeWithdrawFiat()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	currCode := &objects.String{Value: "TEST"}
	amount := &objects.Float{Value: 1.0}
	desc := &objects.String{Value: "2"}
	bankID := &objects.String{Value: "3!"}
	_, err = gct.ExchangeWithdrawFiat(exch, currCode, desc, amount, bankID)
	if err != nil && err.Error() != "exchange Bitstamp bank details not found for TEST" {
		t.Error(err)
	}
}
