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
	Mux:   dispatch.GetNewMux(nil),
}

// Service provides a store for difference exchange orderbooks
type Service struct {
	books map[string]Exchange
	*dispatch.Mux
	mu sync.Mutex
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
	// Determines if there is a max depth of orderbooks and after an append we
	// should remove any items that are outside of this scope. Bittrex and
	// Kraken utilise this field.
	MaxDepth int
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
	maxDepth         int
}

// Action defines a set of differing states required to implement an incoming
// orderbook update used in conjunction with UpdateEntriesByID
type Action uint8

const (
	// Amend applies amount adjustment by ID
	Amend Action = iota + 1
	// Delete removes price level from book by ID
	Delete
	// Insert adds price level to book
	Insert
	// UpdateInsert on conflict applies amount adjustment or appends new amount
	// to book
	UpdateInsert
)

// Update and things and stuff
type Update struct {
	UpdateID   int64 // Used when no time is provided
	UpdateTime time.Time
	Asset      asset.Item
	Action
	Bids []Item
	Asks []Item
	Pair currency.Pair
	// Checksum defines the expected value when the books have been verified
	Checksum uint32
}

// Movement defines orderbook traversal details from either hitting the bids or
// lifting the asks.
type Movement struct {
	// NominalPercentage (real-world) defines how far in percentage terms is
	// your average order price away from the reference price.
	NominalPercentage float64
	// ImpactPercentage defines how far the price has moved on the order book
	// from the reference price.
	ImpactPercentage float64
	// SlippageCost is the cost of the slippage. This is priced in quotation.
	SlippageCost float64
	// StartPrice defines the reference price or the head of the orderbook side.
	StartPrice float64
	// EndPrice defines where the price has ended on the orderbook side.
	EndPrice float64
	// Sold defines the amount of currency sold.
	Sold float64
	// Purchases defines the amount of currency purchased.
	Purchased float64
	// AverageOrderCost defines the average order cost of position as it slips
	// through the orderbook tranches.
	AverageOrderCost float64
	// FullBookSideConsumed defines if the orderbook liquidty has been consumed
	// by the requested amount. This might not represent the actual book on the
	// exchange as they might restrict the amount of information being passed
	// back from either a REST request or websocket stream.
	FullBookSideConsumed bool
}

// SideAmounts define the amounts total for the tranches, total value in
// quotation and the cumulative base amounts.
type SideAmounts struct {
	Tranches   int64
	QuoteValue float64
	BaseAmount float64
}
