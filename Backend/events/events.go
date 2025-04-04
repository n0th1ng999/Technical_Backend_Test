package events

import (
	"fmt"
	"sync"
)

// Define data struct for events

type EventWalletData struct {
	BetBalance float32
	Wallet     float32
}

// Define a type for event handlers
type EventHandler func(data EventWalletData)


type EventEmitter struct {
	listeners map[string]map[int]EventHandler // Using a map of handlers
	nextID    int
	mu        sync.Mutex
}

// Create a new EventEmitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		listeners: make(map[string]map[int]EventHandler),
		nextID:    0,
	}
}

// Subscribe (listen) to an event and return a handler ID
func (e *EventEmitter) On(event string, handler EventHandler) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeners[event] == nil {
		e.listeners[event] = make(map[int]EventHandler)
	}
	id := e.nextID
	e.listeners[event][id] = handler
	e.nextID++

	return id // Return an ID for unsubscribing
}

// Unsubscribe using the handler ID
func (e *EventEmitter) Off(event string, id int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if handlers, found := e.listeners[event]; found {
		delete(handlers, id) // Remove specific handler
		if len(handlers) == 0 {
			delete(e.listeners, event) // Clean up empty event key
		}
	}
}

// Emit (trigger) an event
func (e *EventEmitter) Emit(event string, data EventWalletData) {
	if handlers, found := e.listeners[event]; found {
		for _, handler := range handlers {
			go handler(data) // Run handlers concurrently
		}
	} else {
		fmt.Println("No listeners for event:", event)
	}
}

// Global event emitter instance
var GlobalEmitter = NewEventEmitter()
