package metrics

import (
	"expvar"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	orderbookDesyncMetric    = expvar.NewMap("gct_websocket_orderbook_desync_total")
	orderbookDesyncGapMetric = expvar.NewMap("gct_websocket_orderbook_desync_gap_total")
	orderbookResyncMetric    = expvar.NewMap("gct_websocket_orderbook_resync_total")
	orderbookEmptyMetric     = expvar.NewMap("gct_websocket_orderbook_empty_total")
)

var orderbookSyncStats = orderbookSyncStore{
	lookup:      make(map[orderbookSyncKey]*orderbookSyncCounts),
	emptyLookup: make(map[orderbookSyncKey]struct{}),
}

type orderbookSyncStore struct {
	sync.Mutex
	lookup      map[orderbookSyncKey]*orderbookSyncCounts
	emptyLookup map[orderbookSyncKey]struct{}
}

type orderbookSyncKey struct {
	exchange string
	pair     string
	asset    string
	channel  string
}

type orderbookSyncCounts struct {
	desyncs              int64
	sequenceGapTotal     int64
	lastUpdateID         int64
	lastFirstUpdateID    int64
	lastReceivedUpdateID int64
	resyncStarted        int64
	resyncRequested      int64
	resyncSucceeded      int64
	resyncFailed         int64
	resyncRequestFailed  int64
	resyncDurationTotal  time.Duration
	resyncDurationCount  int64
	resyncPermitWait     time.Duration
	resyncFetchDelay     time.Duration
	resyncSnapshotFetch  time.Duration
	resyncRetryWait      time.Duration
	resyncSnapshotLoad   time.Duration
	resyncCacheLockWait  time.Duration
	resyncPendingApply   time.Duration
	resyncFetchAttempts  int64
	resyncQueuedUpdates  int64
	resyncMaxQueued      int64
	lastSnapshotID       int64
	lastPendingFirstID   int64
	lastPendingUpdateID  int64
	reasons              map[string]int64
}

// OrderbookSyncSummary contains aggregated orderbook desync and resync metrics.
type OrderbookSyncSummary struct {
	Items                []OrderbookSyncSummaryItem
	TotalDesyncs         int64
	TotalSequenceGap     int64
	TotalDroppedBooks    int64
	TotalResyncSuccess   int64
	TotalResyncFailure   int64
	TotalResyncTime      time.Duration
	TotalResyncTimed     int64
	TotalPermitWait      time.Duration
	TotalFetchDelay      time.Duration
	TotalSnapshotFetch   time.Duration
	TotalRetryWait       time.Duration
	TotalSnapshotLoad    time.Duration
	TotalCacheLockWait   time.Duration
	TotalPendingApply    time.Duration
	TotalFetchAttempts   int64
	TotalQueuedUpdates   int64
	MaxQueuedUpdates     int64
	TotalEmptyOrderbooks int64
	WorstOffender        *OrderbookSyncSummaryItem
}

// OrderbookSyncSummaryItem contains orderbook desync and resync metrics for one exchange/pair/asset/channel.
type OrderbookSyncSummaryItem struct {
	Exchange                 string
	Pair                     string
	Asset                    string
	Channel                  string
	Desyncs                  int64
	SequenceGapTotal         int64
	LastUpdateID             int64
	LastFirstUpdateID        int64
	LastReceivedUpdateID     int64
	DroppedOrderbooks        int64
	ResyncStarted            int64
	ResyncRequested          int64
	ResyncSucceeded          int64
	ResyncFailed             int64
	ResyncRequestFailed      int64
	ResyncTime               time.Duration
	ResyncTimed              int64
	PermitWait               time.Duration
	FetchDelay               time.Duration
	SnapshotFetch            time.Duration
	RetryWait                time.Duration
	SnapshotLoad             time.Duration
	CacheLockWait            time.Duration
	PendingApply             time.Duration
	FetchAttempts            int64
	QueuedUpdates            int64
	MaxQueuedUpdates         int64
	LastSnapshotUpdateID     int64
	LastPendingFirstUpdateID int64
	LastPendingUpdateID      int64
	Reasons                  map[string]int64
}

// OrderbookSyncEvent identifies an orderbook desync or resync metric event.
type OrderbookSyncEvent struct {
	Exchange             string
	Pair                 currency.Pair
	Asset                asset.Item
	Channel              string
	Reason               string
	Result               string
	LastUpdateID         int64
	FirstUpdateID        int64
	UpdateID             int64
	Duration             time.Duration
	PermitWait           time.Duration
	FetchDelay           time.Duration
	SnapshotFetch        time.Duration
	RetryWait            time.Duration
	SnapshotLoad         time.Duration
	CacheLockWait        time.Duration
	PendingApply         time.Duration
	FetchAttempts        int64
	QueuedUpdates        int64
	SnapshotUpdateID     int64
	PendingFirstUpdateID int64
	PendingUpdateID      int64
}

// RecordOrderbookDesync increments the websocket orderbook desync metric.
func RecordOrderbookDesync(event *OrderbookSyncEvent) {
	if event == nil {
		return
	}
	orderbookDesyncMetric.Add(formatOrderbookSyncMetricKey(event), 1)
	gap := event.FirstUpdateID - event.LastUpdateID - 1
	if gap > 0 {
		orderbookDesyncGapMetric.Add(formatOrderbookSyncMetricKey(event), gap)
	}
	recordOrderbookSyncDesync(event, gap)
}

// RecordOrderbookResync increments the websocket orderbook resync metric.
func RecordOrderbookResync(event *OrderbookSyncEvent) {
	if event == nil {
		return
	}
	orderbookResyncMetric.Add(formatOrderbookSyncMetricKey(event), 1)
	recordOrderbookSyncResync(event)
}

// RecordEmptyOrderbook records an orderbook which supplied a snapshot with an empty side.
func RecordEmptyOrderbook(event *OrderbookSyncEvent) {
	if event == nil {
		return
	}
	orderbookEmptyMetric.Add(formatOrderbookSyncMetricKey(event), 1)
	orderbookSyncStats.Lock()
	defer orderbookSyncStats.Unlock()
	if orderbookSyncStats.emptyLookup == nil {
		orderbookSyncStats.emptyLookup = make(map[orderbookSyncKey]struct{})
	}
	orderbookSyncStats.emptyLookup[orderbookSyncKey{
		exchange: event.Exchange,
		pair:     event.Pair.String(),
		asset:    event.Asset.String(),
		channel:  event.Channel,
	}] = struct{}{}
}

// SnapshotOrderbookSyncSummary returns a stable snapshot of websocket orderbook sync metrics.
func SnapshotOrderbookSyncSummary() OrderbookSyncSummary {
	orderbookSyncStats.Lock()
	defer orderbookSyncStats.Unlock()

	items := make([]OrderbookSyncSummaryItem, 0, len(orderbookSyncStats.lookup))
	for key, counts := range orderbookSyncStats.lookup {
		item := OrderbookSyncSummaryItem{
			Exchange:                 key.exchange,
			Pair:                     key.pair,
			Asset:                    key.asset,
			Channel:                  key.channel,
			Desyncs:                  counts.desyncs,
			SequenceGapTotal:         counts.sequenceGapTotal,
			LastUpdateID:             counts.lastUpdateID,
			LastFirstUpdateID:        counts.lastFirstUpdateID,
			LastReceivedUpdateID:     counts.lastReceivedUpdateID,
			DroppedOrderbooks:        counts.resyncStarted + counts.resyncRequested + counts.resyncRequestFailed,
			ResyncStarted:            counts.resyncStarted,
			ResyncRequested:          counts.resyncRequested,
			ResyncSucceeded:          counts.resyncSucceeded,
			ResyncFailed:             counts.resyncFailed,
			ResyncRequestFailed:      counts.resyncRequestFailed,
			ResyncTime:               counts.resyncDurationTotal,
			ResyncTimed:              counts.resyncDurationCount,
			PermitWait:               counts.resyncPermitWait,
			FetchDelay:               counts.resyncFetchDelay,
			SnapshotFetch:            counts.resyncSnapshotFetch,
			RetryWait:                counts.resyncRetryWait,
			SnapshotLoad:             counts.resyncSnapshotLoad,
			CacheLockWait:            counts.resyncCacheLockWait,
			PendingApply:             counts.resyncPendingApply,
			FetchAttempts:            counts.resyncFetchAttempts,
			QueuedUpdates:            counts.resyncQueuedUpdates,
			MaxQueuedUpdates:         counts.resyncMaxQueued,
			LastSnapshotUpdateID:     counts.lastSnapshotID,
			LastPendingFirstUpdateID: counts.lastPendingFirstID,
			LastPendingUpdateID:      counts.lastPendingUpdateID,
			Reasons:                  make(map[string]int64, len(counts.reasons)),
		}
		maps.Copy(item.Reasons, counts.reasons)
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DroppedOrderbooks != items[j].DroppedOrderbooks {
			return items[i].DroppedOrderbooks > items[j].DroppedOrderbooks
		}
		if items[i].Desyncs != items[j].Desyncs {
			return items[i].Desyncs > items[j].Desyncs
		}
		if items[i].SequenceGapTotal != items[j].SequenceGapTotal {
			return items[i].SequenceGapTotal > items[j].SequenceGapTotal
		}
		return items[i].label() < items[j].label()
	})

	summary := OrderbookSyncSummary{
		Items:                items,
		TotalEmptyOrderbooks: int64(len(orderbookSyncStats.emptyLookup)),
	}
	for i := range items {
		summary.TotalDesyncs += items[i].Desyncs
		summary.TotalSequenceGap += items[i].SequenceGapTotal
		summary.TotalDroppedBooks += items[i].DroppedOrderbooks
		summary.TotalResyncSuccess += items[i].ResyncSucceeded
		summary.TotalResyncFailure += items[i].ResyncFailed + items[i].ResyncRequestFailed
		summary.TotalResyncTime += items[i].ResyncTime
		summary.TotalResyncTimed += items[i].ResyncTimed
		summary.TotalPermitWait += items[i].PermitWait
		summary.TotalFetchDelay += items[i].FetchDelay
		summary.TotalSnapshotFetch += items[i].SnapshotFetch
		summary.TotalRetryWait += items[i].RetryWait
		summary.TotalSnapshotLoad += items[i].SnapshotLoad
		summary.TotalCacheLockWait += items[i].CacheLockWait
		summary.TotalPendingApply += items[i].PendingApply
		summary.TotalFetchAttempts += items[i].FetchAttempts
		summary.TotalQueuedUpdates += items[i].QueuedUpdates
		summary.MaxQueuedUpdates = max(summary.MaxQueuedUpdates, items[i].MaxQueuedUpdates)
	}
	if len(items) != 0 {
		summary.WorstOffender = &summary.Items[0]
	}
	return summary
}

// OrderbookSyncSummaryLines returns a readable shutdown summary of orderbook sync metrics.
func OrderbookSyncSummaryLines() []string {
	summary := SnapshotOrderbookSyncSummary()
	if len(summary.Items) == 0 {
		return []string{
			fmt.Sprintf("Websocket orderbook sync summary: empty_orderbooks_observed=%d dropped_orderbooks=0 desyncs=0 sequence_gap_total=0 ", summary.TotalEmptyOrderbooks) +
				"resync_success=0 resync_failure=0 resync_time_total=0s resync_time_avg=N/A",
		}
	}

	lines := []string{fmt.Sprintf(
		"Websocket orderbook sync summary: empty_orderbooks_observed=%d dropped_orderbooks=%d desyncs=%d sequence_gap_total=%d "+
			"resync_success=%d resync_failure=%d resync_time_total=%s resync_time_avg=%s",
		summary.TotalEmptyOrderbooks,
		summary.TotalDroppedBooks,
		summary.TotalDesyncs,
		summary.TotalSequenceGap,
		summary.TotalResyncSuccess,
		summary.TotalResyncFailure,
		summary.TotalResyncTime,
		formatOrderbookSyncDuration(summary.TotalResyncTime, summary.TotalResyncTimed),
	)}
	if summary.WorstOffender != nil {
		lines = append(lines, fmt.Sprintf(
			"Worst dropped orderbook offender: %s dropped_orderbooks=%d desyncs=%d sequence_gap_total=%d top_reason=%s",
			summary.WorstOffender.label(),
			summary.WorstOffender.DroppedOrderbooks,
			summary.WorstOffender.Desyncs,
			summary.WorstOffender.SequenceGapTotal,
			summary.WorstOffender.topReason(),
		))
	}

	limit := min(len(summary.Items), 5)
	for i := range limit {
		item := summary.Items[i]
		lines = append(lines, fmt.Sprintf(
			"Orderbook sync #%d: %s dropped_orderbooks=%d desyncs=%d sequence_gap_total=%d resync[started=%d requested=%d succeeded=%d failed=%d request_failed=%d] reasons=%s",
			i+1,
			item.label(),
			item.DroppedOrderbooks,
			item.Desyncs,
			item.SequenceGapTotal,
			item.ResyncStarted,
			item.ResyncRequested,
			item.ResyncSucceeded,
			item.ResyncFailed,
			item.ResyncRequestFailed,
			item.reasonSummary(),
		))
	}
	return lines
}

// OrderbookSyncSummaryLinesForExchange returns a readable shutdown summary for one exchange.
func OrderbookSyncSummaryLinesForExchange(exchangeName string) []string {
	if exchangeName == "" {
		return OrderbookSyncSummaryLines()
	}
	summary := SnapshotOrderbookSyncSummary()
	var items []OrderbookSyncSummaryItem
	for i := range summary.Items {
		if summary.Items[i].Exchange == exchangeName {
			items = append(items, summary.Items[i])
		}
	}
	if len(items) == 0 {
		return []string{fmt.Sprintf(
			"Websocket orderbook sync summary for %s: dropped_orderbooks=0 desyncs=0 "+
				"sequence_gap_total=0 resync_success=0 resync_failure=0 resync_time_total=0s "+
				"resync_time_avg=N/A",
			exchangeName,
		)}
	}

	var totalDesyncs, totalSequenceGap, totalDroppedBooks, totalResyncSuccess, totalResyncFailure int64
	var totalResyncTime time.Duration
	var totalResyncTimed int64
	for i := range items {
		totalDesyncs += items[i].Desyncs
		totalSequenceGap += items[i].SequenceGapTotal
		totalDroppedBooks += items[i].DroppedOrderbooks
		totalResyncSuccess += items[i].ResyncSucceeded
		totalResyncFailure += items[i].ResyncFailed + items[i].ResyncRequestFailed
		totalResyncTime += items[i].ResyncTime
		totalResyncTimed += items[i].ResyncTimed
	}

	lines := []string{fmt.Sprintf(
		"Websocket orderbook sync summary for %s: dropped_orderbooks=%d desyncs=%d "+
			"sequence_gap_total=%d resync_success=%d resync_failure=%d resync_time_total=%s "+
			"resync_time_avg=%s",
		exchangeName,
		totalDroppedBooks,
		totalDesyncs,
		totalSequenceGap,
		totalResyncSuccess,
		totalResyncFailure,
		totalResyncTime,
		formatOrderbookSyncDuration(totalResyncTime, totalResyncTimed),
	)}
	limit := min(len(items), 3)
	for i := range limit {
		item := items[i]
		lines = append(lines, fmt.Sprintf(
			"Orderbook sync %s #%d: pair=%s asset=%s channel=%s dropped_orderbooks=%d desyncs=%d sequence_gap_total=%d resync[started=%d requested=%d succeeded=%d failed=%d request_failed=%d] reasons=%s",
			exchangeName,
			i+1,
			item.Pair,
			item.Asset,
			item.Channel,
			item.DroppedOrderbooks,
			item.Desyncs,
			item.SequenceGapTotal,
			item.ResyncStarted,
			item.ResyncRequested,
			item.ResyncSucceeded,
			item.ResyncFailed,
			item.ResyncRequestFailed,
			item.reasonSummary(),
		))
	}
	return lines
}

func recordOrderbookSyncDesync(event *OrderbookSyncEvent, gap int64) {
	orderbookSyncStats.Lock()
	defer orderbookSyncStats.Unlock()

	counts := orderbookSyncStats.load(event)
	counts.desyncs++
	counts.lastUpdateID = event.LastUpdateID
	counts.lastFirstUpdateID = event.FirstUpdateID
	counts.lastReceivedUpdateID = event.UpdateID
	if gap > 0 {
		counts.sequenceGapTotal += gap
	}
	reason := event.Reason
	if reason == "" {
		reason = "unknown"
	}
	counts.reasons[reason]++
}

func recordOrderbookSyncResync(event *OrderbookSyncEvent) {
	orderbookSyncStats.Lock()
	defer orderbookSyncStats.Unlock()

	counts := orderbookSyncStats.load(event)
	switch event.Result {
	case "started":
		counts.resyncStarted++
	case "requested":
		counts.resyncRequested++
	case "succeeded":
		counts.resyncSucceeded++
	case "failed":
		counts.resyncFailed++
	case "request_failed":
		counts.resyncRequestFailed++
	}
	if event.Duration > 0 {
		counts.resyncDurationTotal += event.Duration
		counts.resyncDurationCount++
	}
	counts.resyncPermitWait += event.PermitWait
	counts.resyncFetchDelay += event.FetchDelay
	counts.resyncSnapshotFetch += event.SnapshotFetch
	counts.resyncRetryWait += event.RetryWait
	counts.resyncSnapshotLoad += event.SnapshotLoad
	counts.resyncCacheLockWait += event.CacheLockWait
	counts.resyncPendingApply += event.PendingApply
	counts.resyncFetchAttempts += event.FetchAttempts
	counts.resyncQueuedUpdates += event.QueuedUpdates
	counts.resyncMaxQueued = max(counts.resyncMaxQueued, event.QueuedUpdates)
	if event.SnapshotUpdateID != 0 {
		counts.lastSnapshotID = event.SnapshotUpdateID
	}
	if event.PendingFirstUpdateID != 0 {
		counts.lastPendingFirstID = event.PendingFirstUpdateID
	}
	if event.PendingUpdateID != 0 {
		counts.lastPendingUpdateID = event.PendingUpdateID
	}
}

func formatOrderbookSyncDuration(total time.Duration, count int64) string {
	if count <= 0 {
		return "N/A"
	}
	return (total / time.Duration(count)).String()
}

func (s *orderbookSyncStore) load(event *OrderbookSyncEvent) *orderbookSyncCounts {
	key := orderbookSyncKey{
		exchange: event.Exchange,
		pair:     event.Pair.String(),
		asset:    event.Asset.String(),
		channel:  event.Channel,
	}
	counts, ok := s.lookup[key]
	if !ok {
		counts = &orderbookSyncCounts{reasons: make(map[string]int64)}
		s.lookup[key] = counts
	}
	return counts
}

func (i *OrderbookSyncSummaryItem) label() string {
	return fmt.Sprintf("exchange=%s pair=%s asset=%s channel=%s", i.Exchange, i.Pair, i.Asset, i.Channel)
}

func (i *OrderbookSyncSummaryItem) topReason() string {
	var reason string
	var count int64
	for r, c := range i.Reasons {
		if c > count || c == count && r < reason {
			reason = r
			count = c
		}
	}
	if reason == "" {
		return "none"
	}
	return fmt.Sprintf("%s:%d", reason, count)
}

func (i *OrderbookSyncSummaryItem) reasonSummary() string {
	if len(i.Reasons) == 0 {
		return "none"
	}
	reasons := make([]string, 0, len(i.Reasons))
	for reason, count := range i.Reasons {
		reasons = append(reasons, fmt.Sprintf("%s:%d", reason, count))
	}
	sort.Strings(reasons)
	return strings.Join(reasons, ",")
}

func formatOrderbookSyncMetricKey(event *OrderbookSyncEvent) string {
	parts := []string{
		"exchange=" + event.Exchange,
		"asset=" + event.Asset.String(),
		"pair=" + event.Pair.String(),
	}
	if event.Channel != "" {
		parts = append(parts, "channel="+event.Channel)
	}
	if event.Reason != "" {
		parts = append(parts, "reason="+event.Reason)
	}
	if event.Result != "" {
		parts = append(parts, "result="+event.Result)
	}
	return strings.Join(parts, ",")
}
