package statistics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Reset returns the struct to defaults
func (s *Statistic) Reset() error {
	if s == nil {
		return gctcommon.ErrNilPointer
	}
	s.StrategyName = ""
	s.StrategyDescription = ""
	s.StrategyNickname = ""
	s.StrategyGoal = ""
	s.StartDate = time.Time{}
	s.EndDate = time.Time{}
	s.CandleInterval = 0
	s.RiskFreeRate = decimal.Zero
	s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	s.CurrencyStatistics = nil
	s.TotalBuyOrders = 0
	s.TotalLongOrders = 0
	s.TotalShortOrders = 0
	s.TotalSellOrders = 0
	s.TotalOrders = 0
	s.BiggestDrawdown = nil
	s.BestStrategyResults = nil
	s.BestMarketMovement = nil
	s.WasAnyDataMissing = false
	s.FundingStatistics = nil
	s.FundManager = nil
	s.HasCollateral = false
	return nil
}

// SetEventForOffset sets up the big map for to store important data at each time interval
func (s *Statistic) SetEventForOffset(ev common.Event) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if ev.GetBase() == nil {
		return fmt.Errorf("%w event base", common.ErrNilEvent)
	}
	ex := ev.GetExchange()
	a := ev.GetAssetType()
	p := ev.Pair()
	if s.ExchangeAssetPairStatistics == nil {
		s.ExchangeAssetPairStatistics = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
	}
	m, ok := s.ExchangeAssetPairStatistics[ex]
	if !ok {
		m = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
		s.ExchangeAssetPairStatistics[ex] = m
	}
	m2, ok := m[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*CurrencyPairStatistic)
		m[a] = m2
	}
	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*CurrencyPairStatistic)
		m2[p.Base.Item] = m3
	}
	lookup, ok := m3[p.Quote.Item]
	if !ok {
		lookup = &CurrencyPairStatistic{
			Exchange:       ev.GetExchange(),
			Asset:          ev.GetAssetType(),
			Currency:       ev.Pair(),
			UnderlyingPair: ev.GetUnderlyingPair(),
		}
		m3[p.Quote.Item] = lookup
	}
	for i := range lookup.Events {
		if lookup.Events[i].Offset != ev.GetOffset() {
			continue
		}
		return applyEventAtOffset(ev, &lookup.Events[i])
	}

	// add to events and then apply the supplied event to it
	lookup.Events = append(lookup.Events, DataAtOffset{
		Offset: ev.GetOffset(),
		Time:   ev.GetTime(),
	})
	err := applyEventAtOffset(ev, &lookup.Events[len(lookup.Events)-1])
	if err != nil {
		return err
	}

	return nil
}

func applyEventAtOffset(ev common.Event, data *DataAtOffset) error {
	switch t := ev.(type) {
	case kline.Event:
		// using kline.Event as signal.Event also matches data.Event
		if data.DataEvent != nil && data.DataEvent != ev {
			return fmt.Errorf("kline event %w %v %v %v %v", ErrAlreadyProcessed, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetOffset())
		}
		data.DataEvent = t
	case signal.Event:
		if data.SignalEvent != nil {
			return fmt.Errorf("signal event %w %v %v %v %v", ErrAlreadyProcessed, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetOffset())
		}
		data.SignalEvent = t
	case order.Event:
		if data.OrderEvent != nil {
			return fmt.Errorf("order event %w %v %v %v %v", ErrAlreadyProcessed, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetOffset())
		}
		data.OrderEvent = t
	case fill.Event:
		if data.FillEvent != nil {
			return fmt.Errorf("fill event %w %v %v %v %v", ErrAlreadyProcessed, ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetOffset())
		}
		data.FillEvent = t
	default:
		return fmt.Errorf("unknown event type received: %v", ev)
	}
	data.Time = ev.GetTime()
	data.ClosePrice = ev.GetClosePrice()

	return nil
}

// AddHoldingsForTime adds all holdings to the statistics at the time period
func (s *Statistic) AddHoldingsForTime(h *holdings.Holding) error {
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	lookup := s.ExchangeAssetPairStatistics[h.Exchange][h.Asset][h.Pair.Base.Item][h.Pair.Quote.Item]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set holding event", errCurrencyStatisticsUnset, h.Exchange, h.Asset, h.Pair)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].Offset == h.Offset {
			lookup.Events[i].Holdings = *h
			return nil
		}
	}
	return fmt.Errorf("%v %v %v %w %v", h.Exchange, h.Asset, h.Pair, errNoDataAtOffset, h.Offset)
}

// AddPNLForTime stores PNL data for tracking purposes
func (s *Statistic) AddPNLForTime(pnl *portfolio.PNLSummary) error {
	if pnl == nil {
		return fmt.Errorf("%w requires PNL", gctcommon.ErrNilPointer)
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	lookup := s.ExchangeAssetPairStatistics[pnl.Exchange][pnl.Asset][pnl.Pair.Base.Item][pnl.Pair.Quote.Item]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set pnl", errCurrencyStatisticsUnset, pnl.Exchange, pnl.Asset, pnl.Pair)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].Offset == pnl.Offset {
			lookup.Events[i].PNL = pnl
			lookup.Events[i].Holdings.BaseSize = pnl.Result.Exposure
			return nil
		}
	}
	return fmt.Errorf("%v %v %v %w %v", pnl.Exchange, pnl.Asset, pnl.Pair, errNoDataAtOffset, pnl.Offset)
}

// AddComplianceSnapshotForTime adds the compliance snapshot to the statistics at the time period
func (s *Statistic) AddComplianceSnapshotForTime(c *compliance.Snapshot, e common.Event) error {
	if c == nil {
		return fmt.Errorf("%w compliance snapshot", common.ErrNilEvent)
	}
	if e == nil {
		return fmt.Errorf("%w fill event", common.ErrNilEvent)
	}
	if s.ExchangeAssetPairStatistics == nil {
		return errExchangeAssetPairStatsUnset
	}
	exch := e.GetExchange()
	a := e.GetAssetType()
	p := e.Pair()
	lookup := s.ExchangeAssetPairStatistics[exch][a][p.Base.Item][p.Quote.Item]
	if lookup == nil {
		return fmt.Errorf("%w for %v %v %v to set compliance snapshot", errCurrencyStatisticsUnset, exch, a, p)
	}
	for i := len(lookup.Events) - 1; i >= 0; i-- {
		if lookup.Events[i].Offset == e.GetOffset() {
			lookup.Events[i].ComplianceSnapshot = c
			return nil
		}
	}
	return fmt.Errorf("%v %v %v %w %v", e.GetExchange(), e.GetAssetType(), e.Pair(), errNoDataAtOffset, e.GetOffset())
}

// CalculateAllResults calculates the statistics of all exchange asset pair holdings,
// orders, ratios and drawdowns
func (s *Statistic) CalculateAllResults() error {
	log.Infoln(common.Statistics, "Calculating backtesting results")
	s.PrintAllEventsChronologically()
	currCount := 0
	var finalResults []FinalResultsHolder
	var err error
	for exchangeName, exchangeMap := range s.ExchangeAssetPairStatistics {
		for assetItem, assetMap := range exchangeMap {
			for b, baseMap := range assetMap {
				for q, stats := range baseMap {
					currCount++
					last := stats.Events[len(stats.Events)-1]
					if last.PNL != nil {
						s.HasCollateral = true
					}
					err = stats.CalculateResults(s.RiskFreeRate)
					if err != nil {
						log.Errorln(common.Statistics, err)
					}
					stats.FinalHoldings = last.Holdings
					stats.InitialHoldings = stats.Events[0].Holdings
					if last.ComplianceSnapshot == nil {
						return errMissingSnapshots
					}
					stats.FinalOrders = *last.ComplianceSnapshot
					s.StartDate = stats.Events[0].Time
					s.EndDate = last.Time
					cp := currency.NewPair(b.Currency(), q.Currency())
					stats.PrintResults(exchangeName, assetItem, cp, s.FundManager.IsUsingExchangeLevelFunding())

					finalResults = append(finalResults, FinalResultsHolder{
						Exchange:         exchangeName,
						Asset:            assetItem,
						Pair:             cp,
						MaxDrawdown:      stats.MaxDrawdown,
						MarketMovement:   stats.MarketMovement,
						StrategyMovement: stats.StrategyMovement,
					})
					if assetItem.IsFutures() {
						s.TotalLongOrders += stats.BuyOrders
						s.TotalShortOrders += stats.SellOrders
					} else {
						s.TotalBuyOrders += stats.BuyOrders
						s.TotalSellOrders += stats.SellOrders
					}
					s.TotalOrders += stats.TotalOrders
					if stats.ShowMissingDataWarning {
						s.WasAnyDataMissing = true
					}
				}
			}
		}
	}
	s.FundingStatistics, err = CalculateFundingStatistics(s.FundManager, s.ExchangeAssetPairStatistics, s.RiskFreeRate, s.CandleInterval)
	if err != nil {
		return err
	}
	err = s.FundingStatistics.PrintResults(s.WasAnyDataMissing)
	if err != nil {
		return err
	}
	if currCount > 1 {
		s.BiggestDrawdown = s.GetTheBiggestDrawdownAcrossCurrencies(finalResults)
		s.BestMarketMovement = s.GetBestMarketPerformer(finalResults)
		s.BestStrategyResults = s.GetBestStrategyPerformer(finalResults)
		s.PrintTotalResults()
	}

	return nil
}

// GetBestMarketPerformer returns the best final market movement
func (s *Statistic) GetBestMarketPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	var result FinalResultsHolder
	for i := range results {
		if results[i].MarketMovement.GreaterThan(result.MarketMovement) || result.MarketMovement.IsZero() {
			result = results[i]
		}
	}

	return &result
}

// GetBestStrategyPerformer returns the best performing strategy result
func (s *Statistic) GetBestStrategyPerformer(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].StrategyMovement.GreaterThan(result.StrategyMovement) || result.StrategyMovement.IsZero() {
			result = &results[i]
		}
	}

	return result
}

// GetTheBiggestDrawdownAcrossCurrencies returns the biggest drawdown across all currencies in a backtesting run
func (s *Statistic) GetTheBiggestDrawdownAcrossCurrencies(results []FinalResultsHolder) *FinalResultsHolder {
	result := &FinalResultsHolder{}
	for i := range results {
		if results[i].MaxDrawdown.DrawdownPercent.GreaterThan(result.MaxDrawdown.DrawdownPercent) || result.MaxDrawdown.DrawdownPercent.IsZero() {
			result = &results[i]
		}
	}

	return result
}

func addEventOutputToTime(events []eventOutputHolder, t time.Time, message string) []eventOutputHolder {
	for i := range events {
		if events[i].Time.Equal(t) {
			events[i].Events = append(events[i].Events, message)
			return events
		}
	}
	events = append(events, eventOutputHolder{Time: t, Events: []string{message}})
	return events
}

// SetStrategyName sets the name for statistical identification
func (s *Statistic) SetStrategyName(name string) {
	s.StrategyName = name
}

// Serialise outputs the Statistic struct in json
func (s *Statistic) Serialise() (string, error) {
	s.CurrencyStatistics = nil
	for _, exchangeMap := range s.ExchangeAssetPairStatistics {
		for _, assetMap := range exchangeMap {
			for _, baseMap := range assetMap {
				for _, stats := range baseMap {
					s.CurrencyStatistics = append(s.CurrencyStatistics, stats)
				}
			}
		}
	}

	resp, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return "", err
	}

	return string(resp), nil
}
