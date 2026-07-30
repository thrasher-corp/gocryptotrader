package htx

import (
	"os"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
)

func init() {
	if os.Getenv("GCT_HTX_RUN_LIVE_TESTS") != "true" {
		apiCredentials = new(accounts.Credentials)
	}
}
