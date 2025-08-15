package quickspy

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corporation/futures-mon/config"
)

var cfg config.Config

func TestMain(m *testing.M) {
	// load config
	// get API credentials
	// use credentials from there instead of here
	m.Run()
}

func TestNewQuickSpy(t *testing.T) {
	t.Parallel()
	qs, err := NewQuickSpy(types.CredKey{
		Key: key.NewExchangePairAssetKey("gateio", asset.Spot, currency.NewBTCUSDT()),
		//Credentials: account.Credentials{
		//		PremiumKey:    "",
		//		Secret: "",
		//	},
	}, []FocusData{
		{
			Type:         OrderBookFocusType,
			Enabled:      true,
			UseWebsocket: false,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tt := time.Now()
	err = qs.Run()
	if err != nil {
		t.Fatal(err)
	}

	focus, _, err := qs.Focuses.GetByKey(OrderBookFocusType)
	if err != nil {
		t.Fatal(err)
	}
	<-focus.HasBeenSuccessfulChan
	t.Log(time.Since(tt))
	focus.m.RLock()
	t.Log(qs.Data.OB)
	focus.m.RUnlock()
}
