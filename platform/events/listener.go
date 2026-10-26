package events

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Listen receives NOTIFY messages on Channel and dispatches them to the hub.
// It runs until the process exits and reconnects on its own.
func Listen(dbSource string, hub *Hub, logger *logrus.Logger) error {
	listener := pq.NewListener(dbSource, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logger.Warn("events: listener: ", err)
		}
	})
	if err := listener.Listen(Channel); err != nil {
		return err
	}

	go func() {
		ping := time.NewTicker(90 * time.Second)
		defer ping.Stop()
		for {
			select {
			case n := <-listener.Notify:
				if n == nil {
					// Reconnected: events may have been lost meanwhile.
					hub.Broadcast(TypeResync)
					continue
				}
				var e Event
				if err := json.Unmarshal([]byte(n.Extra), &e); err != nil {
					logger.Warn("events: bad payload: ", err)
					continue
				}
				hub.Dispatch(e)
			case <-ping.C:
				go func() { _ = listener.Ping() }()
			}
		}
	}()
	return nil
}
