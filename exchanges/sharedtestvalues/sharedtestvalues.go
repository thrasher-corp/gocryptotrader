package sharedtestvalues

import (
	"strings"
	"testing"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
)

// This package is only to be referenced in test files
const (
	// WebsocketResponseDefaultTimeout used in websocket testing
	// Defines wait time for receiving websocket response before cancelling
	WebsocketResponseDefaultTimeout = (3 * time.Second)
	// WebsocketResponseExtendedTimeout used in websocket testing
	// Defines wait time for receiving websocket response before cancelling
	WebsocketResponseExtendedTimeout = (15 * time.Second)
	// WebsocketChannelOverrideCapacity used in websocket testing
	// Defines channel capacity as defaults size can block tests
	WebsocketChannelOverrideCapacity = 500

	MockTesting = "Mock testing framework in use for %s exchange on REST endpoints only"
	LiveTesting = "Mock testing bypassed; live testing of REST endpoints in use for %s exchange"

	warningSkip             = "Skipping test"
	warningKeys             = "API test keys have not been set"
	warningManipulateOrders = "variable `canManipulateRealOrders` is false"
	warningHowTo            = "these values can be set at the top of the test file."
)

// GetWebsocketInterfaceChannelOverride returns a new interface based channel
// with the capacity set to WebsocketChannelOverrideCapacity
func GetWebsocketInterfaceChannelOverride() chan interface{} {
	return make(chan interface{}, WebsocketChannelOverrideCapacity)
}

// GetWebsocketStructChannelOverride returns a new struct based channel
// with the capacity set to WebsocketChannelOverrideCapacity
func GetWebsocketStructChannelOverride() chan struct{} {
	return make(chan struct{}, WebsocketChannelOverrideCapacity)
}

// NewTestWebsocket returns a test websocket object
func NewTestWebsocket() *stream.Websocket {
	return &stream.Websocket{
		Init:              true,
		DataHandler:       make(chan interface{}, WebsocketChannelOverrideCapacity),
		ToRoutine:         make(chan interface{}, 1000),
		TrafficAlert:      make(chan struct{}),
		ReadMessageErrors: make(chan error),
		Subscribe:         make(chan []stream.ChannelSubscription, 10),
		Unsubscribe:       make(chan []stream.ChannelSubscription, 10),
		Match:             stream.NewMatch(),
	}
}

// SkipTestIfCredentialsUnset is a test helper function checking if the
// authenticated function can perform the required test.
func SkipTestIfCredentialsUnset(t *testing.T, exch exchange.IBotExchange, canManipulateOrders ...bool) {
	t.Helper()

	if len(canManipulateOrders) > 1 {
		t.Fatal("more than one canManipulateOrders boolean value has been supplied, please remove")
	}

	areTestAPICredentialsSet := AreAPICredentialsSet(exch)
	supportsManipulatingOrders := len(canManipulateOrders) > 0
	allowedToManipulateOrders := supportsManipulatingOrders && canManipulateOrders[0]

	if (areTestAPICredentialsSet && !supportsManipulatingOrders) ||
		(areTestAPICredentialsSet && allowedToManipulateOrders) {
		return
	}

	message := []string{warningSkip}
	if !areTestAPICredentialsSet {
		message = append(message, warningKeys)
	}

	if supportsManipulatingOrders && !allowedToManipulateOrders {
		message = append(message, warningManipulateOrders)
	}
	message = append(message, warningHowTo)
	t.Skip(strings.Join(message, ", "))
}

// SkipTestIfCannotManipulateOrders will only skip if the credentials are set
// correctly and can manipulate orders is set to false. It will continue normal
// operations if credentials are not set, giving better code coverage.
func SkipTestIfCannotManipulateOrders(t *testing.T, exch exchange.IBotExchange, canManipulateOrders bool) {
	t.Helper()

	if !AreAPICredentialsSet(exch) || canManipulateOrders {
		return
	}

	t.Skip(warningSkip + ", " + warningManipulateOrders)
}

// AreAPICredentialsSet returns if the API credentials are set.
func AreAPICredentialsSet(exch exchange.IBotExchange) bool {
	return exch.VerifyAPICredentials(exch.GetDefaultCredentials()) == nil
}
