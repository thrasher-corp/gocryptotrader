package indicators

import (
	"math/rand"
	"os"
	"testing"
	"time"

	objects "github.com/d5/tengo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	for x := range 100 {
		v := rand.Float64() //nolint:gosec // no need to import crypo/rand for testing
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

	for range 5 {
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	v := &objects.String{Value: testString}
	_, err = rsi(ohlcvData, v)
	assert.ErrorContains(t, err, errFailedConversion)

	_, err = rsi(ohlcvData, &objects.Int{Value: 14})
	assert.NoError(t, err, "rsi should not throw an error on valid input")

	_, err = rsi(v, &objects.Int{Value: 14})
	assert.ErrorContains(t, err, "failed conversion")

	_, err = rsi(ohlcvDataInvalid, &objects.Int{Value: 14})
	assert.ErrorContains(t, err, "failed conversion")

	validator.IsTestExecution.Store(true)
	ret, err := rsi(ohlcvData, &objects.Int{Value: 14})
	require.NoError(t, err, "rsi must not throw an error")
	assert.NotNil(t, ret)

	validator.IsTestExecution.Store(false)
}

func TestEMA(t *testing.T) {
	_, err := ema()
	if err != nil {
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
		assert.ErrorIs(t, err, errInvalidSelector)
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
		assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
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
	t.Parallel()
	for _, tc := range []struct {
		name      string
		input     any
		expected  float64
		expectErr bool
	}{
		{"float64", 45.67, 45.67, false},
		{"int", int(45), 45.0, false},
		{"int32", int32(45), 45.0, false},
		{"int64", int64(45), 45.0, false},
		{"string", "45.67", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := toFloat64(tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
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

	for _, test := range testCases {
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

	for _, test := range testCases {
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
