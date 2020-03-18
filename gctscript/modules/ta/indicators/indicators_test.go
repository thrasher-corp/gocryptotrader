package indicators

import (
	"errors"
	"math/rand"
	"os"
	"reflect"
	"testing"

	objects "github.com/d5/tengo/v2"
)

var (
	testOpen  = &objects.Array{Value: []objects.Object{}}
	testHigh  = &objects.Array{Value: []objects.Object{}}
	testLow   = &objects.Array{Value: []objects.Object{}}
	testClose = &objects.Array{Value: []objects.Object{}}
	testVol   = &objects.Array{Value: []objects.Object{}}
)

func TestMain(m *testing.M) {
	for x := 0; x < 100; x++ {
		v := rand.Float64()
		testOpen.Value = append(testOpen.Value, &objects.Float{Value: v})
		testHigh.Value = append(testHigh.Value, &objects.Float{Value: v + float64(x)})
		testLow.Value = append(testLow.Value, &objects.Float{Value: v - float64(x)})
		testClose.Value = append(testClose.Value, &objects.Float{Value: v})
		testVol.Value = append(testVol.Value, &objects.Float{Value: float64(x)})
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
	_, err = mfi(testHigh, testLow, testClose, testVol, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = mfi(testHigh, testLow, testClose, testVol, &objects.Int{Value: 14})
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
	_, err = rsi(testClose, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = rsi(testClose, &objects.Int{Value: 14})
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
	_, err = ema(testClose, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = ema(testClose, &objects.Int{Value: 14})
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
	_, err = sma(testClose, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = sma(testClose, &objects.Int{Value: 14})
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
	_, err = macd(testClose, &objects.Int{Value: 12}, &objects.Int{Value: 26}, v)
	if err != nil {
		if err.Error() != "0 failed conversion" {
			t.Error(err)
		}
	}

	_, err = macd(testClose, &objects.Int{Value: 12}, &objects.Int{Value: 26}, &objects.Int{Value: 9})
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

	v, err = toFloat64("54")
	if err == nil {
		t.Fatalf("attempting to convert a string should fail but test passed")
	}
}