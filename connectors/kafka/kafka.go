//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package kafka implements the Kafka connector.
// (https://kafka.apache.org/documentation/)
package kafka

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Stream and the UI interfaces.
var _ interface {
	chichi.Stream
	chichi.UI
} = (*Kafka)(nil)

func init() {
	chichi.RegisterStream(chichi.StreamInfo{
		Name: "Kafka",
		Icon: icon,
	}, New)
}

// New returns a new Kafka connector instance.
func New(conf *chichi.StreamConfig) (*Kafka, error) {
	c := Kafka{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Kafka connector")
		}
	}
	return &c, nil
}

type Kafka struct {
	conf     *chichi.StreamConfig
	settings *settings
	client   *kgo.Client
	iter     *fetchesRecordIter
}

// Close closes the stream.
func (kafka *Kafka) Close() error {
	if kafka.client == nil {
		return nil
	}
	kafka.client.Close()
	kafka.client = nil
	return nil
}

// Receive receives an event from the stream.
func (kafka *Kafka) Receive(ctx context.Context) ([]byte, func(), error) {
	err := kafka.connect()
	if err != nil {
		return nil, nil, err
	}
	// Fetch the event.
	if kafka.iter == nil {
		kafka.iter = &fetchesRecordIter{}
	}
	if kafka.iter.Done() {
		kafka.iter.fetches = kafka.client.PollFetches(ctx)
	}
	record, err := kafka.iter.Next()
	if err != nil {
		return nil, nil, err
	}
	ack := func() {
		_ = kafka.client.CommitRecords(ctx, record)
	}
	return record.Value, ack, nil
}

// Send sends an event to the stream.
func (kafka *Kafka) Send(ctx context.Context, event []byte, options chichi.SendOptions, ack func(err error)) error {
	err := kafka.connect()
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
		Topic: kafka.settings.Topic,
	}
	var promise func(*kgo.Record, error)
	if ack != nil {
		promise = func(r *kgo.Record, err error) { ack(err) }
	}
	kafka.client.Produce(ctx, record, promise)
	return nil
}

// ServeUI serves the connector's user interface.
func (kafka *Kafka) ServeUI(ctx context.Context, event string, values []byte) (*chichi.Form, *chichi.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if kafka.settings == nil {
			s.Kafka = &kafkaSettings{Port: 9092}
		} else {
			s = *kafka.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := kafka.ValidateSettings(ctx, values)
		if err != nil {
			if event == "test" {
				return nil, chichi.WarningAlert(err.Error()), nil
			}
			return nil, chichi.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, chichi.SuccessAlert("Connection established"), nil
		}
		err = kafka.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, chichi.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, chichi.ErrEventNotExist
	}

	form := &chichi.Form{
		Fields: []chichi.Component{
			&chichi.AlternativeFieldSets{
				Sets: []chichi.FieldSet{
					{
						Name:  "Kafka",
						Label: "Kafka",
						Fields: []chichi.Component{
							&chichi.Input{Name: "host", Label: "Host", Placeholder: "kafka.example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&chichi.Input{Name: "port", Label: "Port", Placeholder: "9092", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
							&chichi.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1},
							&chichi.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1},
						},
					},
					{
						Name:  "Confluent",
						Label: "Confluent",
						Fields: []chichi.Component{
							&chichi.Input{Name: "server", Label: "Bootstrap server", Placeholder: "12345.aws.confluent.cloud:9092", Type: "text", MinLength: 1, MaxLength: 258},
							&chichi.Input{Name: "key", Label: "Key", Placeholder: "AAAAAAAAAAAAAAAA", Type: "text", MinLength: 16, MaxLength: 16},
							&chichi.Input{Name: "secret", Label: "Secret", Placeholder: "secret", Type: "password", MinLength: 1},
						},
					},
				},
			},
			&chichi.Input{Name: "topic", Label: "Topic", Placeholder: "topic-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Values: values,
		Actions: []chichi.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (kafka *Kafka) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	switch {
	case s.Kafka != nil:
		// Validate Host.
		if n := len(s.Kafka.Host); n == 0 || n > 253 {
			return nil, chichi.Errorf("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Kafka.Port < 1 || s.Kafka.Port > 65536 {
			return nil, chichi.Errorf("port must be in range [1,65536]")
		}
	case s.Confluent != nil:
		// Validate Server.
		host, port, err := net.SplitHostPort(s.Confluent.Server)
		if err != nil {
			return nil, chichi.Errorf("server is not a valid host:port")
		}
		if n := len(host); n == 0 || n > 253 {
			return nil, chichi.Errorf("server host length in bytes must be in range [1,253]")
		}
		if p, _ := strconv.Atoi(port); p < 1 || p > 65536 {
			return nil, chichi.Errorf("server port must be in range [1,65536]")
		}
		// Validate Key.
		if utf8.RuneCountInString(s.Confluent.Key) != 16 {
			return nil, chichi.Errorf("key must be long 16 characters")
		}
	}
	// Validate Topic.
	if n := len(s.Topic); n == 0 || n > 255 {
		return nil, chichi.Errorf("topic length must be in range [1,255]")
	}
	if !validTopicName(s.Topic) {
		return nil, chichi.Errorf("topic name can contain only [A-Za-z0-9_.-]")
	}
	err = testConnection(ctx, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

type kafkaSettings struct {
	Host     string
	Port     int
	Username string
	Password string
}

type confluentSettings struct {
	Server string
	Key    string
	Secret string
}
type settings struct {
	Kafka     *kafkaSettings
	Confluent *confluentSettings
	Topic     string
}

// opts returns s as options to configure a client.
func (s *settings) opts() []kgo.Opt {
	var user, pass, broker string
	switch {
	case s.Kafka != nil:
		broker = net.JoinHostPort(s.Kafka.Host, strconv.Itoa(s.Kafka.Port))
		user = s.Kafka.Username
		pass = s.Kafka.Password
	case s.Confluent != nil:
		broker = s.Confluent.Server
		user = s.Confluent.Key
		pass = s.Confluent.Secret
	}
	auth := plain.Auth{User: user, Pass: pass}
	tlsDialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}}
	opts := []kgo.Opt{
		kgo.SASL(auth.AsMechanism()),
		kgo.SeedBrokers(broker),
		kgo.ConsumeTopics(s.Topic),
		kgo.Dialer(tlsDialer.DialContext),
	}
	return opts
}

// connect establishes a connection to Kafka.
func (kafka *Kafka) connect() error {
	if kafka.client != nil {
		return nil
	}
	cl, err := kgo.NewClient(kafka.settings.opts()...)
	if err != nil {
		return err
	}
	kafka.client = cl
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
