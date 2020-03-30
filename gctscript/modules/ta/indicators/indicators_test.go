package indicators

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/go-talib/indicators"
)

var (
	ohlcvData = &objects.Array{}
)

func TestMain(m *testing.M) {
	for x := 0; x < 100; x++ {
		v := rand.Float64()
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

	os.Exit(m.Run())
}

func TestMfi(t *testing.T) {
	t.Parallel()
	_, err := mfi()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = mfi(ohlcvData, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = mfi(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
}

func TestRsi(t *testing.T) {
	t.Parallel()
	_, err := rsi()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = rsi(ohlcvData, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = rsi(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
}

func TestEMA(t *testing.T) {
	t.Parallel()
	_, err := ema()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = ema(ohlcvData, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = ema(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
}

func TestSMA(t *testing.T) {
	t.Parallel()
	_, err := sma()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = sma(ohlcvData, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = sma(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
}

func TestMACD(t *testing.T) {
	t.Parallel()
	_, err := macd()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = macd(ohlcvData, &objects.Int{Value: 12}, &objects.Int{Value: 26}, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = macd(ohlcvData, &objects.Int{Value: 12}, &objects.Int{Value: 26}, &objects.Int{Value: 9})
	if err != nil {
		t.Error(err)
	}
}

func TestAtr(t *testing.T) {
	t.Parallel()
	_, err := atr()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = atr(ohlcvData, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = atr(ohlcvData, &objects.Int{Value: 14})
	if err != nil {
		t.Error(err)
	}
}

func TestBbands(t *testing.T) {
	t.Parallel()
	_, err := bbands()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	_, err = bbands(&objects.String{Value: "Hello"}, ohlcvData,
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
}

func TestOBV(t *testing.T) {
	t.Parallel()
	_, err := obv()
	if err != nil {
		if !errors.Is(err, objects.ErrWrongNumArguments) {
			t.Error(err)
		}
	}

	v := &objects.String{Value: "Hello"}
	_, err = obv(v, ohlcvData)
	if err != nil {
		if err != errInvalidSelector {
			t.Error(err)
		}
	}

	s := &objects.Int{Value: 1}
	_, err = obv(s, ohlcvData)
	if err != nil {
		if err != errInvalidSelector {
			t.Error(err)
		}
	}

	_, err = obv(&objects.String{Value: "close"}, ohlcvData)
	if err != nil {
		t.Error(err)
	}
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
			indicators.SMA,
			nil,
		},
		{
			"ema",
			indicators.EMA,
			nil,
		},
		{
			"wma",
			indicators.WMA,
			nil,
		},
		{
			"dema",
			indicators.DEMA,
			nil,
		},
		{
			"tema",
			indicators.TEMA,
			nil,
		},
		{
			"trima",
			indicators.TRIMA,
			nil,
		},
		{
			"kama",
			indicators.KAMA,
			nil,
		},
		{
			"mama",
			indicators.MAMA,
			nil,
		},
		{
			"t3ma",
			indicators.T3MA,
			nil,
		},
		{
			"no",
			indicators.SMA,
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
