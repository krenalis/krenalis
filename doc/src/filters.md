{% extends "/layouts/doc.html" %}
{% macro Title string %}Filters{% end %}
{% Article %}

# Filters

Filters allow you to filter users and events processed by an action. For a source action, you can filter the events received or the users read. For a destination action, you can filter the received events to be sent to the destination and the users to be exported.

Common use cases for filters include:

* **Reducing Data Volume**: Filtering out unnecessary data to focus on what's relevant, thus reducing the volume of data processed.
* **Targeted Analysis**: Narrowing down the data to specific subsets, such as particular users or events, for more precise analysis.
* **Improving Performance**: By filtering data early, you can reduce the load on systems and improve performance by processing only the necessary data.
* **Data Quality**: Removing or excluding erroneous or irrelevant data to ensure the quality and accuracy of the remaining dataset.
* **Personalization**: Filtering data to tailor content or actions based on user preferences or behaviors.
* **Regulatory Compliance**: Ensuring that only the required data is processed or retained, in line with privacy regulations or company policies.
* **Debugging and Testing**: Isolating specific events or users for debugging or testing purposes to diagnose issues or validate solutions.

## Operators

In a filter, select the boolean operator `any` if you want an event or user to match any of the conditions to be processed. Select `all` if you want an event or user to match all conditions.

Here are all the operators you can use in filters. The operators you can use for a property depend on the type of property and whether it can be null or not. Texts are compared in a case-sensitive manner.

| Operators                  |                               |
|----------------------------|-------------------------------|
| `is`                       | `is not`                      |
| `is less than`             | `is greater than`             |
| `is less than or equal to` | `is greater than or equal to` |
| `is between`               | `is not between`              |
| `contains`                 | `does not contain`            |
| `is one of`                | `is not one of`               |
| `starts with`              | `ends with`                   |
| `is before`                | `is after`                    |
| `is on or before`          | `is on or after`              |
| `is true`                  | `is false`                    |
| `is null`                  | `is not null`                 |
| `exists`                   | `does not exist`              |

## Best Practices

#### Quoted values

It is not necessary to quote values. However, if a value starts or ends with a space, `"` or `'`, you should quote it with `"` or `'`. Use a backslash (`\`) to escape `"` or `'` within a quoted value.

```
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ code                    ▼ │ │ starts with  ▼ │ │ "  'S'"                    │  ✔ yes
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ code                    ▼ │ │ starts with  ▼ │ │   'S'                      │  ✘ no
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
```

#### Using keys of a JSON property

To refer to a key within a JSON object, specify it right after the property name. For example, given the following object:   

```json
{
  "address": {
    "street": "1234 Sunset Blvd",
    "city": "Los Angeles",
    "state": "CA",
    "zip": "90026",
    "country": "USA"
  }
}
```

To refer to the state, use `address.state` as shown below:

```
┌───────────────────────────┐ ┌──────────────────┐ ┌──────────┐ ┌──────────┐
│ traits                  ▼ │ │ address.state    │ │ is     ▼ │ │ CA       │  ✔ yes
└───────────────────────────┘ └──────────────────┘ └──────────┘ └──────────┘
┌───────────────────────────┐ ┌──────────────────┐ ┌──────────┐ ┌──────────┐
│ traits                  ▼ │ │ "address city"   │ │ is     ▼ │ │ CA       │  ✘ no
└───────────────────────────┘ └──────────────────┘ └──────────┘ └──────────┘
```

Make sure you reference nested keys without quotes and using dot notation.

#### Check if a JSON property exists

To check if a JSON property exists, use the `exists` operator instead of `is null`:

```
┌───────────────────────────┐ ┌───────────────────────────┐ ┌────────────────┐
│ traits                  ▼ │ │ name                      │ │ exists       ▼ │  ✔ yes
└───────────────────────────┘ └───────────────────────────┘ └────────────────┘
┌───────────────────────────┐ ┌───────────────────────────┐ ┌────────────────┐
│ traits                  ▼ │ │ name                      │ │ is null      ▼ │  ✘ no
└───────────────────────────┘ └───────────────────────────┘ └────────────────┘
```

#### Check If true

Use the `is true` operator if you want a JSON property to be the boolean `true`:

```
┌───────────────────────────┐ ┌───────────────────────────┐ ┌────────────────┐
│ traits                  ▼ │ │ active                    │ │ is true      ▼ │  ✔ yes
└───────────────────────────┘ └───────────────────────────┘ └────────────────┘
┌───────────────────────────┐ ┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ traits                  ▼ │ │ active                    │ │ is           ▼ │ │ true                       │  ✘ no
└───────────────────────────┘ └───────────────────────────┘ └────────────────┘ └────────────────────────────┘
```

#### Dates and times

Write values representing date and time (`datetime` property type) using one of the following ISO8601 formats:

* `YYYY-MM-DDThh:mm::ss`
* `YYYY-MM-DDThh:mm::ss.nnnnnnnnn`
* `YYYY-MM-DDThh:mm::ss+hh:mm`
* `YYYY-MM-DDThh:mm::ssZ`

```
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ registration_time       ▼ │ │ is before    ▼ │ │ 2024-09-17T12:34:22.561    │  ✔ yes
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ registration_time       ▼ │ │ is before    ▼ │ │ 09-07-2024 12:34:22.561    │  ✘ no
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
```

Write values representing dates (`date` property type) using the format `YYYY-MM-DD`:

```
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ date_of_birth           ▼ │ │ is after     ▼ │ │ 2024-09-17                 │  ✔ yes
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ date_of_birth           ▼ │ │ is after     ▼ │ │ 09/17/2024                 │  ✘ no
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
```

Write values representing a time in a day (`time` property type) using the format `hh:mm:ss` or, for sub-second precision, `hh:mm:ss.nnnnnnnnn`:

```
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ event_start_time        ▼ │ │ is after     ▼ │ │ 10:30:00                   │  ✔ yes
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
┌───────────────────────────┐ ┌────────────────┐ ┌────────────────────────────┐
│ event_start_time        ▼ │ │ is before    ▼ │ │ 10h 30m                    │  ✘ no
└───────────────────────────┘ └────────────────┘ └────────────────────────────┘
```
