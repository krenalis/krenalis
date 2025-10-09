{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Events{% end %}
{% Article %}

# Events

Meergo captures behavioral events from every customer touchpoint—websites, mobile apps, and more. It unifies event tracking by replacing complex configurations with a single customizable integration that sends data directly to your data warehouse.

#### Why it helps

* Quick to implement and adapt to your schema
* Consistent tracking across platforms
* Warehouse-first design for analytics and activation
* Real-time delivery: events are written to the warehouse in seconds
* Dual routing: the same events can be fanned out to downstream apps (analytics, marketing, product, and ops)
* Interoperable: fully compatible with Segment and RudderStack, so you can ingest existing events, mirror schemas, and run side-by-side without re-instrumentation

{{ Image("Event ingestion", "/docs/events-ingestion.png") }}

#### How it works

1. **Collect** events from SDKs and server sources with a flexible, typed schema.
2. **Validate** incoming data to ensure it meets schema requirements and add relevant context.
3. **Stream to warehouse** in real time for immediate querying and modeling.
4. **Send to apps** simultaneously via connection destinations.

#### Outcomes

Clean, centralized event data available instantly for BI, AI, and operational workflows—plus synchronized delivery to the apps your teams already use.

### Event ingestion

Meergo offers installable SDKs for your app or site to start collecting events. Pick the SDK that matches your stack. Available options include:

- [JavaScript SDK (Browser)](sources/javascript-sdk)
- [Android SDK](sources/android-sdk)
- [Go SDK](sources/go)
- [Java SDK](sources/java)
- [.NET SDK](sources/dotnet)
- [Node.js SDK](sources/nodejs)
- [Python SDK](sources/python)

You can also use any Segment or RudderStack SDK, as they're compatible with Meergo. 

You can also ingest events server-to-server with the [Events API endpoint](/api/events#ingest-events). If you already use Segment or RudderStack, send the same payloads via webhooks through the [connector for Segment](sources/segment) or the [connector for RudderStack](sources/rudderstack), no re-instrumentation required (these connectors are not affiliated with, or endorsed by, Segment or RudderStack).

### Events Spec

The [Meergo Event Spec](events/specs) explains what to send when you track events with Meergo, including [Track](events/specs/track), [Identify](events/specs/identify), [Page](events/specs/page), [Screen](events/specs/screen), and [Group](events/specs/group)—their payloads, and examples.

### Session tracking

Meergo natively supports [session tracking](events/session-tracking). Events loaded into the warehouse include session identifiers, and destinations that handle sessions—currently **Mixpanel**—receive the same session context. If you use RudderStack SDKs or forward events via RudderStack webhooks, session data is preserved.

### Load events and users into your warehouse

Events and user profiles are loaded into your [data warehouse](events/warehouse-destination) in real time. You can choose which sources to include and apply filters to focus on what matters most. Meergo currently supports Snowflake and PostgreSQL, with more platforms available on request — [contact us](mailto:hello@meergo.com) to learn more.

### Send events to apps

In addition to loading data into your warehouse, Meergo can forward events to external apps in real time. No additional SDKs are required. For each destination you can map and transform fields, apply filters, and control delivery so each app receives only the events it needs in the expected format.

* [Google Analytics](destinations/google-analytics)
* [Klaviyo](destinations/klaviyo)
* [Mixpanel](destinations/mixpanel)

If you need additional destinations, [contact us](mailto:hello@meergo.com). You can also [create a custom connector](create-new-connector).
