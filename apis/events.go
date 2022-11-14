//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"log"
	"math/rand"
	"time"
)

const (
	maxEventsQueueLen = 10000
	flushQueueTimeout = 1 * time.Second // interval to flush the queue
)

type Event struct {
	Property       uint32
	Date           string
	Timestamp      string
	Language       string // "it-IT"
	OSName         string
	OSVersion      string
	Browser        string
	BrowserOther   string
	BrowserVersion string
	DeviceType     string
	Referrer       string // "https://example.com"
	Target         string // "https://example.com"
	Event          string // "pageview", "click", ...
	Text           string // "Add to cart"
	Domain         string // "example.com"
	Path           string // "product/x/y"
	QueryString    string // "x=10"
	Title          string // "Product X"
	User           uint32
	Country        string
	City           string
}

// AddEvent adds an event.
func (apis *APIs) AddEvent(event *Event) {
	apis.eventsQueueMutex.Lock()
	apis.eventsQueue = append(apis.eventsQueue, event)
	var toFlush []*Event
	if len(apis.eventsQueue) == maxEventsQueueLen {
		toFlush = apis.eventsQueue
		apis.eventsQueue = nil
	}
	apis.eventsQueueMutex.Unlock()
	if toFlush != nil {
		go apis.flushEvents(toFlush)
	}
}

// startEventFlusher starts a goroutine that flushes the events queue every
// flushQueueTimeout seconds.
func (apis *APIs) startEventFlusher() {
	ticker := time.NewTicker(flushQueueTimeout)
	go func() {
		for {
			select {
			case <-ticker.C:
				apis.eventsQueueMutex.Lock()
				toFlush := apis.eventsQueue
				apis.eventsQueue = nil
				apis.eventsQueueMutex.Unlock()
				go apis.flushEvents(toFlush)
			}
		}
	}()
}

// flushEvents writes a batch of events to ClickHouse.
func (apis *APIs) flushEvents(events []*Event) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		batch, err := apis.chDB.PrepareBatch(context.Background(), "INSERT INTO `events`\n"+
			"(`property`, `date`, `timestamp`, `language`, `osName`, `osVersion`, `browser`, `browserOther`,"+
			" `browserVersion`, `deviceType`, `referrer`, `target`, `event`, `text`, `domain`, `path`,"+
			" `queryString`, `title`, `user`, `country`, `city`)")
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		for _, event := range events {
			err := batch.Append(event.Property, event.Date, event.Timestamp, event.Language, event.OSName,
				event.OSVersion, event.Browser, event.BrowserOther, event.BrowserVersion, event.DeviceType,
				event.Referrer, event.Target, event.Event, event.Text, event.Domain, event.Path, event.QueryString,
				event.Title, event.User, event.Country, event.City)
			if err != nil {
				log.Printf("[error] cannot log events: %s", err)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				continue RETRY
			}
		}
		err = batch.Send()
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
	}
}
