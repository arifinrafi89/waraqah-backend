// Package sse is the in-process broker behind the live endpoints (BACKEND_PLAN.md section 9).
// Topics are per user: inbox:<uid>, sales:<uid>, sales:moderators, notifications:<uid>.
package sse

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// PingInterval keeps proxies from closing an idle stream.
const PingInterval = 25 * time.Second

// Broker fans published events out to the subscribers of a topic.
type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{}
	seq  atomic.Int64
}

// NewBroker makes an empty broker.
func NewBroker() *Broker { return &Broker{subs: map[string]map[chan []byte]struct{}{}} }

// Publish sends payload (any JSON value) to everyone subscribed to topic. A slow subscriber
// misses events instead of blocking the publisher; the app only treats events as "reload".
func (b *Broker) Publish(topic string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[topic] {
		select {
		case ch <- data:
		default:
		}
	}
}

// PublishSeq is Publish with a per-process "seq" counter added to the fields (inbox and sales events).
func (b *Broker) PublishSeq(topic string, fields map[string]any) {
	out := make(map[string]any, len(fields)+1)
	for k, v := range fields {
		out[k] = v
	}
	out["seq"] = b.seq.Add(1)
	b.Publish(topic, out)
}

// Subscribe returns a channel of events for topic and a function that ends the subscription.
func (b *Broker) Subscribe(topic string) (<-chan []byte, func()) {
	ch := make(chan []byte, 16)
	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = map[chan []byte]struct{}{}
	}
	b.subs[topic][ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs[topic], ch)
		if len(b.subs[topic]) == 0 {
			delete(b.subs, topic)
		}
		b.mu.Unlock()
	}
}

// Subscribers counts the open subscriptions on a topic (for tests and diagnostics).
func (b *Broker) Subscribers(topic string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[topic])
}

// Serve streams a topic to one client until the client goes away: one `data: {json}` event
// per message and a `: ping` comment every ping interval.
func (b *Broker) Serve(w http.ResponseWriter, r *http.Request, topic string) {
	b.ServeEvery(w, r, topic, PingInterval)
}

// ServeEvery is Serve with a custom ping interval (tests use a short one).
func (b *Broker) ServeEvery(w http.ResponseWriter, r *http.Request, topic string, ping time.Duration) {
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		return
	}
	events, cancel := b.Subscribe(topic)
	defer cancel()
	ticker := time.NewTicker(ping)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-events:
			if _, err := w.Write(append(append([]byte("data: "), data...), '\n', '\n')); err != nil {
				return
			}
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}
