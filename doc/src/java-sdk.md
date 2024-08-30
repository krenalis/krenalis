# Meergo Java SDK

The Meergo Java SDK lets you send customer event data from your Java server applications to your specified destinations.

## Step 1: Create a Source Java Connection

To create a source Java connection in Meergo:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Java** connector.
4. Click on **Add**.

## Step 2: Import the SDK

1. In the new created Java connection, navigate to **Settings**.
2. Select **Write Keys**.
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

## Step 3: Add an Action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Java connection you just created and click on **Actions**.
2. Under the **Import Events** action, click on **Add**.
3. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

## Step 4: Test the integration

1. Go to the Java connection you just created and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](./events.md) for more information on the supported event types.

### License

The Meergo Java SDK is released under the MIT license.
