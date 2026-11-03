package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var tradeCommand = &cli.Command{
	Name:      "trade",
	Usage:     "execute trade related commands",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:   "setexchangetradeprocessing",
			Usage:  "sets whether an exchange can save trades to the database",
			Action: setExchangeTradeProcessing,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				&cli.BoolFlag{
					Name:     "status",
					Required: true,
					Usage:    "<true>/<false>",
				},
			},
		},
		{
			Name:   "getrecent",
			Usage:  "gets recent trades",
			Action: getRecentTrades,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredPairAliasFlag,
				requiredAssetAliasFlag,
			},
		},
		{
			Name:   "gethistoric",
			Usage:  "gets trades between two periods",
			Action: getHistoricTrades,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredPairAliasFlag,
				requiredAssetAliasFlag,
				&cli.StringFlag{
					Name:  startFlagName,
					Usage: startUsage,
					Value: time.Now().Add(-time.Hour * 6).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:  endFlagName,
					Usage: endUsage + " WARNING: large date ranges may take considerable time",
					Value: time.Now().Format(time.DateTime),
				},
			},
		},
		{
			Name:   "getsaved",
			Usage:  "gets trades from the database",
			Action: getSavedTrades,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredPairAliasFlag,
				requiredAssetAliasFlag,
				&cli.StringFlag{
					Name:  startFlagName,
					Usage: startUsage,
					Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:  endFlagName,
					Usage: endUsage,
					Value: time.Now().Format(time.DateTime),
				},
			},
		},
		{
			Name:   "findmissingsavedtradeintervals",
			Usage:  "will highlight any interval that is missing trade data so you can fill that gap",
			Action: findMissingSavedTradeIntervals,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredPairAliasFlag,
				requiredAssetAliasFlag,
				&cli.StringFlag{
					Name:  startFlagName,
					Usage: startRoundedUsage,
					Value: time.Now().Add(-time.Hour * 24).Truncate(time.Hour).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:  endFlagName,
					Usage: endRoundedUsage,
					Value: time.Now().Truncate(time.Hour).Format(time.DateTime),
				},
			},
		},
		{
			Name:   "convertsavedtradestocandles",
			Usage:  "explicitly converts stored trade data to candles and saves the result to the database",
			Action: convertSavedTradesToCandles,
			Flags: []cli.Flag{
				requiredExchangeAliasFlag,
				requiredPairAliasFlag,
				requiredAssetAliasFlag,
				&cli.Int64Flag{
					Name:    intervalFlagName,
					Aliases: []string{"i"},
					Usage:   klineMessage,
					Value:   86400,
				},
				&cli.StringFlag{
					Name:  startFlagName,
					Usage: startUsage,
					Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:  endFlagName,
					Usage: endUsage,
					Value: time.Now().Format(time.DateTime),
				},
				&cli.BoolFlag{
					Name:    syncFlagName,
					Aliases: []string{"s"},
					Usage:   "will sync the resulting candles to the database <true/false>",
				},
				&cli.BoolFlag{
					Name:    forceFlagName,
					Aliases: []string{"f"},
					Usage:   forceUsage,
				},
			},
		},
	},
}

func findMissingSavedTradeIntervals(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.FindMissingSavedTradeIntervals(c.Context,
		&gctrpc.FindMissingTradePeriodsRequest{
			ExchangeName: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
			Start:     s.Format(common.SimpleTimeFormatWithTimezone),
			End:       e.Format(common.SimpleTimeFormatWithTimezone),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func setExchangeTradeProcessing(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.SetExchangeTradeProcessing(c.Context,
		&gctrpc.SetExchangeTradeProcessingRequest{
			Exchange: c.String(exchangeFlagName),
			Status:   c.Bool("status"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getSavedTrades(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var s, e time.Time
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
	result, err := client.GetSavedTrades(c.Context,
		&gctrpc.GetSavedTradesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
			Start:     s.Format(common.SimpleTimeFormatWithTimezone),
			End:       e.Format(common.SimpleTimeFormatWithTimezone),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getRecentTrades(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetRecentTrades(c.Context,
		&gctrpc.GetSavedTradesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getHistoricTrades(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var s, e time.Time
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

	streamStartTime := time.Now()
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetHistoricTrades(c.Context,
		&gctrpc.GetSavedTradesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
			Start:     s.Format(common.SimpleTimeFormatWithTimezone),
			End:       e.Format(common.SimpleTimeFormatWithTimezone),
		})
	if err != nil {
		return err
	}
	fmt.Printf("%v\t| Beginning stream retrieving trades in 1 hour batches from %v to %v\n",
		time.Now().Format(time.Kitchen),
		s.UTC().Format(common.SimpleTimeFormatWithTimezone),
		e.UTC().Format(common.SimpleTimeFormatWithTimezone))
	fmt.Printf("%v\t| If you have provided a large time range, please be patient\n\n",
		time.Now().Format(time.Kitchen))
	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}
		if len(resp.Trades) == 0 {
			break
		}
		fmt.Printf("%v\t| Processed %v trades between %v and %v\n",
			time.Now().Format(time.Kitchen),
			len(resp.Trades),
			resp.Trades[0].Timestamp,
			resp.Trades[len(resp.Trades)-1].Timestamp)
	}

	fmt.Printf("%v\t| Trade retrieval complete! Process took %v\n",
		time.Now().Format(time.Kitchen),
		time.Since(streamStartTime))

	return nil
}

func convertSavedTradesToCandles(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	sync := c.Bool(syncFlagName)
	force := c.Bool(forceFlagName)

	if force && !sync {
		return errors.New("cannot forcefully overwrite without sync")
	}

	candleInterval := time.Duration(c.Int64(intervalFlagName)) * time.Second
	var s, e time.Time
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
	result, err := client.ConvertTradesToCandles(c.Context,
		&gctrpc.ConvertTradesToCandlesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:    assetType,
			Start:        s.Format(common.SimpleTimeFormatWithTimezone),
			End:          e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval: int64(candleInterval),
			Sync:         sync,
			Force:        force,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
