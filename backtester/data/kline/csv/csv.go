package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var errNoUSDData = errors.New("could not retrieve USD CSV candle data")

// LoadData is a basic csv reader which converts the found CSV file into a kline item
func LoadData(dataType int64, filepath, exchangeName string, interval time.Duration, fPair currency.Pair, a asset.Item, isUSDTrackingPair bool) (*kline.DataFromKline, error) {
	resp := kline.NewDataFromKline()
	csvFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = csvFile.Close()
		if err != nil {
			log.Errorln(common.Data, err)
		}
	}()

	csvData := csv.NewReader(csvFile)

	switch dataType {
	case common.DataCandle:
		candles := gctkline.Item{
			Exchange: exchangeName,
			Pair:     fPair,
			Asset:    a,
			Interval: gctkline.Interval(interval),
		}

		for {
			row, errCSV := csvData.Read()
			if errCSV != nil {
				if errCSV == io.EOF {
					break
				}
				return nil, fmt.Errorf("could not read csv data for %v %v %v, %v", exchangeName, a, fPair, errCSV)
			}

			candle := gctkline.Candle{}
			v, errParse := strconv.ParseInt(row[0], 10, 32)
			if errParse != nil {
				return nil, errParse
			}
			candle.Time = time.Unix(v, 0).UTC()
			if candle.Time.IsZero() {
				err = fmt.Errorf("invalid timestamp received on row %v %v", row[0], err)
				break
			}

			candle.Volume, err = strconv.ParseFloat(row[1], 64)
			if err != nil {
				err = fmt.Errorf("could not process candle volume %v %v", row[1], err)
				break
			}

			candle.Open, err = strconv.ParseFloat(row[2], 64)
			if err != nil {
				err = fmt.Errorf("could not process candle volume %v %v", row[2], err)
				break
			}

			candle.High, err = strconv.ParseFloat(row[3], 64)
			if err != nil {
				err = fmt.Errorf("could not process candle high %v %v", row[3], err)
				break
			}

			candle.Low, err = strconv.ParseFloat(row[4], 64)
			if err != nil {
				err = fmt.Errorf("could not process candle low %v %v", row[4], err)
				break
			}

			candle.Close, err = strconv.ParseFloat(row[5], 64)
			if err != nil {
				err = fmt.Errorf("could not process candle close %v %v", row[5], err)
				break
			}

			candles.Candles = append(candles.Candles, candle)
		}
		if err != nil {
			return nil, fmt.Errorf("could not read csv candle data for %v %v %v, %v", exchangeName, a, fPair, err)
		}
		resp.Item = &candles
	case common.DataTrade:
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
				err = fmt.Errorf("could not process trade price %v, %v", row[1], err)
				break
			}

			t.Amount, err = strconv.ParseFloat(row[2], 64)
			if err != nil {
				err = fmt.Errorf("could not process trade amount %v, %v", row[2], err)
				break
			}

			t.Side, err = order.StringToOrderSide(row[3])
			if err != nil {
				err = fmt.Errorf("could not process trade side %v, %v", row[3], err)
				break
			}

			trades = append(trades, t)
		}
		resp.Item, err = trade.ConvertTradesToCandles(gctkline.Interval(interval), trades...)
		if err != nil {
			return nil, fmt.Errorf("could not read csv trade data for %v %v %v, %v", exchangeName, a, fPair, err)
		}
	default:
		if isUSDTrackingPair {
			return nil, fmt.Errorf("%w for %v %v %v. Please add USD pair data to your CSV or set `disable-usd-tracking` to `true` in your config. %v", errNoUSDData, exchangeName, a, fPair, err)
		}
		return nil, fmt.Errorf("could not process csv data for %v %v %v, %w", exchangeName, a, fPair, common.ErrInvalidDataType)
	}
	resp.Item.Exchange = strings.ToLower(exchangeName)
	resp.Item.Pair = fPair
	resp.Item.Asset = a
	resp.Item.Interval = gctkline.Interval(interval)

	return resp, nil
}
