//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/meergo/meergo/core/connectors"
	meergoMetrics "github.com/meergo/meergo/metrics"
)

// startSenders starts some senders that read from the events channel and write
// to the sent channel once the processed events have been sent to the
// destination. It returns a channel that, when closed, stops the senders.
func startSenders(events <-chan *dispatchingEvent, sent chan<- *dispatchingEvent, conns *connectors.Connectors) chan<- struct{} {

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
					if event.request.URL != "https://example.com/" {
						app := conns.App(event.action.Connection())
						res, err := app.SendEvent(ctx, event.request)
						if err != nil {
							if err != context.Canceled {
								slog.Error("cannot send event", "err", err)
							}
							continue
						}
						if res.StatusCode < 200 || res.StatusCode >= 300 {
							slog.Error(fmt.Sprintf("%q returned status code %d", event.request.URL, res.StatusCode))
							continue
						}
					}
					sent <- event
					meergoMetrics.Increment("sender.startSenders.event_written_on_sent_channel", 1)
				case <-stop:
					return
				}
			}
		}()
	}

	return stop
}
