package margin

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
)

var (
	_ bsonwire.ValueMarshaler   = Type(0)
	_ bsonwire.ValueUnmarshaler = (*Type)(nil)
)

func TestTypeBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(
		bsonvalue.Named(Unset, Isolated, Multi, NoMargin, SpotIsolated, Unknown),
		bsonvalue.Case[Type]{In: Unknown << 1, Want: Unknown},
	)
	bsonvalue.AssertRoundTrip(t, Unknown<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, Unknown<<2, Isolated, "moon", ErrInvalidMarginType)
}
