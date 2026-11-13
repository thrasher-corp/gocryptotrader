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

func TestTickerResponseReservedPriceATH(t *testing.T) {
	t.Parallel()
	response := new(TickerResponse)
	descriptor := response.ProtoReflect().Descriptor()
	assert.True(t, descriptor.ReservedRanges().Has(10), "former price_ath wire number should remain reserved")
	assert.True(t, descriptor.ReservedNames().Has("price_ath"), "former price_ath name should remain reserved")
	assert.Nil(t, descriptor.Fields().ByNumber(10), "reserved wire number should not identify a live field")
	legacy := protowire.AppendTag(nil, 10, protowire.Fixed64Type)
	legacy = protowire.AppendFixed64(legacy, 123)
	require.NoError(t, proto.Unmarshal(legacy, response), "legacy price_ath field must remain decodable as unknown data")
	assert.False(t, response.ProtoReflect().Has(descriptor.Fields().ByName("last_updated")), "legacy price_ath should not populate the timestamp")
}
