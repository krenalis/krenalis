{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Activation{% end %}
{% Article %}

# Activation
## Activate your customer records and behavioral events in real time.

Activation sends **customer records** from your warehouse and live **behavioral events** from your sources to the SaaS tools you use to engage customers. This keeps your external tools accurate and consistent, enabling reliable analytics, AI-driven personalization, and automated actions.

### How activation works

* **Customer records:** Customer profiles stay synchronized between your warehouse and connected SaaS destinations. When a record changes (for example, an email address or subscription status update) Meergo automatically updates the existing destination profile or creates a new one as needed.

    <div style="border-left: 3px solid #5954ab; color: #221e67; padding-left: 1em; margin-bottom: 2em;">
    Examples:
  
    * Sync expired-trial users daily from Snowflake to HubSpot.
    * Trigger a HubSpot workflow that sends a “Renew now” campaign.
    </div>


* **Behavioral events:** Events from your sources are transformed and delivered instantly to your selected SaaS destinations. You can define which events to send, where to send them, and how their properties are mapped.

    <div style="border-left: 3px solid #5954ab; color: #221e67; padding-left: 1em; margin-bottom: 2em;">
    Examples:

    * Send the _"Page Viewed"_ event to Google Analytics for traffic analysis.
    * Stream the _"Purchase Completed"_ event to Mixpanel to update conversion funnels.
    * Forward the _"Email Subscribed"_ event to Klaviyo to trigger an onboarding flow.
    </div>

### Select where to activate users

Below are the destination SaaS applications currently supported by Meergo. Our library of SaaS connectors is continuously expanding. If you'd like to propose a new integration, we'd love to hear from you at [hello@meergo.com](mailto:hello@meergo.com).

<ul class="cards">
  <li>
    <a href="activation/google-analytics">
      <figure>{{ Image("Google Analytics", "google-analytics.svg") }}</figure>
      <div>Google Analytics</div>
      <p>Google Analytics is a web analytics platform that helps businesses track traffic and user behavior.</p>
    </a>
  </li>
  <li>
    <a href="activation/mixpanel">
      <figure>{{ Image("Mixpanel", "mixpanel.svg") }}</figure>
      <div>Mixpanel</div>
      <p>Mixpanel is an analytics platform that helps track user interactions, analyze funnels, and improve retention.</p>
    </a>
  </li>
  <li>
    <a href="activation/mailchimp">
      <figure>{{ Image("Mailchimp", "mailchimp.svg") }}</figure>
      <div>Mailchimp</div>
      <p>Mailchimp is an email marketing platform that helps businesses design and send marketing emails.</p>
    </a>
  </li>
  <li>
    <a href="activation/hubspot">
      <figure>{{ Image("Hubspot", "hubspot.svg") }}</figure>
      <div>HubSpot</div>
      <p>HubSpot offers tools for customer relationship management (CRM), marketing, and sales.</p>
    </a>
  </li>
  <li>
    <a href="activation/klaviyo">
      <figure>{{ Image("Klaviyo", "klaviyo.svg") }}</figure>
      <div>Klaviyo</div>
      <p>Klaviyo is a marketing platform that helps businesses create personalized, targeted email campaigns.</p>
    </a>
  </li>
  <li>
    <a href="activation/stripe">
      <figure>{{ Image("Stripe", "stripe.svg") }}</figure>
      <div>Stripe</div>
      <p>Stripe is a payment platform that lets businesses accept and manage online payments.</p>
    </a>
  </li>
</ul>