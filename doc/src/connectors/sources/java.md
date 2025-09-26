{% extends "/layouts/doc.html" %}
{% macro Title string %}Java SDK (Source){% end %}
{% Article %}

# Java SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Java_SDK-blue?logo=github)](https://github.com/open2b/analytics-java)

The source connector for Java allows you to send customer event data using the Java SDK from your Java applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Add source connection for Java](#1-add-source-connection-for-java)
  - [2. Import the SDK in your Java application](#2-import-the-sdk-in-your-java-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [SDK source code](#sdk-source-code)
- [License](#license)

## Using the SDK

### 1. Add source connection for Java

First of all, you need a connection in Meergo that can receive events from the Java SDK. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Java**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Java**. A panel will open on the right with information about **Java**.
5. Click on **Add source**. The `Add Java source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Import the SDK in your Java application

1. In the new created connection for Java, navigate to **Settings**.
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

1. Go to the source connection for Java you just created and click on **Actions**.
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the source connection for Java created at step 1 and click on **Event debugger**.
2. Execute your application to send some events.
3. Click on a received event in the **Live events** section to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## SDK source code

The source code of the Meergo Java SDK is [available on GitHub](https://github.com/open2b/analytics-java).

## License

The Meergo Java SDK is released under the MIT license.
