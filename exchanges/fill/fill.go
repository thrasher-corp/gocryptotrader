package fill

import "errors"

// ErrFeedDisabled is an error that indicates the fill feed is disabled
var ErrFeedDisabled = errors.New("fill feed disabled")

// Setup sets up the fill processor
func (f *Fills) Setup(fillsFeedEnabled bool, c chan any) {
	f.dataHandler = c
	f.fillsFeedEnabled = fillsFeedEnabled
}

// Update disseminates fill data through the data channel if so
// configured
func (f *Fills) Update(data ...Data) error {
	if len(data) == 0 {
		// nothing to do
		return nil
	}

	if !f.fillsFeedEnabled {
		return ErrFeedDisabled
	}

	f.dataHandler <- data

	return nil
}
