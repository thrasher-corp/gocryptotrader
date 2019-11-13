package tests

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/d5/tengo/objects"

	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/gctscript/gctwrapper"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
)

var (
	settings = engine.Settings{
		ConfigFile:          "../../../testdata/gctscript/config.json",
		EnableDryRun:        true,
		DataDir:             "../../../testdata/gocryptotrader",
		Verbose:             false,
		EnableGRPC:          false,
		EnableDeprecatedRPC: false,
		EnableWebsocketRPC:  false,
	}

	exch = &objects.String{
		Value: "BTC Markets",
	}
	currencyPair = &objects.String{
		Value: "BTC-AUD",
	}
	delimiter = &objects.String{
		Value: "-",
	}
	assetType = &objects.String{
		Value: "SPOT",
	}

	tv = objects.TrueValue
	fv = objects.FalseValue
)

func TestMain(m *testing.M) {
	modules.SetModuleWrapper(gctwrapper.Setup())
	err := setupEngine()
	var t int
	if err != nil {
		fmt.Println("Failed to configure exchange test cannot continue")
		os.Exit(1)
	} else {
		t = m.Run()
	}
	err = cleanup()
	if err != nil {
		fmt.Printf("Clean up failed %v", err)
	}
	os.Exit(t)
}

func TestExchangeOrderbook(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeOrderbook(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}
	_, err = gct.ExchangeOrderbook()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangeTicker(t *testing.T) {
	t.Parallel()
	_, err := gct.ExchangeTicker(exch, currencyPair, delimiter, assetType)
	if err != nil {
		t.Fatal(err)
	}
	_, err = gct.ExchangeTicker()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangeExchanges(t *testing.T) {
	t.Parallel()

	_, err := gct.ExchangeExchanges(tv)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangeExchanges()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestExchangePairs(t *testing.T) {
	t.Parallel()

	_, err := gct.ExchangePairs(exch, tv, assetType)
	if err != nil {
		t.Fatal(err)
	}

	_, err = gct.ExchangePairs()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}
}

func TestAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := gct.ExchangeAccountInfo()
	if !errors.Is(err, objects.ErrWrongNumArguments) {
		t.Fatal(err)
	}

	_, err = gct.ExchangeAccountInfo(exch)
	if err != nil {
		// This is a ghetto fix, but you know 100% test coverage and all
		if err.Error() != "exchange BTC Markets authenticated HTTP request called but not supported due to unset/default API keys" {
			t.Fatal(err)
		}
	}
}

func setupEngine() (err error) {
	engine.Bot, err = engine.NewFromSettings(&settings)
	if engine.Bot == nil || err != nil {
		return err
	}

	return engine.Bot.Start()
}

func cleanup() (err error) {
	err = os.RemoveAll(settings.DataDir)
	if err != nil {
		return
	}
	return nil
}
