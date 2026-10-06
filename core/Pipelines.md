## Pipelines

This file describes the rules that pipelines follow for each supported combination of connection role, connector type, and target.

### Execution, filter and transformation

Batch pipelines can be scheduled; event-based pipelines run when events are received.

When a pipeline supports a transformation, it can be either a mapping or a function.

A constant transformation reads no input property, so its output depends only on constant values.

| Pipeline                            | Execution   | Filter                       | Transformation                                         | Constant transformation |
|-------------------------------------|-------------|------------------------------|--------------------------------------------------------|-------------------------|
| Source / Application / User         | Batch       | Yes                          | Required                                               | Not allowed             |
| Source / Database / User            | Batch       | No (the query filters users) | Required                                               | Not allowed             |
| Source / FileStorage / User         | Batch       | Yes                          | Required                                               | Not allowed             |
| Source / SDK / User                 | Event-based | Yes                          | Optional                                               | Allowed                 |
| Source / SDK / Event                | Event-based | Yes                          | Not supported                                          | —                       |
| Destination / Application / User    | Batch       | Yes                          | Required                                               | Not allowed             |
| Destination / Application / Event   | Event-based | Yes                          | Depends on the event type (see below)                  | Allowed                 |
| Destination / Database / User       | Batch       | Yes                          | Required                                               | Not allowed             |
| Destination / FileStorage / User    | Batch       | Yes                          | Not supported                                          | —                       |

For Destination / Application / Event, the transformation is not supported if the event type has no schema, because there is no output schema. If the event type has a schema, the transformation is optional, unless the schema has properties required for creation: the output schema must include them, so the transformation is required.

### Schemas

| Pipeline                          | Input schema in UI        | Output schema in UI                                  | Input schema properties in state                                                   | Output schema properties in state                     |
|-----------------------------------|---------------------------|------------------------------------------------------|------------------------------------------------------------------------------------|-------------------------------------------------------|
| Source / Application / User       | Application user (source) | User                                                 | Filter properties + transformed properties                                         | Transformed properties                                |
| Source / Database / User          | Query                     | User                                                 | Transformed properties + user ID column + update time column                       | Transformed properties                                |
| Source / FileStorage / User       | File                      | User                                                 | Filter properties + transformed properties + user ID column + update time column  | Transformed properties                                |
| Source / SDK / User               | Event                     | User                                                 | Event schema properties                                                            | Transformed properties                                |
| Source / SDK / Event              | (none)                    | (none)                                               | Event schema properties                                                            | (none)                                                |
| Destination / Application / User  | User                      | Application user (destination)                       | Filter properties + transformed properties + internal matching property            | Transformed properties + external matching property   |
| Destination / Application / Event | Event                     | Event type (none, if the event type has no schema)   | Event schema properties                                                            | Transformed properties                                |
| Destination / Database / User     | User                      | Table                                                | Filter properties + transformed properties                                         | Transformed properties + table key                    |
| Destination / FileStorage / User  | User                      | (none)                                               | Profile schema properties                                                          | (none)                                                |

### Input schema property fields

"Schema of" tells what the input schema describes. When it describes events, the pipeline receives no input schema and uses the event schema, so these rules do not apply.

In every other input schema, `Prefilled` must be empty and `CreateRequired` and `UpdateRequired` must be `false`. `ReadOptional` and `Nullable` must have these values, where "any" means that both `true` and `false` are allowed:

| Pipeline                          | Schema of         | ReadOptional | Nullable |
|-----------------------------------|-------------------|--------------|----------|
| Source / Application / User       | Application users | any          | any      |
| Source / Database / User          | Query results     | `false`      | any      |
| Source / FileStorage / User       | File              | any          | any      |
| Source / SDK / User               | Events            | —            | —        |
| Source / SDK / Event              | Events            | —            | —        |
| Destination / Application / User  | Profiles          | `true`       | `false`  |
| Destination / Application / Event | Events            | —            | —        |
| Destination / Database / User     | Profiles          | `true`       | `false`  |
| Destination / FileStorage / User  | Profiles          | `true`       | `false`  |

### Output schema property fields

"Schema of" tells what the output schema describes. In every output schema, `Prefilled` must be empty. The other fields must have these values, where "any" means that both `true` and `false` are allowed:

| Pipeline                          | Schema of         | CreateRequired                                   | UpdateRequired | ReadOptional | Nullable                  |
|-----------------------------------|-------------------|--------------------------------------------------|----------------|--------------|---------------------------|
| Source / Application / User       | Profiles          | `false`                                          | `false`        | `true`       | `false`                   |
| Source / Database / User          | Profiles          | `false`                                          | `false`        | `true`       | `false`                   |
| Source / FileStorage / User       | Profiles          | `false`                                          | `false`        | `true`       | `false`                   |
| Source / SDK / User               | Profiles          | `false`                                          | `false`        | `true`       | `false`                   |
| Destination / Application / User  | Application users | any                                              | any            | see below    | any                       |
| Destination / Application / Event | Event type        | any                                              | `false`        | `false`      | any                       |
| Destination / Database / User     | Table             | `true` for the table key, `false` for the others | `false`        | `false`      | `false` for the table key |

For Destination / Application / User, `ReadOptional` must be `false`, except for the output matching property and the properties that contain it, where both `true` and `false` are allowed.

Source / SDK / Event and Destination / FileStorage / User have no output schema.

### Additional settings

| Pipeline                          | User ID column | Update time column                | Update time format                                     | Other required settings                                           | File settings            | Required consents        |
|-----------------------------------|----------------|-----------------------------------|--------------------------------------------------------|-------------------------------------------------------------------|--------------------------|--------------------------|
| Source / Application / User       | No             | No                                | No                                                     | —                                                                 | No                       | Profile consent location |
| Source / Database / User          | Required       | Optional, required if incremental | Required for a `string` or `json` column, no otherwise | Query                                                             | No                       | Profile consent location |
| Source / FileStorage / User       | Required       | Optional, required if incremental | Required for a `string` or `json` column, no otherwise | —                                                                 | see below                | Profile consent location |
| Source / SDK / User               | No             | No                                | No                                                     | —                                                                 | No                       | Profile consent location |
| Source / SDK / Event              | No             | No                                | No                                                     | —                                                                 | No                       | Event consent location   |
| Destination / Application / User  | No             | No                                | No                                                     | Export mode, matching properties, settings about duplicated users | No                       | Profile consent location |
| Destination / Application / Event | No             | No                                | No                                                     | —                                                                 | No                       | Event consent location   |
| Destination / Database / User     | No             | No                                | No                                                     | Table name and table key                                          | No                       | Profile consent location |
| Destination / FileStorage / User  | No             | No                                | No                                                     | Order by property path                                            | see below                | Profile consent location |

Source / FileStorage / User and Destination / FileStorage / User require a file format and a path, and can have a compression. They require a sheet if the file format has sheets, and format settings if the file format has settings for the role of the connection; otherwise, they cannot have them.

### Pipeline steps

Every pipeline has the `Receive` and `Finalize` steps. The other steps depend on the pipeline:

| Pipeline                          | InputValidation | Filter | EventConsent | ExportProfileConsent | Transformation | OutputValidation | ImportProfileConsent |
|-----------------------------------|:---------------:|:------:|:------------:|:--------------------:|:--------------:|:----------------:|:--------------------:|
| Source / Application / User       | ✓               | ✓      |              |                      | ✓              | ✓                | ✓                    |
| Source / Database / User          | ✓               |        |              |                      | ✓              | ✓                | ✓                    |
| Source / FileStorage / User       | ✓               | ✓      |              |                      | ✓              | ✓                | ✓                    |
| Source / SDK / User               |                 | ✓      |              |                      | ✓              | ✓                | ✓                    |
| Source / SDK / Event              |                 | ✓      | ✓            |                      |                |                  |                      |
| Destination / Application / User  | ✓               |        |              | ✓                    | ✓              | ✓                |                      |
| Destination / Application / Event |                 | ✓      | ✓            |                      | ✓              | ✓                |                      |
| Destination / Database / User     | ✓               |        |              | ✓                    | ✓              | ✓                |                      |
| Destination / FileStorage / User  | ✓               |        |              | ✓                    |                |                  |                      |

Pipelines that export users, and Source / Database / User, filter users while reading them, so they have no `Filter` step.
