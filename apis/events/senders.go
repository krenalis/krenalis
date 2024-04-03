//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/open2b/chichi/apis/connectors"
)

// startSenders starts some senders that read from the events channel and write
// to the done channel once the processed events have been sent to the
// destination. It returns a channel that, when closed, stops the senders.
func startSenders(events <-chan *processedEvent, done chan<- *processedEvent, st *eventsState) chan<- struct{} {

	stop := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop
		cancel()
	}()

	// Start the workers which send events.
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case event := <-events:
					c := event.action.Connection()
					if !c.Enabled || c.Workspace().Warehouse == nil {
						done <- event
						continue
					}
					// TODO(Gianluca): use correct error handling here.
					app := st.connectors.App(c)
					req, err := app.EventRequest(ctx, event.eventType, event.inEvent, event.extra, event.extraSchema, false)
					if err != nil && err != connectors.ErrEventTypeNotExist {
						if err != context.Canceled {
							slog.Error("cannot send event", "err", err)
						}
						continue
					}
					if req.URL != "https://example.com/" {
						res, err := app.SendEvent(ctx, req)
						if err != nil {
							if err != context.Canceled {
								slog.Error("cannot send event", "err", err)
							}
							continue
						}
						if res.StatusCode < 200 || res.StatusCode >= 300 {
							slog.Error(fmt.Sprintf("%q returned status code %d", req.URL, res.StatusCode))
							continue
						}
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
