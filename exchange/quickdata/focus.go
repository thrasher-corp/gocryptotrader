package quickdata

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// Focus based errors
var (
	ErrUnsetFocusType                  = errors.New("focus type is unset")
	ErrUnsupportedFocusType            = errors.New("unsupported focus type")
	ErrInvalidRESTPollTime             = errors.New("invalid REST poll time")
	ErrInvalidAssetForFocusType        = errors.New("invalid asset for focus type")
	ErrCredentialsRequiredForFocusType = errors.New("credentials required for this focus type")
)

// FocusType is an identifier for data types that quickData can gather
type FocusType uint8

// FocusData contains information on what data quickData should gather
// how it should be gathered as well as a channel for delivering that data
type FocusData struct {
	focusType             FocusType
	useWebsocket          bool
	restPollTime          time.Duration
	m                     sync.RWMutex
	isOnceOff             bool
	hasBeenSuccessful     bool
	hasBeenSuccessfulChan chan any
	Stream                chan any
	FailureTolerance      uint64
	failures              uint64
}

// focusToSub maps FocusType to subscription channels allowing for easy
// websocket subscription generation without needing to know about an exchange's underlying implementation
var focusToSub = map[FocusType]string{
	OrderBookFocusType:       subscription.OrderbookChannel,
	TickerFocusType:          subscription.TickerChannel,
	KlineFocusType:           subscription.CandlesChannel,
	TradesFocusType:          subscription.AllTradesChannel,
	ActiveOrdersFocusType:    subscription.MyOrdersChannel,
	AccountHoldingsFocusType: subscription.MyAccountChannel,
}

// NewFocusData creates a new FocusData instance and initializes its internal fields.
func NewFocusData(focusType FocusType, isOnceOff, useWebsocket bool, restPollTime time.Duration) *FocusData {
	fd := &FocusData{
		focusType:    focusType,
		useWebsocket: useWebsocket,
		restPollTime: restPollTime,
		isOnceOff:    isOnceOff,
	}
	fd.Init()
	return fd
}

// Init called to ensure that data is properly initialised
func (f *FocusData) Init() {
	f.hasBeenSuccessfulChan = make(chan any)
	f.Stream = make(chan any, 1)
	f.hasBeenSuccessful = false
	if f.FailureTolerance == 0 {
		f.FailureTolerance = 5
	}
}

// Validate checks if the FocusData instance is good to go
// It is assumed that this will be called before use rather than willy-nilly
func (f *FocusData) Validate(k *key.ExchangeAssetPair) error {
	if err := common.NilGuard(f, k); err != nil {
		return err
	}
	if f.focusType == UnsetFocusType {
		return ErrUnsetFocusType
	}
	if f.restPollTime <= 0 && !f.useWebsocket {
		return ErrInvalidRESTPollTime
	}
	// lazy initialisation of mutex and channels
	// we could error and cause a fuss because a silly user didn't call init, but what good is it to be annoying?
	if f.hasBeenSuccessfulChan == nil || f.Stream == nil {
		f.Init()
	}
	if f.FailureTolerance == 0 {
		f.FailureTolerance = 5
	}
	if slices.Contains(futuresOnlyFocusList, f.focusType) && !k.Asset.IsFutures() {
		return ErrInvalidAssetForFocusType
	}
	return nil
}

// stream attempts to send data to the Stream channel without blocking
func (f *FocusData) stream(d any) {
	select {
	case f.Stream <- d:
	default: // drop data that doesn't fit or get listened to
	}
}

// setSuccessful sets the hasBeenSuccessful flag to true and closes the
// hasBeenSuccessfulChan channel. It uses a read-write lock to ensure that
// the flag is only set once, preventing multiple goroutines from setting it
// simultaneously. If the flag has already been set, it returns immediately
// without doing anything further.
func (f *FocusData) setSuccessful() {
	f.m.RLock()
	if f.hasBeenSuccessful {
		f.m.RUnlock()
		return
	}
	f.m.RUnlock()
	f.m.Lock()
	defer f.m.Unlock()
	if f.hasBeenSuccessful {
		return
	}
	f.hasBeenSuccessful = true
	close(f.hasBeenSuccessfulChan)
}

// UseWebsocket returns whether the focus type desires a websocket connection
func (f *FocusData) UseWebsocket() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.useWebsocket
}

// RequiresAuth returns whether the focus type requires authentication
func RequiresAuth(ft FocusType) bool {
	return slices.Contains(authFocusList, ft)
}

// HasBeenSuccessful returns whether the focus has successfully received data at least once.
func (f *FocusData) HasBeenSuccessful() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.hasBeenSuccessful
}

// FocusTypes are what quickData uses to grant permission for it to grab data
const (
	UnsetFocusType FocusType = iota
	OpenInterestFocusType
	TickerFocusType
	OrderBookFocusType
	FundingRateFocusType
	TradesFocusType
	AccountHoldingsFocusType
	ActiveOrdersFocusType
	KlineFocusType
	ContractFocusType
	OrderLimitsFocusType
	URLFocusType
)

// allFocusList is a list of all supported FocusTypes
var allFocusList = []FocusType{
	OpenInterestFocusType,
	TickerFocusType,
	OrderBookFocusType,
	FundingRateFocusType,
	TradesFocusType,
	AccountHoldingsFocusType,
	ActiveOrdersFocusType,
	KlineFocusType,
	ContractFocusType,
	OrderLimitsFocusType,
	URLFocusType,
}

var wsSupportedFocusList = []FocusType{
	TickerFocusType,
	OrderBookFocusType,
	KlineFocusType,
	TradesFocusType,
	ActiveOrdersFocusType,
	AccountHoldingsFocusType,
}

// authFocusList is a list of FocusTypes that require authentication
var authFocusList = []FocusType{
	AccountHoldingsFocusType,
	ActiveOrdersFocusType,
}

// futuresOnlyFocusList is a list of FocusTypes that are only valid for futures assets
var futuresOnlyFocusList = []FocusType{
	OpenInterestFocusType,
	FundingRateFocusType,
}

// Valid checks if the FocusType is supported
func (f FocusType) Valid() error {
	if !slices.Contains(allFocusList, f) {
		return ErrUnsupportedFocusType
	}
	return nil
}

// String returns a string representation of the FocusType
func (f FocusType) String() string {
	switch f {
	case OpenInterestFocusType:
		return "OpenInterestFocusType"
	case TickerFocusType:
		return "TickerFocusType"
	case OrderBookFocusType:
		return "OrderBookFocusType"
	case FundingRateFocusType:
		return "FundingRateFocusType"
	case TradesFocusType:
		return "TradesFocusType"
	case AccountHoldingsFocusType:
		return "AccountHoldingsFocusType"
	case ActiveOrdersFocusType:
		return "ActiveOrdersFocusType"
	case KlineFocusType:
		return "KlineFocusType"
	case ContractFocusType:
		return "ContractFocusType"
	case OrderLimitsFocusType:
		return "OrderLimitsFocusType"
	case URLFocusType:
		return "URLFocusType"
	default:
		return "Unset/Unknown FocusType"
	}
}
