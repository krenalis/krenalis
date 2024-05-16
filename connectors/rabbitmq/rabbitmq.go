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

	"github.com/open2b/chichi"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Stream and the UIHandler interfaces.
var _ interface {
	chichi.Stream
	chichi.UIHandler
} = (*RabbitMQ)(nil)

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
	settings   *Settings
	conn       *amqp.Connection
	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

// Close closes the stream.
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

// Receive receives an event from the stream.
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

// Send sends an event to the stream.
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
func (rmq *RabbitMQ) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if rmq.settings != nil {
			s = *rmq.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, rmq.saveValues(ctx, values, false)
	case "test":
		return nil, rmq.saveValues(ctx, values, true)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "URL", Label: "URL", Placeholder: "amqps://user:pass@example.com/vhost", Type: "text", MinLength: 7, MaxLength: 2048},
			&chichi.Input{Name: "Queue", Label: "Queue", Placeholder: "queue-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Values: values,
		Buttons: []chichi.Button{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

type Settings struct {
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

// saveValues saves the user-entered values as settings. If test is true, it
// validates only the values without saving it.
func (rmq *RabbitMQ) saveValues(ctx context.Context, values []byte, test bool) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate URL.
	if n := len(s.URL); n < 7 || n > 2048 {
		return chichi.NewInvalidUIValuesError("URL length in bytes must be in range [7,2048]")
	}
	if _, err := amqp.ParseURI(s.URL); err != nil {
		return chichi.NewInvalidUIValuesError("URL is not a valid RabbitMQ URI")
	}
	// Validate Queue.
	if n := len(s.Queue); n == 0 || n > 255 {
		return chichi.NewInvalidUIValuesError("queue length in bytes must be in range [1,255]")
	}
	if strings.HasPrefix(s.Queue, "amq.") {
		return chichi.NewInvalidUIValuesError("queue names starting with 'amq.' are reserved for internal use by the broker")
	}
	err = rmq.testConnection(ctx)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = rmq.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	rmq.settings = &s
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (rmq *RabbitMQ) testConnection(ctx context.Context) error {
	return rmq.connect(ctx, false) // TO FIX

}
