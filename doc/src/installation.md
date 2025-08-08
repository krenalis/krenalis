{% extends "/layouts/doc.html" %}
{% macro Title string %}Installation{% end %}
{% Article %}

# Installation

There are several ways to install Meergo or simply try out its features. Choose the one that suits you best:

* [**Using Docker**](./using-docker). This method is ideal for local development, testing, and prototyping. 
* [**Pre-compiled binaries**](./pre-compiled-binaries). A convenient method for quickly setting up Meergo without the need to compile from source.
* [**From source**](./from-source). Recommended if you wish to customize the executable or contribute to the project by building Meergo directly from the source.

## First login

When you start **Meergo** for the first time, you can access the admin using the default credentials:

- **Email:** `acme@open2b.com`
- **Password:** `foopass2`

After logging in, you’ll be prompted to create your first **workspace**.

Each workspace operates as an isolated environment with its own **data warehouse**, which stores user data, events, and is used for identity resolution and data unification.

> ⚠️ Once a data warehouse is linked to a workspace, it **cannot be changed** later.
