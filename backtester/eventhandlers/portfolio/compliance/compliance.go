package compliance

import (
	"fmt"
	"time"
)

// AddSnapshot creates a snapshot in time of the orders placed to allow for finer detail tracking
// and to protect against anything modifying order details elsewhere
func (m *Manager) AddSnapshot(orders []SnapshotOrder, t time.Time, offset int64, overwriteExisting bool) error {
	if overwriteExisting {
		if len(m.Snapshots) == 0 {
			return errSnapshotNotFound
		}
		for i := len(m.Snapshots) - 1; i >= 0; i-- {
			if offset == m.Snapshots[i].Offset {
				m.Snapshots[i].Orders = orders
				return nil
			}
		}
		return fmt.Errorf("%w at %v", errSnapshotNotFound, offset)
	}
	m.Snapshots = append(m.Snapshots, Snapshot{
		Orders:    orders,
		Timestamp: t,
		Offset:    offset,
	})

	return nil
}

// GetSnapshotAtTime returns the snapshot of orders a t time
func (m *Manager) GetSnapshotAtTime(t time.Time) (Snapshot, error) {
	for i := len(m.Snapshots) - 1; i >= 0; i-- {
		if t.Equal(m.Snapshots[i].Timestamp) {
			return m.Snapshots[i], nil
		}
	}
	return Snapshot{}, fmt.Errorf("%w at %v", errSnapshotNotFound, t)
}

// GetLatestSnapshot returns the snapshot of t - 1 interval
func (m *Manager) GetLatestSnapshot() Snapshot {
	if len(m.Snapshots) == 0 {
		return Snapshot{
			Offset: 1,
		}
	}

	return m.Snapshots[len(m.Snapshots)-1]
}
