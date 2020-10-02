package stream

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	URL                        string
	DedicatedAuthenticatedConn bool
	AllowableAssets            asset.Items
}

// // ConnectionConfig defines a singular connection configuration
// type ConnectionConfig struct {
// 	DedicatedAuth bool
// }
