{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Collect users from your apps{% end %}
{% Article %}

# Collect users from your apps
## Learn how to collect user data via SDK from the apps you develop.

Collect and unify user data from your applications using Meergo SDKs. Whether mobile, desktop, server-side, CLI, or IoT devices, each SDK lets you track events and identify users to maintain a consistent customer view.

> For integrating Meergo with websites and web apps, see the [websites integration](websites).

## Prerequisites

Before you start, make sure you have:

* Access to your app’s codebase and configuration.
* A basic understanding of the programming language your app uses.

These will help you install the Meergo SDK and connect your app without issues.

## Steps

### 1. Connect an application

Create a **source connection** for your app:

1. Go to the **Sources** page in your Meergo workspace.
2. Click **Add a new source ⊕**, then select the card for your platform or language.
3. Click **Add source...**.
4. (Optional) Rename the connection if needed.
5. Click **Add** to confirm.

The new source connection appears under **Sources** in the sidebar and can be reopened at any time.

### 2. Set up your application

Depending on your application's environment, install or import the Meergo SDK for your platform.

<!-- tabs id:sdk bg:white Own Applications Setup -->

#### Android

1. Add the dependency to your `build.gradle`:

    <!-- codeblocks sync:android Android -->

    ##### Kotlin

    ```kotlin
    repositories { mavenCentral() }
    dependencies { implementation 'com.meergo.analytics.kotlin:android:<latest_version>' }
    ```

   ##### Java

    ```java
    repositories { mavenCentral() }
    dependencies { implementation 'com.meergo.analytics.kotlin:android:<latest_version>' }
    ```

    <!-- end codeblocks -->

2. Initialize and configure the client:

   💡 You can find the _event write key_ in Meergo inside the Android source connection in **Settings > Event write keys**.

    <!-- codeblocks sync:android Android -->

    ##### Kotlin

    ```kotlin
    import com.meergo.analytics.kotlin.android.Analytics
    import com.meergo.analytics.kotlin.core.*

    Analytics("YOUR_WRITE_KEY", applicationContext) {
      trackApplicationLifecycleEvents = true
      flushAt = 3
      flushInterval = 10
    }
    ```

    ##### Java

    ```Java
    AndroidAnalytics analytics = AndroidAnalyticsKt.Analytics(BuildConfig.YOUR_WRITE_KEY, getApplicationContext(), configuration -> {
      configuration.setFlushAt(1);
      configuration.setCollectDeviceId(true);
      configuration.setTrackApplicationLifecycleEvents(true);
      configuration.setTrackDeepLinks(true);
      return Unit.INSTANCE;
    });

    JavaAnalytics analyticsCompat = new JavaAnalytics(analytics);
    ```

    <!-- end codeblocks -->

3. Add the following permissions to your `AndroidManifest.xml` if not already present:
    ```xml
    <!-- Required for internet. -->
    <uses-permission android:name="android.permission.INTERNET"/>
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE"/>
    ``` 

#### Apple

#### .NET

1. In Meergo, open your .NET source connector → **Settings > Event write keys**.
2. Copy the **event write key** and **endpoint**.
3. Install the package:
    ```sh
    $ Install-Package Meergo.Analytics.CSharp -Version <version>
    ```
4. Initialize the client:
    ```csharp
    using Meergo;
    using Meergo.Model;

    var config = new Config().SetEndpoint("<endpoint>");

    Analytics.Initialize("<event write key>", config);

    Analytics.Client.Identify(
        "703991475",
        new Dictionary<string, object>
        {
            { "email", "emily.johnson@example.com" },
            { "plan", "Enterprise" },
            { "company", "Acme Corp" },
            { "jobTitle", "Product Manager" },
            { "country", "United States" }
        },
        new Options().SetAnonymousId("ac3496a8-0782-4173-856f-2f7dd37d7f14")
    );

    Analytics.Client.Flush();
    Analytics.Client.Dispose();
    ```

#### Node.js

The Node SDK can be imported with `import` into Node projects, using ES6 modules.

1. In Meergo, open your Node.js source connection → **Settings > Event write keys**.
2. Copy the **event write key** and **endpoint**.
3. Install the SDK:
    ```sh
    $ npm install meergo-node-sdk --save
    ```
4. Import and use it:
    ```javascript
    import Analytics from 'meergo-node-sdk';
   
    const meergoAnalytics = new Analytics('<event write key>', '<endpoint>');
    meergoAnalytics.identify({
        anonymousId: "ac3496a8-0782-4173-856f-2f7dd37d7f14",   
        userId: "703991475",
        traits: {
            "email": "emily.johnson@example.com",
            "plan": "Enterprise",
            "company": "Acme Corp",
            "jobTitle": "Product Manager",
            "country": "United States"
        }
    });
    ```

#### Java

1. In Meergo, open your Java source connection → **Settings > Event write keys**.
2. Copy the **event write key** and **endpoint**.
4. Add `com.meergo.analytics.java` to `pom.xml`:
    ```xml
    <dependency>
      <groupId>com.meergo.analytics.java</groupId>
      <artifactId>analytics</artifactId>
      <version>LATEST</version>
    </dependency>
    ```
5. Import and use the SDK:
    ```java
    import com.meergo.analytics.Analytics;
    import com.meergo.analytics.messages.IdentifyMessage;

    Map<String, String> traits = new HashMap();
    traits.put("email", "emily.johnson@example.com");
    traits.put("plan", "Enterprise");
    traits.put("company", "Acme Corp");
    traits.put("jobTitle", "Product Manager");
    traits.put("country", "United States");

    final Analytics analytics =
        Analytics.builder("<event write key>")
            .endpoint("<endpoint>")
            .build();

    analytics.enqueue(
        IdentifyMessage.builder()
            .anonymousId("ac3496a8-0782-4173-856f-2f7dd37d7f14")
            .userId("703991475")
            .traits(traits));
    ```

#### Python

1. In Meergo, open your Python source connection → **Settings > Event write keys**.
2. Copy the **event write key** and **endpoint**.
3. Install the SDK:
    ```sh
    $ pip3 install meergo-analytics-python
    ```
4. Import and use the package:
    ```python
    import meergo.analytics as analytics

    analytics.write_key = '<event write key>'
    analytics.endpoint = '<endpoint>'

    analytics.identify(
        user_id='703991475',
        anonymous_id='ac3496a8-0782-4173-856f-2f7dd37d7f14',
        traits={
            'email': 'emily.johnson@example.com',
            'plan': 'Enterprise',
            'company': 'Acme Corp',
            'jobTitle': 'Product Manager',
            'country': 'United States'
        }
    )
    ```

#### Go

1. In Meergo, open your Go source connection → **Settings > Event write keys**.
2. Copy the **event write key** and **endpoint**.
3. In your Go module, go get the `"github.com/meergo/analytics-go"` package:
    ```sh
    $ go get github.com/meergo/analytics-go
    ```
5. Import and use the package:
    ```go
    import "github.com/meergo/analytics-go"

    client := analytics.New("<event write key>", "<endpoint>")
    client.Enqueue(analytics.Identify{
        AnonymousId: "ac3496a8-0782-4173-856f-2f7dd37d7f14",
        UserId:      "703991475",
        Traits: analytics.Traits{
            "email":    "emily.johnson@example.com",
            "plan":     "Enterprise",
            "company":  "Acme Corp",
            "jobTitle": "Product Manager",
            "country":  "United States",
        },
    })
   ```

<!-- end tabs -->

<script type="module" src="switch-sdk-screenshots.js"></script>
<script type="module">
import { updateTimestamps } from './update-event-timestamps.js';

updateTimestamps('#event-payload');
</script>

### 3. Debug the events

Use the **Event debugger** in your source connection to verify that events are received correctly.

1. In Meergo, open your source connection.
2. Go to the **Event debugger** tab.

    {{ Screenshot("Event debugger", "/docs/unification/collect-users/event-debugger.android.png", "sdk", 1067) }}

   It shows a live sample of the most recent events received for this source connection. Use it whenever you need to quickly confirm that events are arriving as expected and to inspect their contents in real time.

2. Run your application and trigger the code that calls `identify`. The event should appear almost immediately:

    {{ Screenshot("Event debugger", "/docs/unification/collect-users/event-debugger-identify.android.png", "sdk", 1067) }}

3. Click the event in the list to view its JSON payload and confirm the data sent from your app.

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
   <!-- code-expand id:event-payload -->

   If no event appears after a few seconds:
   - Check your application logs for errors.
   - Ensure your **event write key** and **endpoint** are correct.

    💡 See the [Event spec](/events/spec) for full details on the event schema.

### 4. Add an action to import users

In your **<span data-tabs="sdk">Android</span>** source connection, open the Actions tab.

Next to the **Import users from <span data-tabs="sdk">Android</span> into the warehouse** action, click **Add action...**

    {{ Screenshot("Import users into warehouse", "/docs/unification/collect-users/import-users-into-warehouse.android.png", "sdk", 1084) }}

This is an “Import users” action type, which transfers identified user profiles from your application into your warehouse.

> Each action defines how and when user data flows from your source into the warehouse. You can add multiple actions per connection to handle different data segments or destinations.

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

The new action will appear in your source connection and can be edited or disabled at any time.

## Continue reading

### Process collected users

Clean and standardize user data using visual mapping, JavaScript, or Python transformations.

{{ render "../_includes/manage-users-cards.html" }}
