package protocol

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errFeaturesIsNil           = errors.New("protocol features is nil")
	errUnsupported             = errors.New("unsupported functionality")
	errWebsocketIsNil          = errors.New("protocol websocket is nil")
	errRESTIsNil               = errors.New("protocol REST is nil")
	errRESTIsAlreadySetup      = errors.New("protocol REST is already setup")
	errWebsocketIsAlreadySetup = errors.New("protocol websocket is already setup")
)

// SetupREST sets up a new protocol component state
func (f *Features) SetupREST(s *State) error {
	if s == nil {
		return errStateIsNil
	}
	f.Lock()
	defer f.Unlock()
	if f.REST != nil {
		return errRESTIsAlreadySetup
	}

	f.REST = s
	return nil
}

// SetupWebsocket sets up a new protocol component state
func (f *Features) SetupWebsocket(s *State) error {
	if s == nil {
		return errStateIsNil
	}
	f.Lock()
	defer f.Unlock()
	if f.Websocket != nil {
		return errWebsocketIsAlreadySetup
	}

	f.Websocket = s
	return nil
}

// SetREST changes protocol component states
func (f *Features) SetREST(s State) error {
	f.Lock()
	defer f.Unlock()
	return f.REST.SetFunctionality(s)
}

// SetWebsocket changes protocol component states
func (f *Features) SetWebsocket(s State) error {
	f.Lock()
	defer f.Unlock()
	return f.Websocket.SetFunctionality(s)
}

// Globals define shared functionality accross protocols
type Globals struct {
	WithdrawalPermissions   uint32                  `json:"withdrawalPermissions,omitempty"`
	AutoPairUpdate          *bool                   `json:"autoPairUpdates,omitempty"`
	KlineSupportedIntervals map[kline.Interval]bool `json:"supportedKlineIntervals,omitempty"`
}

// SetGlobals sets global values that act across protocol functionality
func (f *Features) SetGlobals(g *Globals) error {
	if g == nil {
		return errors.New("globals cannot be nil")
	}
	f.Lock()
	f.autoPairUpdate = g.AutoPairUpdate
	f.withdrawalPermissions = g.WithdrawalPermissions
	f.klineSupportedIntervals = g.KlineSupportedIntervals
	f.Unlock()
	return nil
}

// GetEnabledKlineIntervals returns a slice of all enabled intervals
func (f *Features) GetEnabledKlineIntervals() ([]kline.Interval, error) {
	f.RLock()
	defer f.RUnlock()

	var newMap []kline.Interval
	for k, v := range f.klineSupportedIntervals {
		if v {
			newMap = append(newMap, k)
		}
	}
	return newMap, nil
}

// SetKlineSupportedIntervals sets if a kline interval is enabled
func (f *Features) SetKlineSupportedIntervals(nv map[kline.Interval]bool) error {
	if nv == nil {
		return errors.New("new values cannot be nil")
	}

	f.Lock()
	defer f.Unlock()

	for k, v := range nv {
		_, ok := f.klineSupportedIntervals[k]
		if !ok {
			return fmt.Errorf("kline interval [%s] not supported by exchange", k)
		}
		f.klineSupportedIntervals[k] = v
	}
	return nil
}

// AutoPairUpdateEnabled returns if pairs can be automatically updated
func (f *Features) AutoPairUpdateEnabled() bool {
	f.RLock()
	defer f.RUnlock()
	return isEnabled(f.autoPairUpdate)
}

// SetAutoPairUpdate sets if auto pair updates are enabled
func (f *Features) SetAutoPairUpdate(enabled bool) error {
	f.Lock()
	defer f.Unlock()
	if f.autoPairUpdate == nil {
		return errors.New("autopair updates are not supported")
	}

	if enabled {
		f.autoPairUpdate = convert.BoolPtrT
	} else {
		f.autoPairUpdate = convert.BoolPtrF
	}
	return nil
}

// SetWithdrawalPermissions sets withdrawal permisions into shared protocol
// values
func (f *Features) SetWithdrawalPermissions(p uint32) {
	f.Lock()
	f.Shared.WithdrawPermissions = p
	f.Unlock()
}

// GetWithdrawalPermissions changes protocol component states
func (f *Features) GetWithdrawalPermissions() uint32 {
	f.RLock()
	defer f.RUnlock()
	return f.Shared.WithdrawPermissions
}

// IsWebsocketSupported returns if the websocket protocol is supported
func (f *Features) IsWebsocketSupported() bool {
	return f.Websocket != nil
}

// IsRESTSupported returns if the REST protocol is supported
func (f *Features) IsRESTSupported() bool {
	return f.REST != nil
}

// IsWebsocketEnabled returns if websocket protocol is enabled
func (f *Features) IsWebsocketEnabled() (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.Websocket == nil {
		return false, errWebsocketIsNil
	}
	return f.Websocket.ProtocolEnabled, nil
}

// IsRESTEnabled returns if REST protocol is enabled
func (f *Features) IsRESTEnabled() (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.REST == nil {
		return false, errRESTIsNil
	}
	return f.REST.ProtocolEnabled, nil
}

// IsRestAuthenticationEnabled returns if authentication services are enabled on
// the REST protocol
func (f *Features) IsRestAuthenticationEnabled() (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.REST == nil {
		return false, errRESTIsNil
	}
	return f.REST.AuthenticationEnabled, nil
}

// IsWebsocketAuthenticationEnabled returns if authentication services are
// enabled on the websocket protocol
func (f *Features) IsWebsocketAuthenticationEnabled() (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.Websocket == nil {
		return false, errWebsocketIsNil
	}
	return f.Websocket.AuthenticationEnabled, nil
}

// RESTEnabled returns if a specific protocol component is enabled on REST
func (f *Features) RESTEnabled(c Component) (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.REST == nil {
		return false, errRESTIsNil
	}
	return f.REST.checkComponent(c, false)
}

// WebsocketEnabled returns if a specific protocol component is enabled on
// Websocket
func (f *Features) WebsocketEnabled(c Component) (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.Websocket == nil {
		return false, errWebsocketIsNil
	}
	return f.Websocket.checkComponent(c, false)
}

// RESTSupports returns if a specific protocol component is supported by REST
func (f *Features) RESTSupports(c Component) (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.REST == nil {
		return false, errRESTIsNil
	}
	return f.REST.checkComponent(c, true)
}

// WebsocketSupports returns if a specific protocol component is supported by
// Websocket
func (f *Features) WebsocketSupports(c Component) (bool, error) {
	f.RLock()
	defer f.RUnlock()
	if f.Websocket == nil {
		return false, errWebsocketIsNil
	}
	return f.Websocket.checkComponent(c, true)
}

// UnmarshalJSON conforms feature type to the unmarshler interface
func (f *Features) UnmarshalJSON(payload []byte) error {
	*f = Features{}
	return json.Unmarshal(payload, &f.Protocols)
}

// MarshalJSON conforms feature type to the marshaler interface
func (f *Features) MarshalJSON() ([]byte, error) {
	f.Lock()
	defer f.Unlock()
	return json.Marshal(f.Protocols)
}
