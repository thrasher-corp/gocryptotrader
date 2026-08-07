package mexc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// TestProtoTypesAreUsable asserts every generated MEXC message can be handled by the protobuf
// runtime. The runtime resolves a message's fields lazily, on first use, and panics when a struct
// field's Go type contradicts its descriptor. Editing the generated code by hand — swapping a
// descriptor's string for a float64 wrapper, or its int64 for a string — therefore compiles, links
// and passes every test that never touches that message, and only fails at runtime on the live
// feed. Three of these messages carried such an edit and none of them had a test: the defect was
// invisible until a real frame arrived.
func TestProtoTypesAreUsable(t *testing.T) {
	t.Parallel()
	var checked int
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if !strings.HasSuffix(fd.Path(), "V3Api.proto") && !strings.HasSuffix(fd.Path(), "V3ApiWrapper.proto") {
			return true
		}
		messages := fd.Messages()
		for i := range messages.Len() {
			name := messages.Get(i).FullName()
			mt, err := protoregistry.GlobalTypes.FindMessageByName(name)
			require.NoErrorf(t, err, "%s must be registered", name)
			checked++
			assert.NotPanicsf(t, func() {
				_, err := proto.Marshal(mt.New().Interface())
				assert.NoErrorf(t, err, "%s must marshal", name)
			}, "%s must not panic: its Go field types must match its descriptor", name)
		}
		return true
	})
	assert.GreaterOrEqual(t, checked, 20, "the generated MEXC messages must actually be discovered; a scan over nothing proves nothing")
}
