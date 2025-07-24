package bithumb

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// WsResponse is a generalised response data structure which will defer
// unmarshalling of different contents.
type WsResponse struct {
	Status          string          `json:"status"`
	ResponseMessage string          `json:"resmsg"`
	Type            string          `json:"type"`
	Content         json.RawMessage `json:"content"`
}

// WsTicker defines a websocket ticker
type WsTicker struct {
	Symbol             currency.Pair `json:"symbol"`
	TickType           string        `json:"tickType"`
	Date               string        `json:"date"`
	Time               string        `json:"time"`
	OpenPrice          float64       `json:"openPrice,string"`
	ClosePrice         float64       `json:"closePrice,string"`
	LowPrice           float64       `json:"lowPrice,string"`
	HighPrice          float64       `json:"highPrice,string"`
	Value              float64       `json:"value,string"`
	Volume             float64       `json:"volume,string"`
	SellVolume         float64       `json:"sellVolume,string"`
	BuyVolume          float64       `json:"buyVolume,string"`
	PreviousClosePrice float64       `json:"prevClosePrice,string"`
	ChangeRate         float64       `json:"chgRate,string"`
	ChangeAmount       float64       `json:"chgAmt,string"`
	VolumePower        float64       `json:"volumePower,string"`
}

// WsOrderbooks defines an amalgamated bid ask orderbook level list
type WsOrderbooks struct {
	List     []WsOrderbook `json:"list"`
	DateTime types.Time    `json:"datetime"`
}

// WsOrderbook defines a singular orderbook level
type WsOrderbook struct {
	Symbol    currency.Pair `json:"symbol"`
	OrderSide string        `json:"orderType"`
	Price     float64       `json:"price,string"`
	Quantity  float64       `json:"quantity,string"`
	Total     int32         `json:"total,string"`
}

// WsTransactions defines a transaction list
type WsTransactions struct {
	List []WsTransaction `json:"list"`
}

// WsTransaction defines a trade that has executed via their matching engine
type WsTransaction struct {
	Symbol           currency.Pair `json:"symbol"`
	BuySell          int32         `json:"buySellGb,string"` // 1: Sell 2: Buy
	ContractPrice    float64       `json:"contPrice,string"`
	ContractQuantity float64       `json:"contQty,string"`
	ContractAmount   float64       `json:"contAmt,string"`
	ContractTime     string        `json:"contDtm"` // 2020-01-29 12:24:18.830039
	UpOrDown         string        `json:"updn"`
}

// WsSubscribe is used to subscribe to the ws channel.
type WsSubscribe struct {
	Type      string          `json:"type"`
	Symbols   []currency.Pair `json:"symbols"`
	TickTypes []string        `json:"tickTypes,omitempty"`
}

// orderbookManager defines a way of managing and maintaining synchronisation
// across connections and assets.
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update
	sync.Mutex

	jobs chan job
}

type update struct {
	buffer            chan *WsOrderbooks
	fetchingBook      bool
	initialSync       bool
	needsFetchingBook bool
	lastUpdated       time.Time
}

// job defines a synchronisation job that tells a go routine to fetch an
// orderbook via the REST protocol
type job struct {
	Pair currency.Pair
}
