package event

import (
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	log "github.com/sirupsen/logrus"
)

// Processers processers
type Processers struct {
	handlers []Processer
	bus      *Bus
}

// NewProcessers create default Processers
func NewProcessers() *Processers {
	p := new(Processers)
	l := log.WithField("component", "pubsub")
	logger := watermill.NewStdLoggerWithOut(l.Writer(), false, false)
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{OutputChannelBuffer: 1024, BlockPublishUntilSubscriberAck: true},
		logger,
	)
	p.bus = NewBus(pubSub, pubSub)
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
	// wait for all procs started
	time.Sleep(time.Second * 5)
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
	h.bus.Close()
	return
}

// WaitClose wait for duration after bus is empty,and then close
func (h *Processers) WaitClose(duration time.Duration) {
	time.Sleep(duration)
	h.bus.Close()
}
