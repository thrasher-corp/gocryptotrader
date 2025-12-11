package gct

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	objects "github.com/d5/tengo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
)

func TestMain(m *testing.M) {
	settings := engine.Settings{
		CoreSettings: engine.CoreSettings{
			EnableDryRun:                true,
			EnableDepositAddressManager: true,
		},
		ConfigFile: filepath.Join("..", "..", "..", "testdata", "configtest.json"),
		DataDir:    filepath.Join("..", "..", "..", "testdata", "gocryptotrader"),
	}
	var err error
	engine.Bot, err = engine.NewFromSettings(&settings, nil)
	if err != nil {
		log.Fatalf("Error from engine.NewFromSettings: %s", err)
	}
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(exch.Value)
	if err != nil {
		log.Fatalf("Error from NewExchangeByName: %s", err)
	}
	cfg, err := exchange.GetDefaultConfig(context.Background(), exch)
	if err != nil {
		log.Fatalf("Error during GetDefaultConfig: %s", err)
	}
	if err = exch.Setup(cfg); err != nil {
		log.Fatalf("Error during exch.Setup: %s", err)
	}
	if err = em.Add(exch); err != nil {
		log.Fatalf("Error during ExchangeManager.Add: %s", err)
	}
	engine.Bot.ExchangeManager = em
	engine.Bot.WithdrawManager, err = engine.SetupWithdrawManager(em, nil, true)
	if err != nil {
		log.Fatalf("Error during engine.SetupWithdrawManage: %s", err)
	}

	engine.Bot.DepositAddressManager = engine.SetupDepositAddressManager()
	err = engine.Bot.DepositAddressManager.Sync(engine.Bot.GetAllExchangeCryptocurrencyDepositAddresses())
	if err != nil {
		log.Fatalf("Error syncing DepositAddressManager: %s", err)
	}

	engine.Bot.OrderManager, err = engine.SetupOrderManager(em, &engine.CommunicationManager{}, &engine.Bot.ServicesWG, &config.OrderManager{})
	if err != nil {
		log.Fatalf("Error during SetupOrderManager: %s", err)
	}
	if err = engine.Bot.OrderManager.Start(); err != nil {
		log.Fatalf("Error starting OrderManager: %s", err)
	}
	modules.SetModuleWrapper(Setup())
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Setup(), "Setup must not return nil")
}

var (
	exch = &objects.String{
		Value: "Bitstamp",
	}
	exchError = &objects.String{
		Value: "error",
	}
	currencyPair = &objects.String{
		Value: "BTC-USD",
	}
	delimiter = &objects.String{
		Value: "-",
	}
	assetType = &objects.String{
		Value: "spot",
	}
	orderID = &objects.String{
		Value: "1235",
	}

	ctx = &gct.Context{}

	tv = objects.TrueValue
	fv = objects.FalseValue
)

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderbook(ctx, exch, currencyPair, delimiter, assetType)
	assert.NoError(t, err)

	_, err = gct.ExchangeOrderbook()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeTicker(ctx, exch, currencyPair, delimiter, assetType)
	assert.NoError(t, err)

	_, err = gct.ExchangeTicker(ctx, exchError, currencyPair, delimiter, assetType)
	assert.NoError(t, err)

	_, err = gct.ExchangeTicker()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangeExchanges(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeExchanges(tv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges(exch)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges(fv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangePairs(exch, tv, assetType)
	require.NoError(t, err)

	_, err = gct.ExchangePairs(exchError, tv, assetType)
	assert.NoError(t, err)

	_, err = gct.ExchangePairs()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
}

func TestExchangeAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeAccountBalances()
	require.ErrorIs(t, err, objects.ErrWrongNumArguments)
	obj, err := gct.ExchangeAccountBalances(ctx, exch, assetType)
	require.NoError(t, err)
	rString, ok := objects.ToString(obj)
	require.True(t, ok, "ExchangeAccountBalances return value must return correctly from objects.ToString")
	require.Contains(t, rString, "Bitstamp REST or Websocket authentication support is not enabled")
}

func TestExchangeOrderQuery(t *testing.T) {
	t.Parallel()

	_, err := gct.ExchangeOrderQuery()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	_, err = gct.ExchangeOrderQuery(ctx, exch, orderID)
	assert.NoError(t, err)
}

func TestExchangeOrderCancel(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderCancel()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)
	_, err = gct.ExchangeOrderCancel(ctx, exch, orderID, currencyPair, assetType)
	assert.NoError(t, err)
}

func TestExchangeOrderSubmit(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderSubmit()
	require.ErrorIs(t, err, objects.ErrWrongNumArguments)

	orderSide := &objects.String{Value: "ASK"}
	orderType := &objects.String{Value: "LIMIT"}
	orderPrice := &objects.Float{Value: 1}
	orderAmount := &objects.Float{Value: 1}
	orderAsset := &objects.String{Value: asset.Spot.String()}

	obj, err := gct.ExchangeOrderSubmit(ctx,
		exch,
		currencyPair,
		delimiter,
		orderType,
		orderSide,
		orderPrice,
		orderAmount,
		orderID,
		orderAsset)
	require.NoError(t, err)

	rString, ok := objects.ToString(obj)
	require.True(t, ok, "ExchangeOrderSubmit return value must return correctly from objects.ToString")
	require.Contains(t, rString, "Bitstamp REST or Websocket authentication support is not enabled")
}

func TestAllModuleNames(t *testing.T) {
	t.Parallel()
	assert.IsType(t, []string{}, gct.AllModuleNames(), "AllModuleNames should return a slice of strings")
}

func TestExchangeDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeDepositAddress()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	currCode := &objects.String{Value: "BTC"}
	chain := &objects.String{Value: ""}
	_, err = gct.ExchangeDepositAddress(exch, currCode, chain)
	assert.NoError(t, err)
}

func TestExchangeWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeWithdrawCrypto()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	currCode := &objects.String{Value: "BTC"}
	desc := &objects.String{Value: "HELLO"}
	address := &objects.String{Value: "0xTHISISALEGITBTCADDRESSS"}
	amount := &objects.Float{Value: 1.0}

	_, err = gct.ExchangeWithdrawCrypto(ctx, exch, currCode, address, address, amount, amount, desc)
	assert.NoError(t, err)
}

func TestExchangeWithdrawFiat(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeWithdrawFiat()
	assert.ErrorIs(t, err, objects.ErrWrongNumArguments)

	currCode := &objects.String{Value: "TEST"}
	amount := &objects.Float{Value: 1.0}
	desc := &objects.String{Value: "2"}
	bankID := &objects.String{Value: "3!"}
	_, err = gct.ExchangeWithdrawFiat(ctx, exch, currCode, desc, amount, bankID)
	assert.NoError(t, err)
}
