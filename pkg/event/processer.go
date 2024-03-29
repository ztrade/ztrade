package event

// Processer handler of event
type Processer interface {
	Init(*Bus) error
	GetName() string

	Start() error
	Stop() error
}

// BaseProcesser basic processer
type BaseProcesser struct {
	Bus  *Bus
	Name string
}

// NewBaseProcesser constructor
func NewBaseProcesser(name string) *BaseProcesser {
	bp := new(BaseProcesser)
	bp.Name = name
	return bp
}

// Subscribe event
func (b *BaseProcesser) Subscribe(sub string, cb ProcessCall) (err error) {
	b.Bus.Subscribe(b.Name, sub, cb)
	return
}

// Send send event
func (b *BaseProcesser) Send(name, strType string, data interface{}) {
	b.Bus.Send(NewEvent(name, strType, b.Name, data, nil))
}

// SendExtra send event with extra info
func (b *BaseProcesser) SendWithExtra(name, strType string, data, extra interface{}) {
	b.Bus.Send(NewEvent(name, strType, b.Name, data, extra))
}

// Init call before start
func (b *BaseProcesser) Init(bus *Bus) (err error) {
	b.Bus = bus
	return
}

// Start start the processer
func (b *BaseProcesser) Start() (err error) {
	return
}

// Stop stop the processer
func (b *BaseProcesser) Stop() (err error) {
	return
}

// GetName return the processer name
func (b *BaseProcesser) GetName() string {
	return b.Name
}

// CreateEvent create new event
func (b *BaseProcesser) CreateEvent(name, strType string, data interface{}) *Event {
	return NewEvent(name, strType, b.Name, data, nil)
}
