{% extends "/layouts/doc.html" %}
{% macro Title string %}App Connectors{% end %}
{% Article %}

# App connectors

App connectors allow to connect to apps, such as klaviyo, Salesforce, or Mailchimp, to import and export users and to send events.

App connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and methods.

Note that it is possible to implement an app connector that supports only reading or only writing of records, or only sending of events, as it is not necessary that an app connector supports all of them. It is sufficient to specify the functionalities that the connector implements through the `AppInfo`, described below, then implement the required methods for those functionalities.

## Quick start

In the creation of a new Go module, for your app connector, you can utilize the following template by pasting it into a Go file. Not all methods in the file need to be implemented; see below for descriptions of individual methods. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package klaviyo provides a connector for Klaviyo.
package klaviyo

import (
    "context"
    "net/http"

    "github.com/meergo/meergo"
    "github.com/meergo/meergo/core/types"
)

func init() {
    meergo.RegisterApp(meergo.AppInfo{
        Name: "Klaviyo",
        Categories: meergo.CategoryAutomation | meergo.CategoryMarketing,
        AsSource: &meergo.AsAppSource{
            Targets:       meergo.TargetUser,
            HasSettings:   true,
            Documentation: meergo.ConnectorRoleDocumentation{
                Summary: "Import profiles as users from Klaviyo",	
            },
        },
        AsDestination: &meergo.AsAppDestination{
            Targets:       meergo.TargetEvent | meergo.TargetUser,
            HasSettings:   true,
            SendingMode:   meergo.Server,
            Documentation: meergo.ConnectorRoleDocumentation{
                Summary: "Export users as profiles and send events to Klaviyo",	
            },
        },
        Terms: meergo.AppTerms{
            User:  "client",
            Users: "clients",
        },
        EndpointGroups: []meergo.EndpointGroup{
            {
                Patterns:    []string{"/api/event-bulk-create-jobs"},
                RateLimit:   meergo.RateLimit{RequestsPerSecond: 2.5, Burst: 10},
                RetryPolicy: retryPolicy,
            },
            {
                Patterns:    []string{"/api/profiles/"},
                RateLimit:   meergo.RateLimit{RequestsPerSecond: 11.6, Burst: 75},
                RetryPolicy: retryPolicy,
            },
        },
        Icon: icon,
    }, New)
}

var retryPolicy = meergo.RetryPolicy{
    "429":     meergo.RetryAfterStrategy(),
    "500 503": meergo.ExponentialStrategy(meergo.NetFailure, 100*time.Millisecond),
}

type Klaviyo struct {
    // Your connector fields.
}

// New returns a new connector instance for Klaviyo.
func New(env *meergo.AppEnv) (*Klaviyo, error) {
    // ...
}

// EventTypeSchema returns the schema of the specified event type.
func (ky *Klaviyo) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
    // ...
}

// EventTypes returns the event types of the connector's instance.
func (ky *Klaviyo) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
    // ...
}

// PreviewSendEvents builds and returns the HTTP request that would be used to
// send the given events to the app, without actually sending it.
func (ky *Klaviyo) PreviewSendEvents(ctx context.Context, events Events) (*http.Request, error) {
    // ...
}

{#
// ReceiveWebhook receives a webhook request and returns its payloads.
func (ky *Klaviyo) ReceiveWebhook(r *http.Request) ([]meergo.WebhookPayload, error) {
    // ...
}
#}
// RecordSchema returns the schema of the specified target and role.
func (ky *Klaviyo) RecordSchema(ctx context.Context, target meergo.Targets, role meergo.Role) (types.Type, error) {
    // ...
}

// Records returns the records of the specified target.
func (ky *Klaviyo) Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids []string, cursor string, schema types.Type) ([]meergo.Record, string, error) {
    // ...
}

// SendEvents sends events to an app. events is a non-empty sequence of
// events to send.
func (ky *Klaviyo) SendEvents(ctx context.Context, events Events) error {
    // ...
}

// Upsert updates or creates records in the app for the specified target.
func (ky *Klaviyo) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {
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

Later on, you can [build an executable with your connector](../../from-source#building-using-the-go-tools).

### About the connector

The `AppInfo` type describes information about the app connector:

- `Name`: short name, typically the name of the app. For example, "HubSpot", "Google Analytics", "Salesforce", etc.
- `Categories`: the categories that the connector falls into. There must be at least one category.
- `AsSource`: information about the app connector when it used as source. This should be set only when the app connector can be used as a source, otherwise should be nil.
  - `Targets`: targets supported by the app connector when it is used as source. Can only contain `TargetUser`.
  - `HasSettings`: indicates whether the connection has settings when used as a source
  - `Description`: description of the connector when it is used as a source.
- `AsDestination`: information about the app connector when it used as destination. This should be set only when the app connector can be used as a destination, otherwise should be nil.
  - `Targets`: targets supported by the app connector when it is used as a destination. Can contain `TargetEvent` and `TargetUser`.
  - `HasSettings`: indicates whether the connection has settings when used as destination
  - `SendingMode`: mode used to send the events to the app, if the app supports events. It can be `Client`, `Server`, or `ClientAndServer`.
  - `Description`: description of the connector when it is used as a destination.
  - `Terms`: singular and plural terms used by the app to refer to users—for example, "client"/"clients", "customer"/"customers", or "user"/"users".
{# - `TermForGroups`: term used by the app to indicate the groups, if they are supported. For example "organizations", "teams", or "groups". #}
- `IdentityIDLabel`: descriptive name of the identifier used by the app to identify a user. For example "ID", "User ID", or "HubSpot ID".
{# - `WebhooksPer`: indicates if webhooks are per account, connection, or connector. #}
- `OAuth`: OAuth 2.0 configuration. To be filled in only if OAuth is required. See [OAuth documentation](app/oauth).
- `EndpointGroups`: rate limiting and retry policies per endpoint group. See [Endpoint groups documentation](app/endpoint-groups).
- `Layouts`: layouts for the `datetime`, `date`, and `time` values when they are represented as strings. See [Time Layouts](data-values#time-layouts) in [Data Values](data-values) for more details.
- `Icon`: icon in SVG format representing the app. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterApp` function that, executed during package initialization, registers the app connector:

```go
func init() {
    meergo.RegisterApp(meergo.AppInfo{
        Name: "Klaviyo",
        AsSource: &meergo.AsAppSource{
            Targets:       meergo.TargetUser,
            HasSettings:   true,
            Documentation: meergo.ConnectorRoleDocumentation{
                Summary: "Import profiles as users from Klaviyo",	
            },
        },
        AsDestination: &meergo.AsAppDestination{
            Targets:       meergo.TargetEvent | meergo.TargetUser,
            HasSettings:   true,
            SendingMode:   meergo.Server,
            Documentation: meergo.ConnectorRoleDocumentation{
                Summary: "Export users as profiles and send events to Klaviyo",	
            },
        },
        Terms: meergo.AppTerms{
            User:  "client",
            Users: "clients",
        },
        EndpointGroups: []meergo.EndpointGroup{
            {
                Patterns:    []string{"/api/event-bulk-create-jobs"},
                RateLimit:   meergo.RateLimit{RequestsPerSecond: 2.5, Burst: 10},
                RetryPolicy: retryPolicy,
            },
            {
                Patterns:    []string{"/api/profiles/"},
                RateLimit:   meergo.RateLimit{RequestsPerSecond: 11.6, Burst: 75},
                RetryPolicy: retryPolicy,
            },
        },
        Icon: icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterApp` function is the function utilized for creating an app instance:

```go
func New(env *meergo.AppEnv) (*Klaviyo, error)
```

This function accepts an app environment and yields a value representing your custom type.

The structure of `AppEnv` is defined as follows:

```go
// AppEnv is the environment for an app connector.
type AppEnv struct {

    // Settings holds the raw settings data.
    Settings []byte

    // SetSettings is the function used to update the settings.
    SetSettings SetSettingsFunc

    // OAuthAccount is the OAuth account identifier for authentication.
    OAuthAccount string

    // HTTPClient is the HTTP client to use for all requests.
    HTTPClient HTTPClient
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSettings`: A function that enables the connector to update its settings as necessary.
- `OAuthAccount`: The app's account associated with the OAuth authorization.
- `HTTPClient`: The HTTP client used by the connector to make requests to the app. It seamlessly implements OAuth authorization if required and retries idempotent requests as specified.
{# - `WebhookURL`: The URL where the webhook can be sent, provided the connector supports webhooks. #}

### Continue reading

- [OAuth](app/oauth)
- [Endpoint groups](app/endpoint-groups)
- [Users](app/users)
- [Send events](app/send-events)
