package engine

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

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

	confNTP := Bot.Config.GetNTPClient()

	if confNTP.Level == -1 {
		err = errors.New("NTP client disabled")
		return
	}

	confLog := Bot.Config.GetLogging()

	log.Debugln(log.TimeMgr, "NTP manager starting...")
	if confNTP.Level == 0 && *confLog.Enabled {
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
				log.Debugln(log.TimeMgr, "NTP manager: User disabled NTP prompts. Exiting.")
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
	log.Debugln(log.TimeMgr, "NTP manager started.")
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
	log.Debugln(log.TimeMgr, "NTP manager shutting down...")
	return nil
}

func (n *ntpManager) run() {
	t := time.NewTicker(NTPCheckInterval)
	defer func() {
		t.Stop()
		atomic.CompareAndSwapInt32(&n.stopped, 1, 0)
		atomic.CompareAndSwapInt32(&n.started, 1, 0)
		log.Debugln(log.TimeMgr, "NTP manager shutdown.")
	}()

	for {
		select {
		case <-n.shutdown:
			return
		case <-t.C:
			n.processTime()
		}
	}
}

func (n *ntpManager) FetchNTPTime() time.Time {
	return ntpclient.NTPClient(Bot.Config.GetNTPClient().Pool)
}

func (n *ntpManager) processTime() error {
	NTPTime := n.FetchNTPTime()
	currentTime := time.Now()
	NTPcurrentTimeDifference := NTPTime.Sub(currentTime)
	conf := Bot.Config.GetNTPClient()
	configNTPNegativeTime := (*conf.AllowedNegativeDifference - (*conf.AllowedNegativeDifference * 2))
	if NTPcurrentTimeDifference > *conf.AllowedDifference ||
		NTPcurrentTimeDifference < configNTPNegativeTime {
		log.Warnf(log.TimeMgr,
			"NTP manager: Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v\n",
			NTPTime,
			currentTime,
			NTPcurrentTimeDifference,
			conf.AllowedDifference,
			configNTPNegativeTime)
		if n.inititalCheck {
			n.inititalCheck = false
			disable, err := Bot.Config.DisableNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to disable NTP check: %s", err)
			}
			log.Infoln(log.TimeMgr, disable)
			if conf.Level == -1 {
				return errNTPDisabled
			}
		}
	}
	return nil
}
