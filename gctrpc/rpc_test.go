package gctrpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRetypedTimestampWireNumbers(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		message   proto.Message
		fieldName protoreflect.Name
		oldNumber protoreflect.FieldNumber
		newNumber protoreflect.FieldNumber
	}{
		{"ticker", new(TickerResponse), "last_updated", 2, 31},
		{"orderbook", new(OrderbookResponse), "last_updated", 5, 8},
		{"trade", new(TradeHistory), "creation_time", 1, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reflection := tc.message.ProtoReflect()
			descriptor := reflection.Descriptor()
			field := descriptor.Fields().ByName(tc.fieldName)
			require.NotNil(t, field, "timestamp field must exist")
			assert.Equal(t, tc.newNumber, field.Number(), "timestamp field should use a fresh wire number")
			assert.True(t, descriptor.ReservedRanges().Has(tc.oldNumber), "old timestamp wire number should be reserved")

			legacy := protowire.AppendTag(nil, tc.oldNumber, protowire.VarintType)
			legacy = protowire.AppendVarint(legacy, 123)
			require.NoError(t, proto.Unmarshal(legacy, tc.message), "legacy timestamp must not fail decoding")
			assert.False(t, reflection.Has(field), "legacy timestamp should not populate the retyped field")

			proto.Reset(tc.message)
			reflection.Set(field, protoreflect.ValueOfMessage(timestamppb.New(time.Unix(123, 456)).ProtoReflect()))
			encoded, err := proto.Marshal(tc.message)
			require.NoError(t, err, "new timestamp must marshal")
			number, wireType, consumed := protowire.ConsumeTag(encoded)
			require.Positive(t, consumed, "marshalled timestamp must contain a valid tag")
			assert.Equal(t, tc.newNumber, number, "new timestamp should use its new wire number")
			assert.Equal(t, protowire.BytesType, wireType, "new timestamp should use message wire encoding")
		})
	}
}
