# Krenalis and connectors (context)

This is supporting context for the skill. It is not required for every connector implementation, but it helps when you need to explain concepts or decide capabilities.

## What is Krenalis?

**Krenalis** is a developer-first Customer Data Platform (CDP) that helps teams:

- **Collect** customer data and behavioral events from apps, websites, warehouses, files, and SaaS tools (via connectors and SDKs).
- **Validate** data strictly with schemas.
- **Unify** customer profiles via identity resolution.
- **Activate** data by exporting profiles and/or sending events to downstream tools in real time.

Krenalis is connector-driven: integrations are compiled Go modules that run inside the Krenalis server.

## What are connectors?

In Krenalis, a **connector** is a Go package that is compiled into the Krenalis executable and registered at init time. Connectors represent integrations for different "data problems", including:

- SDK connectors (clients send events)
- Webhook connectors (servers push events)
- Database connectors
- File / file-storage connectors
- Application connectors (SaaS APIs; source and/or destination)

Connectors are declared via a spec struct and then registered with the connectors registry (e.g. `connectors.RegisterApplication(...)`).

## What is an Application connector?

An **Application connector** integrates with an external SaaS product via HTTP APIs (e.g. HubSpot, Klaviyo, Mailchimp, Mixpanel, PostHog, Google Analytics).

An application connector can support any subset of the following:

- **Source**: read users/records from the app (import)
- **Destination**: upsert users/records to the app (export)
- **Destination (events)**: send behavioral events to the app (activation)

You declare which capabilities you support in `connectors.ApplicationSpec` and you must implement the matching interfaces.
