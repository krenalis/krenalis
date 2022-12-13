//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package kafka

// This package is the Kafka connector.
// (https://kafka.apache.org/documentation/)

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"

	"chichi/connector"
	"chichi/connector/ui"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the EventStreamConnection interface.
var _ connector.EventStreamConnection = &connection{}

func init() {
	connector.RegisterEventStream(connector.EventStream{
		Name:    "Kafka",
		Icon:    icon,
		Connect: connect,
	})
}

// connect returns a new Kafka connection.
func connect(ctx context.Context, conf *connector.EventStreamConfig) (connector.EventStreamConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Kafka connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
	client   *kgo.Client
	iter     *fetchesRecordIter
}

// Close closes the stream. Must be called if at least one Send or Receive call
// has been made. It cannot be called concurrently with Send and Receive.
func (c *connection) Close() error {
	if c.client == nil {
		return nil
	}
	c.client.Close()
	c.client = nil
	return nil
}

// Receive receives an event from the stream. Callers call the ack function to
// notify that the event has been received. The connector resends the event if
// not acknowledged.
//
// Caller do not modify the event data, even temporarily, and event is not
// retained after the ack function has been called.
func (c *connection) Receive() ([]byte, func(), error) {
	err := c.connect()
	if err != nil {
		return nil, nil, err
	}
	// Fetch the event.
	if c.iter == nil {
		c.iter = &fetchesRecordIter{}
	}
	if c.iter.Done() {
		c.iter.fetches = c.client.PollFetches(c.ctx)
	}
	record, err := c.iter.Next()
	if err != nil {
		return nil, nil, err
	}
	ack := func() {
		_ = c.client.CommitRecords(c.ctx, record)
	}
	return record.Value, ack, nil
}

// Send sends an event to the stream. If ack is not nil, connector calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but event is not retained after the ack
// function has been called.
func (c *connection) Send(event []byte, options connector.SendOptions, ack func(err error)) error {
	err := c.connect()
	if err != nil {
		return err
	}
	// Send the event.
	var key []byte
	if options.OrderKey != "" {
		key = []byte(options.OrderKey)
	}
	record := &kgo.Record{
		Key:   key,
		Value: event,
		Topic: c.settings.Topic,
	}
	var promise func(*kgo.Record, error)
	if ack != nil {
		promise = func(r *kgo.Record, err error) { ack(err) }
	}
	c.client.Produce(c.ctx, record, promise)
	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings == nil {
			s.Port = 9092
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, err
		}
		// Validate Host.
		if n := len(s.Host); n == 0 || n > 253 {
			return nil, nil, ui.Errorf("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Port < 1 || s.Port > 65536 {
			return nil, nil, ui.Errorf("port must be in range [1,65536]")
		}
		// Validate Topic.
		if n := len(s.Topic); n == 0 || n > 255 {
			return nil, nil, ui.Errorf("topic length must be in range [1,255]")
		}
		if !validTopicName(s.Topic) {
			return nil, nil, ui.Errorf("topic name can contain only [A-Za-z0-9_.-]")
		}
		err = testConnection(c.ctx, &s)
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
			&ui.Input{Name: "host", Label: "Host", Placeholder: "kafka.example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Label: "Port", Placeholder: "9092", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1},
			&ui.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1},
			&ui.Input{Name: "topic", Label: "Topic", Placeholder: "topic-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Topic    string
}

// opts returns s as options to configure a client.
func (s *settings) opts() []kgo.Opt {
	auth := plain.Auth{
		User: s.Username,
		Pass: s.Password,
	}
	tlsDialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}}
	opts := []kgo.Opt{
		kgo.SASL(auth.AsMechanism()),
		kgo.SeedBrokers(net.JoinHostPort(s.Host, strconv.Itoa(s.Port))),
		kgo.ConsumeTopics(s.Topic),
		kgo.Dialer(tlsDialer.DialContext),
	}
	return opts
}

// connect establishes a connection to Kafka.
func (c *connection) connect() error {
	if c.client != nil {
		return nil
	}
	cl, err := kgo.NewClient(c.settings.opts()...)
	if err != nil {
		return err
	}
	c.client = cl
	return nil
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *settings) error {
	cl, err := kgo.NewClient(settings.opts()...)
	if err != nil {
		return err
	}
	defer cl.Close()
	return cl.Ping(ctx)
}

// validTopicName reports whether a topic name is valid.
func validTopicName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

// fetchesRecordIter iterates over records in a fetch.
//
// This code is the same code as kgo.FetchesRecordIter (Copyright 2020, Travis
// Bischel) but reworked to return partition errors.
type fetchesRecordIter struct {
	fetches []kgo.Fetch
	ti      int // index to current topic in fetches[0]
	pi      int // index to current partition in current topic
	ri      int // index to current record in current partition
}

// Done returns whether there are any more records to iterate over.
func (i *fetchesRecordIter) Done() bool {
	return len(i.fetches) == 0
}

// Next returns the next record from a fetch or an error if an error occurred
// while fetching a partition.
//
// Next is like the (*kgo.FetchesRecordIter).Next method but if a partition has
// an error it returns the error.
func (i *fetchesRecordIter) Next() (*kgo.Record, error) {
	partition := i.fetches[0].Topics[i.ti].Partitions[i.pi]
	if partition.Err != nil {
		i.pi++
		i.ri = 0
		i.prepareNext()
		return nil, partition.Err
	}
	record := partition.Records[i.ri]
	i.ri++
	i.prepareNext()
	return record, nil
}

func (i *fetchesRecordIter) prepareNext() {
beforeFetch0:
	if len(i.fetches) == 0 {
		return
	}

	fetch0 := &i.fetches[0]
beforeTopic:
	if i.ti >= len(fetch0.Topics) {
		i.fetches = i.fetches[1:]
		i.ti = 0
		goto beforeFetch0
	}

	topic := &fetch0.Topics[i.ti]
beforePartition:
	if i.pi >= len(topic.Partitions) {
		i.ti++
		i.pi = 0
		goto beforeTopic
	}

	partition := &topic.Partitions[i.pi]
	if i.ri >= len(partition.Records) {
		i.pi++
		i.ri = 0
		goto beforePartition
	}
}
