package versions

import (
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
	v2 "github.com/thrasher-corp/gocryptotrader/config/versions/v2"
)

func init() {
	Manager.registerVersion(&v0.Version{})
	Manager.registerVersion(&v1.Version{})
	Manager.registerVersion(&v2.Version{})
}
