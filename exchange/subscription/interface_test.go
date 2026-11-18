package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchange"
	shared "github.com/thrasher-corp/gocryptotrader/exchange/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchange/subscription"
)

// TestIExchange ensures that IExchange is a subset of IBotExchange, so when an exchange is passed by interface, it can still use ExpandTemplates
func TestIExchange(t *testing.T) {
	assert.Implements(t, (*subscription.IExchange)(nil), exchange.IBotExchange(&shared.CustomEx{}))
	var _ subscription.IExchange = exchange.IBotExchange(nil)
}
