## Pipelines

This file describes the rules that pipelines follow for each supported combination of connection role, connector type, and target.

### Execution, filter and transformation

Batch pipelines can be scheduled; event-based pipelines run when events are received.

When a pipeline supports a transformation, it can be either a mapping or a function.

| Pipeline                            | Execution   | Filter                       | Transformation                                         | Constant transformation |
|-------------------------------------|-------------|------------------------------|--------------------------------------------------------|-------------------------|
| Source / Application / User         | Batch       | Yes                          | Required                                               | No                      |
| Source / Database / User            | Batch       | No (the query filters users) | Required                                               | No                      |
| Source / FileStorage / User         | Batch       | Yes                          | Required                                               | No                      |
| Source / SDK / User                 | Event-based | Yes                          | Optional                                               | Yes                     |
| Source / SDK / Event                | Event-based | Yes                          | Not supported                                          | —                       |
| Destination / Application / User    | Batch       | Yes                          | Required                                               | No                      |
| Destination / Application / Event   | Event-based | Yes                          | Optional, only if the event type has a schema          | Yes                     |
| Destination / Database / User       | Batch       | Yes                          | Required                                               | No                      |
| Destination / FileStorage / User    | Batch       | Yes                          | Not supported                                          | —                       |

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

In every input schema, `Placeholder` must be empty and `CreateRequired` and `UpdateRequired` must be false.

`ReadOptional` and `Nullable` must have these values, where "any" means that both true and false are allowed:

| Pipeline                         | ReadOptional | Nullable |
|----------------------------------|--------------|----------|
| Source / Application / User      | any          | any      |
| Source / Database / User         | false        | any      |
| Source / FileStorage / User      | any          | any      |
| Destination / Application / User | true         | false    |
| Destination / Database / User    | true         | false    |
| Destination / FileStorage / User | true         | false    |

The input schema of Source / SDK / User, Source / SDK / Event and Destination / Application / Event is the event schema, so these rules do not apply to it.

### Output schema property fields

In every output schema, `Placeholder` must be empty. The other fields must have these values, where "any" means that both true and false are allowed:

| Pipeline                          | CreateRequired                                 | UpdateRequired | ReadOptional | Nullable                  |
|-----------------------------------|------------------------------------------------|----------------|--------------|---------------------------|
| Source / Application / User       | false                                          | false          | true         | false                     |
| Source / Database / User          | false                                          | false          | true         | false                     |
| Source / FileStorage / User       | false                                          | false          | true         | false                     |
| Source / SDK / User               | false                                          | false          | true         | false                     |
| Destination / Application / User  | any                                            | any            | false        | any                       |
| Destination / Application / Event | any                                            | false          | false        | any                       |
| Destination / Database / User     | true for the table key, false for the others   | false          | false        | false for the table key   |

Source / SDK / Event and Destination / FileStorage / User have no output schema.

### Additional settings

| Pipeline                          | User ID and update time columns (and format) | Other required settings                                            | File                         | Required consents         |
|-----------------------------------|----------------------------------------------|--------------------------------------------------------------------|------------------------------|---------------------------|
| Source / Application / User       | No                                           | —                                                                  | No                           | Profile consent location  |
| Source / Database / User          | Required                                     | Query                                                              | No                           | Profile consent location  |
| Source / FileStorage / User       | Required                                     | —                                                                  | Path, sheet and settings     | Profile consent location  |
| Source / SDK / User               | No                                           | —                                                                  | No                           | Profile consent location  |
| Source / SDK / Event              | No                                           | —                                                                  | No                           | Event consent location    |
| Destination / Application / User  | No                                           | Export mode, matching properties, settings about duplicated users | No                           | Profile consent location  |
| Destination / Application / Event | No                                           | —                                                                  | No                           | Event consent location    |
| Destination / Database / User     | No                                           | Table name and table key                                           | No                           | Profile consent location  |
| Destination / FileStorage / User  | No                                           | Order by property path                                             | Path, sheet and settings     | Profile consent location  |

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
