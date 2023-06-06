
## App connections

| Method         | Role | Settings       | SetSettings | Resource                | HTTPClient     | Region |
|----------------|------|----------------|-------------|-------------------------|----------------|--------|
| CreateUser     | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓      |
| EventTypes     | ✓    | ✓              | -           | ✓                       | ✓              | ✓      |
| Groups         | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓      |
| GroupSchema    | ✓    | ✓              | -           | ✓                       | ✓              | ✓      |
| ReceiveWebhook | -    | ✓ (connection) | -           | ✓ (connection,resource) | ✓ (connection) | -      |
| Resource       | -    | -              | -           | -                       | ✓              | ✓      |
| SendEvent      | ✓    | ✓              | -           | ✓                       | ✓              | ✓      |
| ServeUI        | ✓    | -/✓            | -/✓         | ✓                       | ✓              | -/✓    |
| SettingsUI     | ✓    | -              | -           | ✓                       | ✓              | -      |
| SetGroup       | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓      |
| UpdateUser     | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓      |
| Users          | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓      |
| UserSchema     | ✓    | ✓              | -           | ✓                       | ✓              | ✓      |


## Database connections

| Method      | Role | Settings | SetSettings |
|-------------|------|----------|-------------|
| Query       | ✓    | ✓        | ✓           |
| ServeUI     | ✓    | -/✓      | -/✓         |
| SettingsUI  | ✓    | -        | -           |


## File connections

| Method      | Role | Settings | SetSettings |
|-------------|------|----------|-------------|
| ContentType | ✓    | ✓        | ✓           |
| Read        | ✓    | ✓        | ✓           |
| ServeUI     | ✓    | -/✓      | -/✓         |
| SettingsUI  | ✓    | -        | -           |
| Write       | ✓    | ✓        | ✓           |


## Storage connections

| Method     | Role | Settings | SetSettings |
|------------|------|----------|-------------|
| Open       | ✓    | ✓        | -/✓         |
| ServeUI    | ✓    | -/✓      | -/✓         |
| SettingsUI | ✓    | -        | -           |
| Write      | ✓    | ✓        | ✓           |


## Stream connections

| Method     | Role | Settings | SetSettings |
|------------|------|----------|-------------|
| Close      | -    | -        | -           |
| Receive    | -    | ✓        | -           |
| Send       | -    | ✓        | -           |
| ServeUI    | ✓    | -/✓      | -/✓         |
| SettingsUI | ✓    | -        | -           |

