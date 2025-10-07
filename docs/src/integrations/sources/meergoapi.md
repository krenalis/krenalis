{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo API (Source){% end %}
{% Article %}

# Meergo API (Source)

The source connector Meergo API is a connector to interact directly with Meergo APIs from your application.

- [Description](#description)
- [How to send events](#how-to-send-events)
  - [1. Add source connection for Meergo API](#1-add-source-connection-for-meergo-api)
  - [2. Add an action](#2-add-an-action)
  - [3. Open the Live events page](#3-open-the-live-events-page)
  - [4. Send some events](#4-send-some-events)

## Description

This connector allows you to call Meergo APIs and import events and users directly from your application, regardless of the technology and programming language you use, since it allows you to interact directly via HTTP calls with Meergo APIs.

**This connector is useful in those cases where Meergo does not provide a dedicated SDK for your language**. It can also be used in those cases where you want to interact with Meergo directly via the command line, making the calls with tools like `curl`.

In this regard, it is therefore useful to see the [API reference documentation](/api/).

Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Meergo API data source, you will need any language or application that can make HTTP calls.

## How to send events

### 1. Add source connection for Meergo API

First of all, you need a connection in Meergo that can receive events from your application that sends HTTP requests. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Meergo API**; you can use the search bar at the top or filter by category.
4. Click on the connection for **Meergo API**. A panel will open on the right.
5. Click on **Add source...**. The `Add source connection for Meergo API` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Meergo API connection you just created and click on **Actions**.
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 3. Open the Live events page

1. Go to the source connection for  Meergo API created at step 1 and click on **Event debugger**.
2. Go on with the next step for sending events; you will see the incoming events in the **Live events** view.

### 4. Send some events

You can use any tool or language you like for sending HTTP requests containing the events to the Meergo API connector.

For example, if you're using `curl`:

```sh
$ curl <MEERGO ENDPOINT> \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <YOUR_WRITE_KEY>" \
    -d '[
    {
            "type": "track",
            "anonymousId": "2ab4facf-ca73-4361-b1b2-4472ce053122",
            "event": "page",
            "properties": {
                "answer": 42
            }
        }
    ]
'
```

You can get the values for `<MEERGO ENDPOINT>` and `<YOUR_WRITE_KEY>` by clicking on the Meergo API connection you created > Settings > Event write keys.

So, for example, you can send an event with `curl` like this:

```sh
$ curl http://localhost:2022/api/v1/events \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer 4qwwvXGRa0R2PeJsUwSv1juy2MRrY8bA" \
    -d '[
    {
            "type": "track",
            "anonymousId": "2ab4facf-ca73-4361-b1b2-4472ce053122",
            "event": "page",
            "properties": {
                "answer": 42
            }
        }
    ]
'
```

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.
