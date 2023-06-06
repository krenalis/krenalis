
## App connections

| Method         | Role | Settings       | SetSettings | Resource                | HTTPClient     | PrivacyRegion |
|----------------|------|----------------|-------------|-------------------------|----------------|---------------|
| CreateUser     | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓             |
| EventTypes     | ✓    | ✓              | -           | ✓                       | ✓              | ✓             |
| Groups         | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓             |
| GroupSchema    | ✓    | ✓              | -           | ✓                       | ✓              | ✓             |
| ReceiveWebhook | -    | ✓ (connection) | -           | ✓ (connection,resource) | ✓ (connection) | -             |
| Resource       | -    | -              | -           | -                       | ✓              | ✓             |
| SendEvent      | ✓    | ✓              | -           | ✓                       | ✓              | ✓             |
| ServeUI        | ✓    | -/✓            | -/✓         | ✓                       | ✓              | -/✓           |
| SettingsUI     | ✓    | -              | -           | ✓                       | ✓              | -             |
| SetGroup       | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓             |
| UpdateUser     | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓             |
| Users          | ✓    | ✓              | ✓           | ✓                       | ✓              | ✓             |
| UserSchema     | ✓    | ✓              | -           | ✓                       | ✓              | ✓             |


## Database connections

| Method | Role | Settings | SetSettings | Resource | HTTPClient | PrivacyRegion |
|--------|------|----------|-------------|----------|------------|---------------|
| Query  | ✓    | ✓        | ✓           | -        | -          | -             |


## File connections

| Method      | Role | Settings | SetSettings | Resource | HTTPClient | PrivacyRegion |
|-------------|------|----------|-------------|----------|------------|---------------|
| ContentType | ✓    | ✓        | ✓           | -        | -          | -             |
| Read        | ✓    | ✓        | ✓           | -        | -          | -             |
| Write       | ✓    | ✓        | ✓           | -        | -          | -             |


## Storage connections

| Method | Role | Settings | SetSettings | Resource | HTTPClient | PrivacyRegion |
|--------|------|----------|-------------|----------|------------|---------------|
| Open   | ✓    | ✓        | -/✓         | -        | -          | -             |
| Write  | ✓    | ✓        | ✓           | -        | -          | -             |


## Stream connections

| Method  | Role | Settings | SetSettings | Resource | HTTPClient | PrivacyRegion |
|---------|------|----------|-------------|----------|------------|---------------|
| Close   | -    | -        | -           | -        | -          | -             |
| Receive | -    | ✓        | -           | -        | -          | -             |
| Send    | -    | ✓        | -           | -        | -          | -             |

