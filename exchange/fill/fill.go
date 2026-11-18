package fill

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
)

// ErrFeedDisabled is an error that indicates the fill feed is disabled
var ErrFeedDisabled = errors.New("fill feed disabled")

// Setup sets up the fill processor
func (f *Fills) Setup(fillsFeedEnabled bool, c *stream.Relay) {
	f.dataHandler = c
	f.fillsFeedEnabled = fillsFeedEnabled
}

// Update disseminates fill data through the data channel if so
// configured
func (f *Fills) Update(data ...Data) error {
	ctx := context.TODO()
	if len(data) == 0 {
		// nothing to do
		return nil
	}

	if !f.fillsFeedEnabled {
		return ErrFeedDisabled
	}

	return f.dataHandler.Send(ctx, data)
}
