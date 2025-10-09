{% extends "/layouts/doc.html" %}
{% import "../external-image.md" %}
{% macro Title string %}Destination connectors{% end %}
{% Article %}

# Destinations

<ul class="grid-list">
  <li><a href="destinations/mailchimp">{{ ExternalImage("mailchimp.svg") }}Mailchimp</a></li>
  <li><a href="destinations/hubspot">{{ ExternalImage("hubspot.svg") }}HubSpot</a></li>
  <li><a href="destinations/klaviyo">{{ ExternalImage("klaviyo.svg") }}Klaviyo</a></li>
  <li><a href="destinations/stripe">{{ ExternalImage("stripe.svg") }}Stripe</a></li>
  <li><a href="destinations/mixpanel">{{ ExternalImage("mixpanel.svg") }}Mixpanel</a></li>
  <li><a href="destinations/google-analytics">{{ ExternalImage("google-analytics.svg") }}Google Analytics</a></li>
  <li><a href="destinations/excel">{{ ExternalImage("excel.svg") }}Excel</a></li>
  <li><a href="destinations/csv">{{ ExternalImage("csv.svg") }}CSV</a></li>
  <li><a href="destinations/json">{{ ExternalImage("json.svg") }}JSON</a></li>
  <li><a href="destinations/parquet">{{ ExternalImage("parquet.svg") }}Parquet</a></li>
  <li><a href="destinations/s3">{{ ExternalImage("s3.svg") }}S3</a></li>
  <li><a href="destinations/http-post">{{ ExternalImage("http-post.svg") }}HTTP POST</a></li>
  <li><a href="destinations/sftp">{{ ExternalImage("sftp.svg") }}SFTP</a></li>
  <li><a href="destinations/filesystem">{{ ExternalImage("filesystem.svg") }}Filesystem</a></li>
  <li><a href="destinations/clickhouse">{{ ExternalImage("clickhouse.svg") }}ClickHouse</a></li>
  <li><a href="destinations/postgresql">{{ ExternalImage("postgresql.svg") }}PostgreSQL</a></li>
  <li><a href="destinations/snowflake">{{ ExternalImage("snowflake.svg") }}Snowflake</a></li>
  <li><a href="destinations/mysql">{{ ExternalImage("mysql.svg") }}MySQL</a></li>
</ul>

### Need more destinations?

If you need additional destinations, [contact us](mailto:hello@meergo.com). You can also [create a custom connector](create-new-connector).
