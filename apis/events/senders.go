//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"log"
)

// startSenders starts some senders that read from the events channel and write
// to the done channel once the processed events have been sent to the
// destination. It returns a channel that, when closed, stops the senders.
func startSenders(events <-chan *processedEvent, done chan<- *processedEvent, st *eventsState) chan<- struct{} {

	stop := make(chan struct{})

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
				case <-stop:
					return
				}
			}
		}()
	}

	return stop
}
