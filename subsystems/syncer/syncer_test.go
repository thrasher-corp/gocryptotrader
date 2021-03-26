package syncer

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

func TestNewCurrencyPairSyncer(t *testing.T) {
	t.Skip()

	if engine.Bot == nil {
		engine.Bot = new(engine.Engine)
	}
	engine.Bot.Config = &config.Cfg
	err := engine.Bot.Config.LoadConfig("", true)
	if err != nil {
		t.Fatalf("TestNewExchangeSyncer: Failed to load config: %s", err)
	}

	engine.Bot.Settings.DisableExchangeAutoPairUpdates = true
	engine.Bot.Settings.EnableExchangeWebsocketSupport = true

	err = engine.Bot.SetupExchanges()
	if err != nil {
		t.Log(err)
	}

	engine.Bot.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(CurrencyPairSyncerConfig{
		SyncTicker:       true,
		SyncOrderbook:    false,
		SyncTrades:       false,
		SyncContinuously: false,
	})
	if err != nil {
		t.Errorf("NewCurrencyPairSyncer failed: err %s", err)
	}

	engine.Bot.ExchangeCurrencyPairManager.Start()
	time.Sleep(time.Second * 15)
	engine.Bot.ExchangeCurrencyPairManager.Stop()
}
