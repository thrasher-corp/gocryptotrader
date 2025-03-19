package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
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
					Usage: "formatted as: " + time.DateTime,
				},
				&cli.StringFlag{
					Name:  "end_date",
					Usage: "formatted as: " + time.DateTime,
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
				nicknameFlag,
			},
		},
		{
			Name:      "getjobstatussummary",
			Usage:     "returns a job with human readable summary of its status",
			ArgsUsage: "<nickname>",
			Action:    getDataHistoryJobSummary,
			Flags: []cli.Flag{
				nicknameFlag,
			},
		},
		dataHistoryJobCommands,
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
			Name:      "updateprerequisite",
			Usage:     "adds or updates a prerequisite job to the job referenced - if the job is active, it will be set as 'paused'",
			ArgsUsage: "<prerequisite> <nickname>",
			Flags:     prerequisiteJobSubCommands,
			Action:    setPrerequisiteJob,
		},
		{
			Name:      "removeprerequisite",
			Usage:     "removes a prerequisite job from the job referenced - if the job is 'paused', it will be set as 'active'",
			ArgsUsage: "<nickname>",
			Flags: []cli.Flag{
				nicknameFlag,
			},
			Action: setPrerequisiteJob,
		},
	},
}

var dataHistoryJobCommands = &cli.Command{
	Name:      "addjob",
	Usage:     "add or update data history jobs",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:   "savecandles",
			Usage:  "will fetch candle data from an exchange and save it to the database",
			Flags:  append(baseJobSubCommands, dataHandlingJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "convertcandles",
			Usage:  "convert candles saved to the database to a new resolution eg 1min -> 5min",
			Flags:  append(baseJobSubCommands, candleConvertJobJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "savetrades",
			Usage:  "will fetch trade data from an exchange and save it to the database",
			Flags:  append(baseJobSubCommands, tradeHandlingJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "converttrades",
			Usage:  "convert trades saved to the database to any candle resolution eg 30min",
			Flags:  append(baseJobSubCommands, dataHandlingJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "validatecandles",
			Usage:  "will compare database candle data with API candle data - useful for validating converted trades and candles",
			Flags:  append(baseJobSubCommands, validationJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "secondaryvalidatecandles",
			Usage:  "will compare database candle data with a different exchange's API candle data - ",
			Flags:  append(baseJobSubCommands, secondaryValidationJobSubCommands...),
			Action: upsertDataHistoryJob,
		},
	},
}

var (
	maxRetryAttempts, requestSizeLimit, batchSize, comparisonDecimalPlaces uint64
	guidExample                                                            = "deadbeef-dead-beef-dead-beef13371337"
	overwriteDataFlag                                                      = &cli.BoolFlag{
		Name:  "overwrite_existing_data",
		Usage: "will process and overwrite data if matching data exists at an interval period. if false, will not process or save data",
	}
	comparisonDecimalPlacesFlag = &cli.Uint64Flag{
		Name:        "comparison_decimal_places",
		Usage:       "the number of decimal places used to compare against API data for accuracy",
		Destination: &comparisonDecimalPlaces,
		Value:       3,
	}
	intolerancePercentageFlag = &cli.Float64Flag{
		Name:  "intolerance_percentage",
		Usage: "the number of decimal places used to compare against API data for accuracy",
	}
	requestSize500Flag = &cli.Uint64Flag{
		Name:        "request_size_limit",
		Usage:       "the number of candle intervals to retrieve per request. eg if interval is 1d and request_size_limit is 500, then retrieve 500 intervals per batch",
		Destination: &requestSizeLimit,
		Value:       500,
	}
	requestSize50Flag = &cli.Uint64Flag{
		Name:        "request_size_limit",
		Usage:       "the number of intervals to retrieve per request. eg if interval is 1d and request_size_limit is 50, then retrieve 50 intervals per batch",
		Destination: &requestSizeLimit,
		Value:       50,
	}
	requestSize10Flag = &cli.Uint64Flag{
		Name:        "request_size_limit",
		Usage:       "the number of intervals worth of trades to retrieve per API request. eg if interval is 1m and request_size_limit is 10, then retrieve 10 minutes of trades per batch",
		Destination: &requestSizeLimit,
		Value:       10,
	}
	nicknameFlag = &cli.StringFlag{
		Name:  "nickname",
		Usage: "binance-spot-btc-usdt-2019-trades",
	}
	prerequisiteJobSubCommands = []cli.Flag{
		nicknameFlag,
		&cli.StringFlag{
			Name:  "prerequisite_job_nickname",
			Usage: "binance-spot-btc-usdt-2018-trades",
		},
	}
	specificJobSubCommands = []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: guidExample,
		},
	}
	baseJobSubCommands = []cli.Flag{
		nicknameFlag,
		&cli.StringFlag{
			Name:     "exchange",
			Usage:    "eg binance",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "asset",
			Usage:    "eg spot",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "pair",
			Usage:    "eg btc-usdt",
			Required: true,
		},
		&cli.StringFlag{
			Name:        "start_date",
			Usage:       "formatted as: " + time.DateTime,
			Value:       time.Now().AddDate(-1, 0, 0).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end_date",
			Usage:       "formatted as: " + time.DateTime,
			Value:       time.Now().AddDate(0, -1, 0).Format(time.DateTime),
			Destination: &endTime,
		},
		&cli.Uint64Flag{
			Name:     "interval",
			Usage:    klineMessage,
			Required: true,
		},
		&cli.Uint64Flag{
			Name:        "max_retry_attempts",
			Usage:       "the maximum retry attempts for an interval period before giving up",
			Value:       3,
			Destination: &maxRetryAttempts,
		},
		&cli.Uint64Flag{
			Name:        "batch_size",
			Usage:       "when a job is processed, the number of processing cycles to run. eg a batch size of 3, an interval of 1m and a request_size_limit of 3 will retrieve 3 batches of 3m per cycle",
			Destination: &batchSize,
			Value:       3,
		},
		&cli.StringFlag{
			Name:  "prerequisite_job_nickname",
			Usage: "if present, adds or updates the job to have a prerequisite, will only run when prerequisite job is complete - use command 'removeprerequisite' to remove a prerequisite",
		},
		&cli.BoolFlag{
			Name:  "upsert",
			Usage: "if true, will update an existing job if the nickname is shared. if false, will reject a job if the nickname already exists",
		},
	}
	dataHandlingJobSubCommands = []cli.Flag{
		requestSize500Flag,
		overwriteDataFlag,
	}
	tradeHandlingJobSubCommands = []cli.Flag{
		requestSize10Flag,
		overwriteDataFlag,
	}
	candleConvertJobJobSubCommands = []cli.Flag{
		&cli.Uint64Flag{
			Name:     "conversion_interval",
			Usage:    "the resulting converted candle interval. Can be converted to any interval, however the following " + klineMessage,
			Required: true,
		},
		requestSize500Flag,
		overwriteDataFlag,
	}
	validationJobSubCommands = []cli.Flag{
		requestSize50Flag,
		comparisonDecimalPlacesFlag,
		intolerancePercentageFlag,
		&cli.Uint64Flag{
			Name:  "replace_on_issue",
			Usage: "if true, when the intolerance percentage is exceeded, then the comparison API candle will replace the database candle",
		},
	}
	secondaryValidationJobSubCommands = []cli.Flag{
		&cli.StringFlag{
			Name:  "secondary_exchange",
			Usage: "the exchange to compare candles data to",
		},
		requestSize50Flag,
		comparisonDecimalPlacesFlag,
		intolerancePercentageFlag,
	}
)

func getDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Id:       id,
		Nickname: nickname,
	}
	if strings.EqualFold(c.Command.Name, "getjobwithdetailedresults") {
		request.FullDetails = true
	}

	result, err := client.GetDataHistoryJobDetails(c.Context, request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getActiveDataHistoryJobs(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetActiveDataHistoryJobs(c.Context,
		&gctrpc.GetInfoRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func upsertDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var (
		err                                 error
		nickname, exchange, assetType, pair string
		interval, dataType                  int64
	)
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	}
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		pair = c.String("pair")
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
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if c.IsSet("interval") {
		interval = c.Int64("interval")
	}
	candleInterval := time.Duration(interval) * time.Second
	if c.IsSet("request_size_limit") {
		requestSizeLimit = c.Uint64("request_size_limit")
	}

	if c.IsSet("max_retry_attempts") {
		maxRetryAttempts = c.Uint64("max_retry_attempts")
	}

	if c.IsSet("batch_size") {
		batchSize = c.Uint64("batch_size")
	}
	var upsert bool
	if c.IsSet("upsert") {
		upsert = c.Bool("upsert")
	}

	var secondaryExchange string
	if c.IsSet("secondary_exchange") {
		secondaryExchange = c.String("secondary_exchange")
	}

	var prerequisiteJobNickname string
	if c.IsSet("prerequisite_job_nickname") {
		prerequisiteJobNickname = c.String("prerequisite_job_nickname")
	}

	var intolerancePercentage float64
	if c.IsSet("intolerance_percentage") {
		intolerancePercentage = c.Float64("intolerance_percentage")
	}

	var replaceOnIssue bool
	if c.IsSet("replace_on_issue") {
		replaceOnIssue = c.Bool("replace_on_issue")
	}

	switch c.Command.Name {
	case "savecandles":
		dataType = 0
	case "savetrades":
		dataType = 1
	case "convertcandles":
		dataType = 3
	case "converttrades":
		dataType = 2
	case "validatecandles":
		dataType = 4
	case "secondaryvalidatecandles":
		dataType = 5
	default:
		return errors.New("unrecognised command, cannot set data type")
	}

	var conversionInterval time.Duration
	var overwriteExistingData bool

	switch dataType {
	case 0, 1:
		if c.IsSet("overwrite_existing_data") {
			overwriteExistingData = c.Bool("overwrite_existing_data")
		}
	case 2, 3:
		var cInterval int64
		if c.IsSet("conversion_interval") {
			cInterval = c.Int64("conversion_interval")
		}
		conversionInterval = time.Duration(cInterval) * time.Second
		if c.IsSet("overwrite_existing_data") {
			overwriteExistingData = c.Bool("overwrite_existing_data")
		}
	case 4:
		if c.IsSet("comparison_decimal_places") {
			comparisonDecimalPlaces = c.Uint64("comparison_decimal_places")
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	request := &gctrpc.UpsertDataHistoryJobRequest{
		Nickname: nickname,
		Exchange: exchange,
		Asset:    assetType,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		StartDate:                s.Format(common.SimpleTimeFormatWithTimezone),
		EndDate:                  e.Format(common.SimpleTimeFormatWithTimezone),
		Interval:                 int64(candleInterval),
		RequestSizeLimit:         requestSizeLimit,
		DataType:                 dataType,
		MaxRetryAttempts:         maxRetryAttempts,
		BatchSize:                batchSize,
		ConversionInterval:       int64(conversionInterval),
		OverwriteExistingData:    overwriteExistingData,
		PrerequisiteJobNickname:  prerequisiteJobNickname,
		InsertOnly:               !upsert,
		DecimalPlaceComparison:   comparisonDecimalPlaces,
		SecondaryExchangeName:    secondaryExchange,
		IssueTolerancePercentage: intolerancePercentage,
		ReplaceOnIssue:           replaceOnIssue,
	}

	result, err := client.UpsertDataHistoryJob(c.Context, request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobsBetween(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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
	s, err := time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, endTime, time.Local)
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
	result, err := client.GetDataHistoryJobsBetween(c.Context,
		&gctrpc.GetDataHistoryJobsBetweenRequest{
			StartDate: s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:   e.Format(common.SimpleTimeFormatWithTimezone),
		})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setDataHistoryJobStatus(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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
		status = 5
	case "unpausejob":
		status = 0
	default:
		return fmt.Errorf("unable to modify data history job status, unrecognised command '%v'", c.Command.Name)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	request := &gctrpc.SetDataHistoryJobStatusRequest{
		Id:       id,
		Nickname: nickname,
		Status:   status,
	}

	result, err := client.SetDataHistoryJobStatus(c.Context, request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobSummary(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	} else {
		nickname = c.Args().First()
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Nickname: nickname,
	}

	result, err := client.GetDataHistoryJobSummary(c.Context, request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setPrerequisiteJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	} else {
		nickname = c.Args().First()
	}

	var prerequisite string
	if c.IsSet("prerequisite_job_nickname") {
		prerequisite = c.String("prerequisite_job_nickname")
	} else {
		prerequisite = c.Args().Get(1)
	}

	if c.Command.Name == "updateprerequisite" && prerequisite == "" {
		return errors.New("prerequisite required")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	request := &gctrpc.UpdateDataHistoryJobPrerequisiteRequest{
		PrerequisiteJobNickname: prerequisite,
		Nickname:                nickname,
	}

	result, err := client.UpdateDataHistoryJobPrerequisite(c.Context, request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
