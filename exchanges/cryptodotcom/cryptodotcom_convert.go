package cryptodotcom

import (
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// SafeNumber represents a number instance with 0 as a default value for null and empty string values.
type SafeNumber float64

// UnmarshalJSON decerializes a []byte instance into safe tensor.
func (a *SafeNumber) UnmarshalJSON(data []byte) error { //nolint:all
	var resp types.Number
	err := json.Unmarshal(data, &resp)
	if err != nil {
		*a = SafeNumber(0)
		return nil //nolint:all
	}
	*a = SafeNumber(resp)
	return nil
}

// Float64 returns a float64 value
func (a SafeNumber) Float64() float64 {
	return float64(a)
}

// Int64 returns the truncated integer component of the number
func (a SafeNumber) Int64() int64 {
	// It's likely this is sufficient, since Numbers probably have not had floating point math performed on them
	// However if issues arise then we can switch to math.Round
	return int64(a)
}

// Decimal returns a decimal.Decimal
func (a SafeNumber) Decimal() decimal.Decimal {
	return decimal.NewFromFloat(float64(a))
}

// String returns a string representation of the number
func (a SafeNumber) String() string {
	return strconv.FormatFloat(float64(a), 'f', -1, 64)
}
