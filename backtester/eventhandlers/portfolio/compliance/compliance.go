package compliance

import (
	"fmt"
	"time"
)

// AddSnapshot creates a snapshot in time of the orders placed to allow for finer detail tracking
// and to protect against anything modifying order details elsewhere
func (m *Manager) AddSnapshot(orders []SnapshotOrder, t time.Time, overwriteExisting bool) error {
	if overwriteExisting {
		if len(m.Snapshots) == 0 {
			return fmt.Errorf("%w at %v", errSnapshotNotFound, t)
		}
		// check if its the latest to save time
		if t.Equal(m.Snapshots[len(m.Snapshots)-1].Timestamp) {
			m.Snapshots[len(m.Snapshots)-1].Orders = orders
			return nil
		}
		for i := range m.Snapshots {
			if t.Equal(m.Snapshots[i].Timestamp) {
				m.Snapshots[i].Orders = orders
				return nil
			}
		}
		return fmt.Errorf("%w at %v", errSnapshotNotFound, t)
	}
	m.Snapshots = append(m.Snapshots, Snapshot{
		Orders:    orders,
		Timestamp: t,
	})

	return nil
}

// GetSnapshotAtTime returns the snapshot of orders a t time
func (m *Manager) GetSnapshotAtTime(t time.Time) (Snapshot, error) {
	// check if its the latest to save time
	if t.Equal(m.Snapshots[len(m.Snapshots)-1].Timestamp) {
		return m.Snapshots[len(m.Snapshots)-1], nil
	}

	for i := range m.Snapshots {
		if t.Equal(m.Snapshots[i].Timestamp) {
			return m.Snapshots[i], nil
		}
	}
	return Snapshot{}, fmt.Errorf("%w at %v", errSnapshotNotFound, t)
}

// GetLatestSnapshot returns the snapshot of t - 1 interval
func (m *Manager) GetLatestSnapshot() Snapshot {
	if len(m.Snapshots) == 0 {
		return Snapshot{}
	}

	return m.Snapshots[len(m.Snapshots)-1]
}
