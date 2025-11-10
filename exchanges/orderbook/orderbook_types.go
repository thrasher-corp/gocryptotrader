package orderbook

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	bidLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Bids: %w"
	askLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Asks: %w"
	bookLengthIssue    = "Potential book issue for exchange %s pair %s asset %s length Bids %d length Asks %d"
)

// Public errors
var (
	ErrOrderbookNotFound = errors.New("cannot find orderbook(s)")
	ErrPriceZero         = errors.New("price cannot be zero")
	ErrExchangeNameEmpty = errors.New("empty orderbook exchange name")
)

var (
	errPairNotSet           = errors.New("orderbook currency pair not set")
	errAssetTypeNotSet      = errors.New("orderbook asset type not set")
	errAmountInvalid        = errors.New("amount cannot be less or equal to zero")
	errPriceOutOfOrder      = errors.New("pricing out of order")
	errIDOutOfOrder         = errors.New("ID out of order")
	errDuplication          = errors.New("price duplication")
	errIDDuplication        = errors.New("id duplication")
	errPeriodUnset          = errors.New("funding rate period is unset")
	errNotEnoughLiquidity   = errors.New("not enough liquidity")
	errChecksumStringNotSet = errors.New("checksum string not set")
)

var s = store{
	orderbooks:      make(map[key.ExchangeAssetPair]book),
	exchangeRouters: make(map[string]uuid.UUID),
	signalMux:       dispatch.GetNewMux(nil),
}

type book struct {
	RouterID uuid.UUID
	Depth    *Depth
}

// store provides a centralised store for orderbooks
type store struct {
	orderbooks      map[key.ExchangeAssetPair]book
	exchangeRouters map[string]uuid.UUID
	signalMux       *dispatch.Mux
	m               sync.RWMutex
}

// Level contains an orderbook price and the aggregated order amount at that price.
type Level struct {
	Amount float64
	// StrAmount is a string representation of the amount. e.g. 0.00000100 this
	// parsed as a float will constrict comparison to 1e-6 not 1e-8 or
	// potentially will round value which is not ideal.
	StrAmount string
	Price     float64
	// StrPrice is a string representation of the price. e.g. 0.00000100 this
	// parsed as a float will constrict comparison to 1e-6 not 1e-8 or
	// potentially will round value which is not ideal.
	StrPrice string
	ID       int64

	// Funding rate field
	Period int64

	// Contract variables
	LiquidationOrders int64
	OrderCount        int64
}

// Book contains an orderbook
type Book struct {
	Bids Levels
	Asks Levels

	Exchange string
	Pair     currency.Pair
	Asset    asset.Item

	// LastUpdated is the time when a change occurred on the exchange books.
	// Note: This does not necessarily indicate the change is out of sync with
	// the exchange. It represents the last known update time from the exchange,
	// which could be stale if there have been no recent changes.
	LastUpdated time.Time

	// LastPushed is the time the exchange pushed this update. This helps
	// determine factors like distance from exchange (latency) and routing
	// time, which can affect the time it takes for an update to reach the user
	// from the exchange.
	LastPushed time.Time

	// InsertedAt is the time the update was inserted into the orderbook
	// management system. This field is used to calculate round-trip times and
	// processing delays, e.g., InsertedAt.Sub(LastPushed) represents the
	// total processing time including network latency.
	InsertedAt time.Time

	LastUpdateID int64
	// PriceDuplication defines whether an orderbook can contain duplicate
	// prices in a payload
	PriceDuplication bool
	IsFundingRate    bool
	// ValidateOrderbook allows for a toggle between orderbook verification set by
	// user configuration, this allows for a potential processing boost but
	// a potential for orderbook integrity being deminished.
	ValidateOrderbook bool
	// RestSnapshot defines if the depth was applied via the REST protocol thus
	// an update cannot be applied via websocket mechanics and a resubscription
	// would need to take place to maintain book integrity
	RestSnapshot bool
	// Checks if the orderbook needs ID alignment as well as price alignment
	IDAlignment bool
	// Determines if there is a max depth of orderbooks and after an append we
	// should remove any items that are outside of this scope. Kraken utilises
	// this field.
	MaxDepth int
	// ChecksumStringRequired defines if the checksum is built from the raw
	// string representations of the price and amount. This helps alleviate any
	// potential rounding issues.
	ChecksumStringRequired bool
}

type options struct {
	exchange               string
	pair                   currency.Pair
	asset                  asset.Item
	lastUpdated            time.Time
	lastPushed             time.Time
	insertedAt             time.Time
	lastUpdateID           int64
	priceDuplication       bool
	isFundingRate          bool
	validateOrderbook      bool
	restSnapshot           bool
	idAligned              bool
	checksumStringRequired bool
	maxDepth               int
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
	// through the orderbook Levels.
	AverageOrderCost float64
	// FullBookSideConsumed defines if the orderbook liquidty has been consumed
	// by the requested amount. This might not represent the actual book on the
	// exchange as they might restrict the amount of information being passed
	// back from either a REST request or websocket update
	FullBookSideConsumed bool
}

// SideAmounts define the amounts total for the Levels, total value in
// quotation and the cumulative base amounts.
type SideAmounts struct {
	Levels     int64
	QuoteValue float64
	BaseAmount float64
}

// LevelsArrayPriceAmount used to unmarshal orderbook levels from JSON slice of arrays
// e.g. [[price, amount], [price, amount]] or [][2]types.Number type declaration
type LevelsArrayPriceAmount Levels

// UnmarshalJSON implements json.Unmarshaler
func (l *LevelsArrayPriceAmount) UnmarshalJSON(data []byte) error {
	var v [][2]types.Number
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*l = make(LevelsArrayPriceAmount, len(v))
	for x := range v {
		(*l)[x].Price = v[x][0].Float64()
		(*l)[x].Amount = v[x][1].Float64()
	}
	return nil
}

// Levels converts the LevelsArrayPriceAmount to a orderbook.Levels type
func (l *LevelsArrayPriceAmount) Levels() Levels {
	return Levels(*l)
}
