
## App connectors

| Method         | Settings       | SetSettings    | OAuthAccount           | HTTPClient             |
|----------------|----------------|----------------|------------------------|------------------------|
| EventRequest   | ✓              | ✓              | ✓                      | ✓                      |
| EventTypes     | ✓              | ✓              | ✓                      | ✓                      |
| OAuthAccount   | -              | -              | -                      | ✓                      |
| ReceiveWebhook | ✓ (connection) | ✓ (connection) | ✓ (account,connection) | ✓ (account,connection) |
| Records        | ✓              | ✓              | ✓                      | ✓                      |
| Schema         | ✓              | ✓              | ✓                      | ✓                      |
| ServeUI        | -/✓            | -/✓            | ✓                      | ✓                      |
| Upsert         | ✓              | ✓              | ✓                      | ✓                      |


## Database connectors

| Method  | Settings | SetSettings |
|---------|----------|-------------|
| Close   | -/✓      | -/✓         |
| Columns | ✓        | ✓           |
| Merge   | ✓        | ✓           |
| Query   | ✓        | ✓           |
| ServeUI | -/✓      | -/✓         |


## File connectors

| Method      | Settings | SetSettings |
|-------------|----------|-------------|
| ContentType | ✓        | ✓           |
| Read        | ✓        | -           |
| ServeUI     | -/✓      | -/✓         |
| Sheets      | ✓        | -           |
| Write       | ✓        | ✓           |


## FileStorage connectors

| Method       | Settings | SetSettings |
|--------------|----------|-------------|
| AbsolutePath | ✓        | ✓           |
| Reader       | ✓        | ✓           |
| ServeUI      | -/✓      | -/✓         |
| Write        | ✓        | ✓           |


## Stream connectors

| Method  | Settings | SetSettings |
|---------|----------|-------------|
| Close   | ✓        | ✓           |
| Receive | ✓        | ✓           |
| Send    | ✓        | ✓           |
| ServeUI | -/✓      | -/✓         |
