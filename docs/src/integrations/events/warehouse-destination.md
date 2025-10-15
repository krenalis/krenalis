{% extends "/layouts/doc.html" %}
{% macro Title string %}Warehouse destination{% end %}
{% Article %}

# Warehouse destination

Meergo creates and manages a single central table called `events` in your data warehouse, which stores all event types from all sources. A separate `events` table is created for each workspace, since every workspace has its own dedicated data warehouse database.

## Table schema

<!-- tabs: table schema -->

### PostgreSQL

| Column                               | Type                    | Description                                                      |
|--------------------------------------|-------------------------|------------------------------------------------------------------|
| `id`                                 | `uuid NOT NULL`         | Event primary key. Globally unique.                              |
| `user`                               | `uuid`                  | Internal user UUID if available.                                 |
| `connection_id`                      | `integer NOT NULL`      | Source connection identifier.                                    |
| `anonymous_id`                       | `varchar NOT NULL`      | Anonymous identifier.                                            |
| `channel`                            | `varchar`               | Ingestion channel (web, mobile, server).                         |
| `category`                           | `varchar`               | Event category or grouping.                                      |
| `context_app_name`                   | `varchar`               | App name.                                                        |
| `context_app_version`                | `varchar`               | App version string.                                              |
| `context_app_build`                  | `varchar`               | App build number.                                                |
| `context_app_namespace`              | `varchar`               | App bundle or package identifier.                                |
| `context_browser_name`               | `event_browser_name`    | Browser name as a custom enum.                                   |
| `context_browser_other`              | `varchar`               | Browser name if not in enum.                                     |
| `context_browser_version`            | `varchar`               | Browser version.                                                 |
| `context_campaign_name`              | `varchar`               | Marketing campaign name.                                         |
| `context_campaign_source`            | `varchar`               | Campaign source (e.g., `google`).                                |
| `context_campaign_medium`            | `varchar`               | Campaign medium (e.g., `cpc`, `email`).                          |
| `context_campaign_term`              | `varchar`               | Campaign keyword or term.                                        |
| `context_campaign_content`           | `varchar`               | Campaign content or creative ID.                                 |
| `context_device_id`                  | `varchar`               | Device identifier.                                               |
| `context_device_advertising_id`      | `varchar`               | Advertising ID (`IDFA` or `AAID`).                               |
| `context_device_ad_tracking_enabled` | `boolean`               | Whether ad tracking is enabled.                                  |
| `context_device_manufacturer`        | `varchar`               | Device manufacturer.                                             |
| `context_device_model`               | `varchar`               | Device model.                                                    |
| `context_device_name`                | `varchar`               | Device name.                                                     |
| `context_device_type`                | `varchar`               | Device type (e.g., `phone`, `tablet`, `desktop`).                |
| `context_device_token`               | `varchar`               | Push notification device token.                                  |
| `context_ip`                         | `inet`                  | Client IP address.                                               |
| `context_library_name`               | `varchar`               | Tracking library name.                                           |
| `context_library_version`            | `varchar`               | Tracking library version.                                        |
| `context_locale`                     | `varchar(5)`            | Locale tag (e.g., `en-US`).                                      |
| `context_location_city`              | `varchar`               | City derived from location.                                      |
| `context_location_country`           | `varchar`               | Country derived from location.                                   |
| `context_location_latitude`          | `double precision`      | Latitude in decimal degrees.                                     |
| `context_location_longitude`         | `double precision`      | Longitude in decimal degrees.                                    |
| `context_location_speed`             | `double precision`      | Reported device speed.                                           |
| `context_network_bluetooth`          | `boolean`               | Bluetooth connectivity present.                                  |
| `context_network_carrier`            | `varchar`               | Mobile carrier name.                                             |
| `context_network_cellular`           | `boolean`               | Using cellular network.                                          |
| `context_network_wifi`               | `boolean`               | Using Wi-Fi.                                                     |
| `context_os_name`                    | `event_os_name`         | OS name as a custom enum.                                        |
| `context_os_other`                   | `varchar`               | OS name if not in enum.                                          |
| `context_os_version`                 | `varchar`               | OS version string.                                               |
| `context_page_path`                  | `varchar`               | URL path of the page.                                            |
| `context_page_referrer`              | `varchar`               | Referrer URL.                                                    |
| `context_page_search`                | `varchar`               | Query string of the page URL.                                    |
| `context_page_title`                 | `varchar`               | Document or page title.                                          |
| `context_page_url`                   | `varchar`               | Full page URL.                                                   |
| `context_referrer_id`                | `varchar`               | Referrer or campaign identifier.                                 |
| `context_referrer_type`              | `varchar`               | Referrer type (`search`, `social`, etc.).                        |
| `context_screen_width`               | `smallint`              | Screen width in pixels.                                          |
| `context_screen_height`              | `smallint`              | Screen height in pixels.                                         |
| `context_screen_density`             | `numeric(3,2)`          | Screen pixel ratio (e.g., `2.00`).                               |
| `context_session_id`                 | `bigint`                | Session identifier.                                              |
| `context_session_start`              | `boolean`               | `true` if this event starts a session.                           |
| `context_timezone`                   | `varchar`               | IANA or offset time zone string.                                 |
| `context_user_agent`                 | `varchar`               | Raw user agent string.                                           |
| `event`                              | `varchar`               | Event name (often for `track`).                                  |
| `group_id`                           | `varchar`               | Group identifier for group events.                               |
| `message_id`                         | `varchar NOT NULL`      | Message identifier provided by the source.                       |
| `name`                               | `varchar`               | Screen or page name, or alias.                                   |
| `properties`                         | `jsonb`                 | Event properties payload.                                        |
| `received_at`                        | `timestamp(3) NOT NULL` | When the warehouse received the event.                           |
| `sent_at`                            | `timestamp(3) NOT NULL` | When the client sent the event.                                  |
| `timestamp`                          | `timestamp(3) NOT NULL` | When the event occurred at the source.                           |
| `traits`                             | `jsonb NOT NULL`        | User or group traits payload.                                    |
| `type`                               | `event_type NOT NULL`   | Event type as a custom enum (`track`, `page`, `identify`, etc.). |
| `previous_id`                        | `varchar`               | Previous user identifier for aliasing.                           |
| `user_id`                            | `varchar`               | User ID supplied by the source along with the event              |

### Snowflake

| Column                               | Type                     | Description                                           |
|--------------------------------------|--------------------------|-------------------------------------------------------|
| `ID`                                 | `VARCHAR(36) NOT NULL`   | Event primary key. Globally unique.                   |
| `USER`                               | `VARCHAR(36)`            | Internal user identifier if available.                |
| `CONNECTION_ID`                      | `INT NOT NULL`           | Source connection identifier.                         |
| `ANONYMOUS_ID`                       | `VARCHAR NOT NULL`       | Anonymous identifier.                                 |
| `CHANNEL`                            | `VARCHAR`                | Ingestion channel (web, mobile, server).              |
| `CATEGORY`                           | `VARCHAR`                | Event category or grouping.                           |
| `CONTEXT_APP_NAME`                   | `VARCHAR`                | App name.                                             |
| `CONTEXT_APP_VERSION`                | `VARCHAR`                | App version string.                                   |
| `CONTEXT_APP_BUILD`                  | `VARCHAR`                | App build number.                                     |
| `CONTEXT_APP_NAMESPACE`              | `VARCHAR`                | App bundle or package identifier.                     |
| `CONTEXT_BROWSER_NAME`               | `VARCHAR`                | Browser name.                                         |
| `CONTEXT_BROWSER_OTHER`              | `VARCHAR`                | Browser name if not standardized.                     |
| `CONTEXT_BROWSER_VERSION`            | `VARCHAR`                | Browser version.                                      |
| `CONTEXT_CAMPAIGN_NAME`              | `VARCHAR`                | Marketing campaign name.                              |
| `CONTEXT_CAMPAIGN_SOURCE`            | `VARCHAR`                | Campaign source (e.g., `GOOGLE`).                     |
| `CONTEXT_CAMPAIGN_MEDIUM`            | `VARCHAR`                | Campaign medium (e.g., `CPC`, `EMAIL`).               |
| `CONTEXT_CAMPAIGN_TERM`              | `VARCHAR`                | Campaign keyword or term.                             |
| `CONTEXT_CAMPAIGN_CONTENT`           | `VARCHAR`                | Campaign content or creative ID.                      |
| `CONTEXT_DEVICE_ID`                  | `VARCHAR`                | Device identifier.                                    |
| `CONTEXT_DEVICE_ADVERTISING_ID`      | `VARCHAR`                | Advertising ID (`IDFA` or `AAID`).                    |
| `CONTEXT_DEVICE_AD_TRACKING_ENABLED` | `BOOLEAN`                | Whether ad tracking is enabled.                       |
| `CONTEXT_DEVICE_MANUFACTURER`        | `VARCHAR`                | Device manufacturer.                                  |
| `CONTEXT_DEVICE_MODEL`               | `VARCHAR`                | Device model.                                         |
| `CONTEXT_DEVICE_NAME`                | `VARCHAR`                | Device name.                                          |
| `CONTEXT_DEVICE_TYPE`                | `VARCHAR`                | Device type (e.g., `phone`, `tablet`, `desktop`).     |
| `CONTEXT_DEVICE_TOKEN`               | `VARCHAR`                | Push notification device token.                       |
| `CONTEXT_IP`                         | `VARCHAR`                | Client IP address.                                    |
| `CONTEXT_LIBRARY_NAME`               | `VARCHAR`                | Tracking library name.                                |
| `CONTEXT_LIBRARY_VERSION`            | `VARCHAR`                | Tracking library version.                             |
| `CONTEXT_LOCALE`                     | `VARCHAR(5)`             | Locale tag (e.g., `EN-US`).                           |
| `CONTEXT_LOCATION_CITY`              | `VARCHAR`                | City derived from location.                           |
| `CONTEXT_LOCATION_COUNTRY`           | `VARCHAR`                | Country derived from location.                        |
| `CONTEXT_LOCATION_LATITUDE`          | `FLOAT`                  | Latitude in decimal degrees.                          |
| `CONTEXT_LOCATION_LONGITUDE`         | `FLOAT`                  | Longitude in decimal degrees.                         |
| `CONTEXT_LOCATION_SPEED`             | `FLOAT`                  | Reported device speed.                                |
| `CONTEXT_NETWORK_BLUETOOTH`          | `BOOLEAN`                | Bluetooth connectivity present.                       |
| `CONTEXT_NETWORK_CARRIER`            | `VARCHAR`                | Mobile carrier name.                                  |
| `CONTEXT_NETWORK_CELLULAR`           | `BOOLEAN`                | Using cellular network.                               |
| `CONTEXT_NETWORK_WIFI`               | `BOOLEAN`                | Using Wi-Fi.                                          |
| `CONTEXT_OS_NAME`                    | `VARCHAR`                | OS name.                                              |
| `CONTEXT_OS_OTHER`                   | `VARCHAR`                | OS name if not standardized.                          |
| `CONTEXT_OS_VERSION`                 | `VARCHAR`                | OS version string.                                    |
| `CONTEXT_PAGE_PATH`                  | `VARCHAR`                | URL path of the page.                                 |
| `CONTEXT_PAGE_REFERRER`              | `VARCHAR`                | Referrer URL.                                         |
| `CONTEXT_PAGE_SEARCH`                | `VARCHAR`                | Query string of the page URL.                         |
| `CONTEXT_PAGE_TITLE`                 | `VARCHAR`                | Document or page title.                               |
| `CONTEXT_PAGE_URL`                   | `VARCHAR`                | Full page URL.                                        |
| `CONTEXT_REFERRER_ID`                | `VARCHAR`                | Referrer or campaign identifier.                      |
| `CONTEXT_REFERRER_TYPE`              | `VARCHAR`                | Referrer type (`search`, `social`, etc.).             |
| `CONTEXT_SCREEN_WIDTH`               | `SMALLINT`               | Screen width in pixels.                               |
| `CONTEXT_SCREEN_HEIGHT`              | `SMALLINT`               | Screen height in pixels.                              |
| `CONTEXT_SCREEN_DENSITY`             | `NUMBER(3,2)`            | Screen pixel ratio (e.g., `2.00`).                    |
| `CONTEXT_SESSION_ID`                 | `BIGINT`                 | Session identifier.                                   |
| `CONTEXT_SESSION_START`              | `BOOLEAN`                | `TRUE` if this event starts a session.                |
| `CONTEXT_TIMEZONE`                   | `VARCHAR`                | IANA or offset time zone string.                      |
| `CONTEXT_USER_AGENT`                 | `VARCHAR`                | Raw user agent string.                                |
| `EVENT`                              | `VARCHAR`                | Event name (often for `TRACK`).                       |
| `GROUP_ID`                           | `VARCHAR`                | Group identifier for group events.                    |
| `MESSAGE_ID`                         | `VARCHAR NOT NULL`       | Message identifier provided by the source.            |
| `NAME`                               | `VARCHAR`                | Screen or page name, or alias.                        |
| `PROPERTIES`                         | `VARIANT`                | Event properties payload.                             |
| `RECEIVED_AT`                        | `TIMESTAMP_NTZ NOT NULL` | When the warehouse received the event.                |
| `SENT_AT`                            | `TIMESTAMP_NTZ NOT NULL` | When the client sent the event.                       |
| `TIMESTAMP`                          | `TIMESTAMP_NTZ NOT NULL` | When the event occurred at the source.                |
| `TRAITS`                             | `VARIANT NOT NULL`       | User or group traits payload.                         |
| `TYPE`                               | `VARCHAR NOT NULL`       | Event type label (`track`, `page`, `identify`, etc.). |
| `PREVIOUS_ID`                        | `VARCHAR`                | Previous user identifier for aliasing.                |
| `USER_ID`                            | `VARCHAR`                | User ID supplied by the source along with the event.  |

<!-- /tabs -->
