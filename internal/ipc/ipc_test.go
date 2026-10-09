package ipc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got := make(chan Message, 1)
	go Subscribe(ctx, s.Addr(), s.Token(), func(m Message) { got <- m })

	// Publish until the subscriber has registered and received one event.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case m := <-got:
			var data struct{ N int }
			if err := json.Unmarshal(m.Data, &data); err != nil {
				t.Fatal(err)
			}
			if m.Type != "test" || data.N != 7 {
				t.Fatalf("got %+v", m)
			}
			return
		case <-tick.C:
			if err := s.Publish("test", map[string]int{"N": 7}); err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for event")
		}
	}
}

func TestSubscribeRejectsBadToken(t *testing.T) {
	s, err := Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	err = Subscribe(context.Background(), s.Addr(), "wrong", func(Message) {})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want 401", err)
	}
}
