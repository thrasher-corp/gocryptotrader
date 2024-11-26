package stream

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// ProcessReporterManager defines an interface for managing ProcessReporter instances across connections, this will
// create a new ProcessReporter instance for each new connection reader.
type ProcessReporterManager interface {
	New() ProcessReporter
}

// DefaultProcessReporter is a default implementation of ProcessReporter
type DefaultProcessReporterManager struct{}

// New returns a new DefaultProcessReporter instance for a connection
func (d DefaultProcessReporterManager) New() ProcessReporter { return &DefaultProcessReporter{} }

// ProcessReporter defines an interface for reporting processed data from a connection
type ProcessReporter interface {
	Report(conn Connection, read time.Time, data []byte)
}

// DefaultProcessReporter provides a thread-safe implementation of the ProcessReporter interface.
// It tracks operation metrics, including the number of operations, average processing time, and peak processing time.
type DefaultProcessReporter struct {
	operations          int64
	totalProcessingTime time.Duration
	peakProcessingTime  time.Duration
	ch                  chan struct{}
	m                   sync.Mutex
}

// Report logs the processing time for a received data packet and updates metrics.
// If `data` is nil, the reporter shuts down its metrics collection routine.
func (r *DefaultProcessReporter) Report(conn Connection, read time.Time, data []byte) {
	processingDuration := time.Since(read)

	r.m.Lock()
	defer r.m.Unlock()
	if data == nil {
		if r.ch != nil {
			close(r.ch)
		}
		return
	}

	if r.ch == nil {
		r.ch = make(chan struct{})
		go r.collectMetrics(conn)
	}

	r.operations++
	r.totalProcessingTime += processingDuration
	if processingDuration > r.peakProcessingTime {
		r.peakProcessingTime = processingDuration
	}
}

// collectMetrics runs in a separate goroutine to periodically log aggregated metrics.
func (r *DefaultProcessReporter) collectMetrics(conn Connection) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ch:
			return
		case <-ticker.C:
			r.m.Lock()
			if r.operations > 0 {
				avgOperationsPerSecond := r.operations / 60
				avgProcessingTime := r.totalProcessingTime / time.Duration(r.operations)
				peakTime := r.peakProcessingTime

				// Reset metrics for the next interval.
				r.operations, r.totalProcessingTime, r.peakProcessingTime = 0, 0, 0

				r.m.Unlock()

				// Log metrics outside of the critical section to avoid blocking other threads.
				log.Debugf(log.WebsocketMgr, "%v: Operations/Second: %d, Avg Processing/Operation: %v, Peak: %v", conn.GetURL(), avgOperationsPerSecond, avgProcessingTime, peakTime)
			} else {
				r.m.Unlock()
			}
		}
	}
}

// SetProcessReportManager sets the ProcessReporterManager for the Websocket instance which will be used to create new ProcessReporter instances.
// This will track metrics for processing websocket data.
func (w *Websocket) SetProcessReportManager(m ProcessReporterManager) {
	w.m.Lock()
	defer w.m.Unlock()
	w.processReporter = m
}
