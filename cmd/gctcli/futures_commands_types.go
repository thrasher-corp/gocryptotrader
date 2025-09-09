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
