{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Collect users from websites{% end %}
{% Article %}

# Collect users from websites
## Learn how to collect users from your websites and web apps.

Collecting data directly from your websites and web apps lets you build accurate, real-time customer profiles from every user interaction. Use the Meergo JavaScript SDK to automatically capture, map, and unify this data across all your systems.

## Prerequisites

Before getting started, make sure you have everything you need:

* Access to the website pages you want to connect with Meergo.
* The ability to edit those pages or web application so you can integrate the Meergo JavaScript SDK.

## Steps

### 1. Connect a website

Create a source connection to start tracking user data from your site. This connection links your web pages to Meergo.  

1. Go to the **Sources** page of your Meergo workspace.
2. Click on **Add a new source ⊕** and click on the **JavaScript** card.
3. Click on **Add source...**.
4. Optionally rename the connection.
5. Click **Add** to confirm. 

The connection you just created is a source connection. You can access it later by clicking **Sources** section in the sidebar. If you manage multiple websites that share the same user base (for example, through a unified login), it's preferable to create a single source connection for all of them.

### 2. Set up your website

Choose **one of the following alternative methods** to integrate the SDK, depending on your project setup

<!-- tabs bg:white Website Setup -->

#### Snippet

💡 Use this method if your website doesn't use a build system and you can directly edit the HTML source.

1. In the new created JavaScript connection, copy the JavaScript snippet:

{{ Screenshot("JavaScript snippet", "/docs/unification/collect-users/javascript-snippet.png", "", 1745) }}

2. Paste it into all pages of your website between `<head>` and `</head>`.
3. As a test, add the following code to one of your pages to collect user data and send it to Meergo:

    ```html
    <script>
    meergo.identify("703991475", {
        email: "emily.johnson@example.com",
        plan: "Enterprise",
        company: "Acme Corp",
        jobTitle: "Product Manager",
        country: "United States"
    });
    </script>
    ```

   💡 For detailed SDK methods and configuration options, see the [JavaScript SDK](/sources/javascript-sdk).

#### Import

💡 Use this method if your project uses ES6 modules or TypeScript and is bundled for the browser.

The JavaScript SDK can be imported with `import` into TypeScript and JavaScript projects, using ES6 modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the _event write key_ and the _endpoint_.
4. In your project, install the `meergo-javascript-sdk` npm package:
    ```sh
    $ npm install meergo-javascript-sdk --save
    ```
5. Import and use the SDK, replacing `<event event write key>` and `<endpoint>` respectively with the previously copied _event write key_ and _endpoint_:
    ```javascript
    import Meergo from 'meergo-javascript-sdk';

    const meergo = new Meergo('<event write key>', '<endpoint>');

    // As a test, collect user data and send it to Meergo:
    meergo.identify("703991475", {
        email: "emily.johnson@example.com",
        plan: "Enterprise",
        company: "Acme Corp",
        jobTitle: "Product Manager",
        country: "United States"
    });
    ```

   💡 For detailed SDK methods and configuration options, see the [JavaScript SDK](/sources/javascript-sdk).

#### Require

💡 Use this method if your project uses CommonJS modules and is bundled for the browser.

The JavaScript SDK can be imported with `require` into JavaScript projects, using CommonJS modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your project, install the `meergo-javascript-sdk` npm package:
    ```sh
    $ npm install meergo-javascript-sdk --save
    ```
5. Import and use the SDK, replacing `<event write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```javascript
    const { Meergo } = require('meergo-javascript-sdk');

    const meergo = new Meergo('<event write key>', '<endpoint>');

    // As a test, collect user data and send it to Meergo:
    meergo.identify("703991475", {
        email: "emily.johnson@example.com",
        plan: "Enterprise",
        company: "Acme Corp",
        jobTitle: "Product Manager",
        country: "United States"
    });
    ```

    💡 For detailed SDK methods and configuration options, see the [JavaScript SDK](/sources/javascript-sdk).

<!-- end tabs -->

### 3. Debug the events

1. In the source JavaScript connection, click on the **Event debugger** tab: 

    {{ Screenshot("Event debugger", "/docs/unification/collect-users/event-debugger.javascript.png", "", 1067) }}

    The **Event debugger** shows a sample of events currently received for the current connection. Use it whenever you need to quickly confirm that events are arriving as expected and to inspect their contents in real time. 

2. Open your website page where you added the `identify` call, the event should appear almost immediately:

    {{ Screenshot("Event debugger", "/docs/unification/collect-users/event-debugger-identify.javascript.png", "", 1067) }}

    If the event **does not appear** after a few seconds, open your browser's console, reload the page, and check that no errors occurred and that the event was successfully sent.

3. Click the collected event in the **Event debugger** list to view its JSON payload, which contains the data sent by the browser. The following example shows what a typical event payload looks like:

    ```json
    {
        "anonymousId": "b27c5d9f-92a7-4d30-b21a-4df21a6872c2",
        "context": {
            "browser": {
                "name": "Safari",
                "version": "17.2.1"
            },
            "ip": "172.91.24.57",
            "library": {
                "name": "meergo.js",
                "version": "1.0.0"
            },
            "locale": "en-US",
            "os": {
                "name": "macOS",
                "version": "14.5"
            },
            "page": {
                "path": "/dashboard",
                "title": "User Dashboard",
                "url": "https://app.example.com/dashboard"
            },
            "screen": {
                "width": 3024,
                "height": 1964,
                "density": 2
            },
            "session": {
                "id": "1766272512048"
            },
            "timezone": "America/Los_Angeles",
            "userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15"
        },
        "messageId": "f9189a52-b37b-4d7d-9f2d-08b91d85fa9c",
        "receivedAt": "2025-10-20T16:15:24.340Z",
        "sentAt": "2025-10-20T16:15:24.327Z",
        "originalTimestamp": "2025-10-20T16:15:23.992Z",
        "timestamp": "2025-10-20T16:15:24.010Z",
        "traits": {
            "email": "emily.johnson@example.com",
            "plan": "Enterprise",
            "company": "Acme Corp",
            "jobTitle": "Product Manager",
            "country": "United States"
        },
        "type": "identify",
        "userId": "703991475"
   }
   ```
   <!-- code-expand -->

    💡 You can read the [Event spec](/events/spec) for full details on the schema of an event after it has been received by Meergo.

### 4. Add an action to import users

1. Click on the **Actions** tab of the JavaScript connection.
2. Next to the **Import users into warehouse** action, click **Add action...**

    {{ Screenshot("Import users into warehouse", "/docs/unification/collect-users/import-users-into-warehouse.javascript.png", "", 1084) }}

### 5. Filter events

The filter selects which users to import based on the collected events: 

{{ Screenshot("Filter events", "/docs/unification/collect-users/filter-users-via-events.png", "", 1480) }}

It is preset to import users only if the event is of type `identify`—with or without traits—or if it is not an `identify` event but includes traits. For now, you can leave it as configured. It's recommended to adjust this filter only after you've gained experience with event handling.

### 6. Transformation

The **Transformation** section lets you populate your Customer Model properties using user traits from collected events. You can choose between a **visual mapping interface** or **advanced transformations** written in JavaScript or Python.

You have full control over which properties to map—assign only those relevant to your business context and leave others unassigned when no matching values are available.

{{ Screenshot("Visual mapping", "/docs/unification/collect-users/user-via-event-visual-mapping.png", "", 2168) }}

For complete details on how transformations work for harmonization, see how to [harmonize data](/unification/harmonization).

### 7. Save your changes

When you're done, click **Add** (or **Save** if you're editing an existing action).

## Continue reading

### Process collected users

Learn how to manage, clean, and unify the users you’ve collected to create a single, accurate, and consistent customer view.

{{ render "../_includes/manage-users-cards.html" }}
