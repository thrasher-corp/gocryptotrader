package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var (
	maxRetryAttempts, requestSizeLimit, batchSize uint64
	guidExample                                   = "deadbeef-dead-beef-dead-beef13371337"
	specificJobSubCommands                        = []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: guidExample,
		},
		&cli.StringFlag{
			Name:  "nickname",
			Usage: "binance-spot-btc-usdt-2019-trades",
		},
	}
	fullJobSubCommands = []cli.Flag{
		&cli.StringFlag{
			Name:     "nickname",
			Usage:    "binance-spot-btc-usdt-2019-trades",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "binance",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "spot",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "btc-usdt",
		},
		&cli.StringFlag{
			Name:        "start_date",
			Usage:       "formatted as: 2006-01-02 15:04:05",
			Value:       time.Now().AddDate(-1, 0, 0).Format(common.SimpleTimeFormat),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end_date",
			Usage:       "formatted as: 2006-01-02 15:04:05",
			Value:       time.Now().AddDate(0, -1, 0).Format(common.SimpleTimeFormat),
			Destination: &endTime,
		},
		&cli.Uint64Flag{
			Name:  "interval",
			Usage: klineMessage,
		},
		&cli.Uint64Flag{
			Name:        "request_size_limit",
			Usage:       "the number of candles to retrieve per API request",
			Destination: &requestSizeLimit,
			Value:       500,
		},
		&cli.Uint64Flag{
			Name:  "data_type",
			Usage: "0 for candles, 1 for trades, 2 for convert trades, 3 for convert candles, 4 for validate candles",
		},
		&cli.Uint64Flag{
			Name:        "max_retry_attempts",
			Usage:       "the maximum retry attempts for an interval period before giving up",
			Value:       3,
			Destination: &maxRetryAttempts,
		},
		&cli.Uint64Flag{
			Name:        "batch_size",
			Usage:       "the amount of API calls to make per run",
			Destination: &batchSize,
			Value:       3,
		},
		&cli.Uint64Flag{
			Name:  "conversion_interval",
			Usage: klineMessage,
		},
		&cli.BoolFlag{
			Name:  "overwrite_existing_data",
			Usage: "when converting candles, if data already exists for the time period in the database, it will be overwritten",
		},
		&cli.StringFlag{
			Name:  "prerequisite_job_nickname",
			Usage: "optional, using an existing job nickname allows you to queue this new job to run after the previous job is finished",
		},
	}
)

var dataHistoryCommands = &cli.Command{
	Name:      "datahistory",
	Usage:     "manage data history jobs to retrieve historic trade or candle data over time",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:   "getactivejobs",
			Usage:  "returns all jobs that are currently active",
			Flags:  []cli.Flag{},
			Action: getActiveDataHistoryJobs,
		},
		{
			Name:  "getjobsbetweendates",
			Usage: "returns all jobs with creation dates between the two provided dates",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "start_date",
					Usage: "formatted as: 2006-01-02 15:04:05",
				},
				&cli.StringFlag{
					Name:  "end_date",
					Usage: "formatted as: 2006-01-02 15:04:05",
				},
			},
			Action: getDataHistoryJobsBetween,
		},
		{
			Name:        "getajob",
			Usage:       "returns a job by either its id or nickname",
			Description: "na-na, why don't you get a job?",
			ArgsUsage:   "<id> or <nickname>",
			Action:      getDataHistoryJob,
			Flags:       specificJobSubCommands,
		},
		{
			Name:        "getjobwithdetailedresults",
			Usage:       "returns a job by either its nickname along with all its data retrieval results",
			Description: "results may be large",
			ArgsUsage:   "<nickname>",
			Action:      getDataHistoryJob,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "nickname",
					Usage: "binance-spot-btc-usdt-2019-trades",
				},
			},
		},
		{
			Name:      "getjobstatussummary",
			Usage:     "returns a job with human readable summary of its status",
			ArgsUsage: "<nickname>",
			Action:    getDataHistoryJobSummary,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "nickname",
					Usage: "binance-spot-btc-usdt-2019-trades",
				},
			},
		},
		{
			Name:   "addnewjob",
			Usage:  "creates a new data history job",
			Flags:  fullJobSubCommands,
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "upsertjob",
			Usage:  "adds a new job, or updates an existing one if it matches jobid OR nickname",
			Flags:  fullJobSubCommands,
			Action: upsertDataHistoryJob,
		},
		{
			Name:      "deletejob",
			Usage:     "sets a jobs status to deleted so it no longer is processed",
			ArgsUsage: "<id> or <nickname>",
			Flags:     specificJobSubCommands,
			Action:    setDataHistoryJobStatus,
		},
		{
			Name:      "pausejob",
			Usage:     "sets a jobs status to paused so it no longer is processed",
			ArgsUsage: "<id> or <nickname>",
			Flags:     specificJobSubCommands,
			Action:    setDataHistoryJobStatus,
		},
		{
			Name:      "unpausejob",
			Usage:     "sets a jobs status to active so it can be processed",
			ArgsUsage: "<id> or <nickname>",
			Flags:     specificJobSubCommands,
			Action:    setDataHistoryJobStatus,
		},
		{
			Name: "updateprerequisitejob",
		},
		{
			Name: "removeprerequisitejob",
		},
	},
}

func getDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	} else {
		id = c.Args().First()
	}
	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	if nickname != "" && id != "" {
		return errors.New("can only set 'id' OR 'nickname'")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Id:       id,
		Nickname: nickname,
	}
	if strings.EqualFold(c.Command.Name, "getjobwithdetailedresults") {
		request.FullDetails = true
	}

	result, err := client.GetDataHistoryJobDetails(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getActiveDataHistoryJobs(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetActiveDataHistoryJobs(context.Background(),
		&gctrpc.GetInfoRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func upsertDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var (
		err                                 error
		nickname, exchange, assetType, pair string
		interval, dataType                  int64
	)
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	} else {
		nickname = c.Args().First()
	}

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().Get(1)
	}
	if !validExchange(exchange) {
		return errInvalidExchange
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(3)
	}
	if !validPair(pair) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(pair, pairDelimiter)
	if err != nil {
		return fmt.Errorf("cannot process pair: %w", err)
	}

	if c.IsSet("start_date") {
		startTime = c.String("start_date")
	}
	if c.IsSet("end_date") {
		endTime = c.String("end_date")
	}

	var s, e time.Time
	s, err = time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.Parse(common.SimpleTimeFormat, endTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if c.IsSet("interval") {
		interval = c.Int64("interval")
	} else {
		interval, err = convert.Int64FromString(c.Args().Get(6))
		if err != nil {
			return fmt.Errorf("cannot process interval: %w", err)
		}
	}
	candleInterval := time.Duration(interval) * time.Second
	if c.IsSet("request_size_limit") {
		requestSizeLimit = c.Uint64("request_size_limit")
	}

	if c.IsSet("data_type") {
		dataType = c.Int64("data_type")
	}

	if c.IsSet("max_retry_attempts") {
		maxRetryAttempts = c.Uint64("max_retry_attempts")
	}

	if c.IsSet("batch_size") {
		batchSize = c.Uint64("batch_size")
	}

	var conversionInterval time.Duration
	var overwriteExistingData bool
	if dataType == 2 || dataType == 3 || dataType == 4 {
		var cInterval int64
		if c.IsSet("conversion_interval") {
			cInterval = c.Int64("conversion_interval")
		} else {
			cInterval, err = convert.Int64FromString(c.Args().Get(11))
			if err != nil {
				return fmt.Errorf("cannot process conversion_interval: %w", err)
			}
		}
		conversionInterval = time.Duration(cInterval) * time.Second

		if c.IsSet("overwrite_existing_data") {
			overwriteExistingData = c.Bool("overwrite_existing_data")
		} else {
			overwriteExistingData, err = strconv.ParseBool(c.Args().Get(12))
			if err != nil {
				return fmt.Errorf("cannot process overwrite_existing_data: %w", err)
			}
		}
	}
	var prerequisiteJobNickname string
	if c.IsSet("prerequisite_job_nickname") {
		assetType = c.String("prerequisite_job_nickname")
	} else {
		prerequisiteJobNickname = c.Args().Get(13)
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.UpsertDataHistoryJobRequest{
		Nickname: nickname,
		Exchange: exchange,
		Asset:    assetType,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		StartDate:        negateLocalOffset(s),
		EndDate:          negateLocalOffset(e),
		Interval:         int64(candleInterval),
		RequestSizeLimit: int64(requestSizeLimit),
		DataType:         dataType,
		MaxRetryAttempts: int64(maxRetryAttempts),
		BatchSize:        int64(batchSize),
		ConversionInterval:    int64(conversionInterval),
		OverwriteExistingData: overwriteExistingData,
		PrerequisiteJobNickname: prerequisiteJobNickname,
	}
	if strings.EqualFold(c.Command.Name, "addnewjob") {
		request.InsertOnly = true
	}

	result, err := client.UpsertDataHistoryJob(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobsBetween(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	if c.IsSet("start_date") {
		startTime = c.String("start_date")
	} else {
		startTime = c.Args().First()
	}
	if c.IsSet("end_date") {
		endTime = c.String("end_date")
	} else {
		endTime = c.Args().Get(1)
	}
	s, err := time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.Parse(common.SimpleTimeFormat, endTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if e.Before(s) {
		return errors.New("start cannot be after end")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetDataHistoryJobsBetween(context.Background(),
		&gctrpc.GetDataHistoryJobsBetweenRequest{
			StartDate: negateLocalOffset(s),
			EndDate:   negateLocalOffset(e),
		})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setDataHistoryJobStatus(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	} else {
		id = c.Args().First()
	}

	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	if nickname != "" && id != "" {
		return errors.New("can only set 'id' OR 'nickname'")
	}

	var status int64
	switch c.Command.Name {
	case "deletejob":
		status = 3
	case "pausejob":
		status = 4
	case "unpausejob":
		status = 5
	default:
		return fmt.Errorf("unrecognised data history job status type")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.SetDataHistoryJobStatusRequest{
		Id:       id,
		Nickname: nickname,
		Status:   status,
	}

	result, err := client.SetDataHistoryJobStatus(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobSummary(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	} else {
		nickname = c.Args().First()
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Nickname: nickname,
	}

	result, err := client.GetDataHistoryJobSummary(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
