package bitget

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

type fixtureConnection struct {
	websocket.Connection
}

func (c fixtureConnection) RequireMatchWithData(any, []byte) error { return nil }

func TestWsConnect(t *testing.T) {
	// exch := &Exchange{}
	// exch.Websocket = sharedtestvalues.NewTestWebsocket()
	// err := exch.Websocket.Disable()
	// assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled)
	// err = exch.WsConnect()
	// assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled)
	// exch.SetDefaults()
	// err = exchangeBaseHelper(exch)
	// require.NoError(t, err)
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// exch.Verbose = true
	// err = exch.WsConnect()
	// assert.NoError(t, err)
}

func TestWsAuth(t *testing.T) {
	// e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	// err := e.WsAuth(t.Context(), nil)
	// assert.ErrorIs(t, err, errAuthenticatedWebsocketDisabled)
	// if e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
	// 	t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	// }
	// e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	// var dialer gws.Dialer
	// go func() {
	// 	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	// 	select {
	// 	case resp := <-e.Websocket.DataHandler:
	// 		t.Errorf("%+v\n%T\n", resp, resp)
	// 	case <-timer.C:
	// 	}
	// 	timer.Stop()
	// 	for {
	// 		<-e.Websocket.DataHandler
	// 	}
	// }()
	// err = e.WsAuth(t.Context(), &dialer)
	// require.NoError(t, err)
	// time.Sleep(sharedtestvalues.WebsocketResponseDefaultTimeout)
}

// func TestWsReadData(t *testing.T) {
// 	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
// 		tb.Helper()
// 		msg, err := json.Marshal("pong")
// 		require.NoError(t, err)
// 		return w.WriteMessage(gws.TextMessage, msg)
// 	}
// 	wsTest := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mock))
// 	wsTest.Websocket.Enable()
// 	err := exchangeBaseHelper(wsTest)
// 	require.NoError(t, err)
// 	var dialer gws.Dialer
// 	err = wsTest.Websocket.Conn.Dial(context.TODO(), &dialer, http.Header{})
// 	require.NoError(t, err)
// 	err = wsTest.Websocket.AuthConn.Dial(context.TODO(), &dialer, http.Header{})
// 	require.NoError(t, err)
// 	// e.Websocket.Wg.Add(1)
// 	// go e.wsReadData(e.Websocket.Conn)
// 	err = wsTest.Subscribe(defaultSubscriptions)
// 	require.NoError(t, err)
// 	// Implement internal/testing/websocket mockws stuff after merging
// 	// See: https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/kraken/kraken_test.go#L1169
// }

func TestWsHandleData(t *testing.T) {
	// // Not sure what issues this is preventing. If you figure that out, add a comment about it
	// ch := make(chan struct{})
	// t.Cleanup(func() {
	// 	close(ch)
	// })
	// go func() {
	// 	for {
	// 		select {
	// 		case <-e.Websocket.DataHandler.C:
	// 			continue
	// 		case <-ch:
	// 			return
	// 		}
	// 	}
	// }()
	verboseTemp := e.Verbose
	e.Verbose = true
	t.Cleanup(func() {
		e.Verbose = verboseTemp
	})
	mockJSON := []byte(`pong`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`notjson`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	errInvalidChar := "invalid char"
	assert.ErrorContains(t, err, errInvalidChar)
	mockJSON = []byte(`{"event":"subscribe"}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"error"}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	expectedErr := fmt.Sprintf(errWebsocketGeneric, "Bitget", 0, "")
	assert.EqualError(t, err, expectedErr)
	mockJSON = []byte(`{"event":"login","code":0}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"login","code":1}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err, "should not error on failed login, this is passed back to the authenticate handler")
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fakeChannelNotReal"}}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"fakeChannelNotReal"}}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"fakeEventNotReal"}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestTickerDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"moo"},"data":[{"InstId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestCandleDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"candle1D"},"data":[["1","2","3","4","5","6","",""]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["a","2","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","a","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","a","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","a","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","a","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","a","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","6","",""]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"candle1D"},"data":[[[{}]]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
}

func TestTradeDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"trade"},"data":[]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
}

func TestOrderbookDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"books"},"data":[]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, common.ErrNoResults)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"bids":[["a","1"]]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","a"]]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, orderbook.ErrAssetTypeNotSet)
	mockJSON = []byte(`{"action":"update","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","2"]]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound)
	mockJSON = []byte(`{"action":"snapshot","arg":{"instType":"SPOT","channel":"books","instId":"A2ZUSDT"},"data":[{"asks":[["0.003511","15645.9"],["0.003512","52415"],["0.003516","5587.8"],["0.003518","5815.6"],["0.00352","3188.7"],["0.003522","5587.8"],["0.003523","37748.8"],["0.00353","59212.8"],["0.003531","98834.6"],["0.003532","43095"],["0.003533","245.7"],["0.003534","41790.4"],["0.003536","57977.1"],["0.003542","47431.1"],["0.003547","104312.9"],["0.00355","152431.1"],["0.003552","319.9"],["0.003553","126256.5"],["0.003554","227.8"],["0.00356","200598.8"],["0.003563","88503.8"],["0.003566","135132.6"],["0.00357","298.3"],["0.003571","245.7"],["0.003572","136390.3"],["0.003577","266853.4"],["0.003579","230739.2"],["0.003581","136699.7"],["0.003586","282217.9"],["0.00359","227.8"],["0.003594","247114.5"],["0.003605","125145.3"],["0.003606","121011.8"],["0.003608","245.7"],["0.00361","196333.9"],["0.003614","202658.6"],["0.003619","216611.2"],["0.00362","9700.5"],["0.003626","7767"],["0.003627","126612.1"],["0.003632","23142.6"],["0.003635","177772.1"],["0.00364","271.2"],["0.003646","245.7"],["0.003647","189904.7"],["0.003648","364655.4"],["0.003651","245293.6"],["0.003659","183895.7"],["0.003662","139249.6"],["0.003663","227.8"],["0.003665","169435.8"],["0.003668","319.9"],["0.003669","141253.1"],["0.003679","1756"],["0.003684","245.7"],["0.003686","196014"],["0.003696","298.3"],["0.003699","227.8"],["0.003703","277049"],["0.003704","271.2"],["0.003721","544.2"],["0.003735","227.8"],["0.003749","196014"],["0.003759","241076.4"],["0.003764","75593"],["0.003768","271.2"],["0.003771","227.8"],["0.003774","172383.1"],["0.003788","206982"],["0.003796","245.7"],["0.0038","205980.8"],["0.003807","143349.8"],["0.003822","298.3"],["0.003823","88320.6"],["0.003832","271.2"],["0.003834","245.7"],["0.003843","72634.1"],["0.003853","96284.5"],["0.003872","315756.7"],["0.003879","227.8"],["0.003896","271.2"],["0.003902","90984.6"],["0.003909","245.7"],["0.003914","97689.6"],["0.003915","116452"],["0.003916","80609.2"],["0.003924","76618.6"],["0.003925","94177"],["0.003926","84599.7"],["0.003947","27255.4"],["0.003948","298.3"],["0.003949","71031.8"],["0.003951","227.8"],["0.003959","271.2"],["0.003984","245.7"],["0.003987","227.8"],["0.00399","64646.9"],["0.003993","94670.4"],["0.003999","80609.2"],["0.004004","87792.2"],["0.004015","70233.7"],["0.004022","245.7"],["0.004023","499"],["0.004033","242904.5"],["0.004059","227.8"],["0.00406","245.7"],["0.004074","298.3"],["0.004077","89388.4"],["0.004087","271.2"],["0.00409","278877.2"],["0.004095","227.8"],["0.004097","245.7"],["0.004101","360721.7"],["0.004104","533504.3"],["0.004112","326215.7"],["0.004114","281908.5"],["0.004115","272814.7"],["0.004131","227.8"],["0.004135","909628.1"],["0.004142","345565.3"],["0.004143","251595.8"],["0.004151","271.2"],["0.004167","227.8"],["0.004168","266752.1"],["0.004172","245.7"],["0.0042","298.3"],["0.004204","227.8"],["0.00421","245.7"],["0.004214","271.2"],["0.00424","227.8"],["0.004248","245.7"],["0.004276","227.8"],["0.004278","271.2"],["0.004285","245.7"],["0.004312","227.8"],["0.004323","245.7"],["0.004325","298.3"],["0.004342","271.2"],["0.004348","227.8"],["0.00436","245.7"],["0.004384","227.8"],["0.004398","245.7"],["0.004406","271.2"],["0.00442","5179.8"],["0.004435","245.7"],["0.004451","298.3"],["0.004456","227.8"],["0.00447","271.2"],["0.004473","245.7"],["0.0045","49866.7"],["0.004511","245.7"],["0.004533","271.2"],["0.004548","245.7"],["0.004577","298.3"],["0.004586","245.7"],["0.004597","271.2"],["0.004623","245.7"],["0.004661","516.9"],["0.004703","298.3"],["0.005","4437.5"],["0.0053","272673.8"],["0.00584","2912.9"],["0.005867","511.3"],["0.00611","59729"],["0.006233","9023.4"],["0.00625","43379.7"],["0.006299","794.5"],["0.00631","104.8"],["0.006427","104.8"],["0.0065","71509.4"],["0.006545","104.8"],["0.006596","1430.7"],["0.00665","1000"],["0.006662","104.8"],["0.006666","153"],["0.00678","3025.1"],["0.00688","2345.1"],["0.006897","104.8"],["0.007","58496.6"],["0.007015","104.8"],["0.007132","104.8"],["0.0072","49929.3"],["0.00725","104.8"],["0.007367","104.8"],["0.007485","104.8"],["0.007602","104.8"],["0.007667","112606.3"],["0.007708","16344.8"],["0.00772","104.8"],["0.007799","19427.9"],["0.007837","104.8"],["0.007858","461.7"],["0.007867","381.3"],["0.007948","505"],["0.007955","104.8"],["0.008072","104.8"],["0.00819","104.8"],["0.008307","104.8"],["0.0088","370"],["0.00975","108.9"]],"bids":[["0.003481","28208.5"],["0.00348","14209"],["0.003479","19173.6"],["0.003478","747984.9"],["0.003458","245.7"],["0.003457","6510.3"],["0.003453","7566.9"],["0.003451","11000"],["0.003449","271.2"],["0.003448","28260.3"],["0.003446","7183.3"],["0.003438","62453.9"],["0.003437","77886.5"],["0.003436","43262.7"],["0.003434","32322.2"],["0.003433","46188.9"],["0.003427","7337"],["0.003426","53002.1"],["0.003424","77695.3"],["0.003422","32167.9"],["0.003421","245.7"],["0.00342","86811.5"],["0.003419","61516"],["0.003418","29717.6"],["0.003417","40898.5"],["0.003413","140478.8"],["0.00341","227.8"],["0.003405","173430.6"],["0.003401","84079.9"],["0.0034","102417.4"],["0.003396","6279"],["0.003395","136792.3"],["0.003385","32204.5"],["0.003383","245.7"],["0.003374","227.8"],["0.003351","73049.6"],["0.003345","245.7"],["0.003341","58820.8"],["0.00334","5988"],["0.003338","227.8"],["0.003324","123925.1"],["0.003321","120390.3"],["0.003319","298.3"],["0.003308","245.7"],["0.003302","227.8"],["0.0033","368952"],["0.003298","140535"],["0.003297","72992.5"],["0.003296","1038.4"],["0.003282","99141.6"],["0.00328","151957.9"],["0.003278","185570.7"],["0.00327","245.7"],["0.003267","18332.4"],["0.003266","227.8"],["0.003258","271.2"],["0.003257","106701.6"],["0.003242","95462.5"],["0.003233","85181.8"],["0.00323","227.8"],["0.003223","40531"],["0.003213","55789.7"],["0.003195","245.7"],["0.003194","499"],["0.003193","298.3"],["0.003169","54836"],["0.003158","227.8"],["0.003157","245.7"],["0.003135","12759"],["0.00313","271.2"],["0.003122","227.8"],["0.00312","245.7"],["0.003117","43324.7"],["0.003102","57576.2"],["0.003098","44740.6"],["0.003089","42915.1"],["0.003087","55442"],["0.003086","227.8"],["0.003084","50640.2"],["0.003082","245.7"],["0.00308","44797.1"],["0.003078","49485.2"],["0.003067","298.3"],["0.003066","271.2"],["0.003064","50544.5"],["0.003062","56743.4"],["0.003058","47949.8"],["0.003057","30215.4"],["0.003053","33564.8"],["0.00305","227.8"],["0.003047","27255.7"],["0.003045","245.7"],["0.003039","48969.7"],["0.003028","49590.8"],["0.003015","16583"],["0.003014","55442"],["0.003013","227.8"],["0.003007","245.7"],["0.003005","59279.1"],["0.003003","271.2"],["0.002977","227.8"],["0.002975","465087.6"],["0.002969","245.7"],["0.002941","526.1"],["0.002939","271.2"],["0.002937","355397.1"],["0.002934","47683.5"],["0.002932","245.7"],["0.002926","522126.7"],["0.002924","1026702.8"],["0.002923","58431"],["0.002912","434374.3"],["0.00291","44345.7"],["0.002905","227.8"],["0.002896","544064.8"],["0.002894","245.7"],["0.002886","52928.7"],["0.002884","386110.5"],["0.002877","399273.3"],["0.002876","465832.8"],["0.002875","491067.2"],["0.002869","227.8"],["0.002857","245.7"],["0.002856","487025.7"],["0.002853","19628"],["0.002852","508963.8"],["0.002849","368560"],["0.002833","227.8"],["0.002819","245.7"],["0.002811","271.2"],["0.002797","227.8"],["0.002782","245.7"],["0.002761","227.8"],["0.002747","271.2"],["0.002744","245.7"],["0.002738","40175"],["0.002706","245.7"],["0.002684","271.2"],["0.002669","245.7"],["0.002649","21140"],["0.002631","245.7"],["0.00262","271.2"],["0.002594","245.7"],["0.002573","27982"],["0.002556","516.9"],["0.002482","60435"],["0.002152","560.8"],["0.001833","818.3"],["0.001133","1765.2"],["0.0003","10370.7"],["0.000125","8000"]],"ts":"1763528714701","checksum":0,"seq":395681152}],"ts":1763528715503}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err, "should correctly handle orderbook snapshot")
	mockJSON = []byte(`{"action":"update","arg":{"instType":"SPOT","channel":"books","instId":"A2ZUSDT"},"data":[{"asks":[],"bids":[["0.00348","0"]],"ts":"1763528715515","checksum":1041484092,"seq":395681167}],"ts":1763528715517}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err, "should correctly handle orderbook update with checksum")
}

func TestAccountSnapshotDataHandler(t *testing.T) {
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:      "TestAccountSnapshotDataHandler",
		Secret:   "TestAccountSnapshotDataHandler",
		ClientID: "TestAccountSnapshotDataHandler",
	})
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account"},"data":[]}`)
	err := e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestFillDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"fill"},"data":[]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{"symbol":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"USDT-FUTURES"},"data":[{"symbol":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestGenOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders"},"data":[]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"USDT-FUTURES"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"USDT-FUTURES"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestTriggerOrderDatHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-algo"},"data":[]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"USDT-FUTURES"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestPositionsDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestPositionsHistoryDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{"instId":"BTCUSD"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestIndexPriceDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"index-price"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"index-price","instType":"spot"},"data":[{"symbol":"BTCUSDT"},{"symbol":"USDT/USDT"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestCrossAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:      "TestCrossAccountDataHandler",
		Secret:   "TestCrossAccountDataHandler",
		ClientID: "TestCrossAccountDataHandler",
	})
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[{}]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestMarginOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-isolated","instId":"BTCUSD"},"data":[{"feeDetail":[{}]}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed","instId":"BTCUSD"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestIsolatedAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[[]]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:      "TestIsolatedAccountDataHandler",
		Secret:   "TestIsolatedAccountDataHandler",
		ClientID: "TestIsolatedAccountDataHandler",
	})
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[{}]}`)
	err = e.wsHandleData(ctx, fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestAccountUpdateDataHandler(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	mockJSON := []byte(`{"event":"update","arg":{"channel":"account"},"data":[]}`)
	err := e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[{"uTime":"1750142570"}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = e.wsHandleData(t.Context(), fixtureConnection{}, mockJSON)
	assert.NoError(t, err)
}

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	ord := orderbook.Book{
		Asks: orderbook.Levels{{StrPrice: "3", StrAmount: "1"}},
		Bids: orderbook.Levels{{StrPrice: "4", StrAmount: "1"}},
	}
	resp := calculateUpdateOrderbookChecksum(&ord)
	assert.Equal(t, uint32(892106381), resp)
	ord.Asks = make(orderbook.Levels, 26)
	data := "3141592653589793238462643383279502884197169399375105"
	for i := range ord.Asks {
		ord.Asks[i] = orderbook.Level{
			StrPrice:  string(data[i*2]),
			StrAmount: string(data[i*2+1]),
		}
	}
	resp = calculateUpdateOrderbookChecksum(&ord)
	assert.Equal(t, uint32(2945115267), resp)
}
