//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"context"
	"log"
)

// startSenders starts some senders which reads from events and writes on done
// when the read events have been sent to the destination.
func startSenders(ctx context.Context, events <-chan *processedEvent, done chan<- *processedEvent, st *eventsState) {

	// Start the workers which send events.
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case event := <-events:
					destination, ok := st.Destination(event.destination)
					if !ok {
						done <- event
						continue
					}
					// TODO(Gianluca): use correct error handling here.
					err := destination.SendEvent(event.inEvent, event.mappedEvent, event.eventType)
					if err != nil {
						log.Printf("cannot send event: %s", err)
						continue
					}
					done <- event
				case <-ctx.Done():
					return
				}
			}
		}()
	}

}
