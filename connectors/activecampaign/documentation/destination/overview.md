ActiveCampaign is a customer experience automation platform for managing contacts, marketing campaigns, and sales workflows.

## What can you do with this?

Using this connector, you can create and update ActiveCampaign Contacts from Krenalis profiles. It maps the email address, first name, last name, and phone number.

New profiles are synchronized by email, so an existing Contact with the same email address is updated. Profiles that already have an ActiveCampaign Contact ID are updated by that ID.

ActiveCampaign exposes its incremental Contact filter at date granularity. When Krenalis needs an exact incremental comparison, the connector scans Contacts in ID order and applies the requested timestamp locally.

## What does it require?

* An ActiveCampaign account.
* The account-specific API URL shown under **Settings > Developer**.
* The API token shown under **Settings > Developer**.

> ActiveCampaign is a trademark of ActiveCampaign, LLC.
> This connector is not affiliated with or endorsed by ActiveCampaign, LLC.
