## Pipelines

Here is a list of supported combinations of pipeline's roles / connection types / targets.

For specific information about them, see the file [Pipelines.csv](Pipelines.csv).

```mermaid
%%{init: {'theme':'neutral'}}%%
graph LR

	conn([Connection]) --> source([Source])
	conn --> dest([Destination])
	
	source --> source_app(["App"]) --> source_app_users(["User"])
	source --> source_db(["Database"]) --> source_db_users(["User"])
	source --> source_fileStorage(["FileStorage"]) --> source_fileStorage_users(["User"])
	source --> source_events_based([SDK])
	
	source_events_based --> source_events_based_users(["User"])
	source_events_based --> source_events_based_events(["Event"])
	
	dest --> dest_app([App])
	dest --> dest_db(["Database"]) --> dest_db_users(["User"])
	dest --> dest_fileStorage(["FileStorage"]) --> dest_fileStorage_users(["User"])
	
	dest_app --> dest_app_users(["User"])
	dest_app --> dest_app_events([Event])
	

```
