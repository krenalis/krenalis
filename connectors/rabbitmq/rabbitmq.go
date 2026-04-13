// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package rabbitmq provides a connector for RabbitMQ.
// (https://www.rabbitmq.com/docs)
//
// RabbitMQ is a trademark of Broadcom, Inc.
// This connector is not affiliated with or endorsed by Broadcom, Inc.
package rabbitmq

import (
	"context"
	_ "embed"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterMessageBroker(connectors.MessageBrokerSpec{
		Code:       "rabbitmq",
		Label:      "RabbitMQ",
		Categories: connectors.CategoryMessageBroker,
		Documentation: connectors.Documentation{
			Source: connectors.RoleDocumentation{
				Summary:  "Import events and users from RabbitMQ",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for RabbitMQ.
func New(env *connectors.MessageBrokerEnv) (*RabbitMQ, error) {
	return &RabbitMQ{env: env}, nil
}

type RabbitMQ struct {
	env        *connectors.MessageBrokerEnv
	conn       *amqp.Connection
	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

// Close closes the message broker.
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

// Receive receives an event from the message broker.
func (rmq *RabbitMQ) Receive(ctx context.Context) ([]byte, func(), error) {
	var s innerSettings
	err := rmq.env.Settings.Load(ctx, &s)
	if err != nil {
		return nil, nil, err
	}
	err = rmq.connect(ctx, &s, true)
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

// Send sends an event to the message broker.
func (rmq *RabbitMQ) Send(ctx context.Context, event []byte, options connectors.SendOptions, ack func(err error)) error {
	var s innerSettings
	err := rmq.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}
	err = rmq.connect(ctx, &s, true)
	if err != nil {
		return err
	}
	msg := amqp.Publishing{Body: event}
	dc, err := rmq.ch.PublishWithDeferredConfirmWithContext(ctx, "", s.Queue, false, false, msg)
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
func (rmq *RabbitMQ) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		err := rmq.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, rmq.saveSettings(ctx, settings, false)
	case "test":
		return nil, rmq.saveSettings(ctx, settings, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "url", Label: "URL", Placeholder: "amqps://user:pass@example.com/vhost", Type: "text", MinLength: 7, MaxLength: 2048},
			&connectors.Input{Name: "queue", Label: "Queue", Placeholder: "queue-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
			connectors.SaveButton,
		},
	}

	return ui, nil
}

type innerSettings struct {
	URL   string `json:"url"`
	Queue string `json:"queue"`
}

const defaultConnectionTimeout = 30 * time.Second

// connect establishes a connection to RabbitMQ. If deliveries is true, it also
// sets the deliveries channel.
func (rmq *RabbitMQ) connect(ctx context.Context, settings *innerSettings, deliveries bool) (err error) {
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
	conn, err := amqp.DialConfig(settings.URL, config)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if deliveries {
		rmq.deliveries, err = ch.Consume(settings.Queue, "", false, false, false, false, nil)
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

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (rmq *RabbitMQ) saveSettings(ctx context.Context, options json.Value, test bool) error {
	var s innerSettings
	err := options.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate URL.
	if n := len(s.URL); n < 7 || n > 2048 {
		return connectors.NewInvalidSettingsError("URL length in bytes must be in range [7,2048]")
	}
	if _, err := amqp.ParseURI(s.URL); err != nil {
		return connectors.NewInvalidSettingsError("URL is not a valid RabbitMQ URI")
	}
	// Validate Queue.
	if n := len(s.Queue); n == 0 || n > 255 {
		return connectors.NewInvalidSettingsError("queue length in bytes must be in range [1,255]")
	}
	if strings.HasPrefix(s.Queue, "amq.") {
		return connectors.NewInvalidSettingsError("queue names starting with 'amq.' are reserved for internal use by the broker")
	}
	err = rmq.testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	return rmq.env.Settings.Store(ctx, &s)
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func (rmq *RabbitMQ) testConnection(ctx context.Context, settings *innerSettings) error {
	return rmq.connect(ctx, settings, false) // TO FIX

}
