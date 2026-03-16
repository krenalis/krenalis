# Events: EventSender

If the connector sends events, implement:

```go
EventTypes(ctx context.Context) ([]*connectors.EventType, error)
EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)
PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error)
SendEvents(ctx context.Context, events connectors.Events) error
```

This file is the canonical guide for `SendEvents` and `PreviewSendEvents`.
Use it to choose the right iteration method and payload-building pattern.

## EventTypes

- Return stable event type IDs (<= 100 runes), names, descriptions, and optionally `DefaultFilter`.
- Return only event types the connector actually supports.

## EventTypeSchema

- Return the extra values needed to send one event of that type.
- If no extra values are needed, return `types.Type{}`.
- If the event type does not exist, return `connectors.ErrEventTypeNotExist`.
- Use schema constraints aggressively:
  - `CreateRequired` for required fields
  - `Prefilled` for recommended mapping expressions
  - `WithPattern`, `WithMaxLength`, `WithValues`, and similar methods for constraints the type system can express

Example:

```go
func (c *MyApp) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	switch eventType {
	case "purchase":
		return types.Object([]types.Property{
			{
				Name:           "event_name",
				Type:           types.String().WithMaxLength(255).WithPattern(purchaseNameRE),
				Prefilled:      "event",
				CreateRequired: true,
				Description:    "Event name",
			},
			{
				Name:           "email",
				Type:           types.String().WithMaxBytes(320),
				Prefilled:      "traits.email",
				CreateRequired: true,
				Description:    "Customer email",
			},
			{
				Name:        "properties",
				Type:        types.Map(types.JSON()),
				Prefilled:   "properties",
				Description: "Event properties",
			},
		}), nil
	default:
		return types.Type{}, connectors.ErrEventTypeNotExist
	}
}
```

## Core rules for SendEvents and PreviewSendEvents

- They receive a non-empty `connectors.Events` sequence.
- One call must send at most one HTTP request, or zero if all events are discarded.
- Do not add your own concurrency.
- `PreviewSendEvents` must build the same request shape that one call to `SendEvents` would send.
- Preview must redact secrets and replace any visible pipeline identifier with `"[PIPELINE]"`.
- Return `(nil, nil)` from preview when every event is discarded locally.
- MUST keep event serialization linear: read values from the current event and write JSON directly into `BodyBuffer` inside the batching loop.
- MUST NOT build an intermediate map/struct for one event and then serialize it in a separate step, unless a documented exception is required.

Meergo guarantees that `event.Type.Values` conforms to `event.Type.Schema`.
Do not re-check presence or basic Go types for schema-defined fields.
Only validate application-specific constraints that cannot be expressed in the schema.

## Choosing the right iteration shape

Choose exactly one of these per call:

- `events.First()` when one request can contain only one event
- `events.SameUser()` when one request can contain multiple events but only for one user
- `events.All()` when one request can contain multiple events from mixed users

### Case 1: one event per request

Use `events.First()` when the API truly accepts only one event per request.

```go
func (c *MyApp) SendEvents(ctx context.Context, events connectors.Events) error {
	event := events.First()

	bb := c.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	bb.WriteByte('{')
	_ = bb.EncodeKeyValue("type", event.Type.ID)
	_ = bb.EncodeKeyValue("properties", event.Type.Values)
	bb.WriteByte('}')

	req, err := bb.NewRequest(ctx, http.MethodPost, "https://api.example.com/v1/event")
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.settings.Token)
	req.Header["Idempotency-Key"] = nil

	res, err := c.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	return nil
}
```

### Case 2: batch of events for one user

Use `events.SameUser()` when the API requires one user per batch.

```go
func (c *MyApp) SendEvents(ctx context.Context, events connectors.Events) error {
	bb := c.env.HTTPClient.GetBodyBuffer(connectors.Gzip)
	defer bb.Close()

	bb.WriteString(`{"events":[`)

	n := 0
	for event := range events.SameUser() {
		if n > 0 {
			bb.WriteByte(',')
		}
		bb.WriteByte('{')
		_ = bb.EncodeKeyValue("type", event.Type.ID)
		_ = bb.EncodeKeyValue("properties", event.Type.Values)
		bb.WriteByte('}')
		n++
		if n == 100 {
			break
		}
	}

	bb.WriteString(`]}`)

	req, err := bb.NewRequest(ctx, http.MethodPost, "https://api.example.com/v1/events")
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.settings.Token)
	req.Header["Idempotency-Key"] = nil

	res, err := c.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	return nil
}
```

### Case 3: batch of events from mixed users

Use `events.All()` when the API accepts mixed-user batches.

```go
func (c *MyApp) SendEvents(ctx context.Context, events connectors.Events) error {
	bb := c.env.HTTPClient.GetBodyBuffer(connectors.Gzip)
	defer bb.Close()

	bb.WriteString(`{"events":[`)

	n := 0
	for event := range events.All() {
		if n > 0 {
			bb.WriteByte(',')
		}
		bb.WriteByte('{')
		_ = bb.EncodeKeyValue("type", event.Type.ID)
		_ = bb.EncodeKeyValue("properties", event.Type.Values)
		bb.WriteByte('}')
		n++
		if n == 100 {
			break
		}
	}

	bb.WriteString(`]}`)

	req, err := bb.NewRequest(ctx, http.MethodPost, "https://api.example.com/v1/events")
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.settings.Token)
	req.Header["Idempotency-Key"] = nil

	res, err := c.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	return nil
}
```

### Case 4: choose the request shape with Peek

Use `events.Peek()` when the first unread event determines the endpoint or payload shape.

```go
func (c *MyApp) SendEvents(ctx context.Context, events connectors.Events) error {
	first, ok := events.Peek()
	if !ok {
		panic("meergo guarantees a non-empty events sequence")
	}

	switch first.Type.ID {
	case "identify":
		return c.sendIdentifyBatch(ctx, events)
	case "track":
		return c.sendTrackBatch(ctx, events)
	default:
		return connectors.ErrEventTypeNotExist
	}
}
```

If the chosen shape cannot include a later event, postpone the first incompatible later event and stop the batch.

## Body size and max-items limits

If the API limits the number of events, stop after the maximum.
If the API limits the request body size, checkpoint the buffer before writing each event.

Canonical body-size pattern:

```go
bb.WriteString(`{"events":[`)

n := 0
for event := range events.All() {
	checkpoint := bb.Len()

	if n > 0 {
		bb.WriteByte(',')
	}
	bb.WriteByte('{')
	_ = bb.EncodeKeyValue("type", event.Type.ID)
	_ = bb.EncodeKeyValue("properties", event.Type.Values)
	bb.WriteByte('}')

	if bb.Len()+len(`]}`) > maxBodySize {
		if n == 0 {
			return fmt.Errorf("event %q cannot fit into one request", event.Type.ID)
		}
		bb.Truncate(checkpoint)
		events.Postpone()
		break
	}

	if err := bb.Flush(); err != nil {
		return err
	}

	n++
	if n == maxEvents {
		break
	}
}

bb.WriteString(`]}`)
```

Rules:

- never call `Postpone()` on the first event of an event iteration
- if the first event alone cannot fit in one valid request, return an error instead of postponing it
- do not modify `event.Type.Values` if you may need to postpone that event
- if the API supports batching, pack as many events as possible into one request within documented limits
- keep the logic for one event readable in one place: extract the needed local variables, validate what must be validated, then write the JSON fields directly
- do not split one event into separate `build...` and `write...` helpers unless a documented exception is materially clearer and reused

Preferred shape for one event inside the loop:

```go
for event := range events.All() {
	eventName := event.Type.Values["event_name"].(string)
	eventProps, _ := event.Type.Values["event_properties"]
	timestamp := event.Received.Timestamp().Format(time.RFC3339Nano)

	checkpoint := bb.Len()
	if n > 0 {
		bb.WriteByte(',')
	}
	bb.WriteByte('{')
	_ = bb.EncodeKeyValue("event_name", eventName)
	_ = bb.EncodeKeyValue("event_date", timestamp)
	if eventProps != nil {
		_ = bb.EncodeKeyValue("event_properties", eventProps)
	}
	bb.WriteByte('}')

	if bb.Len()+len(`]}`) > maxBodySize {
		if n == 0 {
			return fmt.Errorf("event %q cannot fit into one request", eventName)
		}
		bb.Truncate(checkpoint)
		events.Postpone()
		break
	}

	// apply any additional count limits here...
}
```

This keeps the event-specific logic visible in one place. A reader can see, without jumping to helper functions, which values are read, which validations are applied, and how the event is serialized.

Do not do this:

```go
func buildBatchEvent(event *connectors.Event) map[string]any { ... }
func writeBatchEvent(bb *connectors.BodyBuffer, payload map[string]any) error {
	return bb.Encode(payload)
}
```

Use direct serialization in the loop instead.

## Validation and error handling

There are three distinct validation scenarios.

### Case 1: validate locally and discard before sending

Use `events.Discard(err)` only for local, pre-send, non-retryable validation failures.

```go
n := 0
for event := range events.All() {
	if utf8.RuneCountInString(event.Type.Values["event_name"].(string)) > 255 {
		events.Discard(errors.New("«event_name» is longer than 255 characters"))
		continue
	}

	// write event to the request body...
	n++
}

if n == 0 {
	return nil
}
```

In `PreviewSendEvents`, the corresponding outcome is `return nil, nil` when every event is discarded locally.

### Case 2: API returns one global validation error for the whole request

If one bad event causes the whole request to fail and the API does not tell you which event was bad, validate locally as much as possible before sending.

If the API still rejects the whole request with one error, return a generic error, or map every consumed event to the same error if that is the clearest representation for the connector.

### Case 3: API returns per-event validation errors

If the response tells you exactly which consumed events failed, return `connectors.EventsError`.

```go
eventErrs := connectors.EventsError{}
for _, apiErr := range response.Errors {
	eventErrs[apiErr.EventIndex] = fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
}
if len(eventErrs) > 0 {
	return eventErrs
}
```

Important:

- `events.Discard(err)` is pre-send local validation
- `connectors.EventsError` is post-send API feedback
- do not guess per-event outcomes when the API response is ambiguous

## PreviewSendEvents

`PreviewSendEvents` should usually share the same batching code as `SendEvents`.
The common pattern is an internal helper with a `preview bool` flag.

```go
func (c *MyApp) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return c.sendEvents(ctx, events, true)
}

func (c *MyApp) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := c.sendEvents(ctx, events, false)
	return err
}
```

In preview mode:

- build exactly the request that one `SendEvents` call would send
- redact secrets in headers and query parameters
- replace visible pipeline IDs with `"[PIPELINE]"`
- if the API provides a real validation endpoint, you may call it instead of the production ingestion endpoint
- otherwise, preview can only show local validation failures

## Final checklist for event sending

Before finishing:

- confirm the chosen iteration method matches the API shape
- confirm batching is used whenever the API supports it
- confirm one call sends at most one request
- confirm body building is streamed into `BodyBuffer`
- confirm preview mirrors send semantics
- confirm local validation vs `EventsError` is handled deliberately
