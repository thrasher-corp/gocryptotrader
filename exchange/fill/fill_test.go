package fill

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
)

// TestSetup tests the setup function of the Fills struct
func TestSetup(t *testing.T) {
	fill := &Fills{}
	fill.Setup(true, stream.NewRelay(1))

	if fill.dataHandler == nil {
		t.Error("expected dataHandler to be set")
	}

	if !fill.fillsFeedEnabled {
		t.Error("expected fillsFeedEnabled to be true")
	}
}

// TestUpdateDisabledFeed tests the Update function when fillsFeedEnabled is false
func TestUpdateDisabledFeed(t *testing.T) {
	fill := Fills{dataHandler: stream.NewRelay(1), fillsFeedEnabled: false}

	// Send a test data to the Update function
	testData := Data{Timestamp: time.Now(), Price: 15.2, Amount: 3.2}
	assert.ErrorIs(t, fill.Update(testData), ErrFeedDisabled)

	select {
	case <-fill.dataHandler.C:
		t.Errorf("Expected no data on channel, got data")
	default:
		// nothing to do
	}
}

// TestUpdate tests the Update function of the Fills struct.
func TestUpdate(t *testing.T) {
	fill := &Fills{dataHandler: stream.NewRelay(1), fillsFeedEnabled: true}
	receivedData := Data{Timestamp: time.Now(), Price: 15.2, Amount: 3.2}
	if err := fill.Update(receivedData); err != nil {
		t.Errorf("Update returned error %v", err)
	}

	select {
	case data := <-fill.dataHandler.C:
		dataSlice, ok := data.Data.([]Data)
		if !ok {
			t.Errorf("expected []Data, got %T", data)
		}

		if len(dataSlice) != 1 || dataSlice[0] != receivedData {
			t.Errorf("expected data to be sent through channel")
		}
	default:
		t.Errorf("No data sent to channel")
	}
}

// TestUpdateNoData tests the Update function with no Data objects
func TestUpdateNoData(t *testing.T) {
	fill := &Fills{dataHandler: stream.NewRelay(1), fillsFeedEnabled: true}
	if err := fill.Update(); err != nil {
		t.Errorf("Update returned error %v", err)
	}

	select {
	case <-fill.dataHandler.C:
		t.Errorf("Expected no data on channel, got data")
	default:
		// pass, nothing to do
	}
}

// TestUpdateMultipleData tests the Update function with multiple Data objects
func TestUpdateMultipleData(t *testing.T) {
	fill := &Fills{dataHandler: stream.NewRelay(2), fillsFeedEnabled: true}
	receivedData := Data{Timestamp: time.Now(), Price: 15.2, Amount: 3.2}
	receivedData2 := Data{Timestamp: time.Now(), Price: 18.2, Amount: 9.0}
	if err := fill.Update(receivedData, receivedData2); err != nil {
		t.Errorf("Update returned error %v", err)
	}

	select {
	case data := <-fill.dataHandler.C:
		dataSlice, ok := data.Data.([]Data)
		if !ok {
			t.Errorf("expected []Data, got %T", data)
		}

		if len(dataSlice) != 2 || dataSlice[0] != receivedData || dataSlice[1] != receivedData2 {
			t.Errorf("expected data to be sent through channel")
		}
	default:
		t.Errorf("No data sent to channel")
	}
}
