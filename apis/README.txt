This file contains the documentation for the REST APIs of Chichi.

GET    /api/connections                     List the connections
POST   /api/connections/{id}/status         Set the status of a connection
GET    /api/connections/{id}/schema         Get the schema of a connection
GET    /api/connections/{id}/mappings       List the mappings of a connection
PUT    /api/connections/{id}/mappings       Set the mappings of a connection
POST   /api/connections/{id}/import         Import from a connection
POST   /api/connections/{id}/reimport       Import from a connection
POST   /api/connections/{id}/export         Export to connection
PUT    /api/event-listeners/                Add a new event listener
DELETE /api/event-listeners/{id}            Remove an event listener
GET    /api/event-listeners/{id}/events     Returns the processed events
POST   /api/users                           List the Golden Records of the users and the schema
POST   /api/workspace/connect-warehouse     Connect a data warehouse
POST   /api/workspace/disconnect-warehouse  Disconnect a data warehouse
POST   /api/workspace/reload-schemas        Reload the schemas of the data warehouse
POST   /api/workspace/init-warehouse        Initialize the data warehouse