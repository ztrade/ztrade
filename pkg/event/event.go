package event

import (
	"sync"

	"github.com/ztrade/ztrade/pkg/core"
)

var (
	EventError string = "error"
	eventPool         = sync.Pool{New: func() interface{} {
		return new(Event)
	}}
)

// Event base event
type Event struct {
	Data core.EventData
	Name string
	// Time time.Time
	From string
}

func NewErrorEvent(from, msg string) *Event {
	e := new(Event)
	e.Name = msg
	e.Data.Type = EventError
	e.From = from
	// e.Time = time.Now()
	return e
}

func NewEvent(name, strType, from string, data interface{}, extra interface{}) *Event {
	e := eventPool.Get().(*Event)
	// e := new(Event)
	e.Name = name
	e.Data.Type = strType
	e.From = from
	e.Data.Data = data
	// e.Time = time.Now()
	e.Data.Extra = extra
	return e
}

func releaseEvent(e *Event) {
	e.Data.Data = nil
	e.Data.Extra = nil
	eventPool.Put(e)
}

func (e *Event) GetName() string {
	return e.Name
}

func (e *Event) GetType() string {
	return e.Data.Type
}

// func (e *Event) GetTime() time.Time {
// 	return e.Time
// }

func (e *Event) GetFrom() string {
	return e.From
}

func (e *Event) GetData() interface{} {
	return e.Data.Data
}
func (e *Event) GetExtra() interface{} {
	return e.Data.Extra
}
