package vm

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	scriptevent "github.com/thrasher-corp/gocryptotrader/database/repository/script"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/gct"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/loader"
	"github.com/thrasher-corp/gocryptotrader/gctscript/wrappers/validator"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/volatiletech/null"
)

// NewVM attempts to create a new Virtual Machine firstly from pool
func (g *GctScriptManager) NewVM() *VM {
	if !g.IsRunning() {
		log.Errorln(log.GCTScriptMgr, Error{
			Action: "NewVM",
			Cause:  ErrScriptingDisabled,
		})
		return nil
	}
	newUUID, err := uuid.NewV4()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, Error{Action: "New: UUID", Cause: err})
		return nil
	}

	if g.config.Verbose {
		log.Debugln(log.GCTScriptMgr, "New GCTScript VM created")
	}

	s, ok := pool.Get().(*tengo.Script)
	if !ok {
		log.Errorln(log.GCTScriptMgr, Error{
			Action: "NewVM",
			Cause:  common.GetTypeAssertError("*tengo.Script", pool),
		})
		return nil
	}

	return &VM{
		ID:         newUUID,
		Script:     s,
		config:     g.config,
		unregister: func() error { return g.RemoveVM(newUUID) },
	}
}

// SetDefaultScriptOutput sets default output file for scripts
func SetDefaultScriptOutput() {
	loader.SetDefaultScriptOutput(filepath.Join(ScriptPath, "output"))
}

// Load parses and creates a new instance of tengo script vm
func (vm *VM) Load(file string) error {
	if vm == nil {
		return ErrNoVMLoaded
	}

	if filepath.Ext(file) != common.GctExt {
		file += common.GctExt
	}

	if vm.config.Verbose {
		log.Debugf(log.GCTScriptMgr, "Loading script: %s ID: %v", vm.ShortName(), vm.ID)
	}

	code, err := os.ReadFile(file)
	if err != nil {
		return &Error{Action: "Load: ReadFile", Script: file, Cause: err}
	}

	vm.File = file
	vm.Path = filepath.Dir(file)
	vm.Script = tengo.NewScript(code)

	scriptCtx := &gct.Context{}
	scriptCtx.Value = map[string]tengo.Object{
		"script": &tengo.String{Value: vm.ShortName() + "-" + vm.ID.String()},
	}

	err = vm.Script.Add("ctx", scriptCtx)
	if err != nil {
		return err
	}

	vm.Script.SetImports(loader.GetModuleMap())
	vm.Hash = vm.getHash()

	if vm.config.AllowImports {
		if vm.config.Verbose {
			log.Debugf(log.GCTScriptMgr, "File imports enabled for vm: %v", vm.ID)
		}
		vm.Script.EnableFileImport(true)
	}
	vm.event(StatusSuccess, TypeLoad)
	return nil
}

// Compile compiles to byte code loaded copy of vm script
func (vm *VM) Compile() (err error) {
	vm.Compiled, err = vm.Script.Compile()
	return err
}

// RunCtx runs compiled byte code with context.Context support.
func (vm *VM) RunCtx() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.config.ScriptTimeout)
	defer cancel()

	if vm.config.Verbose {
		log.Debugf(log.GCTScriptMgr,
			"Running script: %s ID: %v",
			vm.ShortName(),
			vm.ID)
	}

	err = vm.Compiled.RunContext(ctx)
	if err != nil {
		vm.event(StatusFailure, TypeExecute)
		return Error{Action: "RunCtx", Cause: err}
	}
	vm.event(StatusSuccess, TypeExecute)
	return nil
}

// CompileAndRun Compile and Run script with support for task running
func (vm *VM) CompileAndRun() {
	if vm == nil {
		return
	}
	err := vm.Compile()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, err)
		err = vm.unregister()
		if err != nil {
			log.Errorln(log.GCTScriptMgr, err)
		}
		return
	}

	err = vm.RunCtx()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, err)
		err = vm.unregister()
		if err != nil {
			log.Errorln(log.GCTScriptMgr, err)
		}
		return
	}
	if vm.Compiled.Get("timer").String() != "" {
		vm.T, err = time.ParseDuration(vm.Compiled.Get("timer").String())
		if err != nil {
			log.Errorln(log.GCTScriptMgr, err)
			err = vm.Shutdown()
			if err != nil {
				log.Errorln(log.GCTScriptMgr, err)
			}
			return
		}
		if vm.T > 0 {
			vm.runner()
			return
		}

		if vm.T < 0 {
			log.Errorln(log.GCTScriptMgr, "Repeat timer cannot be under 1 nano second")
		}
	}
	err = vm.Shutdown()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, err)
	}
}

// Shutdown shuts down current VM
func (vm *VM) Shutdown() error {
	if vm == nil {
		return ErrNoVMLoaded
	}
	if vm.S != nil {
		close(vm.S)
	}
	if vm.config.Verbose {
		log.Debugf(log.GCTScriptMgr, "Shutting down script: %s ID: %v", vm.ShortName(), vm.ID)
	}
	vm.Script = nil
	pool.Put(vm.Script)
	vm.event(StatusSuccess, TypeStop)
	return vm.unregister()
}

// Read contents of script back and create script event
func (vm *VM) Read() ([]byte, error) {
	vm.event(StatusSuccess, TypeRead)
	return vm.read()
}

// read contents of script back
func (vm *VM) read() ([]byte, error) {
	if vm.config.Verbose {
		log.Debugf(log.GCTScriptMgr, "Read script: %s ID: %v", vm.ShortName(), vm.ID)
	}
	return os.ReadFile(vm.File)
}

// ShortName returns short (just filename.extension) of running script
func (vm *VM) ShortName() string {
	return filepath.Base(vm.File)
}

func (vm *VM) event(status, executionType string) {
	if validator.IsTestExecution.Load() == true {
		return
	}

	var data null.Bytes
	if executionType == TypeLoad {
		scriptData, err := vm.scriptData()
		if err != nil {
			log.Errorf(log.GCTScriptMgr, "Failed to retrieve scriptData: %v", err)
		}
		data.SetValid(scriptData)
	}
	scriptevent.Event(vm.getHash(), vm.ShortName(), vm.Path, data, executionType, status, time.Now())
}

func (vm *VM) scriptData() ([]byte, error) {
	contents, err := vm.read()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, err := w.Create(vm.ShortName())
	if err != nil {
		return nil, err
	}
	_, err = f.Write(contents)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (vm *VM) getHash() string {
	if vm.Hash != "" {
		return vm.Hash
	}
	contents, err := vm.read()
	if err != nil {
		log.Errorln(log.GCTScriptMgr, err)
	}
	contents = append(contents, vm.ShortName()...)
	s := sha256.Sum256(contents)
	return hex.EncodeToString(s[:])
}

func (vmc *vmscount) add() {
	atomic.AddUint64((*uint64)(vmc), 1)
}

func (vmc *vmscount) remove() {
	atomic.AddUint64((*uint64)(vmc), ^uint64(0))
}

// Len() returns current length vmscount
func (vmc *vmscount) Len() uint64 {
	return atomic.LoadUint64((*uint64)(vmc))
}
