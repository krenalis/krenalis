
## App connections

| Method           | Role | Settings       | SetSettings    | Resource                | HTTPClient              | Region                  |
|------------------|------|----------------|----------------|-------------------------|-------------------------|-------------------------|
| CreateUser       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventTypes       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Groups           | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| GroupSchema      | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ReceiveWebhook   | ✓    | ✓ (connection) | ✓ (connection) | ✓ (connection,resource) | ✓ (connection,resource) | ✓ (connection,resource) |
| Resource         | -    | -              | -              | -                       | ✓                       | ✓                       |
| SendEvent        | ✓    | ✓              | -              | ✓                       | ✓                       | ✓                       |
| ServeUI          | ✓    | -/✓            | -/✓            | ✓                       | ✓                       | -/✓                     |
| SetGroup         | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| UpdateUser       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Users            | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| UserSchema       | ✓    | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ValidateSettings | ✓    | -              | -              | ✓                       | ✓                       | ✓                       |


## Database connections

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Query            | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
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
| Open             | ✓    | ✓        | ✓           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |
| Write            | ✓    | ✓        | ✓           |


## Stream connections

| Method           | Role | Settings | SetSettings |
|------------------|------|----------|-------------|
| Close            | -    | -        | -           |
| Receive          | -    | ✓        | -           |
| Send             | -    | ✓        | -           |
| ServeUI          | ✓    | -/✓      | -/✓         |
| ValidateSettings | ✓    | -        | -           |

