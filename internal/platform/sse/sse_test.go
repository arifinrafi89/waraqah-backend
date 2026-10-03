package sse

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServeSendsEventsPingsAndStopsOnDisconnect(t *testing.T) {
	b := NewBroker()
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.ServeEvery(w, r, "notifications:u1", 30*time.Millisecond)
		close(done)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" || resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("headers: %v", resp.Header)
	}
	// wait for the subscription, then publish
	for i := 0; i < 100 && b.Subscribers("notifications:u1") == 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	b.Publish("notifications:u1", map[string]int{"unread": 3})
	b.Publish("notifications:other", map[string]int{"unread": 9}) // another topic: must not arrive

	sc := bufio.NewScanner(resp.Body)
	var gotEvent, gotPing bool
	for sc.Scan() && !(gotEvent && gotPing) {
		line := sc.Text()
		switch {
		case line == `data: {"unread":3}`:
			gotEvent = true
		case line == ": ping":
			gotPing = true
		case strings.Contains(line, "9"):
			t.Fatalf("event from another topic: %s", line)
		}
	}
	if !gotEvent || !gotPing {
		t.Fatalf("event=%v ping=%v", gotEvent, gotPing)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after the client left")
	}
	if b.Subscribers("notifications:u1") != 0 {
		t.Error("subscription leaked")
	}
}

func TestPublishSeqIncreases(t *testing.T) {
	b := NewBroker()
	ch, cancel := b.Subscribe("inbox:u1")
	defer cancel()
	b.PublishSeq("inbox:u1", map[string]any{"threadId": "th-1"})
	b.PublishSeq("inbox:u1", map[string]any{"threadId": "th-1"})
	first, second := string(<-ch), string(<-ch)
	if !strings.Contains(first, `"seq":1`) || !strings.Contains(second, `"seq":2`) || !strings.Contains(first, `"threadId":"th-1"`) {
		t.Errorf("%s %s", first, second)
	}
}

func TestSlowSubscriberDoesNotBlockPublisher(t *testing.T) {
	b := NewBroker()
	_, cancel := b.Subscribe("t")
	defer cancel()
	for i := 0; i < 100; i++ {
		b.Publish("t", i) // buffer is 16; the rest are dropped, never blocked
	}
}
