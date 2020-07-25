package event

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/ztrade/ztrade/pkg/define"

	. "github.com/ThreeDotsLabs/watermill/message"
	log "github.com/sirupsen/logrus"
)

// ProcessCall callback to process event
type ProcessCall func(e Event) error

// Bus event bus
type Bus struct {
	Sub Subscriber
	Pub Publisher
}

func NewBus(pub Publisher, sub Subscriber) *Bus {
	b := new(Bus)
	b.Sub = sub
	b.Pub = pub
	return b
}

// Subscribe event
func (b *Bus) Subscribe(sub string, cb ProcessCall) (err error) {
	typ, ok := define.EventTypes[sub]
	msgs, err := b.Sub.Subscribe(context.Background(), sub)
	if err != nil {
		log.Error("subscribe %s failed: %s", sub, err.Error())
		return
	}
	go func() {
		var err error
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
			err = cb(e)
			if err != nil {
				log.Errorf("subscribe %s process error: %s", sub, err.Error())
				continue
			}
			msg.Ack()
		}
	}()
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
	time.Sleep(time.Millisecond)
}

func (b *Bus) Close() {
	b.Sub.Close()
	b.Pub.Close()
}
