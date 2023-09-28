package convert

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
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
	test := FloatToHumanFriendlyString(0, 3, ".", ",")
	if strings.Contains(test, ",") {
		t.Error("unexpected ','")
	}
	test = FloatToHumanFriendlyString(100, 3, ".", ",")
	if strings.Contains(test, ",") {
		t.Error("unexpected ','")
	}
	test = FloatToHumanFriendlyString(1000, 3, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = FloatToHumanFriendlyString(-1000, 3, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = FloatToHumanFriendlyString(-1000, 10, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = FloatToHumanFriendlyString(1000.1337, 1, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}
	dec := strings.Split(test, ".")
	if len(dec) == 1 {
		t.Error("expected decimal place")
	}
	if dec[1] != "1" {
		t.Error("expected decimal place")
	}
}

func TestDecimalToHumanFriendlyString(t *testing.T) {
	t.Parallel()
	test := DecimalToHumanFriendlyString(decimal.Zero, 0, ".", ",")
	if strings.Contains(test, ",") {
		t.Log(test)
		t.Error("unexpected ','")
	}
	test = DecimalToHumanFriendlyString(decimal.NewFromInt(100), 0, ".", ",")
	if strings.Contains(test, ",") {
		t.Log(test)
		t.Error("unexpected ','")
	}
	test = DecimalToHumanFriendlyString(decimal.NewFromInt(1000), 0, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = DecimalToHumanFriendlyString(decimal.NewFromFloat(1000.1337), 1, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}
	dec := strings.Split(test, ".")
	if len(dec) == 1 {
		t.Error("expected decimal place")
	}
	if dec[1] != "1" {
		t.Error("expected decimal place")
	}

	test = DecimalToHumanFriendlyString(decimal.NewFromFloat(-1000.1337), 1, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = DecimalToHumanFriendlyString(decimal.NewFromFloat(-1000.1337), 100000, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = DecimalToHumanFriendlyString(decimal.NewFromFloat(1000.1), 10, ".", ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}
	dec = strings.Split(test, ".")
	if len(dec) == 1 {
		t.Error("expected decimal place")
	}
	if dec[1] != "1" {
		t.Error("expected decimal place")
	}
}

func TestIntToHumanFriendlyString(t *testing.T) {
	t.Parallel()
	test := IntToHumanFriendlyString(0, ",")
	if strings.Contains(test, ",") {
		t.Log(test)
		t.Error("unexpected ','")
	}
	test = IntToHumanFriendlyString(100, ",")
	if strings.Contains(test, ",") {
		t.Log(test)
		t.Error("unexpected ','")
	}
	test = IntToHumanFriendlyString(1000, ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = IntToHumanFriendlyString(-1000, ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}

	test = IntToHumanFriendlyString(1000000, ",")
	if !strings.Contains(test, ",") {
		t.Error("expected ','")
	}
	dec := strings.Split(test, ",")
	if len(dec) <= 2 {
		t.Error("expected two commas place")
	}
}

func TestNumberToHumanFriendlyString(t *testing.T) {
	resp := numberToHumanFriendlyString("1", 1337, ".", ",", false)
	if strings.Contains(resp, ".") {
		t.Error("expected no comma")
	}
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

func TestStringToFloat64(t *testing.T) {
	t.Parallel()
	resp := struct {
		Data StringToFloat64 `json:"data"`
	}{}

	err := json.Unmarshal([]byte(`{"data":"0.00000001"}`), &resp)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Data.Float64() != 1e-8 {
		t.Fatalf("expected 1e-8, got %v", resp.Data.Float64())
	}

	err = json.Unmarshal([]byte(`{"data":""}`), &resp)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal([]byte(`{"data":1337.37}`), &resp)
	if !errors.Is(err, errUnhandledType) {
		t.Fatalf("received %v but expected %v", err, errUnhandledType)
	}

	// Demonstrates that a suffix check is not needed.
	err = json.Unmarshal([]byte(`{"data":"1337.37}`), &resp)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = json.Unmarshal([]byte(`{"data":"MEOW"}`), &resp)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	data, err := json.Marshal(StringToFloat64(0))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `""` {
		t.Fatalf("expected empty string, got %v", string(data))
	}

	data, err = json.Marshal(StringToFloat64(1337.1337))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `"1337.1337"` {
		t.Fatalf("expected \"1337.1337\" string, got %v", string(data))
	}
}

func TestStringToFloat64Decimal(t *testing.T) {
	t.Parallel()
	resp := struct {
		Data StringToFloat64 `json:"data"`
	}{}
	err := json.Unmarshal([]byte(`{"data":"0.00000001"}`), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Data.Decimal().Equal(decimal.NewFromFloat(0.00000001)) {
		t.Errorf("received '%v' expected '%v'", resp.Data.Decimal(), 0.00000001)
	}
}

// 2677173	       428.9 ns/op	     240 B/op	       5 allocs/op
func BenchmarkStringToFloat64(b *testing.B) {
	resp := struct {
		Data StringToFloat64 `json:"data"`
	}{}

	for i := 0; i < b.N; i++ {
		err := json.Unmarshal([]byte(`{"data":"0.00000001"}`), &resp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestExchangeTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	unmarshaledResult := &struct {
		Timestamp ExchangeTime `json:"ts"`
	}{}
	data1 := `{"ts":""}`
	result := time.Time{}
	err := json.Unmarshal([]byte(data1), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data2 := `{"ts":"1685564775371"}`
	result = time.UnixMilli(1685564775371)
	err = json.Unmarshal([]byte(data2), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data3 := `{"ts":1685564775371}`
	err = json.Unmarshal([]byte(data3), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data4 := `{"ts":"1685564775"}`
	result = time.Unix(1685564775, 0)
	err = json.Unmarshal([]byte(data4), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data5 := `{"ts":1685564775}`
	err = json.Unmarshal([]byte(data5), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data6 := `{"ts":"1685564775371320000"}`
	result = time.Unix(int64(1685564775371320000)/1e9, int64(1685564775371320000)%1e9)
	err = json.Unmarshal([]byte(data6), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
	data7 := `{"ts":"abcdefg"}`
	err = json.Unmarshal([]byte(data7), &unmarshaledResult)
	if err == nil {
		t.Fatal("expecting error but found nil")
	}
	data8 := `{"ts":0}`
	result = time.Time{}
	err = json.Unmarshal([]byte(data8), &unmarshaledResult)
	if err != nil {
		t.Fatal(err)
	} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
		t.Errorf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
	}
}

// 2239239               516.1 ns/op           424 B/op          9 allocs/op
func BenchmarkExchangeTimeUnmarshaling(b *testing.B) {
	unmarshaledResult := &struct {
		Timestamp ExchangeTime `json:"ts"`
	}{}
	data5 := `{"ts":1685564775}`
	result := time.Unix(1685564775, 0)
	var err error
	for i := 0; i < b.N; i++ {
		if err = json.Unmarshal([]byte(data5), &unmarshaledResult); err != nil {
			b.Fatal(err)
		} else if !unmarshaledResult.Timestamp.Time().Equal(result) {
			b.Fatalf("found %v, but expected %v", unmarshaledResult.Timestamp.Time(), result)
		}
	}
}
