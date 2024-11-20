# App Connectors

App connectors allow to connect to apps, such as klaviyo, Salesforce, or Mailchimp, to import and export users and groups and to dispatch events.

App connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

## Quick Start

In the creation of a new Go module, for your app connector, you can utilize the following template by pasting it into a Go file. Not all methods in the file need to be implemented; see below for descriptions of individual methods. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package klaviyo implements the Klaviyo app connector.
package klaviyo

import (
	"context"
	"net/http"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:                   "Klaviyo",
		Targets:                meergo.Events | meergo.Users,
		SourceDescription:      "import clients as users from Klaviyo",
		DestinationDescription: "export users as clients and dispatch events to Klaviyo",
		TermForUsers:           "clients",
		SendingMode:            meergo.Cloud,
	}, New)
}

type Klaviyo struct {
	// Your connector fields.
}

// New returns a new Klaviyo connector instance.
func New(conf *meergo.AppConfig) (*Klaviyo, error) {
	// ...
}

// EventRequest returns a request to dispatch an event to the app.
func (ky *Klavyio) EventRequest(ctx context.Context, typ string, event *meergo.Event, extra map[string]any, schema types.Type, redacted bool) (*meergo.EventRequest, error) {
	// ...
}

// EventTypes returns the event types of the connector's instance.
func (ky *Klavyio) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	// ...
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (ky *Klavyio) ReceiveWebhook(r *http.Request) ([]meergo.WebhookPayload, error) {
	// ...
}

// Records returns the records of the specified target.
func (ky *Klavyio) Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string) ([]meergo.Record, string, error) {
	// ...
}

// Schema returns the schema of the specified target in the specified role
func (ky *Klavyio) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	// ...
}

// Upsert updates or creates records in the app for the specified target.
func (ky *Klavyio) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {
	// ...
}
```

## Implementation

Let's explore how to implement an app connector, for example for Klaviyo.

First create a Go module:

```sh
$ mkdir klaviyo
$ cd klaviyo
$ go mod init klaviyo
```

Then add a Go file to the new directory. For example copy the previous template file.

Later on, you can [build an executable with your connector](../../getting-started.md#build-with-your-custom-connectors).

### About the Connector

The `AppInfo` type describes information about the app connector:

- `Name`: short name, typically the name of the app. For example, "HubSpot", "Google Analytics", "Salesforce", etc.
- `Targets`: targets supported by the app connector. Can contain `Events`, `Users`, and `Groups.
- `SourceDescription`: brief description of the connector when the connector is used as a source. It should complete the sentence "Add an action to ...".
- `DestinationDescription`: brief description of the connector when the connector is used as a destination. It should complete the sentence "Add an action to ...".
- `TermForUsers`: term used by the app to indicate the users. For example "clients", "customers", or "users".
- `TermForGroups`: term used by the app to indicate the groups, if they are supported. For example "organizations", "teams", or "groups".
- `IdentityIDLabel`: descriptive name of the identifier used by the app to identify a user. For example "ID", "User ID", or "HubSpot ID".
- `WebhooksPer`: indicates if webhooks are per account, connection, or connector.
- `OAuth`: OAuth 2.0 configuration. To be filled in only if OAuth is required. See [OAuth documentation](app/oauth.md).
- `BackoffPolicy`: Backoff policy. It controls retry timing using provided strategies or custom ones. See [Backoff documentation](app/backoff.md).
- `SendingMode`: mode used to dispatch the events to the app, if the app supports events. It can be `Cloud`, `Device`, or `Combined`.
- `Layouts`: layouts for the `DateTime`, `Date`, and `Time` values when they are represented as strings. See [Time Layouts](data-values.md#time-layouts) in [Data Values](data-values.md) for more details.
- `Icon`: icon in SVG format representing the app. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterApp` function that, executed during package initialization, registers the app connector:

```go
func init() {
    meergo.RegisterApp(meergo.AppInfo{
        Name:                   "Klaviyo",
        Targets:                meergo.Events | meergo.Users,
        SourceDescription:      "import clients as users from Klaviyo",
        DestinationDescription: "export users as clients and dispatch events to Klaviyo",
        TermForUsers:           "clients",
        SendingMode:            meergo.Cloud,
        Icon:                   icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterApp` function is the function utilized for creating an app instance:

```go
func New(conf *meergo.AppConfig) (*Klaviyo, error)
```

This function accepts an app configuration and yields a value representing your custom type.

The structure of `AppConfig` is defined as follows:

```go
type AppConfig struct {
    Settings     []byte
    SetSettings  meergo.SetSettingsFunc
    OAuthAccount string
    HTTPClient   meergo.HTTPClient
    Region       meergo.PrivacyRegion
    WebhookURL   string
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSettings`: A function that enables the connector to update its settings as necessary.
- `OAuthAccount`: The app's account associated with the OAuth authorization.
- `HTTPClient`: The HTTP client used by the connector to make requests to the app. It seamlessly implements OAuth authorization if required and retries idempotent requests as specified.
- `Region`: Indicates the privacy region of the workspace. The connector must adhere to the specified privacy region if supported. It defaults to `PrivacyRegionNotSpecified` if no region is specified, or `PrivacyRegionEurope` if the Europe region is specified.
- `WebhookURL`: The URL where the webhook can be sent, provided the connector supports webhooks.

### Continue Reading

- [OAuth](app/oauth.md)
- [Backoff](app/backoff.md)
- [Users and Groups](app/users-and-groups.md)
- [Dispatch Events](app/dispatch-events.md)
