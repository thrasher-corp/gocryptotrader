package quickspy

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	ErrNilFocusData                    = errors.New("focus data is nil")
	ErrUnsetFocusType                  = errors.New("focus type is unset")
	ErrInvalidRESTPollTime             = errors.New("invalid REST poll time")
	ErrInvalidAssetForFocusType        = errors.New("invalid asset for focus type")
	ErrCredentialsRequiredForFocusType = errors.New("credentials required for this focus type")
)

var focusToSub = map[FocusType]string{
	OrderBookFocusType: subscription.OrderbookChannel,
	TickerFocusType:    subscription.TickerChannel,
	KlineFocusType:     subscription.CandlesChannel,
}

func (f *FocusData) Validate(k *CredentialsKey) error {
	if f == nil {
		return ErrNilFocusData
	}
	if f.Type == UnsetFocusType {
		return ErrUnsetFocusType
	}
	if f.RESTPollTime <= 0 && !f.UseWebsocket {
		return ErrInvalidRESTPollTime
	}
	if !k.ExchangeAssetPair.Asset.IsFutures() {
		switch f.Type {
		case OpenInterestFocusType, FundingRateFocusType, ContractFocusType, OrderExecutionFocusType:
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

func NewFocusData(focusType FocusType, isOnceOff, useWebsocket bool, restPollTime, wsInterval time.Duration) *FocusData {
	return &FocusData{
		Type:              focusType,
		UseWebsocket:      useWebsocket,
		RESTPollTime:      restPollTime,
		IsOnceOff:         isOnceOff,
		WebsocketInterval: wsInterval,
	}
}

// Init called to ensure that lame data is initialised
func (f *FocusData) Init() {
	f.m = new(sync.RWMutex)
	f.HasBeenSuccessfulChan = make(chan any)
	f.Stream = make(chan any)
	f.hasBeenSuccessful = false
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

func (f *FocusData) RequiresAuth() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.Type == AccountHoldingsFocusType || f.Type == ActiveOrdersFocusType || f.Type == OrderPlacementFocusType
}
