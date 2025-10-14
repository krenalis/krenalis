{% extends "/layouts/doc.html" %}
{% macro Title string %}Unify users{% end %}
{% Article %}

# Unify users

Once data has been collected and harmonized at the connection action level, Meergo performs the critical processes that turn raw, fragmented records into trusted, unified customer profiles.
This happens through two main steps: **Data Cleaning & Suppression** and the **Identity Management Layer**, culminating in the creation of a **Golden Record**, the single source of truth for each customer.

The unified customer profile feature within Meergo is designed to centralize and enrich customer data from various sources, creating a single, comprehensive view of each customer. This process is vital for delivering personalized and cohesive experiences, improving segmentation, communication, and automation. By aggregating and correlating data from multiple touchpoints and systems, this functionality ensures that organizations have an up-to-date, accessible, and holistic view of their customers.

This process brings together all the customer data from different sources into a single, comprehensive profile, ensuring consistency, completeness, and reliability across your ecosystem.

## How to start the process

> You can start the identity resolution process from the **Meergo Admin Console**, under the **User Profiles** page.
If you have already run identity resolution before, you’ll see a list of all existing users, the **Golden Records**, that have been created and continuously kept up to date.
To start the process for the first time, or to run it again, simply click the **Resolve Identities ⊕** button.
A confirmation message will appear. When you’re ready, click **Run Identity Resolution ⊕** button to launch the process and wait for it to complete.
Once finished, Meergo will automatically update or create new unified profiles based on the latest data available.

## What is a unified customer profile?

A unified customer profile is a consolidated and consistent representation of an individual customer, built by combining all relevant data from various systems and touchpoints.
It provides a **real-time, 360°** view of the customer, which may include:

- **Demographic data:** name, email, phone number, address, etc.
- **Interactions:** purchase history, website interactions, email responses, customer service conversations, etc.
- **Behavioral data:** product preferences, searches, clicks, and engagement patterns.
- **Transactional data:** purchases made, total spend, purchase frequency, payment methods, etc.
- **Behavioral segmentation:** categories of customers based on specific activities, such as frequent buyers, cart abandoners, or loyalty program members.

Meergo’s unified profile acts as the **central hub** for integrating and aligning all customer-related data into a single entity, enabling accurate analytics, personalization, and activation.

## How unified customer profiles work

Unified profiles are created by **aggregating and correlating** user data from multiple systems, including:

- **Websites and mobile apps:** sessions, interactions, and transactions.
- **CRM systems:** contact details, sales history, and customer support interactions.
- **Social media and advertising:** data on interests, social activity, and campaign engagement.
- **Marketing and email platforms:** campaign data, open rates, clicks, preferences, and segmentation.
<!-- - **E-commerce systems**: purchase and product data -->

Once collected, Meergo applies a structured unification pipeline that transforms this data into a coherent, reliable view of each customer.


### The unification process:

Meergo’s unification process consists of two main phases: **Data Cleaning & Suppression** and **Identity Management**, both essential to ensuring data quality and compliance.

### 1. Data Cleaning & Suppression

At this stage, Meergo refines and filters data **before identity resolution** takes place:

- **Deduplication**: Detects and merges duplicate customer records across sources.

- **Obsolete data removal**: Suppresses records tied to revoked consent, outdated interactions, or prolonged inactivity.

<!-- - **Privacy and compliance enforcement**: Applies rules to anonymize or suppress data in line with GDPR, CCPA, and other regulations. -->

Performing suppression and cleaning at this point ensures that the CDP operates on refined, compliant datasets, reducing noise in analytics, improving campaign precision, and maintaining regulatory alignment.


### 2. Identity Management Layer

This is the heart of the unification process. Meergo links and merges records that belong to the same individual, creating a single, consistent identity.

1. **Data Storage:** Imported user data is stored in the **data warehouse** as **identities**, maintaining traceability to its original sources.
2. **Identity Graph Construction:** Meergo builds an identity graph, a network of identifiers (emails, IDs, cookies, device tokens, etc.) that link records belonging to the same individual across different systems.
3. **Identity Resolution**: Through advanced matching algorithms, Meergo detects which identifiers belong to the same person and unifies them into a single profile.
4. **Conflict Resolution**: Overlapping or conflicting records are automatically resolved to preserve accuracy and completeness.
5. **Enrichment:** The unified profile is then enriched with additional data and attributes from other sources, enhancing completeness and context.


The output of this phase is the **Golden Record**, a trustworthy, complete, and authoritative view of each customer, serving as the backbone for all downstream operations.

### 3. Distribution

The unified customer profiles are then stored in the customers table of your data warehouse and made available to connected systems such as CRM, marketing automation tools, analytics platforms, and activation environments.

> Note: It’s important to distinguish **Normalization** (performed at the connection action level) from **Unification** (performed at the identity level): Normalization ensures that incoming data uses consistent structures, formats, and types, preparing it for later correlation; Unification identifies and merges records that belong to the same user, across systems and time, forming a single customer view. Together, they guarantee data consistency and enable accurate downstream processes such as segmentation, personalization, and reporting.


## Benefits of unified customer profiles

### **Complete customer view**
This functionality enables a comprehensive and up-to-date view of each customer by collecting data from all touchpoints and channels. Businesses can better understand their customers' needs, preferences, and behaviors.

### **Advanced personalization**
A unified customer profile allows for personalized experiences by offering tailored content, recommendations, and messaging based on the customer's past actions and preferences.

### **Improved segmentation**
With unified profiles, businesses can create more accurate and dynamic customer segments. This improves the effectiveness of marketing campaigns and increases conversion rates.

### **Optimized customer journey**
Centralizing customer data allows for a smoother and more consistent customer journey, ensuring that users receive coherent experiences across all channels. Teams in marketing, sales, and support can access real-time data to enhance customer interactions and streamline operations.

### **Reduction of data duplication**
By eliminating reduntant records, this functionality ensures that all teams operate on clean, accurate, and reliable customer data, improving operational efficiency.

Next: [Activate Users](/activate-users)
