package subscriptionstest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// Equal is a utility function to compare subscription lists and show a pretty failure message
// It overcomes the verbose depth of assert.ElementsMatch spewConfig
func Equal(tb testing.TB, a, b subscription.List) {
	tb.Helper()
	s, err := subscription.NewStoreFromList(a)
	require.NoError(tb, err, "NewStoreFromList must not error")
	added, missing := s.Diff(b)
	if len(added) > 0 || len(missing) > 0 {
		fail := "Differences:"
		if len(added) > 0 {
			fail = fail + "\n + " + strings.Join(added.Strings(), "\n + ")
		}
		if len(missing) > 0 {
			fail = fail + "\n - " + strings.Join(missing.Strings(), "\n - ")
		}
		assert.Fail(tb, fail, "Subscriptions should be equal")
	}
}
