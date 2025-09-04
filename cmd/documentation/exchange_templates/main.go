package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/quickspy"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	focusData := quickspy.NewFocusData(quickspy.TickerFocusType, false, true, time.Second)
	focusList := []*quickspy.FocusData{focusData}
	k := &quickspy.CredentialsKey{
		ExchangeAssetPair: key.NewExchangeAssetPair("binance", asset.Spot, currency.NewBTCUSDT()),
	}
	q, err := quickspy.NewQuickSpy(
		ctx,
		k,
		focusList)
	if err != nil {
		log.Fatal(err)
	}
	if err := q.WaitForInitialData(ctx, quickspy.TickerFocusType); err != nil {
		log.Fatal(err)
	}
	d, err := q.DumpJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", d)
}
