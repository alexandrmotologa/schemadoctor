package telemetry

import (
	"encoding/json"
	"sync"
)

// EventBus provides thread-safe Pub/Sub distribution of real-time telemetry events.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

// NewEventBus initializes a new pub/sub event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Subscribe returns a channel that receives newly published event JSON bytes.
func (b *EventBus) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan []byte, 64)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a channel from the active subscribers.
func (b *EventBus) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish broadcasts an event to all subscribers without blocking slow receivers.
func (b *EventBus) Publish(evt *RepairEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.subscribers) == 0 {
		return
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}

	for ch := range b.subscribers {
		select {
		case ch <- payload:
		default:
			// Subscriber buffer full; skip to keep low latency
		}
	}
}
