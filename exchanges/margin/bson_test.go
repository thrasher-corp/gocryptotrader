package margin

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	_ bson.ValueMarshaler   = Type(0)
	_ bson.ValueUnmarshaler = (*Type)(nil)
)

func TestTypeBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(
		bsonvalue.Named(Unset, Isolated, Multi, NoMargin, SpotIsolated, Unknown),
		bsonvalue.Case[Type]{In: Unknown << 1, Want: Unknown},
	)
	bsonvalue.AssertRoundTrip(t, Unknown<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, Isolated, "moon", ErrInvalidMarginType)
}
