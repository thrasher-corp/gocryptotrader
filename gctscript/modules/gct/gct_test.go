package gct

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

var (
	ctx  = &Context{}
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
	_, err := ExchangeOrderbook(ctx, exch, currencyPair, delimiter, assetType)
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
	_, err := ExchangeTicker(ctx, exch, currencyPair, delimiter, assetType)
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

	_, err = ExchangeAccountInfo(ctx, exch, assetType)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeAccountInfo(ctx, exchError, assetType)
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

	_, err = ExchangeOrderQuery(ctx, exch, orderID)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderQuery(ctx, exchError, orderID)
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

	_, err = ExchangeOrderCancel(ctx, exch, orderID)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderCancel(ctx, exch, orderID, currencyPair)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderCancel(ctx, exch, orderID, currencyPair, assetType)
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

	_, err = ExchangeOrderSubmit(ctx, exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil && !errors.Is(err, errTestFailed) {
		t.Error(err)
	}

	_, err = ExchangeOrderSubmit(ctx, exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeOrderSubmit(ctx, objects.TrueValue, currencyPair, delimiter,
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
	chain := &objects.String{Value: ""}
	_, err = ExchangeDepositAddress(exch, currCode, chain)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeDepositAddress(exchError, currCode, chain)
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

	_, err = ExchangeWithdrawCrypto(ctx, exch, currCode, address, address, amount, amount, desc)
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
	_, err = ExchangeWithdrawFiat(ctx, exch, currCode, desc, amount, bankID)
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

func TestSetVerbose(t *testing.T) {
	t.Parallel()
	_, err := setVerbose()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatalf("received: '%v' but expected: '%v'", err, objects.ErrWrongNumArguments)
	}

	_, err = setVerbose(objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	resp, err := setVerbose(&Context{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx, ok := objects.ToInterface(resp).(*Context)
	if !ok {
		t.Fatal("should be of type *Context")
	}

	val := ctx.Value["verbose"]
	if val.String() != objects.TrueValue.String() {
		t.Fatal("should contain verbose string in map")
	}
}

var dummyStr = &objects.String{Value: "xxxx"}

func TestSetAccount(t *testing.T) {
	t.Parallel()
	_, err := setAccount()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatalf("received: '%v' but expected: '%v'", err, objects.ErrWrongNumArguments)
	}

	_, err = setAccount(objects.TrueValue, objects.TrueValue, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, objects.TrueValue, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, dummyStr, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, dummyStr, dummyStr, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	resp, err := setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx, ok := objects.ToInterface(resp).(*Context)
	if !ok {
		t.Fatal("should be of type *Context")
	}

	val := ctx.Value["apikey"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain apikey string in map")
	}
	val = ctx.Value["apisecret"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain apisecret string in map")
	}
	val = ctx.Value["subaccount"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain subaccount string in map")
	}
	val = ctx.Value["clientid"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain clientid string in map")
	}
	val = ctx.Value["pemkey"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain pemkey string in map")
	}
	val = ctx.Value["otp"]
	if val.String() != dummyStr.String() {
		t.Fatal("should contain otp string in map")
	}
}

func TestSetSubAccount(t *testing.T) {
	t.Parallel()
	_, err := setSubAccount()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatalf("received: '%v' but expected: '%v'", err, objects.ErrWrongNumArguments)
	}

	_, err = setSubAccount(objects.TrueValue, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	_, err = setSubAccount(&Context{}, objects.TrueValue)
	if !errors.Is(err, common.ErrTypeAssertFailure) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrTypeAssertFailure)
	}

	subby, err := setSubAccount(&Context{}, dummyStr)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctxWSubAcc, ok := subby.(*Context)
	if !ok {
		t.Fatal("unexpected type returned")
	}

	if ctxWSubAcc.Value["subaccount"].String() != dummyStr.String() {
		t.Fatalf("received: '%v' but expected: '%v'", ctxWSubAcc.Value["subaccount"].String(), dummyStr.String())
	}

	// Deploy override to actual context.Context type
	ctx := processScriptContext(ctxWSubAcc)
	if ctx == nil {
		t.Fatal("should not be nil")
	}

	subaccount, ok := ctx.Value(account.ContextSubAccountFlag).(string)
	if !ok {
		t.Fatal("wrong type")
	}

	if subaccount != dummyStr.String()[1:5] {
		t.Fatalf("received: '%v' but expected: '%v'", subaccount, dummyStr.String()[1:5])
	}
}

func TestProcessScriptContext(t *testing.T) {
	t.Parallel()
	ctx := processScriptContext(nil)
	if ctx == nil {
		t.Fatal("should not be nil")
	}

	fromScript, err := setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	fromScript, err = setVerbose(fromScript)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	scriptCTX, ok := objects.ToInterface(fromScript).(*Context)
	if !ok {
		t.Fatal("should assert correctly")
	}

	ctx = processScriptContext(scriptCTX)
	if ctx == nil {
		t.Fatal("should not be nil")
	}
}

func TestScriptCredentialTypeName(t *testing.T) {
	t.Parallel()
	if name := (&Context{}).TypeName(); name != "scriptContext" {
		t.Fatal("unexpected value")
	}
}
