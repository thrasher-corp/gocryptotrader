package compliance

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// AddSnapshot creates a snapshot in time of the orders placed to allow for finer detail tracking
// and to protect against anything modifying order details elsewhere
func (m *Manager) AddSnapshot(orders []SnapshotOrder, t time.Time, force bool) error {
	found := false
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Time) {
			found = true
			if force {
				m.Snapshots[i].Orders = orders
			} else {
				return errors.New("already exists buttts")
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

// GetSnapshot returns the snapshot of orders a t time
func (m *Manager) GetSnapshot(t time.Time) (Snapshot, error) {
	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Time) {
			return m.Snapshots[i], nil
		}
	}
	return Snapshot{}, errors.New("not found")
}

func (m *Manager) SetInterval(i kline.Interval) {
	m.Interval = i
}

// GetPreviousSnapshot returns the snapshot of t - 1 interval
func (m *Manager) GetPreviousSnapshot(t time.Time) Snapshot {
	for i := range m.Snapshots {
		if t.Add(-m.Interval.Duration()).Equal(m.Snapshots[i].Time) {
			return m.Snapshots[i]
		}
	}
	return Snapshot{
		Time:   t.Add(-m.Interval.Duration()),
		Orders: []SnapshotOrder{},
	}
}
