{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Destination connectors{% end %}
{% Article %}

# Destinations

<ul class="grid-list connectors">
  <li><a href="destinations/mailchimp">{{ Image("Mailchimp", "mailchimp.svg") }}Mailchimp</a></li>
  <li><a href="destinations/hubspot">{{ Image("HubSpot", "hubspot.svg") }}HubSpot</a></li>
  <li><a href="destinations/klaviyo">{{ Image("Klaviyo", "klaviyo.svg") }}Klaviyo</a></li>
  <li><a href="destinations/stripe">{{ Image("Stripe", "stripe.svg") }}Stripe</a></li>
  <li><a href="destinations/mixpanel">{{ Image("Mixpanel", "mixpanel.svg") }}Mixpanel</a></li>
  <li><a href="destinations/google-analytics">{{ Image("Google Analytics", "google-analytics.svg") }}Google Analytics</a></li>
  <li><a href="destinations/excel">{{ Image("Excel", "excel.svg") }}Excel</a></li>
  <li><a href="destinations/csv">{{ Image("CSV", "csv.svg") }}CSV</a></li>
  <li><a href="destinations/json">{{ Image("JSON", "json.svg") }}JSON</a></li>
  <li><a href="destinations/parquet">{{ Image("Parquet", "parquet.svg") }}Parquet</a></li>
  <li><a href="destinations/s3">{{ Image("S3", "s3.svg") }}S3</a></li>
  <li><a href="destinations/http-post">{{ Image("HTTP POST", "http-post.svg") }}HTTP POST</a></li>
  <li><a href="destinations/sftp">{{ Image("SFTP", "sftp.svg") }}SFTP</a></li>
  <li><a href="destinations/filesystem">{{ Image("Filesystem", "filesystem.svg") }}Filesystem</a></li>
  <li><a href="destinations/clickhouse">{{ Image("ClickHouse", "clickhouse.svg") }}ClickHouse</a></li>
  <li><a href="destinations/postgresql">{{ Image("PostgreSQL", "postgresql.svg") }}PostgreSQL</a></li>
  <li><a href="destinations/snowflake">{{ Image("Snowflake", "snowflake.svg") }}Snowflake</a></li>
  <li><a href="destinations/mysql">{{ Image("MySQL", "mysql.svg") }}MySQL</a></li>
</ul>

### Need more destinations?

If you need additional destinations, [contact us](mailto:hello@meergo.com). You can also [create a custom connector](create-new-connector).
