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
	"Start",
	"SetDefaults",
	"UpdateTradablePairs",
	"GetDefaultConfig",
	"FetchTradablePairs",
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

// getPairFromPairs prioritises more normal pairs for an increased
// likelihood of returning data from API endpoints
func getPairFromPairs(p currency.Pairs) (currency.Pair, error) {
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
			em := SetupExchangeManager()
			var exch exchange.IBotExchange
			exch, err = em.NewExchangeByName(name)
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
					p, err = getPairFromPairs(pairs)
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
					p, err = getPairFromPairs(pairs)
					if err != nil {
						t.Fatalf("getPairFromPairs %v %v", err, assets[j])
					}
				}
				p, err = b.FormatExchangeCurrency(p, assets[j])
				if err != nil {
					t.Fatal(err)
				}
				p, err = disruptFormatting(p)
				if err != nil {
					t.Fatal(err)
				}
				testMap[j] = assetPair{
					Pair:  p,
					Asset: assets[j],
				}
			}
			what, err := executeExchangeWrapperTests(t, exch, testMap)
			if err != nil {
				t.Error(err)
			}
			for zz := range what {
				t.Log(what[zz])
			}
		})
	}
}

func executeExchangeWrapperTests(t *testing.T, exch exchange.IBotExchange, assetParams []assetPair) ([]string, error) {
	var acceptableErr error
	for i := range acceptableErrors {
		acceptableErr = common.AppendError(acceptableErr, acceptableErrors[i])
	}
	iExchange := reflect.TypeOf(&exch).Elem()
	actualExchange := reflect.ValueOf(exch)
	errType := reflect.TypeOf(common.ErrNotYetImplemented)

	contextParam := reflect.TypeOf((*context.Context)(nil)).Elem()
	cpParam := reflect.TypeOf((*currency.Pair)(nil)).Elem()
	assetParam := reflect.TypeOf((*asset.Item)(nil)).Elem()
	klineParam := reflect.TypeOf((*kline.Interval)(nil)).Elem()
	timeParam := reflect.TypeOf((*time.Time)(nil)).Elem()
	codeParam := reflect.TypeOf((*currency.Code)(nil)).Elem()

	e := time.Now().Add(-time.Hour * 24)
	var funcs []string
methods:
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
			s = time.Now().Add(-time.Minute * 5).Truncate(time.Hour)
		}
		for y := 0; y <= assetLen; y++ {
			inputs := make([]reflect.Value, method.Type().NumIn())
			setStartTime := false
			for z := 0; z < method.Type().NumIn(); z++ {
				input := method.Type().In(z)
				switch {
				case input.Implements(contextParam):
					// Need to deploy a context.Context value as nil value is not
					// checked throughout codebase.
					inputs[z] = reflect.ValueOf(context.Background())
					continue
				case input.AssignableTo(cpParam):
					inputs[z] = reflect.ValueOf(assetParams[y].Pair)
				case input.AssignableTo(assetParam):
					inputs[z] = reflect.ValueOf(assetParams[y].Asset)
				case input.AssignableTo(klineParam):
					inputs[z] = reflect.ValueOf(kline.OneDay)
				case input.AssignableTo(codeParam):
					if name == "GetAvailableTransferChains" {
						inputs[z] = reflect.ValueOf(currency.ETH)
					} else {
						inputs[z] = reflect.ValueOf(assetParams[y].Pair.Quote)
					}
				case input.AssignableTo(timeParam):
					if !setStartTime {
						inputs[z] = reflect.ValueOf(s)
						setStartTime = true
					} else {
						inputs[z] = reflect.ValueOf(e)
					}
				default:
					resp, err := buildRequest(exch, exch.GetName(), name, assetParams[y].Asset, assetParams[y].Pair, input)
					if err != nil {
						t.Errorf("%v %v %v %v %v %v", exch, name, assetParams[y].Asset, assetParams[y].Pair, input.Name(), err)
					}
					if resp == nil {
						// unsupported request
						continue methods
					} else {
						inputs[z] = reflect.ValueOf(resp)
					}
				}
			}

			t.Run(name+"-"+assetParams[y].Asset.String()+"-"+assetParams[y].Pair.String(), func(t *testing.T) {
				t.Parallel()
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
			})
		}
	}

	return funcs, nil
}

func isFiat(c string) bool {
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

// buildRequest returns more complex struct requirements for a wrapper
// implementation
func buildRequest(exch exchange.IBotExchange, exchName, funcName string, a asset.Item, p currency.Pair, input reflect.Type) (interface{}, error) {
	pairs := reflect.TypeOf((*currency.Pairs)(nil)).Elem()
	wr := reflect.TypeOf((**withdraw.Request)(nil)).Elem()
	os := reflect.TypeOf((**order.Submit)(nil)).Elem()
	om := reflect.TypeOf((**order.Modify)(nil)).Elem()
	oc := reflect.TypeOf((**order.Cancel)(nil)).Elem()
	occ := reflect.TypeOf((*[]order.Cancel)(nil)).Elem()
	gor := reflect.TypeOf((**order.GetOrdersRequest)(nil)).Elem()

	switch {
	case input.AssignableTo(pairs):
		return currency.Pairs{
			p,
		}, nil
	case input.AssignableTo(wr):
		req := &withdraw.Request{
			Exchange:      exchName,
			Description:   "1337",
			Amount:        1,
			ClientOrderID: "1337",
		}
		if funcName == "WithdrawCryptocurrencyFunds" {
			req.Type = withdraw.Crypto
			if !isFiat(p.Base.Item.Lower) {
				req.Currency = p.Base
			} else if !isFiat(p.Quote.Item.Lower) {
				req.Currency = p.Quote
			} else {
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
					SupportedExchanges:  exchName,
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
		return req, nil
	case input.AssignableTo(os):
		return &order.Submit{
			Exchange:          exchName,
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              p,
			AssetType:         a,
			Price:             1337,
			Amount:            1,
			ClientID:          "1337",
			ClientOrderID:     "13371337",
			ImmediateOrCancel: true,
		}, nil
	case input.AssignableTo(om):
		return &order.Modify{
			Exchange:          exchName,
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              p,
			AssetType:         a,
			Price:             1337,
			Amount:            1,
			ClientOrderID:     "13371337",
			OrderID:           "1337",
			ImmediateOrCancel: true,
		}, nil
	case input.AssignableTo(oc):
		return &order.Cancel{
			Exchange:      exchName,
			Type:          order.Limit,
			Side:          order.Buy,
			Pair:          p,
			AssetType:     a,
			ClientOrderID: "13371337",
		}, nil
	case input.AssignableTo(occ):
		return []order.Cancel{
			{
				Exchange:      exchName,
				Type:          order.Market,
				Side:          order.Buy,
				Pair:          p,
				AssetType:     a,
				ClientOrderID: "13371337",
			},
		}, nil
	case input.AssignableTo(gor):
		return &order.GetOrdersRequest{
			Type:      order.AnyType,
			Side:      order.AnySide,
			OrderID:   "1337",
			AssetType: a,
			Pairs:     currency.Pairs{p},
		}, nil
	}
	return nil, nil
}

// disruptFormatting adds in an unused delimiter and strange casing features to
// ensure format currency pair is used throughout the code base.
func disruptFormatting(p currency.Pair) (currency.Pair, error) {
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
