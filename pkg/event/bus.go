package event

import (
	"context"
	"reflect"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/ztrade/ztrade/pkg/define"

	. "github.com/ThreeDotsLabs/watermill/message"
	log "github.com/sirupsen/logrus"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// ProcessCall callback to process event
type ProcessCall func(e Event) error
type ProcessList []ProcessCall

// Bus event bus
type Bus struct {
	Sub        Subscriber
	Pub        Publisher
	procs      map[string]ProcessList
	procsMutex sync.RWMutex
}

func NewBus(pub Publisher, sub Subscriber) *Bus {
	b := new(Bus)
	b.Sub = sub
	b.Pub = pub
	b.procs = make(map[string]ProcessList)
	return b
}

func (b *Bus) runProc(sub string) (err error) {
	log.Debug("Bus runProc of ", sub)
	typ, ok := define.EventTypes[sub]
	msgs, err := b.Sub.Subscribe(context.Background(), sub)
	if err != nil {
		log.Error("subscribe %s failed: %s", sub, err.Error())
		return
	}
	b.procsMutex.RLock()
	procs := b.procs[sub]
	b.procsMutex.RUnlock()
	for msg := range msgs {
		var e Event
		if ok {
			e.Data = reflect.New(typ).Interface()
		} else {
			e.Data = map[string]interface{}{}
		}
		err = json.Unmarshal(msg.Payload, &e)
		if err != nil {
			log.Errorf("subscribe %s error: %s", sub, err.Error())

			continue
		}
		// var wg sync.WaitGroup
		// wg.Add(len(procs))
		// fmt.Printf("proc:%s\n", string(msg.Payload))
		for _, cb := range procs {
			// go func() {
			err = cb(e)
			if err != nil {
				log.Errorf("subscribe %s process error: %s", sub, err.Error())
				continue
			}
			// wg.Done()
			// }()
		}
		// wg.Wait()
		// fmt.Printf("proc finished:%s\n", string(msg.Payload))
		msg.Ack()

	}
	return
}

// Subscribe event
func (b *Bus) Subscribe(sub string, cb ProcessCall) (err error) {
	b.procsMutex.Lock()
	_, ok := b.procs[sub]
	if !ok {
		b.procs[sub] = ProcessList{cb}
	} else {
		b.procs[sub] = append(b.procs[sub], cb)
	}
	b.procsMutex.Unlock()
	return
}

func (b *Bus) Send(e *Event) (err error) {
	buf, err := json.Marshal(e)
	if err != nil {
		return
	}
	err = b.Pub.Publish(e.GetType(), NewMessage("", buf))
	return
}

func (b *Bus) WaitEmpty() {
	// time.Sleep(time.Millisecond)
}

func (b *Bus) Close() {
	b.Sub.Close()
	b.Pub.Close()
}
func (b *Bus) Start() {
	for k := range b.procs {
		go b.runProc(k)
	}
}
