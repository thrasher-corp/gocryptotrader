package orderbook

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for orderbook package
const (
	errExchangeNameUnset = "orderbook exchange name not set"
	errPairNotSet        = "orderbook currency pair not set"
	errAssetTypeNotSet   = "orderbook asset type not set"
	errNoOrderbook       = "orderbook bids and asks are empty"
)

// Vars for the orderbook package
var (
	service *Service
)

func init() {
	service = new(Service)
	service.mux = dispatch.GetNewMux()
	service.Books = make(map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book)
	service.Exchange = make(map[string]uuid.UUID)
}

// Book defines an orderbook with its links to different dispatch outputs
type Book struct {
	b     *Base
	Main  uuid.UUID
	Assoc []uuid.UUID
}

// Service holds orderbook information for each individual exchange
type Service struct {
	Books    map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Book
	Exchange map[string]uuid.UUID
	mux      *dispatch.Mux
	sync.RWMutex
}

// Item stores the amount and price values
type Item struct {
	Amount float64
	Price  float64
	ID     int64

	// Contract variables
	LiquidationOrders int64
	OrderCount        int64
}

// Base holds the fields for the orderbook base
type Base struct {
	Pair         currency.Pair `json:"pair"`
	Bids         []Item        `json:"bids"`
	Asks         []Item        `json:"asks"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	LastUpdateID int64         `json:"lastUpdateId"`
	AssetType    asset.Item    `json:"assetType"`
	ExchangeName string        `json:"exchangeName"`
}

type byOBPrice []Item

func (a byOBPrice) Len() int           { return len(a) }
func (a byOBPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byOBPrice) Less(i, j int) bool { return a[i].Price < a[j].Price }
