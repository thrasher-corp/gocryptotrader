package convert

import (
	"math"
	"testing"
	"time"
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
	expectedOutput := 1337

	actualOutput, err := IntFromString(testString)
	if actualOutput != expectedOutput || err != nil {
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
	actualResult, err = UnixTimestampStrToTime(incorrectTime)
	if err == nil {
		t.Error("Common UnixTimestampStrToTime error")
	}
}

func TestUnixMillis(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2014, time.October, 28, 0, 32, 0, 0, time.UTC)
	expectedOutput := int64(1414456320000)

	actualOutput := UnixMillis(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Common UnixMillis. Expected '%d'. Actual '%d'.",
			expectedOutput, actualOutput)
	}
}

func TestRecvWindow(t *testing.T) {
	t.Parallel()
	testTime := time.Duration(24760000)
	expectedOutput := int64(24)

	actualOutput := RecvWindow(testTime)
	if actualOutput != expectedOutput {
		t.Errorf("Common RecvWindow. Expected '%d'. Actual '%d'",
			expectedOutput, actualOutput)
	}
}

// TestSplitFloatDecimals ensures SplitFloatDecimals
// accurately splits decimals into integers
func TestSplitFloatDecimals(t *testing.T) {
	x, y, err := SplitFloatDecimals(1.2)
	if err != nil {
		t.Error(err)
	}
	if x != 1 && y != 2 {
		t.Error("Conversion error")
	}
	x, y, err = SplitFloatDecimals(123456.654321)
	if err != nil {
		t.Error(err)
	}
	if x != 123456 && y != 654321 {
		t.Error("Conversion error")
	}
	x, y, err = SplitFloatDecimals(123.111000)
	if err != nil {
		t.Error(err)
	}
	if x != 123 && y != 111 {
		t.Error("Conversion error")
	}
	x, y, err = SplitFloatDecimals(0123.111001)
	if err != nil {
		t.Error(err)
	}
	if x != 123 && y != 111001 {
		t.Error("Conversion error")
	}
	x, y, err = SplitFloatDecimals(1)
	if err != nil {
		t.Error(err)
	}
	if x != 1 && y != 0 {
		t.Error("Conversion error")
	}
	_, _, err = SplitFloatDecimals(float64(math.MaxInt64) + 1)
	if err == nil {
		t.Error("Expected conversion error")
	}
	_, _, err = SplitFloatDecimals(1797693134862315700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000.111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111)
	if err == nil {
		t.Error("Expected conversion error")
	}
	x, y, err = SplitFloatDecimals(-1.2)
	if err != nil {
		t.Error(err)
	}
	if x != -1 && y != -2 {
		t.Error("Conversion error")
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
