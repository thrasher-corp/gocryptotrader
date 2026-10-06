package convert

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestFloatToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0.000", FloatToHumanFriendlyString(0, 3, ".", ","))
	assert.Equal(t, "100.000", FloatToHumanFriendlyString(100, 3, ".", ","))
	assert.Equal(t, "1,000.000", FloatToHumanFriendlyString(1000, 3, ".", ","))
	assert.Equal(t, "-1,000.000", FloatToHumanFriendlyString(-1000, 3, ".", ","))
	assert.Equal(t, "-1,000.0000000000", FloatToHumanFriendlyString(-1000, 10, ".", ","))
	assert.Equal(t, "1!000.1", FloatToHumanFriendlyString(1000.1337, 1, ".", "!"))
	assert.Equal(t, "1234567", FloatToHumanFriendlyString(1234567, 0, ".", ""), "empty separator should be omitted")
	assert.Equal(t, "0.00", FloatToHumanFriendlyString(math.Copysign(0, -1), 2, ".", ","), "negative zero should not carry a sign")
}

func TestDecimalToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", DecimalToHumanFriendlyString(decimal.Zero, 0, ".", ","))
	assert.Equal(t, "100", DecimalToHumanFriendlyString(decimal.NewFromInt(100), 0, ".", ","))
	assert.Equal(t, "1,000", DecimalToHumanFriendlyString(decimal.NewFromInt(1000), 0, ".", ","))
	assert.Equal(t, "-1,000", DecimalToHumanFriendlyString(decimal.NewFromInt(-1000), 0, ".", ","))
	assert.Equal(t, "-1~000!42", DecimalToHumanFriendlyString(decimal.MustFromFloat(-1000.42069), 2, "!", "~"))
	assert.Equal(t, "1,000.42069", DecimalToHumanFriendlyString(decimal.MustFromFloat(1000.42069), 5, ".", ","))
	assert.Equal(t, "1,000.42069", DecimalToHumanFriendlyString(decimal.MustFromFloat(1000.42069), 100, ".", ","), "rounding should clamp to the available decimal places")
}

func TestIntToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", IntToHumanFriendlyString(0, ","))
	assert.Equal(t, "100", IntToHumanFriendlyString(100, ","))
	assert.Equal(t, "1,000", IntToHumanFriendlyString(1000, ","))
	assert.Equal(t, "-1,000", IntToHumanFriendlyString(-1000, ","))
	assert.Equal(t, "-1!000", IntToHumanFriendlyString(-1000, "!"))
	assert.Equal(t, "9,223,372,036,854,775,807", IntToHumanFriendlyString(math.MaxInt64, ","))
	assert.Equal(t, "-9,223,372,036,854,775,808", IntToHumanFriendlyString(math.MinInt64, ","), "the most negative int64 should not overflow when negated")
}

func TestNumberToHumanFriendlyString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", numberToHumanFriendlyString("0", 0, "", ",", false))
	assert.Equal(t, "1,337.69", numberToHumanFriendlyString("1337.69", 2, ".", ",", false))
	assert.Equal(t, "-1!000.1", numberToHumanFriendlyString("1000.1", 1, ".", "!", true))
	assert.Equal(t, "1,000", numberToHumanFriendlyString("1000", 20, ".", ",", false), "decimals longer than the input should be dropped")
	assert.Empty(t, numberToHumanFriendlyString("", 0, ".", ",", false))
	assert.Equal(t, "1~~234~~567", numberToHumanFriendlyString("1234567", 0, ".", "~~", false), "multi-byte separators should be preserved")
}
