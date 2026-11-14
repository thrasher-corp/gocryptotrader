package engine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// How completely fillRequest populates a request; each level adds to the one before
const (
	fillNothing = iota
	fillStrings
	fillPairsAndTimes
	fillMessages
	fillIntegers
	fillLevels
)

var errNoNetwork = errors.New("network access is disabled in this test")

type noNetworkTransport struct{}

func (noNetworkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errNoNetwork
}

// TestRPCServerIncompleteRequests calls every unary handler with requests which leave fields out
// A handler which reads an optional field before checking it panics here rather than in a running engine
func TestRPCServerIncompleteRequests(t *testing.T) {
	t.Parallel()
	methods := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods()
	for i := range methods.Len() {
		method := methods.Get(i)
		if method.IsStreamingClient() || method.IsStreamingServer() {
			continue
		}
		t.Run(string(method.Name()), func(t *testing.T) {
			t.Parallel()
			messageType, err := protoregistry.GlobalTypes.FindMessageByName(method.Input().FullName())
			require.NoError(t, err, "FindMessageByName must not error")
			s, exchangeName := newIncompleteRequestServer(t)
			handler := reflect.ValueOf(s).MethodByName(string(method.Name()))
			require.True(t, handler.IsValid(), "RPCServer must implement the method")
			for level := range fillLevels {
				request := messageType.New()
				fillRequest(request, exchangeName, level, 0)
				assert.Emptyf(t, callHandler(t, handler, request.Interface()), "handler should not panic at fill level %d", level)
			}
		})
	}
}

// newIncompleteRequestServer returns a server with one exchange which cannot reach the network
func newIncompleteRequestServer(t *testing.T) (s *RPCServer, exchangeName string) {
	t.Helper()
	em := NewExchangeManager()
	b := newFakeExchange(t, em, map[asset.Item]currency.Pairs{asset.Spot: {currency.NewBTCUSD()}})
	require.NoError(t, b.SetHTTPClient(&http.Client{Transport: noNetworkTransport{}}), "SetHTTPClient must not error")
	// LoadExchange always gives a running exchange its config, which handlers that toggle features write to
	b.Config = &config.Exchange{Name: b.Name, Features: &config.FeaturesConfig{}}
	return &RPCServer{Engine: &Engine{Config: &config.Config{}, ExchangeManager: em}}, b.Name
}

// callHandler invokes a unary handler and describes the panic if it had one
func callHandler(t *testing.T, handler reflect.Value, request any) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Sprintf("%v\n%s", r, debug.Stack())
				return
			}
			done <- ""
		}()
		handler.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(request)})
	}()
	select {
	case panicked := <-done:
		return panicked
	case <-time.After(30 * time.Second):
		require.FailNow(t, "handler must return")
		return ""
	}
}

// fillRequest populates the fields of a request which the fill level covers, descending into nested messages
func fillRequest(m protoreflect.Message, exchangeName string, level, depth int) {
	if level == fillNothing {
		return
	}
	end := time.Now().UTC().Truncate(time.Hour).Add(-24 * time.Hour)
	start := end.Add(-24 * time.Hour)
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		name := string(field.Name())
		if field.IsList() || field.IsMap() {
			continue
		}
		switch field.Kind() {
		case protoreflect.StringKind:
			if value, ok := requestString(name, exchangeName, start, end); ok {
				m.Set(field, protoreflect.ValueOfString(value))
			}
		case protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.Int32Kind, protoreflect.Uint32Kind:
			if level < fillIntegers {
				continue
			}
			value := int64(14)
			if strings.Contains(name, "interval") {
				value = int64(time.Hour)
			}
			switch field.Kind() {
			case protoreflect.Int64Kind:
				m.Set(field, protoreflect.ValueOfInt64(value))
			case protoreflect.Uint64Kind:
				m.Set(field, protoreflect.ValueOfUint64(uint64(value)))
			case protoreflect.Int32Kind:
				m.Set(field, protoreflect.ValueOfInt32(14))
			default:
				m.Set(field, protoreflect.ValueOfUint32(14))
			}
		case protoreflect.MessageKind:
			if level < fillPairsAndTimes {
				continue
			}
			switch field.Message().FullName() {
			case "gctrpc.CurrencyPair":
				pair := &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD", Delimiter: "-"}
				m.Set(field, protoreflect.ValueOfMessage(pair.ProtoReflect()))
			case "google.protobuf.Timestamp":
				timestamp := timestamppb.New(start)
				if strings.Contains(name, "end") || name == "to" {
					timestamp = timestamppb.New(end)
				}
				m.Set(field, protoreflect.ValueOfMessage(timestamp.ProtoReflect()))
			default:
				if level < fillMessages || depth > 3 {
					continue
				}
				nested := m.NewField(field).Message()
				fillRequest(nested, exchangeName, level, depth+1)
				m.Set(field, protoreflect.ValueOfMessage(nested))
			}
		}
	}
}

// requestString returns a plausible value for the string fields handlers validate first
func requestString(name, exchangeName string, start, end time.Time) (string, bool) {
	switch {
	case name == "exchange" || name == "exchange_name" || name == "exchanges":
		return exchangeName, true
	case strings.Contains(name, "asset"):
		return asset.Spot.String(), true
	case name == "start" || name == "start_date" || name == "start_time" || name == "from":
		return start.Format(common.SimpleTimeFormatWithTimezone), true
	case name == "end" || name == "end_date" || name == "end_time" || name == "to":
		return end.Format(common.SimpleTimeFormatWithTimezone), true
	case name == "side":
		return "BUY", true
	case name == "order_type":
		return "LIMIT", true
	case name == "currency" || name == "cryptocurrency" || name == "code" || name == "base":
		return "BTC", true
	case name == "quote":
		return "USD", true
	case name == "pair" || name == "symbol":
		return "BTC-USD", true
	}
	return "", false
}
