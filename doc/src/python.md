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

The table below outlines the various Chichi types and their corresponding representations in the Python code for the transformation.

| Chichi&nbsp;Type | Python&nbsp;Type    | Example                                        |
|------------------|---------------------|------------------------------------------------|
| `Boolean`        | `bool`              | `True`                                         |
| `Int(n)`         | `int`               | `-2586`                                        |
| `Uint(n)`        | `int`               | `4063`                                         |
| `Float(n)`       | `float`             | `37.81`                                        |
| `Decimal(p,s)`   | `decimal.Decimal`   | `Decimal('5930174.18')`                        |
| `DateTime`       | `datetime.datetime` | `datetime(2024,1,15,8,51,49,822309)`           |
| `Date`           | `datetime.date`     | `date(2024,1,15)`                              |
| `Time`           | `datetime.time`     | `time(8,51,49,822309)`                         |
| `Year`           | `int`               | `2024`                                         |
| `UUID`           | `uuid.UUID`         | `UUID('f956622d-c421-4eca-8d20-efef87f9749c')` |
| `JSON`           | `str`               | `'{"score":10}'`                               |
| `Inet`           | `str`               | `'172.16.254.1'`                               |
| `Text`           | `str`               | `'123 Main Street'`                            |
| `Array`          | `list`              | `[472,182,604]`                                |
| `Object`         | `dict`              | `{'fistName':'Emily','lastName':'Johnson'}`    |
| `Map`            | `dict`              | `{'a':8073,'c':206}`                           |
