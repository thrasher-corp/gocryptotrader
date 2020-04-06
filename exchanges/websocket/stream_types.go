package stream

// Streamer defines functionality for different exchange streaming services
type Streamer interface {
	Setup()
	Connect() error
	Disconnect() error
	GenerateAuthSubscriptions()
	GenerateMarketDataSubscriptions()
	Subscribe()
	UnSubscribe()
	Refresh()
	GetName() string
	SetProxyAddress(string) error
	GetProxy() string
}

type Manager interface {
	Connector
	Disconnector
}

type Connector interface{}

type Disconnector interface{}

// Websocket defines a websocket connection
type Websocket struct{}
