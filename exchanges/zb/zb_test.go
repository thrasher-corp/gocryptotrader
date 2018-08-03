package zb

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/idoall/gocryptotrader/config"
)

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "zb",
		Enabled:                 true,
		Verbose:                 true,
		Websocket:               false,
		BaseAsset:               "eth",
		QuoteAsset:              "usdt",
		UseSandbox:              false,
		RESTPollingDelay:        10,
		HTTPTimeout:             5 * time.Second,
		AuthenticatedAPISupport: true,
		APIKey:                  "",
		APISecret:               "",
		ClientID:                "",
		ConfigCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: false,
			Delimiter: "-",
		},
	}
}

var z ZB

func TestSetDefaults(t *testing.T) {
	z.SetDefaults()
}

func TestSetup(t *testing.T) {
	z.Setup(getDefaultConfig())
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	arg := SpotNewOrderRequestParams{
		Symbol: z.GetSymbol(),
		Type:   SpotNewOrderRequestParamsTypeSell,
		Amount: 0.01,
		Price:  10246.1,
	}
	orderid, err := z.SpotNewOrder(arg)
	if err != nil {
		t.Errorf("Test failed - ZB SpotNewOrder: %s", err)
	} else {
		fmt.Println(orderid)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	err := z.CancelOrder(20180629145864850)
	if err != nil {
		t.Errorf("Test failed - ZB CancelOrder: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := z.GetLatestSpotPrice(z.GetSymbol())
	if err != nil {
		t.Errorf("Test failed - ZB GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := z.GetTicker(z.GetSymbol())
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := z.GetMarkets()
	if err != nil {
		t.Errorf("Test failed - ZB GetMarkets: %s", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	res, err := z.GetAccountInfo()
	if err != nil {
		t.Errorf("Test failed - ZB GetAccountInfo: %s", err)
	} else {
		for _, v := range res.Result.Coins {
			b, _ := json.Marshal(v)
			fmt.Printf("%s \n", b)
		}
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	TestSetDefaults(t)
	TestSetup(t)

	arg := KlinesRequestParams{
		Symbol: z.GetSymbol(),
		Type:   TimeInterval_FiveMinutes,
		Size:   10,
	}
	_, err := z.GetSpotKline(arg)
	if err != nil {
		t.Errorf("Test failed - ZB GetSpotKline: %s", err)
	}
}
