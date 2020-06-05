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

// Send send event
func (b *BaseProcesser) Send(name, strType string, data Data) {
	b.Bus.Send(NewEvent(name, strType, b.Name, data))
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
func (b *BaseProcesser) CreateEvent(name, strType string, data Data) *Event {
	return NewEvent(name, strType, b.Name, data)
}
