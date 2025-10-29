{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Events{% end %}
{% Article %}

# Events
## Track, validate, and route behavioral data.

Meergo lets you collect, validate, and route behavioral events from all your digital touchpoints — including websites, mobile apps, server-side sources, and connected devices. It unifies event tracking across systems and ensures every event is structured, reliable, and ready for analytics or activation.

### How events work

1. **Collect events**\
   Events from your apps and websites with a Segment-compatible typed schema.

2. **Validate and enrich**\
   Validate incoming events to ensure they meets schema requirements and add relevant context.

3. **Load into your data warehouse**\
   Events are streamed in real time to your connected data warehouse, ready for querying and AI modeling.

4. **Activate your data**\
   Send validated events to your SaaS tools to trigger automations or personalization.

### Explore detailed guides

Select where you want to collect events from:

<ul class="cards" data-columns="2">
  <li>
    <a href="/unification/collect-users/websites" title="Learn how to collect events from your websites and web apps">
      <figure>{{ Image("Website", "/docs/images/website.svg") }}</figure>
      <div>Websites</div>
      <p>Capture browser and web app activity with the JavaScript SDK.</p>
    </a>
  </li>
  <li>
    <a href="/unification/collect-users/your-apps" title="Learn how to collect events from apps you developed">
      <figure>{{ Image("App", "/docs/images/app.svg") }}</figure>
      <div>Applications</div>
      <p>Collect events from mobile, desktop, backend, or IoT devices using the appropriate SDK.</p>
    </a>
  </li>
</ul>

### Learn more about events

Explore the technical details behind event collection in Meergo.

<ul class="cards">
  <li>
    <a href="events/spec">
      <div>Learn events spec</div>
      <p>Understand event types, schema structure, and how Meergo enriches data.</p>
    </a>
  </li>
  <li>
    <a href="events/session-tracking">
      <div>Session tracking</div>
      <p>Learn how Meergo tracks user sessions to model behavior over time.</p>
    </a>
  </li>
  <li>
    <a href="events/session-tracking">
      <div>Enrichment</div>
      <p>Explore how additional context is added to each event automatically.</p>
    </a>
  </li>
</ul>

### Process collected events

Once collected, events can be loaded, transformed, and distributed for analytics or activation.

<ul class="cards">
  <li>
    <a href="events/load-into-warehouse">
      <div>Load into data warehouse</div>
      <p>Stream enriched events in real time for analytics and AI workloads.</p>
    </a>
  </li>
  <li>
    <a href="unification/map-and-harmonize">
      <div>Collect users</div>
      <p>Build unified user profiles directly from event traits.</p>
    </a>
  </li>
  <li>
    <a href="events/send-to-apps">
      <div>Send to SaaS applications</div>
      <p>Deliver real-time events to external apps through destination connectors.</p>
    </a>
  </li>
</ul>
