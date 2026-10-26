package events

import "sync"

const subscriberBuffer = 32

// Hub fans events out to the subscribers of each presentation in this process.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan Event]struct{}{}}
}

// Subscribe returns a channel of events for one presentation and a function
// that must be called to stop receiving them.
func (h *Hub) Subscribe(presentationID string) (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)
	h.mu.Lock()
	if h.subs[presentationID] == nil {
		h.subs[presentationID] = map[chan Event]struct{}{}
	}
	h.subs[presentationID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs[presentationID], ch)
			if len(h.subs[presentationID]) == 0 {
				delete(h.subs, presentationID)
			}
			h.mu.Unlock()
		})
	}
}

// Dispatch sends an event to every subscriber of its presentation. A slow
// subscriber whose buffer is full gets a resync instead, so it never blocks
// the others and still ends up up to date.
func (h *Hub) Dispatch(e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[e.PresentationID] {
		deliver(ch, e)
	}
}

// Broadcast sends an event to every subscriber of every presentation.
func (h *Hub) Broadcast(eventType string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for id, set := range h.subs {
		for ch := range set {
			deliver(ch, Event{PresentationID: id, Type: eventType})
		}
	}
}

// SubscriberCount is the number of open streams for a presentation.
func (h *Hub) SubscriberCount(presentationID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[presentationID])
}

func deliver(ch chan Event, e Event) {
	select {
	case ch <- e:
	default:
		// Buffer full: make room for one resync so the client catches up.
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- Event{PresentationID: e.PresentationID, Type: TypeResync}:
		default:
		}
	}
}
