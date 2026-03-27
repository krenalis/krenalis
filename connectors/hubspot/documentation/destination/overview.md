
HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

## What can you do with this?

Using this connector you can create and update unified Krenalis users from your data warehouse in HubSpot as contacts.

## What does it require?

* An account with [HubSpot](https://www.hubspot.com/).
* A HubSpot private app access token with the `crm.objects.contacts.read`, `crm.objects.contacts.write`, and `crm.schemas.contacts.read` scopes.

## How to generate the access token

1. In HubSpot, click the settings icon (⚙) in the top navigation bar.
2. Go to **Integrations → Legacy Apps** in the left sidebar.
3. Click **Create legacy app app**.
4. When HubSpots asks you *What kind of legacy app do you want to create?*, click **Private**
5. In the **Basic Info** tab, give the app a name (e.g. "Krenalis").
6. Go to the **Scopes** tab and add: `crm.objects.contacts.read`, `crm.objects.contacts.write` and `crm.schemas.contacts.read`.
7. Click **Create app** in the top right and confirm.
8. In the **Auth** tab of the app details page, in the section **Access token**, click **Show token**, then **Copy**.

> HubSpot is a trademark of HubSpot, Inc.
> This connector is not affiliated with or endorsed by HubSpot, Inc.
