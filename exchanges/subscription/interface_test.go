package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	shared "github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// TestIExchange ensures that IExchange is a subset of IBotExchange, so when an exchange is passed by interface, it can still use ExpandTemplates
func TestIExchange(t *testing.T) {
	assert.Implements(t, (*subscription.IExchange)(nil), exchange.IBotExchange(&shared.CustomEx{}))
	var _ subscription.IExchange = exchange.IBotExchange(nil)
}
