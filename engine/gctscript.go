package engine

import (
	"fmt"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/gctscript/vm"

	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const name = "gctscript"

type gctScriptManager struct {
	running  atomic.Value
	shutdown chan struct{}
}

func (g *gctScriptManager) Started() bool {
	return g.running.Load() == true
}

func (g *gctScriptManager) Start() (err error) {
	if g.Started() {
		return fmt.Errorf("%s %s", name, ErrSubSystemAlreadyStarted)
	}
	log.Debugf(log.Global, "%s %s", name, MsgSubSystemStarting)

	g.shutdown = make(chan struct{})

	go g.run()
	return nil
}

func (g *gctScriptManager) Stop() error {
	if !g.Started() {
		return fmt.Errorf("%s %s", name, ErrSubSystemAlreadyStarted)
	}

	log.Debugf(log.Global, "%s %s", name, MsgSubSystemShuttingDown)
	close(g.shutdown)
	err := vm.TemrinateAllVM()
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (g *gctScriptManager) run() {
	log.Debugf(log.GCTScriptMgr, "%s %s", name, MsgSubSystemStarted)

	Bot.ServicesWG.Add(1)
	g.running.Store(true)
	//t := time.NewTicker(time.Microsecond)

	defer func() {
		g.running.Store(false)
		Bot.ServicesWG.Done()
		//t.Stop()
		log.Debugf(log.GCTScriptMgr, "%s %s", name, MsgSubSystemShutdown)
	}()

	for {
		select {
		case <-g.shutdown:
			return

		}
	}
}

func (g *gctScriptManager) scheduler() {
	//err := vm.RunVMTasks()
	//if err != nil {
	//	log.Errorln(log.GCTScriptMgr, err)
	//}
	return
}
