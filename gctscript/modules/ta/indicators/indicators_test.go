package indicators

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
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
