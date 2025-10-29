{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Collect users from SaaS applications{% end %}
{% Article %}

# Collect users from SaaS apps
## Learn how to collect users from SaaS applications.

Meergo makes it easy to collect customer data from SaaS applications. Each integration automatically aligns external data with your unified Customer Model schema and loads it into your data warehouse — ensuring a consistent and comprehensive customer view.

### Overview

Integrating SaaS sources helps you bring marketing, CRM, and transactional data into one place — without writing code. Once connected, Meergo keeps your data synchronized and ready for harmonization and activation workflows.

### Available source connectors

Below are the SaaS applications currently supported by Meergo. Our library of SaaS connectors is continuously expanding. If you'd like to propose a new integration, we'd love to hear from you at [hello@meergo.com](mailto:hello@meergo.com).

<ul class="cards">
  <li>
    <a href="saas-apps/mailchimp">
      <figure>{{ Image("Mailchimp", "/mailchimp.svg") }}</figure>
      <div>Mailchimp</div>
      <p>Mailchimp is an email marketing platform that helps businesses design and send marketing emails.</p>
    </a>
  </li>
  <li>
    <a href="saas-apps/hubspot">
      <figure>{{ Image("Hubspot", "/hubspot.svg") }}</figure>
      <div>HubSpot</div>
      <p>HubSpot offers tools for customer relationship management (CRM), marketing, and sales.</p>
    </a>
  </li>
  <li>
    <a href="saas-apps/klaviyo">
      <figure>{{ Image("Klaviyo", "/klaviyo.svg") }}</figure>
      <div>Klaviyo</div>
      <p>Klaviyo is a marketing platform that helps businesses create personalized, targeted email campaigns.</p>
    </a>
  </li>
  <li>
    <a href="saas-apps/stripe">
      <figure>{{ Image("Stripe", "/stripe.svg") }}</figure>
      <div>Stripe</div>
      <p>Stripe is a payment platform that lets businesses accept and manage online payments.</p>
    </a>
  </li>
  <li>
    <a href="saas-apps/segment">
      <figure>{{ Image("Segment", "/segment.svg") }}</figure>
      <div>Segment</div>
      <p>Segment is a platform that unifies data to improve analytics and personalization.</p>
    </a>
  </li>
  <li>
    <a href="saas-apps/rudderstack">
      <figure>{{ Image("RudderStack", "/rudderstack.svg") }}</figure>
      <div>RudderStack</div>
      <p>RudderStack is a platform that collects, unifies, and routes customer data across tools and systems.</p>
    </a>
  </li>
</ul>

Once connected, these integrations seamlessly feed unified user data into Meergo — ready for harmonization and activation workflows.
