package main

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

// futuresCommands contains all commands related to futures
// position data, funding rates, collateral, pnl etc
var futuresCommands = &cli.Command{
	Name:      "futures",
	Aliases:   []string{"f"},
	Usage:     "contains all futures based rpc commands",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "getmanagedposition",
			Aliases:   []string{"managedposition", "mp"},
			Usage:     "retrieves an open position monitored by the order manager",
			ArgsUsage: "<exchange> <asset> <pair> <includeorderdetails> <getfundingdata> <includefundingentries> <includepredictedrate>",
			Action:    getManagedPosition,
			Flags:     FlagsFromStruct(&GetManagedPositionsParams{}),
		},
		{
			Name:      "getallmanagedpositions",
			Aliases:   []string{"managedpositions", "mps"},
			Usage:     "retrieves all open positions monitored by the order manager",
			ArgsUsage: "<includeorderdetails> <getfundingdata> <includefundingentries> <includepredictedrate>",
			Action:    getAllManagedPositions,
			Flags:     FlagsFromStruct(&GetAllManagedPositions{}),
		},
		{
			Name:      "getcollateral",
			Aliases:   []string{"collateral", "c"},
			Usage:     "returns total collateral for an exchange asset, with optional per currency breakdown",
			ArgsUsage: "<exchange> <asset> <calculateoffline> <includebreakdown> <includezerovalues>",
			Action:    getCollateral,
			Flags:     FlagsFromStruct(&GetCollateralParams{}),
		},
		{
			Name:      "getfundingrates",
			Aliases:   []string{"funding", "f"},
			Usage:     "returns funding rate data between two dates",
			ArgsUsage: "<exchange> <asset> <pair> <start> <end> <paymentcurrency> <includepredicted> <includepayments> <respecthistorylimits>",
			Action:    getFundingRates,
			Flags: FlagsFromStruct(&GetFundingRates{
				Start: time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
				End:   time.Now().Format(time.DateTime),
			}),
		},
		{
			Name:      "getlatestfundingrate",
			Aliases:   []string{"latestrate", "lr", "r8"},
			Usage:     "returns the latest funding rate data",
			ArgsUsage: "<exchange> <asset> <pair> <includepredicted>",
			Action:    getLatestFundingRate,
			Flags:     FlagsFromStruct(&GetLatestFundingRateParams{}),
		},
		{
			Name:      "getcollateralmode",
			Aliases:   []string{"gcm"},
			Usage:     "gets the collateral mode for an exchange asset",
			ArgsUsage: "<exchange> <asset>",
			Action:    getCollateralMode,
			Flags:     FlagsFromStruct(&GetCollateralMode{}),
		},
		{
			Name:      "setcollateralmode",
			Aliases:   []string{"scm"},
			Usage:     "sets the collateral mode for an exchange asset",
			ArgsUsage: "<exchange> <asset> <collateralmode>",
			Action:    setCollateralMode,
			Flags:     FlagsFromStruct(&SetCollateralMode{}),
		},
		{
			Name:      "setleverage",
			Aliases:   []string{"sl"},
			Usage:     "sets the initial leverage level for an exchange currency pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype> <leverage> <orderside>",
			Action:    setLeverage,
			Flags:     FlagsFromStruct(&SetLeverage{}),
		},
		{
			Name:      "getleverage",
			Aliases:   []string{"gl"},
			Usage:     "gets the initial leverage level for an exchange currency pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype> <orderside>",
			Action:    getLeverage,
			Flags:     FlagsFromStruct(&LeverageInfo{}),
		},
		{
			Name:      "changepositionmargin",
			Aliases:   []string{"cpm"},
			Usage:     "sets isolated margin levels for an existing position",
			ArgsUsage: "<exchange> <asset> <pair> <margintype> <originalallocatedmargin> <newallocatedmargin> <marginside>",
			Action:    changePositionMargin,
			Flags:     FlagsFromStruct(&ChangePositionMargin{}),
		},
		{
			Name:      "getfuturespositionsummary",
			Aliases:   []string{"summary", "fps"},
			Usage:     "return a summary of your futures position",
			ArgsUsage: "<exchange> <asset> <pair> <underlyingpair>",
			Action:    getFuturesPositionSummary,
			Flags:     FlagsFromStruct(&GetFuturesPositionSummary{}),
		},
		{
			Name:      "getfuturepositionorders",
			Aliases:   []string{"orders", "fpo"},
			Usage:     "return a slice of orders that make up your position",
			ArgsUsage: "<exchange> <asset> <pair> <start> <end> <respectorderhistorylimits> <underlyingpair> <syncwithordermanager>",
			Action:    getFuturePositionOrders,
			Flags: FlagsFromStruct(&GetFuturePositionOrders{
				Start: time.Now().AddDate(0, 0, -7).Truncate(time.Hour).Format(time.DateTime),
				End:   time.Now().Format(time.DateTime),
			}),
		},
		{
			Name:      "setmargintype",
			Aliases:   []string{"smt"},
			Usage:     "sets the margin type for a exchange asset pair",
			ArgsUsage: "<exchange> <asset> <pair> <margintype>",
			Action:    setMarginType,
			Flags:     FlagsFromStruct(&SetMarginType{}),
		},
		{
			Name:      "getopeninterest",
			Aliases:   []string{"goi", "oi"},
			Usage:     "gets the open interest for provided exchange asset pair, if asset pair is not present, return all available if supported",
			ArgsUsage: "<exchange> <asset> <pair>",
			Action:    getOpenInterest,
			Flags:     FlagsFromStruct(&GetOpenInterest{}),
		},
	},
}

func getManagedPosition(c *cli.Context) error {
	println("runningg")
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetManagedPositionsParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	err = isFuturesAsset(arg.Asset)
	if err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	err = futures.CheckFundingRatePrerequisites(arg.GetFundingData, arg.IncludePredictedRate, arg.IncludeFundingEntries)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetManagedPosition(c.Context,
		&gctrpc.GetManagedPositionRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludeFullOrderData:    arg.IncludeOrderDetails,
			GetFundingPayments:      arg.GetFundingData,
			IncludeFullFundingRates: arg.IncludeFundingEntries,
			IncludePredictedRate:    arg.IncludePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getAllManagedPositions(c *cli.Context) error {
	arg := &GetAllManagedPositions{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	err = futures.CheckFundingRatePrerequisites(arg.GetFundingData, arg.IncludePredictedRate, arg.IncludeFundingEntries)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAllManagedPositions(c.Context,
		&gctrpc.GetAllManagedPositionsRequest{
			IncludeFullOrderData:    arg.IncludeOrderDetails,
			GetFundingPayments:      arg.GetFundingData,
			IncludeFullFundingRates: arg.IncludeFundingEntries,
			IncludePredictedRate:    arg.IncludePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateral(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetCollateralParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	err = isFuturesAsset(arg.Asset)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCollateral(c.Context,
		&gctrpc.GetCollateralRequest{
			Exchange:          arg.Exchange,
			Asset:             arg.Asset,
			IncludeBreakdown:  arg.IncludeBreakdown,
			CalculateOffline:  arg.CalculateOffline,
			IncludeZeroValues: arg.IncludeZeroValues,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFundingRates(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetFundingRates{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	err = isFuturesAsset(arg.Asset)
	if err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}
	var p currency.Pair
	p, err = currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetFundingRates(c.Context,
		&gctrpc.GetFundingRatesRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			StartDate:            s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:              e.Format(common.SimpleTimeFormatWithTimezone),
			IncludePredicted:     arg.IncludePredicted,
			IncludePayments:      arg.IncludePayments,
			RespectHistoryLimits: arg.RespectHistoryLimits,
			PaymentCurrency:      arg.Currency,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLatestFundingRate(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetLatestFundingRateParams{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}
	var (
		p   currency.Pair
		err error
	)
	p, err = currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetLatestFundingRate(c.Context,
		&gctrpc.GetLatestFundingRateRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludePredicted: arg.IncludePredicted,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateralMode(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetCollateralMode{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}
	err = isFuturesAsset(arg.Asset)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCollateralMode(c.Context,
		&gctrpc.GetCollateralModeRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setCollateralMode(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &SetCollateralMode{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}

	if !collateral.IsValidCollateralModeString(arg.CollateralMode) {
		return fmt.Errorf("%w: %v", collateral.ErrInvalidCollateralMode, arg.CollateralMode)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetCollateralMode(c.Context,
		&gctrpc.SetCollateralModeRequest{
			Exchange:       arg.Exchange,
			Asset:          arg.Asset,
			CollateralMode: arg.CollateralMode,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setLeverage(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &SetLeverage{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !margin.IsValidString(arg.MarginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, arg.MarginType)
	}

	if arg.Side != "" {
		_, err = order.StringToOrderSide(arg.Side)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetLeverage(c.Context,
		&gctrpc.SetLeverageRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: arg.MarginType,
			Leverage:   arg.Leverage,
			OrderSide:  arg.Side,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLeverage(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &LeverageInfo{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !margin.IsValidString(arg.MarginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, arg.MarginType)
	}
	if arg.Side != "" {
		_, err = order.StringToOrderSide(arg.Side)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetLeverage(c.Context,
		&gctrpc.GetLeverageRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: arg.MarginType,
			OrderSide:  arg.Side,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func changePositionMargin(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &ChangePositionMargin{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !margin.IsValidString(arg.MarginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, arg.MarginType)
	}
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.ChangePositionMargin(c.Context,
		&gctrpc.ChangePositionMarginRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType:              arg.MarginType,
			OriginalAllocatedMargin: arg.OriginalAllocatedMargin,
			NewAllocatedMargin:      arg.NewAllocatedMargin,
			MarginSide:              arg.MarginSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturesPositionSummary(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetFuturesPositionSummary{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	var underlying currency.Pair
	if arg.UnderlyingPair != "" {
		underlying, err = currency.NewPairFromString(arg.UnderlyingPair)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetFuturesPositionsSummary(c.Context,
		&gctrpc.GetFuturesPositionsSummaryRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			UnderlyingPair: &gctrpc.CurrencyPair{
				Delimiter: underlying.Delimiter,
				Base:      underlying.Base.Upper().String(),
				Quote:     underlying.Quote.Upper().String(),
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturePositionOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetFuturePositionOrders{
		Start: time.Now().AddDate(0, 0, -7).Truncate(time.Hour).Format(time.DateTime),
		End:   time.Now().Format(time.DateTime),
	}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	var s, e time.Time
	if arg.Start != "" {
		s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format for start: %v", err)
		}
	}
	if arg.End != "" {
		e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format for start: %v", err)
		}
	}
	err = common.StartEndTimeCheck(s, e)
	if err != nil {
		return err
	}

	var underlying currency.Pair
	if arg.UnderlyingPair != "" {
		underlying, err = currency.NewPairFromString(arg.UnderlyingPair)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetFuturesPositionsOrders(c.Context,
		&gctrpc.GetFuturesPositionsOrdersRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			StartDate: s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:   e.Format(common.SimpleTimeFormatWithTimezone),
			UnderlyingPair: &gctrpc.CurrencyPair{
				Delimiter: underlying.Delimiter,
				Base:      underlying.Base.Upper().String(),
				Quote:     underlying.Quote.Upper().String(),
			},
			SyncWithOrderManager:      arg.SyncWithOrderManager,
			RespectOrderHistoryLimits: arg.RespectOrderHistoryLimits,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setMarginType(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &SetMarginType{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if err := isFuturesAsset(arg.Asset); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
	}
	pair, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !margin.IsValidString(arg.MarginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, arg.MarginType)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetMarginType(c.Context,
		&gctrpc.SetMarginTypeRequest{
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: arg.MarginType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getOpenInterest(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetOpenInterest{}
	if err := UnmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if arg.Asset != "" {
		err := isFuturesAsset(arg.Asset)
		if err != nil {
			return err
		}
	}

	var pair currency.Pair
	if arg.Pair != "" {
		if !validPair(arg.Pair) {
			return fmt.Errorf("%w currencypair:%v", errInvalidPair, arg.Pair)
		}
		var err error
		pair, err = currency.NewPairDelimiter(arg.Pair, pairDelimiter)
		if err != nil {
			return err
		}
	}

	data := make([]*gctrpc.OpenInterestDataRequest, 0, 1)
	if !pair.IsEmpty() {
		data = append(data, &gctrpc.OpenInterestDataRequest{
			Asset: arg.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
		})
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOpenInterest(c.Context,
		&gctrpc.GetOpenInterestRequest{
			Exchange: arg.Exchange,
			Data:     data,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
