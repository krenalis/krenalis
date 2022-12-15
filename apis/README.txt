This file contains the documentation for the REST APIs of Chichi.

GET    /api/connections                       List the connections
GET    /api/connections/{id}/properties       List the properties of a connection
GET    /api/connections/{id}/transformations  List the transformations of a connection
PUT    /api/connections/{id}/transformations  Set the transformations of a connection
POST   /api/connections/{id}/import           Import from a connection
POST   /api/connections/{id}/reimport         Import from a connection
PUT    /api/event-listeners/                  Add a new event listener
DELETE /api/event-listeners/{id}              Remove an event listener
GET    /api/event-listeners/{id}/events       Returns the processed events
POST   /api/connections/{id}/export           Export to connection
PUT    /api/transformations/                  Create a new transformation
PATCH  /api/transformations/{id}              Update a transformation
