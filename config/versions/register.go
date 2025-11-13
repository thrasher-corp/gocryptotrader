package versions

import (
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
	v10 "github.com/thrasher-corp/gocryptotrader/config/versions/v10"
	v2 "github.com/thrasher-corp/gocryptotrader/config/versions/v2"
	v3 "github.com/thrasher-corp/gocryptotrader/config/versions/v3"
	v4 "github.com/thrasher-corp/gocryptotrader/config/versions/v4"
	v5 "github.com/thrasher-corp/gocryptotrader/config/versions/v5"
	v6 "github.com/thrasher-corp/gocryptotrader/config/versions/v6"
	v7 "github.com/thrasher-corp/gocryptotrader/config/versions/v7"
	v8 "github.com/thrasher-corp/gocryptotrader/config/versions/v8"
	v9 "github.com/thrasher-corp/gocryptotrader/config/versions/v9"
)

func init() {
	Manager.registerVersion(0, &v0.Version{})
	Manager.registerVersion(1, &v1.Version{})
	Manager.registerVersion(2, &v2.Version{})
	Manager.registerVersion(3, &v3.Version{})
	Manager.registerVersion(4, &v4.Version{})
	Manager.registerVersion(5, &v5.Version{})
	Manager.registerVersion(6, &v6.Version{})
	Manager.registerVersion(7, &v7.Version{})
	Manager.registerVersion(8, &v8.Version{})
	Manager.registerVersion(9, &v9.Version{})
	Manager.registerVersion(10, &v10.Version{})
}
