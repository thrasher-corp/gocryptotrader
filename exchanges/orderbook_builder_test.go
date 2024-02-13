package exchange

import (
	"testing"

	"github.com/d5/tengo/assert"
)

func FetcherTester() {}

func TestXxx(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewOrderbookBuilder(nil, nil, nil))
}
