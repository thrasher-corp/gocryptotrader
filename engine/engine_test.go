package engine

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

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
				EnableDryRun: true,
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
				EnableDryRun: true,
			},
			want:    &somePath,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
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
		EnableDryRun: true,
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
		EnableDryRun: true,
		DataDir:      tempDir,
	}, nil)
	if err != nil {
		t.Error(err)
	}
	botOne.Settings.EnableGRPCProxy = false

	botTwo, err := NewFromSettings(&Settings{
		ConfigFile:   config.TestFile,
		EnableDryRun: true,
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
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received: %v expected: %v", err, ErrNilSubsystem)
	}

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	exch.SetDefaults()
	exch.SetEnabled(true)
	em.Add(exch)
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
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received: %v expected: %v", err, ErrExchangeNotFound)
	}
}

func TestUnloadExchange(t *testing.T) {
	t.Parallel()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	exch.SetDefaults()
	exch.SetEnabled(true)
	em.Add(exch)
	e := &Engine{ExchangeManager: em,
		Config: &config.Config{Exchanges: []config.Exchange{{Name: testExchange}}},
	}
	err = e.UnloadExchange("asdf")
	if !errors.Is(err, config.ErrExchangeNotFound) {
		t.Errorf("error '%v', expected '%v'", err, config.ErrExchangeNotFound)
	}

	err = e.UnloadExchange(testExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = e.UnloadExchange(testExchange)
	if !errors.Is(err, ErrNoExchangesLoaded) {
		t.Errorf("error '%v', expected '%v'", err, ErrNoExchangesLoaded)
	}
}

func TestDryRunParamInteraction(t *testing.T) {
	t.Parallel()
	bot := &Engine{
		ExchangeManager: SetupExchangeManager(),
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
	if err := bot.LoadExchange(testExchange, nil); err != nil {
		t.Error(err)
	}
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
	if err = bot.LoadExchange(testExchange, nil); err != nil {
		t.Error(err)
	}

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
	if !errors.Is(err, errNilBot) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilBot)
	}

	e = &Engine{websocketRoutineManager: &websocketRoutineManager{}}
	err = e.RegisterWebsocketDataHandler(func(_ string, _ interface{}) error { return nil }, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestSetDefaultWebsocketDataHandler(t *testing.T) {
	t.Parallel()
	var e *Engine
	err := e.SetDefaultWebsocketDataHandler()
	if !errors.Is(err, errNilBot) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNilBot)
	}

	e = &Engine{websocketRoutineManager: &websocketRoutineManager{}}
	err = e.SetDefaultWebsocketDataHandler()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

type assetPair struct {
	Pair  currency.Pair
	Asset asset.Item
}

// unsupportedFunctionNames represent the functions that are not
// currently tested under this suite due to irrelevance
// or not worth checking yet
var unsupportedFunctionNames = []string{
	"Start",               // Is run via test setup
	"SetDefaults",         // Is run via test setup
	"UpdateTradablePairs", // Is run via test setup
	"GetDefaultConfig",    // Is run via test setup
	"FetchTradablePairs",  // Is run via test setup
	"GetCollateralCurrencyForContract",
	"GetCurrencyForRealisedPNL",
	"FlushWebsocketChannels",
	"GetOrderExecutionLimits",
	"IsPerpetualFutureCurrency",
	"UpdateCurrencyStates",
	"UpdateOrderExecutionLimits",
	"CanTradePair",
	"CanTrade",
	"CanWithdraw",
	"CanDeposit",
	"GetCurrencyStateSnapshot",
	"GetPositionSummary",
	"ScaleCollateral",
	"CalculateTotalCollateral",
	"GetFuturesPositions",
	"GetFundingRates",
	"IsPerpetualFutureCurrency",
	"GetMarginRatesHistory",
	"CalculatePNL",
	"AuthenticateWebsocket",
}

var unsupportedExchangeNames = []string{
	"alphapoint",
	"bitflyer", // Bitflyer has many "ErrNotYetImplemented, which is true, but not what we care to test for here
	"bittrex",  // the api is about to expire in March, and we haven't updated it yet
	"itbit",    // itbit has no way of retrieving pair data
}

var acceptableErrors = []error{
	common.ErrFunctionNotSupported,
	asset.ErrNotSupported,
	request.ErrAuthRequestFailed,
	order.ErrUnsupportedOrderType,
}

func TestAllExchanges(t *testing.T) {
	t.Parallel()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.json", true)
	if err != nil {
		t.Fatal("load config error", err)
	}
	for i := range cfg.Exchanges {
		name := cfg.Exchanges[i].Name
		if common.StringDataContains(unsupportedExchangeNames, strings.ToLower(name)) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			exch, testMap := setupAllExchanges(t, name, cfg)
			executeExchangeWrapperTests(t, exch, testMap)
		})
	}
}

// getPairFromPairs prioritises more normal pairs for an increased
// likelihood of returning data from API endpoints
func getPairFromPairs(t *testing.T, p currency.Pairs) (currency.Pair, error) {
	t.Helper()
	for i := range p {
		if p[i].Base.Equal(currency.BTC) {
			return p[i], nil
		}
	}
	for i := range p {
		if p[i].Base.Equal(currency.ETH) {
			return p[i], nil
		}
	}
	return p.GetRandomPair()
}

func setupAllExchanges(t *testing.T, name string, cfg *config.Config) (exchange.IBotExchange, []assetPair) {
	t.Helper()
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(name)
	if err != nil {
		t.Fatal(err)
	}
	var exchCfg *config.Exchange
	exchCfg, err = cfg.GetExchangeConfig(name)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.Credentials.Key = "realKey"
	exchCfg.API.Credentials.Secret = "realSecret"
	exchCfg.API.Credentials.ClientID = "realClientID"
	err = exch.Setup(exchCfg)
	if err != nil {
		t.Fatal(err)
	}

	err = exch.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	b := exch.GetBase()
	assets := b.CurrencyPairs.GetAssetTypes(false)
	if len(assets) == 0 {
		t.Fatal(name)
	}
	for j := range assets {
		err = b.CurrencyPairs.SetAssetEnabled(assets[j], true)
		if err != nil && !errors.Is(err, currency.ErrAssetAlreadyEnabled) {
			t.Fatal(err)
		}
	}
	testMap := make([]assetPair, len(assets))
	for j := range assets {
		var pairs currency.Pairs
		pairs, err = b.CurrencyPairs.GetPairs(assets[j], true)
		if err != nil {
			t.Fatal(err)
		}
		var p currency.Pair
		if len(pairs) == 0 {
			pairs, err = b.CurrencyPairs.GetPairs(assets[j], false)
			if err != nil {
				t.Fatalf("GetPairs %v %v", err, assets[j])
			}
			p, err = getPairFromPairs(t, pairs)
			if err != nil {
				t.Fatalf("getPairFromPairs %v %v", err, assets[j])
			}
			p, err = b.FormatExchangeCurrency(p, assets[j])
			if err != nil {
				t.Fatalf("FormatExchangeCurrency %v %v", err, assets[j])
			}
			err = b.CurrencyPairs.EnablePair(assets[j], p)
			if err != nil {
				t.Fatalf("EnablePair %v %v", err, assets[j])
			}
		} else {
			p, err = getPairFromPairs(t, pairs)
			if err != nil {
				t.Fatalf("getPairFromPairs %v %v", err, assets[j])
			}
		}
		p, err = b.FormatExchangeCurrency(p, assets[j])
		if err != nil {
			t.Fatal(err)
		}
		p, err = disruptFormatting(t, p)
		if err != nil {
			t.Fatal(err)
		}
		testMap[j] = assetPair{
			Pair:  p,
			Asset: assets[j],
		}
	}
	return exch, testMap
}

func executeExchangeWrapperTests(t *testing.T, exch exchange.IBotExchange, assetParams []assetPair) {
	t.Helper()
	var acceptableErr error
	for i := range acceptableErrors {
		acceptableErr = common.AppendError(acceptableErr, acceptableErrors[i])
	}
	iExchange := reflect.TypeOf(&exch).Elem()
	actualExchange := reflect.ValueOf(exch)
	errType := reflect.TypeOf(common.ErrNotYetImplemented)

	assetParam := reflect.TypeOf((*asset.Item)(nil)).Elem()

	e := time.Now().Add(-time.Hour * 24)
	for x := 0; x < iExchange.NumMethod(); x++ {
		name := iExchange.Method(x).Name
		if common.StringDataContains(unsupportedFunctionNames, name) {
			continue
		}
		method := actualExchange.MethodByName(name)

		var assetLen int
		for y := 0; y < method.Type().NumIn(); y++ {
			input := method.Type().In(y)
			if input.AssignableTo(assetParam) {
				assetLen = len(assetParams) - 1
			}
		}

		s := time.Now().Add(-time.Hour * 24 * 7).Truncate(time.Hour)
		if name == "GetHistoricTrades" {
			// limit trade history
			s = time.Now().Add(-time.Minute * 5)
		}
		for y := 0; y <= assetLen; y++ {
			inputs := make([]reflect.Value, method.Type().NumIn())
			setStartTime := false
			for z := 0; z < method.Type().NumIn(); z++ {
				inputType := method.Type().In(z)
				tt := s
				if setStartTime {
					tt = e
				}
				funcArg := createFuncArgs(t, exch, &assetParams[y], method, inputType, name, tt)
				inputs[z] = *funcArg
				setStartTime = true
			}
			t.Run(name+"-"+assetParams[y].Asset.String()+"-"+assetParams[y].Pair.String(), func(t *testing.T) {
				t.Parallel()
				callFunction(t, method, inputs, exch, errType, name)
			})
		}
	}
}

func createFuncArgs(t *testing.T, exch exchange.IBotExchange, assetParams *assetPair, method reflect.Value, inputType reflect.Type, functionName string, tt time.Time) *reflect.Value {
	t.Helper()
	cpParam := reflect.TypeOf((*currency.Pair)(nil)).Elem()
	klineParam := reflect.TypeOf((*kline.Interval)(nil)).Elem()
	contextParam := reflect.TypeOf((*context.Context)(nil)).Elem()
	timeParam := reflect.TypeOf((*time.Time)(nil)).Elem()
	codeParam := reflect.TypeOf((*currency.Code)(nil)).Elem()
	assetParam := reflect.TypeOf((*asset.Item)(nil)).Elem()
	pairs := reflect.TypeOf((*currency.Pairs)(nil)).Elem()
	wr := reflect.TypeOf((**withdraw.Request)(nil)).Elem()
	os := reflect.TypeOf((**order.Submit)(nil)).Elem()
	om := reflect.TypeOf((**order.Modify)(nil)).Elem()
	oc := reflect.TypeOf((**order.Cancel)(nil)).Elem()
	occ := reflect.TypeOf((*[]order.Cancel)(nil)).Elem()
	gor := reflect.TypeOf((**order.GetOrdersRequest)(nil)).Elem()

	var input reflect.Value
	switch {
	case inputType.Implements(contextParam):
		// Need to deploy a context.Context value as nil value is not
		// checked throughout codebase.
		input = reflect.ValueOf(context.Background())
	case inputType.AssignableTo(cpParam):
		input = reflect.ValueOf(assetParams.Pair)
	case inputType.AssignableTo(assetParam):
		input = reflect.ValueOf(assetParams.Asset)
	case inputType.AssignableTo(klineParam):
		input = reflect.ValueOf(kline.OneDay)
	case inputType.AssignableTo(codeParam):
		if functionName == "GetAvailableTransferChains" {
			input = reflect.ValueOf(currency.ETH)
		} else {
			input = reflect.ValueOf(assetParams.Pair.Quote)
		}
	case inputType.AssignableTo(timeParam):
		input = reflect.ValueOf(tt)
	case inputType.AssignableTo(pairs):
		input = reflect.ValueOf(currency.Pairs{
			assetParams.Pair,
		})
	case inputType.AssignableTo(wr):
		req := &withdraw.Request{
			Exchange:      exch.GetName(),
			Description:   "1337",
			Amount:        1,
			ClientOrderID: "1337",
		}
		if functionName == "WithdrawCryptocurrencyFunds" {
			req.Type = withdraw.Crypto
			switch {
			case !isFiat(t, assetParams.Pair.Base.Item.Lower):
				req.Currency = assetParams.Pair.Base
			case !isFiat(t, assetParams.Pair.Quote.Item.Lower):
				req.Currency = assetParams.Pair.Quote
			default:
				req.Currency = currency.ETH
			}

			req.Crypto = withdraw.CryptoRequest{
				Address:    "1337",
				AddressTag: "1337",
				Chain:      "ERC20",
			}
		} else {
			req.Type = withdraw.Fiat
			b := exch.GetBase()
			if len(b.Config.BaseCurrencies) > 0 {
				req.Currency = b.Config.BaseCurrencies[0]
			} else {
				req.Currency = currency.USD
			}
			req.Fiat = withdraw.FiatRequest{
				Bank: banking.Account{
					Enabled:             true,
					ID:                  "1337",
					BankName:            "1337",
					BankAddress:         "1337",
					BankPostalCode:      "1337",
					BankPostalCity:      "1337",
					BankCountry:         "1337",
					AccountName:         "1337",
					AccountNumber:       "1337",
					SWIFTCode:           "1337",
					IBAN:                "1337",
					BSBNumber:           "1337",
					BankCode:            1337,
					SupportedCurrencies: req.Currency.String(),
					SupportedExchanges:  exch.GetName(),
				},
				IsExpressWire:                 false,
				RequiresIntermediaryBank:      false,
				IntermediaryBankAccountNumber: 1338,
				IntermediaryBankName:          "1338",
				IntermediaryBankAddress:       "1338",
				IntermediaryBankCity:          "1338",
				IntermediaryBankCountry:       "1338",
				IntermediaryBankPostalCode:    "1338",
				IntermediarySwiftCode:         "1338",
				IntermediaryBankCode:          1338,
				IntermediaryIBAN:              "1338",
				WireCurrency:                  "1338",
			}
		}
		input = reflect.ValueOf(req)
	case inputType.AssignableTo(os):
		input = reflect.ValueOf(&order.Submit{
			Exchange:          exch.GetName(),
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              assetParams.Pair,
			AssetType:         assetParams.Asset,
			Price:             1337,
			Amount:            1,
			ClientID:          "1337",
			ClientOrderID:     "13371337",
			ImmediateOrCancel: true,
		})
	case inputType.AssignableTo(om):
		input = reflect.ValueOf(&order.Modify{
			Exchange:          exch.GetName(),
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              assetParams.Pair,
			AssetType:         assetParams.Asset,
			Price:             1337,
			Amount:            1,
			ClientOrderID:     "13371337",
			OrderID:           "1337",
			ImmediateOrCancel: true,
		})
	case inputType.AssignableTo(oc):
		input = reflect.ValueOf(&order.Cancel{
			Exchange:      exch.GetName(),
			Type:          order.Limit,
			Side:          order.Buy,
			Pair:          assetParams.Pair,
			AssetType:     assetParams.Asset,
			ClientOrderID: "13371337",
		})
	case inputType.AssignableTo(occ):
		input = reflect.ValueOf([]order.Cancel{
			{
				Exchange:      exch.GetName(),
				Type:          order.Market,
				Side:          order.Buy,
				Pair:          assetParams.Pair,
				AssetType:     assetParams.Asset,
				ClientOrderID: "13371337",
			},
		})
	case inputType.AssignableTo(gor):
		input = reflect.ValueOf(&order.GetOrdersRequest{
			Type:      order.AnyType,
			Side:      order.AnySide,
			OrderID:   "1337",
			AssetType: assetParams.Asset,
			Pairs:     currency.Pairs{assetParams.Pair},
		})
	default:
		input = reflect.Zero(inputType)
	}
	return &input
}

func callFunction(t *testing.T, method reflect.Value, inputs []reflect.Value, exch exchange.IBotExchange, errType reflect.Type, name string) {
	t.Helper()
	outputs := method.Call(inputs)
	if method.Type().NumIn() == 0 {
		// Some empty functions will reset the exchange struct to defaults,
		// so turn off verbosity.
		exch.GetBase().Verbose = false
	}
errProcessing:
	for i := range outputs {
		incoming := outputs[i].Interface()
		if reflect.TypeOf(incoming) == errType {
			err, ok := incoming.(error)
			if !ok {
				t.Errorf("%s type assertion failure for %v", name, incoming)
				continue
			}
			for z := range acceptableErrors {
				if errors.Is(err, acceptableErrors[z]) {
					break errProcessing
				}
			}
			literalInputs := make([]interface{}, len(inputs))
			for j := range inputs {
				literalInputs[j] = inputs[j].Interface()
			}
			t.Errorf("Error: '%v'. Inputs: %v", err, literalInputs)
			break
		}
	}
}

// isFiat helps determine fiat currency without using currency.storage
func isFiat(t *testing.T, c string) bool {
	t.Helper()
	var fiats = []string{
		currency.USD.Item.Lower,
		currency.AUD.Item.Lower,
		currency.EUR.Item.Lower,
		currency.CAD.Item.Lower,
		currency.TRY.Item.Lower,
		currency.UAH.Item.Lower,
		currency.RUB.Item.Lower,
		currency.RUR.Item.Lower,
		currency.JPY.Item.Lower,
		currency.HKD.Item.Lower,
		currency.SGD.Item.Lower,
		currency.ZUSD.Item.Lower,
		currency.ZEUR.Item.Lower,
		currency.ZCAD.Item.Lower,
		currency.ZJPY.Item.Lower,
	}
	for i := range fiats {
		if fiats[i] == c {
			return true
		}
	}
	return false
}

// disruptFormatting adds in an unused delimiter and strange casing features to
// ensure format currency pair is used throughout the code base.
func disruptFormatting(t *testing.T, p currency.Pair) (currency.Pair, error) {
	t.Helper()
	if p.Base.IsEmpty() {
		return currency.EMPTYPAIR, errors.New("cannot disrupt formatting as base is not populated")
	}
	// NOTE: Quote can be empty for margin funding
	return currency.Pair{
		Base:      p.Base.Upper(),
		Quote:     p.Quote.Lower(),
		Delimiter: "-TEST-DELIM-",
	}, nil
}
