package order

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/bsonvalue"
)

var (
	_ bsonwire.ValueMarshaler   = Side(0)
	_ bsonwire.ValueUnmarshaler = (*Side)(nil)
	_ bsonwire.ValueMarshaler   = Type(0)
	_ bsonwire.ValueUnmarshaler = (*Type)(nil)
	_ bsonwire.ValueMarshaler   = Status(0)
	_ bsonwire.ValueUnmarshaler = (*Status)(nil)
	_ bsonwire.ValueMarshaler   = TimeInForce(0)
	_ bsonwire.ValueUnmarshaler = (*TimeInForce)(nil)
)

// flags returns zero followed by every single-bit value from first to last
func flags[T ~uint16 | ~uint32 | ~uint64](first, last T) []T {
	values := []T{0}
	for v := first; v != 0 && v <= last; v <<= 1 {
		values = append(values, v)
	}
	return values
}

func TestSideBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(bsonvalue.Named(flags(Buy, MissingData)...), bsonvalue.Case[Side]{In: MissingData << 1, Want: UnknownSide})
	bsonvalue.AssertRoundTrip(t, MissingData<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, MissingData<<2, Buy, "SIDEWAYS", ErrSideIsInvalid)
}

func TestTypeBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(
		bsonvalue.Named(append(flags(Limit, MarketMakerProtection), StopLimit, StopMarket, TakeProfitMarket, TrailingStopLimit, Bracket)...),
		bsonvalue.Case[Type]{In: Limit | Market, Want: UnknownType},
		bsonvalue.Case[Type]{In: MarketMakerProtection << 1, Want: UnknownType},
	)
	bsonvalue.AssertRoundTrip(t, MarketMakerProtection<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, MarketMakerProtection<<2, Limit, "SIDEWAYS", ErrUnrecognisedOrderType)
}

func TestStatusBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(bsonvalue.Named(flags(AnyStatus, STP)...), bsonvalue.Case[Status]{In: STP << 1, Want: UnknownStatus})
	bsonvalue.AssertRoundTrip(t, STP<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, STP<<2, Filled, "SIDEWAYS", errUnrecognisedOrderStatus)
}

func TestTimeInForceBSONValue(t *testing.T) {
	t.Parallel()
	cases := append(
		bsonvalue.Named(append(flags(GoodTillCancel, StopOrReduce), GoodTillCancel|PostOnly, GoodTillCrossing|PostOnly, supportedTimeInForceFlag)...),
		bsonvalue.Case[TimeInForce]{In: StopOrReduce << 1, Want: UnknownTIF},
		bsonvalue.Case[TimeInForce]{In: GoodTillCancel | StopOrReduce<<1, Want: GoodTillCancel},
	)
	bsonvalue.AssertRoundTrip(t, StopOrReduce<<2, cases)
	bsonvalue.AssertUnmarshalErrors(t, StopOrReduce<<2, GoodTillCancel, "GTC,SIDEWAYS", ErrInvalidTimeInForce)
}
