//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package dummy

import (
	"context"
	"net/http"
	"time"

	"chichi/connectors"
)

// Make sure it implements the Connector interface.
var _ connectors.AppConnecter = &Connector{}

type Connector struct {
	Firehose     connectors.Firehose
	ClientSecret string
}

func init() {
	connectors.RegisterConnector("Dummy", (*Connector)(nil))
}

func (c *Connector) Groups(ctx context.Context, cursor string, properties [][]string) error {
	panic("not implemented")
}

func (c *Connector) Properties(ctx context.Context) ([]connectors.Property, []connectors.Property, error) {
	panic("not implemented")
}

func (c *Connector) ReceiveWebhook(ctx context.Context, r *http.Request) ([]connectors.Event, error) {
	panic("not implemented")
}

func (c *Connector) Resource(ctx context.Context) (string, error) {
	return "dummy-resource", nil
}

func (c *Connector) ServeUserInterface(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

type user struct {
	ID         string
	Properties map[string]any
}

var now = time.Now()

var users = []user{
	{
		ID: "1",
		Properties: map[string]any{
			"first_name": "Mario",
			"last_name":  connectors.TimestampedValue{Value: "Rossi", Timestamp: now.Add(5 * time.Second)},
			"email":      connectors.TimestampedValue{Value: "mariorossi@example.com", Timestamp: now.Add(1 * time.Second)},
		},
	},
	{
		ID: "2",
		Properties: map[string]any{
			"first_name": "Luigi",
			"last_name":  connectors.TimestampedValue{Value: "Verdi", Timestamp: now.Add(7 * time.Second)},
			"email":      connectors.TimestampedValue{Value: "luigiverdi@example.com", Timestamp: now.Add(3 * time.Second)},
		},
	},
}

func (c *Connector) SetUsers(ctx context.Context, users []connectors.User) error {
	panic("not implemented")
}

func (c *Connector) Users(ctx context.Context, cursor string, properties [][]string) error {
	c.setContext(ctx)
	for _, user := range users {
		c.Firehose.SetUser(user.ID, now, user.Properties)
	}
	return nil
}

// setContext sets ctx as the context for c.
func (c *Connector) setContext(ctx context.Context) {
	c.Firehose, _ = ctx.Value(connectors.FirehoseContextKey{}).(connectors.Firehose)
}
