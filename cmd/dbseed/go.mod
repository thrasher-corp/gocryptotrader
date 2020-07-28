module github.com/thrasher-corp/gocryptotrader/cmd/dbseed

go 1.14

require (
	github.com/thrasher-corp/gocryptotrader v0.0.0-20200724031809-14c72c9c6b45
	github.com/thrasher-corp/sqlboiler v1.0.1-0.20191001234224-71e17f37a85e
	github.com/urfave/cli/v2 v2.2.0
)

replace github.com/thrasher-corp/gocryptotrader => ./../../
