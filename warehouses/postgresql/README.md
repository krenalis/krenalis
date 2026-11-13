# PostgreSQL warehouse

## Cancellation

When the pool destroys a connection that pgx has already closed, for example
after a context cancellation, it makes additional best-effort
cancellation attempts for at most 10 seconds. Each attempt has a one-second
limit. Retries start after 50 milliseconds, with increasing delays capped at
one second, and stop when the driver's local cleanup finishes. Healthy
connections are closed normally, without additional cancellation attempts.

The caller still returns promptly after cancellation. Completion of local
cleanup does not confirm that the server stopped the query. If the cancellation
channel is unavailable, the server may continue executing. The outcome of a
canceled autocommit write can remain unknown: the write may have committed
before the cancellation took effect.
