package okx

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestEstimatedCompleteTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var estCompleteTime EstimatedCompleteTime
	require.NoError(t, json.Unmarshal([]byte(`"01/09/2023, 8:10:48 PM"`), &estCompleteTime), "the documented wall-clock form must decode")
	expected := time.Date(2023, time.January, 9, 20, 10, 48, 0, time.FixedZone("UTC+8", 8*60*60))
	assert.True(t, estCompleteTime.Time().Equal(expected), "the documented UTC+8 wall-clock time should decode")
	for _, form := range []string{`null`, `""`} {
		var empty EstimatedCompleteTime
		require.NoErrorf(t, json.Unmarshal([]byte(form), &empty), "the %s placeholder must decode", form)
		assert.True(t, empty.Time().IsZero(), "the placeholder should stay the zero time")
	}
	var invalid EstimatedCompleteTime
	err := json.Unmarshal([]byte(`"not-a-time"`), &invalid)
	require.ErrorIs(t, err, types.ErrInvalidTimestampFormat, "an unparsable timestamp must wrap the shared sentinel")
	err = json.Unmarshal([]byte(`1673266248000`), &invalid)
	require.ErrorIs(t, err, types.ErrInvalidTimestampFormat, "a non-string estCompleteTime must wrap the shared sentinel")
	var escaped EstimatedCompleteTime
	require.NoError(t, json.Unmarshal([]byte(`"01\/09\/2023, 8:10:48 PM"`), &escaped), "a JSON-escaped wall-clock string must decode")
	assert.True(t, escaped.Time().Equal(expected), "the escaped form should decode to the same time")
	encoded, err := json.Marshal(estCompleteTime)
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, `"2023-01-09T20:10:48+08:00"`, string(encoded), "the estimate should serialise as an RFC 3339 timestamp")
	assert.Equal(t, "2023-01-09 20:10:48 +0800 UTC+8", estCompleteTime.String(), "the estimate should print as its time")
}

const instrumentJSON = `{"alias":"","baseCcy":"","category":"1","ctMult":"1","ctType":"linear","ctVal":"0.0001","ctValCcy":"BTC","expTime":"","instFamily":"BTC-USDC","instId":"BTC-USDC-SWAP","instType":"SWAP","lever":"125","listTime":"1666076190000","lotSz":"1","maxIcebergSz":"100000000.0000000000000000","maxLmtSz":"100000000","maxMktSz":"85000","maxStopSz":"85000","maxTriggerSz":"100000000.0000000000000000","maxTwapSz":"","minSz":"1","optType":"","quoteCcy":"","settleCcy":"USDC","state":"live","stk":"","tickSz":"0.1","uly":"BTC-USD"}`

func TestInstrument(t *testing.T) {
	t.Parallel()
	var i InstrumentResponse
	err := json.Unmarshal([]byte(instrumentJSON), &i)
	require.NoError(t, err)
	require.Empty(t, i.Alias, "expected empty alias")
	assert.Empty(t, i.BaseCurrency, "expected empty base currency")
	assert.Equal(t, "1", i.Category, "expected 1 category")
	assert.Equal(t, 1, int(i.ContractMultiplier.Int64()), "expected 1 contract multiplier")
	assert.Equal(t, "linear", i.ContractType, "expected linear contract type")
	assert.Equal(t, 0.0001, i.ContractValue.Float64(), "expected 0.0001 contract value")
	assert.Equal(t, currency.BTC.String(), i.ContractValueCurrency.String(), "expected BTC contract value currency")
	assert.True(t, i.ExpiryTime.Time().IsZero(), "expected empty expiry time")
	assert.Equal(t, "BTC-USDC", i.InstrumentFamily, "expected BTC-USDC instrument family")
	assert.Equal(t, "BTC-USDC-SWAP", i.InstrumentID.String(), "expected BTC-USDC-SWAP instrument ID")

	swap := GetInstrumentTypeFromAssetItem(asset.PerpetualSwap)
	assert.Equal(t, swap, i.InstrumentType, "expected SWAP instrument type")
	assert.Equal(t, 125, int(i.MaxLeverage), "expected 125 leverage")
	assert.Equal(t, int64(1666076190000), i.ListTime.Time().UnixMilli(), "expected 1666076190000 listing time")
	assert.Equal(t, 1, int(i.LotSize))
	assert.Equal(t, 100000000.0000000000000000, i.MaxIcebergSize.Float64())
	assert.Equal(t, 100000000, int(i.MaxLimitSize))
	assert.Equal(t, 85000, int(i.MaxMarketSize))
	assert.Equal(t, 85000, int(i.MaxStopSize))
	assert.Equal(t, 100000000.0000000000000000, i.MaxTriggerSize.Float64())
	assert.Equal(t, 0, int(i.MaxTWAPSize), "expected empty max TWAP size")
	assert.Equal(t, 1, int(i.MinimumOrderSize))
	assert.Empty(t, i.OptionType, "expected empty option type")
	assert.Empty(t, i.QuoteCurrency, "expected empty quote currency")
	assert.Equal(t, currency.USDC.String(), i.SettlementCurrency.String(), "expected USDC settlement currency")
	assert.Equal(t, "live", i.State)
	assert.Empty(t, i.StrikePrice, "expected empty strike price")
	assert.Equal(t, 0.1, i.TickSize.Float64())
	assert.Equal(t, "BTC-USD", i.Underlying, "expected BTC-USD underlying")
}

func TestOptionalNumberUnmarshalJSON(t *testing.T) {
	t.Parallel()
	zero, negative, positive := types.Number(0), types.Number(-0.001), types.Number(0.002)
	for _, tc := range []struct {
		name, payload string
		want          OptionalNumber
		invalid       bool
	}{
		{name: "zero string", payload: `"0"`, want: OptionalNumber{Value: &zero}},
		{name: "zero number", payload: `0`, want: OptionalNumber{Value: &zero}},
		{name: "negative string", payload: `"-0.001"`, want: OptionalNumber{Value: &negative}},
		{name: "positive number", payload: `0.002`, want: OptionalNumber{Value: &positive}},
		{name: "empty string", payload: `""`},
		{name: "null", payload: `null`},
		{name: "padded empty string", payload: `  ""  `},
		{name: "padded null", payload: `  null  `},
		{name: "padded zero", payload: `  "0"  `, want: OptionalNumber{Value: &zero}},
		{name: "invalid number", payload: `"invalid"`, invalid: true},
		{name: "boolean", payload: `true`, invalid: true},
		{name: "object", payload: `{}`, invalid: true},
		{name: "array", payload: `[]`, invalid: true},
		{name: "empty input", payload: ``, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			previousValue := types.Number(1)
			previous := OptionalNumber{Value: &previousValue}
			got := previous
			err := got.UnmarshalJSON([]byte(tc.payload))
			if tc.invalid {
				require.Error(t, err, "invalid numeric field must fail")
				assert.Equal(t, previous, got, "invalid input should preserve the receiver")
				return
			}
			require.NoError(t, err, "valid numeric field must decode")
			assert.Equal(t, tc.want, got, "numeric value and presence should match the API field")
		})
	}
}

func TestTradeFeeRateResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("reused response", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRateResponse
		require.NoError(t, json.Unmarshal([]byte(`{"level":"1","maker":"0","feeGroup":[{"groupId":"4","maker":"0"}]}`), &row), "initial response must decode")
		require.NotEqual(t, TradeFeeRateResponse{}, row, "initial response must populate fields")
		expected := row
		expected.FeeRateMaker = nil
		require.NoError(t, json.Unmarshal([]byte(`{}`), &row), "reused response must decode")
		assert.Equal(t, expected, row, "ordinary fields should be retained while omitted presence-aware rates should clear")
	})

	for _, tc := range []struct {
		name, payload string
		present       bool
		invalid       bool
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0","makerU":"0","takerU":"0","makerUSDC":"0","takerUSDC":"0"}`, present: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0,"makerU":0,"takerU":0,"makerUSDC":0,"takerUSDC":0}`, present: true},
		{name: "present charges", payload: `{"maker":"-0.001","taker":"-0.002","makerU":"-0.001","takerU":"-0.002","makerUSDC":"-0.001","takerUSDC":"-0.002"}`, present: true},
		{name: "empty", payload: `{"maker":"","taker":"","makerU":"","takerU":"","makerUSDC":"","takerUSDC":""}`},
		{name: "null", payload: `{"maker":null,"taker":null,"makerU":null,"takerU":null,"makerUSDC":null,"takerUSDC":null}`},
		{name: "missing", payload: `{}`},
		{name: "invalid", payload: `{"maker":"invalid"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			previous := types.Number(1)
			row := TradeFeeRateResponse{FeeRateMaker: &previous, FeeRateTaker: &previous, FeeRateMakerUSDT: &previous, FeeRateTakerUSDT: &previous, FeeRateMakerUSDC: &previous, FeeRateTakerUSDC: &previous}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fees must fail decoding")
				return
			}
			require.NoError(t, err, "valid fees must decode")
			assert.Equal(t, tc.present, row.FeeRateMaker != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateTaker != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateMakerUSDT != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateTakerUSDT != nil, "presence should match the response")
			assert.Equal(t, tc.present, row.FeeRateMakerUSDC != nil, "USDC maker presence should match")
			assert.Equal(t, tc.present, row.FeeRateTakerUSDC != nil, "USDC taker presence should match")
		})
	}
	t.Run("independent legacy fields", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRateResponse
		require.NoError(t, json.Unmarshal([]byte(`{"maker":"-0.001","taker":null,"makerU":"","takerU":"-0.002","makerUSDC":"0.003","takerUSDC":"0"}`), &row), "mixed legacy fields must decode")
		maker, takerUSDT, makerUSDC, takerUSDC := types.Number(-0.001), types.Number(-0.002), types.Number(0.003), types.Number(0)
		assert.Equal(t, TradeFeeRateResponse{
			FeeRateMaker: &maker, FeeRateTakerUSDT: &takerUSDT, FeeRateMakerUSDC: &makerUSDC, FeeRateTakerUSDC: &takerUSDC,
		}, row, "each legacy rate and presence flag should follow its own field")
	})
	t.Run("modern fee group", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeRateResponse
		require.NoError(t, json.Unmarshal([]byte(`{"feeGroup":[{"groupId":"4","maker":"0","taker":"-0.0005"}]}`), &row), "fee group must decode")
		require.Len(t, row.FeeGroup, 1, "fee group must decode")
		maker, taker := types.Number(0), types.Number(-0.0005)
		assert.Equal(t, TradeFeeGroup{GroupID: "4", MakerRate: &maker, TakerRate: &taker}, row.FeeGroup[0], "fee group should preserve rates and presence")
		var instrument InstrumentResponse
		require.NoError(t, json.Unmarshal([]byte(`{"instId":"BTC-USDT-SWAP","groupId":"4"}`), &instrument), "instrument must decode")
		assert.Equal(t, "4", instrument.TradingFeeGroupID, "instrument should retain its fee group")
	})
}

func TestTradeFeeGroupUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("reused response", func(t *testing.T) {
		t.Parallel()
		var row TradeFeeGroup
		require.NoError(t, json.Unmarshal([]byte(`{"groupId":"4","maker":"0"}`), &row), "initial response must decode")
		require.NotEqual(t, TradeFeeGroup{}, row, "initial response must populate fields")
		expected := row
		expected.MakerRate = nil
		require.NoError(t, json.Unmarshal([]byte(`{}`), &row), "reused response must decode")
		assert.Equal(t, expected, row, "ordinary fields should be retained while omitted presence-aware rates should clear")
	})

	for _, tc := range []struct {
		name, payload                       string
		makerPresent, takerPresent, invalid bool
		maker, taker                        float64
	}{
		{name: "explicit zero strings", payload: `{"maker":"0","taker":"0"}`, makerPresent: true, takerPresent: true},
		{name: "explicit zero numbers", payload: `{"maker":0,"taker":0}`, makerPresent: true, takerPresent: true},
		{name: "charges and rebates", payload: `{"maker":"0.001","taker":"-0.002"}`, makerPresent: true, takerPresent: true, maker: 0.001, taker: -0.002},
		{name: "maker only", payload: `{"maker":"0","taker":""}`, makerPresent: true},
		{name: "taker only", payload: `{"maker":null,"taker":"-0.002"}`, takerPresent: true, taker: -0.002},
		{name: "absent", payload: `{}`},
		{name: "null", payload: `{"maker":null,"taker":null}`},
		{name: "empty", payload: `{"maker":"","taker":""}`},
		{name: "invalid", payload: `{"maker":"invalid","taker":"0"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			previous := types.Number(1)
			row := TradeFeeGroup{MakerRate: &previous, TakerRate: &previous}
			err := json.Unmarshal([]byte(tc.payload), &row)
			if tc.invalid {
				require.Error(t, err, "invalid fee values must fail decoding")
				return
			}
			require.NoError(t, err, "valid fee responses must decode")
			assert.Equal(t, tc.makerPresent, row.MakerRate != nil, "maker presence should match the response")
			if tc.makerPresent {
				require.NotNil(t, row.MakerRate, "maker rate must be available")
				assert.Equal(t, tc.maker, row.MakerRate.Float64(), "maker rate should match the response")
			}
			assert.Equal(t, tc.takerPresent, row.TakerRate != nil, "taker presence should match the response")
			if tc.takerPresent {
				require.NotNil(t, row.TakerRate, "taker rate must be available")
				assert.Equal(t, tc.taker, row.TakerRate.Float64(), "taker rate should match the response")
			}
		})
	}
}

func TestFundingRateResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("reused response", func(t *testing.T) {
		t.Parallel()
		var row FundingRateResponse
		require.NoError(t, json.Unmarshal([]byte(`{"instId":"BTC-USDT-SWAP","fundingRate":"0.001","realizedRate":"0"}`), &row), "initial response must decode")
		require.NotEqual(t, FundingRateResponse{}, row, "initial response must populate fields")
		expected := row
		expected.RealisedRate = nil
		require.NoError(t, json.Unmarshal([]byte(`{}`), &row), "reused response must decode")
		assert.Equal(t, expected, row, "ordinary fields should be retained while omitted presence-aware rates should clear")
	})

	for _, tc := range []struct {
		name, payload string
		present       bool
		want          float64
		invalid       bool
	}{
		{name: "zero string", payload: `{"realizedRate":"0"}`, present: true},
		{name: "zero number", payload: `{"realizedRate":0}`, present: true},
		{name: "positive", payload: `{"realizedRate":"0.001"}`, present: true, want: 0.001},
		{name: "negative", payload: `{"realizedRate":"-0.001"}`, present: true, want: -0.001},
		{name: "missing", payload: `{}`},
		{name: "empty", payload: `{"realizedRate":""}`},
		{name: "null", payload: `{"realizedRate":null}`},
		{name: "invalid", payload: `{"realizedRate":"unknown"}`, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			previous := types.Number(1)
			r := FundingRateResponse{RealisedRate: &previous}
			err := json.Unmarshal([]byte(tc.payload), &r)
			if tc.invalid {
				require.Error(t, err, "invalid rate must fail decoding")
				return
			}
			require.NoError(t, err, "valid rate must decode")
			assert.Equal(t, tc.present, r.RealisedRate != nil, "presence should match settled evidence")
			if tc.present {
				require.NotNil(t, r.RealisedRate, "settled rate must be available")
				assert.Equal(t, tc.want, r.RealisedRate.Float64(), "rate should preserve the supplied value")
			}
		})
	}
}

func TestOrderActionResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  *OrderActionResponse
	}{
		{
			name: "placed",
			data: `{"clOrdId":"oktswap6","ordId":"12345689","reqId":"b12344","tag":"tag6","ts":"1695190491421","sCode":"0","sMsg":"","subCode":""}`,
			exp:  &OrderActionResponse{OrderID: "12345689", RequestID: "b12344", ClientOrderID: "oktswap6", Tag: "tag6", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		},
		{
			name: "rejected",
			data: `{"clOrdId":"oktswap7","ordId":"","tag":"","ts":"1695190491421","sCode":"51008","sMsg":"Order failed. Insufficient USDT balance in account","subCode":"1000"}`,
			exp:  &OrderActionResponse{ClientOrderID: "oktswap7", StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account", SubCode: "1000", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		},
		{name: "no status code", data: `{"ordId":"1001"}`, exp: &OrderActionResponse{OrderID: "1001", statusCodeMissing: true}},
		{name: "null status code", data: `{"ordId":"1001","sCode":null}`, exp: &OrderActionResponse{OrderID: "1001", statusCodeMissing: true}},
		{name: "empty row", data: `{}`, exp: &OrderActionResponse{statusCodeMissing: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := new(OrderActionResponse)
			require.NoError(t, json.Unmarshal([]byte(tc.data), got), "Unmarshal must decode the row")
			assert.Equal(t, tc.exp, got, "Unmarshal should decode every field and whether the row states its status code")
		})
	}
	t.Run("rows of a batch", func(t *testing.T) {
		t.Parallel()
		var rows []*OrderActionResponse
		require.NoError(t, json.Unmarshal([]byte(`[{"ordId":"1","sCode":"0"},null,{"ordId":"3"}]`), &rows), "Unmarshal must decode the rows")
		assert.Equal(t, []*OrderActionResponse{{OrderID: "1"}, nil, {OrderID: "3", statusCodeMissing: true}}, rows, "Unmarshal should decode each row's status code on its own")
	})
	t.Run("status code which is no integer", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, json.Unmarshal([]byte(`{"ordId":"1001","sCode":"5XXXX"}`), new(OrderActionResponse)), strconv.ErrSyntax, "Unmarshal should refuse a status code which is no integer")
	})
}

func TestOrderActionResponseError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&OrderActionResponse{OrderID: "1001"}).Error(), "Error should report nothing for a row stating success")
	assert.EqualError(t, (&OrderActionResponse{StatusCode: 51008, StatusMessage: "insufficient balance"}).Error(), "status code: `51008` status message: \"insufficient balance\"", "Error should report a failed row's status")
	assert.ErrorIs(t, (&OrderActionResponse{OrderID: "1001", statusCodeMissing: true}).Error(), errMissingStatusCode, "Error should report a row stating no status")
}

func TestAlgoOrderActionResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  *AlgoOrderActionResponse
	}{
		{
			name: "placed",
			data: `{"algoClOrdId":"order1234","algoId":"1836487817828872192","clOrdId":"client1234","sCode":"0","sMsg":"","tag":"tag1"}`,
			exp:  &AlgoOrderActionResponse{AlgoID: "1836487817828872192", ClientOrderID: "client1234", AlgoClientOrderID: "order1234", Tag: "tag1"},
		},
		{
			name: "rejected",
			data: `{"algoClOrdId":"order1234","algoId":"","sCode":"51000","sMsg":"Parameter error"}`,
			exp:  &AlgoOrderActionResponse{AlgoClientOrderID: "order1234", StatusCode: 51000, StatusMessage: "Parameter error"},
		},
		{name: "no status code", data: `{"algoId":"1001"}`, exp: &AlgoOrderActionResponse{AlgoID: "1001", statusCodeMissing: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := new(AlgoOrderActionResponse)
			require.NoError(t, json.Unmarshal([]byte(tc.data), got), "Unmarshal must decode the row")
			assert.Equal(t, tc.exp, got, "Unmarshal should decode every field and whether the row states its status code")
		})
	}
	t.Run("status code which is no integer", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, json.Unmarshal([]byte(`{"algoId":"1001","sCode":"5XXXX"}`), new(AlgoOrderActionResponse)), strconv.ErrSyntax, "Unmarshal should refuse a status code which is no integer")
	})
}

func TestSpreadOrderResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  *SpreadOrderResponse
	}{
		{
			name: "placed",
			data: `{"clOrdId":"b15","ordId":"312269865356374016","tag":"tag15","reqId":"b12344","sCode":"0","sMsg":""}`,
			exp:  &SpreadOrderResponse{OrderID: "312269865356374016", ClientOrderID: "b15", Tag: "tag15", RequestID: "b12344"},
		},
		{
			name: "rejected",
			data: `{"clOrdId":"b15","ordId":"","sCode":"51008","sMsg":"Order failed. Insufficient balance"}`,
			exp:  &SpreadOrderResponse{ClientOrderID: "b15", StatusCode: 51008, StatusMessage: "Order failed. Insufficient balance"},
		},
		{name: "no status code", data: `{"ordId":"1001"}`, exp: &SpreadOrderResponse{OrderID: "1001", statusCodeMissing: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := new(SpreadOrderResponse)
			require.NoError(t, json.Unmarshal([]byte(tc.data), got), "Unmarshal must decode the row")
			assert.Equal(t, tc.exp, got, "Unmarshal should decode every field and whether the row states its status code")
		})
	}
	t.Run("status code which is no integer", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, json.Unmarshal([]byte(`{"ordId":"1001","sCode":"5XXXX"}`), new(SpreadOrderResponse)), strconv.ErrSyntax, "Unmarshal should refuse a status code which is no integer")
	})
}

func TestSpreadOrderResponseError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&SpreadOrderResponse{OrderID: "1001"}).Error(), "Error should report nothing for a row stating success")
	assert.EqualError(t, (&SpreadOrderResponse{StatusCode: 51008, StatusMessage: "insufficient balance"}).Error(), "status code: `51008` status message: \"insufficient balance\"", "Error should report a failed row's status")
	assert.ErrorIs(t, (&SpreadOrderResponse{OrderID: "1001", statusCodeMissing: true}).Error(), errMissingStatusCode, "Error should report a row stating no status")
}

func TestDecodeStatusCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		sCode      *string
		expCode    uint64
		expMissing bool
		expErr     error
	}{
		{name: "absent", expMissing: true},
		{name: "success", sCode: new("0")},
		{name: "failure", sCode: new("51008"), expCode: 51008},
		{name: "no integer", sCode: new("5XXXX"), expErr: strconv.ErrSyntax},
		{name: "empty", sCode: new(""), expErr: strconv.ErrSyntax},
		{name: "negative", sCode: new("-1"), expErr: strconv.ErrSyntax},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, missing, err := decodeStatusCode(tc.sCode)
			assert.ErrorIs(t, err, tc.expErr, "decodeStatusCode should return the expected error")
			assert.Equal(t, tc.expCode, code, "decodeStatusCode should return the status code")
			assert.Equal(t, tc.expMissing, missing, "decodeStatusCode should report whether the row carried a status code")
		})
	}
}

func TestReplyRowStatusError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, replyRowStatusError(0, "", false), "replyRowStatusError should report nothing for success")
	assert.EqualError(t, replyRowStatusError(51008, "insufficient balance", false), "status code: `51008` status message: \"insufficient balance\"", "replyRowStatusError should report a failed status")
	assert.ErrorIs(t, replyRowStatusError(0, "", true), errMissingStatusCode, "replyRowStatusError should report a row without a status code")
	assert.ErrorIs(t, replyRowStatusError(0, "", true), common.ErrInvalidResponse, "a row without a status code should be an invalid response")
}
