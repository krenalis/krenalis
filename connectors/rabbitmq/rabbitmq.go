//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package rabbitmq

// This package is the RabbitMQ connector.
// (https://www.rabbitmq.com/documentation.html)

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
var icon []byte

// Make sure it implements the EventStreamConnection interface.
var _ connector.EventStreamConnection = &connection{}

func init() {
	connector.RegisterEventStream("RabbitMQ", newConnection)
}

// newConnection returns a new RabbitMQ connection.
func newConnection(ctx context.Context, conf *connector.EventStreamConfig) (connector.EventStreamConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
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
	settings   *settings
	firehose   connector.Firehose
	conn       *amqp.Connection
	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "RabbitMQ",
		Type: connector.EventStreamType,
		Icon: icon,
	}
}

// Close closes the stream. Must be called if at least one Send or Receive call
// has been made. It cannot be called concurrently with Send and Receive.
func (c *connection) Close() error {
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

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings != nil {
			s = *c.settings
		}
	case "test", "save":
		// Test the connection and save the settings if required.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, err
		}
		// Validate URL.
		if n := len(s.URL); n < 7 || n > 2048 {
			return nil, nil, ui.Errorf("URL length in bytes must be in range [7,2048]")
		}
		if _, err := amqp.ParseURI(s.URL); err != nil {
			return nil, nil, ui.Errorf("URL is not a valid RabbitMQ URI")
		}
		// Validate Queue.
		if n := len(s.Queue); n == 0 || n > 255 {
			return nil, nil, ui.Errorf("queue length in bytes must be in range [1,255]")
		}
		if strings.HasPrefix(s.Queue, "amq.") {
			return nil, nil, ui.Errorf("queue names starting with 'amq.' are reserved for internal use by the broker")
		}
		err = c.testConnection()
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, nil, err
		}
		err = c.firehose.SetSettings(b)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "url", Value: s.URL, Label: "URL", Placeholder: "amqps://user:pass@example.com/vhost", Type: "text", MinLength: 7, MaxLength: 2048},
			&ui.Input{Name: "queue", Value: s.Queue, Label: "Queue", Placeholder: "queue-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
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
