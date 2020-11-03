package stream

import (
	"fmt"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestAddSuccessfulSubscriptions(t *testing.T) {
	m := SubscriptionManager{
		m: make(map[Subscription]*[]ChannelSubscription),
	}
	err := m.AddSuccessfulSubscriptions([]ChannelSubscription{
		{
			Channel: "test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveSuccessfulUnsubscriptions(t *testing.T) {
	m := SubscriptionManager{
		m: make(map[Subscription]*[]ChannelSubscription),
	}
	err := m.AddSuccessfulSubscriptions([]ChannelSubscription{
		{
			Channel: "test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range m.m {
		fmt.Println(k, *v)
	}

	err = m.RemoveSuccessfulUnsubscriptions([]ChannelSubscription{
		{
			Channel: "test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range m.m {
		fmt.Println(k, *v)
	}
}

func TestGetAllSubscriptions(t *testing.T) {
	m := SubscriptionManager{
		m: make(map[Subscription]*[]ChannelSubscription),
	}
	err := m.AddSuccessfulSubscriptions([]ChannelSubscription{
		{
			Channel: "test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wow := m.GetAllSubscriptions()
	fmt.Println(wow)
}

func TestGetAssetsBySubscriptionType(t *testing.T) {
	m := SubscriptionManager{
		m: make(map[Subscription]*[]ChannelSubscription),
	}
	p, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	err = m.AddSuccessfulSubscriptions([]ChannelSubscription{
		{
			Channel:          "test",
			SubscriptionType: 1,
			Asset:            asset.Spot,
			Currency:         p,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wow, err := m.GetAssetsBySubscriptionType(1, p)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(wow)
}
