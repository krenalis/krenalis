{% extends "/layouts/doc.html" %}
{% macro Title string %}Mixpanel data destination{% end %}
{% Article %}

# Mixpanel data destination

The **Mixpanel** data destination allows you to send events to Mixpanel.

Mixpanel is an analytics platform that tracks user interactions across digital platforms. It provides tools for event tracking, funnel analysis, retention monitoring, and cohort analysis, helping businesses optimize user experiences and make data-driven decisions.

### On this page

* [Add a Mixpanel data destination](#add-a-mixpanel-data-destination)

### Add a Mixpanel data destination

Before you can add a Mixpanel data destination, you need to retrieve the **Project ID** and **Project Token** for your Mixpanel project. To do this:

1. Log in to your <a href="https://mixpanel.com/" target="_blank">Mixpanel</a> account.
2. From the top-left menu, select the project you want to send events to.
3. Click the **Settings** icon ⚙️ at the bottom left, then go to **Settings** > **Project Settings**.
4. In the **Project Details** section, copy the **Project ID**. It looks like *12345678*.
5. In the **Access Keys** section, also copy the **Project Token**. It looks like *09f7e02f1290be211da707a266f153b3*.

Now proceed to add a Mixpanel data destination:

1. In Meergo Admin console, go to **Connections > Destinations**.
2. On the **Sources** page, click **Add New source**.
3. Search for **Mixpanel**; you can use the search bar at the top or filter by category.
4. Click on the **Mixpanel** connector. A panel will open on the right with information about **Mixpanel**.
5. Click on **Add destination**. The `Add Mixpanel API destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **Project ID** field, enter the Project ID you copied earlier.
8. In the **Project Token** field, enter the Project Token you copied earlier.
9. (Optional) Select **Use the European Endpoint** if needed.
10. Click **Add**.
