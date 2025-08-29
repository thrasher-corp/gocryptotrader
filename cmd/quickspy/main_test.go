package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchange/quickspy"
)

func TestParseFocusType(t *testing.T) {
	cases := []struct {
		in    string
		ft    quickspy.FocusType
		useWS bool
	}{
		{"ticker", quickspy.TickerFocusType, true},
		{"orderbook", quickspy.OrderBookFocusType, true},
		{"ob", quickspy.OrderBookFocusType, true},
		{"kline", quickspy.KlineFocusType, false},
		{"trades", quickspy.TradesFocusType, false},
		{"openinterest", quickspy.OpenInterestFocusType, false},
		{"fundingrate", quickspy.FundingRateFocusType, false},
		{"accountholdings", quickspy.AccountHoldingsFocusType, false},
		{"activeorders", quickspy.ActiveOrdersFocusType, false},
		{"orderexecution", quickspy.OrderExecutionFocusType, false},
		{"url", quickspy.URLFocusType, false},
		{"contract", quickspy.ContractFocusType, false},
		{"unknown", quickspy.UnsetFocusType, false},
	}
	for _, c := range cases {
		got, ws := parseFocusType(c.in)
		if got != c.ft || ws != c.useWS {
			t.Fatalf("parseFocusType(%q) = (%v,%v), want (%v,%v)", c.in, got, ws, c.ft, c.useWS)
		}
	}
}

func TestFallbackPoll(t *testing.T) {
	if got := fallbackPoll(0); got != 5*time.Second {
		t.Fatalf("fallbackPoll(0) = %v, want 5s", got)
	}
	if got := fallbackPoll(-1 * time.Second); got != 5*time.Second {
		t.Fatalf("fallbackPoll(-1s) = %v, want 5s", got)
	}
	if got := fallbackPoll(1500 * time.Millisecond); got != 1500*time.Millisecond {
		t.Fatalf("fallbackPoll(1.5s) = %v, want 1.5s", got)
	}
}

func TestRequiresAuth(t *testing.T) {
	if !requiresAuth(quickspy.AccountHoldingsFocusType) {
		t.Fatalf("requiresAuth(AccountHoldings) = false, want true")
	}
	if !requiresAuth(quickspy.ActiveOrdersFocusType) {
		t.Fatalf("requiresAuth(ActiveOrders) = false, want true")
	}
	if !requiresAuth(quickspy.OrderPlacementFocusType) {
		t.Fatalf("requiresAuth(OrderPlacement) = false, want true")
	}
	if requiresAuth(quickspy.TickerFocusType) {
		t.Fatalf("requiresAuth(Ticker) = true, want false")
	}
}

func TestEmitWritesNDJSON(t *testing.T) {
	var buf bytes.Buffer
	enc = json.NewEncoder(&buf)
	now := time.Now().UTC()
	ev := eventEnvelope{Timestamp: now, Focus: "ticker", Data: map[string]any{"x": 1}}
	emit(ev)
	if buf.Len() == 0 {
		t.Fatalf("emit() wrote nothing")
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["focus"].(string) != "ticker" {
		t.Fatalf("unexpected focus: %v", m["focus"])
	}
}
