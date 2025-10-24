{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Source connectors{% end %}
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

## All sources

<ul class="grid-list connectors">
  <li><a href="sources/mailchimp">{{ Image("Mailchimp", "mailchimp.svg")}} Mailchimp</a></li>
  <li><a href="sources/hubspot">{{ Image("HubSpot", "hubspot.svg") }} HubSpot</a></li>
  <li><a href="sources/klaviyo">{{ Image("Klaviyo", "klaviyo.svg") }} Klaviyo</a></li>
  <li><a href="sources/stripe">{{ Image("Stripe", "stripe.svg") }} Stripe</a></li>
  <li><a href="sources/segment">{{ Image("Segment", "segment.svg") }} Segment</a></li>
  <li><a href="sources/rudderstack">{{ Image("RudderStack", "rudderstack.svg") }} RudderStack</a></li>
  <li><a href="sources/excel">{{ Image("Excel", "excel.svg") }} Excel</a></li>
  <li><a href="sources/csv">{{ Image("CSV", "csv.svg") }} CSV</a></li>
  <li><a href="sources/json">{{ Image("JSON", "json.svg") }} JSON</a></li>
  <li><a href="sources/parquet">{{ Image("Parquet", "parquet.svg") }} Parquet</a></li>
  <li><a href="sources/s3">{{ Image("S3", "s3.svg") }} S3</a></li>
  <li><a href="sources/http-get">{{ Image("HTTP GET", "http-get.svg") }} HTTP GET</a></li>
  <li><a href="sources/sftp">{{ Image("SFTP", "sftp.svg") }} SFTP</a></li>
  <li><a href="sources/filesystem">{{ Image("Filesystem", "filesystem.svg") }} Filesystem</a></li>
  <li><a href="sources/clickhouse">{{ Image("ClickHouse", "clickhouse.svg") }} ClickHouse</a></li>
  <li><a href="sources/postgresql">{{ Image("PostgreSQL", "postgresql.svg") }} PostgreSQL</a></li>
  <li><a href="sources/snowflake">{{ Image("Snowflake", "snowflake.svg") }} Snowflake</a></li>
  <li><a href="sources/mysql">{{ Image("MySQL", "mysql.svg") }} MySQL</a></li>
  <li><a href="sources/javascript-sdk">{{ Image("JavaScript", "javascript.svg") }} JavaScript</a></li>
  <li><a href="sources/dotnet">{{ Image(".NET", "dotnet.svg") }} .NET</a></li>
  <li><a href="sources/android-sdk">{{ Image("Android", "android.svg") }} Android</a></li>
  <li><a href="sources/go">{{ Image("Go", "go.svg") }} Go</a></li>
  <li><a href="sources/java">{{ Image("Java", "java.svg") }} Java</a></li>
  <li><a href="sources/nodejs">{{ Image("Node.js", "node.svg") }} Node.js</a></li>
  <li><a href="sources/python">{{ Image("Python", "python.svg") }} Python</a></li>
  <li><a href="sources/webhook">{{ Image("Webhook", "webhook.svg") }} Webhook</a></li>
</ul>

### Need more sources?

If you need additional destinations, [contact us](mailto:hello@meergo.com). You can also [create a custom connector](create-new-connector).
