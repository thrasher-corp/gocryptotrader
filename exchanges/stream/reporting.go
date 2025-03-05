package stream

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// ProcessReporterManager defines an interface for managing ProcessReporter instances across connections, this will
// create a new ProcessReporter instance for each new connection reader as they spawn.
type ProcessReporterManager interface {
	New(conn Connection) ProcessReporter
}

// ProcessReporter defines an interface for reporting processed data from a connection
type ProcessReporter interface {
	// Report logs the processing time for a received data packet and updates metrics.
	// read is the time the data was read from the connection.
	// data is the raw data received from the connection.
	// err is any error that occurred while processing the data.
	Report(read time.Time, data []byte, err error)
	// Close closes the process reporter and handles any cleanup.
	Close()
}

// SetProcessReportManager sets the ProcessReporterManager for the Websocket instance which will be used to create new ProcessReporter instances.
// This will track metrics for processing websocket data.
func (w *Websocket) SetProcessReportManager(m ProcessReporterManager) {
	w.m.Lock()
	defer w.m.Unlock()
	w.processReporter = m
}

// NewDefaultProcessReporterManager returns a new defaultProcessReporterManager instance
func NewDefaultProcessReporterManager() ProcessReporterManager {
	return defaultProcessReporterManager{period: time.Minute}
}

// defaultProcessReporterManager is a default implementation of ProcessReporter
type defaultProcessReporterManager struct{ period time.Duration }

// New returns a new DefaultProcessReporter instance for a connection
func (d defaultProcessReporterManager) New(conn Connection) ProcessReporter {
	reporter := &defaultProcessReporter{ch: make(chan struct{})}
	go reporter.collectMetrics(conn, d.period)
	return reporter
}

// DefaultProcessReporter provides a thread-safe implementation of the ProcessReporter interface.
// It tracks operation metrics, including the number of operations, average processing time, and peak processing time.
type defaultProcessReporter struct {
	operations          int64
	errors              int64
	totalProcessingTime time.Duration
	peakProcessingTime  time.Duration
	peakCause           []byte
	ch                  chan struct{}
	m                   sync.Mutex
}

// Report logs the processing time for a received data packet and updates metrics.
func (r *defaultProcessReporter) Report(read time.Time, data []byte, err error) {
	processingDuration := time.Since(read)
	r.m.Lock()
	defer r.m.Unlock()
	r.operations++
	if err != nil {
		r.errors++
	}
	r.totalProcessingTime += processingDuration
	if processingDuration > r.peakProcessingTime {
		r.peakProcessingTime = processingDuration
		r.peakCause = data
	}
}

// Close closes the process reporter
func (r *defaultProcessReporter) Close() {
	r.m.Lock()
	close(r.ch)
	r.m.Unlock()
}

// collectMetrics runs in a separate goroutine to periodically log aggregated metrics.
func (r *defaultProcessReporter) collectMetrics(conn Connection, period time.Duration) {
	if period == 0 {
		panic("period duration for collecting metrics must be greater than 0")
	}
	timer := time.NewTimer(time.Until(time.Now().Truncate(period).Add(period)))
	defer timer.Stop()

	for {
		select {
		case <-r.ch:
			return
		case <-timer.C:
			r.m.Lock()
			if r.operations > 0 {
				avgOperationsPerSecond := float64(r.operations) / 60
				avgProcessingTime := r.totalProcessingTime / time.Duration(r.operations)
				peakTime := r.peakProcessingTime
				peakCause := r.peakCause
				errors := r.errors
				// Reset metrics for the next interval.
				r.operations, r.totalProcessingTime, r.peakProcessingTime, r.peakCause, r.errors = 0, 0, 0, nil, 0
				r.m.Unlock()
				if len(peakCause) > 100 {
					peakCause = append(peakCause[:100], []byte("...")...)
				}
				// Log metrics outside of the critical section to avoid blocking other threads.
				log.Debugf(log.WebsocketMgr, "Connection: %v Operations/Second: %.2f, Avg Processing/Operation: %v, Errors: %v Peak: %v Cause: %v", conn.GetURL(), avgOperationsPerSecond, avgProcessingTime, errors, peakTime, string(peakCause))
			} else {
				r.m.Unlock()
			}
			timer.Reset(time.Until(time.Now().Truncate(period).Add(period)))
		}
	}
}
