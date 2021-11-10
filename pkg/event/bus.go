package event

import (
	"fmt"
	"sync"

	jsoniter "github.com/json-iterator/go"

	log "github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ProcessCall callback to process event
type ProcessCall func(e Event) error

type ProcessCallInfo struct {
	Cb   ProcessCall
	Name string
}
type ProcessList []ProcessCallInfo

// Bus event bus
type Bus struct {
	chs        map[string]chan *Event
	chsMutex   sync.Mutex
	cache      int
	procs      map[string]ProcessList
	procsMutex sync.RWMutex
}

func NewBus(cache int) *Bus {
	b := new(Bus)
	b.cache = cache
	b.chs = make(map[string]chan *Event)
	b.procs = make(map[string]ProcessList)
	return b
}

func (b *Bus) runProc(sub string, ch chan *Event) (err error) {
	log.Debug("Bus runProc of ", sub)
	if ch == nil {
		err = fmt.Errorf("no such event channel: %s", sub)
		return
	}
	b.procsMutex.RLock()
	procs := b.procs[sub]
	b.procsMutex.RUnlock()
	for e := range ch {
		event := *e
		for _, p := range procs {
			err = p.Cb(event)
			if err != nil {
				// b.Send(NewErrorEvent(err.Error(), p.Name))
				log.Errorf("subscribe %s process error: %s", sub, err.Error())
				continue
			}
		}
	}
	return
}

// Subscribe event
func (b *Bus) Subscribe(from, sub string, cb ProcessCall) (err error) {
	b.procsMutex.Lock()
	pi := ProcessCallInfo{Cb: cb, Name: from}
	_, ok := b.procs[sub]
	if !ok {
		b.procs[sub] = ProcessList{pi}
	} else {
		b.procs[sub] = append(b.procs[sub], pi)
	}
	b.procsMutex.Unlock()
	return
}

func (b *Bus) Send(e *Event) (err error) {
	typ := e.GetType()
	_, ok := b.procs[typ]
	if !ok {
		log.Warnf("Send %s event,but no subscribers, skip", e.GetType())
		return
	}
	chs := b.chs[typ]
	chs <- e
	return
}

func (b *Bus) WaitEmpty() {
	// time.Sleep(time.Millisecond)
}

func (b *Bus) Close() {

}
func (b *Bus) Start() {
	for k := range b.procs {
		ch := make(chan *Event, b.cache)
		b.chs[k] = ch
		go b.runProc(k, ch)
	}
}
