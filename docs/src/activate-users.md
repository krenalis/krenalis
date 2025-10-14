{% extends "/layouts/doc.html" %}
{% macro Title string %}Unify users{% end %}
{% Article %}

# Activate users

Once customer identities have been unified and enriched, Meergo allows you to activate users by sending clean, standardized, and up-to-date profiles to external systems and marketing destinations.
Activation makes your unified data operational, powering segmentation, personalization, and automation across all your tools and channels.

Meergo supports two activation modes:

**Batch exports**: send unified user data to external systems (e.g., CRMs, marketing automation tools, e-commerce platforms, analytics systems).

**Real-time streaming**: deliver user updates and events instantly to APIs or destinations that support real-time activation.

Meergo ensures data quality and schema consistency during export, reducing integration errors and ensuring downstream systems receive harmonized, privacy-compliant data.

## Add a Destination Connection

A **destination connection** defines where unified customer data will be delivered.
Each destination corresponds to an external platform — for example, a CRM, advertising network, or email automation system.

You can create a destination connection from the **Meergo Admin Console**, on the **Destinations** page of your workspace, by clicking **Add a new destination ⊕** button.
For detailed setup instructions, refer to the specific --Destination Connector documentation--.
You can also create a destination connection programmatically using the Create Connection API endpoint.

## Add a Connection Action

Once a destination connection is created, you can add one or more actions to define what data will be exported and how it will be transformed before delivery.

Meergo supports two main types of destination actions:

**User Exports** – distribute unified user profiles and attributes (from Golden Records) to external systems.

**Event Exports** – send event data or behavioral signals to downstream platforms for real-time engagement and analytics.

## Map and Transform Outgoing Data

Just as data was harmonized at the source level during collection, Meergo allows you to map unified profile fields (from the Customer Model) to destination properties defined by the target platform.

Mappings can be configured:

**Visually**, by matching Customer Model fields to destination fields via the Admin Console.

**Programmatically**, using JavaScript or Python transformations to adjust, filter, or reformat data before export.

This mapping process ensures full compatibility between Meergo’s unified schema and the target system’s data structure, maintaining consistency and reducing manual intervention.

Meergo’s export pipeline guarantees:

- Schema alignment: data sent to each destination matches its expected format.

- Data filtering: only relevant users or segments are exported.

- Transformation flexibility: dynamic formatting, renaming, or enrichment before delivery.

## Data Privacy and Compliance

During activation, Meergo enforces privacy and suppression rules defined in your workspace, including:

- Consent enforcement: users who have withdrawn consent are automatically excluded.

- Data minimization: only necessary fields are shared with each destination.

- Compliance filtering: exports comply with GDPR, CCPA, and other privacy regulations.

## Example Workflow

1. Unified profiles are created and stored in your data warehouse.

2. You create a destination connection (e.g., Salesforce, HubSpot, Meta Ads, or a webhook endpoint).

3. You add a User Export Action, mapping Customer Model fields to destination properties.

4. Optionally, define a filter or segment to select which users to activate.

5. Click Run Export or schedule it for recurring sync.

6. Meergo delivers the formatted, compliant data to the external system.

## Benefits of User Activation

**Real-time personalization**: deliver targeted experiences across all channels.

**Consistent data distribution**: ensure all teams and platforms use the same up-to-date customer information.

**Operational efficiency**: automate recurring exports and integrations.

**Compliance-ready sharing**: enforce privacy and consent policies across destinations.

