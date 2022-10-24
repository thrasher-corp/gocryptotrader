package compliance

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestAddSnapshot(t *testing.T) {
	t.Parallel()
	m := Manager{}
	tt := time.Now()
	err := m.AddSnapshot(&Snapshot{}, true)
	if !errors.Is(err, errSnapshotNotFound) {
		t.Errorf("received: %v, expected: %v", err, errSnapshotNotFound)
	}

	err = m.AddSnapshot(&Snapshot{
		Timestamp: tt,
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(m.Snapshots) != 1 {
		t.Error("expected 1")
	}
	err = m.AddSnapshot(&Snapshot{
		Timestamp: tt,
	}, true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(m.Snapshots) != 1 {
		t.Error("expected 1")
	}
}

func TestGetSnapshotAtTime(t *testing.T) {
	t.Parallel()
	m := Manager{}
	tt := time.Now()
	err := m.AddSnapshot(&Snapshot{Offset: 0,
		Timestamp: tt,
		Orders: []SnapshotOrder{
			{
				Order: &gctorder.Detail{
					Price: 1337,
				},
				ClosePrice: decimal.NewFromInt(1337),
			},
		},
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	var snappySnap Snapshot
	snappySnap, err = m.GetSnapshotAtTime(tt)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(snappySnap.Orders) == 0 {
		t.Fatal("expected an order")
	}
	if !snappySnap.Orders[0].ClosePrice.Equal(decimal.NewFromInt(1337)) {
		t.Error("expected decimal.NewFromInt(1337)")
	}
	if !snappySnap.Timestamp.Equal(tt) {
		t.Errorf("expected %v, received %v", tt, snappySnap.Timestamp)
	}

	_, err = m.GetSnapshotAtTime(time.Now().Add(time.Hour))
	if !errors.Is(err, errSnapshotNotFound) {
		t.Errorf("received: %v, expected: %v", err, errSnapshotNotFound)
	}
}

func TestGetLatestSnapshot(t *testing.T) {
	t.Parallel()
	m := Manager{}
	snappySnap := m.GetLatestSnapshot()
	if !snappySnap.Timestamp.IsZero() {
		t.Error("expected blank snapshot")
	}
	tt := time.Now()
	err := m.AddSnapshot(&Snapshot{
		Timestamp: tt,
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	err = m.AddSnapshot(&Snapshot{
		Offset:    1,
		Timestamp: tt.Add(time.Hour),
		Orders:    nil,
	}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	snappySnap = m.GetLatestSnapshot()
	if snappySnap.Timestamp.Equal(tt) {
		t.Errorf("expected %v", tt.Add(time.Hour))
	}
	if !snappySnap.Timestamp.Equal(tt.Add(time.Hour)) {
		t.Errorf("expected %v", tt.Add(time.Hour))
	}
}
