{% extends "/layouts/doc.html" %}
{% macro Title string %}Python{% end %}
{% Article %}

# Python

This is how a Python transform function looks like:

```python
def transform(user: dict) -> dict:
    return {}
```

So, for example, it could be written like this:

```python
def transform(user: dict) -> dict:
    return {
        "email": user["email"],
        "first_name": user.get("first_name", ""),
    }
```

## Types

The table below outlines the various Meergo types and their corresponding representations in the Python code for the transformation.

| Meergo&nbsp;Type | Python&nbsp;Type    | Example                                        |
|------------------|---------------------|------------------------------------------------|
| `text`           | `str`               | `'123 Main Street'`                            |
| `boolean`        | `bool`              | `True`                                         |
| `int(n)`         | `int`               | `-2586`                                        |
| `uint(n)`        | `int`               | `4063`                                         |
| `float(n)`       | `float`             | `37.81`                                        |
| `decimal(p,s)`   | `decimal.Decimal`   | `Decimal('5930174.18')`                        |
| `datetime`       | `datetime.datetime` | `datetime(2024,1,15,8,51,49,822309)`           |
| `date`           | `datetime.date`     | `date(2024,1,15)`                              |
| `time`           | `datetime.time`     | `time(8,51,49,822309)`                         |
| `year`           | `int`               | `2024`                                         |
| `uuid`           | `uuid.UUID`         | `UUID('f956622d-c421-4eca-8d20-efef87f9749c')` |
| `json`           | `str`               | `'{"score":10}'`                               |
| `inet`           | `str`               | `'172.16.254.1'`                               |
| `array`          | `list`              | `[472,182,604]`                                |
| `object`         | `dict`              | `{'fistName':'Emily','lastName':'Johnson'}`    |
| `map`            | `dict`              | `{'a':8073,'c':206}`                           |
