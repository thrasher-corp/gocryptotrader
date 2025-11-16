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
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "setexchangetradeprocessing",
			Usage:     "sets whether an exchange can save trades to the database",
			ArgsUsage: "<exchange> <status>",
			Action:    setExchangeTradeProcessing,
			Flags:     FlagsFromStruct(&SetExchangeTradeProcessingParams{}),
		},
		{
			Name:      "getrecent",
			Usage:     "gets recent trades",
			ArgsUsage: "<exchange> <pair> <asset>",
			Action:    getRecentTrades,
			Flags:     FlagsFromStruct(&GetRecentTradesParams{}),
		},
		{
			Name:      "gethistoric",
			Usage:     "gets trades between two periods",
			ArgsUsage: "<exchange> <pair> <asset> <start> <end>",
			Action:    getHistoricTrades,
			Flags: FlagsFromStruct(&GetTradesParams{
				Start: time.Now().Add(-time.Hour * 6).Format(time.DateTime),
				End:   time.Now().Format(time.DateTime),
			}),
		},
		{
			Name:      "getsaved",
			Usage:     "gets trades from the database",
			ArgsUsage: "<exchange> <pair> <asset> <start> <end>",
			Action:    getSavedTrades,
			Flags: FlagsFromStruct(&GetTradesParams{
				Start: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				End:   time.Now().Format(time.DateTime),
			}),
		},
		{
			Name:      "findmissingsavedtradeintervals",
			Usage:     "will highlight any interval that is missing trade data so you can fill that gap",
			ArgsUsage: "<exchange> <pair> <asset> <start> <end>",
			Action:    findMissingSavedTradeIntervals,
			Flags: FlagsFromStruct(&FindMisingSavedTradeIntervalsParams{
				Start: time.Now().Add(-time.Hour * 24).Truncate(time.Hour).Format(time.DateTime),
				End:   time.Now().Truncate(time.Hour).Format(time.DateTime),
			}),
		},
		{
			Name:      "convertsavedtradestocandles",
			Usage:     "explicitly converts stored trade data to candles and saves the result to the database",
			ArgsUsage: "<exchange> <pair> <asset> <interval> <start> <end>",
			Action:    convertSavedTradesToCandles,
			Flags: FlagsFromStruct(&ConvertSavedTradesToCandlesParams{
				Interval: 86400,
				Start:    time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				End:      time.Now().Format(time.DateTime),
			}),
		},
	},
}

func findMissingSavedTradeIntervals(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &FindMisingSavedTradeIntervalsParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		FindMissingSavedTradeIntervals(c.Context,
			&gctrpc.FindMissingTradePeriodsRequest{
				ExchangeName: arg.Exchange,
				Pair: &gctrpc.CurrencyPair{
					Delimiter: p.Delimiter,
					Base:      p.Base.String(),
					Quote:     p.Quote.String(),
				},
				AssetType: arg.Asset,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &SetExchangeTradeProcessingParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		SetExchangeTradeProcessing(c.Context,
			&gctrpc.SetExchangeTradeProcessingRequest{
				Exchange: arg.Exchange,
				Status:   arg.Status,
			})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getSavedTrades(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetTradesParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}
	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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

	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		GetSavedTrades(c.Context,
			&gctrpc.GetSavedTradesRequest{
				Exchange: arg.Exchange,
				Pair: &gctrpc.CurrencyPair{
					Delimiter: p.Delimiter,
					Base:      p.Base.String(),
					Quote:     p.Quote.String(),
				},
				AssetType: arg.Asset,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetRecentTradesParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		GetRecentTrades(c.Context,
			&gctrpc.GetSavedTradesRequest{
				Exchange: arg.Exchange,
				Pair: &gctrpc.CurrencyPair{
					Delimiter: p.Delimiter,
					Base:      p.Base.String(),
					Quote:     p.Quote.String(),
				},
				AssetType: arg.Asset,
			})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func getHistoricTrades(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetTradesParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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

	streamStartTime := time.Now()
	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		GetHistoricTrades(c.Context,
			&gctrpc.GetSavedTradesRequest{
				Exchange: arg.Exchange,
				Pair: &gctrpc.CurrencyPair{
					Delimiter: p.Delimiter,
					Base:      p.Base.String(),
					Quote:     p.Quote.String(),
				},
				AssetType: arg.Asset,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &ConvertSavedTradesToCandlesParams{}
	if err := unmarshalCLIFields(c, arg); err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairFromString(arg.Pair)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}
	if arg.Force && !arg.Sync {
		return errors.New("cannot forcefully overwrite without sync")
	}

	candleInterval := time.Duration(candleGranularity) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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

	result, err := gctrpc.NewGoCryptoTraderServiceClient(conn).
		ConvertTradesToCandles(c.Context,
			&gctrpc.ConvertTradesToCandlesRequest{
				Exchange: arg.Exchange,
				Pair: &gctrpc.CurrencyPair{
					Delimiter: p.Delimiter,
					Base:      p.Base.String(),
					Quote:     p.Quote.String(),
				},
				AssetType:    arg.Asset,
				Start:        s.Format(common.SimpleTimeFormatWithTimezone),
				End:          e.Format(common.SimpleTimeFormatWithTimezone),
				TimeInterval: int64(candleInterval),
				Sync:         arg.Sync,
				Force:        arg.Force,
			})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
