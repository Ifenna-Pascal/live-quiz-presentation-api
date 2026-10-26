package events

import "testing"

func TestHubDeliversOnlyToThatPresentation(t *testing.T) {
	hub := NewHub()
	a, cancelA := hub.Subscribe("a")
	b, cancelB := hub.Subscribe("b")
	defer cancelB()

	hub.Dispatch(Event{PresentationID: "a", Type: TypeVote})

	if got := (<-a).Type; got != TypeVote {
		t.Fatalf("got %q, want vote", got)
	}
	select {
	case e := <-b:
		t.Fatalf("b should get nothing, got %v", e)
	default:
	}

	cancelA()
	cancelA() // safe to call twice
	if n := hub.SubscriberCount("a"); n != 0 {
		t.Fatalf("got %d subscribers after cancel, want 0", n)
	}
}

func TestSlowSubscriberGetsResyncInsteadOfBlocking(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.Subscribe("a")
	defer cancel()

	for i := 0; i < subscriberBuffer+5; i++ {
		hub.Dispatch(Event{PresentationID: "a", Type: TypeVote})
	}
	last := Event{}
	for len(ch) > 0 {
		last = <-ch
	}
	if last.Type != TypeResync {
		t.Fatalf("last event should be resync, got %q", last.Type)
	}
}

func TestBroadcastReachesEveryone(t *testing.T) {
	hub := NewHub()
	a, ca := hub.Subscribe("a")
	b, cb := hub.Subscribe("b")
	defer ca()
	defer cb()
	hub.Broadcast(TypeResync)
	if (<-a).Type != TypeResync || (<-b).Type != TypeResync {
		t.Fatal("everyone should get the broadcast")
	}
}
