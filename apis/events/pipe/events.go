//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package pipe

// Event represents an event.
type Event struct {
	ID          int
	AnonymousId string
	Connection  int
	Endpoint    int
	Err         error
}

// Channel represents a pair of Go channels used in the events pipe, which is
// used to communicate between an event producer and a consumer. The Events
// channel is used by the producer to send events to the consumer, and the Done
// channel is used by the consumer to send confirmations back to the producer.
//
// The producer closes the Events channel when it has finished sending all
// events, and the consumer closes the Done channel when it has sent all
// confirmations.
type Channel struct {
	Events <-chan *Event
	Done   chan<- *Event
}
