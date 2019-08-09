package wshandler

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebsocketConnection contains all the data needed to send a message to a WS
type WebsocketConnection struct {
	sync.Mutex
	Verbose      bool
	RateLimit    float64
	ExchangeName string
	URL          string
	ProxyURL     string
	Wg           sync.WaitGroup
	Connection   *websocket.Conn
	Shutdown     chan struct{}
	// These are the request IDs and the corresponding response JSON
	IDResponses          map[int64][]byte
	ResponseCheckTimeout time.Duration
	ResponseMaxLimit     time.Duration
}
