package versions

import (
	v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"
	v1 "github.com/thrasher-corp/gocryptotrader/config/versions/v1"
	v10 "github.com/thrasher-corp/gocryptotrader/config/versions/v10"
	v11 "github.com/thrasher-corp/gocryptotrader/config/versions/v11"
	v12 "github.com/thrasher-corp/gocryptotrader/config/versions/v12"
	v13 "github.com/thrasher-corp/gocryptotrader/config/versions/v13"
	v2 "github.com/thrasher-corp/gocryptotrader/config/versions/v2"
	v3 "github.com/thrasher-corp/gocryptotrader/config/versions/v3"
	v4 "github.com/thrasher-corp/gocryptotrader/config/versions/v4"
	v5 "github.com/thrasher-corp/gocryptotrader/config/versions/v5"
	v6 "github.com/thrasher-corp/gocryptotrader/config/versions/v6"
	v7 "github.com/thrasher-corp/gocryptotrader/config/versions/v7"
	v8 "github.com/thrasher-corp/gocryptotrader/config/versions/v8"
	v9 "github.com/thrasher-corp/gocryptotrader/config/versions/v9"
)

func newManager() *manager {
	m := new(manager)
	m.registerVersion(0, &v0.Version{})
	m.registerVersion(1, &v1.Version{})
	m.registerVersion(2, &v2.Version{})
	m.registerVersion(3, &v3.Version{})
	m.registerVersion(4, &v4.Version{})
	m.registerVersion(5, &v5.Version{})
	m.registerVersion(6, &v6.Version{})
	m.registerVersion(7, &v7.Version{})
	m.registerVersion(8, &v8.Version{})
	m.registerVersion(9, &v9.Version{})
	m.registerVersion(10, &v10.Version{})
	m.registerVersion(11, &v11.Version{})
	m.registerVersion(12, &v12.Version{})
	m.registerVersion(13, &v13.Version{})
	return m
}
