package engine

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	log "github.com/thrasher-/gocryptotrader/logger"
	ntpclient "github.com/thrasher-/gocryptotrader/ntpclient"
)

// vars related to the NTP manager
var (
	NTPCheckInterval = time.Second * 30
	NTPRetryLimit    = 3
	errNTPDisabled   = errors.New("ntp client disabled")
)

// ntpManager starts the NTP manager
type ntpManager struct {
	started       int32
	stopped       int32
	inititalCheck bool
	shutdown      chan struct{}
}

func (n *ntpManager) Started() bool {
	return atomic.LoadInt32(&n.started) == 1
}

func (n *ntpManager) Start() (err error) {
	if atomic.AddInt32(&n.started, 1) != 1 {
		return errors.New("NTP manager already started")
	}

	var disable bool
	defer func() {
		if err != nil || disable {
			atomic.CompareAndSwapInt32(&n.started, 1, 0)
		}
	}()

	log.Debugln(log.SubSystemTimeMgr, "NTP manager starting...")
	if Bot.Config.NTPClient.Level == 0 {
		// Initial NTP check (prompts user on how we should proceed)
		n.inititalCheck = true

		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
		for i := 0; i < NTPRetryLimit; i++ {
			err = n.processTime()
			switch err {
			case nil:
				break
			case errNTPDisabled:
				log.Debugln(log.SubSystemTimeMgr, "NTP manager: User disabled NTP prompts. Exiting.")
				disable = true
				err = nil
				return
			default:
				if i == NTPRetryLimit-1 {
					return err
				}
			}
		}
	}
	n.shutdown = make(chan struct{})
	go n.run()
	log.Debugln(log.SubSystemTimeMgr, "NTP manager started.")
	return nil
}

func (n *ntpManager) Stop() error {
	if atomic.LoadInt32(&n.started) == 0 {
		return errors.New("NTP manager not started")
	}

	if atomic.AddInt32(&n.stopped, 1) != 1 {
		return errors.New("NTP manager is already stopped")
	}

	close(n.shutdown)
	log.Debugln(log.SubSystemTimeMgr, "NTP manager shutting down...")
	return nil
}

func (n *ntpManager) run() {
	t := time.NewTicker(NTPCheckInterval)
	defer func() {
		t.Stop()
		atomic.CompareAndSwapInt32(&n.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&n.started, 1, 0)
		log.Debugln(log.SubSystemTimeMgr, "NTP manager shutdown.")
	}()

	for {
		select {
		case <-n.shutdown:
			return
		case <-t.C:
			n.processTime()
			if Bot.Config.NTPClient.Level == 0 {
				close(n.shutdown)
			}
		}
	}
}

func (n *ntpManager) FetchNTPTime() (time.Time, error) {
	return ntpclient.NTPClient(Bot.Config.NTPClient.Pool)
}

func (n *ntpManager) processTime() error {
	NTPTime, err := n.FetchNTPTime()
	if err != nil {
		return err
	}

	currentTime := time.Now()
	NTPcurrentTimeDifference := NTPTime.Sub(currentTime)
	configNTPTime := *Bot.Config.NTPClient.AllowedDifference
	configNTPNegativeTime := (*Bot.Config.NTPClient.AllowedNegativeDifference - (*Bot.Config.NTPClient.AllowedNegativeDifference * 2))
	if NTPcurrentTimeDifference > configNTPTime || NTPcurrentTimeDifference < configNTPNegativeTime {
		log.Warnf(log.SubSystemTimeMgr, "NTP manager: Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v\n", NTPTime, currentTime, NTPcurrentTimeDifference, configNTPTime, configNTPNegativeTime)
		if n.inititalCheck {
			n.inititalCheck = false
			disable, err := Bot.Config.DisableNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to disable NTP check: %s", err)
			}
			log.Infoln(log.SubSystemTimeMgr, disable)
			if Bot.Config.NTPClient.Level == -1 {
				return errNTPDisabled
			}
		}
	}
	return nil
}
