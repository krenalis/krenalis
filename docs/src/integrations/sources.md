{% extends "/layouts/doc.html" %}
{% import "../external-image.md" %}
{% macro Title string %}Source connectors{% end %}
{% Article %}

# Sources

<ul class="grid-list">
  <li><a href="sources/mailchimp">{{ ExternalImage("mailchimp.svg") }} Mailchimp</a></li>
  <li><a href="sources/hubspot">{{ ExternalImage("hubspot.svg") }} HubSpot</a></li>
  <li><a href="sources/klaviyo">{{ ExternalImage("klaviyo.svg") }} Klaviyo</a></li>
  <li><a href="sources/stripe">{{ ExternalImage("stripe.svg") }} Stripe</a></li>
  <li><a href="sources/segment">{{ ExternalImage("segment.svg") }} Segment</a></li>
  <li><a href="sources/rudderstack">{{ ExternalImage("rudderstack.svg") }} RudderStack</a></li>
  <li><a href="sources/excel">{{ ExternalImage("excel.svg") }} Excel</a></li>
  <li><a href="sources/csv">{{ ExternalImage("csv.svg") }} CSV</a></li>
  <li><a href="sources/json">{{ ExternalImage("json.svg") }} JSON</a></li>
  <li><a href="sources/parquet">{{ ExternalImage("parquet.svg") }} Parquet</a></li>
  <li><a href="sources/s3">{{ ExternalImage("s3.svg") }} S3</a></li>
  <li><a href="sources/http-get">{{ ExternalImage("http-get.svg") }} HTTP GET</a></li>
  <li><a href="sources/sftp">{{ ExternalImage("sftp.svg") }} SFTP</a></li>
  <li><a href="sources/filesystem">{{ ExternalImage("filesystem.svg") }} Filesystem</a></li>
  <li><a href="sources/clickhouse">{{ ExternalImage("clickhouse.svg") }} ClickHouse</a></li>
  <li><a href="sources/postgresql">{{ ExternalImage("postgresql.svg") }} PostgreSQL</a></li>
  <li><a href="sources/snowflake">{{ ExternalImage("snowflake.svg") }} Snowflake</a></li>
  <li><a href="sources/mysql">{{ ExternalImage("mysql.svg") }} MySQL</a></li>
  <li><a href="sources/javascript-sdk">{{ ExternalImage("javascript.svg") }} JavaScript</a></li>
  <li><a href="sources/dotnet">{{ ExternalImage("dotnet.svg") }} .NET</a></li>
  <li><a href="sources/android-sdk">{{ ExternalImage("android.svg") }} Android</a></li>
  <li><a href="sources/go">{{ ExternalImage("go.svg") }} Go</a></li>
  <li><a href="sources/java">{{ ExternalImage("java.svg") }} Java</a></li>
  <li><a href="sources/nodejs">{{ ExternalImage("node.svg") }} Node.js</a></li>
  <li><a href="sources/python">{{ ExternalImage("python.svg") }} Python</a></li>
  <li><a href="sources/meergoapi">{{ ExternalImage("meergo-api.svg") }} Meergo API</a></li>
</ul>

## See also

* See also how to [create new connector](create-new-connector).
