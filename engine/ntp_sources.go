package engine

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/ntp"
)

var (
	errNTPSource        = errors.New("invalid NTP source")
	errNTPUncertain     = errors.New("measurement uncertainty overlaps the configured tolerance")
	errNTPMeasurement   = errors.New("invalid NTP measurement interval")
	errNTPSourceAddress = errors.New("no distinct address available for NTP source")
	errNTPSourceBackoff = errors.New("NTP source asked for fewer queries")
)

type ntpSource struct {
	host         string
	port         string
	addresses    []netip.Addr
	refreshAfter time.Duration
}

// ntpEndpoint records a server's kiss-o'-death restrictions for one IP address.
// It is kept across DNS refreshes, so an address reappearing in a later answer still respects them.
type ntpEndpoint struct {
	nextAllowed time.Duration
	backoff     time.Duration
	denied      bool
}

type ntpSurvey struct {
	sources   []ntpSource
	endpoints map[netip.Addr]ntpEndpoint
	resolve   func(context.Context, string, string) ([]netip.Addr, error)
	query     func(context.Context, string) (*ntp.Response, error)
	now       func() time.Time
}

// ntpFailures holds the reasons some servers gave no usable measurement.
// Several reasons can apply together, so each reason is a bit flag.
// If one address for a name works, problems with that name's other addresses are not reported.
// A failed DNS refresh can still appear in debug logs even when a cached address works.
// This combination is GCT's own policy.
type ntpFailures uint8

const (
	ntpLookupFailed ntpFailures = 1 << iota
	ntpUnreachable
	ntpInvalidReply
	ntpRefused
	ntpRateLimited
	ntpDuplicateAddress
	ntpNameNotFound
)

type ntpObservation struct {
	state       ntpClockState
	failures    ntpFailures
	interval    ntpInterval
	configured  int
	usable      int
	agreeing    int
	reason      error
	diagnostics error
}

func newNTPSurvey(names []string) (*ntpSurvey, error) {
	survey := &ntpSurvey{
		endpoints: make(map[netip.Addr]ntpEndpoint),
		resolve:   net.DefaultResolver.LookupNetIP,
		query:     queryNTP,
		now:       time.Now,
	}
	seen := make(map[string]bool)
	for _, name := range names {
		host, port, err := parseNTPSource(name)
		if err != nil {
			return nil, err
		}
		key := net.JoinHostPort(host, port)
		if ip, err := netip.ParseAddr(host); err == nil {
			// Different spellings of one server must not get several votes.
			// Compare servers by IP alone, as chrony does, which stores no zone: https://github.com/mlichvar/chrony/blob/8df4f1263e206585e6c4e61352d042c2dd12916a/ntp_sources.c#L374-L379
			key = net.JoinHostPort(ip.WithZone("").String(), port)
		}
		if !seen[key] {
			seen[key] = true
			source := ntpSource{host: host, port: port}
			// Go's resolver drops an IPv6 zone, so query an IP literal exactly as configured and never resolve it.
			if ip, err := netip.ParseAddr(host); err == nil {
				source.addresses, source.refreshAfter = []netip.Addr{ip}, math.MaxInt64
			}
			survey.sources = append(survey.sources, source)
		}
	}
	return survey, nil
}

// parseNTPSource adapts beevik's fixHostPort address forms, then canonicalises
// configured voting identities and validates the port. Copyright Brett Vickers.
// See THIRD_PARTY_NOTICES at the repository root and the original implementation: https://github.com/beevik/ntp/blob/953b63646f5273d44de88b68e5862ad155ed4660/ntp.go#L733
// Different spellings of the same entry count as one vote.
// Duplicate IP addresses are handled later, when addresses are chosen.
func parseNTPSource(name string) (host, port string, err error) {
	name = strings.TrimSpace(name)
	switch {
	case strings.HasPrefix(name, "["):
		end := strings.IndexByte(name, ']')
		switch {
		case end < 0:
			return "", "", fmt.Errorf("%w %q: missing closing bracket", errNTPSource, name)
		case end+1 == len(name):
			name += ":123"
		case name[end+1] != ':':
			return "", "", fmt.Errorf("%w %q: unexpected character after closing bracket", errNTPSource, name)
		}
	case !strings.Contains(name, ":"):
		name = net.JoinHostPort(name, "123")
	case strings.Count(name, ":") > 1:
		name = net.JoinHostPort(name, "123")
	}
	host, port, err = net.SplitHostPort(name)
	if err != nil {
		return "", "", fmt.Errorf("%w %q: %w", errNTPSource, name, err)
	}
	if ip, ipErr := netip.ParseAddr(host); ipErr == nil {
		host = ip.Unmap().String()
	} else if strings.Contains(host, ":") {
		return "", "", fmt.Errorf("%w %q: %w", errNTPSource, name, ipErr)
	} else {
		host = strings.TrimSuffix(strings.ToLower(host), ".")
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if host == "" || err != nil || number == 0 {
		return "", "", fmt.Errorf("%w %q", errNTPSource, name)
	}
	return host, strconv.FormatUint(number, 10), nil
}

// observe runs one check round across configured servers.
// It resolves names when due, sends at most one request per entry, and combines replies into one result.
// An entry may try several addresses when earlier ones can't be dialled.
// Servers without a usable reply still count against required majority.
func (s *ntpSurvey) observe(ctx context.Context, elapsed, allowedDifference, allowedNegativeDifference time.Duration) ntpObservation {
	observation := ntpObservation{configured: len(s.sources)}
	started := s.now()
	responses := make([]*ntp.Response, 0, len(s.sources))
	var failures []error
	used := make(map[netip.Addr]bool)
	for i := range s.sources {
		if ctx.Err() != nil {
			break
		}
		source := &s.sources[i]
		lookupFailure := ntpNameNotFound
		if source.refreshAfter == 0 || elapsed >= source.refreshAfter {
			// Re-resolve names at most hourly after successful lookups, and retry failed lookups at next check.
			// Datadog also resolves names on every check: https://github.com/DataDog/datadog-agent/blob/9d9f15efe41650e4dc937be15c02f54dae42c887/pkg/collector/corechecks/net/ntp/ntp.go#L278
			// NTP Pool's once-an-hour rule is about replacing servers that stop responding: https://www.ntppool.org/en/vendors.html
			source.refreshAfter = 0
			resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			addresses, err := s.resolve(resolveCtx, "ip", source.host)
			cancel()
			// If lookup fails or returns nothing, keep using last known addresses.
			// Votes need fresh NTP replies.
			// ntpd-rs also keeps its existing servers when a lookup fails: https://github.com/pendulum-project/ntpd-rs/blob/46ec9bb4d5b6cb24f814f5543d85b9138afb4cba/ntpd/src/daemon/spawn/pool.rs#L55
			switch {
			case err != nil:
				lookupFailure = ntpLookupFailed
				// Tell "no address for this name" apart from other lookup failures, so message suggests right checks.
				// This classification does not change GCT's scheduled retries.
				// chrony likewise separates permanent from temporary resolution failures: https://github.com/mlichvar/chrony/blob/8df4f1263e206585e6c4e61352d042c2dd12916a/ntp_sources.c#L651
				if dnsError, ok := errors.AsType[*net.DNSError](err); ok && dnsError.IsNotFound {
					lookupFailure = ntpNameNotFound
				}
				failures = append(failures, fmt.Errorf("resolve NTP source %s: %w", source.host, err))
			case len(addresses) == 0:
				failures = append(failures, fmt.Errorf("%w %s: DNS returned no IP address", errNTPSourceAddress, source.host))
			default:
				var current netip.Addr
				if len(source.addresses) != 0 {
					current = source.addresses[0].Unmap()
				}
				source.addresses = addresses
				source.refreshAfter = elapsed + time.Hour
				rand.Shuffle(len(addresses), func(a, b int) { addresses[a], addresses[b] = addresses[b], addresses[a] }) //nolint:gosec // Only spreads load across DNS answers, so predictable order is harmless
				// Refresh must not swap working address for one that may time out and cost entry its vote.
				// Keep current address as chrony does on refresh: https://github.com/mlichvar/chrony/blob/8df4f1263e206585e6c4e61352d042c2dd12916a/ntp_sources.c#L543-L562
				for j, address := range source.addresses {
					if address.Unmap() == current {
						source.addresses[0], source.addresses[j] = source.addresses[j], source.addresses[0]
						break
					}
				}
			}
		}
		if len(source.addresses) == 0 {
			observation.failures |= lookupFailure
			continue
		}
		var address netip.Addr
		var response *ntp.Response
		var err error
		var unavailable error
		var blocked ntpFailures
		var failed []netip.Addr
		for _, candidate := range source.addresses {
			candidate = candidate.Unmap()
			if !candidate.IsValid() {
				blocked |= ntpNameNotFound
				continue
			}
			if used[candidate.WithZone("")] {
				blocked |= ntpDuplicateAddress
				continue
			}
			endpoint := s.endpoints[candidate.WithZone("")]
			if endpoint.denied {
				blocked |= ntpRefused
				unavailable = errors.Join(unavailable, fmt.Errorf("NTP source %s at %s refused further queries: %w", source.host, candidate, ntp.ErrKissOfDeath))
				continue
			}
			if elapsed < endpoint.nextAllowed {
				blocked |= ntpRateLimited
				unavailable = errors.Join(unavailable, fmt.Errorf("%w: %s at %s, remaining %s", errNTPSourceBackoff, source.host, candidate, endpoint.nextAllowed-elapsed))
				continue
			}
			if address.IsValid() {
				// The previous address failed to dial, so no request was sent to it.
				blocked |= ntpUnreachable
				unavailable = errors.Join(unavailable, fmt.Errorf("query NTP source %s at %s: %w", source.host, address, err))
				failed = append(failed, address)
			}
			address = candidate
			// Count each IP address at most once, even if several names or ports lead to it.
			// Different IP addresses don't prove different operators.
			used[address.WithZone("")] = true
			response, err = s.query(ctx, net.JoinHostPort(address.String(), source.port))
			// A socket that couldn't be opened sent nothing, such as an IPv6 address on an IPv4-only host.
			// Try the entry's next address this round, so DNS answers for one address family can't cost the entry its vote.
			if opError, ok := errors.AsType[*net.OpError](err); !ok || opError.Op != "dial" || ctx.Err() != nil {
				break
			}
		}
		if !address.IsValid() {
			observation.failures |= blocked
			if unavailable == nil {
				unavailable = fmt.Errorf("%w %s: DNS returned no unused IP address", errNTPSourceAddress, source.host)
			}
			failures = append(failures, unavailable)
			continue
		}
		var queryFailure ntpFailures
		if err != nil {
			queryFailure = ntpInvalidReply
			if _, networkError := errors.AsType[net.Error](err); networkError || errors.Is(err, context.DeadlineExceeded) {
				queryFailure = ntpUnreachable
			}
		}
		if errors.Is(err, ntp.ErrKissOfDeath) && response != nil {
			endpoint := s.endpoints[address.WithZone("")]
			switch response.KissCode {
			case "DENY", "RSTR":
				endpoint.denied = true
				queryFailure = ntpRefused
			case "RATE":
				queryFailure = ntpRateLimited
				// Increase wait after repeated RATE replies, starting at no less than 30 minutes and capped at 8192 seconds.
				// Cap requested delay because reply is not authenticated (RFC 8633 section 5.4).
				endpoint.backoff = min(max(defaultNTPCheckInterval*2, endpoint.backoff*2, response.Poll), 8192*time.Second)
				endpoint.nextAllowed = elapsed + endpoint.backoff
			}
			s.endpoints[address.WithZone("")] = endpoint
		}
		if err != nil {
			failed = append(failed, address)
		}
		// Move failed addresses to back so later rounds try another eligible address first.
		// Do not rotate on shutdown.
		// A different address is not necessarily a different server.
		// systemd-timesyncd also moves to next address after failure, and this reordering is GCT's implementation: https://github.com/systemd/systemd/blob/885fe07ee37cff7316680b5088d11081e01813b1/src/timesync/timesyncd-manager.c#L926
		if len(failed) != 0 && ctx.Err() == nil {
			rank := func(a netip.Addr) int {
				if slices.Contains(failed, a.Unmap()) {
					return 1
				}
				return 0
			}
			slices.SortStableFunc(source.addresses, func(a, b netip.Addr) int { return cmp.Compare(rank(a), rank(b)) })
		}
		if err != nil {
			// Report skipped addresses' reasons too, because no address for this name gave a reply.
			observation.failures |= blocked | queryFailure
			failures = append(failures, errors.Join(unavailable, fmt.Errorf("query NTP source %s at %s: %w", source.host, address, err)))
			continue
		}
		responses = append(responses, response)
	}
	intervals := make([]ntpInterval, 0, len(responses))
	// Local clock may drift while replies are collected from several servers.
	// Widen each reply's range by NTP's assumed 15 ppm drift allowance (RFC 5905 "PHI") over entire collection time.
	// beevik's single-reply root distance leaves this out: https://github.com/beevik/ntp/blob/953b63646f5273d44de88b68e5862ad155ed4660/ntp.go#L829
	// Sudden clock jumps are handled separately by the worker.
	age := s.now().Sub(started)
	for _, response := range responses {
		interval, err := ntpResponseInterval(response, age)
		if err == nil {
			intervals = append(intervals, interval)
		} else {
			observation.failures |= ntpInvalidReply
			failures = append(failures, err)
		}
	}
	observation.usable = len(intervals)
	observation.interval, observation.agreeing, observation.reason = selectNTPInterval(intervals, observation.configured)
	if observation.reason == nil {
		observation.state = observation.interval.classify(allowedDifference, allowedNegativeDifference)
		if observation.state == ntpUnknown {
			observation.reason = errNTPUncertain
		}
	}
	observation.diagnostics = errors.Join(failures...)
	return observation
}

// ntpResponseInterval turns one reply into an estimated clock-error range.
// It widens reply's root distance by 15 ppm drift allowance over collection age, and rejects invalid or overflowing values.
func ntpResponseInterval(response *ntp.Response, age time.Duration) (ntpInterval, error) {
	if response == nil || age < 0 || response.RootDistance < 0 {
		return ntpInterval{}, errNTPMeasurement
	}
	correction, distance := response.ClockOffset, response.RootDistance
	// Round 15 ppm allowance upwards so integer rounding does not reduce it, and divide first to avoid overflow.
	drift := age/1_000_000*15 + (age%1_000_000*15+999_999)/1_000_000
	if distance > time.Duration(math.MaxInt64)-drift {
		return ntpInterval{}, errNTPMeasurement
	}
	distance += drift
	if correction < time.Duration(math.MinInt64)+distance || correction > time.Duration(math.MaxInt64)-distance {
		return ntpInterval{}, errNTPMeasurement
	}
	return ntpInterval{correction - distance, correction + distance}, nil
}
