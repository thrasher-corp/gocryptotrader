package htx

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestLegacyOrderPriceTypeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code uint64
		name string
	}{
		{1, "limit"},
		{2, "market"},
		{3, "opponent"},
		{4, "lightning"},
		{5, "trigger"},
		{6, "post_only"},
		{7, "optimal_5"},
		{8, "optimal_10"},
		{9, "optimal_20"},
		{10, "fok"},
		{11, "ioc"},
		{12, "opponent_ioc"},
		{13, "lightning_ioc"},
		{14, "optimal_5_ioc"},
		{15, "optimal_10_ioc"},
		{16, "optimal_20_ioc"},
		{17, "opponent_fok"},
		{18, "lightning_fok"},
		{19, "optimal_5_fok"},
		{40, "optimal_10_fok"},
		{41, "optimal_20_fok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, payload := range []string{strconv.FormatUint(tc.code, 10), strconv.Quote(tc.name)} {
				var priceType LegacyOrderPriceType
				require.NoError(t, json.Unmarshal([]byte(payload), &priceType), "documented order price type must decode")
				assert.Equal(t, LegacyOrderPriceType(tc.name), priceType, "order price type should be normalised")
			}
		})
	}
	for _, payload := range []string{"0", "20", "42", "-1", "1.5", "true", "{}", "[]", "null", "", `"limit`} {
		t.Run("invalid "+payload, func(t *testing.T) {
			t.Parallel()
			priceType := LegacyOrderPriceType("limit")
			require.ErrorIs(t, priceType.UnmarshalJSON([]byte(payload)), errInvalidOrderPriceType, "invalid order price type must return its sentinel")
			assert.Equal(t, LegacyOrderPriceType("limit"), priceType, "failed decode should preserve the previous value")
		})
	}
}
