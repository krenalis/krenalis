{% extends "/layouts/doc.html" %}
{% macro Title string %}Importing User Identities{% end %}
{% Article %}

# Importing user identities

When importing users from a connection, be it an app, file, database or a connection that sends events, these users are imported into Meergo in the form of "user identities" and are associated with that connection.

The **Identity Resolution** procedure will then evaluate whether or not these user identities correspond to the same user, thus establishing and updating the actual users within Meergo.

## Behavior

> NOTE: this section needs to be reviewed.

When a user identity is imported from a connection, the identities are updated like this:

* If it is **imported for the first time**, a new identity is created
* If it **has already been imported** previously, the properties of the already imported identity are overwritten with those of the new one (including overwriting values which are null)
