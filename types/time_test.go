package types

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var testTime Time

	require.NoError(t, json.Unmarshal([]byte(`null`), &testTime))
	assert.Equal(t, time.Time{}, testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`0`), &testTime))
	assert.Equal(t, time.Time{}, testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`""`), &testTime))
	assert.Equal(t, time.Time{}, testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`"0"`), &testTime))
	assert.Equal(t, time.Time{}, testTime.Time())

	// seconds
	require.NoError(t, json.Unmarshal([]byte(`"1628736847"`), &testTime))
	assert.Equal(t, time.Unix(1628736847, 0), testTime.Time())

	// milliseconds
	require.NoError(t, json.Unmarshal([]byte(`"1726104395.5"`), &testTime))
	assert.Equal(t, time.UnixMilli(1726104395500), testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`"1726104395.56"`), &testTime))
	assert.Equal(t, time.UnixMilli(1726104395560), testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`"1628736847325"`), &testTime))
	assert.Equal(t, time.UnixMilli(1628736847325), testTime.Time())

	// microseconds
	require.NoError(t, json.Unmarshal([]byte(`"1628736847325123"`), &testTime))
	assert.Equal(t, time.UnixMicro(1628736847325123), testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`"1726106210903.0"`), &testTime))
	assert.Equal(t, time.UnixMicro(1726106210903000), testTime.Time())

	// nanoseconds
	require.NoError(t, json.Unmarshal([]byte(`"1606292218213.4578"`), &testTime))
	assert.Equal(t, time.Unix(0, 1606292218213457800), testTime.Time())

	require.NoError(t, json.Unmarshal([]byte(`"1606292218213457800"`), &testTime))
	assert.Equal(t, time.Unix(0, 1606292218213457800), testTime.Time())

	require.ErrorIs(t, json.Unmarshal([]byte(`"blurp"`), &testTime), strconv.ErrSyntax)
	require.Error(t, json.Unmarshal([]byte(`"123456"`), &testTime))

	// Captures bad syntax when type should be time.Time (RFC3339)
	require.ErrorIs(t, json.Unmarshal([]byte(`"2025-03-28T08:00:00Z"`), &testTime), strconv.ErrSyntax)
	// parse int failure
	require.ErrorIs(t, json.Unmarshal([]byte(`"1606292218213.45.8"`), &testTime), strconv.ErrSyntax)
}

// 5030734	       240.1 ns/op	     168 B/op	       2 allocs/op (current)
// 2716176	       441.9 ns/op	     352 B/op	       6 allocs/op (previous)
func BenchmarkUnmarshalJSON(b *testing.B) {
	var testTime Time
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal([]byte(`"1691122380942.173000"`), &testTime)
		require.NoError(b, err)
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
