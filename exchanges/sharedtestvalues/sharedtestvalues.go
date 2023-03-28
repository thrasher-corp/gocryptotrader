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

	warningSkip             = "Skipping function test"
	warningKeys             = "API test keys have not been set"
	warningManipulateOrders = "can manipulate real orders has been set to false"
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

// SkipUnsetCredentials is a test helper function checking if the authenticated function
// can perform the required test.
func SkipUnsetCredentials(t *testing.T, exch exchange.IBotExchange, canManipulateOrders ...bool) {
	t.Helper()

	areTestAPIKeysSet := AreAPIkeysSet(exch)
	supportsManipulatingOrders := len(canManipulateOrders) > 0
	allowedToManipulateOrders := supportsManipulatingOrders && canManipulateOrders[0]

	if areTestAPIKeysSet && !supportsManipulatingOrders ||
		areTestAPIKeysSet && allowedToManipulateOrders {
		return
	}

	message := []string{warningSkip}
	if !areTestAPIKeysSet {
		message = append(message, warningKeys)
	}

	if supportsManipulatingOrders && !allowedToManipulateOrders {
		message = append(message, warningManipulateOrders)
	}
	message = append(message, warningHowTo)
	t.Skip(strings.Join(message, ", "))
}

// SkipCredentialsSetCantManipulate will only skip if the credentials are set
// correctly and can manipulate orders is set to false. It will continue normal
// operations if credentials are not set, giving better code coverage.
func SkipCredentialsSetCantManipulate(t *testing.T, exch exchange.IBotExchange, canManipulateOrders bool) {
	t.Helper()

	if !AreAPIkeysSet(exch) || canManipulateOrders {
		return
	}

	message := []string{warningSkip, warningManipulateOrders}
	t.Skip(strings.Join(message, ", "))
}

// AreAPIkeysSet returns if the API keys are set.
func AreAPIkeysSet(exch exchange.IBotExchange) bool {
	return exch.VerifyAPICredentials(exch.GetDefaultCredentials()) == nil
}
