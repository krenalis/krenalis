This file contains the documentation for the REST APIs of Chichi.

GET     /api/connections                                       List the connections
GET     /api/connections/{id}                                  Get a connection
DELETE  /api/connections/{id}                                  Delete a connection
GET     /api/connections/{id}/actions                          List the actions of a connection.
POST    /api/connections/{id}/actions                          Add an action to a connection.
GET     /api/connections/{id}/actions/{id}                     Get the action of a connection.
PUT     /api/connections/{id}/actions/{id}                     Update the action of a connection.
DELETE  /api/connections/{id}/actions/{id}                     Delete the action of a connection.
POST    /api/connections/{id}/actions/{id}/execute             Execute an action of a connection.
POST    /api/connections/{id}/actions/{id}/schedule-period     Set the schedule period of an action of a connection
POST    /api/connections/{id}/actions/{id}/status              Set the status of an action of a connection
GET     /api/connections/{id}/action-types                     List the action types of a connection.
GET     /api/connections/{id}/action-types/Users               Get information about the Users action type.
GET     /api/connections/{id}/action-types/Groups              Get information about the Groups action type.
GET     /api/connections/{id}/action-types/Events              Get information about the Events action type with no event type.
GET     /api/connections/{id}/action-types/Events/{eventType}  Get information about the Events action type for the event type.
POST    /api/connections/{id}/exec-query                       Execute the query of a database connection.
GET     /api/connections/{id}/sheets                           List the sheets of a multiple sheets file connection
POST    /api/connections/{id}/status                           Set the status of a connection
GET     /api/connections/{id}/stats                            Get the stats of a connection
POST    /api/connections/{id}/reload                           Reload the schemas of the actions of a connection.
GET     /api/connections/{id}/keys                             Get the API keys of a connection
POST    /api/connections/{id}/keys                             Generate a new API key for a connection
DELETE  /api/connections/{id}/keys/{id}                        Delete the API key of a connection
PUT     /api/connections/{id}/storage/{id}                     Set the storage of a connection
GET     /api/connections/{id}/ui                               Get the user interface of a connection
POST    /api/connections/{id}/ui-event                         Execute the user interface event of a connection
GET     /api/connectors                                        List the connectors
GET     /api/connectors/{id}                                   Get a connector
POST    /api/connectors/{id}/ui                                Get the user interface of a connector
POST    /api/connectors/{id}/ui-event                          Execute the user interface event of a connector
PUT     /api/event-listeners/                                  Add a new event listener
DELETE  /api/event-listeners/{id}                              Remove an event listener
GET     /api/event-listeners/{id}/events                       Returns the processed events
POST    /api/users                                             List the Golden Records of the users and the schema
GET     /api/users/{id}/events                                 List the events of a user
GET     /api/users/{id}/traits                                 List the traits of a user
POST    /api/workspace/connect-warehouse                       Connect a data warehouse
POST    /api/workspace/disconnect-warehouse                    Disconnect a data warehouse
POST    /api/workspace/reload-schemas                          Reload the schemas of the data warehouse
POST    /api/workspace/init-warehouse                          Initialize the data warehouse
GET     /api/workspace/user-schema                             Get the user schema of the workspace
POST    /api/workspace/oauth-token                             Generate an OAuth token not yet associated with a connection
POST    /api/workspace/add-connection                          Add a new connection
GET     /api/workspace/privacy-region                          Get the workspace privacy region
POST    /api/workspace/privacy-region                          Set the workspace privacy region
GET     /api/events-schema                                     Get the events schema