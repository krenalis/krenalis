{% extends "/layouts/doc.html" %}
{% macro Title string %}Rate Limits{% end %}
{% Article %}

# Rate limits

Meergo applies client-side rate limits to avoid hitting the limits imposed by an app.
When you register your connector you must provide them using the `RateLimits` field of
`AppInfo`.

Each entry in `RateLimits` associates an HTTP request pattern with a `RateLimit` value.
Patterns follow the same syntax used by [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux).
Meergo picks the most specific pattern that matches the request method, host and path.
If a request does not match any pattern, an error is returned.

A `RateLimit` defines how quickly requests can be made and optionally how many can be
in flight at the same time:

```go
type RateLimit struct {
    RequestsPerSecond     float64 // max allowed requests per second (> 0)
    Burst                 int     // number of requests that can be sent in a burst (> 0)
    MaxConcurrentRequests int     // maximum number of concurrent requests (>= 0)
}
```

`RequestsPerSecond` is the average rate permitted for the matched requests. `Burst`
allows temporarily exceeding that rate after a period of low usage. When set,
`MaxConcurrentRequests` limits how many of those requests can be running at the
same time; use `0` for no concurrency limit.

## Example

Here is how the Klaviyo connector defines two limits, one for event ingestion and one
for profile management:

```go
meergo.RegisterApp(meergo.AppInfo{
    // ... other fields ...
    RateLimits: meergo.RateLimits{
        "/api/event-bulk-create-jobs": {RequestsPerSecond: 2.5, Burst: 10},
        "/api/profiles/":              {RequestsPerSecond: 11.6, Burst: 75},
    },
}, New)
```

When your connector performs HTTP calls through the provided HTTP client, Meergo will
handle rate limiting according to these rules.

Here are a few more examples:

### Single rate limit for all calls

```go
    RateLimits: meergo.RateLimits{
        "/": {RequestsPerSecond: 10, Burst: 20},
    },
```

### Rate limits based on HTTP method

```go
    RateLimits: meergo.RateLimits{
        "GET  /users": {RequestsPerSecond: 12, Burst: 18},
        "POST /users": {RequestsPerSecond: 5, Burst: 5, MaxConcurrentRequests: 2},
        "PUT  /users": {RequestsPerSecond: 5, Burst: 5, MaxConcurrentRequests: 2},
    },
```

### Specify the domain name

```go
    RateLimits: meergo.RateLimits{
        "POST api.example.com/events": {RequestsPerSecond: 50, Burst: 100},
        "POST eu.example.com/events": {RequestsPerSecond: 30, Burst: 80},
    },
```
