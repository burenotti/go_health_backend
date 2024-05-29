package domain

import (
	"sync"
	"time"
)

type Event interface {
	Type() string
	PublishedAt() time.Time
}

type NoCopy struct {
	sync.Mutex
}

type Aggregate struct {
	NoCopy
	events []Event
}

func (a *Aggregate) PopEvents() []Event {
	events := a.events
	a.events = make([]Event, 0)
	return events
}

func (a *Aggregate) PushEvent(e Event) {
	a.events = append(a.events, e)
}
