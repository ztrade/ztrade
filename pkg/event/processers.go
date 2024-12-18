package event

import (
	"fmt"
	"time"

	"github.com/ztrade/ztrade/pkg/core"
)

type ErrorCallback func(error)

// Processers processers
type Processers struct {
	handlers []Processer
	bus      *Bus
	errorCb  ErrorCallback
}

// NewProcessers create default Processers
func NewProcessers() *Processers {
	p := new(Processers)
	p.bus = NewBus(1024)
	return p
}

// NewSyncProcessers create sync Processers
func NewSyncProcessers() *Processers {
	p := new(Processers)
	p.bus = NewSyncBus()
	return p
}

func (h *Processers) SetErrorCallback(fn ErrorCallback) {
	h.errorCb = fn

}

func (h *Processers) onError(e *Event) error {
	errInfo := e.Data.Data.(error)
	if h.errorCb == nil {
		return nil
	}
	h.errorCb(errInfo)
	return nil
}

// Adds add processer
func (h *Processers) Adds(ehs ...Processer) (err error) {
	for _, v := range ehs {
		err = h.Add(v)
		if err != nil {
			return
		}
	}
	return
}

// Add add proocesser
func (h *Processers) Add(eh Processer) (err error) {
	h.handlers = append(h.handlers, eh)
	return
}

// Start start all processers
func (h *Processers) Start() (err error) {
	for _, p := range h.handlers {
		err = p.Init(h.bus)
		if err != nil {
			return
		}
	}
	h.bus.Subscribe("Processers", core.EventError, h.onError)
	h.bus.Start()
	for _, p := range h.handlers {
		err = p.Start()
		if err != nil {
			err = fmt.Errorf("start processer %s failed:%s", p.GetName(), err.Error())
			return
		}
	}
	return
}

// Stop stop all processers
func (h *Processers) Stop() (err error) {
	for _, p := range h.handlers {
		err = p.Stop()
		if err != nil {
			return
		}
	}
	return
}

// WaitClose wait for duration after bus is empty,and then close
func (h *Processers) WaitClose(duration time.Duration) {
	time.Sleep(duration)
	h.bus.Close()
}
