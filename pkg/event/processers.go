package event

import (
	"time"
)

// Processers processers
type Processers struct {
	handlers []Processer
	bus      *Bus
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
	h.bus.Start()
	for _, p := range h.handlers {
		err = p.Start()
		if err != nil {
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
