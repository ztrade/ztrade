package event

import (
	"context"
	"reflect"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/ztrade/ztrade/pkg/core"

	"github.com/ThreeDotsLabs/watermill/message"
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
	sub        message.Subscriber
	pub        message.Publisher
	procs      map[string]ProcessList
	procsMutex sync.RWMutex
}

func NewBus(pub message.Publisher, sub message.Subscriber) *Bus {
	b := new(Bus)
	b.sub = sub
	b.pub = pub
	b.procs = make(map[string]ProcessList)
	return b
}

func (b *Bus) runProc(sub string) (err error) {
	log.Debug("Bus runProc of ", sub)
	typ, ok := core.EventTypes[sub]
	msgs, err := b.sub.Subscribe(context.Background(), sub)
	if err != nil {
		log.Errorf("subscribe %s failed: %s", sub, err.Error())
		return
	}
	b.procsMutex.RLock()
	procs := b.procs[sub]
	b.procsMutex.RUnlock()
	var e Event
	for msg := range msgs {
		if ok {
			e.Data = reflect.New(typ).Interface()
		} else {
			e.Data = map[string]interface{}{}
		}
		err = json.Unmarshal(msg.Payload, &e)
		if err != nil {
			log.Errorf("subscribe %s error: %s", sub, err.Error())
			b.Send(NewErrorEvent("unmarshal json:"+err.Error(), "Bus"))
			continue
		}
		if in, _ := e.Data.(core.Initer); in != nil {
			err = in.Init()
			if err != nil {
				log.Errorf("init data %s error: %s", sub, err.Error())
				b.Send(NewErrorEvent("init data error:"+err.Error(), "Bus"))
				continue
			}
		}
		for _, p := range procs {
			err = p.Cb(e)
			if err != nil {
				b.Send(NewErrorEvent(err.Error(), p.Name))
				log.Errorf("subscribe %s process error: %s", sub, err.Error())
				continue
			}
		}
		msg.Ack()
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
	buf, err := json.Marshal(e)
	if err != nil {
		return
	}
	err = b.pub.Publish(e.GetType(), message.NewMessage("", buf))
	return
}

func (b *Bus) WaitEmpty() {
	// time.Sleep(time.Millisecond)
}

func (b *Bus) Close() {
	b.sub.Close()
	b.pub.Close()
}
func (b *Bus) Start() {
	for k := range b.procs {
		go b.runProc(k)
	}
}
