package event

import (
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// GetOffset returns the offset
func (b *Base) GetOffset() int64 {
	return b.Offset
}

// SetOffset sets the offset
func (b *Base) SetOffset(o int64) {
	b.Offset = o
}

// IsEvent returns whether the event is an event
func (b *Base) IsEvent() bool {
	return true
}

// GetTime returns the time
func (b *Base) GetTime() time.Time {
	return b.Time
}

// Pair returns the currency pair
func (b *Base) Pair() currency.Pair {
	return b.CurrencyPair
}

// GetUnderlyingPair returns the currency pair
func (b *Base) GetUnderlyingPair() currency.Pair {
	return b.UnderlyingPair
}

// GetExchange returns the exchange
func (b *Base) GetExchange() string {
	return strings.ToLower(b.Exchange)
}

// GetAssetType returns the asset type
func (b *Base) GetAssetType() asset.Item {
	return b.AssetType
}

// GetInterval returns the interval
func (b *Base) GetInterval() kline.Interval {
	return b.Interval
}

// AppendReason adds reasoning for a decision being made
func (b *Base) AppendReason(y string) {
	b.Reason += y + ". "
	b.Reasons = append(b.Reasons, y)
}

// AppendReasonf adds reasoning for a decision being made
// but with formatting
func (b *Base) AppendReasonf(y string, addons ...interface{}) {
	y = fmt.Sprintf(y, addons...)
	b.Reason += y + ". "
	b.Reasons = append(b.Reasons, y)
}

// GetReason returns the why
func (b *Base) GetReason() string {
	return b.Reason
}

// GetReasons returns each individual reason
func (b *Base) GetReasons() []string {
	return b.Reasons
}

func (b *Base) GetBase() Base {
	return *b
}
