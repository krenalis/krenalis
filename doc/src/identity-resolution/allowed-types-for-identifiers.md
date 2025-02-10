{% extends "/layouts/doc.html" %}
{% macro Title string %}Allowed Types For Identifiers{% end %}
{% Article %}

# Allowed types for identifiers

Here is a list of allowed types for the [identifiers](../identity-resolution#identifiers) used in the [Identity Resolution](../identity-resolution):

* `int(n)`       
* `uint(n)`      
* `decimal(p,s)`, but only if scale `s` is 0
* `uuid`         
* `inet`         
* `text`         

> Note that meta properties cannot be used as identifiers. This should be
> documented in a consistent way.