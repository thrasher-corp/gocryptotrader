package main

// GetManagedPositionsParams defines command-line flags for exchange managed positions retrieval and unmarshal their values.
type GetManagedPositionsParams struct {
	Exchange              string `name:"exchange,e"                          required:"t"                                                                        usage:"the exchange to retrieve futures positions from"`
	Asset                 string `name:"asset,a"                             required:"t"                                                                        usage:"the asset type of the currency pair, must be a futures type"`
	Pair                  string `name:"pair,p"                              required:"t"                                                                        usage:"the currency pair of the position"`
	IncludeOrderDetails   bool   `name:"includeorderdetails,orders"          usage:"includes all orders that make up a position in the response"`
	GetFundingData        bool   `name:"getfundingdata,funding,fd"           usage:"if true, will return funding rate summary"`
	IncludeFundingEntries bool   `name:"includefundingentries,allfunding,af" usage:"if true, will return all funding rate entries - requires --getfundingdata"`
	IncludePredictedRate  bool   `name:"includepredictedrate,predicted,pr"   usage:"if true, will return the predicted funding rate - requires --getfundingdata"`
}

// GetAllManagedPositions defines command-line flags for exchange all managed positions retrieval and unmarshal their values.
type GetAllManagedPositions struct {
	IncludeOrderDetails   bool `name:"includeorderdetails,orders"          usage:"includes all orders that make up a position in the response"`
	GetFundingData        bool `name:"getfundingdata,funding,fd"           usage:"if true, will return funding rate summary"`
	IncludeFundingEntries bool `name:"includefundingentries,allfunding,af" usage:"if true, will return all funding rate entries - requires --getfundingdata"`
	IncludePredictedRate  bool `name:"includepredictedrate,predicted,pr"   usage:"if true, will return the predicted funding rate - requires --getfundingdata"`
}

// GetCollateralParams defines command-line flags for exchange asset collateral retrieval and unmarshal their values.
type GetCollateralParams struct {
	Exchange          string `name:"exchange,e"          required:"true"                                                                                                                       usage:"the exchange to retrieve futures positions from"`
	Asset             string `name:"asset,a"             required:"true"                                                                                                                       usage:"the asset type of the currency pair, must be a futures type"`
	CalculateOffline  bool   `name:"calculateoffline,c"  usage:"use local scaling calculations instead of requesting the collateral values directly, depending on individual exchange support"`
	IncludeBreakdown  bool   `name:"includebreakdown,i"  usage:"include a list of each held currency and its contribution to the overall collateral value"`
	IncludeZeroValues bool   `name:"includezerovalues,z" usage:"include collateral values that are zero"`
}

// GetFundingRates defines command-line flags for exchange currency pair funding rate retrieval and unmarshal their values.
type GetFundingRates struct {
	Exchange             string `name:"exchange,e"                     required:"t"                                                                                                             usage:"the exchange to retrieve futures positions from"`
	Asset                string `name:"asset,a"                        required:"t"                                                                                                             usage:"the asset type of the currency pair, must be a futures type"`
	Pair                 string `name:"pair,p"                         required:"t"                                                                                                             usage:"currency pair"`
	Start                string `name:"start,sd"                       usage:"<start> rounded down to the nearest hour"`
	End                  string `name:"end,ed"                         usage:"<end>"`
	Currency             string `name:"paymentcurrency,pc"             usage:"optional - if you are paid in a currency that isn't easily inferred from the Pair, eg BTCUSD-PERP use this field"`
	IncludePredicted     bool   `name:"includepredicted,ip,predicted"  usage:"optional - include the predicted next funding rate"`
	IncludePayments      bool   `name:"includepayments,pay"            usage:"optional - include funding rate payments, must be authenticated"`
	RespectHistoryLimits bool   `name:"respecthistorylimits,respect,r" usage:"optional - if true, will change the starting date to the maximum allowable limit if start date exceeds it"`
}

// GetLatestFundingRateParams defines command-line flags for exchange latest funding rate retrieval and unmarshal their values.
type GetLatestFundingRateParams struct {
	Exchange         string `name:"exchange,e"                    required:"t"                                               usage:"the exchange to retrieve futures positions from"`
	Asset            string `name:"asset,a"                       required:"t"                                               usage:"the asset type of the currency pair, must be a futures type"`
	Pair             string `name:"pair,p"                        required:"t"                                               usage:"currency pair"`
	IncludePredicted bool   `name:"includepredicted,ip,predicted" usage:"optional - include the predicted next funding rate"`
}

// GetCollateralMode defines command-line flags for exchange collateral mode retrieval and unmarshal their values.
type GetCollateralMode struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to retrieve futures positions from"`
	Asset    string `name:"asset,a"    required:"t" usage:"the asset type of the currency pair, must be a futures type"`
}

// SetCollateralMode defines command-line flags for exchange collateral mode setting and unmarshal their values.
type SetCollateralMode struct {
	Exchange       string `name:"exchange,e"                     required:"t" usage:"the exchange to retrieve futures positions from"`
	Asset          string `name:"asset,a"                        required:"t" usage:"the asset type of the currency pair, must be a futures type"`
	CollateralMode string `name:"collateralmode,collateral,cm,c" required:"t" usage:"the collateral mode type, such as 'single', 'multi' or 'global'"`
}

// LeverageInfo defines command-line flags for exchange leverage setting/retrieval and unmarshal their values.
type LeverageInfo struct {
	Exchange   string `name:"exchange,e"             required:"t"                                                     usage:"the exchange to retrieve futures positions from"`
	Asset      string `name:"asset,a"                required:"t"                                                     usage:"the asset type of the currency pair, must be a futures type"`
	Pair       string `name:"pair,p"                 required:"t"                                                     usage:"the currency pair"`
	MarginType string `name:"margintype,margin,mt,m" required:"t"                                                     usage:"the margin type, such as 'isolated', 'multi' or 'cross'"`
	Side       string `name:"orderside,side,os,s,o"  usage:"optional - some exchanges distinguish between order side"`
}

// SetLeverage defines command-line flags for exchange leverage setting and unmarshal their values.
type SetLeverage struct {
	Exchange   string  `name:"exchange,e"             required:"t"                                                     usage:"the exchange to retrieve futures positions from"`
	Asset      string  `name:"asset,a"                required:"t"                                                     usage:"the asset type of the currency pair, must be a futures type"`
	Pair       string  `name:"pair,p"                 required:"t"                                                     usage:"the currency pair"`
	MarginType string  `name:"margintype,margin,mt,m" required:"t"                                                     usage:"the margin type, such as 'isolated', 'multi' or 'cross'"`
	Side       string  `name:"orderside,side,os,s,o"  usage:"optional - some exchanges distinguish between order side"`
	Leverage   float64 `name:"leverage,l"             required:"t"                                                     usage:"the level of leverage you want, increase it to lose your capital faster"`
}

// ChangePositionMargin defines command-line flags for exchange position margin setting and unmarshal their values.
type ChangePositionMargin struct {
	Exchange                string  `name:"exchange,e"                  required:"t"                                      usage:"the exchange to retrieve futures positions from"`
	Asset                   string  `name:"asset,a"                     required:"t"                                      usage:"the asset type of the currency pair, must be a futures type"`
	Pair                    string  `name:"pair,p"                      required:"t"                                      usage:"the currency pair"`
	MarginType              string  `name:"margintype,margin,mt,m"      required:"t"                                      usage:"the margin type, most likely 'isolated'"`
	OriginalAllocatedMargin float64 `name:"originalallocatedmargin,oac" required:"t"                                      usage:"the original allocated margin, is used by some exchanges to determine differences to apply"`
	NewAllocatedMargin      float64 `name:"newallocatedmargin,nac"      required:"t"                                      usage:"the new allocated margin level you desire"`
	MarginSide              string  `name:"marginside,side,ms"          usage:"the new allocated margin level you desire"`
}

// GetFuturesPositionSummary defines command-line flags for exchange futures positions summary retrieval and unmarshal their values.
type GetFuturesPositionSummary struct {
	Exchange       string `name:"exchange,e"        required:"t"                                                                                                                                                                                                    usage:"the exchange to retrieve futures positions from"`
	Asset          string `name:"asset,a"           required:"t"                                                                                                                                                                                                    usage:"the asset type of the currency pair, must be a futures type"`
	Pair           string `name:"pair,p"            required:"t"                                                                                                                                                                                                    usage:"the currency pair"`
	UnderlyingPair string `name:"underlyingpair,up" usage:"optional - used to distinguish the underlying currency of a futures pair eg pair is BTCUSD-1337-C, the underlying pair could be BTC-USD, or if pair is LTCUSD-PERP the underlying pair could be LTC-USD"`
}

// GetFuturePositionOrders defines command-line flags for exchange futures positions order retrieval and unmarshal their values.
type GetFuturePositionOrders struct {
	Exchange                  string `name:"exchange,e"                  required:"t"                                                                                                usage:"the exchange to retrieve futures positions from"`
	Asset                     string `name:"asset,a"                     required:"t"                                                                                                usage:"the asset type of the currency pair, must be a futures type"`
	Pair                      string `name:"pair,p"                      required:"t"                                                                                                usage:"the currency pair"`
	Start                     string `name:"start,sd"                    usage:"<start> rounded down to the nearest hour"`
	End                       string `name:"end,ed"                      usage:"<end>"`
	RespectOrderHistoryLimits bool   `name:"respectorderhistorylimits,r" usage:"recommended true - if set to true, will not request orders beyond its API limits, preventing errors"`
	UnderlyingPair            string `name:"underlyingpair,up"           usage:"optional - the underlying currency pair"`
	SyncWithOrderManager      bool   `name:"syncwithordermanager,sync,s" usage:"if true, will sync the orders with the order manager if supported"`
}

// SetMarginType defines command-line flags for exchange margin type setting and unmarshal their values.
type SetMarginType struct {
	Exchange   string `name:"exchange,e"             required:"t" usage:"the exchange to retrieve futures positions from"`
	Asset      string `name:"asset,a"                required:"t" usage:"the asset type of the currency pair, must be a futures type"`
	Pair       string `name:"pair,p"                 required:"t" usage:"the currency pair"`
	MarginType string `name:"margintype,margin,mt,m" required:"t" usage:"the margin type, such as 'isolated', 'multi' or 'cross'"`
}

// GetOpenInterest defines command-line flags for exchange open interest retrieval and unmarshal their values.
type GetOpenInterest struct {
	Exchange string `name:"exchange,e" required:"t"                                                                   usage:"the exchange to retrieve open interest from"`
	Asset    string `name:"asset,a"    usage:"optional - the asset type of the currency pair, must be a futures type"`
	Pair     string `name:"pair,p"     usage:"optional - the currency pair"`
}
