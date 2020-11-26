package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctkline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// LoadData is a basic csv reader which converts the found CSV file into a kline item
func LoadData(filepath, dataType, exchangeName string, interval time.Duration, fPair currency.Pair, a asset.Item) (*gctkline.DataFromKline, error) {
	resp := &gctkline.DataFromKline{}
	csvFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = csvFile.Close()
		if err != nil {
			log.Errorln(log.BackTester, err)
		}
	}()

	csvData := csv.NewReader(csvFile)

	switch dataType {
	case common.CandleStr:
		candles := kline.Item{
			Exchange: exchangeName,
			Pair:     fPair,
			Asset:    a,
			Interval: kline.Interval(interval),
		}

		for {
			row, errCSV := csvData.Read()
			if errCSV != nil {
				if errCSV == io.EOF {
					break
				}
				return nil, errCSV
			}

			candle := kline.Candle{}
			v, errParse := strconv.ParseInt(row[0], 10, 32)
			if errParse != nil {
				return nil, errParse
			}
			candle.Time = time.Unix(v, 0).UTC()
			if candle.Time.IsZero() {
				err = fmt.Errorf("invalid timestamp received on row %v", row)
				break
			}

			candle.Volume, err = strconv.ParseFloat(row[1], 64)
			if err != nil {
				break
			}

			candle.Open, err = strconv.ParseFloat(row[2], 64)
			if err != nil {
				break
			}

			candle.High, err = strconv.ParseFloat(row[3], 64)
			if err != nil {
				break
			}

			candle.Low, err = strconv.ParseFloat(row[4], 64)
			if err != nil {
				break
			}

			candle.Close, err = strconv.ParseFloat(row[5], 64)
			if err != nil {
				break
			}

			candles.Candles = append(candles.Candles, candle)
		}

		resp.Item = candles
	case common.TradeStr:
		var trades []trade.Data
		for {
			row, errCSV := csvData.Read()
			if errCSV != nil {
				if errCSV == io.EOF {
					break
				}
				return nil, errCSV
			}

			t := trade.Data{}
			v, errParse := strconv.ParseInt(row[0], 10, 32)
			if errParse != nil {
				return nil, errParse
			}
			t.Timestamp = time.Unix(v, 0).UTC()
			if t.Timestamp.IsZero() {
				err = fmt.Errorf("invalid timestamp received on row %v", row)
				break
			}

			t.Price, err = strconv.ParseFloat(row[1], 64)
			if err != nil {
				break
			}

			t.Amount, err = strconv.ParseFloat(row[2], 64)
			if err != nil {
				break
			}

			t.Side, err = order.StringToOrderSide(row[3])
			if err != nil {
				return nil, err
			}

			trades = append(trades, t)
		}
		resp.Item, err = trade.ConvertTradesToCandles(kline.Interval(interval), trades...)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognised csv datatype received: '%v'", dataType)
	}
	resp.Item.Exchange = strings.ToLower(resp.Item.Exchange)

	return resp, nil
}
