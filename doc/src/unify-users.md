{% extends "/layouts/doc.html" %}
{% macro Title string %}Unify users{% end %}
{% Article %}

# Unify users

The unified customer profile feature within Meergo CDP is designed to centralize and enrich customer data from various sources, creating a single, comprehensive view of each customer. This process is vital for delivering personalized and cohesive experiences, improving segmentation, communication, and automation. By aggregating and correlating data from multiple touchpoints and systems, this functionality ensures that organizations have an up-to-date, accessible, and holistic view of their customers.

## What is a unified customer profile?

A unified customer profile is a consolidated and consistent representation of a customer, combining all relevant data from different sources. It provides a comprehensive and real-time view of the customer that can including for example:

- **Demographic data:** name, email, phone number, address, etc.
- **Interactions:** purchase history, website interactions, email responses, customer service conversations, etc.
- **Behavioral data:** social media activity, product preferences, website searches, and clicks.
- **Transactional data:** purchases made, total spend, purchase frequency, payment methods, etc.
- **Behavioral segmentation:** categories of customers based on specific activities, such as frequent buyers, cart abandoners, or loyalty program members.

This functionality acts as the central hub for integrating and streamlining all customer-related data into a unified profile, enabling a more accurate and actionable understanding of each individual.

## How unified customer profiles work

The unified customer profile is created by collecting data from various sources, which can include:

- **Websites and mobile apps:** session data, interactions, and online purchases.
- **CRM systems:** contact details, sales history, and customer support interactions.
- **Social media and advertising:** data on interests, social activity, and campaign engagement.
- **Marketing and email platforms:** campaign data, open rates, clicks, preferences, and segmentation.

This feature centralizes and normalizes data from these sources, transforming them into a unified profile through processes like data stitching and deduplication, ensuring that each customer is accurately represented by a single profile.

### Steps in the unified profile process:

1. **Data acquisition:** Collect data from internal and external sources such as CRM systems, marketing platforms, social media, e-commerce, and mobile apps.
2. **Normalization:** Normalize and structure data uniformly to ensure consistency across various sources.
3. **Enrichment:** Enrich the unified profiles with additional data to enhance the customer's profile.
4. **Unification:** Combine data from different sources into a single customer profile, eliminating duplicates and providing a unified customer view.
5. **Distribution:** The unified profiles are then distributed and made accessible across various systems, such as marketing automation tools, CRM systems, analytics platforms, and customer support systems.

## How it works

- User data, retrieved through connections, is stored as identities in the data warehouse.
- Based on these identities, Meergo builds the identity graph.
- From the graph, it constructs customer profiles by unifying them based on identifiers.
- Stores these profiles in the customers table of the data warehouse.


## Benefits of unified customer profiles

### 1. **Complete customer view**
This functionality enables a comprehensive and up-to-date view of each customer by collecting data from all touchpoints and channels. Businesses can better understand their customers' needs, preferences, and behaviors.

### 2. **Advanced personalization**
A unified customer profile allows for personalized experiences by offering tailored content, recommendations, and messaging based on the customer's past actions and preferences.

### 3. **Improved segmentation**
With unified profiles, businesses can create more accurate and dynamic customer segments, based not only on demographics but also on real-time behaviors and past interactions. This improves the effectiveness of marketing campaigns and increases conversion rates.

### 4. **Optimized customer journey**
Centralizing customer data allows for a smoother and more consistent customer journey, ensuring that users receive coherent experiences across all channels. Teams in marketing, sales, and support can access real-time data to enhance customer interactions and streamline operations.

### 5. **Reduction of data duplication**
By consolidating customer data into a single profile, this functionality helps reduce data duplication. This ensures that the organization works with clean, accurate, and reliable customer data, improving operational efficiency.

## Conclusion

The unified customer profile feature is an essential component of the Meergo CDP, providing businesses with a single, centralized view of their customers. By aggregating and enriching data from multiple sources, it allows for highly personalized customer experiences, better segmentation, and more effective marketing strategies. With this unified approach, businesses can gain deeper insights into their customers and optimize every aspect of the customer journey.