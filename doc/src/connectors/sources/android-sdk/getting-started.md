{% extends "/layouts/doc.html" %}
{% macro Title string %}Getting started{% end %}
{% Article %}

# Getting started

This guide provides clear instructions for integrating the Android SDK into Android applications.

## Using the SDK

- [1. Create a source Android connection](#1-create-a-source-android-connection)
- [2. Import the SDK](#2-import-the-sdk)
- [3. Add an action](#3-add-an-action)

### 1. Create a source Android connection

To create a source Android connection in Meergo:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Android** source; you can use the search bar at the top or filter by category.
4. Click on the **Android** connector. A panel will open on the right with information about **Android**.
5. Click on **Add source**. The `Add Android source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Strategy** field, choose the strategy with which anonymous users will be treated.
8. Click **Add**.

### 2. Import the SDK

To integrate the Android SDK inside your application:

1. Add the dependency to your `build.gradle`. Make sure to replace `<latest_version>` with the latest version of the SDK.

    **Kotlin**

    ```kotlin
    repositories {
      mavenCentral()
    }
    dependencies {
      implementation 'com.meergo.analytics.kotlin:android:<latest_version>'
    }
    ```

   **Java**

    ```java
    repositories {
      mavenCentral()
    }
    dependencies {
      implementation 'com.meergo.analytics.kotlin:android:<latest_version>'
    }
    ```

2. Initialize and configure the client. You can find the write key in Meergo inside the Android connection in **Settings > Event write keys**. See [Options](options) for the list of configuration options.

    **Kotlin**

    ```kotlin
    import com.meergo.analytics.kotlin.android.Analytics
    import com.meergo.analytics.kotlin.core.*

    Analytics("YOUR_WRITE_KEY", applicationContext) {
      trackApplicationLifecycleEvents = true
      flushAt = 3
      flushInterval = 10
      // ...other config options
    }
    ```

    **Java**

    ```Java
    AndroidAnalytics analytics = AndroidAnalyticsKt.Analytics(BuildConfig.YOUR_WRITE_KEY, getApplicationContext(), configuration -> {
      configuration.setFlushAt(1);
      configuration.setCollectDeviceId(true);
      configuration.setTrackApplicationLifecycleEvents(true);
      configuration.setTrackDeepLinks(true);
      //...other config options
      return Unit.INSTANCE;
    });

    JavaAnalytics analyticsCompat = new JavaAnalytics(analytics);
    ```

3. Add the required permissions to `AndroidManifest.xml` (if they are not yet present).
    ```xml
    <!-- Required for internet. -->
    <uses-permission android:name="android.permission.INTERNET"/>
    <uses-permission android:name="android.permission.ACCESS_NETWORK_STATE"/>
    ``` 

### 3. Add an action

When the Android SDK is imported in your application, you can choose to collect only the events, or import the users, or both:

1. Go to the Android connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
5. Enable the action by toggling the switch in the **Enabled** column.
