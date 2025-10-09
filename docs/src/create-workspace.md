{% extends "/layouts/doc.html" %}
{% import "external-image.md" %}
{% macro Title string %}Create a workspace{% end %}
{% Article %}

# Create a workspace

Meergo is a **warehouse-native** Customer Data Platform (CDP). This means that your customer data remains stored directly in **your own company's data warehouse** - not within the application itself.

Before you can start using Meergo, the very first step is to create a **workspace**. A workspace is the foundational environment where all of your customer data operations will take place.
You can choose a name for the workspace, so it can be easily recognized among other workspaces. It can be changed later.

Each workspace acts as a dedicated container that brings together three critical elements:

**Connection to your Data Warehouse** – This is where Meergo operates natively, ensuring that your customer data remains under your full control.

**Connections to Data Sources and Destinations** – Workspaces manage the flow of customer data in and out of your system, from CRMs and web apps to analytics tools and marketing platforms.

**Customer Model** – Within a workspace, you will define the structure of your customer data. This schema is used to create the **Golden Record**, the unified and trusted single profile of each customer, and is the foundation for **Identity Resolution**.

By creating a workspace, you establish the core environment where data is ingested, unified, and activated. This step is essential: it lays the groundwork for building accurate customer profiles, running identity resolution processes, and ultimately delivering the insights and activations your teams need.

In Meergo, each **workspace** connects to its own dedicated database (or schema) inside your company’s data warehouse. This ensures full isolation between workspaces while allowing Meergo to operate directly within your existing data infrastructure.

You cannot create a workspace until you fill all the required fields to connect to data warehouse.

Next: [Connect a warehouse](connect-warehouse)
