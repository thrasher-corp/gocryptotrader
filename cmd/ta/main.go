package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	gctta "github.com/thrasher-corp/gct-ta/indicators"
)

const (
	cryptoWatchAPIMarkets   string = "https://api.cryptowat.ch/markets/"
	cryptoWatchOHLCEndPoint string = "/ohlc"
	timeFormat                     = "2006-01-02 15:04:05"
)

var (
	exchange      string
	currencyPair  string
	start         string
	end           string
	period        string
	indicator     string
	indicatorArgs string
)

type candle struct {
	open  float64
	high  float64
	close float64
	low   float64
	vol   float64
}

type ohlcResponse struct {
	Err    string `json:"error,omitempty"`
	Result struct {
		Num60          [][]float64 `json:"60,omitempty"`
		Num180         [][]float64 `json:"180,omitempty"`
		Num300         [][]float64 `json:"300,omitempty"`
		Num900         [][]float64 `json:"900,omitempty"`
		Num1800        [][]float64 `json:"1800,omitempty"`
		Num3600        [][]float64 `json:"3600,omitempty"`
		Num7200        [][]float64 `json:"7200,omitempty"`
		Num14400       [][]float64 `json:"14400,omitempty"`
		Num21600       [][]float64 `json:"21600,omitempty"`
		Num43200       [][]float64 `json:"43200,omitempty"`
		Num86400       [][]float64 `json:"86400,omitempty"`
		Num259200      [][]float64 `json:"259200,omitempty"`
		Num604800      [][]float64 `json:"604800,omitempty"`
		Six04800Monday [][]float64 `json:"604800_Monday,omitempty"`
	} `json:"result"`
}

type output struct {
	Indicator string
	Start     string
	End       string
	Interval  int
	Data      [][]float64
}

func main() {
	flag.StringVar(&exchange, "exchange", "binance", "exchange <name>")
	flag.StringVar(&currencyPair, "currency", "btcusdt", "currency <pair>")
	flag.StringVar(&start, "start", time.Now().Add(-time.Hour*24).Format(timeFormat), "period <interval>")
	flag.StringVar(&end, "end", time.Now().Format(timeFormat), "period <interval>")
	flag.StringVar(&period, "period", "60", "period <interval>")
	flag.StringVar(&indicator, "indicator", "rsi", "indicator <type>")
	flag.StringVar(&indicatorArgs, "args", "14", "args 14")
	flag.Parse()

	startTime, err := time.Parse(timeFormat, start)
	if err != nil {
		log.Fatal(err)
	}
	endTime, err := time.Parse(timeFormat, end)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Exchange: %v\n", exchange)
	fmt.Printf("Currency: %v\n", currencyPair)
	fmt.Printf("Period: %v\n", period)
	fmt.Printf("Start: %v\n", start)
	fmt.Printf("End: %v\n", end)
	fmt.Printf("Indicator: %v args: %v\n\n", indicator, indicatorArgs)
	data := getCryptoWatchData(exchange, currencyPair, startTime, endTime, period)
	parsed := parseData(data, period)
	ret, err := indicatorParse(parsed, strings.ToLower(indicator), indicatorArgs)
	if err != nil {
		log.Fatal(err)
	}

	out, err := json.MarshalIndent(ret, " ", " ")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%s", out)
}

// nolint gocyclo alternatives are to use reflection this is code increase v performance cost
func indicatorParse(input []candle, indicator, args string) (output, error) {
	out := make([][]float64, 1)
	var interval int
	type ohlcStruct struct {
		open  []float64
		high  []float64
		low   []float64
		close []float64
		vol   []float64
	}

	var ohlcvData ohlcStruct
	ohlcvData.open = make([]float64, len(input))
	ohlcvData.high = make([]float64, len(input))
	ohlcvData.low = make([]float64, len(input))
	ohlcvData.close = make([]float64, len(input))
	ohlcvData.vol = make([]float64, len(input))

	for x := range input {
		ohlcvData.open[x] = input[x].open
		ohlcvData.high[x] = input[x].high
		ohlcvData.low[x] = input[x].low
		ohlcvData.close[x] = input[x].close
		ohlcvData.vol[x] = input[x].vol
	}

	switch indicator {
	case "ema":
		timeInput, err := strconv.Atoi(args)
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		out[0] = gctta.EMA(ohlcvData.close, timeInput)
	case "sma":
		timeInput, err := strconv.Atoi(args)
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		out[0] = gctta.SMA(ohlcvData.close, timeInput)
	case "rsi":
		timeInput, err := strconv.Atoi(args)
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		out[0] = gctta.RSI(ohlcvData.close, timeInput)
	case "atr":
		timeInput, err := strconv.Atoi(args)
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		out[0] = gctta.ATR(ohlcvData.high, ohlcvData.low, ohlcvData.close, timeInput)
	case "obv":
		out[0] = gctta.OBV(ohlcvData.close, ohlcvData.vol)
	case "mfi":
		timeInput, err := strconv.Atoi(args)
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		out[0] = gctta.MFI(ohlcvData.high, ohlcvData.low, ohlcvData.close, ohlcvData.vol, timeInput)
	case "macd":
		args := strings.Split(args, ",")
		if len(args) != 3 {
			return output{}, fmt.Errorf("MACD requires fast, slow, signal periods")
		}
		fast, err := strconv.Atoi(args[0])
		if err != nil {
			return output{}, err
		}
		slow, err := strconv.Atoi(args[1])
		if err != nil {
			return output{}, err
		}
		signal, err := strconv.Atoi(args[2])
		if err != nil {
			return output{}, err
		}
		out = make([][]float64, 3)
		out[0], out[1], out[2] = gctta.MACD(ohlcvData.close, fast, slow, signal)
	case "bbands":
		args := strings.Split(args, ",")
		if len(args) != 3 {
			return output{}, fmt.Errorf("bbands requires time, up & down params")
		}
		timeInput, err := strconv.Atoi(args[0])
		if err != nil {
			return output{}, err
		}
		interval = timeInput
		up, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return output{}, err
		}
		down, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return output{}, err
		}
		out = make([][]float64, 3)
		out[0], out[1], out[2] = gctta.BBANDS(ohlcvData.high, timeInput, up, down, gctta.Ema)
	}

	return output{
		Indicator: indicator,
		Start:     start,
		End:       end,
		Interval:  interval,
		Data:      out,
	}, nil
}

// nolint gocyclo alternatives are to use reflection this is code increase v performance cost
func parseData(data *ohlcResponse, period string) []candle {
	switch period {
	case "60":
		candles := make([]candle, len(data.Result.Num60))
		for x := range data.Result.Num60 {
			candles[x].open = data.Result.Num60[x][0]
			candles[x].high = data.Result.Num60[x][1]
			candles[x].low = data.Result.Num60[x][2]
			candles[x].close = data.Result.Num60[x][3]
			candles[x].vol = data.Result.Num60[x][4]
		}
		return candles
	case "180":
		candles := make([]candle, len(data.Result.Num180))
		for x := range data.Result.Num180 {
			candles[x].open = data.Result.Num180[x][0]
			candles[x].high = data.Result.Num180[x][1]
			candles[x].low = data.Result.Num180[x][2]
			candles[x].close = data.Result.Num180[x][3]
			candles[x].vol = data.Result.Num180[x][4]
		}
		return candles
	case "300":
		candles := make([]candle, len(data.Result.Num300))
		for x := range data.Result.Num300 {
			candles[x].open = data.Result.Num300[x][0]
			candles[x].high = data.Result.Num300[x][1]
			candles[x].low = data.Result.Num300[x][2]
			candles[x].close = data.Result.Num300[x][3]
			candles[x].vol = data.Result.Num300[x][4]
		}
		return candles
	case "1800":
		candles := make([]candle, len(data.Result.Num1800))
		for x := range data.Result.Num1800 {
			candles[x].open = data.Result.Num1800[x][0]
			candles[x].high = data.Result.Num1800[x][1]
			candles[x].low = data.Result.Num1800[x][2]
			candles[x].close = data.Result.Num1800[x][3]
			candles[x].vol = data.Result.Num1800[x][4]
		}
		return candles
	case "3600":
		candles := make([]candle, len(data.Result.Num3600))
		for x := range data.Result.Num3600 {
			candles[x].open = data.Result.Num3600[x][0]
			candles[x].high = data.Result.Num3600[x][1]
			candles[x].low = data.Result.Num3600[x][2]
			candles[x].close = data.Result.Num3600[x][3]
			candles[x].vol = data.Result.Num3600[x][4]
		}
		return candles
	case "7200":
		candles := make([]candle, len(data.Result.Num7200))
		for x := range data.Result.Num7200 {
			candles[x].open = data.Result.Num7200[x][0]
			candles[x].high = data.Result.Num7200[x][1]
			candles[x].low = data.Result.Num7200[x][2]
			candles[x].close = data.Result.Num7200[x][3]
			candles[x].vol = data.Result.Num7200[x][4]
		}
		return candles
	case "14400":
		candles := make([]candle, len(data.Result.Num14400))
		for x := range data.Result.Num14400 {
			candles[x].open = data.Result.Num14400[x][0]
			candles[x].high = data.Result.Num14400[x][1]
			candles[x].low = data.Result.Num14400[x][2]
			candles[x].close = data.Result.Num14400[x][3]
			candles[x].vol = data.Result.Num14400[x][4]
		}
		return candles
	case "21600":
		candles := make([]candle, len(data.Result.Num21600))
		for x := range data.Result.Num21600 {
			candles[x].open = data.Result.Num21600[x][0]
			candles[x].high = data.Result.Num21600[x][1]
			candles[x].low = data.Result.Num21600[x][2]
			candles[x].close = data.Result.Num21600[x][3]
			candles[x].vol = data.Result.Num21600[x][4]
		}
	case "43200":
		candles := make([]candle, len(data.Result.Num43200))
		for x := range data.Result.Num43200 {
			candles[x].open = data.Result.Num43200[x][0]
			candles[x].high = data.Result.Num43200[x][1]
			candles[x].low = data.Result.Num43200[x][2]
			candles[x].close = data.Result.Num43200[x][3]
			candles[x].vol = data.Result.Num43200[x][4]
		}
		return candles
	case "86400":
		candles := make([]candle, len(data.Result.Num86400))
		for x := range data.Result.Num86400 {
			candles[x].open = data.Result.Num86400[x][0]
			candles[x].high = data.Result.Num86400[x][1]
			candles[x].low = data.Result.Num86400[x][2]
			candles[x].close = data.Result.Num86400[x][3]
			candles[x].vol = data.Result.Num86400[x][4]
		}
		return candles
	}

	return nil
}

func getCryptoWatchData(exchange, currencyPair string, start, end time.Time, periods string) *ohlcResponse {
	cryptoWatchURL := cryptoWatchAPIMarkets + exchange + "/" + currencyPair + cryptoWatchOHLCEndPoint

	client := &http.Client{}
	req, err := http.NewRequest("GET", cryptoWatchURL, nil)
	if err != nil {
		log.Fatalln(err)
	}

	query := req.URL.Query()
	query.Add("before", strconv.FormatInt(end.Unix(), 10))
	query.Add("after", strconv.FormatInt(start.Unix(), 10))
	query.Add("periods", periods)

	req.URL.RawQuery = query.Encode()
	req.Header.Set("User-Agent", "gct-ta/0.1")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		log.Fatalln(err)
	}
	resp.Body.Close()
	var data = ohlcResponse{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalln(err)
	}
	if data.Err != "" {
		log.Fatalln(data.Err)
	}
	return &data
}
