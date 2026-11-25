package types

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		input    string
		want     time.Time
		expError error
	}{
		{"null", time.Time{}, nil},
		{"0", time.Time{}, nil},
		{`""`, time.Time{}, nil},
		{`"0"`, time.Time{}, nil},
		{`"0.0"`, time.Time{}, nil},
		{`"0.00000"`, time.Time{}, nil},
		{`"0.0.0.0"`, time.Time{}, strconv.ErrSyntax},
		{`"0.1"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"20200325"`, time.Date(2020, 3, 25, 0, 0, 0, 0, time.UTC), nil},
		{"1628736847", time.Unix(1628736847, 0), nil},
		{`"1628736847"`, time.Unix(1628736847, 0), nil},
		{`"-123456789.5"`, time.UnixMilli(-123456789500), nil},
		{`"-1700000000123"`, time.UnixMilli(-1700000000123), nil},
		{`"-16287368473"`, time.UnixMilli(-16287368473), nil},
		{`"-16287368473251"`, time.UnixMicro(-16287368473251), nil},
		{`"-1628736847325123"`, time.UnixMicro(-1628736847325123), nil},
		{`"-1606292218213457800"`, time.Unix(0, -1606292218213457800), nil},
		{`"-9999999"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-10000000"`, time.Unix(-10000000, 0), nil},
		{`"-123456789"`, time.Unix(-123456789, 0), nil},
		{"-1628736847", time.Unix(-1628736847, 0), nil},
		{`"-1628736847"`, time.Unix(-1628736847, 0), nil},
		{`"-10000000000"`, time.UnixMilli(-10000000000), nil},
		{`"-315619200000"`, time.UnixMilli(-315619200000), nil},
		{`"-315619200000000"`, time.UnixMicro(-315619200000000), nil},
		{`"-10000000000000000"`, time.Unix(0, -10000000000000000), nil},
		{`"-922337203685477581"`, time.Unix(0, -922337203685477581), nil},
		{`"-12345678901234567890"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1628736847.0"`, time.Unix(-1628736847, 0), nil},
		{"-1628736847.5", time.UnixMilli(-1628736847500), nil},
		{`"-1628736847.5"`, time.UnixMilli(-1628736847500), nil},
		{`"-999999999.123456789"`, time.Unix(-999999999, -123456789), nil},
		{`"-999999999.1234567891"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1700000000123.456789"`, time.Unix(0, -1700000000123456789), nil},
		{`"-1700000000123.4567891"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1628736847325123.456"`, time.Unix(0, -1628736847325123456), nil},
		{`"-1628736847325123.4567"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1606292218213457800.5"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1628736847.-5"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"--1628736847"`, time.Time{}, strconv.ErrSyntax},
		{`"-+1628736847"`, time.Time{}, strconv.ErrSyntax},
		{`"-9999999999"`, time.Unix(-9999999999, 0), nil},
		{`"-9223372036854775808"`, time.Unix(0, -9223372036854775808), nil},
		{`"-9223372036854775809"`, time.Time{}, strconv.ErrRange},
		{`"-1699999999999999999"`, time.Unix(0, -1699999999999999999), nil},
		{"-1606292218213457800", time.Unix(0, -1606292218213457800), nil},
		{`"-01234567"`, time.Unix(-1234567, 0), nil},
		{`"-10000000.5"`, time.UnixMilli(-10000000500), nil},
		{`"-10000000.50"`, time.UnixMilli(-10000000500), nil},
		{`"-10000000.08"`, time.UnixMilli(-10000000080), nil},
		{`"-10000000.9"`, time.UnixMilli(-10000000900), nil},
		{`"-10000000000.5"`, time.UnixMicro(-10000000000500), nil},
		{`"-315619200000.12"`, time.UnixMicro(-315619200000120), nil},
		{`"-10000000000000.5"`, time.Unix(0, -10000000000000500), nil},
		{`"-0"`, time.Time{}, nil},
		{`"-0.00000000"`, time.Time{}, nil},
		{`"-00000000000000000000.0"`, time.Time{}, nil},
		{"-0.00000000", time.Time{}, nil},
		{`"-10000000.0"`, time.Unix(-10000000, 0), nil},
		{`"-00000000.5"`, time.UnixMilli(-500), nil},
		{`"-9999999999.5"`, time.UnixMilli(-9999999999500), nil},
		{`"-10000000.000000015"`, time.Unix(-10000000, -15), nil},
		{`"-9999999999999.5"`, time.UnixMicro(-9999999999999500), nil},
		{`"-9007199254740993"`, time.UnixMicro(-9007199254740993), nil},
		{"-1700000000123.456789", time.Unix(0, -1700000000123456789), nil},
		{`"-10000000.1234"`, time.Unix(-10000000, -123400000), nil},
		{`"-0.1"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-1.0"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"-"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"1726104395.5"`, time.UnixMilli(1726104395500), nil},
		{`"1726104395.56"`, time.UnixMilli(1726104395560), nil},
		{`"16287368473"`, time.UnixMilli(1628736847300), nil},
		{`"162873684732"`, time.UnixMilli(1628736847320), nil},
		{`"1628736847325"`, time.UnixMilli(1628736847325), nil},
		{`"16287368473251"`, time.UnixMicro(1628736847325100), nil},
		{`"162873684732512"`, time.UnixMicro(1628736847325120), nil},
		{`"1628736847325123"`, time.UnixMicro(1628736847325123), nil},
		{`"1726106210903.0"`, time.UnixMicro(1726106210903000), nil},
		{`"1747278712.09328"`, time.UnixMicro(1747278712093280), nil},
		{`"16062922182134578"`, time.Unix(0, 1606292218213457800), nil},
		{`"160629221821345780"`, time.Unix(0, 1606292218213457800), nil},
		{`"1606292218213.4578"`, time.Unix(0, 1606292218213457800), nil},
		{`"1560516023.070651"`, time.Unix(0, 1560516023070651000), nil},
		{`"1606292218213457800"`, time.Unix(0, 1606292218213457800), nil},
		{`"00000000000000000000.0"`, time.Time{}, nil},
		{`"12345678901234567890.1"`, time.Time{}, strconv.ErrRange},
		{`"blurp"`, time.Time{}, strconv.ErrSyntax},
		{`"123456"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"12345678"`, time.Time{}, ErrInvalidTimestampFormat},
		{`"2025-03-28T08:00:00Z"`, time.Time{}, strconv.ErrSyntax}, // RFC3339 format (currently unsupported)
		{`"1606292218213.45.8"`, time.Time{}, strconv.ErrSyntax},   // parse int failure
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			var testTime Time
			err := testTime.UnmarshalJSON([]byte(tc.input))
			require.ErrorIsf(t, err, tc.expError, "UnmarshalJSON must return the expected error for input %q", tc.input)
			assert.Equalf(t, tc.want, testTime.Time(), "UnmarshalJSON should set Time correctly for input %q", tc.input)

			var decoded Time
			err = json.Unmarshal([]byte(tc.input), &decoded)
			require.ErrorIsf(t, err, tc.expError, "json.Unmarshal must return the expected error for input %q", tc.input)
			assert.Equalf(t, tc.want, decoded.Time(), "json.Unmarshal should set Time correctly for input %q", tc.input)
		})
	}
}

func TestTimeUnmarshalJSONEmptyInput(t *testing.T) {
	t.Parallel()

	initial := Time(time.Date(2020, 3, 25, 0, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name  string
		input []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testTime := initial
			err := testTime.UnmarshalJSON(tc.input)
			require.NoErrorf(t, err, "Time.UnmarshalJSON must not error for %s input", tc.name)
			assert.Equalf(t, initial, testTime, "Time.UnmarshalJSON should preserve Time for %s input", tc.name)
		})
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	for _, tc := range []struct {
		name  string
		input string
		want  time.Time
	}{
		{"seconds", `"1628736847"`, time.Unix(1628736847, 0)},
		{"padded_width", `"16287368473"`, time.UnixMilli(1628736847300)},
		{"fractional", `"1691122380942.173000"`, time.Unix(0, 1691122380942173000)},
		{"date", `"20200325"`, time.Date(2020, 3, 25, 0, 0, 0, 0, time.UTC)},
	} {
		data := []byte(tc.input)
		b.Run("direct/"+tc.name, func(b *testing.B) {
			b.StopTimer()
			var testTime Time
			if err := testTime.UnmarshalJSON(data); err != nil {
				b.Fatal(err)
			}
			if got := testTime.Time(); !got.Equal(tc.want) {
				b.Fatalf("UnmarshalJSON returned %v, want %v", got, tc.want)
			}
			b.ReportAllocs()
			b.StartTimer()
			for b.Loop() {
				if err := testTime.UnmarshalJSON(data); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("decoder/"+tc.name, func(b *testing.B) {
			b.StopTimer()
			var testTime Time
			if err := json.Unmarshal(data, &testTime); err != nil {
				b.Fatal(err)
			}
			if got := testTime.Time(); !got.Equal(tc.want) {
				b.Fatalf("json.Unmarshal returned %v, want %v", got, tc.want)
			}
			b.ReportAllocs()
			b.StartTimer()
			for b.Loop() {
				if err := json.Unmarshal(data, &testTime); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestTime(t *testing.T) {
	t.Parallel()
	testTime := Time(time.Time{})
	assert.Equal(t, time.Time{}, testTime.Time())
	assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", testTime.String())
}

func TestTime_MarshalJSON(t *testing.T) {
	t.Parallel()
	testTime := Time(time.Time{})
	data, err := testTime.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"0001-01-01T00:00:00Z"`, string(data))
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var (
		testTime   DateTime
		jsonError  *json.UnmarshalTypeError
		parseError *time.ParseError
	)
	err := json.Unmarshal([]byte(`69`), &testTime)
	if json.Implementation == "bytedance/sonic" {
		require.ErrorContains(t, err, "Mismatch type string with value number", "Unmarshal must return the correct error text for sonic")
	} else {
		require.ErrorAs(t, err, &jsonError, "Unmarshal must return the correct error type for Go standard encoding/json")
	}
	require.ErrorAs(t, json.Unmarshal([]byte(`"2025"`), &testTime), &parseError)
	require.NoError(t, json.Unmarshal([]byte(`"2018-08-20 19:20:46"`), &testTime))
	assert.Equal(t, time.Date(2018, 8, 20, 19, 20, 46, 0, time.UTC), testTime.Time())
}
