{% extends "/layouts/doc.html" %}
{% macro Title string %}Rate limits and retry{% end %}
{% Article %}

# Rate limits and retry

Endpoint groups define how HTTP requests are collectively subject to rate limiting and retry policies. Each group specifies one or more patterns (using Go's [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux) syntax) that determine which requests are tracked together by the same rate limiter instance. Requests matching any pattern within the same group consume from the same quota and are counted together toward the group's rate limits.

This makes it possible to enforce independent rate limits for distinct sets of endpoints—even if they share the same rate limiting or retry policy settings—by simply assigning them to separate groups. For example, API endpoints deployed in different regions can be grouped separately to ensure rate limiting is applied independently per region, even if their configuration is otherwise identical.

```go
// EndpointGroup defines a group of API endpoints—specified by one or more
// patterns using the same syntax as http.ServeMux. All HTTP requests matching
// any of the given patterns will be subject to the specified rate limiting and
// retry policies.
type EndpointGroup struct {
    Patterns     []string    // patterns (e.g. "GET /api/users/", "api.example.com/v1/items") matched by ServeMux-style matching
    RequireOAuth bool        // require OAuth authentication
    RateLimit    RateLimit   // rate limiting configuration applied to all matching requests
    RetryPolicy  RetryPolicy // retry policy for handling failed requests to these endpoints
}
```

The following example demonstrates how to configure endpoint groups to enforce separate rate limits for different sets of Klaviyo API endpoints:

```go
[]meergo.EndpointGroup{
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
}
```

In this configuration, requests matching `/api/event-bulk-create-jobs` share a dedicated rate limiter that allows up to 2.5 requests per second, with a burst capacity of 10. Requests matching `/api/profiles/` are managed by a separate rate limiter with its own, higher threshold. Even though both groups use the same retry policy, each group is rate-limited independently. This means that traffic to one set of endpoints will not affect the rate limits available to the other, preventing a spike on one endpoint from causing throttling or errors on another.

## Patterns

Each endpoint group is defined by one or more patterns that determine which HTTP requests are included in the group. Patterns follow the syntax of Go's [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux), supporting both path prefixes and optional host qualifiers. You can also specify an HTTP method as a prefix, such as `"GET /api/users"`, to match only requests using that method for the given path.

A pattern that ends with a slash (`"/"`) matches any request whose path starts with that prefix. For example, `"/api/profiles/"` matches `/api/profiles/123` and `/api/profiles/foo/bar`. A pattern without a trailing slash matches only the exact path specified, so `"/api/profiles"` matches only requests to `/api/profiles` and not `/api/profiles/123`.

If the `Patterns` slice is omitted or left empty for an endpoint group, it defaults to `[]string{"/"}`. This means the group will match all requests, regardless of path, method, or host.

> See [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux) for more details on patterns.

## Rate limit

Meergo applies client-side rate limits to avoid hitting the limits imposed by an API. When you register your connector you must provide them using the `RateLimit` field of `EndpointGroup`.

A `RateLimit` defines how quickly requests can be made and optionally how many can be in flight at the same time:

```go
type RateLimit struct {
    RequestsPerSecond     float64 // max allowed requests per second (> 0)
    Burst                 int     // number of requests that can be sent in a burst (> 0)
    MaxConcurrentRequests int     // maximum number of concurrent requests (>= 0)
}
```

`RequestsPerSecond` is the average rate permitted for the matched requests. `Burst` allows temporarily exceeding that rate after a period of low usage. When set, `MaxConcurrentRequests` limits how many of those requests can be running at the
same time; use `0` for no concurrency limit.

### Examples

#### Single rate limit for all calls

```go
EndpointGroups: []meergo.EndpointGroup{
    {
        RateLimit: meergo.RateLimit{RequestsPerSecond: 10, Burst: 20},
    },
},
```

#### Rate limits based on HTTP method

```go
EndpointGroups: []meergo.EndpointGroup{
    {
        Patterns:  []string{"GET /users"},
        RateLimit: meergo.RateLimit{RequestsPerSecond: 12, Burst: 18},
    },
    {
        Patterns:  []string{"POST /users", "PUT /users"},
        RateLimit: meergo.RateLimit{RequestsPerSecond: 5, Burst: 5, MaxConcurrentRequests: 2},
    },
},
```

#### Specify the domain name

```go
EndpointGroups: []meergo.EndpointGroup{
    {
        Patterns:  []string{"api.example.com/events"},
        RateLimit: meergo.RateLimit{RequestsPerSecond: 50, Burst: 100},
    },
    {
        Patterns:  []string{"eu.example.com/events"},
        RateLimit: meergo.RateLimit{RequestsPerSecond: 30, Burst: 80},
    },
},
```

## Retry policy

API connectors can use retry policies to manage retries of idempotent HTTP requests based on the API's response status code.

You only need to set up the retry policy when you register the connector. After that, use the provided HTTP client to make calls, and Meergo will handle retries using the policy you set.

Here's an example of how to register a connector with two retry strategies: one that uses the "Retry-After" header for status code 429 (Too Many Requests) and another that uses an exponential backoff strategy for status codes 500 (Internal Server Error) and 503 (Service Unavailable).

```go
meergo.RegisterAPI(meergo.APISpec{
    EndpointGroups: []meergo.EndPointGroup{
        RetryPolicy: meergo.RetryPolicy{
            "429":     meergo.RetryAfterStrategy(),
            "500 503": meergo.ExponentialStrategy(meergo.NetFailure, 200 * time.Millisecond),
        },
        // other fields are omitted.
    },
    // other fields are omitted.
}, New)
```

The `RetryPolicy` field specifies the retry policy for the connector. It maps one or more HTTP status codes to the corresponding retry strategy. If an idempotent HTTP request fails, Meergo will look up the status code in the connector policy and retry the request using the corresponding strategy.

You can use the strategies provided by Meergo, so you do not have to implement it, or create your own. If the API documentation does not specify how to handle errors, do not set a retry policy. Meergo will use a default policy in that case.

### Retriable requests

Only idempotent requests can be retried. The `Do` method of the HTTP client passed to a connector decides whether a request is retriable using the same rules as Go's standard library (see [http.Transport](https://pkg.go.dev/net/http#Transport) for details).

A request is considered retriable if:

* it is **idempotent**, and
* it either **has no body** or provides a `Request.GetBody` function to recreate the body if needed.

A request is idempotent if:

* its HTTP method is `GET`, `HEAD`, `OPTIONS`, or `TRACE`, or
* it includes the header `Idempotency-Key` or `X-Idempotency-Key`.

If the idempotency header is present but is empty (`nil` or empty slice), the request is still treated as idempotent — but the header won't be sent over the network.

#### Example

If the API supports idempotency and requires an idempotency key in the header called `Idempotency-Key`, you can do this:

```go
// Mark the request as idempotent.
req.Header.Set("Idempotency-Key", key)
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

The `Request.GetBody` function is called if the request needs to be retried and must return the original request body.

If you want a random UUID to use as the idempotency key, you can use:  

```go
// Mark the request as idempotent.
req.Header.Set("Idempotency-Key", meergo.UUID())
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

#### Idempotent events

Events are usually idempotent, meaning APIs don't require an idempotency key to receive them. Instead, an API generally uses each event's unique ID for deduplication.

However, even if an idempotency key is not needed, the request must still be marked as idempotent. You can mark the request as idempotent without sending the header like this:

```go
// Mark the request as idempotent.
req.Header["Idempotency-Key"] = nil
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

Setting the header value to `nil` tells the HTTP client that the request is idempotent and should be retried on failure, but the header itself will not be sent to the API.

### Retry strategies

Meergo offers **Constant**, **Exponential**, **RetryAfter**, and **Header** strategies for managing retries. Jitter is automatically added to the wait time calculated by strategies to introduce variability.

#### Constant strategy

This strategy waits a fixed amount of time before retrying. For example, to wait 1 second:

```go
meergo.ConstantStrategy(meergo.NetFailure, time.Second)
```

#### Exponential strategy

This strategy increases the wait time exponentially, starting from a base value. For example, to start with 100ms:

```go
meergo.ExponentialStrategy(meergo.Slowdown, 100 * time.Millisecond)
```

#### RetryAfter strategy

This strategy uses the "Retry-After" header from the response. The header can specify either the number of seconds to wait or a specific date and time (as specified in [RFC 9110](https://httpwg.org/specs/rfc9110.html#field.retry-after)). The strategy returned by the `RetryAfter` function handles both:

```go
meergo.RetryAfterStrategy()
```

The strategy returned by `RetryAfterStrategy` also supports fractional seconds.

#### Header strategy

This strategy reads the wait time from a specific header. The `HeaderStrategy` function takes a header name and a function to parse the header value and returns the strategy. For example, to use the "Wait-Seconds" header and parse it as a duration:

```go
meergo.HeaderStrategy(meergo.Slowdown, "Wait-Seconds", time.ParseDuration)
```

The `parse` function has the following type:

```go
func(s string) (time.Duration, error)
```

If the `parse` function returns an error, the request will not be retried. If `parse` is `nil`, the strategy defaults to the behavior of the RetryAfter strategy.


#### Custom strategy

To create a custom strategy, implement a function with the following type:

```go
// RetryStrategy represents a strategy for determining retry behavior.
// It returns a FailureReason and the duration to wait before the next attempt,
// based on the HTTP response from the previous attempt and the number of
// retries made. retries parameter starts at 0 before the first retry and
// increments by 1 on each retry.
//
// If the returned waitTime is negative, it is considered zero.
type RetryStrategy func(res *http.Response, retries int) (reason FailureReason, waitTime time.Duration)
```

This function takes the failed response and the number of retries so far, and returns a failure reason and time to wait before retrying. Parameters include: 

- `res`: The HTTP response from the API.
- `retries`: The number of times the request has been retried, starting from 0.
- `reason`: The reason of the failure.
- `waitTime`: The amount of time to wait before retrying.

Do not add jitter to the wait time; it is added automatically.

#### Failure reasons

The failure reason determines whether a request is eligible for retry and how it affects the rate control system: 

* `meergo.PermanentFailure`: the request cannot be retried, and the error contributes to the  API's error rate.
* `meergo.NetFailure`: the request can be retried, and the error still counts toward the error rate.
* `meergo.Unauthorized`: the request can be retried if the connector supports OAuth.
* `meergo.Slowdown`: the request can be retried. Upon receiving the first `Slowdown`, request rate is significantly reduced; further slowdowns also contribute to the error rate.
* `meergo.RateLimited`: the request can be retried, but future requests are suspended until the returned wait time has passed.
