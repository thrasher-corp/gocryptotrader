package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestBooleanUnmarshal(t *testing.T) {
	t.Parallel()
	data := []byte(`{"value": true, "another_value": "true", "third_value": "false", "fourth_value": 1, "fifth_value": 0, "sixth_value": "1", "seventh_value": "0"}`)
	var result map[string]Boolean
	err := json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.True(t, result["value"].Bool())
	assert.True(t, result["another_value"].Bool())
	assert.False(t, result["third_value"].Bool())
	assert.True(t, result["fourth_value"].Bool())
	assert.False(t, result["fifth_value"].Bool())
	assert.True(t, result["sixth_value"].Bool())
	assert.False(t, result["seventh_value"].Bool())

	data = []byte(`{"value": "3"}`)
	err = json.Unmarshal(data, &result)
	require.ErrorIs(t, err, errInvalidBooleanValue)
}
