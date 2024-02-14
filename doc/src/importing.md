# Importing

## Importing user identities from events

In order for a user identity to be imported from an event, it is necessary that:

* the event has at least one key in either one of the two JSON objects `traits` or `context.traits`
* and, in case of anonymous events, the mapping applied to the event must result in at least one property.
