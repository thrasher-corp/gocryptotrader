package engine

// func TestNewCurrencyPairSyncer(t *testing.T) {
// 	t.Skip()

// 	if Bot == nil {
// 		Bot = new(Engine)
// 	}
// 	Bot.Config = &config.Cfg
// 	err := Bot.Config.LoadConfig("", true)
// 	if err != nil {
// 		t.Fatalf("TestNewExchangeSyncer: Failed to load config: %s", err)
// 	}

// 	Bot.Settings.DisableExchangeAutoPairUpdates = true
// 	Bot.Settings.Verbose = true
// 	Bot.Settings.EnableExchangeWebsocketSupport = true

// 	SetupExchanges()

// 	if err != nil {
// 		t.Log("failed to start exchange syncer")
// 	}

// 	Bot.ExchangeCurrencyPairManager, err = NewCurrencyPairSyncer(CurrencyPairSyncerConfig{
// 		SyncTicker:       true,
// 		SyncOrderbook:    false,
// 		SyncTrades:       false,
// 		SyncContinuously: false,
// 	})
// 	if err != nil {
// 		t.Errorf("NewCurrencyPairSyncer failed: err %s", err)
// 	}

// 	Bot.ExchangeCurrencyPairManager.Start()
// 	time.Sleep(time.Second * 15)
// 	Bot.ExchangeCurrencyPairManager.Stop()
// }
