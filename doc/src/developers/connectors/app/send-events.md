{% extends "/layouts/doc.html" %}
{% macro Title string %}Send Events to Apps{% end %}
{% Article %}

# Send events to apps

Meergo makes it easy to send events to apps that can receive them.

Here’s how to get started with setting up your connector to send events:

```go
meergo.RegisterApp(meergo.AppInfo{
    ...
    AsDestination: &meergo.AsAppDestination{
        ...
        Targets:  meergo.UsersTarget | meergo.EventsTarget,
        ...
    },
    ...
}, New)
```

This piece of code registers your connector, telling Meergo that it's ready to manage events (as well as users) when used as destination. Next, you'll need to implement two key methods within your connector:

- `EventTypes`: Lists the types of events the app can work with.
- `EventRequest`: Takes an event and turns it into an HTTP request for sending the event to the app.
- `Schema`: Provides the schema of an event, given its event type. If the connector already handles users, probably this method is already implemented and should only be extended to support events.

Let's look more closely at what each part does.

## Understanding event types

Your connector can be set up to handle different types of events. You'll use the `EventTypes` method to tell Meergo what these types are:

```go
EventTypes(ctx context.Context) ([]*meergo.EventType, error)
```

For every event type you support, you'll define a unique ID, a user-friendly name, and a description. Here's how you define an event type:

```go
type EventType struct {
    ID          string // unique identifier for the event type. Cannot be longer than 100 runes.
    Name        string // display name for the event type
    Description string // description of the event type
}
```

You have the freedom to decide on the identifiers, names, and descriptions, as long as each event type has a unique ID.

### Adding schema

Sometimes, the event might lack necessary information required for sending to the app. In such cases, the schema of the event type specifies the extra information needed.

Actions based on an event type involve a transformation that, given an event, provides the extra information required by the connector. This information, along with the event, is passed to the connector's `EventRequest` method.

The schema of an event type is provided by the connector's `Schema` method when the target is `EventsTarget`. If no extra information is needed for an event type, it must return the invalid schema.

For instance, if you need to send a ["share"](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events?hl=en#share) event to Google Analytics, you might require parameters like "method," "content_type," and "item_id," which could vary for each event. However, during the connector implementation stage, you might not have values for these parameters or know where to obtain them. In such cases, you can specify how to determine these parameters using a transformation in the action for the "share" event.

For example, the "share" event type might have the following schema:

```go
types.Object([]types.Property{
    {Name: "method", Type: types.Text()},
    {Name: "content_type", Type: types.Text()},
    {Name: "item_id", Type: types.Text()},
})
```

In the action of the "share" event type, if a mapping is chosen as a transformation, the schema of the event type would be represented as follows:

```
┌─────────────────────────────────┐
│                                 │ ->  method
└─────────────────────────────────┘
┌─────────────────────────────────┐
│                                 │ ->  content_type
└─────────────────────────────────┘
┌─────────────────────────────────┐
│                                 │ ->  item_id
└─────────────────────────────────┘
```

When sending the event, the `EventRequest` method receives, as an argument, the result of the transformation, i.e., the values of the three parameters "method," "content_type," and "item_id" conforming to the event type schema.

If a field in the schema is mandatory, set the `Required` field in the `types.Property` struct to `true`. Additionally, you can specify a placeholder using the `Placeholder` field for easier mapping compilation.

> When selecting a placeholder, consider that certain property names and traits hold specific meanings and can thus serve as suitable placeholders. Refer to the prefilled properties and traits sections within the events for further details:
>
>    - [page](/events/page#prefilled-properties)
>    - [screen](/events/screen#prefilled-properties)
>    - [track](/events/track#prefilled-properties)
>    - [identify](/events/identify#prefilled-traits)
>    - [group](/events/group#prefilled-traits)

Now, let's move on to sending events to the app using the `EventRequest` method.

## Sending an event

Finally, to actually send an event to the app, the `EventRequest` method prepares an HTTP request with all the needed details:

```go
EventRequest(ctx context.Context, event RawEvent, eventType string, schema types.Type, properties map[string]any, redacted bool) (*EventRequest, error)
```

Given the event, `EventRequest` returns an HTTP request used to send the event to the destination. The parameters are:

- `event`: The event to be sent.
- `eventType`: The identifier of the event type, one of the types returned by the connector's `EventTypes` method.
- `schema`: The schema of the event type. It is the invalid schema if the event type does not have a schema.
- `properties`: The values for the properties of the event type, required to prepare the request. These values conform to the schema of the event type. It is `nil` if the event type does not have a schema.
- `redacted`: Reports whether authentication data in the returned request must be redacted. It's `true` when the returned request is previewed.

The `EventRequest` method returns the HTTP request to be sent to the app. The returned value has the following type:

```go
type EventRequest struct {
    Endpoint string      // Destination identifier, e.g., "us", "europe", or leave empty.
    Method   string      // HTTP method (e.g., "POST").
    URL      string      // URL to which the request will be sent.
    Header   http.Header // Header fields to be included with the request.
    Body     []byte      // The body of the request.
}
```

The `Endpoint` field identifies the specific destination for sending events. For instance, you can label endpoints as "us" or "europe" if events go to different locations based, for example, on privacy regions. Events going to the same endpoint get grouped into one queue, which avoids bottlenecks and keeps the system efficient. If a server where the events with the same endpoint are sent becomes unavailable, it affects only events routed to that queue. You can assign any value to this field, and if all events go to one destination, you can leave it empty.

To redact the returned request, when the `redacted` argument is `true`, replace authorization data in the URL, header, and body with "REDACTED".
