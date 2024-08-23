package statistics

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	data2 "github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	limit12 = 12
	limit14 = 14
	limit10 = 10
)

// addReason basic helper to append event reason if one is there
func addReason(reason, msg string) string {
	if reason != "" {
		msg += "\tReason: " + reason
	}
	return msg
}

// PrintTotalResults outputs all results to the CMD
func (s *Statistic) PrintTotalResults() {
	log.Infoln(common.Statistics, common.CMDColours.H1+"------------------Strategy-----------------------------------"+common.CMDColours.Default)
	log.Infof(common.Statistics, "Strategy Name: %v", s.StrategyName)
	log.Infof(common.Statistics, "Strategy Nickname: %v", s.StrategyNickname)
	log.Infof(common.Statistics, "Strategy Goal: %v\n\n", s.StrategyGoal)

	log.Infoln(common.Statistics, common.CMDColours.H2+"------------------Total Results------------------------------"+common.CMDColours.Default)
	log.Infoln(common.Statistics, common.CMDColours.H3+"------------------Orders-------------------------------------"+common.CMDColours.Default)
	log.Infof(common.Statistics, "Total buy orders: %v", convert.IntToHumanFriendlyString(s.TotalBuyOrders, ","))
	log.Infof(common.Statistics, "Total sell orders: %v", convert.IntToHumanFriendlyString(s.TotalSellOrders, ","))
	log.Infof(common.Statistics, "Total long orders: %v", convert.IntToHumanFriendlyString(s.TotalLongOrders, ","))
	log.Infof(common.Statistics, "Total short orders: %v", convert.IntToHumanFriendlyString(s.TotalShortOrders, ","))
	log.Infof(common.Statistics, "Total orders: %v\n\n", convert.IntToHumanFriendlyString(s.TotalOrders, ","))

	if s.BiggestDrawdown != nil {
		log.Infoln(common.Statistics, common.CMDColours.H3+"------------------Biggest Drawdown-----------------------"+common.CMDColours.Default)
		log.Infof(common.Statistics, "Exchange: %v Asset: %v Currency: %v", s.BiggestDrawdown.Exchange, s.BiggestDrawdown.Asset, s.BiggestDrawdown.Pair)
		log.Infof(common.Statistics, "Highest Price: %s", convert.DecimalToHumanFriendlyString(s.BiggestDrawdown.MaxDrawdown.Highest.Value, 8, ".", ","))
		log.Infof(common.Statistics, "Highest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Highest.Time)
		log.Infof(common.Statistics, "Lowest Price: %s", convert.DecimalToHumanFriendlyString(s.BiggestDrawdown.MaxDrawdown.Lowest.Value, 8, ".", ","))
		log.Infof(common.Statistics, "Lowest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Lowest.Time)
		log.Infof(common.Statistics, "Calculated Drawdown: %s%%", convert.DecimalToHumanFriendlyString(s.BiggestDrawdown.MaxDrawdown.DrawdownPercent, 2, ".", ","))
		log.Infof(common.Statistics, "Difference: %s", convert.DecimalToHumanFriendlyString(s.BiggestDrawdown.MaxDrawdown.Highest.Value.Sub(s.BiggestDrawdown.MaxDrawdown.Lowest.Value), 8, ".", ","))
		log.Infof(common.Statistics, "Drawdown length: %v candles\n\n", convert.IntToHumanFriendlyString(s.BiggestDrawdown.MaxDrawdown.IntervalDuration, ","))
	}
	if s.BestMarketMovement != nil && s.BestStrategyResults != nil {
		log.Infoln(common.Statistics, common.CMDColours.H4+"------------------Orders----------------------------------"+common.CMDColours.Default)
		log.Infof(common.Statistics, "Best performing market movement: %v %v %v %v%%", s.BestMarketMovement.Exchange, s.BestMarketMovement.Asset, s.BestMarketMovement.Pair, convert.DecimalToHumanFriendlyString(s.BestMarketMovement.MarketMovement, 2, ".", ","))
		log.Infof(common.Statistics, "Best performing strategy movement: %v %v %v %v%%\n\n", s.BestStrategyResults.Exchange, s.BestStrategyResults.Asset, s.BestStrategyResults.Pair, convert.DecimalToHumanFriendlyString(s.BestStrategyResults.StrategyMovement, 2, ".", ","))
	}
}

// PrintAllEventsChronologically outputs all event details in the CMD
// rather than separated by exchange, asset and currency pair, it's
// grouped by time to allow a clearer picture of events
func (s *Statistic) PrintAllEventsChronologically() {
	log.Infoln(common.Statistics, common.CMDColours.H1+"------------------Events-------------------------------------"+common.CMDColours.Default)
	var errs error
	var results []eventOutputHolder
	for _, currencyStatistic := range s.ExchangeAssetPairStatistics {
		for i := range currencyStatistic.Events {
			var result string
			var tt time.Time
			var err error
			switch {
			case currencyStatistic.Events[i].FillEvent != nil:
				result, err = s.CreateLog(currencyStatistic.Events[i].FillEvent)
				if err != nil {
					errs = gctcommon.AppendError(errs, err)
					continue
				}
				tt = currencyStatistic.Events[i].FillEvent.GetTime()
			case currencyStatistic.Events[i].SignalEvent != nil:
				result, err = s.CreateLog(currencyStatistic.Events[i].SignalEvent)
				if err != nil {
					errs = gctcommon.AppendError(errs, err)
					continue
				}
				tt = currencyStatistic.Events[i].SignalEvent.GetTime()
			case currencyStatistic.Events[i].DataEvent != nil:
				result, err = s.CreateLog(currencyStatistic.Events[i].DataEvent)
				if err != nil {
					errs = gctcommon.AppendError(errs, err)
					continue
				}
				tt = currencyStatistic.Events[i].DataEvent.GetTime()
			}
			results = addEventOutputToTime(results, tt, result)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		b1 := results[i]
		b2 := results[j]
		return b1.Time.Before(b2.Time)
	})
	for i := range results {
		for j := range results[i].Events {
			log.Infoln(common.Statistics, results[i].Events[j])
		}
	}
	if errs != nil {
		log.Infoln(common.Statistics, common.CMDColours.Error+"------------------Errors-------------------------------------"+common.CMDColours.Default)
		for err := errors.Unwrap(errs); err != nil; err = errors.Unwrap(errs) {
			log.Errorln(common.Statistics, err.Error())
		}
	}
}

// CreateLog renders a string log depending on what events are populated
// at a given offset. Can render logs live, or at the end of a backtesting run
func (s *Statistic) CreateLog(data common.Event) (string, error) {
	var (
		result string
		colour = common.CMDColours.Default
	)
	switch ev := data.(type) {
	case fill.Event:
		direction := ev.GetDirection()
		if direction == order.CouldNotBuy ||
			direction == order.CouldNotSell ||
			direction == order.CouldNotLong ||
			direction == order.CouldNotShort ||
			direction == order.MissingData ||
			direction == order.DoNothing ||
			direction == order.TransferredFunds ||
			direction == order.UnknownSide {
			if direction == order.DoNothing {
				colour = common.CMDColours.DarkGrey
			}
			result = fmt.Sprintf(colour+
				"%v %v%v%v| Price: %v\tDirection: %v",
				ev.GetTime().Format(time.DateTime),
				fSIL(ev.GetExchange(), limit12),
				fSIL(ev.GetAssetType().String(), limit10),
				fSIL(ev.Pair().String(), limit14),
				ev.GetClosePrice().Round(8),
				ev.GetDirection())
			result = addReason(ev.GetConcatReasons(), result)
			result += common.CMDColours.Default
		} else {
			// successful order!
			colour = common.CMDColours.Success
			if ev.IsLiquidated() {
				colour = common.CMDColours.Error
			}
			result = fmt.Sprintf(colour+
				"%v %v%v%v| Price: %v\tDirection %v\tOrder placed: Amount: %v\tFee: %v\tTotal: %v",
				ev.GetTime().Format(time.DateTime),
				fSIL(ev.GetExchange(), limit12),
				fSIL(ev.GetAssetType().String(), limit10),
				fSIL(ev.Pair().String(), limit14),
				ev.GetPurchasePrice().Round(8),
				ev.GetDirection(),
				ev.GetAmount().Round(8),
				ev.GetExchangeFee(),
				ev.GetTotal().Round(8))
			result = addReason(ev.GetConcatReasons(), result)
			result += common.CMDColours.Default
		}
	case signal.Event:
		result = fmt.Sprintf("%v %v%v%v| Price: $%v",
			ev.GetTime().Format(time.DateTime),
			fSIL(ev.GetExchange(), limit12),
			fSIL(ev.GetAssetType().String(), limit10),
			fSIL(ev.Pair().String(), limit14),
			ev.GetClosePrice().Round(8))
		result = addReason(ev.GetConcatReasons(), result)
		result += common.CMDColours.Default
	case data2.Event:
		result = fmt.Sprintf("%v %v%v%v| Price: $%v",
			ev.GetTime().Format(time.DateTime),
			fSIL(ev.GetExchange(), limit12),
			fSIL(ev.GetAssetType().String(), limit10),
			fSIL(ev.Pair().String(), limit14),
			ev.GetClosePrice().Round(8))
		result = addReason(ev.GetConcatReasons(), result)
		result += common.CMDColours.Default
	default:
		return "", fmt.Errorf(common.CMDColours.Error+"unexpected data received %T %+v"+common.CMDColours.Default, data, data)
	}
	return result, nil
}

// PrintResults outputs all calculated statistics to the command line
func (c *CurrencyPairStatistic) PrintResults(e string, a asset.Item, p currency.Pair, usingExchangeLevelFunding bool) error {
	if len(c.Events) == 0 {
		return errCurrencyStatisticsUnset
	}
	sort.Slice(c.Events, func(i, j int) bool {
		return c.Events[i].Time.Before(c.Events[j].Time)
	})
	last := c.Events[len(c.Events)-1]
	first := c.Events[0]
	if first.DataEvent == nil {
		return errNoDataAtOffset
	}
	c.StartingClosePrice.Value = first.DataEvent.GetClosePrice()
	c.StartingClosePrice.Time = first.Time
	c.EndingClosePrice.Value = last.DataEvent.GetClosePrice()
	c.EndingClosePrice.Time = last.Time
	c.TotalOrders = c.BuyOrders + c.SellOrders
	last.Holdings.TotalValueLost = last.Holdings.TotalValueLostToSlippage.Add(last.Holdings.TotalValueLostToVolumeSizing)
	sep := fmt.Sprintf("%v %v %v |\t", fSIL(e, limit12), fSIL(a.String(), limit10), fSIL(p.String(), limit14))
	currStr := fmt.Sprintf(common.CMDColours.H1+"------------------Stats for %v %v %v------------------------------------------------------"+common.CMDColours.Default, e, a, p)
	log.Infoln(common.CurrencyStatistics, currStr[:70])
	if a.IsFutures() {
		log.Infof(common.CurrencyStatistics, "%s Long orders: %s", sep, convert.IntToHumanFriendlyString(c.BuyOrders, ","))
		log.Infof(common.CurrencyStatistics, "%s Short orders: %s", sep, convert.IntToHumanFriendlyString(c.SellOrders, ","))
		log.Infof(common.CurrencyStatistics, "%s Highest Unrealised PNL: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestUnrealisedPNL.Value, 8, ".", ","), c.HighestUnrealisedPNL.Time)
		log.Infof(common.CurrencyStatistics, "%s Lowest Unrealised PNL: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.LowestUnrealisedPNL.Value, 8, ".", ","), c.LowestUnrealisedPNL.Time)
		log.Infof(common.CurrencyStatistics, "%s Highest Realised PNL: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestRealisedPNL.Value, 8, ".", ","), c.HighestRealisedPNL.Time)
		log.Infof(common.CurrencyStatistics, "%s Lowest Realised PNL: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.LowestRealisedPNL.Value, 8, ".", ","), c.LowestRealisedPNL.Time)
		log.Infof(common.CurrencyStatistics, "%s Highest committed funds: %s %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestCommittedFunds.Value, 8, ".", ","), c.UnderlyingPair.Quote, c.HighestCommittedFunds.Time)
	} else {
		log.Infof(common.CurrencyStatistics, "%s Buy orders: %s", sep, convert.IntToHumanFriendlyString(c.BuyOrders, ","))
		log.Infof(common.CurrencyStatistics, "%s Buy amount: %s %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.BoughtAmount, 8, ".", ","), last.Holdings.Pair.Base)
		log.Infof(common.CurrencyStatistics, "%s Sell orders: %s", sep, convert.IntToHumanFriendlyString(c.SellOrders, ","))
		log.Infof(common.CurrencyStatistics, "%s Sell amount: %s %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.SoldAmount, 8, ".", ","), last.Holdings.Pair.Base)
		log.Infof(common.CurrencyStatistics, "%s Highest committed funds: %s %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestCommittedFunds.Value, 8, ".", ","), last.Holdings.Pair.Quote, c.HighestCommittedFunds.Time)
	}

	log.Infof(common.CurrencyStatistics, "%s Total orders: %s", sep, convert.IntToHumanFriendlyString(c.TotalOrders, ","))

	log.Infoln(common.CurrencyStatistics, common.CMDColours.H2+"------------------Max Drawdown-------------------------------"+common.CMDColours.Default)
	log.Infof(common.CurrencyStatistics, "%s Highest Price of drawdown: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Highest.Value, 8, ".", ","), c.MaxDrawdown.Highest.Time)
	log.Infof(common.CurrencyStatistics, "%s Lowest Price of drawdown: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Lowest.Value, 8, ".", ","), c.MaxDrawdown.Lowest.Time)
	log.Infof(common.CurrencyStatistics, "%s Calculated Drawdown: %s%%", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.DrawdownPercent, 8, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Difference: %s", sep, convert.DecimalToHumanFriendlyString(c.MaxDrawdown.Highest.Value.Sub(c.MaxDrawdown.Lowest.Value), 2, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Drawdown length: %s", sep, convert.IntToHumanFriendlyString(c.MaxDrawdown.IntervalDuration, ","))
	if !usingExchangeLevelFunding && c.TotalOrders > 1 {
		log.Infoln(common.CurrencyStatistics, common.CMDColours.H2+"------------------Ratios------------------------------------------------"+common.CMDColours.Default)
		log.Infoln(common.CurrencyStatistics, common.CMDColours.H3+"------------------Rates-------------------------------------------------"+common.CMDColours.Default)
		log.Infof(common.CurrencyStatistics, "%s Compound Annual Growth Rate: %s", sep, convert.DecimalToHumanFriendlyString(c.CompoundAnnualGrowthRate, 2, ".", ","))
		log.Infoln(common.CurrencyStatistics, common.CMDColours.H4+"------------------Arithmetic--------------------------------------------"+common.CMDColours.Default)
		if c.ShowMissingDataWarning {
			log.Infoln(common.CurrencyStatistics, "Missing data was detected during this backtesting run")
			log.Infoln(common.CurrencyStatistics, "Ratio calculations will be skewed")
		}
		log.Infof(common.CurrencyStatistics, "%s Sharpe ratio: %v", sep, c.ArithmeticRatios.SharpeRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Sortino ratio: %v", sep, c.ArithmeticRatios.SortinoRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Information ratio: %v", sep, c.ArithmeticRatios.InformationRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Calmar ratio: %v", sep, c.ArithmeticRatios.CalmarRatio.Round(4))

		log.Infoln(common.CurrencyStatistics, common.CMDColours.H4+"------------------Geometric--------------------------------------------"+common.CMDColours.Default)
		if c.ShowMissingDataWarning {
			log.Infoln(common.CurrencyStatistics, "Missing data was detected during this backtesting run")
			log.Infoln(common.CurrencyStatistics, "Ratio calculations will be skewed")
		}
		log.Infof(common.CurrencyStatistics, "%s Sharpe ratio: %v", sep, c.GeometricRatios.SharpeRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Sortino ratio: %v", sep, c.GeometricRatios.SortinoRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Information ratio: %v", sep, c.GeometricRatios.InformationRatio.Round(4))
		log.Infof(common.CurrencyStatistics, "%s Calmar ratio: %v", sep, c.GeometricRatios.CalmarRatio.Round(4))
	}

	log.Infoln(common.CurrencyStatistics, common.CMDColours.H2+"------------------Results------------------------------------"+common.CMDColours.Default)
	log.Infof(common.CurrencyStatistics, "%s Starting Close Price: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.StartingClosePrice.Value, 8, ".", ","), c.StartingClosePrice.Time)
	log.Infof(common.CurrencyStatistics, "%s Finishing Close Price: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.EndingClosePrice.Value, 8, ".", ","), c.EndingClosePrice.Time)
	log.Infof(common.CurrencyStatistics, "%s Lowest Close Price: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.LowestClosePrice.Value, 8, ".", ","), c.LowestClosePrice.Time)
	log.Infof(common.CurrencyStatistics, "%s Highest Close Price: %s at %v", sep, convert.DecimalToHumanFriendlyString(c.HighestClosePrice.Value, 8, ".", ","), c.HighestClosePrice.Time)

	log.Infof(common.CurrencyStatistics, "%s Market movement: %s%%", sep, convert.DecimalToHumanFriendlyString(c.MarketMovement, 2, ".", ","))
	if !usingExchangeLevelFunding {
		log.Infof(common.CurrencyStatistics, "%s Strategy movement: %s%%", sep, convert.DecimalToHumanFriendlyString(c.StrategyMovement, 2, ".", ","))
		log.Infof(common.CurrencyStatistics, "%s Did it beat the market: %v", sep, c.StrategyMovement.GreaterThan(c.MarketMovement))
	}

	log.Infof(common.CurrencyStatistics, "%s Value lost to volume sizing: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLostToVolumeSizing, 2, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Value lost to slippage: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLostToSlippage, 2, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Total Value lost: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalValueLost, 2, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Total Fees: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalFees, 8, ".", ","))
	log.Infof(common.CurrencyStatistics, "%s Final holdings value: %s", sep, convert.DecimalToHumanFriendlyString(c.TotalAssetValue, 8, ".", ","))
	if !usingExchangeLevelFunding {
		// the following have no direct translation to individual exchange level funds as they
		// combine base and quote values
		log.Infof(common.CurrencyStatistics, "%s Final funds: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.QuoteSize, 8, ".", ","))
		log.Infof(common.CurrencyStatistics, "%s Final holdings: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.BaseSize, 8, ".", ","))
		log.Infof(common.CurrencyStatistics, "%s Final total value: %s", sep, convert.DecimalToHumanFriendlyString(last.Holdings.TotalValue, 8, ".", ","))
	}

	if last.PNL != nil {
		var unrealised, realised portfolio.BasicPNLResult
		unrealised = last.PNL.GetUnrealisedPNL()
		realised = last.PNL.GetRealisedPNL()
		log.Infof(common.CurrencyStatistics, "%s Final Unrealised PNL: %s", sep, convert.DecimalToHumanFriendlyString(unrealised.PNL, 8, ".", ","))
		log.Infof(common.CurrencyStatistics, "%s Final Realised PNL: %s", sep, convert.DecimalToHumanFriendlyString(realised.PNL, 8, ".", ","))
	}
	return nil
}

// PrintResults outputs all calculated funding statistics to the command line
func (f *FundingStatistics) PrintResults(wasAnyDataMissing bool) error {
	if f.Report == nil {
		return fmt.Errorf("%w requires report to be generated", gctcommon.ErrNilPointer)
	}
	var spotResults, futuresResults []FundingItemStatistics
	for i := range f.Items {
		if f.Items[i].ReportItem.Asset.IsFutures() {
			futuresResults = append(futuresResults, f.Items[i])
		} else {
			spotResults = append(spotResults, f.Items[i])
		}
	}
	if len(spotResults) > 0 || len(futuresResults) > 0 {
		log.Infoln(common.FundingStatistics, common.CMDColours.H1+"------------------Funding------------------------------------"+common.CMDColours.Default)
	}
	if len(spotResults) > 0 {
		log.Infoln(common.FundingStatistics, common.CMDColours.H2+"------------------Funding Spot Item Results------------------"+common.CMDColours.Default)
		for i := range spotResults {
			if spotResults[i].ReportItem.AppendedViaAPI {
				continue
			}
			sep := fmt.Sprintf("%v%v%v| ", fSIL(spotResults[i].ReportItem.Exchange, limit12), fSIL(spotResults[i].ReportItem.Asset.String(), limit10), fSIL(spotResults[i].ReportItem.Currency.String(), limit14))
			if !spotResults[i].ReportItem.PairedWith.IsEmpty() {
				log.Infof(common.FundingStatistics, "%s Paired with: %v", sep, spotResults[i].ReportItem.PairedWith)
			}
			log.Infof(common.FundingStatistics, "%s Initial funds: %s", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.InitialFunds, 8, ".", ","))
			log.Infof(common.FundingStatistics, "%s Final funds: %s", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.FinalFunds, 8, ".", ","))

			if !f.Report.DisableUSDTracking && f.Report.UsingExchangeLevelFunding {
				log.Infof(common.FundingStatistics, "%s Initial funds in USD: $%s", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.USDInitialFunds, 2, ".", ","))
				log.Infof(common.FundingStatistics, "%s Final funds in USD: $%s", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.USDFinalFunds, 2, ".", ","))
			}
			if spotResults[i].ReportItem.ShowInfinite {
				log.Infof(common.FundingStatistics, "%s Difference: ∞%%", sep)
			} else {
				log.Infof(common.FundingStatistics, "%s Difference: %s%%", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.Difference, 8, ".", ","))
			}
			if spotResults[i].ReportItem.TransferFee.GreaterThan(decimal.Zero) {
				log.Infof(common.FundingStatistics, "%s Transfer fee: %s", sep, convert.DecimalToHumanFriendlyString(spotResults[i].ReportItem.TransferFee, 8, ".", ","))
			}
			if i != len(spotResults)-1 {
				log.Infoln(common.FundingStatistics, "")
			}
		}
	}
	if len(futuresResults) > 0 {
		log.Infoln(common.FundingStatistics, common.CMDColours.H2+"------------------Funding Futures Item Results---------------"+common.CMDColours.Default)
		for i := range futuresResults {
			if futuresResults[i].ReportItem.AppendedViaAPI {
				continue
			}
			sep := fmt.Sprintf("%v%v%v| ", fSIL(futuresResults[i].ReportItem.Exchange, limit12), fSIL(futuresResults[i].ReportItem.Asset.String(), limit10), fSIL(futuresResults[i].ReportItem.Currency.String(), limit14))
			log.Infof(common.FundingStatistics, "%s Is Collateral: %v", sep, futuresResults[i].IsCollateral)
			if futuresResults[i].IsCollateral {
				log.Infof(common.FundingStatistics, "%s Initial Collateral: %v %v at %v", sep, futuresResults[i].InitialCollateral.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].InitialCollateral.Time)
				log.Infof(common.FundingStatistics, "%s Final Collateral: %v %v at %v", sep, futuresResults[i].FinalCollateral.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].FinalCollateral.Time)
				log.Infof(common.FundingStatistics, "%s Lowest Collateral: %v %v at %v", sep, futuresResults[i].LowestCollateral.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].LowestCollateral.Time)
				log.Infof(common.FundingStatistics, "%s Highest Collateral: %v %v at %v", sep, futuresResults[i].HighestCollateral.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].HighestCollateral.Time)
			} else {
				if !futuresResults[i].ReportItem.PairedWith.IsEmpty() {
					log.Infof(common.FundingStatistics, "%s Collateral currency: %v", sep, futuresResults[i].ReportItem.PairedWith)
				}
				log.Infof(common.FundingStatistics, "%s Lowest Contract Holdings: %v %v at %v", sep, futuresResults[i].LowestHoldings.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].LowestHoldings.Time)
				log.Infof(common.FundingStatistics, "%s Highest Contract Holdings: %v %v at %v", sep, futuresResults[i].HighestHoldings.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].HighestHoldings.Time)
				log.Infof(common.FundingStatistics, "%s Initial Contract Holdings: %v %v at %v", sep, futuresResults[i].InitialHoldings.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].InitialHoldings.Time)
				log.Infof(common.FundingStatistics, "%s Final Contract Holdings: %v %v at %v", sep, futuresResults[i].FinalHoldings.Value, futuresResults[i].ReportItem.Currency, futuresResults[i].FinalHoldings.Time)
			}
			if i != len(futuresResults)-1 {
				log.Infoln(common.FundingStatistics, "")
			}
		}
	}
	if f.Report.DisableUSDTracking {
		return nil
	}
	log.Infoln(common.FundingStatistics, common.CMDColours.H2+"------------------USD Tracking Totals------------------------"+common.CMDColours.Default)
	sep := "USD Tracking Total |\t"

	log.Infof(common.FundingStatistics, "%s Initial value: $%s", sep, convert.DecimalToHumanFriendlyString(f.Report.InitialFunds, 8, ".", ","))
	log.Infof(common.FundingStatistics, "%s Final value: $%s", sep, convert.DecimalToHumanFriendlyString(f.Report.FinalFunds, 8, ".", ","))
	log.Infof(common.FundingStatistics, "%s Benchmark Market Movement: %s%%", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.BenchmarkMarketMovement, 8, ".", ","))
	log.Infof(common.FundingStatistics, "%s Strategy Movement: %s%%", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.HoldingValueDifference, 8, ".", ","))
	log.Infof(common.FundingStatistics, "%s Did strategy make a profit: %v", sep, f.TotalUSDStatistics.DidStrategyMakeProfit)
	log.Infof(common.FundingStatistics, "%s Did strategy beat the benchmark: %v", sep, f.TotalUSDStatistics.DidStrategyBeatTheMarket)
	log.Infof(common.FundingStatistics, "%s Highest funds: $%s at %v", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.HighestHoldingValue.Value, 8, ".", ","), f.TotalUSDStatistics.HighestHoldingValue.Time)
	log.Infof(common.FundingStatistics, "%s Lowest funds: $%s at %v", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.LowestHoldingValue.Value, 8, ".", ","), f.TotalUSDStatistics.LowestHoldingValue.Time)

	log.Infoln(common.FundingStatistics, common.CMDColours.H3+"------------------Ratios------------------------------------------------"+common.CMDColours.Default)
	log.Infoln(common.FundingStatistics, common.CMDColours.H4+"------------------Rates-------------------------------------------------"+common.CMDColours.Default)
	log.Infof(common.FundingStatistics, "%s Risk free rate: %s%%", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.RiskFreeRate.Mul(decimal.NewFromInt(100)), 2, ".", ","))
	log.Infof(common.FundingStatistics, "%s Compound Annual Growth Rate: %v%%", sep, convert.DecimalToHumanFriendlyString(f.TotalUSDStatistics.CompoundAnnualGrowthRate, 8, ".", ","))
	if f.TotalUSDStatistics.ArithmeticRatios == nil || f.TotalUSDStatistics.GeometricRatios == nil {
		return fmt.Errorf("%w missing ratio calculations", gctcommon.ErrNilPointer)
	}
	log.Infoln(common.FundingStatistics, common.CMDColours.H4+"------------------Arithmetic--------------------------------------------"+common.CMDColours.Default)
	if wasAnyDataMissing {
		log.Infoln(common.FundingStatistics, "Missing data was detected during this backtesting run")
		log.Infoln(common.FundingStatistics, "Ratio calculations will be skewed")
	}
	log.Infof(common.FundingStatistics, "%s Sharpe ratio: %v", sep, f.TotalUSDStatistics.ArithmeticRatios.SharpeRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Sortino ratio: %v", sep, f.TotalUSDStatistics.ArithmeticRatios.SortinoRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Information ratio: %v", sep, f.TotalUSDStatistics.ArithmeticRatios.InformationRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Calmar ratio: %v", sep, f.TotalUSDStatistics.ArithmeticRatios.CalmarRatio.Round(4))

	log.Infoln(common.FundingStatistics, common.CMDColours.H4+"------------------Geometric--------------------------------------------"+common.CMDColours.Default)
	if wasAnyDataMissing {
		log.Infoln(common.FundingStatistics, "Missing data was detected during this backtesting run")
		log.Infoln(common.FundingStatistics, "Ratio calculations will be skewed")
	}
	log.Infof(common.FundingStatistics, "%s Sharpe ratio: %v", sep, f.TotalUSDStatistics.GeometricRatios.SharpeRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Sortino ratio: %v", sep, f.TotalUSDStatistics.GeometricRatios.SortinoRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Information ratio: %v", sep, f.TotalUSDStatistics.GeometricRatios.InformationRatio.Round(4))
	log.Infof(common.FundingStatistics, "%s Calmar ratio: %v\n\n", sep, f.TotalUSDStatistics.GeometricRatios.CalmarRatio.Round(4))

	return nil
}
