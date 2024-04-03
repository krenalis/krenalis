# App Connectors

App connectors allow to connect to cloud apps, such as klaviyo, Salesforce, or Mailchimp, to import and export users and groups and to send events.

App connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

## Quick Start

In the creation of a new Go module, for your app connector, you can utilize the following template by pasting it into a Go file. Not all methods in the file need to be implemented; see below for descriptions of individual methods. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package klaviyo implements the Klaviyo app connector.
package klaviyo

import (
	"context"
	_ "embed"
	"net/http"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Klaviyo",
		Targets:                chichi.Events | chichi.Users,
		SourceDescription:      "import clients as users from Klaviyo",
		DestinationDescription: "export users as clients and send events to Klaviyo",
		TermForUsers:           "clients",
		SendingMode:            chichi.Cloud,
		Icon:                   icon,
	}, New)
}

type klaviyo struct {
	// Your connector fields.
}

// New returns a new klaviyo connector instance.
func New(conf *chichi.AppConfig) (*klaviyo, error) {
	// ...
}

// Create creates a record for the specified target with the given properties.
func (ky *Klavyio) Create(ctx context.Context, target chichi.Targets, record map[string]any) error {
	// ...
}

// EventRequest returns a request that can be used to dispatch an event to the
// app.
func (ky *Klavyio) EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error) {
	// ...
}

// EventTypes returns the event types of the connector's instance.
func (ky *Klavyio) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	// ...
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (ky *Klavyio) ReceiveWebhook(r *http.Request) ([]chichi.WebhookPayload, error) {
	// ...
}

// Records returns the records of the specified target, starting from the given
// cursor.
func (ky *Klavyio) Records(ctx context.Context, target chichi.Targets, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
	// ...
}

// Schema returns the schema of the specified target.
func (ky *Klavyio) Schema(ctx context.Context, target Targets, eventType string) (types.Type, error) {
	// ...
}

// Update updates the record of the specified target with the identifier id,
// setting the given properties.
func (ky *Klavyio) Update(ctx context.Context, target chichi.Targets, id string, record map[string]any) error {
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

As you can see, the template file embeds a SVG file, this is the icon that represent the connector. Choose an appropriate SVG icon and put it into the module directory.

### About the Connector

The `AppInfo` type describes information about the app connector:

- `Name`: short name, typically the name of the cloud app. For example, "HubSpot", "Google Analytics", "Salesforce", etc.

- `Targets`: targets supported by the app connector. Can contain Events, Users, and Groups.

- `SourceDescription`: brief description of the connector when the connector is used as a source. It should complete the sentence "Add an action to ...".

- `DestinationDescription`: brief description of the connector when the connector is used as a destination. It should complete the sentence "Add an action to ...".

- `TermForUsers`: term used by the cloud app to indicate the users. For example "clients", "customers", or "users". 

- `TermForGroups`: term used by the cloud app to indicate the groups, if they are supported. For example "organizations", "teams", or "groups".

- `ExternalIDLabel`: descriptive name of the identifier used by the cloud app to identify a user. For example "ID", "User ID", or "HubSpot ID".

- `SendingMode`: mode used to send the events to the cloud app, if the app supports events. It can be `Cloud`, `Device`, or `Combined`.

- `SuggestedDisplayedID`: suggestion for the property name to use as the displayed identifier. This field may be empty if there is no property to suggest, and it is not required to always exist as a property.

- `Icon`: icon in SVG format representing the app. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterApp` function that, executed during package initialization, registers the app connector:

```go
func init() {
    chichi.RegisterApp(chichi.AppInfo{
        Name:                   "Klaviyo",
        Targets:                chichi.Events | chichi.Users,
        SourceDescription:      "import clients as users from Klaviyo",
        DestinationDescription: "export users as clients and send events to Klaviyo",
        TermForUsers:           "clients",
        SendingMode:            chichi.Cloud,
        Icon:                   icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterApp` function is the function utilized for creating an app instance:

```go
func New(conf *chichi.AppConfig) (*Klaviyo, error)
```

This function accepts an app configuration and yields a value representing your custom type. A connector can be instantiated either as a source or a destination, but not both simultaneously. Consequently, an instance of a connector will be responsible for either receiving or sending to a cloud app, depending on its role.

The documentation is mostly correct but can be improved for clarity and consistency. Here's a revised version:

### App Configuration

The structure of `AppConfig` is defined as follows:

```go
type AppConfig struct {
    Role         chichi.Role
    Settings     []byte
    SetSettings  chichi.SetSettingsFunc
    Resource     string
    HTTPClient   chichi.HTTPClient
    Region       chichi.PrivacyRegion
    WebhookURL   string
}
```

- `Role`: Specifies the intended role of the resulting instance, which can be either `Source` or `Destination`.

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.

- `SetSettings`: A function that enables the connector to update its settings as necessary.

- `HTTPClient`: The HTTP client used by the connector to make requests to the cloud app. It seamlessly implements OAuth authorization if required.

- `Region`: Indicates the privacy region of the workspace. The connector must adhere to the specified privacy region if supported. It defaults to `PrivacyRegionNotSpecified` if no region is specified, or `PrivacyRegionEurope` if the Europe region is specified.

- `WebhookURL`: The URL where the webhook can be sent, provided the connector supports webhooks.
