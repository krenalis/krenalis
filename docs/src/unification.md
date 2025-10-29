{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Unification{% end %}
{% Article %}

# Unification
## Use Meergo to unify any customer profiles.

Meergo unifies all customer profiles into a single, reliable view. It collects data from every source, cleans it, removes duplicates, and links identities to build one accurate and up-to-date customer record.

### How it works

1. **Collect data**\
   Import events, databases, files, or SaaS apps using a typed schema.

2. **Validate and enrich**\
   Ensure incoming data matches the schema and add contextual attributes.

3. **Harmonize**\
   Transform and normalize data to align with your customer model.

4. **Load into your data warehouse**\
   Send harmonized data in real time to your warehouse for analysis and AI models.

5. **Unify identities**\
   Link multiple profiles belonging to the same person to create a complete “single customer view.”

### Explore detailed guides

Select where you want to collect users from:

{{ render "unification/_includes/sources-cards.html" }}

### Process collected users

Learn how to manage and unify your collected user data for a complete and reliable customer view.

{{ render "unification/_includes/manage-users-cards.html" }}
