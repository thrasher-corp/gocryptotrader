package bitflyer

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const canManipulateRealOrders = false

var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var e *Exchange

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bitflyer Setup error: %s", err)
	}

	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiCredentials)
	}
	if !mockTests {
		log.Printf(sharedtestvalues.LiveTesting, e.Name)
	}

	os.Exit(m.Run())
}

const (
	genesisBlockResponse  = `{"block_hash":"000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f","height":0,"is_main":true,"version":1,"prev_block":"0000000000000000000000000000000000000000000000000000000000000000","merkle_root":"4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b","timestamp":"2009-01-03T18:15:05Z","bits":486604799,"nonce":2083236893,"txnum":1,"total_fees":0,"tx_hashes":["4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"]}`
	recordedBlockResponse = `{"block_hash":"00000000000000000000ebfbb163cda54e9be3bd6322a821f0278a59e74ab303","height":924496,"is_main":true,"version":589856768,"prev_block":"00000000000000000000054cca3f6caf0fe809321a7586361fe89befc49c9fcf","merkle_root":"600ab5166d50eb5b7fb738c645f25e830ab68577465eeaee3f28c6cb2d5d71e2","timestamp":"2025-11-21T02:00:10Z","bits":385997110,"nonce":3009599600,"txnum":3367,"total_fees":3989756,"tx_hashes":["c995838345475b0e25841cbec9dee730c451f1004ebf3ee482c78ad78c7f7aaa","f343a7af8774151ee9afb2934ad55b245d17e695790cd672a7a437ee3a2fa849","624999b5ef7e528a6719c9623f296144e6d3d1b2b2475a640ecdd382c478240c"]}`
)

var (
	genesisBlock = ChainAnalysisBlockResponse{
		BlockHash:         "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
		IsMain:            true,
		Version:           1,
		PreviousBlock:     "0000000000000000000000000000000000000000000000000000000000000000",
		MerkleRoot:        "4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b",
		Timestamp:         time.Date(2009, time.January, 3, 18, 15, 5, 0, time.UTC),
		Bits:              486604799,
		Nonce:             2083236893,
		TransactionNumber: 1,
		TransactionHashes: []string{"4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"},
	}
	recordedBlock = ChainAnalysisBlockResponse{
		BlockHash:          "00000000000000000000ebfbb163cda54e9be3bd6322a821f0278a59e74ab303",
		Height:             924496,
		IsMain:             true,
		Version:            589856768,
		PreviousBlock:      "00000000000000000000054cca3f6caf0fe809321a7586361fe89befc49c9fcf",
		MerkleRoot:         "600ab5166d50eb5b7fb738c645f25e830ab68577465eeaee3f28c6cb2d5d71e2",
		Timestamp:          time.Date(2025, time.November, 21, 2, 0, 10, 0, time.UTC),
		Bits:               385997110,
		Nonce:              3009599600,
		TransactionNumber:  3367,
		TotalFeesInSatoshi: 3989756,
		TransactionHashes: []string{
			"c995838345475b0e25841cbec9dee730c451f1004ebf3ee482c78ad78c7f7aaa",
			"f343a7af8774151ee9afb2934ad55b245d17e695790cd672a7a437ee3a2fa849",
			"624999b5ef7e528a6719c9623f296144e6d3d1b2b2475a640ecdd382c478240c",
		},
	}
)

func chainAnalysisTestExchange(t *testing.T, requestURI, response string) *Exchange {
	t.Helper()
	if !mockTests {
		return e
	}
	var server *httptest.Server
	server = httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request method should be GET")
		assert.Equal(t, server.URL, "http://"+r.Host, "request should go to the chain analysis URL")
		assert.Equal(t, requestURI, r.URL.RequestURI(), "request URI should name the endpoint and its argument")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "writing the response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.ChainAnalysis.String(), server.URL+"/v1/"), "SetRunningURL must not error")
	return ex
}

func assertChainAnalysisBlock(t *testing.T, exp, got *ChainAnalysisBlockResponse) {
	t.Helper()
	if !mockTests {
		require.GreaterOrEqual(t, len(got.TransactionHashes), len(exp.TransactionHashes), "live block must list at least the recorded transaction hashes")
		got.TransactionHashes = got.TransactionHashes[:len(exp.TransactionHashes)]
	}
	assert.Equal(t, exp, got, "block should match the recorded response")
}

func TestGetLatestBlockChainAnalysis(t *testing.T) {
	t.Parallel()
	ex := chainAnalysisTestExchange(t, "/v1/block/latest", recordedBlockResponse)
	got, err := ex.GetLatestBlockChainAnalysis(t.Context())
	require.NoError(t, err, "GetLatestBlockChainAnalysis must not error")
	require.NotNil(t, got, "GetLatestBlockChainAnalysis must return a block")
	if !mockTests {
		assert.GreaterOrEqual(t, got.Height, recordedBlock.Height, "latest block height should not be below the recorded block")
		assert.EqualValues(t, got.TransactionNumber, len(got.TransactionHashes), "latest block should list every transaction hash")
		return
	}
	assert.Equal(t, &recordedBlock, got, "GetLatestBlockChainAnalysis should decode the latest block")
}

func TestGetBlockChainAnalysis(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		blockHash  string
		requestURI string
		response   string
		exp        ChainAnalysisBlockResponse
	}{
		{
			name:       "genesis block",
			blockHash:  "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			requestURI: "/v1/block/000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			response:   genesisBlockResponse,
			exp:        genesisBlock,
		},
		{
			name:       "block with fees",
			blockHash:  "000000000000000001434fb0e5cc5e521de59bff70c325185a5c67bd5418c61a",
			requestURI: "/v1/block/000000000000000001434fb0e5cc5e521de59bff70c325185a5c67bd5418c61a",
			response:   `{"block_hash":"000000000000000001434fb0e5cc5e521de59bff70c325185a5c67bd5418c61a","height":471561,"is_main":true,"version":536870912,"prev_block":"000000000000000000a1782a9d78672672e4291347c47ac3206ed8716bb4952b","merkle_root":"417c5c54b0b8c77731e8b346dfd3d7e2f7285f6be13f92bb7ab8dc2bbda1e6a5","timestamp":"2017-06-16T15:46:26Z","bits":402759343,"nonce":4044123665,"txnum":1831,"total_fees":207976286,"tx_hashes":["f7dccccc239415dc637b5ac69e8e82d2912428c323d2760cbea23609167538d9","3d331a285cce965e0fc0fe0d2d4de8c50a07f03b33faf2406f431acae2bb8673","ebd99b10cc5d62fb9410cfb35482da5a2a25dcbf779e4ec745e169851baf416e"]}`,
			exp: ChainAnalysisBlockResponse{
				BlockHash:          "000000000000000001434fb0e5cc5e521de59bff70c325185a5c67bd5418c61a",
				Height:             471561,
				IsMain:             true,
				Version:            536870912,
				PreviousBlock:      "000000000000000000a1782a9d78672672e4291347c47ac3206ed8716bb4952b",
				MerkleRoot:         "417c5c54b0b8c77731e8b346dfd3d7e2f7285f6be13f92bb7ab8dc2bbda1e6a5",
				Timestamp:          time.Date(2017, time.June, 16, 15, 46, 26, 0, time.UTC),
				Bits:               402759343,
				Nonce:              4044123665,
				TransactionNumber:  1831,
				TotalFeesInSatoshi: 207976286,
				TransactionHashes: []string{
					"f7dccccc239415dc637b5ac69e8e82d2912428c323d2760cbea23609167538d9",
					"3d331a285cce965e0fc0fe0d2d4de8c50a07f03b33faf2406f431acae2bb8673",
					"ebd99b10cc5d62fb9410cfb35482da5a2a25dcbf779e4ec745e169851baf416e",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := chainAnalysisTestExchange(t, tc.requestURI, tc.response)
			got, err := ex.GetBlockChainAnalysis(t.Context(), tc.blockHash)
			require.NoError(t, err, "GetBlockChainAnalysis must not error")
			require.NotNil(t, got, "GetBlockChainAnalysis must return a block")
			assertChainAnalysisBlock(t, &tc.exp, got)
		})
	}
}

func TestGetBlockByHeightChainAnalysis(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		height     uint64
		requestURI string
		response   string
		exp        ChainAnalysisBlockResponse
	}{
		{name: "genesis height", height: genesisBlock.Height, requestURI: "/v1/block/height/0", response: genesisBlockResponse, exp: genesisBlock},
		{name: "recorded block height", height: recordedBlock.Height, requestURI: "/v1/block/height/924496", response: recordedBlockResponse, exp: recordedBlock},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := chainAnalysisTestExchange(t, tc.requestURI, tc.response)
			got, err := ex.GetBlockByHeightChainAnalysis(t.Context(), tc.height)
			require.NoError(t, err, "GetBlockByHeightChainAnalysis must not error")
			require.NotNil(t, got, "GetBlockByHeightChainAnalysis must return a block")
			assertChainAnalysisBlock(t, &tc.exp, got)
		})
	}
}

func TestGetTransactionByHashChainAnalysis(t *testing.T) {
	t.Parallel()
	const documentedResponse = `{"tx_hash":"0562d1f063cd4127053d838b165630445af5e480ceb24e1fd9ecea52903cb772","block_height":370470,"confirmed":325,"fees":20000,"size":257,"received_date":"2015-08-19T00:46:01.017","version":1,"lock_time":0,"inputs":[{"prev_hash":"d43008e8019a641565615ec02b8c7d107fc470c4d500dcbb9299756afb9ee4b5","prev_index":1,"value":350080000,"script":"473044022062c6c81df6726573297c2288b7ed7be1d85c5b79ccf7e79a5c808c0e13b5e4f202200f2ae49c1119b2d74239086d12ec3ce96e408fb24adffe0dbb2424fbd7d1da27014104e8c8c15f6b714b46ed7ee49c023c009de5ed5b98bd60ca2d762be235274761f1db206f85b0072cc3ff62f727492740bfe0879d6135fe63b63454fe8aab268e13","address":"1ygL6TVoUiGXVGcjeGLee5zsarrbqYgwk","sequence":4294967295}],"outputs":[{"value":60000,"script":"76a914b066879afa8a2e35419a4826a6f8702438c5fc4388ac","address":"1H5ikemcbo71w1JXw1J3J8aSuPHwABEHEH"},{"value":350000000,"script":"76a914e84e80236d767cb7b4c344e3007af25ea7a5f12788ac","address":"1NBKopSCvBSD5vrAamyrgQ7W9a9wLpo4Gx"}]}`
	documented := ChainAnalysisTransactionResponse{
		TransactionHash: "0562d1f063cd4127053d838b165630445af5e480ceb24e1fd9ecea52903cb772",
		BlockHeight:     370470,
		Confirmed:       325,
		FeesInSatoshi:   20000,
		SizeInBytes:     257,
		ReceivedDate:    "2015-08-19T00:46:01.017",
		Version:         1,
		Inputs: []ChainAnalysisTransactionInput{{
			PreviousHash:   "d43008e8019a641565615ec02b8c7d107fc470c4d500dcbb9299756afb9ee4b5",
			PreviousIndex:  1,
			ValueInSatoshi: 350080000,
			Script:         "473044022062c6c81df6726573297c2288b7ed7be1d85c5b79ccf7e79a5c808c0e13b5e4f202200f2ae49c1119b2d74239086d12ec3ce96e408fb24adffe0dbb2424fbd7d1da27014104e8c8c15f6b714b46ed7ee49c023c009de5ed5b98bd60ca2d762be235274761f1db206f85b0072cc3ff62f727492740bfe0879d6135fe63b63454fe8aab268e13",
			Address:        "1ygL6TVoUiGXVGcjeGLee5zsarrbqYgwk",
			Sequence:       4294967295,
		}},
		Outputs: []ChainAnalysisTransactionOutput{
			{ValueInSatoshi: 60000, Script: "76a914b066879afa8a2e35419a4826a6f8702438c5fc4388ac", Address: "1H5ikemcbo71w1JXw1J3J8aSuPHwABEHEH"},
			{ValueInSatoshi: 350000000, Script: "76a914e84e80236d767cb7b4c344e3007af25ea7a5f12788ac", Address: "1NBKopSCvBSD5vrAamyrgQ7W9a9wLpo4Gx"},
		},
	}
	withLockTime := func(lockTime uint64) ChainAnalysisTransactionResponse {
		tx := documented
		tx.LockTime = lockTime
		return tx
	}
	notInBlock := documented
	notInBlock.BlockHeight, notInBlock.Confirmed = -1, 0
	for _, tc := range []struct {
		name     string
		txHash   string
		response string
		mockOnly bool
		exp      ChainAnalysisTransactionResponse
	}{
		{name: "documented transaction", txHash: documented.TransactionHash, response: documentedResponse, exp: documented},
		{name: "block height lock time", txHash: documented.TransactionHash, response: strings.Replace(documentedResponse, `"lock_time":0,`, `"lock_time":370469,`, 1), mockOnly: true, exp: withLockTime(370469)},
		{name: "unix time lock time", txHash: documented.TransactionHash, response: strings.Replace(documentedResponse, `"lock_time":0,`, `"lock_time":1439945161,`, 1), mockOnly: true, exp: withLockTime(1439945161)},
		{name: "transaction not in a block", txHash: documented.TransactionHash, response: strings.Replace(documentedResponse, `"block_height":370470,"confirmed":325,`, `"block_height":-1,"confirmed":0,`, 1), mockOnly: true, exp: notInBlock},
		{
			name:     "coinbase transaction",
			txHash:   "f7dccccc239415dc637b5ac69e8e82d2912428c323d2760cbea23609167538d9",
			response: `{"tx_hash":"f7dccccc239415dc637b5ac69e8e82d2912428c323d2760cbea23609167538d9","block_height":471561,"confirmed":453110,"fees":0,"size":182,"received_date":"2017-06-16T15:45:56.517","version":1,"lock_time":0,"inputs":[{"prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","prev_index":-1,"value":0,"script":"03093207122f5669614254432f4249503130302f42322f36fabe6d6d2b0cacff8fdcd81883e1ae71614b2221c8f7ff55d88c7773a12a466ae3af86a70100000000000000092f4542312f4144362f1202c58830cea2134b635a8a7612bbc5400000","address":"","sequence":4294967295}],"outputs":[{"value":1457976286,"script":"76a914536ffa992491508dca0354e52f32a3a7a679a53a88ac","address":"18cBEMRxXHqzWWCxZNtU91F5sbUNKhL5PX"}]}`,
			exp: ChainAnalysisTransactionResponse{
				TransactionHash: "f7dccccc239415dc637b5ac69e8e82d2912428c323d2760cbea23609167538d9",
				BlockHeight:     471561,
				Confirmed:       453110,
				SizeInBytes:     182,
				ReceivedDate:    "2017-06-16T15:45:56.517",
				Version:         1,
				Inputs: []ChainAnalysisTransactionInput{{
					PreviousHash:  "0000000000000000000000000000000000000000000000000000000000000000",
					PreviousIndex: -1,
					Script:        "03093207122f5669614254432f4249503130302f42322f36fabe6d6d2b0cacff8fdcd81883e1ae71614b2221c8f7ff55d88c7773a12a466ae3af86a70100000000000000092f4542312f4144362f1202c58830cea2134b635a8a7612bbc5400000",
					Sequence:      4294967295,
				}},
				Outputs: []ChainAnalysisTransactionOutput{
					{ValueInSatoshi: 1457976286, Script: "76a914536ffa992491508dca0354e52f32a3a7a679a53a88ac", Address: "18cBEMRxXHqzWWCxZNtU91F5sbUNKhL5PX"},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("response is synthetic and only runs mocked")
			}
			ex := chainAnalysisTestExchange(t, "/v1/tx/"+tc.txHash, tc.response)
			got, err := ex.GetTransactionByHashChainAnalysis(t.Context(), tc.txHash)
			require.NoError(t, err, "GetTransactionByHashChainAnalysis must not error")
			require.NotNil(t, got, "GetTransactionByHashChainAnalysis must return a transaction")
			if !mockTests {
				assert.GreaterOrEqual(t, got.Confirmed, tc.exp.Confirmed, "Confirmed should not be below the recorded count")
				_, err = time.Parse("2006-01-02T15:04:05.999", got.ReceivedDate)
				assert.NoError(t, err, "ReceivedDate should be a date and time without a zone designator")
				got.Confirmed, got.ReceivedDate = tc.exp.Confirmed, tc.exp.ReceivedDate
			}
			assert.Equal(t, &tc.exp, got, "GetTransactionByHashChainAnalysis should decode the requested transaction")
		})
	}
}

func TestGetAddressInfoChainAnalysis(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		address  string
		response string
		mockOnly bool
		liveOnly bool
		exp      ChainAnalysisAddressResponse
	}{
		{
			name:     "documented address",
			address:  "1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY",
			response: `{"address":"1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY","unconfirmed_balance":1190111,"confirmed_balance":1190111}`,
			exp:      ChainAnalysisAddressResponse{Address: "1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY", UnconfirmedBalance: 1190111, ConfirmedBalance: 1190111},
		},
		{
			name:     "unconfirmed deposit",
			address:  "1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY",
			response: `{"address":"1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY","unconfirmed_balance":1240111,"confirmed_balance":1190111}`,
			mockOnly: true,
			exp:      ChainAnalysisAddressResponse{Address: "1MrrmuErZXxVMo8Agj9CgQhPBmxEEt6DgY", UnconfirmedBalance: 1240111, ConfirmedBalance: 1190111},
		},
		{name: "bech32 donation address", address: core.BitcoinDonationAddress, liveOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("response is synthetic and only runs mocked")
			}
			if tc.liveOnly && mockTests {
				t.Skip("no recorded response, only runs live")
			}
			ex := chainAnalysisTestExchange(t, "/v1/address/"+tc.address, tc.response)
			got, err := ex.GetAddressInfoChainAnalysis(t.Context(), tc.address)
			require.NoError(t, err, "GetAddressInfoChainAnalysis must not error")
			require.NotNil(t, got, "GetAddressInfoChainAnalysis must return an address")
			if !mockTests {
				assert.Equal(t, tc.address, got.Address, "GetAddressInfoChainAnalysis should return the requested address")
				return
			}
			assert.Equal(t, &tc.exp, got, "GetAddressInfoChainAnalysis should decode the requested address")
		})
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	markets, err := e.GetMarkets(t.Context())
	if err != nil {
		t.Error("Bitflyer - GetMarkets() error:", err)
	}
	for _, market := range markets {
		if market.ProductCode == "" {
			t.Error("Bitflyer - ProductCode is empty in GetMarkets()")
		}
		if market.MarketType == "" {
			t.Error("Bitflyer - MarketType is empty in GetMarkets()")
		}
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetOrderBook() error:", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetTicker() error:", err)
	}
}

func TestUpdateTickerMocked(t *testing.T) {
	t.Parallel()
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "BTC_JPY", r.URL.Query().Get("product_code"), "product_code should be the requested pair")
		// BTC_JPY's volume counts every BTC product, FX_BTC_JPY's 1314.72645136 included
		_, err := w.Write([]byte(`{"product_code":"BTC_JPY","state":"RUNNING","timestamp":"2026-09-17T05:53:16.77","tick_id":405559,"best_bid":11907656.0,"best_ask":11910855.0,"best_bid_size":0.1032,"best_ask_size":0.01,"total_bid_depth":111.53187895,"total_ask_depth":130.68945887,"market_bid_size":0.0,"market_ask_size":0.0,"ltp":11905672.0,"volume":1511.32732944,"volume_by_product":196.60087808}`))
		assert.NoError(t, err, "writing the ticker response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	pair := currency.NewPairWithDelimiter("BTC", "JPY", "_")
	got, err := ex.UpdateTicker(t.Context(), pair, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	exp := &ticker.Price{
		Pair:         pair,
		Ask:          11910855,
		AskSize:      0.01,
		Bid:          11907656,
		BidSize:      0.1032,
		Last:         11905672,
		BaseVolume:   196.60087808,
		ExchangeName: t.Name(),
		AssetType:    asset.Spot,
		LastUpdated:  time.Date(2026, time.September, 17, 5, 53, 16, 770000000, time.UTC),
	}
	assert.Equal(t, exp, got, "UpdateTicker should record the product's own volume, sizes and the response timestamp")
}

func TestGetExecutionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetExecutionHistory(t.Context(), "BTC_JPY")
	if err != nil {
		t.Error("Bitflyer - GetExecutionHistory() error:", err)
	}
}

func TestGetExchangeStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeStatus(t.Context())
	if err != nil {
		t.Error("Bitflyer - GetExchangeStatus() error:", err)
	}
}

func TestCheckFXString(t *testing.T) {
	t.Parallel()
	p, err := currency.NewPairDelimiter("FXBTC_JPY", "_")
	if err != nil {
		t.Fatal(err)
	}
	p = e.CheckFXString(p)
	if p.Base.String() != "FX_BTC" {
		t.Error("Bitflyer - CheckFXString() error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice:       1,
		FiatCurrency:        currency.JPY,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	if sharedtestvalues.AreAPICredentialsSet(e) {
		// CryptocurrencyTradeFee Basic
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := e.GetFee(feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.JPY
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.JPY
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawFiatText + " & " + exchange.WithdrawCryptoViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received '%v'", common.ErrNotYetImplemented, err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.LTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	_, err := e.SubmitOrder(t.Context(), orderSubmission)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)

	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	_, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not Yet Implemented', received %v", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}

	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}

	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_JPY")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_JPY")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Fatal(err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	err := e.CurrencyPairs.SetAssetEnabled(asset.Futures, false)
	require.NoError(t, err, "SetAssetEnabled must not error")
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err, "GetCurrencyTradeURL must not error")
		assert.NotEmpty(t, resp, "GetCurrencyTradeURL should return an url")
	}
}
