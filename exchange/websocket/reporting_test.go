package websocket

import (
	"context"
	"testing"
	"time"
)

type DummyConnection struct {
	Connection
	ch chan []byte
}

func (d *DummyConnection) ReadMessage() Response {
	return Response{Raw: <-d.ch}
}

func (d *DummyConnection) Push(data []byte) {
	d.ch <- data
}

func (d *DummyConnection) GetURL() string {
	return "ws://test"
}

func ProcessWithSomeSweetLag(context.Context, []byte) error {
	time.Sleep(time.Millisecond)
	return nil
}

func TestDefaultProcessReporter(t *testing.T) {
	t.Parallel()
	w := &Manager{}
	reporterManager := defaultProcessReporterManager{period: time.Millisecond * 10}
	w.SetProcessReportManager(&reporterManager)
	conn := &DummyConnection{ch: make(chan []byte)}
	w.Wg.Add(1)
	go w.Reader(context.Background(), conn, ProcessWithSomeSweetLag)

	for range 100 {
		conn.Push([]byte("test"))
	}
	conn.Push(nil)
}
