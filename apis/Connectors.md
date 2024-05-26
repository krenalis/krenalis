
## App connectors

| Method         | Settings       | SetSettings    | OAuthAccount           | HTTPClient             | Region                 |
|----------------|----------------|----------------|------------------------|------------------------|------------------------|
| Create         | ✓              | ✓              | ✓                      | ✓                      | ✓                      |
| EventRequest   | ✓              | ✓              | ✓                      | ✓                      | ✓                      |
| EventTypes     | ✓              | ✓              | ✓                      | ✓                      | ✓                      |
| OAuthAccount   | -              | -              | -                      | ✓                      | ✓                      |
| ReceiveWebhook | ✓ (connection) | ✓ (connection) | ✓ (account,connection) | ✓ (account,connection) | ✓ (account,connection) |
| Records        | ✓              | ✓              | ✓                      | ✓                      | ✓                      |
| Schema         | ✓              | ✓              | ✓                      | ✓                      | ✓                      |
| ServeUI        | -/✓            | -/✓            | ✓                      | ✓                      | -/✓                    |
| Update         | ✓              | ✓              | ✓                      | ✓                      | ✓                      |


## Database connectors

| Method           | Settings | SetSettings |
|------------------|----------|-------------|
| Close            | -/✓      | -/✓         |
| Columns          | ✓        | ✓           |
| Query            | ✓        | ✓           |
| ServeUI          | -/✓      | -/✓         |
| Upsert           | ✓        | ✓           |


## File connectors

| Method           | Settings | SetSettings |
|------------------|----------|-------------|
| ContentType      | ✓        | ✓           |
| Read             | ✓        | -           |
| ServeUI          | -/✓      | -/✓         |
| Sheets           | ✓        | -           |
| Write            | ✓        | ✓           |


## FileStorage connectors

| Method           | Settings | SetSettings |
|------------------|----------|-------------|
| CompletePath     | ✓        | ✓           |
| Reader           | ✓        | ✓           |
| ServeUI          | -/✓      | -/✓         |
| Write            | ✓        | ✓           |


## Stream connectors

| Method           | Settings | SetSettings |
|------------------|----------|-------------|
| Close            | ✓        | ✓           |
| Receive          | ✓        | ✓           |
| Send             | ✓        | ✓           |
| ServeUI          | -/✓      | -/✓         |

