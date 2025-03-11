package convert

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFloatFromString(t *testing.T) {
	t.Parallel()
	testString := "1.41421356237"
	expectedOutput := float64(1.41421356237)

	actualOutput, err := FloatFromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Common FloatFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = FloatFromString(testByte)
	if err == nil {
		t.Error("Common FloatFromString. Converted non-string.")
	}

	testString = "   something unconvertible  "
	_, err = FloatFromString(testString)
	if err == nil {
		t.Error("Common FloatFromString. Converted invalid syntax.")
	}
}

func TestIntFromString(t *testing.T) {
	t.Parallel()
	testString := "1337"
	actualOutput, err := IntFromString(testString)
	if expectedOutput := 1337; actualOutput != expectedOutput || err != nil {
		t.Errorf("Common IntFromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = IntFromString(testByte)
	if err == nil {
		t.Error("Common IntFromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = IntFromString(testString)
	if err == nil {
		t.Error("Common IntFromString. Converted invalid syntax.")
	}
}

func TestInt64FromString(t *testing.T) {
	t.Parallel()
	testString := "4398046511104"
	expectedOutput := int64(1 << 42)

	actualOutput, err := Int64FromString(testString)
	if actualOutput != expectedOutput || err != nil {
		t.Errorf("Common Int64FromString. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	var testByte []byte
	_, err = Int64FromString(testByte)
	if err == nil {
		t.Error("Common Int64FromString. Converted non-string.")
	}

	testString = "1.41421356237"
	_, err = Int64FromString(testString)
	if err == nil {
		t.Error("Common Int64FromString. Converted invalid syntax.")
	}
}

func TestTimeFromUnixTimestampFloat(t *testing.T) {
	t.Parallel()
	testTimestamp := float64(1414456320000)
	expectedOutput := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)

	actualOutput, err := TimeFromUnixTimestampFloat(testTimestamp)
	if actualOutput.UTC().String() != expectedOutput.UTC().String() || err != nil {
		t.Errorf("Common TimeFromUnixTimestampFloat. Expected '%v'. Actual '%v'. Error: %s",
			expectedOutput, actualOutput, err)
	}

	testString := "Time"
	_, err = TimeFromUnixTimestampFloat(testString)
	if err == nil {
		t.Error("Common TimeFromUnixTimestampFloat. Converted invalid syntax.")
	}
}

func TestTimeFromUnixTimestampDecimal(t *testing.T) {
	for in, exp := range map[float64]time.Time{
		1590633982.5714:   time.Date(2020, 5, 28, 2, 46, 22, 571400000, time.UTC),
		1560516023.070651: time.Date(2019, 6, 14, 12, 40, 23, 70651000, time.UTC),
		// Examples from Kraken
		1373750306.9819:   time.Date(2013, 7, 13, 21, 18, 26, 981900000, time.UTC),
		1534614098.345543: time.Date(2018, 8, 18, 17, 41, 38, 345543000, time.UTC),
	} {
		got := TimeFromUnixTimestampDecimal(in)
		z, _ := got.Zone()
		assert.Equal(t, "UTC", z, "TimeFromUnixTimestampDecimal should return a UTC time")
		assert.WithinRangef(t, got, exp.Add(-time.Microsecond), exp.Add(time.Microsecond), "TimeFromUnixTimestampDecimal(%f) should parse a unix timestamp correctly", in)
	}
}

func TestUnixTimestampToTime(t *testing.T) {
	t.Parallel()
	testTime := int64(1489439831)
	tm := time.Unix(testTime, 0)
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult := UnixTimestampToTime(testTime)
	if tm.String() != actualResult.String() {
		t.Errorf(
			"Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
}

func TestUnixTimestampStrToTime(t *testing.T) {
	t.Parallel()
	testTime := "1489439831"
	incorrectTime := "DINGDONG"
	expectedOutput := "2017-03-13 21:17:11 +0000 UTC"
	actualResult, err := UnixTimestampStrToTime(testTime)
	if err != nil {
		t.Error(err)
	}
	if actualResult.UTC().String() != expectedOutput {
		t.Errorf(
			"Expected '%s'. Actual '%s'.", expectedOutput, actualResult)
	}
	_, err = UnixTimestampStrToTime(incorrectTime)
	if err == nil {
		t.Error("should throw an error")
	}
}

func TestBoolPtr(t *testing.T) {
	y := BoolPtr(true)
	if !*y {
		t.Fatal("true expected received false")
	}
	z := BoolPtr(false)
	if *z {
		t.Fatal("false expected received true")
	}
}

func TestFloatToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0.000", FloatToHumanFriendlyString(0, 3, ".", ","))
	assert.Equal(t, "100.000", FloatToHumanFriendlyString(100, 3, ".", ","))
	assert.Equal(t, "1,000.000", FloatToHumanFriendlyString(1000, 3, ".", ","))
	assert.Equal(t, "-1,000.000", FloatToHumanFriendlyString(-1000, 3, ".", ","))
	assert.Equal(t, "-1,000.0000000000", FloatToHumanFriendlyString(-1000, 10, ".", ","))
	assert.Equal(t, "1!000.1", FloatToHumanFriendlyString(1000.1337, 1, ".", "!"))
}

func TestDecimalToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", DecimalToHumanFriendlyString(decimal.Zero, 0, ".", ","))
	assert.Equal(t, "100", DecimalToHumanFriendlyString(decimal.NewFromInt(100), 0, ".", ","))
	assert.Equal(t, "1,000", DecimalToHumanFriendlyString(decimal.NewFromInt(1000), 0, ".", ","))
	assert.Equal(t, "-1,000", DecimalToHumanFriendlyString(decimal.NewFromInt(-1000), 0, ".", ","))
	assert.Equal(t, "-1~000!42", DecimalToHumanFriendlyString(decimal.NewFromFloat(-1000.42069), 2, "!", "~"))
	assert.Equal(t, "1,000.42069", DecimalToHumanFriendlyString(decimal.NewFromFloat(1000.42069), 5, ".", ","))
	assert.Equal(t, "1,000.42069", DecimalToHumanFriendlyString(decimal.NewFromFloat(1000.42069), 100, ".", ","))
}

func TestIntToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", IntToHumanFriendlyString(0, ","))
	assert.Equal(t, "100", IntToHumanFriendlyString(100, ","))
	assert.Equal(t, "1,000", IntToHumanFriendlyString(1000, ","))
	assert.Equal(t, "-1,000", IntToHumanFriendlyString(-1000, ","))
	assert.Equal(t, "-1!000", IntToHumanFriendlyString(-1000, "!"))
}

func TestNumberToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", numberToHumanFriendlyString("0", 0, "", ",", false))
	assert.Equal(t, "1,337.69", numberToHumanFriendlyString("1337.69", 2, ".", ",", false))
	assert.Equal(t, "-1!000.1", numberToHumanFriendlyString("1000.1", 1, ".", "!", true))
	assert.Equal(t, "1,000", numberToHumanFriendlyString("1000", 20, ".", ",", false))
}

func TestInterfaceToFloat64OrZeroValue(t *testing.T) {
	var x interface{}
	if r := InterfaceToFloat64OrZeroValue(x); r != 0 {
		t.Errorf("expected 0, got: %v", r)
	}
	x = float64(420)
	if r := InterfaceToFloat64OrZeroValue(x); r != 420 {
		t.Errorf("expected 420, got: %v", x)
	}
}

func TestInterfaceToIntOrZeroValue(t *testing.T) {
	var x interface{}
	if r := InterfaceToIntOrZeroValue(x); r != 0 {
		t.Errorf("expected 0, got: %v", r)
	}
	x = int(420)
	if r := InterfaceToIntOrZeroValue(x); r != 420 {
		t.Errorf("expected 420, got: %v", x)
	}
}

func TestInterfaceToStringOrZeroValue(t *testing.T) {
	var x interface{}
	if r := InterfaceToStringOrZeroValue(x); r != "" {
		t.Errorf("expected empty string, got: %v", r)
	}
	x = string("meow")
	if r := InterfaceToStringOrZeroValue(x); r != "meow" {
		t.Errorf("expected meow, got: %v", x)
	}
}
