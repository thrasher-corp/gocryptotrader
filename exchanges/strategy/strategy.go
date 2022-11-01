package strategy

import (
	"context"
	"errors"
	"sync"

	"github.com/gofrs/uuid"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

var (
	errStrategyIsNil    = errors.New("strategy is nil")
	errInvalidUUID      = errors.New("invalid UUID")
	errStrategyNotFound = errors.New("strategy not found")
)

// Manager defines strategy management - NOTE: This is a POC wrapper layer for
// management purposes.
type Manager struct {
	strategies map[uuid.UUID]strategy.Requirements
	mu         sync.Mutex
}

// Register stores the current strategy for management
func (m *Manager) Register(strat strategy.Requirements) (uuid.UUID, error) {
	if strat == nil {
		return uuid.Nil, errStrategyIsNil
	}
	id, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.strategies == nil {
		m.strategies = make(map[uuid.UUID]strategy.Requirements)
	}
	m.strategies[id] = strat
	return id, nil
}

// Run runs the applicable strategy
func (m *Manager) Run(ctx context.Context, id uuid.UUID) error {
	if id.IsNil() {
		return errInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	strat, ok := m.strategies[id]
	if !ok {
		return errStrategyNotFound
	}
	return strat.Run(ctx)
}

// RunStream runs then hooks into the strategy and reports events.
func (m *Manager) RunStream(ctx context.Context, id uuid.UUID) (strategy.Reporter, error) {
	if id.IsNil() {
		return nil, errInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	strat, ok := m.strategies[id]
	if !ok {
		return nil, errStrategyNotFound
	}
	err := strat.Run(ctx)
	if err != nil {
		return nil, err
	}
	return strat.GetReporter()
}

// GetAllStrategies returns all strategies if running set true will only return
// operating strategies.
func (m *Manager) GetAllStrategies(running bool) ([]strategy.State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ss := make([]strategy.State, len(m.strategies))
	target := 0
	for id, obj := range m.strategies {
		state, err := obj.GetState()
		if err != nil {
			return nil, err
		}
		state.ID = id // TODO: Change this implementation.
		ss[target] = *state
		target++
	}
	return ss, nil
}

// Stop stops a strategy from executing further orders
func (m *Manager) Stop(id uuid.UUID) error {
	if id.IsNil() {
		return errInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	strat, ok := m.strategies[id]
	if !ok {
		return errStrategyNotFound
	}
	return strat.Stop()
}
