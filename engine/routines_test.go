package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

func TestWebsocketDataHandlerProcess(t *testing.T) {
	ws := wshandler.New()
	err := ws.Setup(&wshandler.WebsocketSetup{Enabled: true})
	if err != nil {
		t.Error(err)
	}
	go WebsocketDataReceiver(ws)
	ws.DataHandler <- "string"
	time.Sleep(time.Second)
	close(shutdowner)
	wg.Wait()
}

func TestHandleData(t *testing.T) {
	SetupTest(t)
	Bot.Settings.Verbose = true
	var exchName = "exch"
	err := WebsocketDataHandler(exchName, wshandler.WebsocketNotEnabled)
	if err == nil {
		t.Error("Expected error")
	}
	err = WebsocketDataHandler(exchName, errors.New("error"))
	if err == nil {
		t.Error("Error not handled correctly")
	}
	err = WebsocketDataHandler(exchName, nil)
	if err == nil {
		t.Error("Expected nil data error")
	}
	err = WebsocketDataHandler(exchName, wshandler.TradeData{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, wshandler.FundingData{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, &ticker.Price{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, wshandler.KlineData{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, wshandler.WebsocketOrderbookUpdate{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, &order.Detail{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, &order.Cancel{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, &order.Modify{})
	if err != nil {
		t.Error(err)
	}
	err = WebsocketDataHandler(exchName, order.Stop)
	if err != nil {
		t.Error(err)
	}
	Bot.Settings.Verbose = false
}
