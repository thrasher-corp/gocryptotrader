package engine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

func TestPlatformHistory(t *testing.T) {
	var p = PlatformHistory{}

	err := p.ValidatData()
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	err = p.Sort()
	if err == nil {
		t.Error("Test Failed - PlatformHistory Sort() error cannot be nil")
	}

	tn := time.Now()

	p = PlatformHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2"},
		{Timestamp: tn.Add(time.Minute), TID: "1"},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3"},
	}

	err = p.ValidatData()
	if err == nil {
		t.Error("Test Failed - PlatformHistory Sort() error cannot be nil")
	}

	err = p.Sort()
	if err != nil {
		t.Error("Test Failed - PlatformHistory Sort() error", err)
	}

	if p[0].TID != "1" {
		t.Errorf("Test Failed - PlatformHistory Sort() error expected 1 but received %s",
			p[0].TID)
	}

	if p[1].TID != "2" {
		t.Errorf("Test Failed - PlatformHistory Sort() error expected 2 but received %s",
			p[1].TID)
	}

	p = PlatformHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 0},
	}

	err = p.ValidatData()
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	p = PlatformHistory{
		{TID: "2", Amount: 1, Price: 0},
	}

	err = p.ValidatData()
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	p = PlatformHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 1000},
		{Timestamp: tn.Add(time.Minute), TID: "1", Amount: 1, Price: 1001},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3", Amount: 1, Price: 1001.5},
	}

	err = p.ValidatData()
	if err != nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error", err)
	}
}

func TestOHLC(t *testing.T) {
	var p PlatformHistory
	rand.Seed(time.Now().Unix())
	for i := 0; i < 24000; i++ {
		p = append(p, &exchange.PlatformTrade{
			Timestamp: time.Now().Add((time.Duration(rand.Intn(10)) * time.Minute) + (time.Duration(rand.Intn(10)) * time.Second)),
			TID:       common.HexEncodeToString([]byte(string(i))),
			Amount:    float64(rand.Intn(20)) + 1,
			Price:     1000 + float64(rand.Intn(1000)),
		})
	}

	_, err := CreateOHLC(p, 5*time.Minute)
	if err != nil {
		t.Error("Test Failed - CreateOHLC error", err)
	}
}
