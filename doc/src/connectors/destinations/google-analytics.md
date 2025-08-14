{% extends "/layouts/doc.html" %}
{% macro Title string %}Google Analytics data destination{% end %}
{% Article %}

# Google Analytics data destination

The **Google Analytics** data destination allows you to send the received events to [Google Analytics](https://developers.google.com/analytics).

Google Analytics is a web analytics service that provides insights into website traffic and user behavior. It helps track and analyze data on visitor interactions, traffic sources, and conversion rates, enabling informed decision-making to improve online performance.

In Meergo, it is possible to send events to Google Analytics using the [Measurement Protocol](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference?client_type=gtag). Once this data destination is configured, the events ingested by Meergo (e.g., from a website or server) are sent to Google Analytics.

## On this page

- [Add a Google Analytics data destination](#add-a-google-analytics-data-destination)
- [Events that can be sent](#events-that-can-be-sent)

## Add a Google Analytics data destination

Before you can add a Google Analytics data destination, you need to create a private key in your Google Analytics account:

1. Log in to your <a href="https://360suite.google.com/" target="_blank">Google Analytics</a> account.
2. Click **Admin** at the bottom left of the page.
3. On the **Admin** page, click **Data streams**.
4. On the **Data stream** page, click the data stream to which you want to send events.
5. On the data stream details page, copy and save the **Measurement ID** for later use.
6. On the data stream details page, click **Measurement Protocol API secrets**.
7. On the **Measurement Protocol API secrets** page, click **Create**.
8. On the **Create new API secret** page, enter a name for the key to create, for example, "Meergo data destination."
9. Click **Create**.
10. On the **Measurement Protocol API secrets** page, copy the secret value corresponding to the previously created key.

Now proceed to add a Google Analytics data destination:

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **Google Analytics** destination; you can use the search bar at the top or filter by category.
4. Next to the **Google Analytics** destination, click the **+** icon. The destination addition page will open.
5. (Optional) In the **Name** field, enter a name for the destination to easily recognize it later.
6. In the **Measurement ID** field, enter previously copied Measurement ID.
7. In the **API Secret** field, enter the previously copied API secret.
8. Click **Add**.

Once the Google Analytics data destination is added, the **Actions** page will be displayed, indicating the actions required to send events to Google Analytics.

## Events that can be sent

Meergo supports all events from Google Analytics' Measurement Protocol. For each event you wish to send, you can add a specific action.

[Here](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events) is a complete list of these events, including documentation for each one as well as for every field associated with each type of event.

Once the type of event you want to send has been determined, you can add the corresponding action from the Meergo interface.

## What to do if events don't show up

If the Google Analytics connector on Meergo sends events but you don't see them in your Google Analytics dashboard, try the following checks:

* Verify that the connector settings (Measurement ID and API secret) are correct. Google Analytics doesn’t validate credentials, so incorrect settings won’t trigger any error messages;

* Enter the action’s testing mode and try previewing some events. In testing mode, Google Analytics uses the Measurement Protocol debug server to check the request format (but not the credentials). If any errors are reported, fix them by adjusting the event transformation. Then exit testing mode, save the action, and try sending some events again.
