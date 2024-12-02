{% extends "/layouts/doc.html" %}
{% macro Title string %}Google Analytics data destination{% end %}
{% Article %}

# Google Analytics data destination

The **Google Analytics** data destination allows you to send the received events to Google Analytics.

Google Analytics is a web analytics service that provides insights into website traffic and user behavior. It helps track and analyze data on visitor interactions, traffic sources, and conversion rates, enabling informed decision-making to improve online performance.

### On this page

* [Add a Google Analytics data destination](#add-a-google-analytics-data-destination)

### Add a Google Analytics data destination

Before you can add a Google Analytics data destination, you need to create a private key in your Google Analytics account:

1. Log in to your <a href="https://360suite.google.com/" target="_blank">Google Analytics</a> account.
2. Click **Admin** at the bottom left of the page.
3. On the **Admin** page, click **Data stream**.
4. On the **Data stream** page, click the data stream to which you want to send events.
5. On the data stream details page, click **Measurement Protocol API secrets**.
6. On the **Measurement Protocol API secrets** page, click **Create**.
7. On the **Create new API secret** page, enter a name for the key to create, for example, "Meergo data destination."
8. Click **Create**.
9. On the **Measurement Protocol API secrets** page, copy the secret value corresponding to the previously created key.

Now proceed to add a Google Analytics data destination:

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **Google Analytics** destination; you can use the search bar at the top to assist you.
4. Next to the **Google Analytics** destination, click the **+** icon. The destination addition page will open.
5. Optional: In the **Name** field, enter a name for the destination to easily recognize it later.
6. In the **API Secret** field, enter the previously copied API secret.
7. Click **Add**.

Once the Google Analytics data destination is added, the **Actions** page will be displayed, indicating the actions required to send events to Google Analytics.