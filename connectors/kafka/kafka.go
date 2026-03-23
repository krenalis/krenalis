// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package kafka provides a connector for Kafka.
// (https://kafka.apache.org/documentation/)
//
// Kafka is a trademark of Apache Software Foundation.
// This connector is not affiliated with or endorsed by Apache Software
// Foundation.
package kafka

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"net"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterMessageBroker(connectors.MessageBrokerSpec{
		Code:       "kafka",
		Label:      "Kafka",
		Categories: connectors.CategoryMessageBroker,
		Documentation: connectors.Documentation{
			Source: connectors.RoleDocumentation{
				Summary:  "Import events and users from Kafka",
				Overview: overview,
			},
		},
	}, New)
}

// New returns a new connector instance for Kafka.
func New(env *connectors.MessageBrokerEnv) (*Kafka, error) {
	c := Kafka{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Kafka")
		}
	}
	return &c, nil
}

type Kafka struct {
	env      *connectors.MessageBrokerEnv
	settings *innerSettings
	client   *kgo.Client
	iter     *fetchesRecordIter
}

// Close closes the message broker.
func (kafka *Kafka) Close() error {
	if kafka.client == nil {
		return nil
	}
	kafka.client.Close()
	kafka.client = nil
	return nil
}

// Receive receives an event from the message broker.
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

// Send sends an event to the message broker.
func (kafka *Kafka) Send(ctx context.Context, event []byte, options connectors.SendOptions, ack func(err error)) error {
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
func (kafka *Kafka) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if kafka.settings == nil {
			s.Kafka = &kafkaSettings{Port: 9092}
		} else {
			s = *kafka.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, kafka.saveSettings(ctx, settings, false)
	case "test":
		return nil, kafka.saveSettings(ctx, settings, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.AlternativeFieldSets{
				Sets: []connectors.FieldSet{
					{
						Name:  "kafka",
						Label: "Kafka",
						Fields: []connectors.Component{
							&connectors.Input{Name: "host", Label: "Host", Placeholder: "kafka.example.com", Type: "text", MinLength: 1, MaxLength: 253},
							&connectors.Input{Name: "port", Label: "Port", Placeholder: "9092", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
							&connectors.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1},
							&connectors.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1},
						},
					},
					{
						Name:  "confluent",
						Label: "Confluent",
						Fields: []connectors.Component{
							&connectors.Input{Name: "server", Label: "Bootstrap server", Placeholder: "12345.aws.confluent.cloud:9092", Type: "text", MinLength: 1, MaxLength: 258},
							&connectors.Input{Name: "key", Label: "Key", Placeholder: "AAAAAAAAAAAAAAAA", Type: "text", MinLength: 16, MaxLength: 16},
							&connectors.Input{Name: "secret", Label: "Secret", Placeholder: "secret", Type: "password", MinLength: 1},
						},
					},
				},
			},
			&connectors.Input{Name: "topic", Label: "Topic", Placeholder: "topic-name", Type: "text", MinLength: 1, MaxLength: 255},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (kafka *Kafka) saveSettings(ctx context.Context, settings json.Value, test bool) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	switch {
	case s.Kafka != nil:
		// Validate Host.
		if n := len(s.Kafka.Host); n == 0 || n > 253 {
			return connectors.NewInvalidSettingsError("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Kafka.Port < 1 || s.Kafka.Port > 65535 {
			return connectors.NewInvalidSettingsError("port must be in range [1,65535]")
		}
	case s.Confluent != nil:
		// Validate Server.
		host, port, err := net.SplitHostPort(s.Confluent.Server)
		if err != nil {
			return connectors.NewInvalidSettingsError("server is not a valid host:port")
		}
		if n := len(host); n == 0 || n > 253 {
			return connectors.NewInvalidSettingsError("server host length in bytes must be in range [1,253]")
		}
		if p, _ := strconv.Atoi(port); p < 1 || p > 65535 {
			return connectors.NewInvalidSettingsError("server port must be in range [1,65535]")
		}
		// Validate Key.
		if utf8.RuneCountInString(s.Confluent.Key) != 16 {
			return connectors.NewInvalidSettingsError("key must be long 16 characters")
		}
	}
	// Validate Topic.
	if n := len(s.Topic); n == 0 || n > 255 {
		return connectors.NewInvalidSettingsError("topic length must be in range [1,255]")
	}
	if !validTopicName(s.Topic) {
		return connectors.NewInvalidSettingsError("topic name can contain only [A-Za-z0-9_.-]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = kafka.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	kafka.settings = &s
	return nil
}

type kafkaSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type confluentSettings struct {
	Server string `json:"server"`
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

type innerSettings struct {
	Kafka     *kafkaSettings     `json:"kafka"`
	Confluent *confluentSettings `json:"confluent"`
	Topic     string             `json:"topic"`
}

// opts returns s as options to configure a client.
func (s *innerSettings) opts() []kgo.Opt {
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
func testConnection(ctx context.Context, settings *innerSettings) error {
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
