package gct

import (
	"os"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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

	tv = objects.TrueValue
	fv = objects.FalseValue
)

func TestMain(m *testing.M) {
	modules.SetModuleWrapper(validator.Wrapper{})
	os.Exit(m.Run())
}

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderbook(ctx, exch, currencyPair, delimiter, assetType)
	assert.NoError(t, err)

	_, err = ExchangeOrderbook()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := ExchangeTicker(ctx, exch, currencyPair, delimiter, assetType)
	assert.NoError(t, err)

	_, err = ExchangeTicker()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()

	_, err := ExchangePairs(exch, tv, assetType)
	assert.NoError(t, err)

	_, err = ExchangePairs(exchError, tv, assetType)
	assert.NoError(t, err)

	_, err = ExchangePairs()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestAccountBalances(t *testing.T) {
	t.Parallel()

	_, err := ExchangeAccountBalances()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = ExchangeAccountBalances(ctx, exch, assetType)
	assert.NoError(t, err)

	_, err = ExchangeAccountBalances(ctx, exchError, assetType)
	assert.NoError(t, err)
}

func TestExchangeOrderQuery(t *testing.T) {
	t.Parallel()

	_, err := ExchangeOrderQuery()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = ExchangeOrderQuery(ctx, exch, orderID)
	assert.NoError(t, err)

	_, err = ExchangeOrderQuery(ctx, exchError, orderID)
	assert.NoError(t, err)
}

func TestExchangeOrderCancel(t *testing.T) {
	t.Parallel()
	_, err := ExchangeOrderCancel()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

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
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	orderSide := &objects.String{Value: "ASK"}
	orderType := &objects.String{Value: "LIMIT"}
	orderPrice := &objects.Float{Value: 1}
	orderAmount := &objects.Float{Value: 1}
	orderAsset := &objects.String{Value: asset.Spot.String()}

	_, err = ExchangeOrderSubmit(ctx, exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	assert.NoError(t, err)

	_, err = ExchangeOrderSubmit(ctx, exch, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	assert.NoError(t, err)

	_, err = ExchangeOrderSubmit(ctx, objects.TrueValue, currencyPair, delimiter,
		orderType, orderSide, orderPrice, orderAmount, orderID, orderAsset)
	assert.NoError(t, err)
}

func TestAllModuleNames(t *testing.T) {
	t.Parallel()
	require.NotEmpty(t, AllModuleNames(), "AllModuleNames must not return an empty slice")
}

func TestExchangeDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := ExchangeDepositAddress()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	currCode := &objects.String{Value: "BTC"}
	chain := &objects.String{Value: ""}
	_, err = ExchangeDepositAddress(exch, currCode, chain)
	if err != nil {
		t.Error(err)
	}

	_, err = ExchangeDepositAddress(exchError, currCode, chain)
	assert.NoError(t, err)
}

func TestExchangeWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := ExchangeWithdrawCrypto()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

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
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

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
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "parseInterval should return invalid interval for 6m")
}

func TestSetVerbose(t *testing.T) {
	t.Parallel()
	_, err := setVerbose()
	require.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = setVerbose(objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	resp, err := setVerbose(&Context{})
	require.NoError(t, err)

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
	require.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = setAccount(objects.TrueValue, objects.TrueValue, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, objects.TrueValue, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, dummyStr, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, dummyStr, dummyStr, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	resp, err := setAccount(&Context{}, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr, dummyStr)
	require.NoError(t, err)

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
	require.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = setSubAccount(objects.TrueValue, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	_, err = setSubAccount(&Context{}, objects.TrueValue)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	subby, err := setSubAccount(&Context{}, dummyStr)
	require.NoError(t, err)

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

	subaccount, ok := ctx.Value(accounts.ContextSubAccountFlag).(string)
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
	require.NoError(t, err)

	fromScript, err = setVerbose(fromScript)
	require.NoError(t, err)

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
