
## App connections

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


## Database connections

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | -/✓      | -/✓         |
| Columns          | ✓    | ✓        | ✓           |
| Query            | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| Upsert           | ✓    | ✓        | ✓           |
| ValidateSettings | ✓    | -        | -           |


## File connections

| Method           | Role | Settings  | SetSettings |
|------------------|------|-----------|-------------|
| ContentType      | ✓    | ✓         | ✓           |
| Read             | ✓    | ✓         | ✓           |
| ServeUI          | ✓    | -/✓       | -/✓         |
| Sheets           | ✓    | ✓         | ✓           |
| ValidateSettings | ✓    | -         | -           |
| Write            | ✓    | ✓         | ✓           |


## Storage connections

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| CompletePath     | ✓    | ✓        | ✓           |
| Reader           | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |
| Write            | ✓    | ✓        | ✓           |


## Stream connections

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | ✓    | ✓        | ✓           |
| Receive          | ✓    | ✓        | ✓           |
| Send             | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |

