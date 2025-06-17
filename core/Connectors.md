
## App connectors

| Method            | Settings       | SetSettings    | OAuthAccount           | HTTPClient             |
|-------------------|----------------|----------------|------------------------|------------------------|
| EventTypeSchema   | ✓              | ✓              | ✓                      | ✓                      |
| EventTypes        | ✓              | ✓              | ✓                      | ✓                      |
| OAuthAccount      | -              | -              | -                      | ✓                      |
| PreviewSendEvents | ✓              | ✓              | ✓                      | ✓                      |
| ReceiveWebhook    | ✓ (connection) | ✓ (connection) | ✓ (account,connection) | ✓ (account,connection) |
| RecordSchema      | ✓              | ✓              | ✓                      | ✓                      |
| Records           | ✓              | ✓              | ✓                      | ✓                      |
| ServeUI           | -/✓            | -/✓            | ✓                      | ✓                      |
| SendEvents        | ✓              | ✓              | ✓                      | ✓                      |
| Upsert            | ✓              | ✓              | ✓                      | ✓                      |


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
