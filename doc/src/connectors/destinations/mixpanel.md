{% extends "/layouts/doc.html" %}
{% macro Title string %}Mixpanel data destination{% end %}
{% Article %}

# Mixpanel data destination

The **Mixpanel** data destination allows you to send events to Mixpanel.

Mixpanel is an analytics platform that tracks user interactions across digital platforms. It provides tools for event tracking, funnel analysis, retention monitoring, and cohort analysis, helping businesses optimize user experiences and make data-driven decisions.

### On this page

* [Add a Mixpanel data destination](#add-a-mixpanel-data-destination)

### Add a Mixpanel data destination

Before you can add a Mixpanel data destination, you need to create a service account in your Mixpanel account:

1. Log in to your <a href="https://mixpanel.com/" target="_blank">Mixpanel</a> account.
2. Click the **Settings** icon at the top right and then click **Organization Settings**.
3. On the **Organization** page, click **Service Accounts**.
4. Click **Add Service Account**.
5. Under **NAME**, enter a descriptive name for the account (e.g., “meergo“.)
6. Under **ORGANIZATION ROLE**, select **Member**.
7. Under **PROJECTS**, choose the project you want to receive events for.
8. Click **Create**.
9. Under **PROJECT ROLE**, select **Messenger**.
10. Copy **Username** and **Secret**. Note that after the dialog is closed, you will never be able to access to the secret again.
11. Click **Done**.

Now proceed to add a Mixpanel data destination:

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Sources** page, click **Add New source**.
3. Search for the **Mixpanel** destination; you can use the search bar at the top to assist you.
4. Next to the **Mixpanel** destination, click the **+** icon.
5. On the **Add Mixpanel destination connection** page, in the **Name** field, enter a name for the destination to easily recognize it later.
6. In the **Project ID** field, enter the ID number of the project.
7. In the **Service Account Username** field, enter the username of the service account.
8. In the **Service Account Secret** field, enter the secret of the service account.
9. Click **Add**.
