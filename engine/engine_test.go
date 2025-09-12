package engine

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
)

// blockedCIExchanges are exchanges that are not able to be tested on CI
var blockedCIExchanges = []string{
	"binance", // binance API is banned from executing within the US where github Actions is ran
	"bybit",   // bybit API is banned from executing within the US where github Actions is ran
}

func isCITest() bool {
	return os.Getenv("CI") == "true"
}

func TestLoadConfigWithSettings(t *testing.T) {
	empty := ""
	somePath := "somePath"
	// Clean up after the tests
	defer os.RemoveAll(somePath)
	tests := []struct {
		name     string
		flags    []string
		settings *Settings
		want     *string
		wantErr  bool
	}{
		{
			name: "invalid file",
			settings: &Settings{
				ConfigFile: "nonExistent.json",
			},
			wantErr: true,
		},
		{
			name: "test file",
			settings: &Settings{
				ConfigFile:   config.TestFile,
				CoreSettings: CoreSettings{EnableDryRun: true},
			},
			want:    &empty,
			wantErr: false,
		},
		{
			name:  "data dir in settings overrides config data dir",
			flags: []string{"datadir"},
			settings: &Settings{
				ConfigFile:   config.TestFile,
				DataDir:      somePath,
				CoreSettings: CoreSettings{EnableDryRun: true},
			},
			want:    &somePath,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare the 'flags'
			flagSet := make(map[string]bool)
			for _, v := range tt.flags {
				flagSet[v] = true
			}
			// Run the test
			got, err := loadConfigWithSettings(tt.settings, flagSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigWithSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil || tt.want != nil {
				if (got == nil && tt.want != nil) || (got != nil && tt.want == nil) {
					t.Errorf("loadConfigWithSettings() = is nil %v, want nil %v", got == nil, tt.want == nil)
				} else if got.DataDirectory != *tt.want {
					t.Errorf("loadConfigWithSettings() = %v, want %v", got.DataDirectory, *tt.want)
				}
			}
		})
	}
}

func TestStartStopDoesNotCausePanic(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	botOne, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		CoreSettings: CoreSettings{EnableDryRun: true},
		DataDir:      tempDir,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	botOne.Settings.EnableGRPCProxy = false
	for i := range botOne.Config.Exchanges {
		if botOne.Config.Exchanges[i].Name != testExchange {
			// there is no need to load all exchanges for this test
			botOne.Config.Exchanges[i].Enabled = false
		}
	}
	if err = botOne.Start(); err != nil {
		t.Error(err)
	}

	botOne.Stop()
}

var enableExperimentalTest = false

func TestStartStopTwoDoesNotCausePanic(t *testing.T) {
	t.Parallel()
	if !enableExperimentalTest {
		t.Skip("test is functional, however does not need to be included in go test runs")
	}
	tempDir := t.TempDir()
	tempDir2 := t.TempDir()
	botOne, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		CoreSettings: CoreSettings{EnableDryRun: true},
		DataDir:      tempDir,
	}, nil)
	if err != nil {
		t.Error(err)
	}
	botOne.Settings.EnableGRPCProxy = false

	botTwo, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		CoreSettings: CoreSettings{EnableDryRun: true},
		DataDir:      tempDir2,
	}, nil)
	if err != nil {
		t.Error(err)
	}
	botTwo.Settings.EnableGRPCProxy = false

	if err = botOne.Start(); err != nil {
		t.Error(err)
	}
	if err = botTwo.Start(); err != nil {
		t.Error(err)
	}

	botOne.Stop()
	botTwo.Stop()
}

func TestGetExchangeByName(t *testing.T) {
	t.Parallel()
	_, err := (*ExchangeManager)(nil).GetExchangeByName("tehehe")
	assert.ErrorIs(t, err, ErrNilSubsystem)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)

	exch.SetDefaults()
	exch.SetEnabled(true)
	err = em.Add(exch)
	require.NoError(t, err)

	e := &Engine{ExchangeManager: em}

	if !exch.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch.SetEnabled(false)
	bfx, err := e.GetExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	if bfx.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}
	if exch.GetName() != testExchange {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	_, err = e.GetExchangeByName("Asdasd")
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}

func TestUnloadExchange(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err)

	exch.SetDefaults()
	exch.SetEnabled(true)
	err = em.Add(exch)
	require.NoError(t, err)

	e := &Engine{
		ExchangeManager: em,
		Config:          &config.Config{Exchanges: []config.Exchange{{Name: testExchange}}},
	}
	err = e.UnloadExchange("asdf")
	assert.ErrorIs(t, err, config.ErrExchangeNotFound)

	err = e.UnloadExchange(testExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = e.UnloadExchange(testExchange)
	assert.ErrorIs(t, err, ErrExchangeNotFound)
}

func TestDryRunParamInteraction(t *testing.T) {
	t.Parallel()
	bot := &Engine{
		ExchangeManager: NewExchangeManager(),
		Settings:        Settings{},
		Config: &config.Config{
			Exchanges: []config.Exchange{
				{
					Name:                    testExchange,
					WebsocketTrafficTimeout: time.Second,
				},
			},
		},
	}
	err := bot.LoadExchange(testExchange)
	assert.NoError(t, err, "LoadExchange should not error")

	exchCfg, err := bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}
	if exchCfg.Verbose {
		t.Error("verbose should have been disabled")
	}
	if err = bot.UnloadExchange(testExchange); err != nil {
		t.Error(err)
	}

	// Now set dryrun mode to true,
	// enable exchange verbose mode and verify that verbose mode
	// will be set on Bitfinex
	bot.Settings.EnableDryRun = true
	bot.Settings.CheckParamInteraction = true
	bot.Settings.EnableExchangeVerbose = true

	err = bot.LoadExchange(testExchange)
	assert.NoError(t, err, "LoadExchange should not error")

	exchCfg, err = bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}
	if !bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun should be true and verbose should be true")
	}
}

func TestFlagSetWith(t *testing.T) {
	var isRunning bool
	flags := make(FlagSet)
	// Flag not set default to config
	flags.WithBool("NOT SET", &isRunning, true)
	if !isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, true)
	}
	flags.WithBool("NOT SET", &isRunning, false)
	if isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, false)
	}

	flags["IS SET"] = true
	isRunning = true
	// Flag set true which will override config
	flags.WithBool("IS SET", &isRunning, true)
	if !isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, true)
	}
	flags.WithBool("IS SET", &isRunning, false)
	if !isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, true)
	}

	flags["IS SET"] = true
	isRunning = false
	// Flag set false which will override config
	flags.WithBool("IS SET", &isRunning, true)
	if isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, false)
	}
	flags.WithBool("IS SET", &isRunning, false)
	if isRunning {
		t.Fatalf("received: '%v' but expected: '%v'", isRunning, false)
	}
}

func TestRegisterWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	var e *Engine
	err := e.RegisterWebsocketDataHandler(nil, false)
	require.ErrorIs(t, err, errNilBot)

	e = &Engine{WebsocketRoutineManager: &WebsocketRoutineManager{}}
	err = e.RegisterWebsocketDataHandler(func(_ string, _ any) error { return nil }, false)
	require.NoError(t, err)
}

func TestSetDefaultWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	var e *Engine
	err := e.SetDefaultWebsocketDataHandler()
	require.ErrorIs(t, err, errNilBot)

	e = &Engine{WebsocketRoutineManager: &WebsocketRoutineManager{}}
	err = e.SetDefaultWebsocketDataHandler()
	require.NoError(t, err)
}

func TestSettingsPrint(t *testing.T) {
	t.Parallel()
	var s *Settings
	s.PrintLoadedSettings()

	s = &Settings{}
	s.PrintLoadedSettings()
}

var unsupportedDefaultConfigExchanges = []string{
	"poloniex", // poloniex has dropped support for the API GCT has implemented //TODO: drop this when supported
}

func TestGetDefaultConfigurations(t *testing.T) {
	t.Parallel()
	em := NewExchangeManager()
	for i := range exchange.Exchanges {
		name := strings.ToLower(exchange.Exchanges[i])
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			exch, err := em.NewExchangeByName(name)
			require.NoError(t, err, "NewExchangeByName must not error")

			if isCITest() && slices.Contains(blockedCIExchanges, name) {
				t.Skipf("skipping %s due to CI test restrictions", name)
			}

			if slices.Contains(unsupportedDefaultConfigExchanges, name) {
				t.Skipf("skipping %s unsupported", name)
			}

			defaultCfg, err := exchange.GetDefaultConfig(t.Context(), exch)
			require.NoError(t, err, "GetDefaultConfig must not error")
			require.NotNil(t, defaultCfg)
			assert.NotEmpty(t, defaultCfg.Name, "Name should not be empty")
			assert.True(t, defaultCfg.Enabled, "Enabled should have the correct value")

			if exch.SupportsWebsocket() {
				assert.Positive(t, defaultCfg.WebsocketResponseCheckTimeout, "WebsocketResponseCheckTimeout should be positive")
				assert.Positive(t, defaultCfg.WebsocketResponseMaxLimit, "WebsocketResponseMaxLimit should be positive")
				assert.Positive(t, defaultCfg.WebsocketTrafficTimeout, "WebsocketTrafficTimeout should be positive")
			}

			require.NoError(t, exch.Setup(defaultCfg), "Setup must not error")
		})
	}
}

func TestSetupExchanges(t *testing.T) {
	t.Parallel()

	t.Run("No enabled exchanges", func(t *testing.T) {
		t.Parallel()
		e := &Engine{
			Config: &config.Config{Exchanges: []config.Exchange{{Name: testExchange}}},
		}
		assert.ErrorIs(t, e.SetupExchanges(), ErrNoExchangesLoaded)
	})

	t.Run("EnableAllExchanges with specific exchanges set", func(t *testing.T) {
		t.Parallel()
		e := &Engine{
			Config: &config.Config{},
			Settings: Settings{
				CoreSettings: CoreSettings{
					EnableAllExchanges: true,
					Exchanges:          "Bitstamp,Bitfinex",
				},
			},
		}
		assert.EqualError(t, e.SetupExchanges(), "cannot enable all exchanges and specific exchanges concurrently")
	})

	t.Run("Settings dry run toggling", func(t *testing.T) {
		t.Parallel()
		e := &Engine{
			Config: &config.Config{},
			Settings: Settings{
				CoreSettings: CoreSettings{
					EnableAllPairs:     true,
					EnableAllExchanges: true,
				},
				ExchangeTuningSettings: ExchangeTuningSettings{
					EnableExchangeVerbose:          true,
					EnableExchangeWebsocketSupport: true,
					EnableExchangeAutoPairUpdates:  true,
					DisableExchangeAutoPairUpdates: true,
					HTTPUserAgent:                  "test",
					HTTPProxy:                      "test",
					HTTPTimeout:                    1,
					EnableExchangeHTTPDebugging:    true,
				},
			},
		}
		assert.ErrorIs(t, e.SetupExchanges(), ErrNoExchangesLoaded)
		assert.False(t, e.Settings.EnableDryRun)
		e.Settings.CheckParamInteraction = true
		assert.ErrorIs(t, e.SetupExchanges(), ErrNoExchangesLoaded)
		assert.True(t, e.Settings.EnableDryRun)
	})

	// Test that overridden exchange inputs are handled correctly
	testCases := []struct {
		name           string
		exchangeString string
		expectedError  string
	}{
		{"Invalid exchange pair", "bob|jill", "exchange bob|jill not found"},
		{"Single invalid exchange", "bob", "exchange bob not found"},
		{"Mixed valid and invalid exchanges", "bob,bitstamp", "exchange bob not found"},
		{"Valid exchange", "BiTSTaMp", "no exchanges have been loaded"}, // Proper exchange name, but not loaded
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := &Engine{
				Config:   &config.Config{},
				Settings: Settings{CoreSettings: CoreSettings{Exchanges: tc.exchangeString}},
			}
			assert.ErrorContains(t, e.SetupExchanges(), tc.expectedError)
		})
	}

	t.Run("Two valid exchanges with exchanges flag toggled", func(t *testing.T) {
		t.Parallel()
		e := &Engine{Config: &config.Config{}}

		exchLoader := func(exch exchange.IBotExchange) {
			exch.SetDefaults()
			exch.GetBase().Features.Supports.RESTCapabilities.AutoPairUpdates = false
			cfg, err := exchange.GetDefaultConfig(t.Context(), exch)
			require.NoError(t, err)
			e.Config.Exchanges = append(e.Config.Exchanges, *cfg)
		}

		e.ExchangeManager = NewExchangeManager()
		exchLoader(new(bitstamp.Exchange))
		exchLoader(new(bitfinex.Exchange))
		assert.ElementsMatch(t, []string{"Bitstamp", "Bitfinex"}, e.Config.GetEnabledExchanges())

		t.Run("Load specific exchange", func(t *testing.T) {
			e.Settings.Exchanges = "BiTfInEx"
			assert.NoError(t, e.SetupExchanges(), "SetupExchanges with a valid exchange should not error")
			exchanges, err := e.ExchangeManager.GetExchanges()
			require.NoError(t, err)
			require.Len(t, exchanges, 1)
			assert.Equal(t, "Bitfinex", exchanges[0].GetName())
		})

		t.Run("Load all enabled exchanges", func(t *testing.T) {
			e.Settings.Exchanges = ""
			assert.NoError(t, e.SetupExchanges(), "SetupExchanges with all enabled exchanges should not error")
			exchanges, err := e.ExchangeManager.GetExchanges()
			require.NoError(t, err)
			require.Len(t, exchanges, 2)
			exchangeNames := []string{exchanges[0].GetName(), exchanges[1].GetName()}
			assert.ElementsMatch(t, []string{"Bitstamp", "Bitfinex"}, exchangeNames)
		})
	})
}
