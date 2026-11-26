package asset

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
)

var (
	_ bsonwire.ValueMarshaler   = Item(0)
	_ bsonwire.ValueUnmarshaler = (*Item)(nil)
)

func TestItemBSONValue(t *testing.T) {
	t.Parallel()
	items := make([]Item, 0, All+1)
	for a := Empty; a <= All; a++ {
		items = append(items, a)
	}
	cases := append(bsonvalue.Named(items...), bsonvalue.Case[Item]{In: All + 1, Want: Empty})
	bsonvalue.AssertRoundTrip(t, All+2, cases)
	bsonvalue.AssertUnmarshalErrors(t, All+2, Spot, "moon", ErrNotSupported)
}
