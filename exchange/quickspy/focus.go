package quickspy

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// FocusType is an identifier for data types that quickspy can gather
type FocusType int

// FocusData contains information on what data quickspy should gather
// how it should be gathered as well as a channel for delivering that data
type FocusData struct {
	Type                  FocusType
	UseWebsocket          bool
	RESTPollTime          time.Duration
	m                     *sync.RWMutex
	IsOnceOff             bool
	hasBeenSuccessful     bool
	HasBeenSuccessfulChan chan any
	Stream                chan any
}

// Focus based errors
var (
	ErrUnsetFocusType                  = errors.New("focus type is unset")
	ErrInvalidRESTPollTime             = errors.New("invalid REST poll time")
	ErrInvalidAssetForFocusType        = errors.New("invalid asset for focus type")
	ErrCredentialsRequiredForFocusType = errors.New("credentials required for this focus type")
	ErrNoCredentials                   = errors.New("no credentials provided")
)

// focusToSub maps FocusType to subscription channels allowing for easy
// websocket subscription generation without needing to know about an exchange's underlying implementation
var focusToSub = map[FocusType]string{
	OrderBookFocusType: subscription.OrderbookChannel,
	TickerFocusType:    subscription.TickerChannel,
	KlineFocusType:     subscription.CandlesChannel,
}

// NewFocusData creates a new FocusData instance and initializes its internal fields.
func NewFocusData(focusType FocusType, isOnceOff, useWebsocket bool, restPollTime time.Duration) *FocusData {
	fd := &FocusData{
		Type:         focusType,
		UseWebsocket: useWebsocket,
		RESTPollTime: restPollTime,
		IsOnceOff:    isOnceOff,
	}
	fd.Init()
	return fd
}

// Init called to ensure that lame data is initialised
func (f *FocusData) Init() {
	f.m = new(sync.RWMutex)
	f.HasBeenSuccessfulChan = make(chan any)
	f.Stream = make(chan any)
	f.hasBeenSuccessful = false
}

// Validate checks if the FocusData instance is good to go
func (f *FocusData) Validate(k *CredentialsKey) error {
	if err := common.NilGuard(f); err != nil {
		return err
	}
	if err := common.NilGuard(k); err != nil {
		return err
	}
	if f.Type == UnsetFocusType {
		return ErrUnsetFocusType
	}
	if f.RESTPollTime <= 0 && !f.UseWebsocket {
		return ErrInvalidRESTPollTime
	}
	// lazy initialisation of mutex and channels
	// we could error and cause a fuss because a silly user didn't call init, but what good is it to be annoying?
	if f.m == nil {
		f.m = new(sync.RWMutex)
	}
	if f.HasBeenSuccessfulChan == nil {
		f.HasBeenSuccessfulChan = make(chan any)
	}
	if f.Stream == nil {
		f.Stream = make(chan any)
	}
	if k.Credentials != nil && k.Credentials.IsEmpty() {
		return ErrNoCredentials
	}
	if !k.ExchangeAssetPair.Asset.IsFutures() {
		switch f.Type {
		case OpenInterestFocusType, FundingRateFocusType, ContractFocusType:
			return ErrInvalidAssetForFocusType
		}
	}
	if k.Credentials == nil {
		switch f.Type {
		case AccountHoldingsFocusType, ActiveOrdersFocusType:
			return ErrCredentialsRequiredForFocusType
		}
	}
	return nil
}

// SetSuccessful sets the hasBeenSuccessful flag to true and closes the
// HasBeenSuccessfulChan channel. It uses a read-write lock to ensure that
// the flag is only set once, preventing multiple goroutines from setting it
// simultaneously. If the flag has already been set, it returns immediately
// without doing anything further.
func (f *FocusData) SetSuccessful() {
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
	close(f.HasBeenSuccessfulChan)
}

func (f *FocusData) RequiresWebsocket() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.UseWebsocket
}

// FocusTypes are what quickspy uses to grant permission for it to grab data
const (
	UnsetFocusType FocusType = iota
	OpenInterestFocusType
	TickerFocusType
	OrderBookFocusType
	FundingRateFocusType
	TradesFocusType
	AccountHoldingsFocusType
	ActiveOrdersFocusType
	OrderPlacementFocusType
	KlineFocusType
	ContractFocusType
	OrderExecutionFocusType
	URLFocusType
)

var focusList = []FocusType{
	OpenInterestFocusType,
	TickerFocusType,
	OrderBookFocusType,
	FundingRateFocusType,
	TradesFocusType,
	AccountHoldingsFocusType,
	ActiveOrdersFocusType,
	OrderPlacementFocusType,
	KlineFocusType,
	ContractFocusType,
	OrderExecutionFocusType,
	URLFocusType,
}

func (f *FocusData) RequiresAuth() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.Type == AccountHoldingsFocusType || f.Type == ActiveOrdersFocusType || f.Type == OrderPlacementFocusType
}

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
	case OrderPlacementFocusType:
		return "OrderPlacementFocusType"
	case KlineFocusType:
		return "KlineFocusType"
	case ContractFocusType:
		return "ContractFocusType"
	case OrderExecutionFocusType:
		return "OrderExecutionFocusType"
	case URLFocusType:
		return "URLFocusType"
	default:
		return "Unset/Unknown FocusType"
	}
}
