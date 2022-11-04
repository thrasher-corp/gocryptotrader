package common

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const SimulationTag = "SIMULATION"

// Requirement define baseline functionality for strategy implementation to GCT
type Requirement interface {
	Run(ctx context.Context) error
	Stop() error
	GetReporter() (Reporter, error)
	GetState() (*State, error)
}

// State defines basic identification for strategy
type State struct {
	ID         uuid.UUID
	Registered time.Time
	Exchange   string
	Pair       currency.Pair
	Asset      asset.Item
	Strategy   string
	Running    bool
}
