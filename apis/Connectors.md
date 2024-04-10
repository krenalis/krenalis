
## App connectors

| Method           | Role | Settings       | SetSettings    | Resource                | HTTPClient              | Region                  |
|------------------|------|----------------|----------------|-------------------------|-------------------------|-------------------------|
| Create           | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventRequest     | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventTypes       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ReceiveWebhook   | ✓    | ✓ (connection) | ✓ (connection) | ✓ (connection,resource) | ✓ (connection,resource) | ✓ (connection,resource) |
| Records          | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Resource         | -    | -              | -              | -                       | ✓                       | ✓                       |
| Schema           | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ServeUI          | ✓    | -/✓            | -/✓            | ✓                       | ✓                       | -/✓                     |
| Update           | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |


## Database connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | -/✓      | -/✓         |
| Columns          | ✓    | ✓        | ✓           |
| Query            | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Upsert           | ✓    | ✓        | ✓           |


## File connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| ContentType      | ✓    | ✓        | ✓           |
| Read             | ✓    | ✓        | -           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Sheets           | ✓    | ✓        | -           |
| Write            | ✓    | ✓        | ✓           |


## FileStorage connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| CompletePath     | ✓    | ✓        | ✓           |
| Reader           | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Write            | ✓    | ✓        | ✓           |


## Stream connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | ✓        | ✓           |
| Receive          | ✓    | ✓        | ✓           |
| Send             | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |

