package indicators

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
)

const errFailedConversion = "0 failed conversion"

var (
	ohlcvData        = &objects.Array{}
	ohlcvDataInvalid = &objects.Array{}
	testString       = "1D10TH0R53"
)

func TestMain(m *testing.M) {
	for x := 0; x < 100; x++ {
		v := rand.Float64() // nolint:gosec // no need to import crypo/rand for testing
		candle := &objects.Array{}
		candle.Value = append(candle.Value, &objects.Time{Value: time.Now()},
			&objects.Float{Value: v},
			&objects.Float{Value: v + float64(x)},
			&objects.Float{Value: v - float64(x)},
			&objects.Float{Value: v},
			&objects.Float{Value: v},
		)
		ohlcvData.Value = append(ohlcvData.Value, candle)
	}

	for x := 0; x < 5; x++ {
		candle := &objects.Array{}
		candle.Value = append(candle.Value, &objects.String{Value: testString},
			&objects.String{Value: testString},
			&objects.String{Value: testString},
			&objects.String{Value: testString},
			&objects.String{Value: testString},
			&objects.String{Value: testString},
		)
		ohlcvDataInvalid.Value = append(ohlcvDataInvalid.Value, candle)
	}

	os.Exit(m.Run())
}

func TestMfi(t *testing.T) {
	_, err := mfi()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = mfi(ohlcvData, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = mfi(ohlcvDataInvalid, &objects.Int{Value: 14})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = mfi(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}

	_, err = mfi(v, &objects.Int{Value: 14})
	if err != nil {
		if err.Error() != "OHLCV data failed conversion" {
			t.Error(err)
		}
	}

	validator.IsTestExecution.Store(true)
	ret, err := mfi(ohlcvData, &objects.Int{Value: 10})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestRsi(t *testing.T) {
	_, err := rsi()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = rsi(ohlcvData, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = rsi(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}

	_, err = rsi(v, &objects.Int{Value: 14})
	if err == nil {
		if err.Error() != "OHLCV data failed conversion" {
			t.Error(err)
		}
	}

	_, err = rsi(ohlcvDataInvalid, &objects.Int{Value: 14})
	if err == nil {
		if err.Error() != "OHLCV data failed conversion" {
			t.Error(err)
		}
	}

	validator.IsTestExecution.Store(true)
	ret, err := rsi(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestEMA(t *testing.T) {
	_, err := ema()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = ema(ohlcvData, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = ema(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}

	_, err = ema(ohlcvDataInvalid, &objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = ema(&objects.String{Value: testString}, &objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	validator.IsTestExecution.Store(true)
	ret, err := ema(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestSMA(t *testing.T) {
	_, err := sma()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = sma(ohlcvData, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = sma(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}

	_, err = sma(ohlcvDataInvalid, &objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = sma(&objects.String{Value: testString}, &objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	validator.IsTestExecution.Store(true)
	ret, err := sma(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestMACD(t *testing.T) {
	_, err := macd()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = macd(ohlcvData, &objects.Int{Value: 12}, &objects.Int{Value: 26}, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = macd(ohlcvData, &objects.Int{Value: 12}, &objects.Int{Value: 26}, &objects.Int{Value: 9})
	if err != nil {
		t.Error(err)
	}

	_, err = macd(ohlcvDataInvalid,
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = macd(&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	validator.IsTestExecution.Store(true)
	ret, err := macd(ohlcvData, &objects.Int{Value: 12}, &objects.Int{Value: 26}, &objects.Int{Value: 9})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestAtr(t *testing.T) {
	_, err := atr()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: testString}
	_, err = atr(ohlcvData, v)
	if err != nil {
		if err.Error() != errFailedConversion {
			t.Error(err)
		}
	}

	_, err = atr(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}

	_, err = atr(v, &objects.Int{Value: 14})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = atr(ohlcvDataInvalid, &objects.Int{Value: 14})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	validator.IsTestExecution.Store(true)
	ret, err := atr(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestBbands(t *testing.T) {
	_, err := bbands()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	_, err = bbands(&objects.String{Value: testString}, ohlcvData,
		&objects.Int{Value: 5},
		&objects.Float{Value: 2.0},
		&objects.Float{Value: 2.0},
		&objects.String{Value: "sma"})
	if err != nil {
		if err != errInvalidSelector {
			t.Error(err)
		}
	}

	_, err = bbands(&objects.String{Value: "close"}, ohlcvData,
		&objects.Int{Value: 5},
		&objects.Float{Value: 2.0},
		&objects.Float{Value: 2.0},
		&objects.String{Value: "sma"})
	if err != nil {
		t.Error(err)
	}

	validator.IsTestExecution.Store(true)
	ret, err := bbands(&objects.String{Value: "close"}, ohlcvData,
		&objects.Int{Value: 5},
		&objects.Float{Value: 2.0},
		&objects.Float{Value: 2.0},
		&objects.String{Value: "sma"})
	if err != nil {
		t.Error(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)

	_, err = bbands(&objects.String{Value: "close"}, ohlcvDataInvalid,
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		objects.UndefinedValue)
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = bbands(&objects.String{Value: "close"}, &objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: testString},
		&objects.String{Value: "ema"})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = bbands(&objects.String{Value: "close"}, ohlcvData,
		&objects.Int{Value: 5},
		&objects.Float{Value: 2.0},
		&objects.Float{Value: 2.0},
		&objects.String{Value: testString})
	if err != nil {
		if !errors.Is(err, errInvalidSelector) {
			t.Error(err)
		}
	}

	_, err = bbands(objects.UndefinedValue, ohlcvData,
		&objects.Int{Value: 5},
		&objects.Float{Value: 2.0},
		&objects.Float{Value: 2.0},
		&objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}
}

func TestOBV(t *testing.T) {
	_, err := obv()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	_, err = obv(ohlcvData)
	if err != nil {
		t.Error(err)
	}

	_, err = obv(ohlcvDataInvalid)
	if err == nil {
		t.Error("expected conversion failed error")
	}

	_, err = obv(&objects.String{Value: testString})
	if err == nil {
		t.Error("expected conversion failed error")
	}

	validator.IsTestExecution.Store(true)
	ret, err := obv(ohlcvData)
	if err != nil {
		t.Fatal(err)
	}
	if (ret == &objects.Array{}) {
		t.Error("expected empty Array on test execution received data")
	}
	validator.IsTestExecution.Store(false)
}

func TestToFloat64(t *testing.T) {
	value := 54.0
	v, err := toFloat64(value)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(v).Kind() != reflect.Float64 {
		t.Fatalf("expected toFloat to return kind float64 received: %v", reflect.TypeOf(v).Kind())
	}

	v, err = toFloat64(int(value))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(v).Kind() != reflect.Float64 {
		t.Fatalf("expected toFloat to return kind float64 received: %v", reflect.TypeOf(v).Kind())
	}

	v, err = toFloat64(int32(value))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(v).Kind() != reflect.Float64 {
		t.Fatalf("expected toFloat to return kind float64 received: %v", reflect.TypeOf(v).Kind())
	}

	v, err = toFloat64(int64(value))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.TypeOf(v).Kind() != reflect.Float64 {
		t.Fatalf("expected toFloat to return kind float64 received: %v", reflect.TypeOf(v).Kind())
	}

	_, err = toFloat64("54")
	if err == nil {
		t.Fatalf("attempting to convert a string should fail but test passed")
	}
}

func TestParseIndicatorSelector(t *testing.T) {
	testCases := []struct {
		name     string
		expected int
		err      error
	}{
		{
			"open",
			1,
			nil,
		},
		{
			"high",
			2,
			nil,
		},
		{
			"low",
			3,
			nil,
		},
		{
			"close",
			4,
			nil,
		},
		{
			"vol",
			5,
			nil,
		},
		{
			"invalid",
			0,
			errInvalidSelector,
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			v, err := ParseIndicatorSelector(test.name)
			if err != nil {
				if err != test.err {
					t.Fatal(err)
				}
			}
			if v != test.expected {
				t.Fatalf("expected %v received %v", test.expected, v)
			}
		})
	}
}

func TestParseMAType(t *testing.T) {
	testCases := []struct {
		name     string
		expected indicators.MaType
		err      error
	}{
		{
			"sma",
			indicators.Sma,
			nil,
		},
		{
			"ema",
			indicators.Ema,
			nil,
		},
		{
			"no",
			indicators.Sma,
			errInvalidSelector,
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			v, err := ParseMAType(test.name)
			if err != nil {
				if err != test.err {
					t.Fatal(err)
				}
			}
			if v != test.expected {
				t.Fatalf("expected %v received %v", test.expected, v)
			}
		})
	}
}
