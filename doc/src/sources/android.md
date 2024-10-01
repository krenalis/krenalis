# Android data source

The **Android** data source allows you to receive events, including user information, from an Android app. The received events can be:

* Sent to destinations, particularly applications that can receive events.
* Stored in the workspace's data warehouse.
* Extracted to identify users, both recognized and anonymous, for unification in the workspace's data warehouse.

The Android data source requires the [Android SDK](../android-sdk.md) from Meergo, which provides functionalities for sending various types of events and ensures seamless integration with the Meergo platform.

> The [Android SDK](../android-sdk.md) is an open-source Android library licensed under MIT, compatible with the Segment's Analytics-Kotlin SDK.

### On this page

* [Add an Android data source](#add-an-android-data-source)

### Add an Android data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Android** source; you can use the search bar at the top to help you.
4. Next to the **Android** source, click the **+** icon. The source addition page will open.
5. Optional: In the **Name** field, enter a name for the source to easily recognize it later. This could be the name of the Android app, for example.
6. Optional: From the **Strategy** menu, select a strategy. You can change it later if needed.
7. Click **Add**.

Once the Android data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the events received from this source.

