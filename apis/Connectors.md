
## App connectors

| Method           | Role | Settings       | SetSettings    | Resource                | HTTPClient              | Region                  |
|------------------|------|----------------|----------------|-------------------------|-------------------------|-------------------------|
| CreateUser       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventRequest     | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventTypes       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Groups           | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| GroupSchema      | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ReceiveWebhook   | ✓    | ✓ (connection) | ✓ (connection) | ✓ (connection,resource) | ✓ (connection,resource) | ✓ (connection,resource) |
| Resource         | -    | -              | -              | -                       | ✓                       | ✓                       |
| ServeUI          | ✓    | -/✓            | -/✓            | ✓                       | ✓                       | -/✓                     |
| SetGroup         | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| UpdateUser       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Users            | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| UserSchema       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
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


## Storage connectors

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

