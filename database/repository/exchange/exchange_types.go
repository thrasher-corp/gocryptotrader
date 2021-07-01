package exchange

import (
	"errors"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
)

var (
	exchangeCache = cache.New(30)
	// ErrNoExchangeFound is a basic predefined error
	ErrNoExchangeFound = errors.New("exchange not found")
)

// Details holds exchange information such as Name
type Details struct {
	UUID uuid.UUID
	Name string
}
