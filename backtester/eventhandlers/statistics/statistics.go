package statistics

import (
	"encoding/json"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Reset returns the struct to defaults
func (s *Statistic) Reset() {
	*s = Statistic{}
}

// SetupEventForTime sets up the big map for to store important data at each time interval
func (s *Statistic) SetupEventForTime(e common.DataEventHandler) error {
	if e == nil {
		return common.ErrNilEvent
	}
	ex := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	s.setupMap(ex, a)
	lookup := s.ExchangeAssetPairStatistics[ex][a][p]
	if lookup == nil {
		lookup = &currencystatistics.CurrencyStatistic{}
	}
	lookup.Events = append(lookup.Events,
		currencystatistics.EventStore{
			DataEvent: e,
		},
	)
	s.ExchangeAssetPairStatistics[ex][a][p] = lookup

	return nil
}

func (s *Statistic) setupMap(ex string, a asset.Item) {
	if s.ExchangeAssetPairStatistics == nil {
		s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	}
	if s.ExchangeAssetPairStatistics[ex] == nil {
		s.ExchangeAssetPairStatistics[ex] = make(map[asset.Item]map[currency.Pair]*currencystatistics.CurrencyStatistic)
	}
	if s.ExchangeAssetPairStatistics[ex][a] == nil {
		s.ExchangeAssetPairStatistics[ex][a] = make(map[currency.Pair]*currencystatistics.CurrencyStatistic)
	}
}

// SetEventForOffset sets the event for the time period in the event
func (s *Statistic) SetEventForOffset(e common.EventHandler) error {
	if e == nil {
		return common.ErrNilEvent
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	exch := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	offset := e.GetOffset()
	lookup := s.ExchangeAssetPairStatistics[exch][a][p]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set signal event", errCurrencyStatisticsUnset, exch, a, p)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].DataEvent.GetOffset() == offset {
			return applyEventAtOffset(e, lookup, i)
		}
	}

	return nil
}

func applyEventAtOffset(e common.EventHandler, lookup *currencystatistics.CurrencyStatistic, i int) error {
	switch t := e.(type) {
	case common.DataEventHandler:
		lookup.Events[i].DataEvent = t
	case signal.Event:
		lookup.Events[i].SignalEvent = t
	case order.Event:
		lookup.Events[i].OrderEvent = t
	case fill.Event:
		lookup.Events[i].FillEvent = t
	default:
		return fmt.Errorf("unknown event type received: %v", e)
	}
	return nil
}

// AddHoldingsForTime adds all holdings to the statistics at the time period
func (s *Statistic) AddHoldingsForTime(h *holdings.Holding) error {
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	lookup := s.ExchangeAssetPairStatistics[h.Exchange][h.Asset][h.Pair]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set holding event", errCurrencyStatisticsUnset, h.Exchange, h.Asset, h.Pair)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].DataEvent.GetOffset() == h.Offset {
			lookup.Events[i].Holdings = *h
			break
		}
	}
	return nil
}

// AddComplianceSnapshotForTime adds the compliance snapshot to the statistics at the time period
func (s *Statistic) AddComplianceSnapshotForTime(c compliance.Snapshot, e fill.Event) error {
	if e == nil {
		return common.ErrNilEvent
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	exch := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	lookup := s.ExchangeAssetPairStatistics[exch][a][p]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set compliance snapshot", errCurrencyStatisticsUnset, exch, a, p)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].DataEvent.GetOffset() == e.GetOffset() {
			lookup.Events[i].Transactions = c
			break
		}
	}

	return nil
}

// CalculateAllResults calculates the statistics of all exchange asset pair holdings,
// orders, ratios and drawdowns
func (s *Statistic) CalculateAllResults() error {
	log.Info(log.BackTester, "calculating backtesting results")
	s.PrintAllEvents()
	currCount := 0
	var finalResults []FinalResultsHolder
	for exchangeName, exchangeMap := range s.ExchangeAssetPairStatistics {
		for assetItem, assetMap := range exchangeMap {
			for pair, stats := range assetMap {
				currCount++
				err := stats.CalculateResults()
				if err != nil {
					return err
				}
				stats.PrintResults(exchangeName, assetItem, pair)
				last := stats.Events[len(stats.Events)-1]
				stats.FinalHoldings = last.Holdings
				stats.FinalOrders = last.Transactions
				s.AllStats = append(s.AllStats, *stats)

				finalResults = append(finalResults, FinalResultsHolder{
					Exchange:         exchangeName,
					Asset:            assetItem,
					Pair:             pair,
					MaxDrawdown:      stats.MaxDrawdown,
					MarketMovement:   stats.MarketMovement,
					StrategyMovement: stats.StrategyMovement,
				})
				s.TotalBuyOrders += stats.BuyOrders
				s.TotalSellOrders += stats.SellOrders
				if stats.ShowMissingDataWarning {
					s.WasAnyDataMissing = true
				}
			}
		}
	}
	s.TotalOrders = s.TotalBuyOrders + s.TotalSellOrders
	if currCount > 1 {
		s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies(finalResults)
		s.BestMarketMovement = s.GetBestMarketPerformer(finalResults)
		s.BestStrategyResults = s.GetBestStrategyPerformer(finalResults)
		s.PrintTotalResults()
	}

	return nil
}

// PrintTotalResults outputs all results to the CMD
func (s *Statistic) PrintTotalResults() {
	log.Info(log.BackTester, "------------------Strategy-----------------------------------")
	log.Infof(log.BackTester, "Strategy Name: %v", s.StrategyName)
	log.Infof(log.BackTester, "Strategy Nickname: %v", s.StrategyNickname)
	log.Infof(log.BackTester, "Strategy Goal: %v\n\n", s.StrategyGoal)
	log.Info(log.BackTester, "------------------Total Results------------------------------")
	log.Info(log.BackTester, "------------------Orders----------------------------------")
	log.Infof(log.BackTester, "Total buy orders: %v", s.TotalBuyOrders)
	log.Infof(log.BackTester, "Total sell orders: %v", s.TotalSellOrders)
	log.Infof(log.BackTester, "Total orders: %v\n\n", s.TotalOrders)

	if s.BiggestDrawdown != nil {
		log.Info(log.BackTester, "------------------Biggest Drawdown------------------------")
		log.Infof(log.BackTester, "Exchange: %v Asset: %v Currency: %v", s.BiggestDrawdown.Exchange, s.BiggestDrawdown.Asset, s.BiggestDrawdown.Pair)
		log.Infof(log.BackTester, "Highest Price: $%.2f", s.BiggestDrawdown.MaxDrawdown.Highest.Price)
		log.Infof(log.BackTester, "Highest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Highest.Time)
		log.Infof(log.BackTester, "Lowest Price: $%.2f", s.BiggestDrawdown.MaxDrawdown.Lowest.Price)
		log.Infof(log.BackTester, "Lowest Price Time: %v", s.BiggestDrawdown.MaxDrawdown.Lowest.Time)
		log.Infof(log.BackTester, "Calculated Drawdown: %.2f%%", s.BiggestDrawdown.MaxDrawdown.DrawdownPercent)
		log.Infof(log.BackTester, "Difference: $%.2f", s.BiggestDrawdown.MaxDrawdown.Highest.Price-s.BiggestDrawdown.MaxDrawdown.Lowest.Price)
		log.Infof(log.BackTester, "Drawdown length: %v\n\n", s.BiggestDrawdown.MaxDrawdown.IntervalDuration)
	}
	if s.BestMarketMovement != nil && s.BestStrategyResults != nil {
		log.Info(log.BackTester, "------------------Orders----------------------------------")
		log.Infof(log.BackTester, "Best performing market movement: %v %v %v %f%%", s.BestMarketMovement.Exchange, s.BestMarketMovement.Asset, s.BestMarketMovement.Pair, s.BestMarketMovement.MarketMovement)
		log.Infof(log.BackTester, "Best performing strategy movement: %v %v %v %f%%\n\n", s.BestStrategyResults.Exchange, s.BestStrategyResults.Asset, s.BestStrategyResults.Pair, s.BestStrategyResults.StrategyMovement)
	}
}

// GetBestMarketPerformer returns the best final market movement
func (s *Statistic) GetBestMarketPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].MarketMovement > result.MarketMovement || result.MarketMovement == 0 {
			result = &results[i]
			break
		}
	}

	return result
}

// GetBestStrategyPerformer returns the best performing strategy result
func (s *Statistic) GetBestStrategyPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].StrategyMovement > result.StrategyMovement || result.StrategyMovement == 0 {
			result = &results[i]
		}
	}

	return result
}

// GetTheBiggestDrawdownAcrossCurrencies returns the biggest drawdown across all currencies in a backtesting run
func (s *Statistic) GetTheBiggestDrawdownAcrossCurrencies(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].MaxDrawdown.DrawdownPercent > result.MaxDrawdown.DrawdownPercent || result.MaxDrawdown.DrawdownPercent == 0 {
			result = &results[i]
		}
	}

	return result
}

// PrintAllEvents outputs all event details in the CMD
func (s *Statistic) PrintAllEvents() {
	log.Info(log.BackTester, "------------------Events-------------------------------------")
	var errs gctcommon.Errors
	for e, x := range s.ExchangeAssetPairStatistics {
		for a, y := range x {
			for p, c := range y {
				for i := range c.Events {
					switch {
					case c.Events[i].FillEvent != nil:
						direction := c.Events[i].FillEvent.GetDirection()
						if direction == common.CouldNotBuy ||
							direction == common.CouldNotSell ||
							direction == common.DoNothing ||
							direction == common.MissingData ||
							direction == "" {
							log.Infof(log.BackTester, "%v | Price: $%f - Direction: %v - Reason: %s",
								c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								c.Events[i].FillEvent.GetClosePrice(),
								c.Events[i].FillEvent.GetDirection(),
								c.Events[i].FillEvent.GetReason())
						} else {
							log.Infof(log.BackTester, "%v | Price: $%f - Amount: %f - Fee: $%f - Total: $%f - Direction %v - Reason: %s",
								c.Events[i].FillEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
								c.Events[i].FillEvent.GetPurchasePrice(),
								c.Events[i].FillEvent.GetAmount(),
								c.Events[i].FillEvent.GetExchangeFee(),
								c.Events[i].FillEvent.GetTotal(),
								c.Events[i].FillEvent.GetDirection(),
								c.Events[i].FillEvent.GetReason(),
							)
						}
					case c.Events[i].SignalEvent != nil:
						log.Infof(log.BackTester, "%v | Price: $%f - Reason: %v",
							c.Events[i].SignalEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
							c.Events[i].SignalEvent.GetPrice(),
							c.Events[i].SignalEvent.GetReason())
					case c.Events[i].DataEvent != nil:
						log.Infof(log.BackTester, "%v | Price: $%f - Reason: %v",
							c.Events[i].DataEvent.GetTime().Format(gctcommon.SimpleTimeFormat),
							c.Events[i].DataEvent.ClosePrice(),
							c.Events[i].DataEvent.GetReason())
					default:
						errs = append(errs, fmt.Errorf("%v %v %v unexpected data received %+v", e, a, p, c.Events[i]))
					}
				}
			}
		}
	}
	if len(errs) > 0 {
		log.Info(log.BackTester, "------------------Errors-------------------------------------")
		for i := range errs {
			log.Info(log.BackTester, errs[i].Error())
		}
	}
}

// SetStrategyName sets the name for statistical identification
func (s *Statistic) SetStrategyName(name string) {
	s.StrategyName = name
}

// Serialise outputs the Statistic struct in json
func (s *Statistic) Serialise() (string, error) {
	resp, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return "", err
	}

	return string(resp), nil
}
