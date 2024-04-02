
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
| ValidateSettings | ✓    | -              | -              | ✓                       | ✓                       | ✓                       |


## Database connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | -/✓      | -/✓         |
| Columns          | ✓    | ✓        | ✓           |
| Query            | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Upsert           | ✓    | ✓        | ✓           |
| ValidateSettings | ✓    | -        | -           |


## File connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| ContentType      | ✓    | ✓        | ✓           |
| Read             | ✓    | ✓        | -           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Sheets           | ✓    | ✓        | -           |
| ValidateSettings | ✓    | -        | -           |
| Write            | ✓    | ✓        | ✓           |


## FileStorage connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| CompletePath     | ✓    | ✓        | ✓           |
| Reader           | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |
| Write            | ✓    | ✓        | ✓           |


## Stream connectors

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | ✓        | ✓           |
| Receive          | ✓    | ✓        | ✓           |
| Send             | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |

