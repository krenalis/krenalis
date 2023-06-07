//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package rabbitmq implements the RabbitMQ connector.
// (https://www.rabbitmq.com/documentation.html)
package rabbitmq

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"

	"chichi/connector"
	"chichi/connector/ui"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI interface.
var _ connector.UI = (*connection)(nil)

func init() {
	connector.RegisterStream(connector.Stream{
		Name: "RabbitMQ",
		Icon: icon,
	}, open)
}

// open opens a RabbitMQ connection and returns it.
func open(ctx context.Context, conf *connector.StreamConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of RabbitMQ connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx        context.Context
	conf       *connector.StreamConfig
	settings   *settings
	conn       *amqp.Connection
	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

// Close closes the stream. When Close is called, no other calls to connection
// methods are in progress and no more will be made.
func (c *connection) Close() error {
	if c.conn == nil {
		return nil
	}
	c.deliveries = nil
	err := c.ch.Close()
	if err2 := c.conn.Close(); err == nil {
		err = err2
	}
	c.ch = nil
	c.conn = nil
	return err
}

// Receive receives an event from the stream. Callers call the ack function to
// notify that the event has been received. The connector resends the event if
// not acknowledged.
//
// Caller do not modify the event data, even temporarily, and event is not
// retained after the ack function has been called.
//
// Receive can be used by multiple goroutines at the same time.
func (c *connection) Receive() ([]byte, func(), error) {
	err := c.connect(true)
	if err != nil {
		return nil, nil, err
	}
	select {
	case delivery := <-c.deliveries:
		tag := delivery.DeliveryTag
		ack := func() {
			_ = c.ch.Ack(tag, false)
		}
		return delivery.Body, ack, nil
	case <-c.ctx.Done():
		return nil, nil, c.ctx.Err()
	}
}

// Send sends an event to the stream. If ack is not nil, connector calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but event is not retained after the ack
// function has been called.
//
// Send can be used by multiple goroutines at the same time.
func (c *connection) Send(event []byte, options connector.SendOptions, ack func(err error)) error {
	err := c.connect(true)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{Body: event}
	dc, err := c.ch.PublishWithDeferredConfirmWithContext(c.ctx, "", c.settings.Queue, false, false, msg)
	if err != nil {
		return err
	}
	if ack != nil {
		go func() {
			if ok := dc.Wait(); ok {
				ack(nil)
			} else {
				ack(errors.New("event not received"))
			}
		}()
	}
	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := c.ValidateSettings(values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		err = c.conf.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "url", Label: "URL", Placeholder: "amqps://user:pass@example.com/vhost", Type: "text", MinLength: 7, MaxLength: 2048},
			&ui.Input{Name: "queue", Label: "Queue", Placeholder: "queue-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate URL.
	if n := len(s.URL); n < 7 || n > 2048 {
		return nil, ui.Errorf("URL length in bytes must be in range [7,2048]")
	}
	if _, err := amqp.ParseURI(s.URL); err != nil {
		return nil, ui.Errorf("URL is not a valid RabbitMQ URI")
	}
	// Validate Queue.
	if n := len(s.Queue); n == 0 || n > 255 {
		return nil, ui.Errorf("queue length in bytes must be in range [1,255]")
	}
	if strings.HasPrefix(s.Queue, "amq.") {
		return nil, ui.Errorf("queue names starting with 'amq.' are reserved for internal use by the broker")
	}
	err = c.testConnection()
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

type settings struct {
	URL   string
	Queue string
}

const defaultConnectionTimeout = 30 * time.Second

// connect establishes a connection to RabbitMQ. If deliveries is true, it also
// sets the deliveries channel.
func (c *connection) connect(deliveries bool) (err error) {
	if c.conn != nil {
		return nil
	}
	var netConn net.Conn
	defer func() {
		if err != nil && netConn != nil {
			_ = netConn.Close()
		}
	}()
	config := amqp.Config{
		Dial: func(network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: defaultConnectionTimeout}
			netConn, err = d.DialContext(c.ctx, network, address)
			if err != nil {
				return nil, err
			}
			err = netConn.SetDeadline(time.Now().Add(defaultConnectionTimeout))
			if err != nil {
				return nil, err
			}
			return netConn, nil
		},
	}
	conn, err := amqp.DialConfig(c.settings.URL, config)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if deliveries {
		c.deliveries, err = ch.Consume(c.settings.Queue, "", false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return err
		}
	}
	c.conn = conn
	c.ch = ch
	go func() {
		select {
		case <-c.ctx.Done():
			_ = c.ch.Close()
			_ = c.conn.Close()
			c.ch = nil
			c.conn = nil
		}
	}()
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (c *connection) testConnection() error {
	ctx, cancel := context.WithCancel(c.ctx)
	c.ctx = ctx
	defer cancel()
	return c.connect(false)
}
