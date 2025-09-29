{% extends "/layouts/doc.html" %}
{% macro Title string %}Sources{% end %}
{% Article %}

# Sources

**Sources**, also known as **data sources**, are where **Meergo** collects information about customers and events. They form the foundation for analyzing your customer data. Sources can include websites, mobile apps, servers, databases, files, and cloud services, such as marketing tools or CRM systems.

When you add a source to Meergo, the system automatically starts gathering data from it. For example, if your source is a website, Meergo tracks events like page views and clicks, collecting data from both anonymous users and those who are logged in. If the source is a cloud application, Meergo extracts information about users within the app.

The way data is collected varies depending on the type of source: websites, mobile apps, and servers send data in real-time, allowing Meergo to access user events and information immediately. In contrast, sources like cloud apps, databases, and file storage are read periodically, ensuring that the data is updated, even if not in real-time.

### **Unifying sources**

One of the main benefits of Meergo is its ability to **unify** data from different sources. When multiple sources (like a website, a mobile app, and a CRM system) are connected, Meergo can combine the information collected from each, creating a complete and consistent view of customers.

This means that events tracked across different platforms, like a purchase made on a website and a visit from a mobile device, can be combined into a single customer profile. Unifying data helps create a 360-degree view of customer interactions with your business, improving analysis and personalizing experiences.

### **Integration with data warehouse and destinations**

Customer and event data collected from various sources can be stored in real-time in the **data warehouse** of Meergo's workspace, making the information available for advanced analysis and reporting, as well as accessible on Meergo’s dashboard. Additionally, events can be sent immediately to a **destination**, like a cloud application, to be processed and used in real-time.

## In this section

* [Add and update sources](add-update-sources)
