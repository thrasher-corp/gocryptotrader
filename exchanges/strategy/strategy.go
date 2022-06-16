package strategy

import (
	"context"
	"errors"
	"sync"

	"github.com/gofrs/uuid"
)

// StrategyManager defines management processes
type StrategyManager struct {
	Strategies []Requirement
	m          sync.Mutex
}

// Requirement defines baseline functionality for strategy implementation to
// GCT.
type Requirement interface {
	GetBase() (*Base, error)
	GetContext() (context.Context, error)
	IsRunning() (bool, error)
	Stop() error
	Pause() error
	Report() error
	Run() error
}

// Register
func (sm *StrategyManager) Register(obj Requirement) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// Run runs the applicable strategy
func (sm *StrategyManager) Run(id uuid.UUID) error {
	return nil
}

// Backtest runs the applicable strategy through in sample and out of sample
// data.
func (sm *StrategyManager) Backtest(id uuid.UUID) error {
	return nil
}

func (sm *StrategyManager) Pause(id uuid.UUID) error {
	return nil
}

func (sm *StrategyManager) Unpause(id uuid.UUID) error {
	return nil
}

func (sm *StrategyManager) Stop(id uuid.UUID) error {
	return nil
}

// StreamAction hooks into the strategy and reports events
func (sm *StrategyManager) StreamAction(id uuid.UUID) (chan interface{}, error) {
	return nil, nil
}

// GetState returns a reportable history of actions; pnl, errors, etc.
func (sm *StrategyManager) IsRunning(id uuid.UUID) (chan interface{}, error) {
	return nil, nil
}

// Base defines the base strategy application for quick implementation and usage.
type Base struct {
	Context   context.Context
	History   []interface{} // Logs/trades
	Awareness []interface{} // Systems affected
	Verbose   bool
}

var errBaseNotFound = errors.New("strategy base not found")

// GetBase returns the strategy base
func (b *Base) GetBase() (*Base, error) {
	if b == nil {
		return nil, errBaseNotFound
	}
	return b, nil
}
