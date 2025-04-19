package versions

import (
	v7 "github.com/thrasher-corp/gocryptotrader/config/versions/v7"
)

func init() {
	Manager.registerVersion(7, &v7.Version{})
}
