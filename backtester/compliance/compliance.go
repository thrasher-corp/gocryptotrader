package compliance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// AddSnapshot
func (m *Manager) AddSnapshot(orders []order.Detail, t time.Time, force bool) error {
	found := false
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Time) {
			found = true
			if force {
				m.Snapshots[i].Orders = orders
			} else {
				return errors.New("already exists")
			}
		}
	}
	if !found {
		m.Snapshots = append(m.Snapshots, Snapshot{
			Orders: orders,
			Time:   t,
		})
	}

	return nil
}

func (m *Manager) GetSnapshot(t time.Time) (Snapshot, error) {
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Time) {
			return m.Snapshots[i], nil
		}
	}
	return Snapshot{}, errors.New("not found")
}
