package quickspy

import (
	"errors"
	"slices"
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
	ErrUnsupportedFocusType            = errors.New("unsupported focus type")
	ErrInvalidRESTPollTime             = errors.New("invalid REST poll time")
	ErrInvalidAssetForFocusType        = errors.New("invalid asset for focus type")
	ErrCredentialsRequiredForFocusType = errors.New("credentials required for this focus type")
	ErrNoCredentials                   = errors.New("no credentials provided")
)

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
// It is assumed that this will be called before use rather than willy-nilly
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
	if f.m == nil || f.HasBeenSuccessfulChan == nil || f.Stream == nil {
		f.Init()
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
	if slices.Contains(authFocusList, f.Type) && k.Credentials == nil {
		return ErrCredentialsRequiredForFocusType
	}
	if slices.Contains(futuresOnlyFocusList, f.Type) && !k.ExchangeAssetPair.Asset.IsFutures() {
		return ErrInvalidAssetForFocusType
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
	OrderLimitsFocusType
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
	KlineFocusType,
	ContractFocusType,
	OrderLimitsFocusType,
	URLFocusType,
}

var authFocusList = []FocusType{
	AccountHoldingsFocusType,
	ActiveOrdersFocusType,
	OrderLimitsFocusType,
}

var futuresOnlyFocusList = []FocusType{
	OpenInterestFocusType,
	FundingRateFocusType,
}

func (f *FocusData) RequiresAuth() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.Type == AccountHoldingsFocusType || f.Type == ActiveOrdersFocusType || f.Type == OrderPlacementFocusType
}

func (f *FocusData) HasBeenSuccessful() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.hasBeenSuccessful
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
	case OrderLimitsFocusType:
		return "OrderLimitsFocusType"
	case URLFocusType:
		return "URLFocusType"
	default:
		return "Unset/Unknown FocusType"
	}
}
