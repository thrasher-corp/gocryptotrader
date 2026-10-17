package engine

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
)

// ntpReporting reports clock-state changes and warns once when verification stays unavailable.
// sfptpd's separation of observations and alarm changes informed this design: https://github.com/Xilinx-CNS/sfptpd/blob/5ac5b58e8f0d68a802ea9de5f4879498c302406d/src/sfptpd_engine.c#L2032
// A previous clock error stays open across unknown results, and unknown results never count as recovery.
type ntpReporting struct {
	// last is latest conclusive clock state, or ntpUnknown before first one.
	last         ntpClockState
	unknown      bool
	unknownSince time.Duration
	warned       bool
}

type ntpNotice struct {
	warning bool
	message string
}

// update records a check result and returns messages to log.
// It reports clock-state changes, one warning when checks stay unclear, and recovery only after a fresh healthy result.
func (h *ntpReporting) update(observation *ntpObservation, elapsed time.Duration) []ntpNotice {
	if observation.state == ntpUnknown {
		warning := false
		if !h.unknown {
			h.unknown, h.unknownSince, h.warned = true, elapsed, false
		} else {
			if h.warned || elapsed-h.unknownSince < defaultNTPCheckInterval {
				return nil
			}
			// This warning concerns prolonged inability to verify clock accuracy.
			// It is not evidence that computer's clock is incorrect.
			h.warned = true
			warning = true
		}
		reason := "The measurements are not precise enough to judge the configured tolerance. Time servers closer to you, set in ntpclient.pool, give more precise measurements."
		switch {
		case errors.Is(observation.reason, errNTPInsufficientSources):
			details := []string{fmt.Sprintf("%d/%d configured time servers provided usable time measurements.", observation.usable, observation.configured)}
			// Report rate-limit requests and refusals as separate reasons, as ntpd-rs does: https://github.com/pendulum-project/ntpd-rs/blob/46ec9bb4d5b6cb24f814f5543d85b9138afb4cba/ntp-proto/src/source.rs#L715
			// Several reasons may appear together, and none is presented as sole cause.
			for _, cause := range []struct {
				failures ntpFailures
				message  string
			}{
				{ntpRefused, "Some time servers refused further queries. Check your time server configuration."},
				{ntpRateLimited, "Some time servers requested fewer queries. GoCryptoTrader will wait before querying them again."},
				{ntpNameNotFound, "Some configured time server names have no IP address. Check the server names, DNS settings and network connection."},
				{ntpLookupFailed, "Some configured time server names could not be looked up. Check your network connection and DNS settings."},
				{ntpUnreachable, "Not enough time servers responded. Check your connection to the configured time servers."},
				{ntpInvalidReply, "Some time measurements were unusable."},
				{ntpDuplicateAddress, "Some configured entries resolve to the same IP address."},
			} {
				if observation.failures&cause.failures != 0 {
					details = append(details, cause.message)
				}
			}
			reason = strings.Join(details, " ")
		case errors.Is(observation.reason, errNTPDisagreement):
			reason = "The configured time servers disagree. Check your time server configuration."
		}
		message := "GoCryptoTrader couldn't verify whether your computer's clock is accurate. " + reason + " GoCryptoTrader will keep running."
		if warning {
			message = "GoCryptoTrader still couldn't verify whether your computer's clock is accurate. " + reason + " GoCryptoTrader will keep running. Set ntpclient.enabled to -1 to disable these checks."
		}
		return []ntpNotice{{warning: warning, message: message}}
	}
	previous, wasUnknown := h.last, h.unknown
	h.last, h.unknown = observation.state, false
	if observation.state == ntpHealthy {
		if wasUnknown || previous != ntpUnknown && previous != ntpHealthy {
			if observation.configured == 1 {
				return []ntpNotice{{message: "Your computer's clock appears to be within the configured tolerance, based on one configured time server, without independent corroboration."}}
			}
			return []ntpNotice{{message: "GoCryptoTrader verified that your computer's clock is within the configured tolerance."}}
		}
		return nil
	}
	if previous == observation.state {
		return nil
	}
	direction := "ahead of"
	if observation.state == ntpBehind {
		direction = "behind"
	}
	agreement := fmt.Sprintf("based on agreement from %d/%d configured time servers", observation.agreeing, observation.configured)
	if observation.configured == 1 {
		agreement = "based on one configured time server, without independent corroboration"
	}
	return []ntpNotice{{warning: true, message: "Your computer's clock appears to be " + direction + " the correct time, " + agreement + ". Check your automatic time synchronisation settings. GoCryptoTrader will keep running."}}
}

func (m *ntpManager) report(observation *ntpObservation, elapsed time.Duration) {
	notices := m.history.update(observation, elapsed)
	if !m.loggingEnabled {
		return
	}
	for _, notice := range notices {
		if notice.warning {
			log.Warnln(log.TimeMgr, notice.message)
		} else {
			log.Infoln(log.TimeMgr, notice.message)
		}
	}
	log.Debugf(log.TimeMgr, "NTP correction interval [%s, %s], tolerance [-%s, +%s], agreeing/usable/configured %d/%d/%d, reason: %v, diagnostics: %v",
		observation.interval.lower, observation.interval.upper, m.allowedNegativeDifference, m.allowedDifference,
		observation.agreeing, observation.usable, observation.configured, observation.reason, observation.diagnostics)
}
