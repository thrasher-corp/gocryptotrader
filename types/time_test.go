package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTime(t *testing.T) {
	t.Parallel()
	var testTime Time

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
}

// 5046307	       216.0 ns/op	     168 B/op	       2 allocs/op (current)
// 2716176	       441.9 ns/op	     352 B/op	       6 allocs/op (previous)
func BenchmarkTime(b *testing.B) {
	var testTime Time
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal([]byte(`"1691122380942.173000"`), &testTime)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestExchangeTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	unmarshaledResult := &struct {
		Timestamp Time `json:"ts"`
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
