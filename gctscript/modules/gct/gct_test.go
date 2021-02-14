package gct

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

var (
	exch = &objects.String{
		Value: "BTC Markets",
	}
	exchError = &objects.String{
		Value: "error",
	}
	currencyPair = &objects.String{
		Value: "BTC-AUD",
	}
	delimiter = &objects.String{
		Value: "-",
	}
	assetType = &objects.String{
		Value: "SPOT",
	}
	orderID = &objects.String{
		Value: "1235",
	}
	blank = &objects.String{
		Value: "",
	}

	tv            = objects.TrueValue
	fv            = objects.FalseValue
	errTestFailed = errors.New("test failed")
)

func TestMain(m *testing.M) {
	modules.SetModuleWrapper(validator.Wrapper{})
	os.Exit(m.Run())
}

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderbook(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderbook(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Error(err)
	}

	_, err = ExchangeOrderbook()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := ExchangeTicker(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeTicker(exchError, currencyPair, delimiter, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Error(err)
	}

	_, err = ExchangeTicker()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangeExchanges(t *testing.T) {
	t.Parallel()

	_, err := ExchangeExchanges(tv)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeExchanges(exch)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeExchanges(fv)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeExchanges()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()

	_, err := ExchangePairs(exch, tv, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangePairs(exchError, tv, assetType)
	if err != nil && errors.Is(err, errTestFailed) {
		t.Error(err)
	}

	_, err = ExchangePairs()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}
}

func TestAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := ExchangeAccountInfo()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	_, err = ExchangeAccountInfo(exch, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeAccountInfo(exchError, assetType)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Error(err)
	}
}

func TestExchangeOrderQuery(t *testing.T) {
	t.Parallel()

	_, err := ExchangeOrderQuery()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	_, err = ExchangeOrderQuery(exch, orderID)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderQuery(exchError, orderID)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Error(err)
	}
}

func TestExchangeOrderCancel(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderCancel()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	_, err = ExchangeOrderCancel(blank, orderID, currencyPair, assetType)
	if err == nil {
		t.Error("expecting error")
	}

	_, err = ExchangeOrderCancel(exch, blank, currencyPair, assetType)
	if err == nil {
		t.Error("expecting error")
	}

	_, err = ExchangeOrderCancel(exch, orderID)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderCancel(exch, orderID, currencyPair)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderCancel(exch, orderID, currencyPair, assetType)
	if err != nil {
		t.Error(err)
	}
}

func TestExchangeOrderSubmit(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderSubmit()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	orderSide := &objects.String{Value: "ASK"}
	orderType := &objects.String{Value: "LIMIT"}
	orderPrice := &objects.Float{Value: 1}
	orderAmount := &objects.Float{Value: 1}
	orderAsset := &objects.String{Value: asset.Spot.String()}

	_, err = ExchangeOrderSubmit(exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Error(err)
	}

	_, err = ExchangeOrderSubmit(exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderSubmit(objects.TrueValue, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil {
		t.Error(err)
	}
}

func TestAllModuleNames(t *testing.T) {
	t.Parallel()
	x := AllModuleNames()
	xType := reflect.TypeOf(x).Kind()
	if xType != reflect.Slice {
		t.Errorf("AllModuleNames() should return slice instead received: %v", x)
	}
}

func TestExchangeDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := ExchangeDepositAddress()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	currCode := &objects.String{Value: "BTC"}
	_, err = ExchangeDepositAddress(exch, currCode)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeDepositAddress(exchError, currCode)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Error(err)
	}
}

func TestExchangeWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := ExchangeWithdrawCrypto()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	currCode := &objects.String{Value: "BTC"}
	desc := &objects.String{Value: "HELLO"}
	address := &objects.String{Value: "0xTHISISALEGITBTCADDRESSS"}
	amount := &objects.Float{Value: 1.0}

	_, err = ExchangeWithdrawCrypto(exch, currCode, address, address, amount, amount, desc)
	if err != nil {
		t.Error(err)
	}
}

func TestExchangeWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := ExchangeWithdrawFiat()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Error(err)
	}

	currCode := &objects.String{Value: "AUD"}
	desc := &objects.String{Value: "Hello"}
	amount := &objects.Float{Value: 1.0}
	bankID := &objects.String{Value: "test-bank-01"}
	_, err = ExchangeWithdrawFiat(exch, currCode, desc, amount, bankID)
	if err != nil {
		t.Error(err)
	}
}

func TestParseInterval(t *testing.T) {
	t.Parallel()
	v, err := parseInterval("1h")
	if err != nil {
		t.Error(err)
	}
	if v != time.Hour {
		t.Fatalf("unexpected value return expected %v received %v", time.Hour, v)
	}

	v, err = parseInterval("1d")
	if err != nil {
		t.Error(err)
	}
	if v != time.Hour*24 {
		t.Errorf("unexpected value return expected %v received %v", time.Hour*24, v)
	}

	v, err = parseInterval("3d")
	if err != nil {
		t.Error(err)
	}
	if v != time.Hour*72 {
		t.Errorf("unexpected value return expected %v received %v", time.Hour*72, v)
	}

	v, err = parseInterval("1w")
	if err != nil {
		t.Error(err)
	}
	if v != time.Hour*168 {
		t.Errorf("unexpected value return expected %v received %v", time.Hour*168, v)
	}

	_, err = parseInterval("6m")
	if err != nil {
		if !errors.Is(err, errInvalidInterval) {
			t.Error(err)
		}
	}
}
