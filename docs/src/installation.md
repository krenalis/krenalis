{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Installation{% end %}
{% Article %}

# Installation

There are two ways to install Meergo or simply try out its features. Choose the one that suits you best:

<ul class="cards" data-columns="1">
  <li>
    <a href="installation/using-docker">
      <figure>{{ Image("Docker", "/docs/images/docker.svg") }}</figure>
      <div>Using Docker</div>
      <p>This is the recommended way to quickly start experimenting with Meergo. In just a few steps, you can run a pre-configured local instance of Meergo — complete with its own local data warehouse — which you can later customize.</p>
    </a>
  </li>
  <li>
    <a href="installation/from-source">
      <figure>{{ Image("Source code", "/docs/images/source-code.svg") }}</figure>
      <div>From source</div>
      <p>This is the most advanced installation method, offering maximum control and flexibility. Recommended if you want to customize the executable or contribute to the project by building Meergo directly from the source.</p>
    </a>
  </li>
</ul>

_Don't sue which one to choose?_ Start with the simplest: [**Using Docker**](installation/using-docker). It's the fastest method and requires no configuration. You can explore the other method later if needed.
