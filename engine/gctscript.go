package engine

import (
	"fmt"
	"sync/atomic"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

type gctScriptManager struct {
	name     string
	running  atomic.Value
	shutdown chan struct{}
}

func (g *gctScriptManager) Started() bool {
	return g.running.Load() == true
}

func (g *gctScriptManager) Start() (err error) {
	if g.Started() {
		return fmt.Errorf("%s %s", g.name, ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.Global, "%s %s", g.name, MsgSubSystemStarting)

	g.shutdown = make(chan struct{})

	go g.run()
	return nil
}

func (g *gctScriptManager) Stop() error {
	if !g.Started() {
		return fmt.Errorf("%s %s", g.name, ErrSubSystemAlreadyStarted)
	}

	log.Debugf(log.Global, "%s %s", g.name, MsgSubSystemShuttingDown)
	close(g.shutdown)

	return nil
}

func (g *gctScriptManager) run() {
	log.Debugf(log.Global, "%s %s", g.name, MsgSubSystemStarted)

	Bot.ServicesWG.Add(1)
	g.running.Store(true)

	defer func() {
		g.running.Store(false)
		Bot.ServicesWG.Done()
		log.Debugf(log.Global, "%s %s", g.name, MsgSubSystemShutdown)
	}()

	for {
		select {
		case <-g.shutdown:
			return
		}
	}
}
