package exchange

import (
	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
)

var (
	exchangeCache = cache.New(10)
)

// Details holds exchange information such as Name
type Details struct {
	UUID uuid.UUID
	Name string
}