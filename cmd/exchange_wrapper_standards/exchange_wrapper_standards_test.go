package exchangewrapperstandards

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
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

func TestMain(m *testing.M) {
	// only run testing suite for one CI/CD environment
	if isAppVeyor() || is32BitJob() {
		return
	}
	request.MaxRequestJobs = 200
	os.Exit(m.Run())
}

// singleExchangeOverride enter an exchange name to only test that exchange
var singleExchangeOverride = ""

func TestAllExchangeWrappers(t *testing.T) {
	t.Parallel()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatal("load config error", err)
	}
	for i := range cfg.Exchanges {
		name := strings.ToLower(cfg.Exchanges[i].Name)
		t.Run(name+" wrapper tests", func(t *testing.T) {
			t.Parallel()
			if common.StringDataContains(unsupportedExchangeNames, name) {
				t.Skipf("skipping unsupported exchange %v", name)
			}
			if singleExchangeOverride != "" && name != singleExchangeOverride {
				t.Skip("skipping ", name, " due to override")
			}
			ctx := context.Background()
			if isCITest() && common.StringDataContains(blockedCIExchanges, name) {
				// rather than skipping tests where execution is blocked, provide an expired
				// context, so no executions can take place
				var cancelFn context.CancelFunc
				ctx, cancelFn = context.WithTimeout(context.Background(), 0)
				cancelFn()
			}
			exch, assetPairs := setupExchange(ctx, t, name, cfg)
			executeExchangeWrapperTests(ctx, t, exch, assetPairs)
		})
	}
}

func setupExchange(ctx context.Context, t *testing.T, name string, cfg *config.Config) (exchange.IBotExchange, []assetPair) {
	t.Helper()
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(name)
	if err != nil {
		t.Fatalf("Cannot setup %v NewExchangeByName  %v", name, err)
	}
	var exchCfg *config.Exchange
	exchCfg, err = cfg.GetExchangeConfig(name)
	if err != nil {
		t.Fatalf("Cannot setup %v GetExchangeConfig %v", name, err)
	}
	exch.SetDefaults()
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.Credentials = getExchangeCredentials(name)

	err = exch.Setup(exchCfg)
	if err != nil {
		t.Fatalf("Cannot setup %v exchange Setup %v", name, err)
	}

	err = exch.UpdateTradablePairs(ctx, true)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Cannot setup %v UpdateTradablePairs %v", name, err)
	}
	b := exch.GetBase()

	assets := b.CurrencyPairs.GetAssetTypes(false)
	if len(assets) == 0 {
		t.Fatalf("Cannot setup %v, exchange has no assets", name)
	}
	for j := range assets {
		err = b.CurrencyPairs.SetAssetEnabled(assets[j], true)
		if err != nil && !errors.Is(err, currency.ErrAssetAlreadyEnabled) {
			t.Fatalf("Cannot setup %v SetAssetEnabled %v", name, err)
		}
	}

	// Add +1 to len to verify that exchanges can handle requests with unset pairs and assets
	assetPairs := make([]assetPair, 0, len(assets)+1)
	for j := range assets {
		var pairs currency.Pairs
		pairs, err = b.CurrencyPairs.GetPairs(assets[j], false)
		if err != nil {
			t.Fatalf("Cannot setup %v asset %v GetPairs %v", name, assets[j], err)
		}
		var p currency.Pair
		p, err = getPairFromPairs(t, pairs)
		if err != nil {
			if errors.Is(err, currency.ErrCurrencyPairsEmpty) {
				continue
			}
			t.Fatalf("Cannot setup %v asset %v getPairFromPairs %v", name, assets[j], err)
		}
		err = b.CurrencyPairs.EnablePair(assets[j], p)
		if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
			t.Fatalf("Cannot setup %v asset %v EnablePair %v", name, assets[j], err)
		}
		p, err = b.FormatExchangeCurrency(p, assets[j])
		if err != nil {
			t.Fatalf("Cannot setup %v asset %v FormatExchangeCurrency %v", name, assets[j], err)
		}
		p, err = disruptFormatting(t, p)
		if err != nil {
			t.Fatalf("Cannot setup %v asset %v disruptFormatting %v", name, assets[j], err)
		}
		assetPairs = append(assetPairs, assetPair{
			Pair:  p,
			Asset: assets[j],
		})
	}
	assetPairs = append(assetPairs, assetPair{})

	return exch, assetPairs
}

// isUnacceptableError sentences errs to 10 years dungeon if unacceptable
func isUnacceptableError(t *testing.T, err error) error {
	t.Helper()
	for i := range acceptableErrors {
		if errors.Is(err, acceptableErrors[i]) {
			return nil
		}
	}
	for i := range warningErrors {
		if errors.Is(err, warningErrors[i]) {
			t.Log(err)
			return nil
		}
	}
	return err
}

func executeExchangeWrapperTests(ctx context.Context, t *testing.T, exch exchange.IBotExchange, assetParams []assetPair) {
	t.Helper()
	iExchange := reflect.TypeOf(&exch).Elem()
	actualExchange := reflect.ValueOf(exch)
	for x := 0; x < iExchange.NumMethod(); x++ {
		methodName := iExchange.Method(x).Name
		if _, ok := excludedMethodNames[methodName]; ok {
			continue
		}
		method := actualExchange.MethodByName(methodName)

		var assetLen int
		for y := 0; y < method.Type().NumIn(); y++ {
			input := method.Type().In(y)
			if input.AssignableTo(assetParam) ||
				input.AssignableTo(orderSubmitParam) ||
				input.AssignableTo(orderModifyParam) ||
				input.AssignableTo(orderCancelParam) ||
				input.AssignableTo(orderCancelsParam) ||
				input.AssignableTo(getOrdersRequestParam) {
				// this allows wrapper functions that support assets types
				// to be tested with all supported assets
				assetLen = len(assetParams) - 1
			}
		}
		tt := time.Now()
		e := time.Date(tt.Year(), tt.Month(), tt.Day()-1, 0, 0, 0, 0, time.UTC)
		s := e.Add(-time.Hour * 24 * 2)
		if methodName == "GetHistoricTrades" {
			// limit trade history
			e = time.Now()
			s = e.Add(-time.Minute * 3)
		}
		for y := 0; y <= assetLen; y++ {
			inputs := make([]reflect.Value, method.Type().NumIn())
			argGenerator := &MethodArgumentGenerator{
				Exchange:    exch,
				AssetParams: assetParams[y],
				MethodName:  methodName,
				Start:       s,
				End:         e,
			}
			for z := 0; z < method.Type().NumIn(); z++ {
				argGenerator.MethodInputType = method.Type().In(z)
				generatedArg := generateMethodArg(ctx, t, argGenerator)
				inputs[z] = *generatedArg
			}
			assetY := assetParams[y].Asset.String()
			pairY := assetParams[y].Pair.String()
			t.Run(methodName+"-"+assetY+"-"+pairY, func(t *testing.T) {
				t.Parallel()
				CallExchangeMethod(t, method, inputs, methodName, exch)
			})
		}
	}
}

// CallExchangeMethod will call an exchange's method using generated arguments
// and determine if the error is friendly
func CallExchangeMethod(t *testing.T, methodToCall reflect.Value, methodValues []reflect.Value, methodName string, exch exchange.IBotExchange) {
	t.Helper()
	outputs := methodToCall.Call(methodValues)
	for i := range outputs {
		outputInterface := outputs[i].Interface()
		err, ok := outputInterface.(error)
		if !ok {
			continue
		}
		if isUnacceptableError(t, err) != nil {
			literalInputs := make([]interface{}, len(methodValues))
			for j := range methodValues {
				literalInputs[j] = methodValues[j].Interface()
			}
			t.Errorf("%v Func '%v' Error: '%v'. Inputs: %v.", exch.GetName(), methodName, err, literalInputs)
		}
		break
	}
}

// MethodArgumentGenerator is used to create arguments for
// an IBotExchange method
type MethodArgumentGenerator struct {
	Exchange        exchange.IBotExchange
	AssetParams     assetPair
	MethodInputType reflect.Type
	MethodName      string
	Start           time.Time
	End             time.Time
	StartTimeSet    bool
	argNum          int64
}

var (
	currencyPairParam    = reflect.TypeOf((*currency.Pair)(nil)).Elem()
	klineParam           = reflect.TypeOf((*kline.Interval)(nil)).Elem()
	contextParam         = reflect.TypeOf((*context.Context)(nil)).Elem()
	timeParam            = reflect.TypeOf((*time.Time)(nil)).Elem()
	codeParam            = reflect.TypeOf((*currency.Code)(nil)).Elem()
	currencyPairsParam   = reflect.TypeOf((*currency.Pairs)(nil)).Elem()
	withdrawRequestParam = reflect.TypeOf((**withdraw.Request)(nil)).Elem()
	stringParam          = reflect.TypeOf((*string)(nil)).Elem()
	feeBuilderParam      = reflect.TypeOf((**exchange.FeeBuilder)(nil)).Elem()
	credentialsParam     = reflect.TypeOf((**account.Credentials)(nil)).Elem()
	// types with asset in params
	assetParam            = reflect.TypeOf((*asset.Item)(nil)).Elem()
	orderSubmitParam      = reflect.TypeOf((**order.Submit)(nil)).Elem()
	orderModifyParam      = reflect.TypeOf((**order.Modify)(nil)).Elem()
	orderCancelParam      = reflect.TypeOf((**order.Cancel)(nil)).Elem()
	orderCancelsParam     = reflect.TypeOf((*[]order.Cancel)(nil)).Elem()
	getOrdersRequestParam = reflect.TypeOf((**order.MultiOrderRequest)(nil)).Elem()
)

// generateMethodArg determines the argument type and returns a pre-made
// response, else an empty version of the type
func generateMethodArg(ctx context.Context, t *testing.T, argGenerator *MethodArgumentGenerator) *reflect.Value {
	t.Helper()
	exchName := strings.ToLower(argGenerator.Exchange.GetName())
	var input reflect.Value
	switch {
	case argGenerator.MethodInputType.AssignableTo(stringParam):
		switch argGenerator.MethodName {
		case "GetDepositAddress":
			if argGenerator.argNum == 2 {
				// account type
				input = reflect.ValueOf("trading")
			} else {
				// Crypto Chain
				input = reflect.ValueOf(cryptoChainPerExchange[exchName])
			}
		default:
			// OrderID
			input = reflect.ValueOf("1337")
		}
	case argGenerator.MethodInputType.AssignableTo(credentialsParam):
		input = reflect.ValueOf(&account.Credentials{
			Key:             "test",
			Secret:          "test",
			ClientID:        "test",
			PEMKey:          "test",
			SubAccount:      "test",
			OneTimePassword: "test",
		})
	case argGenerator.MethodInputType.Implements(contextParam):
		// Need to deploy a context.Context value as nil value is not
		// checked throughout codebase.
		input = reflect.ValueOf(ctx)
	case argGenerator.MethodInputType.AssignableTo(feeBuilderParam):
		input = reflect.ValueOf(&exchange.FeeBuilder{
			FeeType:       exchange.OfflineTradeFee,
			Amount:        1337,
			PurchasePrice: 1337,
			Pair:          argGenerator.AssetParams.Pair,
		})
	case argGenerator.MethodInputType.AssignableTo(currencyPairParam):
		input = reflect.ValueOf(argGenerator.AssetParams.Pair)
	case argGenerator.MethodInputType.AssignableTo(assetParam):
		input = reflect.ValueOf(argGenerator.AssetParams.Asset)
	case argGenerator.MethodInputType.AssignableTo(klineParam):
		input = reflect.ValueOf(kline.OneDay)
	case argGenerator.MethodInputType.AssignableTo(codeParam):
		if argGenerator.MethodName == "GetAvailableTransferChains" {
			input = reflect.ValueOf(currency.ETH)
		} else {
			input = reflect.ValueOf(argGenerator.AssetParams.Pair.Base)
		}
	case argGenerator.MethodInputType.AssignableTo(timeParam):
		if !argGenerator.StartTimeSet {
			input = reflect.ValueOf(argGenerator.Start)
			argGenerator.StartTimeSet = true
		} else {
			input = reflect.ValueOf(argGenerator.End)
		}
	case argGenerator.MethodInputType.AssignableTo(currencyPairsParam):
		b := argGenerator.Exchange.GetBase()
		if argGenerator.AssetParams.Asset != asset.Empty {
			input = reflect.ValueOf(b.CurrencyPairs.Pairs[argGenerator.AssetParams.Asset].Available)
		} else {
			input = reflect.ValueOf(currency.Pairs{
				argGenerator.AssetParams.Pair,
			})
		}
	case argGenerator.MethodInputType.AssignableTo(withdrawRequestParam):
		req := &withdraw.Request{
			Exchange:      exchName,
			Description:   "1337",
			Amount:        1,
			ClientOrderID: "1337",
		}
		if argGenerator.MethodName == "WithdrawCryptocurrencyFunds" {
			req.Type = withdraw.Crypto
			switch {
			case !isFiat(t, argGenerator.AssetParams.Pair.Base.Item.Lower):
				req.Currency = argGenerator.AssetParams.Pair.Base
			case !isFiat(t, argGenerator.AssetParams.Pair.Quote.Item.Lower):
				req.Currency = argGenerator.AssetParams.Pair.Quote
			default:
				req.Currency = currency.ETH
			}

			req.Crypto = withdraw.CryptoRequest{
				Address:    "1337",
				AddressTag: "1337",
				Chain:      cryptoChainPerExchange[exchName],
			}
		} else {
			req.Type = withdraw.Fiat
			b := argGenerator.Exchange.GetBase()
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
		input = reflect.ValueOf(req)
	case argGenerator.MethodInputType.AssignableTo(orderSubmitParam):
		input = reflect.ValueOf(&order.Submit{
			Exchange:          exchName,
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              argGenerator.AssetParams.Pair,
			AssetType:         argGenerator.AssetParams.Asset,
			Price:             1337,
			Amount:            1,
			ClientID:          "1337",
			ClientOrderID:     "13371337",
			ImmediateOrCancel: true,
		})
	case argGenerator.MethodInputType.AssignableTo(orderModifyParam):
		input = reflect.ValueOf(&order.Modify{
			Exchange:          exchName,
			Type:              order.Limit,
			Side:              order.Buy,
			Pair:              argGenerator.AssetParams.Pair,
			AssetType:         argGenerator.AssetParams.Asset,
			Price:             1337,
			Amount:            1,
			ClientOrderID:     "13371337",
			OrderID:           "1337",
			ImmediateOrCancel: true,
		})
	case argGenerator.MethodInputType.AssignableTo(orderCancelParam):
		input = reflect.ValueOf(&order.Cancel{
			Exchange:  exchName,
			Type:      order.Limit,
			Side:      order.Buy,
			Pair:      argGenerator.AssetParams.Pair,
			AssetType: argGenerator.AssetParams.Asset,
			OrderID:   "1337",
		})
	case argGenerator.MethodInputType.AssignableTo(orderCancelsParam):
		input = reflect.ValueOf([]order.Cancel{
			{
				Exchange:  exchName,
				Type:      order.Market,
				Side:      order.Buy,
				Pair:      argGenerator.AssetParams.Pair,
				AssetType: argGenerator.AssetParams.Asset,
				OrderID:   "1337",
			},
		})
	case argGenerator.MethodInputType.AssignableTo(getOrdersRequestParam):
		input = reflect.ValueOf(&order.MultiOrderRequest{
			Type:        order.AnyType,
			Side:        order.AnySide,
			FromOrderID: "1337",
			AssetType:   argGenerator.AssetParams.Asset,
			Pairs:       currency.Pairs{argGenerator.AssetParams.Pair},
		})
	default:
		input = reflect.Zero(argGenerator.MethodInputType)
	}
	argGenerator.argNum++

	return &input
}

// assetPair holds a currency pair associated with an asset
type assetPair struct {
	Pair  currency.Pair
	Asset asset.Item
}

// excludedMethodNames represent the functions that are not
// currently tested under this suite due to irrelevance
// or not worth checking yet
var excludedMethodNames = map[string]struct{}{
	"Setup":                          {}, // Is run via test setup
	"Start":                          {}, // Is run via test setup
	"SetDefaults":                    {}, // Is run via test setup
	"UpdateTradablePairs":            {}, // Is run via test setup
	"GetDefaultConfig":               {}, // Is run via test setup
	"FetchTradablePairs":             {}, // Is run via test setup
	"AuthenticateWebsocket":          {}, // Unnecessary websocket test
	"FlushWebsocketChannels":         {}, // Unnecessary websocket test
	"UnsubscribeToWebsocketChannels": {}, // Unnecessary websocket test
	"SubscribeToWebsocketChannels":   {}, // Unnecessary websocket test
	"GetOrderExecutionLimits":        {}, // Not widely supported/implemented feature
	"UpdateCurrencyStates":           {}, // Not widely supported/implemented feature
	"UpdateOrderExecutionLimits":     {}, // Not widely supported/implemented feature
	"CheckOrderExecutionLimits":      {}, // Not widely supported/implemented feature
	"CanTradePair":                   {}, // Not widely supported/implemented feature
	"CanTrade":                       {}, // Not widely supported/implemented feature
	"CanWithdraw":                    {}, // Not widely supported/implemented feature
	"CanDeposit":                     {}, // Not widely supported/implemented feature
	"GetCurrencyStateSnapshot":       {}, // Not widely supported/implemented feature
	"SetHTTPClientUserAgent":         {}, // standard base implementation
	"SetClientProxyAddress":          {}, // standard base implementation
	// Not widely supported/implemented futures endpoints
	"GetCollateralCurrencyForContract": {},
	"GetCurrencyForRealisedPNL":        {},
	"GetFuturesPositions":              {},
	"GetFundingRates":                  {},
	"IsPerpetualFutureCurrency":        {},
	"GetMarginRatesHistory":            {},
	"CalculatePNL":                     {},
	"CalculateTotalCollateral":         {},
	"ScaleCollateral":                  {},
	"GetPositionSummary":               {},
	"GetLatestFundingRate":             {},
}

// blockedCIExchanges are exchanges that are not able to be tested on CI
var blockedCIExchanges = []string{
	"binance", // binance API is banned from executing within the US where github Actions is ran
	"bybit",   // bybit API is banned from executing within the US where github Actions is ran
}

var unsupportedExchangeNames = []string{
	"testexch",
	"alphapoint",
	"bitflyer", // Bitflyer has many "ErrNotYetImplemented, which is true, but not what we care to test for here
	"bittrex",  // the api is about to expire in March, and we haven't updated it yet
	"itbit",    // itbit has no way of retrieving pair data
}

// cryptoChainPerExchange holds the deposit address chain per exchange
var cryptoChainPerExchange = map[string]string{
	"binanceus": "ERC20",
	"bybit":     "ERC20",
	"gateio":    "ERC20",
}

// acceptable errors do not throw test errors, see below for why
var acceptableErrors = []error{
	common.ErrFunctionNotSupported,       // Shows API cannot perform function and developer has recognised this
	asset.ErrNotSupported,                // Shows that valid and invalid asset types are handled
	request.ErrAuthRequestFailed,         // We must set authenticated requests properly in order to understand and better handle auth failures
	order.ErrUnsupportedOrderType,        // Should be returned if an ordertype like ANY is requested and the implementation knows to throw this specific error
	currency.ErrCurrencyPairEmpty,        // Demonstrates handling of EMPTYPAIR scenario and returns the correct error
	currency.ErrCurrencyNotSupported,     // Ensures a standard error is used for when a particular currency/pair is not supported by an exchange
	currency.ErrCurrencyNotFound,         // Semi-randomly selected currency pairs may not be found at an endpoint, so long as this is returned it is okay
	asset.ErrNotEnabled,                  // Allows distinction when checking for supported versus enabled
	request.ErrRateLimiterAlreadyEnabled, // If the rate limiter is already enabled, it is not an error
	context.DeadlineExceeded,             // If the context deadline is exceeded, it is not an error as only blockedCIExchanges use expired contexts by design
	order.ErrPairIsEmpty,                 // Is thrown when the empty pair and asset scenario for an order submission is sent in the Validate() function
	deposit.ErrAddressNotFound,           // Is thrown when an address is not found due to the exchange requiring valid API keys
}

// warningErrors will t.Log(err) when thrown to diagnose things, but not necessarily suggest
// that the implementation is in error
var warningErrors = []error{
	kline.ErrNoTimeSeriesDataToConvert, // No data returned for a candle isn't worth failing the test suite over necessarily
}

// getPairFromPairs prioritises more normal pairs for an increased
// likelihood of returning data from API endpoints
func getPairFromPairs(t *testing.T, p currency.Pairs) (currency.Pair, error) {
	t.Helper()
	pFmt, err := p.GetFormatting()
	if err != nil {
		return currency.Pair{}, err
	}
	goodEth := currency.NewPair(currency.ETH, currency.USDT).Format(pFmt)
	if p.Contains(goodEth, true) {
		return goodEth, nil
	}
	for i := range p {
		if p[i].Base.Equal(currency.ETH) {
			return p[i], nil
		}
	}
	goodBtc := currency.NewPair(currency.BTC, currency.USDT).Format(pFmt)
	if p.Contains(goodBtc, true) {
		return goodBtc, nil
	}
	for i := range p {
		if p[i].Base.Equal(currency.BTC) {
			return p[i], nil
		}
	}
	return p.GetRandomPair()
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

func getExchangeCredentials(exchangeName string) config.APICredentialsConfig {
	var resp config.APICredentialsConfig
	switch exchangeName {
	case "lbank":
		// these are just random keys, they are not usable
		resp.Key = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA3R2vuz3cpQUbCX0TgYZL
TiLSxUXdrvVEIyoqQyxNf+9fmHLEBrsO1s1msIKvWg24gdbLWXQ6NBCygO8OvZpm
+lfXD4MRv/0PxxIAkaD6Iplhv+qbae8nJkYQOpDJF3bPC9LCKfchCnRpZoGqkHgS
GqOBU13UDZ8BM1SaOLVBzcmE/iJCLPQPORNSzfLSb8TC+woe0AcaDmF9KjIzXPd0
Slacp1ZgZ+yIi1B5/akwxu6sGzHov6weXj/v9K8nUhL9+oPMd8FNzZ+z3viHY0fm
yWiHBywwlh4LgzrjGTUdUk9msjSr2rwjTdCp268A8ECC1fChvhdJfO3lYVj8ltDb
OQIDAQAB`
		resp.Secret = `MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDdHa+7PdylBRsJ
fROBhktOItLFRd2u9UQjKipDLE1/71+YcsQGuw7WzWawgq9aDbiB1stZdDo0ELKA
7w69mmb6V9cPgxG//Q/HEgCRoPoimWG/6ptp7ycmRhA6kMkXds8L0sIp9yEKdGlm
gaqQeBIao4FTXdQNnwEzVJo4tUHNyYT+IkIs9A85E1LN8tJvxML7Ch7QBxoOYX0q
MjNc93RKVpynVmBn7IiLUHn9qTDG7qwbMei/rB5eP+/0rydSEv36g8x3wU3Nn7Pe
+IdjR+bJaIcHLDCWHguDOuMZNR1ST2ayNKvavCNN0KnbrwDwQILV8KG+F0l87eVh
WPyW0Ns5AgMBAAECggEAdBs7hJmWO7yzlsbrsC7BajUU8eue3VkCv2hLqtwfkdcz
HkzdLB+bSiWvD25//0yHHv6X5tAGJALEiLl+xwbFnhzz27xaXLLYTxLf45hg4Dwk
PO9HTlf6+bj+mpIeVcjYLYAs3nZbDi9UjTP3SUcTUpOavBjf2YstyTNai/55oEF/
x+ulzP/OISVhKrk5iiSKgjB4KyFpQnBWyluTmnlNS17/T/k6FkECQFgNpzbUmHTH
Yq+s0I9fGXMMvsNnnoJjX6ALe9fkMjY6ijeA45plDeBZp+5J8uGOKV+/iTCNzm5o
wrQKPz335+tTZgsDdKLUFA9Rwmkcpn4PShOtnR6aZQKBgQD9tzFlomqt/mSWbHAV
Gfjog9snlvgEWBIUjfP5Ow79rbz0cGcL3GAexwKK1dwNmMHDx+fu4uVAIf0dM5aT
xfdp/I4OTkxOFcIupu+L4gmz1vY32pFLPQYbp+9oOAMy4thUFb5o/Dsq2g65e2BC
+gNALEWxPuhNYbI7c0cu5Y7AJwKBgQDfG1ovhNlETJO+oli25csayRwgm/qll4fH
sOnYospQiJ3ka0WjPT6NY8m2anWDp7+/guIwq+xXVF6wQNxNZc+6/MgNJo2R3XG5
FKPH5FYgI52Zv6VN1AUhdfInDpKQXQ8vWO6HV+/uJmHeZK2+D6nycN4dL2h7ElK/
sCthmNtFnwKBgQDCGdaGpLzspAScOBV/b0FH0Shmn07bM+2RIBCYiaAsXzCB6URM
hKpcoW/Ge1pAZK9IcrVzws4URGx6XK9EGl3wDbE4LJqf2nGWc0wsPh+iIEB59pLV
drgnjFDR8Jgx4+4QVho4A0/Ytr4xFLxOQSsfez9OHIxoNue+J7E7pY+SXQKBgBTT
0tl4x2eO1oQHV8zLKui3OX750K5AtRY5N7tXhxd5iXPXZ8rTXtGILT5wNcQylr3k
FAWDJy8H20cM5wP6qyfDjVFc9f5V89XZTWjNshSR/pZpw56+WjRDdHWc8KW1akN7
Q9kypl1PC/fc4jNJ9w2A59tFn7VNgpgOdB5KTL31AoGBAN3BIjKXzoOJnVGL3bja
SYC2m+JcRn/mVO7I5Hop8GDoWXPFAnPNx1YKSpRLM/EV+ukUJsOV/LTPb7BsXMsJ
IY9SZceJS6glsxt+blFxGEpypyv13xW+jeCrPjlxQX2TNbL0KwHqvm1zMnM9bss/
Rsd80LrBCVI8ctzrvYRFSugC`
	default:
		resp.Key = "realKey"
		resp.Secret = "YXBpU2VjcmV0" // base64 encoded "apiSecret"
		resp.ClientID = "realClientID"
	}
	return resp
}

func isCITest() bool {
	ci := os.Getenv("CI")
	return ci == "true" /* github actions */ || ci == "True" /* appveyor */
}

func isAppVeyor() bool {
	ci := os.Getenv("APPVEYOR")
	return ci == "True"
}

func is32BitJob() bool {
	ci := os.Getenv("GITHUB_JOB")
	return ci == "backend-32bit"
}
