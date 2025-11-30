package broker

import "sync"

// Broken implement a simple fan-out message broker.
type Broker[T any] struct {
	mu          sync.Mutex
	subscribers map[string]chan T
}

func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		subscribers: make(map[string]chan T),
	}
}

// Subscribe registers a new subscriber with the given name and channel buffer size.
// It returns a receive-only channel that will receive published messages.
func (b *Broker[T]) Subscribe(name string, size int) <-chan T {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan T, size)

	b.subscribers[name] = ch

	return ch
}

// Publish sends a message to all registered subscribers asynchronously.
func (b *Broker[T]) Publish(t T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		go func() { ch <- t }()
	}
}

// Close closes all subscriber channels, signaling that no more messages will be published.
func (b *Broker[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		close(ch)
	}
}
