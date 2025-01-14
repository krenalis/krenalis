{% extends "/layouts/doc.html" %}
{% macro Title string %}Authentication{% end %}
{% Article %}

# Authentication

<section class="api-spec" id="api-section-authorization">
<div class="spec">
<div>

The Meergo API uses API keys for authenticating requests. You can manage these keys through the Meergo UI. The API keys utilize HTTP Bearer Authentication.

When accessing resources within a workspace, you can specify the workspace ID by passing the `Meergo-Workspace` header.

### Restricted keys

API keys can be optionally restricted to a specific workspace during their creation. A **restricted key** can only be used within the assigned workspace. In this case, you do not need to include the `Meergo-Workspace` header in your request.

</div>
<div>

  <div class="api-request-box">
    <div>Authenticated request</div>
    <div>
      <div>curl -X GET https://api.meergo.com/v0/connections \</div>
      <div>   -H "Authorization: Bearer &lt;YOUR_API_KEY&gt;"</div>
    </div>
  </div>

  <div class="api-request-box">
    <div>Authenticated request with explicit workspace ID</div>
    <div>
      <div>curl -X GET https://api.meergo.com/v0/connections \</div>
      <div>   -H "Authorization: Bearer &lt;YOUR_API_KEY&gt;"</div>
      <div>   -H "Meergo-Workspace: &lt;WORKSPACE_ID&gt;"</div>
    </div>
  </div>

</div>
</div>
</section>

<section class="api-spec" id="api-section-authorization">
<div class="spec">
<div>

## Write keys

For event ingestion, it is recommended to use a **write key**. A write key is a more limited form of an API key, granting access only for event ingestion and specific to a particular connection type (e.g., **Website**, **Mobile**, or **Server**). Write keys provide better security and focus for these operations.

Write keys are managed through the Meergo UI for each relevant source connection type.

Using a write key allows authentication exclusively for the following endpoints:

- [Ingest Batch Events](/api/events#ingest-batch-events)
- [Ingest Event](/api/events#ingest-event)

API Keys use HTTP Bearer Authentication.

</div>
<div>
  <div class="api-request-box">
  <div>Authenticated request with a write key</div>
        <div>
          <div>curl -X GET https://api.meergo.com/v0/api/events/batch \</div>
          <div>   -u &lt;YOUR_WRITE_KEY&gt;:</div>
        </div>
      </div>
  </div>
</div>
</section>
