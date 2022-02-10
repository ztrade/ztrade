package event

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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
	syncMode   bool
	chs        map[string]chan *Event
	chsMutex   sync.Mutex
	cache      int
	procs      map[string]ProcessList
	procsMutex sync.RWMutex

	processEvent  int64
	lastEventTime time.Time
}

func NewBus(cache int) *Bus {
	b := new(Bus)
	b.cache = cache
	b.chs = make(map[string]chan *Event)
	b.procs = make(map[string]ProcessList)
	return b
}

func NewSyncBus() *Bus {
	b := new(Bus)
	b.syncMode = true
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
		atomic.AddInt64(&b.processEvent, -1)
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
	procs, ok := b.procs[typ]
	if !ok {
		log.Warnf("Send %s event,but no subscribers, skip", e.GetType())
		return
	}
	atomic.AddInt64(&b.processEvent, 1)
	if b.syncMode {
		return b.sendSync(procs, e)
	}

	chs := b.chs[typ]
	b.lastEventTime = time.Now()
	chs <- e
	return
}
func (b *Bus) sendSync(procs ProcessList, e *Event) (err error) {
	event := *e
	for _, p := range procs {
		err = p.Cb(event)
		if err != nil {
			log.Errorf("subscribe %s process error: %s", e.GetType(), err.Error())
			continue
		}
	}
	atomic.AddInt64(&b.processEvent, -1)
	return
}

func (b *Bus) WaitEmpty() {
	//	time.Sleep(time.Millisecond)
	value := atomic.LoadInt64(&b.processEvent)
	for value != 0 {
		time.Sleep(time.Millisecond)
		value = atomic.LoadInt64(&b.processEvent)
	}
}

func (b *Bus) Close() {
	var value int64
	for {
		time.Sleep(time.Nanosecond)
		value = atomic.LoadInt64(&b.processEvent)
		if value != 0 {
			continue
		}
		if time.Since(b.lastEventTime) > time.Second*5 {
			break
		}
	}
}

func (b *Bus) Start() {
	if b.syncMode {
		return
	}
	for k := range b.procs {
		ch := make(chan *Event, b.cache)
		b.chs[k] = ch
		go b.runProc(k, ch)
	}
}
