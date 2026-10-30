package asset

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	_ bson.ValueMarshaler   = Item(0)
	_ bson.ValueUnmarshaler = (*Item)(nil)
)

func TestItemBSONValue(t *testing.T) {
	t.Parallel()
	items := make([]Item, 0, All+1)
	for a := Empty; a <= All; a++ {
		items = append(items, a)
	}
	cases := append(bsonvalue.Named(items...), bsonvalue.Case[Item]{In: All + 1, Want: Empty})
	bsonvalue.AssertRoundTrip(t, All+2, cases)
	bsonvalue.AssertUnmarshalErrors(t, Spot, "moon", ErrNotSupported)
}
