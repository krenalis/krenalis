{% extends "/layouts/doc.html" %}
{% macro Title string %}Allowed Types For Identifiers{% end %}
{% Article %}

# Allowed Types For Identifiers

Here is a list of allowed types for the [identifiers](../identity-resolution.md#identifiers) used in the [Identity Resolution](../identity-resolution.md):

* `Int(n)`       
* `UInt(n)`      
* `Decimal(p,s)`, but only if scale `s` is 0
* `UUID`         
* `Inet`         
* `Text`         

> Note that meta properties cannot be used as identifiers. This should be
> documented in a consistent way.