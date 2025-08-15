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
		{`"0.1"`, time.Time{}, errInvalidTimestampFormat},
		{`"1628736847"`, time.Unix(1628736847, 0), nil},
		{`"1726104395.5"`, time.UnixMilli(1726104395500), nil},
		{`"1726104395.56"`, time.UnixMilli(1726104395560), nil},
		{`"1628736847325"`, time.UnixMilli(1628736847325), nil},
		{`"1628736847325123"`, time.UnixMicro(1628736847325123), nil},
		{`"1726106210903.0"`, time.UnixMicro(1726106210903000), nil},
		{`"1747278712.09328"`, time.UnixMicro(1747278712093280), nil},
		{`"1606292218213.4578"`, time.Unix(0, 1606292218213457800), nil},
		{`"1560516023.070651"`, time.Unix(0, 1560516023070651000), nil},
		{`"1606292218213457800"`, time.Unix(0, 1606292218213457800), nil},
		{`"blurp"`, time.Time{}, strconv.ErrSyntax},
		{`"123456"`, time.Time{}, errInvalidTimestampFormat},
		{`"2025-03-28T08:00:00Z"`, time.Time{}, strconv.ErrSyntax}, // RFC3339 format
		{`"1606292218213.45.8"`, time.Time{}, strconv.ErrSyntax},   // parse int failure
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			var testTime Time
			err := json.Unmarshal([]byte(tc.input), &testTime)
			require.ErrorIsf(t, err, tc.expError, "Unmarshal must not error for input %q", tc.input)
			assert.Equal(t, tc.want, testTime.Time())
		})
	}
}

// 3948978         303.5 ns/op       168 B/op          2 allocs/op (current) after more stringent checks
// 6152384         195.5 ns/op       168 B/op          2 allocs/op (previous)
func BenchmarkUnmarshalJSON(b *testing.B) {
	var testTime Time
	for b.Loop() {
		if err := json.Unmarshal([]byte(`"1691122380942.173000"`), &testTime); err != nil {
			b.Fatal(err)
		}
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
