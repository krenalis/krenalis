{% extends "/layouts/doc.html" %}
{% macro Title string %}Transformers{% end %}
{% Article %}

# Transformers

Transformers allow you to transform values that conform to one scheme (the input one to the transformation) into values that conform to another scheme (the output one).

They therefore allow, for example, to adapt the data read from a source to the data warehouse, or to adapt the data read from the data warehouse before exporting it to a destination.

## Types of transformers

There are two types of transformers:

* the [mapping](./mapping), that is simple and immediate
* transformation functions, that allow you to express greater complexity and control

For transformation functions, you can choose between:

* the [JavaScript](./javascript) transformation functions
* the [Python](./python) transformation functions

> Please note that, in certain contexts of use, the transformation functions may not be available.
