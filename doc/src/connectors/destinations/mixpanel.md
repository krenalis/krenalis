{% extends "/layouts/doc.html" %}
{% macro Title string %}Mixpanel data destination{% end %}
{% Article %}

# Mixpanel data destination

The **Mixpanel** data destination allows you to send events to Mixpanel.

Mixpanel is an analytics platform that tracks user interactions across digital platforms. It provides tools for event tracking, funnel analysis, retention monitoring, and cohort analysis, helping businesses optimize user experiences and make data-driven decisions.

### On this page

* [Add a Mixpanel data destination](#add-a-mixpanel-data-destination)
* [Track user sessions](#track-user-sessions)

### Add a Mixpanel data destination

#### Create a Mixpanel project

If you have not yet created a project in Mixpanel, or if you want to create a new one to use with Meergo, follow these steps:

1. Log in to your <a href="https://mixpanel.com/" target="_blank">Mixpanel</a> account.
2. From the top-left menu, click **Create project**.
3. In the **Project Name** field, enter a name for the project so you can easily recognize it later.
4. Select the data storage location, your timezone, and the organization. 
5. Click **Create**.
6. (Optional, but recommended) Make sure the project is using the Simplified ID Merge API:
   * Click the **Settings** icon ⚙️ at the bottom left, then go to **Settings** > **Project Settings** > **Identity Merge**.
   * On the **Identity Merge** page, verify that the project is using the Simplified ID Merge API. If not, click the button to switch to **Simplified API**.
7. From the top-left menu, click **Overview**.
8. In the **Project Details** section, copy the **Project ID**. It looks like *12345678*.
9. In the **Access Keys** section, also copy the **Project Token**. It looks like *09f7e02f1290be211da707a266f153b3*. 

#### Retrieve Project ID and Project Token from an existing project

Before you can add a Mixpanel data destination, you need to retrieve the **Project ID** and **Project Token** for your Mixpanel project. To do this:

1. Log in to your <a href="https://mixpanel.com/" target="_blank">Mixpanel</a> account.
2. From the top-left menu, select the project you want to send events to.
3. Click the **Settings** icon ⚙️ at the bottom left, then go to **Settings** > **Project Settings**.
4. (Optional, but recommended if the project is empty) Make sure the project is using the Simplified ID Merge API:
   * Click the **Settings** icon ⚙️ at the bottom left, then go to **Settings** > **Project Settings** > **Identity Merge**.
   * On the **Identity Merge** page, check that this project is using the simplified ID Merge API. Otherwise, click on the button to switch to **Simplified API**. 
5. From the top-left menu, click **Overview**.
6. In the **Project Details** section, copy the **Project ID**. It looks like *12345678*.
7. In the **Access Keys** section, also copy the **Project Token**. It looks like *09f7e02f1290be211da707a266f153b3*.

#### Add data destination

Now proceed to add a Mixpanel data destination:

1. In Meergo Admin console, go to **Connections > Destinations**.
2. On the **Sources** page, click **Add New source**.
3. Search for **Mixpanel**; you can use the search bar at the top or filter by category.
4. Click on the **Mixpanel** connector. A panel will open on the right with information about **Mixpanel**.
5. Click on **Add destination**. The `Add Mixpanel API destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **Project ID** field, enter the Project ID you copied earlier.
8. In the **Project Token** field, enter the Project Token you copied earlier.
9. Select **This project is configured to use Mixpanel's EU data residency** if your project is set up for EU data residency.
10. Click **Add**.

### Track user sessions

If you want the user sessions tracked by Meergo to also appear in Mixpanel, you just need to adjust a few settings in your Mixpanel project. This ensures that events coming from Meergo stay connected to the same session as the original events. Thanks to <a href="https://docs.mixpanel.com/docs/features/sessions" target="_blank">Mixpanel sessions</a>, related actions are grouped together into one visit, so you can see the full picture of what users do instead of looking at separate events.

1. Log in to your <a href="https://mixpanel.com/" target="_blank">Mixpanel</a> account.
2. From the top-left menu, select the project you want to send events to.
3. Click the **Settings** icon ⚙️ at the bottom left, then go to **Settings** > **Project Settings**.
4. In the **Session Settings** section, next to **Definition Type**, click the pencil icon on the right.
5. In the **Definition Type** field, select **Property Based**.
6. In the **Session ID Property Name** field, select or enter the property **session_id**.
7. Click **Update**.
