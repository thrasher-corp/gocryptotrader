package kraken

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	errFuturesRFQUIDEmpty        = errors.New("RFQ UID is empty")
	errFuturesRFQPriceEmpty      = errors.New("an RFQ offer needs a bid or an ask")
	errFuturesRFQPricingConflict = errors.New("package and per-leg RFQ prices cannot both be set")
	errFuturesRFQLegsEmpty       = errors.New("an RFQ needs at least one leg")
	errFuturesRFQTooManyLegs     = errors.New("an RFQ takes at most 100 legs")
	errFuturesRFQDuplicateLeg    = errors.New("RFQ legs cannot repeat a contract")
	errFuturesRFQLegSizeZero     = errors.New("RFQ leg size is zero")
	errFuturesRFQAcceptSide      = errors.New("exactly one of the bid or ask must be accepted")
)

// FuturesOpenRFQsResponse holds the open RFQs
type FuturesOpenRFQsResponse struct {
	RFQs       []FuturesRFQ `json:"rfqs"`
	ServerTime time.Time    `json:"serverTime"`
}

// FuturesRFQResponse holds an open or recently closed RFQ
type FuturesRFQResponse struct {
	RFQ        FuturesRFQ `json:"rfq"`
	ServerTime time.Time  `json:"serverTime"`
}

// FuturesRFQ is a request for quotes on a package of contract positions
type FuturesRFQ struct {
	UID    string    `json:"uid"`
	Expiry time.Time `json:"expiry"`
	// MarkPrice is the package's reference price, the sum of each leg's size times its mark price, so it is negative
	// when the short legs are worth more than the long legs
	MarkPrice float64         `json:"markPrice"`
	Legs      []FuturesRFQLeg `json:"legs"`
	// Status is open while the RFQ takes offers, expired or cancelled when it closed without a trade, or filled_bid_side
	// or filled_ask_side when its requester accepted an offer on that side
	Status string `json:"status"`
}

// FuturesRFQLeg is a contract position an RFQ requests quotes on
type FuturesRFQLeg struct {
	Symbol string `json:"symbol"`
	// Size is negative for a short position
	Size      float64 `json:"size"`
	MarkPrice float64 `json:"markPrice"`
}

// FuturesRFQOffersResponse holds the account's offers on RFQs
type FuturesRFQOffersResponse struct {
	Offers     []FuturesRFQOffer `json:"offers"`
	ServerTime time.Time         `json:"serverTime"`
}

// FuturesRFQOffer is an offer the account placed on an RFQ
type FuturesRFQOffer struct {
	UID            string    `json:"uid"`
	RFQUID         string    `json:"rfqUid"`
	PlacementTime  time.Time `json:"placementDate"`
	LastUpdateTime time.Time `json:"lastUpdateDate"`
	// Bid and Ask are the package prices offered, nil for a side the offer does not price as a package
	Bid *types.Number `json:"bid"`
	Ask *types.Number `json:"ask"`
	// BidSide and AskSide price each leg of a side the offer does not price as a package
	BidSide []FuturesRFQLegPrice `json:"bidSide"`
	AskSide []FuturesRFQLegPrice `json:"askSide"`
	// Status is the RFQ's status when the offer was read: open for an open offer, or how the RFQ closed
	Status string `json:"status"`
}

// FuturesRFQLegPrice is the price offered for one leg of an RFQ
type FuturesRFQLegPrice struct {
	Symbol string  `json:"tradeable"`
	Price  float64 `json:"price"`
}

// FuturesPlaceRFQOfferRequest holds the parameters of Place new offer on an open RFQ. Prices are in USD, and the price
// of a package with short legs can be negative or 0, so a package price is nil when it is not set
type FuturesPlaceRFQOfferRequest struct {
	RFQUID string
	// Bid and Ask price the whole package
	Bid *float64
	Ask *float64
	// BidSide and AskSide price each leg instead, and cannot be set with Bid or Ask
	BidSide []FuturesRFQOfferLeg
	AskSide []FuturesRFQOfferLeg
}

// FuturesRFQOfferLeg is the price an offer quotes for one leg of an RFQ
type FuturesRFQOfferLeg struct {
	Contract currency.Pair
	Price    float64
}

// FuturesPlaceRFQOfferResponse holds the UID of an offer placed on an RFQ
type FuturesPlaceRFQOfferResponse struct {
	OfferUID   string    `json:"offerUId"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesCancelRFQOfferRequest holds the parameters of Cancel open offer on open RFQ
type FuturesCancelRFQOfferRequest struct {
	RFQUID string
	// OfferUID selects the offer to cancel; the account's open offer on the RFQ is cancelled when it is empty
	OfferUID string
}

// FuturesAccountOpenRFQsResponse holds the open RFQs the account created
type FuturesAccountOpenRFQsResponse struct {
	RFQs       []FuturesAccountRFQ `json:"rfqs"`
	ServerTime time.Time           `json:"serverTime"`
}

// FuturesAccountRFQ is an open RFQ the account created, with the best offers on it
type FuturesAccountRFQ struct {
	FuturesRFQ
	// BestBid and BestAsk are the best package prices offered, nil when no offer quotes that side
	BestBid *float64 `json:"bestBid"`
	BestAsk *float64 `json:"bestAsk"`
	// BidSide and AskSide price each leg of the offers giving BestBid and BestAsk, when those offers priced each leg
	BidSide []FuturesRFQLegPrice `json:"bidSide"`
	AskSide []FuturesRFQLegPrice `json:"askSide"`
}

// FuturesCreateRFQRequest holds the parameters of Create a new RFQ
type FuturesCreateRFQRequest struct {
	// Legs are the positions quotes are requested on, up to 100, each contract at most once
	Legs []FuturesRFQRequestedLeg
	// Expiry is when the RFQ expires; Kraken applies a default duration when it is zero
	Expiry time.Time
}

// FuturesRFQRequestedLeg is a contract position an RFQ requests quotes on
type FuturesRFQRequestedLeg struct {
	Contract currency.Pair
	// Size is negative for a short position
	Size float64
}

// FuturesRFQUIDResponse holds the UID of the RFQ a request created, cancelled or accepted an offer on
type FuturesRFQUIDResponse struct {
	RFQUID     string    `json:"rfqUid"`
	ServerTime time.Time `json:"serverTime"`
}

// FuturesAcceptRFQOfferRequest holds the parameters of Accept an offer on an open RFQ. Exactly one of Bid and Ask is
// set, to the USD price of the offer accepted, which can be negative or 0 for a package with short legs
type FuturesAcceptRFQOfferRequest struct {
	RFQUID string
	Bid    *float64
	Ask    *float64
}

// futuresRFQCreation is the JSON document Create a new RFQ takes as its json parameter
type futuresRFQCreation struct {
	Legs   []futuresRFQCreationLeg `json:"legs"`
	Expiry string                  `json:"expiry,omitempty"`
}

// futuresRFQCreationLeg is a leg of futuresRFQCreation
type futuresRFQCreationLeg struct {
	Tradeable string  `json:"tradeable"`
	Size      float64 `json:"size"`
}

// futuresRFQRefusal holds the reason Kraken gives, beside a success result, for refusing an RFQ request
type futuresRFQRefusal struct {
	Reason string `json:"reason"`
}
