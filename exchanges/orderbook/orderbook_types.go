package orderbook

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for orderbook package
const (
	bidLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Bids: %w"
	askLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Asks: %w"
	bookLengthIssue    = "Potential book issue for exchange %s pair %s asset %s length Bids %d length Asks %d"
)

// Vars for the orderbook package
var (
	errExchangeNameUnset   = errors.New("orderbook exchange name not set")
	errPairNotSet          = errors.New("orderbook currency pair not set")
	errAssetTypeNotSet     = errors.New("orderbook asset type not set")
	errCannotFindOrderbook = errors.New("cannot find orderbook(s)")
	errPriceNotSet         = errors.New("price cannot be zero")
	errAmountInvalid       = errors.New("amount cannot be less or equal to zero")
	errPriceOutOfOrder     = errors.New("pricing out of order")
	errIDOutOfOrder        = errors.New("ID out of order")
	errDuplication         = errors.New("price duplication")
	errIDDuplication       = errors.New("id duplication")
	errPeriodUnset         = errors.New("funding rate period is unset")
	errNotEnoughLiquidity  = errors.New("not enough liquidity")
)

var service = Service{
	books: make(map[string]Exchange),
	Mux:   dispatch.GetNewMux(),
}

// Service provides a store for difference exchange orderbooks
type Service struct {
	books map[string]Exchange
	*dispatch.Mux
	sync.Mutex
}

// Exchange defines a holder for the exchange specific depth items with a
// specific ID associated with that exchange
type Exchange struct {
	m  map[asset.Item]map[*currency.Item]map[*currency.Item]*Depth
	ID uuid.UUID
}

// Item stores the amount and price values
type Item struct {
	Amount float64
	Price  float64
	ID     int64

	// Funding rate field
	Period int64

	// Contract variables
	LiquidationOrders int64
	OrderCount        int64
}

// Items defines a slice of orderbook items
type Items []Item

// Base holds the fields for the orderbook base
type Base struct {
	Bids Items
	Asks Items

	Exchange string
	Pair     currency.Pair
	Asset    asset.Item

	LastUpdated  time.Time
	LastUpdateID int64
	// PriceDuplication defines whether an orderbook can contain duplicate
	// prices in a payload
	PriceDuplication bool
	IsFundingRate    bool
	// VerifyOrderbook allows for a toggle between orderbook verification set by
	// user configuration, this allows for a potential processing boost but
	// a potential for orderbook integrity being deminished.
	VerifyOrderbook bool `json:"-"`
	// RestSnapshot defines if the depth was applied via the REST protocol thus
	// an update cannot be applied via websocket mechanics and a resubscription
	// would need to take place to maintain book integrity
	RestSnapshot bool
	// Checks if the orderbook needs ID alignment as well as price alignment
	IDAlignment bool
}

type byOBPrice []Item

func (a byOBPrice) Len() int           { return len(a) }
func (a byOBPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byOBPrice) Less(i, j int) bool { return a[i].Price < a[j].Price }

type options struct {
	exchange         string
	pair             currency.Pair
	asset            asset.Item
	lastUpdated      time.Time
	lastUpdateID     int64
	priceDuplication bool
	isFundingRate    bool
	VerifyOrderbook  bool
	restSnapshot     bool
	idAligned        bool
}
