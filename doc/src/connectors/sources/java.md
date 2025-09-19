{% extends "/layouts/doc.html" %}
{% macro Title string %}Java SDK data source{% end %}
{% Article %}

# Java SDK data source

The **Java** data source allows you to send customer event data using the **Java SDK** from your Java applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Create a source Java connection](#1-create-a-source-python-connection)
  - [2. Import the SDK in your Java application](#2-import-the-sdk-in-your-python-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [License](#license)

## Using the SDK

### 1. Create a source Java connection

First of all, you need a connection in Meergo that can receive events from the Java SDK. To do so:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Java** connector.
4. Click on **Add**.

### 2. Import the SDK in your Java application

1. In the new created Java connection, navigate to **Settings**.
2. Select **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. Add `com.meergo.analytics.java` to `pom.xml`:
    ```xml
    <dependency>
      <groupId>com.meergo.analytics.java</groupId>
      <artifactId>analytics</artifactId>
      <version>LATEST</version>
    </dependency>
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```java
    import com.meergo.analytics.Analytics;
    import com.meergo.analytics.messages.TrackMessage;

    final Analytics analytics =
        Analytics.builder("<write key>")
            .endpoint("<endpoint>")
            .build();

    analytics.enqueue(
        TrackMessage.builder("Test")
            .anonymousId(anonymousId)
            .userId(userId));
    ```

### 3. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Java connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the Java connection created at step 1 and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## License

The Meergo Java SDK is released under the MIT license.
