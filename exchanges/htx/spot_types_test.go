package htx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestIsEmptyHTXData(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		data []byte
		want bool
	}
	for _, tt := range []testCase{
		{name: "empty", want: true},
		{name: "empty string", data: []byte(`""`), want: true},
		{name: "null", data: []byte(` null `), want: true},
		{name: "array", data: []byte(`[]`)},
		{name: "object", data: []byte(`{}`)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isEmptyHTXData(tt.data), "isEmptyHTXData should identify empty HTX data values")
		})
	}
}

func TestOrderInfoJSON(t *testing.T) {
	t.Parallel()
	var info OrderInfo
	require.NoError(t, json.Unmarshal([]byte(`{"id":9007199254740993,"account-id":123,"user-id":456,"created-at":1511169724531,"canceled-at":1511169725531}`), &info), "order response must decode")
	assert.Equal(t, uint64(9007199254740993), info.ID, "order ID should retain its exact value")
	assert.Equal(t, time.UnixMilli(1511169724531), info.CreatedAt.Time(), "creation time should retain millisecond precision")
	assert.Equal(t, time.UnixMilli(1511169725531), info.CanceledAt.Time(), "cancellation time should retain millisecond precision")
}

func TestMarginOrderJSON(t *testing.T) {
	t.Parallel()
	var info MarginOrder
	require.NoError(t, json.Unmarshal([]byte(`{"id":9007199254740993,"account-id":123,"user-id":456,"created-at":1511169724531,"accrued-at":1511169725531,"updated-at":1511169726531}`), &info), "margin loan response must decode")
	assert.Equal(t, uint64(9007199254740993), info.ID, "margin loan ID should retain its exact value")
	assert.Equal(t, time.UnixMilli(1511169724531), info.CreatedAt.Time(), "creation time should retain millisecond precision")
	assert.Equal(t, time.UnixMilli(1511169725531), info.AccruedAt.Time(), "accrual time should retain millisecond precision")
	assert.Equal(t, time.UnixMilli(1511169726531), info.UpdatedAt.Time(), "update time should retain millisecond precision")
}

func TestCurrencyChainsJSON(t *testing.T) {
	t.Parallel()
	var info CurrencyChains
	require.NoError(t, json.Unmarshal([]byte(`{"currency":"usdt","chains":[{"chain":"usdterc20","withdrawPrecision":6,"numOfConfirmations":999,"numOfFastConfirmations":999}]}`), &info), "currency chain response must decode")
	require.Len(t, info.Chains, 1, "currency response must contain its chain")
	require.NotNil(t, info.Chains[0], "currency chain must be populated")
	assert.Equal(t, uint64(6), info.Chains[0].WithdrawPrecision, "withdrawal precision should match the API value")
}
