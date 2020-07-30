package convert

import (
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

func TestTimeFromUnixTimestampDecimal(t *testing.T) {
	r := TimeFromUnixTimestampDecimal(1590633982.5714)
	if r.Year() != 2020 ||
		r.Month().String() != "May" ||
		r.Day() != 28 {
		t.Error("unexpected result")
	}

	r = TimeFromUnixTimestampDecimal(1560516023.070651)
	if r.Year() != 2019 ||
		r.Month().String() != "June" ||
		r.Day() != 14 {
		t.Error("unexpected result")
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

func TestUnixMillisToNano(t *testing.T) {
	v := UnixMillisToNano(1588653603424)
	if v != 1588653603424000000 {
		t.Fatalf("unexpected result received %v", v)
	}
}
