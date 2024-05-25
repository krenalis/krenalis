
## App connectors

| Method           | Settings       | SetSettings    | Resource                | HTTPClient              | Region                  |
|------------------|----------------|----------------|-------------------------|-------------------------|-------------------------|
| Create           | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventRequest     | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| EventTypes       | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ReceiveWebhook   | ✓ (connection) | ✓ (connection) | ✓ (connection,resource) | ✓ (connection,resource) | ✓ (connection,resource) |
| Records          | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| Resource         | -              | -              | -                       | ✓                       | ✓                       |
| Schema           | ✓              | ✓              | ✓                       | ✓                       | ✓                       |
| ServeUI          | -/✓            | -/✓            | ✓                       | ✓                       | -/✓                     |
| Update           | ✓              | ✓              | ✓                       | ✓                       | ✓                       |


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

