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

	"chichi"
	"chichi/ui"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI interface.
var _ chichi.UI = (*RabbitMQ)(nil)

func init() {
	chichi.RegisterStream(chichi.StreamInfo{
		Name: "RabbitMQ",
		Icon: icon,
	}, New)
}

// New returns a new RabbitMQ connector instance.
func New(conf *chichi.StreamConfig) (*RabbitMQ, error) {
	c := RabbitMQ{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of RabbitMQ connector")
		}
	}
	return &c, nil
}

type RabbitMQ struct {
	conf       *chichi.StreamConfig
	settings   *settings
	conn       *amqp.Connection
	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

// Close closes the stream. When Close is called, no other calls to connector's
// methods are in progress and no more will be made.
func (rmq *RabbitMQ) Close() error {
	if rmq.conn == nil {
		return nil
	}
	rmq.deliveries = nil
	err := rmq.ch.Close()
	if err2 := rmq.conn.Close(); err == nil {
		err = err2
	}
	rmq.ch = nil
	rmq.conn = nil
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
func (rmq *RabbitMQ) Receive(ctx context.Context) ([]byte, func(), error) {
	err := rmq.connect(ctx, true)
	if err != nil {
		return nil, nil, err
	}
	select {
	case delivery := <-rmq.deliveries:
		tag := delivery.DeliveryTag
		ack := func() {
			_ = rmq.ch.Ack(tag, false)
		}
		return delivery.Body, ack, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

// Send sends an event to the stream. If ack is not nil, connector calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but event is not retained after the ack
// function has been called.
//
// Send can be used by multiple goroutines at the same time.
func (rmq *RabbitMQ) Send(ctx context.Context, event []byte, options chichi.SendOptions, ack func(err error)) error {
	err := rmq.connect(ctx, true)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{Body: event}
	dc, err := rmq.ch.PublishWithDeferredConfirmWithContext(ctx, "", rmq.settings.Queue, false, false, msg)
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
func (rmq *RabbitMQ) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if rmq.settings != nil {
			s = *rmq.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := rmq.ValidateSettings(ctx, values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		err = rmq.conf.SetSettings(ctx, s)
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
func (rmq *RabbitMQ) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
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
	err = rmq.testConnection(ctx)
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
func (rmq *RabbitMQ) connect(ctx context.Context, deliveries bool) (err error) {
	if rmq.conn != nil {
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
			netConn, err = d.DialContext(ctx, network, address)
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
	conn, err := amqp.DialConfig(rmq.settings.URL, config)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if deliveries {
		rmq.deliveries, err = ch.Consume(rmq.settings.Queue, "", false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return err
		}
	}
	rmq.conn = conn
	rmq.ch = ch
	go func() {
		select {
		case <-ctx.Done():
			_ = rmq.ch.Close()
			_ = rmq.conn.Close()
			rmq.ch = nil
			rmq.conn = nil
		}
	}()
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (rmq *RabbitMQ) testConnection(ctx context.Context) error {
	return rmq.connect(ctx, false) // TO FIX

}
