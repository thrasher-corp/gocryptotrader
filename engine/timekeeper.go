package engine

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/log"
	ntpclient "github.com/thrasher-corp/gocryptotrader/ntpclient"
)

// vars related to the NTP manager
var (
	NTPCheckInterval = time.Second * 30
	NTPRetryLimit    = 3
	errNTPDisabled   = errors.New("ntp client disabled")
)

// ntpManager starts the NTP manager
type ntpManager struct {
	started      int32
	initialCheck bool
	shutdown     chan struct{}
}

func (n *ntpManager) Started() bool {
	return atomic.LoadInt32(&n.started) == 1
}

func (n *ntpManager) Start() error {
	if !atomic.CompareAndSwapInt32(&n.started, 0, 1) {
		return fmt.Errorf("NTP manager %w", subsystem.ErrSubSystemAlreadyStarted)
	}

	if Bot.Config.NTPClient.Level == -1 {
		atomic.CompareAndSwapInt32(&n.started, 1, 0)
		return errors.New("NTP client disabled")
	}

	log.Debugln(log.TimeMgr, "NTP manager starting...")
	if Bot.Config.NTPClient.Level == 0 && *Bot.Config.Logging.Enabled {
		// Initial NTP check (prompts user on how we should proceed)
		n.initialCheck = true
		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
	check:
		for i := 0; i < NTPRetryLimit; i++ {
			err := n.processTime()
			switch err {
			case nil:
				break check
			case errNTPDisabled:
				log.Debugln(log.TimeMgr, "NTP manager: User disabled NTP prompts. Exiting.")
				atomic.CompareAndSwapInt32(&n.started, 1, 0)
				return nil
			default:
				if i == NTPRetryLimit-1 {
					return err
				}
			}
		}
	}
	n.shutdown = make(chan struct{})
	go n.run()
	log.Debugln(log.TimeMgr, "NTP manager started.")
	return nil
}

func (n *ntpManager) Stop() error {
	if atomic.LoadInt32(&n.started) == 0 {
		return fmt.Errorf("NTP manager %w", subsystem.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&n.started, 1, 0)
	}()
	log.Debugln(log.TimeMgr, "NTP manager shutting down...")
	close(n.shutdown)
	return nil
}

func (n *ntpManager) run() {
	t := time.NewTicker(NTPCheckInterval)
	defer func() {
		t.Stop()
		log.Debugln(log.TimeMgr, "NTP manager shutdown.")
	}()

	for {
		select {
		case <-n.shutdown:
			return
		case <-t.C:
			err := n.processTime()
			if err != nil {
				log.Error(log.TimeMgr, err)
			}
		}
	}
}

func (n *ntpManager) FetchNTPTime() time.Time {
	return ntpclient.NTPClient(Bot.Config.NTPClient.Pool)
}

func (n *ntpManager) processTime() error {
	NTPTime := n.FetchNTPTime()
	currentTime := time.Now()
	diff := NTPTime.Sub(currentTime)
	configNTPTime := *Bot.Config.NTPClient.AllowedDifference
	negDiff := *Bot.Config.NTPClient.AllowedNegativeDifference
	configNTPNegativeTime := -negDiff
	if diff > configNTPTime || diff < configNTPNegativeTime {
		log.Warnf(log.TimeMgr, "NTP manager: Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v\n",
			NTPTime,
			currentTime,
			diff,
			configNTPTime,
			configNTPNegativeTime)
		if n.initialCheck {
			n.initialCheck = false
			disable, err := Bot.Config.DisableNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to disable NTP check: %s", err)
			}
			log.Infoln(log.TimeMgr, disable)
			if Bot.Config.NTPClient.Level == -1 {
				return errNTPDisabled
			}
		}
	}
	return nil
}
