package strategy

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Manager defines strategy management - NOTE: This is a POC wrapper layer for
// management purposes.
type Manager struct {
	strategies map[uuid.UUID]strategy.Requirements
	mu         sync.Mutex
}

// Register stores the current strategy for management
func (m *Manager) Register(st strategy.Requirements) (uuid.UUID, error) {
	if st == nil {
		return uuid.Nil, strategy.ErrIsNil
	}
	id, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, err
	}

	err = st.LoadID(id)
	if err != nil {
		return uuid.Nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.strategies == nil {
		m.strategies = make(map[uuid.UUID]strategy.Requirements)
	}
	log.Debugf(log.Strategy, "ID: [%s] has been registered. Details: %s", st.GetID(), st.GetDescription())
	m.strategies[id] = st
	st.ReportRegister()
	return id, nil
}

// Run runs the applicable strategy
func (m *Manager) Run(ctx context.Context, id uuid.UUID) error {
	if id.IsNil() {
		return strategy.ErrInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.strategies[id]
	if !ok {
		return strategy.ErrNotFound
	}
	log.Debugf(log.Strategy, "ID: [%s] has been run.", st.GetID())
	return st.Run(ctx, st)
}

// RunStream runs then hooks into the strategy and reports events.
func (m *Manager) RunStream(ctx context.Context, id uuid.UUID, verbose bool) (<-chan *strategy.Report, error) {
	if id.IsNil() {
		return nil, strategy.ErrInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.strategies[id]
	if !ok {
		return nil, strategy.ErrNotFound
	}
	err := st.Run(ctx, st)
	if err != nil {
		return nil, err
	}
	return st.GetReporter(verbose)
}

// GetAllStrategies returns all strategies if running set true will only return
// operating strategies.
func (m *Manager) GetAllStrategies(running bool) ([]strategy.Details, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	loaded := make([]strategy.Details, 0, len(m.strategies))
	for _, obj := range m.strategies {
		details, err := obj.GetDetails()
		if err != nil {
			return nil, err
		}
		if running && !details.Running {
			continue
		}
		loaded = append(loaded, *details)
	}
	return loaded, nil
}

// Stop stops a strategy from executing further orders
func (m *Manager) Stop(id uuid.UUID) error {
	if id.IsNil() {
		return strategy.ErrInvalidUUID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	strat, ok := m.strategies[id]
	if !ok {
		return strategy.ErrNotFound
	}
	log.Debugf(log.Strategy, "ID: [%s] trading operations have been stopped.", strat.GetID())
	return strat.Stop()
}
