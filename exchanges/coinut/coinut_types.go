package coinut

type CoinutGenericResponse struct {
	Nonce     int64    `json:"nonce"`
	Reply     string   `json:"reply"`
	Status    []string `json:"status"`
	TransID   int64    `json:"trans_id"`
	Timestamp int64    `json:"timestamp"`
}

type CoinutInstrumentBase struct {
	Base          string `json:"base"`
	DecimalPlaces int    `json:"decimal_places"`
	InstID        int    `json:"inst_id"`
	Quote         string `json:"quote"`
}

type CoinutInstruments struct {
	Instruments map[string][]CoinutInstrumentBase `json:"SPOT"`
}

type CoinutTicker struct {
	HighestBuy   float64 `json:"highest_buy,string"`
	InstrumentID int     `json:"inst_id"`
	Last         float64 `json:"last,string"`
	LowestSell   float64 `json:"lowest_sell,string"`
	OpenInterest float64 `json:"open_interest,string"`
	Timestamp    float64 `json:"timestamp"`
	TransID      int64   `json:"trans_id"`
	Volume       float64 `json:"volume,string"`
	Volume24     float64 `json:"volume24,string"`
}

type CoinutOrderbookBase struct {
	Count    int     `json:"count"`
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"qty,string"`
}

type CoinutOrderbook struct {
	Buy          []CoinutOrderbookBase `json:"buy"`
	Sell         []CoinutOrderbookBase `json:"sell"`
	InstrumentID int                   `json:"inst_id"`
	TotalBuy     float64               `json:"total_buy,string"`
	TotalSell    float64               `json:"total_sell,string"`
	TransID      int64                 `json:"trans_id"`
}

type CoinutTradeBase struct {
	Price     float64 `json:"price,string"`
	Quantity  float64 `json:"quantity,string"`
	Side      string  `json:"side"`
	Timestamp float64 `json:"timestamp"`
	TransID   int64   `json:"trans_id"`
}

type CoinutTrades struct {
	Trades []CoinutTradeBase `json:"trades"`
}

type CoinutUserBalance struct {
	BTC               float64 `json:"btc,string"`
	ETC               float64 `json:"etc,string"`
	ETH               float64 `json:"eth,string"`
	LTC               float64 `json:"ltc,string"`
	Equity            float64 `json:"equity,string,string"`
	InitialMargin     float64 `json:"initial_margin,string"`
	MaintenanceMargin float64 `json:"maintenance_margin,string"`
	RealizedPL        float64 `json:"realized_pl,string"`
	TransID           int64   `json:"trans_id"`
	UnrealizedPL      float64 `json:"unrealized_pl,string"`
}

type CoinutOrder struct {
	InstrumentID  int64   `json:"inst_id"`
	Price         float64 `json:"price,string"`
	Quantity      float64 `json:"qty,string"`
	ClientOrderID int     `json:"client_ord_id"`
	Side          string  `json:"side,string"`
}

type CoinutOrderResponse struct {
	OrderID       int64   `json:"order_id"`
	OpenQuantity  float64 `json:"open_qty,string"`
	Price         float64 `json:"price,string"`
	Quantity      float64 `json:"qty,string"`
	InstrumentID  int64   `json:"inst_id"`
	ClientOrderID int64   `json:"client_ord_id"`
	Timestamp     int64   `json:"timestamp"`
	OrderPrice    float64 `json:"order_price,string"`
	Side          string  `json:"side"`
}

type CoinutCommission struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount,string"`
}

type CoinutOrderFilledResponse struct {
	CoinutGenericResponse
	Commission   CoinutCommission    `json:"commission"`
	FillPrice    float64             `json:"fill_price,string"`
	FillQuantity float64             `json:"fill_qty,string"`
	Order        CoinutOrderResponse `json:"order"`
}

type CoinutOrderRejectResponse struct {
	CoinutOrderResponse
	Reasons []string `json:"reasons"`
}

type CoinutOrdersBase struct {
	CoinutGenericResponse
	CoinutOrderResponse
}

type CoinutOrdersResponse struct {
	Data []CoinutOrdersBase
}

type CoinutCancelOrders struct {
	InstrumentID int   `json:"int"`
	OrderID      int64 `json:"order_id"`
}

type CoinutCancelOrdersResponse struct {
	CoinutGenericResponse
	Results []struct {
		OrderID      int64  `json:"order_id"`
		Status       string `json:"status"`
		InstrumentID int    `json:"inst_id"`
	} `json:"results"`
}

type CoinutTradeHistory struct {
	TotalNumber int64                       `json:"total_number"`
	Trades      []CoinutOrderFilledResponse `json:"trades"`
}

type CoinutIndexTicker struct {
	Asset string  `json:"asset"`
	Price float64 `json:"price,string"`
}

type CoinutOption struct {
	HighestBuy   float64 `json:"highest_buy,string"`
	InstrumentID int     `json:"inst_id"`
	Last         float64 `json:"last,string"`
	LowestSell   float64 `json:"lowest_sell,string"`
	OpenInterest float64 `json:"open_interest,string"`
}

type CoinutOptionChainResponse struct {
	ExpiryTime   int64  `json:"expiry_time"`
	SecurityType string `json:"sec_type"`
	Asset        string `json:"asset"`
	Entries      []struct {
		Call   CoinutOption `json:"call"`
		Put    CoinutOption `json:"put"`
		Strike float64      `json:"strike,string"`
	}
}

type CoinutOptionChainUpdate struct {
	CoinutOption
	CoinutGenericResponse
	Asset        string  `json:"asset"`
	ExpiryTime   int64   `json:"expiry_time"`
	SecurityType string  `json:"sec_type"`
	Volume       float64 `json:"volume,string"`
}

type CoinutPositionHistory struct {
	Positions []struct {
		PositionID int `json:"position_id"`
		Records    []struct {
			Commission    CoinutCommission `json:"commission"`
			FillPrice     float64          `json:"fill_price,string,omitempty"`
			TransactionID int              `json:"trans_id"`
			FillQuantity  float64          `json:"fill_qty,omitempty"`
			Position      struct {
				Commission CoinutCommission `json:"commission"`
				Timestamp  int64            `json:"timestamp"`
				OpenPrice  float64          `json:"open_price,string"`
				RealizedPL float64          `json:"realized_pl,string"`
				Quantity   float64          `json:"qty,string"`
			} `json:"position"`
			AssetAtExpiry float64 `json:"asset_at_expiry,string,omitempty"`
		} `json:"records"`
		Instrument struct {
			ExpiryTime     int64   `json:"expiry_time"`
			ContractSize   float64 `json:"contract_size,string"`
			ConversionRate float64 `json:"conversion_rate,string"`
			OptionType     string  `json:"option_type"`
			InstrumentID   int     `json:"inst_id"`
			SecType        string  `json:"sec_type"`
			Asset          string  `json:"asset"`
			Strike         float64 `json:"strike,string"`
		} `json:"inst"`
		OpenTimestamp int64 `json:"open_timestamp"`
	} `json:"positions"`
	TotalNumber int `json:"total_number"`
}

type CoinutOpenPosition struct {
	PositionID    int              `json:"position_id"`
	Commission    CoinutCommission `json:"commission"`
	OpenPrice     float64          `json:"open_price,string"`
	RealizedPL    float64          `json:"realized_pl,string"`
	Quantity      float64          `json:"qty,string"`
	OpenTimestamp int64            `json:"open_timestamp"`
	InstrumentID  int              `json:"inst_id"`
}
