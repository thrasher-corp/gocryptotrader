package bithumb

import (
	"testing"
)

var (
	a                 = `{"status":"5100","resmsg":"Invalid Filter Syntax"}`
	wsSuccesfulFilter = `{"status":"0000","resmsg":"Filter Registered Successfully"}`
	wsTickerResp      = []byte(`{"type":"ticker","content":{"tickType":"24H","date":"20210811","time":"132017","openPrice":"33400","closePrice":"34010","lowPrice":"32660","highPrice":"34510","value":"45741663716.89916828275244531","volume":"1359398.496892086826189907","sellVolume":"198021.237915860451480504","buyVolume":"1161377.258976226374709403","prevClosePrice":"33530","chgRate":"1.83","chgAmt":"610","volumePower":"500","symbol":"UNI_KRW"}}`)
)

func TestWsHandleData(t *testing.T) {
	welcomeMsg := []byte(`{"status":"0000","resmsg":"Connected Successfully"}`)
	err := b.wsHandleData(welcomeMsg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	sub, err := b.GenerateSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	if sub == nil {
		t.Fatal("unexpected value")
	}
}

type dummyConn struct{}

func TestSubscribe(t *testing.T) {

}
