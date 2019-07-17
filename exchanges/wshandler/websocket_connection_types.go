package wshandler

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// IWebsocketConnection an interface for testing
type IWebsocketConnection interface {
	AddResponseWithID(id int64, data []byte)
	Dial(dialer *websocket.Dialer, headers http.Header) error
	SendMessage(data interface{}) error
	SendMessageReturnResponse(id int64, request interface{}) ([]byte, error)
	WaitForResult(id int64, wg *sync.WaitGroup)
	ReadMessage() (WebsocketResponse, error)
	GenerateMessageID(useNano bool) int64
}

// WebsocketConnection contains all the datas needed to send a message to a WS
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
	// These are the requests and responses
	IDResponses map[int64][]byte
}
