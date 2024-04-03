# Dispatch Events to Apps

Chichi facilitates dispatching events to apps capable of receiving them. This involves implementing the `AppEvents` interface within the connector and adapting the `Schema` method to also return schema information for event types.

To enable event dispatching, you must include the `Events` flag as a target during connector registration:

```go
chichi.RegisterApp(chichi.AppInfo{
    ...
    Targets: chichi.Users | chichi.Events,
    ...
})
```

Subsequently, implement the two methods of the `AppEvents` interface within the connector type:

- `EventTypes`: This method returns the types of events supported by the app.
- `EventRequest`: It takes an event as input and returns an HTTP request for dispatching the event to the app.

Let's delve into these methods in detail.

## Event Types

A connector for an app capable of receiving events may support one or more types of events. These event types are exposed through the `EventTypes` method:

```go
EventTypes(ctx context.Context) ([]*chichi.EventType, error)
```

The `EventTypes` method returns a non-empty slice of `EventType` values, defined as follows:

```go
type EventType struct {
    ID          string // identifier; must be unique for each event type
    Name        string // name for display
    Description string // description for display
}
```

You have the liberty to choose an identifier for an event type, provided it is unique among all event types of the connector and not an empty string. The name and description are used for display purposes only.

Additionally, event types may have a schema, which we'll explore further below.

### Schema

Sometimes, the received event might lack essential information required for dispatching to the app. In such cases, the schema of the event type defines the extra information needed.

Actions based on an event type will involve a transformation that, given an event, provides the extra information required by the connector. This information, along with the event, is passed to the connector's `EventRequest` method.

The schema of an event type is returned by the `Schema` method when the target is `Events`. If no extra information is needed for an event type, it must return the invalid schema.

For instance, if you need to dispatch a ["share"](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events?hl=en#share) event to Google Analytics, you might require parameters like "method," "content_type," and "item_id," which could vary for each event. However, at the connector implementation stage, you might not have values for these parameters or know where to obtain them. In such cases, you can specify how to determine these parameters using a transformation in the action for the "share" event.

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

When dispatching the event, the `EventRequest` method receives, as an argument, the result of the transformation, i.e., the values of the three parameters "method," "content_type," and "item_id" conforming to their schema.

If a field in the schema is mandatory, simply set the `Required` field in the `types.Property` struct to `true`. Additionally, for easier mapping compilation, you can specify a placeholder using the `Placeholder` field.

> When selecting a placeholder, consider that certain property names and traits hold specific meanings and can thus serve as suitable placeholders. Refer to the prefilled properties and traits sections within the events for further details:
>
>    - [page](../events/page.md#prefilled-properties)
>    - [screen](../events/screen.md#prefilled-properties)
>    - [track](../events/track.md#prefilled-properties)
>    - [identify](../events/identify.md#prefilled-traits)
>    - [group](../events/group.md#prefilled-traits)

Now, let's move on to dispatching events to the app using the `EventRequest` method.

## Event Request

To dispatch an event to an app, and to preview an event to dispatch, Chichi invokes the `EventRequest` method of the connector:

```go
EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error)
```

Given the event, `EventRequest` returns an HTTP request used to dispatch the event to the destination. The following are the parameters:

- `typ`: The type of the event; one of the types returned by the `EventTypes` method.

- `event`: The event to be dispatched.

- `extra`: Extra information required to prepare the request, conforming to the schema of the event type. It is `nil` if the event type does not have a schema.

- `schema`: The schema of the extra information. It is the invalid schema if the event type does not have a schema.

- `redacted`: Reports whether authentication data in the returned request must be redacted. It is `true` when the returned request is previewed.

`EventRequest` returns the HTTP request to be sent to the app:

```go
type EventRequest struct {
    Method string      // HTTP method (e.g., "POST").
    URL    string      // URL to which the request will be sent.
    Header http.Header // Header fields to be included with the request.
    Body   []byte      // The body of the request.
}
```

To redact the returned request, when the `redacted` argument is `true`, replace authorization data in the URL, header, and body with "REDACTED".
