package kline

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// LoadFromDatabase returns Item from database seeded data
func LoadFromDatabase(exchange string, pair currency.Pair, a asset.Item, interval Interval, start, end time.Time) (*Item, error) {
	retCandle, err := candle.Series(exchange,
		pair.Base.String(), pair.Quote.String(),
		int64(interval.Duration().Seconds()), a.String(), start, end)
	if err != nil {
		return nil, err
	}

	ret := Item{
		Exchange: exchange,
		Pair:     pair,
		Interval: interval,
		Asset:    a,
	}

	for x := range retCandle.Candles {
		if ret.SourceJobID == uuid.Nil && retCandle.Candles[x].SourceJobID != "" {
			ret.SourceJobID, err = uuid.FromString(retCandle.Candles[x].SourceJobID)
			if err != nil {
				return nil, err
			}
		}
		if ret.ValidationJobID == uuid.Nil && retCandle.Candles[x].ValidationJobID != "" {
			ret.ValidationJobID, err = uuid.FromString(retCandle.Candles[x].ValidationJobID)
			if err != nil {
				return nil, err
			}
		}
		ret.Candles = append(ret.Candles, Candle{
			Time:             retCandle.Candles[x].Timestamp,
			Open:             retCandle.Candles[x].Open,
			High:             retCandle.Candles[x].High,
			Low:              retCandle.Candles[x].Low,
			Close:            retCandle.Candles[x].Close,
			Volume:           retCandle.Candles[x].Volume,
			ValidationIssues: retCandle.Candles[x].ValidationIssues,
		})
	}
	return &ret, nil
}

// StoreInDatabase returns Item from database seeded data
func StoreInDatabase(in *Item, force bool) (uint64, error) {
	if in.Exchange == "" {
		return 0, errors.New("name cannot be blank")
	}
	if in.Pair.IsEmpty() {
		return 0, errors.New("currency pair cannot be empty")
	}
	if !in.Asset.IsValid() {
		return 0, errors.New("asset cannot be blank")
	}
	if len(in.Candles) < 1 {
		return 0, errors.New("candle data is empty")
	}

	exchangeUUID, err := exchange.UUIDByName(in.Exchange)
	if err != nil {
		return 0, err
	}

	databaseCandles := candle.Item{
		ExchangeID: exchangeUUID.String(),
		Base:       in.Pair.Base.Upper().String(),
		Quote:      in.Pair.Quote.Upper().String(),
		Interval:   int64(in.Interval.Duration().Seconds()),
		Asset:      in.Asset.String(),
	}

	for x := range in.Candles {
		can := candle.Candle{
			Timestamp: in.Candles[x].Time.Truncate(in.Interval.Duration()),
			Open:      in.Candles[x].Open,
			High:      in.Candles[x].High,
			Low:       in.Candles[x].Low,
			Close:     in.Candles[x].Close,
			Volume:    in.Candles[x].Volume,
		}
		if in.ValidationJobID != uuid.Nil {
			can.ValidationJobID = in.ValidationJobID.String()
			can.ValidationIssues = in.Candles[x].ValidationIssues
		}
		if in.SourceJobID != uuid.Nil {
			can.SourceJobID = in.SourceJobID.String()
		}

		databaseCandles.Candles = append(databaseCandles.Candles, can)
	}
	if force {
		_, err := candle.DeleteCandles(&databaseCandles)
		if err != nil {
			return 0, err
		}
	}
	return candle.Insert(&databaseCandles)
}

// LoadFromGCTScriptCSV loads kline data from a CSV file
func LoadFromGCTScriptCSV(file string) (out []Candle, errRet error) {
	csvFile, err := os.Open(file)
	if err != nil {
		return out, err
	}

	defer func() {
		err = csvFile.Close()
		if err != nil {
			log.Errorln(log.Global, err)
		}
	}()

	csvData := csv.NewReader(csvFile)

	for {
		row, errCSV := csvData.Read()
		if errCSV != nil {
			if errCSV == io.EOF {
				break
			}
			return out, errCSV
		}

		tempCandle := Candle{}
		v, errParse := strconv.ParseInt(row[0], 10, 32)
		if errParse != nil {
			err = errParse
			break
		}
		tempCandle.Time = time.Unix(v, 0).UTC()
		if tempCandle.Time.IsZero() {
			err = fmt.Errorf("invalid timestamp received on row %v", row)
			break
		}

		tempCandle.Volume, err = strconv.ParseFloat(row[1], 64)
		if err != nil {
			break
		}

		tempCandle.Open, err = strconv.ParseFloat(row[2], 64)
		if err != nil {
			break
		}

		tempCandle.High, err = strconv.ParseFloat(row[3], 64)
		if err != nil {
			break
		}

		tempCandle.Low, err = strconv.ParseFloat(row[4], 64)
		if err != nil {
			break
		}

		tempCandle.Close, err = strconv.ParseFloat(row[5], 64)
		if err != nil {
			break
		}
		out = append(out, tempCandle)
	}
	return out, err
}
