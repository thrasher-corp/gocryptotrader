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
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:    "getmanagedposition",
			Aliases: []string{"managedposition", "mp"},
			Usage:   "retrieves an open position monitored by the order manager",
			Action:  getManagedPosition,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.BoolFlag{
					Name:    includeOrderDetailsFlagName,
					Aliases: []string{"orders"},
					Usage:   includeOrderDetailsUsage,
				},
				&cli.BoolFlag{
					Name:    getFundingDataFlagName,
					Aliases: []string{"funding", "fd"},
					Usage:   getFundingDataUsage,
				},
				&cli.BoolFlag{
					Name:    includeFundingEntriesFlagName,
					Aliases: []string{"allfunding", "af"},
					Usage:   includeFundingEntriesUsage,
				},
				&cli.BoolFlag{
					Name:    includePredictedRateFlagName,
					Aliases: []string{"predicted", "pr"},
					Usage:   includePredictedRateUsage,
				},
			},
		},
		{
			Name:    "getallmanagedpositions",
			Aliases: []string{"managedpositions", "mps"},
			Usage:   "retrieves all open positions monitored by the order manager",
			Action:  getAllManagedPositions,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    includeOrderDetailsFlagName,
					Aliases: []string{"orders"},
					Usage:   includeOrderDetailsUsage,
				},
				&cli.BoolFlag{
					Name:    getFundingDataFlagName,
					Aliases: []string{"funding", "fd"},
					Usage:   getFundingDataUsage,
				},
				&cli.BoolFlag{
					Name:    includeFundingEntriesFlagName,
					Aliases: []string{"allfunding", "af"},
					Usage:   includeFundingEntriesUsage,
				},
				&cli.BoolFlag{
					Name:    includePredictedRateFlagName,
					Aliases: []string{"predicted", "pr"},
					Usage:   includePredictedRateUsage,
				},
			},
		},
		{
			Name:    "getcollateral",
			Aliases: []string{"collateral", "c"},
			Usage:   "returns total collateral for an exchange asset, with optional per currency breakdown",
			Action:  getCollateral,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				&cli.BoolFlag{
					Name:    "calculateoffline",
					Aliases: []string{"c"},
					Usage:   "use local scaling calculations instead of requesting the collateral values directly, depending on individual exchange support",
				},
				&cli.BoolFlag{
					Name:    "includebreakdown",
					Aliases: []string{"i"},
					Usage:   "include a list of each held currency and its contribution to the overall collateral value",
				},
				&cli.BoolFlag{
					Name:    "includezerovalues",
					Aliases: []string{"z"},
					Usage:   "include collateral values that are zero",
				},
			},
		},
		{
			Name:    "getfundingrates",
			Aliases: []string{"funding", "f"},
			Usage:   "returns funding rate data between two dates",
			Action:  getFundingRates,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    startFlagName,
					Aliases: []string{"sd"},
					Usage:   startRoundedUsage,
					Value:   time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:    endFlagName,
					Aliases: []string{"ed"},
					Usage:   endUsage,
					Value:   time.Now().Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:    "paymentcurrency",
					Aliases: []string{"pc"},
					Usage:   "optional - if you are paid in a currency that isn't easily inferred from the Pair, eg BTCUSD-PERP use this field",
				},
				&cli.BoolFlag{
					Name:    includePredictedFlagName,
					Aliases: []string{"ip", "predicted"},
					Usage:   includePredictedUsage,
				},
				&cli.BoolFlag{
					Name:    "includepayments",
					Aliases: []string{"pay"},
					Usage:   "optional - include funding rate payments, must be authenticated",
				},
				&cli.BoolFlag{
					Name:    "respecthistorylimits",
					Aliases: []string{"respect", "r"},
					Usage:   "optional - if true, will change the starting date to the maximum allowable limit if start date exceeds it",
				},
			},
		},
		{
			Name:    "getlatestfundingrate",
			Aliases: []string{"latestrate", "lr", "r8"},
			Usage:   "returns the latest funding rate data",
			Action:  getLatestFundingRate,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.BoolFlag{
					Name:    includePredictedFlagName,
					Aliases: []string{"ip", "predicted"},
					Usage:   includePredictedUsage,
				},
			},
		},
		{
			Name:    "getcollateralmode",
			Aliases: []string{"gcm"},
			Usage:   "gets the collateral mode for an exchange asset",
			Action:  getCollateralMode,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
			},
		},
		{
			Name:    "setcollateralmode",
			Aliases: []string{"scm"},
			Usage:   "sets the collateral mode for an exchange asset",
			Action:  setCollateralMode,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				&cli.StringFlag{
					Name:     "collateralmode",
					Required: true,
					Aliases:  []string{"collateral", "cm", "c"},
					Usage:    "the collateral mode type, such as 'single', 'multi' or 'global'",
				},
			},
		},
		{
			Name:    "setleverage",
			Aliases: []string{"sl"},
			Usage:   "sets the initial leverage level for an exchange currency pair",
			Action:  setLeverage,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    marginTypeFlagName,
					Aliases: []string{"margin", "mt", "m"},
					Usage:   marginTypeUsage,
				},
				&cli.Float64Flag{
					Name:     "leverage",
					Required: true,
					Aliases:  []string{"l"},
					Usage:    "the level of leverage you want, increase it to lose your capital faster",
				},
				&cli.StringFlag{
					Name:    orderSideFlagName,
					Aliases: []string{sideFlagName, "os", "o"},
					Usage:   orderSideUsage,
				},
			},
		},
		{
			Name:    "getleverage",
			Aliases: []string{"gl"},
			Usage:   "gets the initial leverage level for an exchange currency pair",
			Action:  getLeverage,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    marginTypeFlagName,
					Aliases: []string{"margin", "mt", "m"},
					Usage:   marginTypeUsage,
				},
				&cli.StringFlag{
					Name:    orderSideFlagName,
					Aliases: []string{sideFlagName, "os", "o"},
					Usage:   orderSideUsage,
				},
			},
		},
		{
			Name:    "changepositionmargin",
			Aliases: []string{"cpm"},
			Usage:   "sets isolated margin levels for an existing position",
			Action:  changePositionMargin,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    marginTypeFlagName,
					Aliases: []string{"margin", "mt", "m"},
					Usage:   "the margin type, most likely 'isolated'",
				},
				&cli.Float64Flag{
					Name:    "originalallocatedmargin",
					Aliases: []string{"oac"},
					Usage:   "the original allocated margin, is used by some exchanges to determine differences to apply",
				},
				&cli.Float64Flag{
					Name:     "newallocatedmargin",
					Required: true,
					Aliases:  []string{"nac"},
					Usage:    "the new allocated margin level you desire",
				},
				&cli.StringFlag{
					Name:    "marginside",
					Aliases: []string{sideFlagName, "ms"},
					Usage:   "optional - the margin side, typically 'buy' or 'sell'",
				},
			},
		},
		{
			Name:    "getfuturespositionsummary",
			Aliases: []string{"summary", "fps"},
			Usage:   "return a summary of your futures position",
			Action:  getFuturesPositionSummary,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    underlyingPairFlagName,
					Aliases: []string{"up"},
					Usage:   underlyingPairUsage,
				},
			},
		},
		{
			Name:    "getfuturepositionorders",
			Aliases: []string{"orders", "fpo"},
			Usage:   "return a slice of orders that make up your position",
			Action:  getFuturePositionOrders,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:    startFlagName,
					Aliases: []string{"sd"},
					Usage:   startRoundedUsage,
					Value:   time.Now().AddDate(0, 0, -7).Truncate(time.Hour).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:    endFlagName,
					Aliases: []string{"ed"},
					Usage:   endUsage,
					Value:   time.Now().Format(time.DateTime),
				},
				&cli.BoolFlag{
					Name:    "respectorderhistorylimits",
					Aliases: []string{"r"},
					Usage:   "recommended true - if set to true, will not request orders beyond its API limits, preventing errors",
				},
				&cli.StringFlag{
					Name:    underlyingPairFlagName,
					Aliases: []string{"up"},
					Usage:   underlyingPairUsage,
				},
				&cli.BoolFlag{
					Name:    "syncwithordermanager",
					Aliases: []string{"sync", "s"},
					Usage:   "if true, will sync the orders with the order manager if supported",
				},
			},
		},
		{
			Name:    "setmargintype",
			Aliases: []string{"smt"},
			Usage:   "sets the margin type for a exchange asset pair",
			Action:  setMarginType,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredFuturesAssetFlag,
				requiredPairAliasFlag,
				&cli.StringFlag{
					Name:     marginTypeFlagName,
					Required: true,
					Aliases:  []string{"margin", "mt", "m"},
					Usage:    marginTypeUsage,
				},
			},
		},
		{
			Name:    "getopeninterest",
			Aliases: []string{"goi", "oi"},
			Usage:   "gets the open interest for provided exchange asset pair, if asset pair is not present, return all available if supported",
			Action:  getOpenInterest,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				&cli.StringFlag{
					Name:    assetFlagName,
					Aliases: []string{"a"},
					Usage:   futuresAssetUsage,
				},
				&cli.StringFlag{
					Name:    pairFlagName,
					Aliases: []string{"p"},
					Usage:   pairUsage,
				},
			},
		},
	},
}

func getManagedPosition(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)
	if err := isFuturesAsset(assetType); err != nil {
		return err
	}
	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	getFundingData := c.Bool(getFundingDataFlagName)
	includeFundingEntries := c.Bool(includeFundingEntriesFlagName)

	includePredictedRate := c.Bool(includePredictedRateFlagName)

	if err := futures.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries); err != nil {
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludeFullOrderData:    c.Bool(includeOrderDetailsFlagName),
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
			IncludePredictedRate:    includePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getAllManagedPositions(c *cli.Context) error {
	getFundingData := c.Bool(getFundingDataFlagName)
	includeFundingEntries := c.Bool(includeFundingEntriesFlagName)

	includePredictedRate := c.Bool(includePredictedRateFlagName)

	if err := futures.CheckFundingRatePrerequisites(getFundingData, includePredictedRate, includeFundingEntries); err != nil {
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
			IncludeFullOrderData:    c.Bool(includeOrderDetailsFlagName),
			GetFundingPayments:      getFundingData,
			IncludeFullFundingRates: includeFundingEntries,
			IncludePredictedRate:    includePredictedRate,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateral(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	assetType := c.String(assetFlagName)
	if err := isFuturesAsset(assetType); err != nil {
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
			Exchange:          c.String(exchangeFlagName),
			Asset:             assetType,
			IncludeBreakdown:  c.Bool("includebreakdown"),
			CalculateOffline:  c.Bool("calculateoffline"),
			IncludeZeroValues: c.Bool("includezerovalues"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFundingRates(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		p    currency.Pair
		s, e time.Time
		err  error
	)

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}
	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}
	p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			StartDate:            s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:              e.Format(common.SimpleTimeFormatWithTimezone),
			IncludePredicted:     c.Bool(includePredictedFlagName),
			IncludePayments:      c.Bool("includepayments"),
			RespectHistoryLimits: c.Bool("respecthistorylimits"),
			PaymentCurrency:      c.String("paymentcurrency"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLatestFundingRate(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		p   currency.Pair
		err error
	)

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}
	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}
	p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			IncludePredicted: c.Bool(includePredictedFlagName),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getCollateralMode(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setCollateralMode(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	collateralMode := c.String("collateralmode")

	if !collateral.IsValidCollateralModeString(collateralMode) {
		return fmt.Errorf("invalid collateral mode: %v", collateralMode)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetCollateralMode(c.Context,
		&gctrpc.SetCollateralModeRequest{
			Exchange:       c.String(exchangeFlagName),
			Asset:          assetType,
			CollateralMode: collateralMode,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setLeverage(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	marginType := c.String(marginTypeFlagName)
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	orderSide := c.String(orderSideFlagName)
	if orderSide != "" {
		_, err = order.StringToOrderSide(orderSide)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
			Leverage:   c.Float64("leverage"),
			OrderSide:  orderSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getLeverage(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	marginType := c.String(marginTypeFlagName)
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	orderSide := c.String(orderSideFlagName)
	if orderSide != "" {
		_, err = order.StringToOrderSide(orderSide)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
			OrderSide:  orderSide,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func changePositionMargin(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	marginType := c.String(marginTypeFlagName)
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.ChangePositionMargin(c.Context,
		&gctrpc.ChangePositionMarginRequest{
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType:              marginType,
			OriginalAllocatedMargin: c.Float64("originalallocatedmargin"),
			NewAllocatedMargin:      c.Float64("newallocatedmargin"),
			MarginSide:              c.String("marginside"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getFuturesPositionSummary(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	underlyingPair := c.String(underlyingPairFlagName)
	var underlying currency.Pair
	if underlyingPair != "" {
		underlying, err = currency.NewPairDelimiter(underlyingPair, pairDelimiter)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
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
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	var (
		s, e time.Time
		err  error
	)

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}
	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}

	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}
	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	underlyingPair := c.String(underlyingPairFlagName)
	var underlying currency.Pair
	if underlyingPair != "" {
		underlying, err = currency.NewPairDelimiter(underlyingPair, pairDelimiter)
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
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
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
			SyncWithOrderManager:      c.Bool("syncwithordermanager"),
			RespectOrderHistoryLimits: c.Bool("respectorderhistorylimits"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setMarginType(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if err := isFuturesAsset(assetType); err != nil {
		return err
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
	}
	pair, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	marginType := c.String(marginTypeFlagName)
	if !margin.IsValidString(marginType) {
		return fmt.Errorf("%w margintype:%v", margin.ErrInvalidMarginType, marginType)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetMarginType(c.Context,
		&gctrpc.SetMarginTypeRequest{
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pair.Delimiter,
				Base:      pair.Base.String(),
				Quote:     pair.Quote.String(),
			},
			MarginType: marginType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getOpenInterest(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if assetType != "" {
		if err := isFuturesAsset(assetType); err != nil {
			return err
		}
	}

	currencyPair := c.String(pairFlagName)
	var pair currency.Pair
	if currencyPair != "" {
		if !validPair(currencyPair) {
			return fmt.Errorf("%w currencypair:%v", errInvalidPair, currencyPair)
		}
		var err error
		pair, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
		if err != nil {
			return err
		}
	}

	data := make([]*gctrpc.OpenInterestDataRequest, 0, 1)
	if !pair.IsEmpty() {
		data = append(data, &gctrpc.OpenInterestDataRequest{
			Asset: assetType,
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
			Exchange: c.String(exchangeFlagName),
			Data:     data,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
