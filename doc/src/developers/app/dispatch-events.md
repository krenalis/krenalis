# Dispatching Events to Apps

Chichi makes it easy to dispatch events to apps that can receive them. This involves implementing the `AppEvents` interface within the connector and adjusting the `Schema` method to also provide schema information for event types.

Here’s how to get started with setting up your connector to dispatch events:

```go
chichi.RegisterApp(chichi.AppInfo{
    ...
    Targets: chichi.Users | chichi.Events,
    ...
})
```

This piece of code registers your connector, telling Chichi that it's ready to manage events (as well as users). Next, you'll need to implement two key methods within your connector:

- `EventTypes`: Lists the types of events the app can work with.
- `EventRequest`: Takes an event and turns it into an HTTP request for dispatching the event to the app.

Let's look more closely at what each part does.

## Understanding Event Types

Your connector can be set up to handle different types of events. You'll use the `EventTypes` method to tell Chichi what these types are:

```go
EventTypes(ctx context.Context) ([]*chichi.EventType, error)
```

For every event type you support, you'll define a unique ID, a user-friendly name, and a description. Here's how you define an event type:

```go
type EventType struct {
    ID          string // unique identifier for the event type
    Name        string // display name for the event type
    Description string // description of the event type
}
```

You have the freedom to decide on the identifiers, names, and descriptions, as long as each event type has a unique ID.

### Adding Schema

Sometimes, the event might lack necessary information required for dispatching to the app. In such cases, the schema of the event type specifies the extra information needed.

Actions based on an event type involve a transformation that, given an event, provides the extra information required by the connector. This information, along with the event, is passed to the connector's `EventRequest` method.

The schema of an event type is provided by the connector's `Schema` method when the target is `Events`. If no extra information is needed for an event type, it must return the invalid schema.

For instance, if you need to dispatch a ["share"](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events?hl=en#share) event to Google Analytics, you might require parameters like "method," "content_type," and "item_id," which could vary for each event. However, during the connector implementation stage, you might not have values for these parameters or know where to obtain them. In such cases, you can specify how to determine these parameters using a transformation in the action for the "share" event.

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

When dispatching the event, the `EventRequest` method receives, as an argument, the result of the transformation, i.e., the values of the three parameters "method," "content_type," and "item_id" conforming to the event type schema.

If a field in the schema is mandatory, set the `Required` field in the `types.Property` struct to `true`. Additionally, you can specify a placeholder using the `Placeholder` field for easier mapping compilation.

> When selecting a placeholder, consider that certain property names and traits hold specific meanings and can thus serve as suitable placeholders. Refer to the prefilled properties and traits sections within the events for further details:
>
>    - [page](/events/page.md#prefilled-properties)
>    - [screen](/events/screen.md#prefilled-properties)
>    - [track](/events/track.md#prefilled-properties)
>    - [identify](/events/identify.md#prefilled-traits)
>    - [group](/events/group.md#prefilled-traits)

Now, let's move on to dispatching events to the app using the `EventRequest` method.

## Dispatching an Event

Finally, to actually dispatch an event to the app, the `EventRequest` method prepares an HTTP request with all the needed details:

```go
EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error)
```

Given the event, `EventRequest` returns an HTTP request used to dispatch the event to the destination. The parameters are:

- `typ`: The type of the event, one of the types returned by the connector's `EventTypes` method.
- `event`: The event to be dispatched.
- `extra`: Extra information required to prepare the request, conforming to the schema of the event type. It's `nil` if the event type doesn't have a schema.
- `schema`: The schema of the extra information. It's the invalid schema if the event type doesn't have a schema.
- `redacted`: Reports whether authentication data in the returned request must be redacted. It's `true` when the returned request is previewed.

`EventRequest` returns the HTTP request to be sent to the app:

```go
type EventRequest struct {
    Endpoint string      // Destination identifier, e.g., "us", "europe", or leave empty.
    Method   string      // HTTP method (e.g., "POST").
    URL      string      // URL to which the request will be sent.
    Header   http.Header // Header fields to be included with the request.
    Body     []byte      // The body of the request.
}
```

The `Endpoint` field identifies the specific destination for dispatching events. For instance, you can label endpoints as "us" or "europe" if events go to different locations based on privacy regions. Events going to the same endpoint get grouped into one queue, which avoids bottlenecks and keeps the system efficient. If a server where the events with the same endpoint are dispatched becomes unavailable, it affects only events routed to that queue. You can name this field anything, and if all events go to one destination, you can leave it empty.

To redact the returned request, when the `redacted` argument is `true`, replace authorization data in the URL, header, and body with "REDACTED".
