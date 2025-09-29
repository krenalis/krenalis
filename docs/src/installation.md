{% extends "/layouts/doc.html" %}
{% macro Title string %}Installation{% end %}
{% Article %}

# Installation

There are three ways to install Meergo or simply try out its features. Choose the one that suits you best:

* [**Using Docker**](./using-docker). This is the recommended way to quickly start experimenting with Meergo. In just a few steps, you can run a pre-configured local instance of Meergo — complete with its own local data warehouse — which you can later customize.
* [**Pre-compiled binaries**](./pre-compiled-binaries). If you want more control, this is a convenient way to set up Meergo by running the downloadable binary directly. It requires manual configuration of a PostgreSQL database and a data warehouse.
* [**From source**](./from-source). This is the most advanced installation method, offering maximum control and flexibility. Recommended if you want to customize the executable or contribute to the project by building Meergo directly from the source. 

> 💡 Don't know which one to choose? Start with the simplest: [launching Meergo with Docker](./using-docker). It's the fastest method and requires no configuration. You can try other options later if needed.
