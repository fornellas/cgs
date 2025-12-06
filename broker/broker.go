package broker

import (
	"errors"
	"sync"
)

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
func (b *Broker[T]) Publish(t T) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.subscribers) == 0 {
		return errors.New("no subscribers")
	}

	for _, ch := range b.subscribers {
		go func() {
			// We run this concurrently to prevent a single subscriber from blocking the others.
			// As a consequence, when Close() is called, there's a race where we may attempt to
			// send to a closed channel.
			// Adding this recover() here prevents it from panicing, which is expected in this
			// scenario.
			defer func() { recover() }()
			ch <- t
		}()
	}

	return nil
}

// Close closes all subscriber channels, signaling that no more messages will be published.
func (b *Broker[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subscribers {
		close(ch)
	}

	b.subscribers = make(map[string]chan T)
}
