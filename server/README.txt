This file contains the documentation for the REST APIs of Chichi.

GET     /api/workspaces/{id}/connections                                         List the connections.
GET     /api/workspaces/{id}/connections/{id}                                    Get a connection.
POST    /api/workspaces/{id}/connections/{id}                                    Set a connection.
DELETE  /api/workspaces/{id}/connections/{id}                                    Delete a connection.
POST    /api/workspaces/{id}/connections/{id}/actions                            Add an action to a connection.
GET     /api/workspaces/{id}/connections/{id}/actions/{id}                       Get the action of a connection.
PUT     /api/workspaces/{id}/connections/{id}/actions/{id}                       Update the action of a connection.
DELETE  /api/workspaces/{id}/connections/{id}/actions/{id}                       Delete the action of a connection.
POST    /api/workspaces/{id}/connections/{id}/actions/{id}/execute               Execute an action of a connection.
POST    /api/workspaces/{id}/connections/{id}/actions/{id}/schedule-period       Set the schedule period of an action of a connection.
POST    /api/workspaces/{id}/connections/{id}/actions/{id}/status                Set the status of an action of a connection.
GET     /api/workspaces/{id}/connections/{id}/action-schemas/Users               Get the input and output schemas of the Users action type.
GET     /api/workspaces/{id}/connections/{id}/action-schemas/Groups              Get the input and output schemas of the Groups action type.
GET     /api/workspaces/{id}/connections/{id}/action-schemas/Events              Get the input and output schemas of the Events action type with no event type.
GET     /api/workspaces/{id}/connections/{id}/action-schemas/Events/{eventType}  Get the input and output schemas of the Events action type for the event type.
POST    /api/workspaces/{id}/connections/{id}/app-users                          Get the app users.
GET     /api/workspaces/{id}/connections/{id}/complete-path/{path}               Return the complete representation of a path for a connection storage.
POST    /api/workspaces/{id}/connections/{id}/event-preview                      Return an event preview.
POST    /api/workspaces/{id}/connections/{id}/exec-query                         Execute the query of a database connection.
GET     /api/workspaces/{id}/connections/{id}/executions                         Return the executions of a connection.
POST    /api/workspaces/{id}/connections/{id}/identities                         List the user identities of a connection.
GET     /api/workspaces/{id}/connections/{id}/records                            Return the records and the schema of a file of a multiple sheets file connection.
GET     /api/workspaces/{id}/connections/{id}/sheets                             List the sheets of a multiple sheets file connection.
GET     /api/workspaces/{id}/connections/{id}/stats                              Get the stats of a connection.
GET     /api/workspaces/{id}/connections/{id}/tables/{table}/schema              Return the schema of the given table of a database connection.
GET     /api/workspaces/{id}/connections/{id}/keys                               Get the write keys of a connection.
POST    /api/workspaces/{id}/connections/{id}/keys                               Generate a new write key for a connection.
DELETE  /api/workspaces/{id}/connections/{id}/keys/{key}                         Delete a write key of a connection.
GET     /api/workspaces/{id}/connections/{id}/ui                                 Get the user interface of a connection.
POST    /api/workspaces/{id}/connections/{id}/ui-event                           Execute the user interface event of a connection.
GET     /api/connectors                                                          List the connectors.
GET     /api/connectors/{id}                                                     Get a connector.
POST    /api/workspaces/{id}/ui                                                  Get the user interface of a connector.
POST    /api/workspaces/{id}/ui-event                                            Execute the user interface event of a connector.
GET     /api/connectors/{id}/auth-code-url                                       Return the URL that directs to the consent page of an OAuth 2.0 provider.
PUT     /api/workspaces/{id}/event-listeners/                                    Add a new event listener.
DELETE  /api/workspaces/{id}/event-listeners/{id}                                Remove an event listener.
GET     /api/workspaces/{id}/event-listeners/{id}/events                         Return the processed events.
POST    /api/workspaces/{id}/users                                               List the Golden Records of the users, their schema and an estimated count.
GET     /api/workspaces/{id}/users/{id}/events                                   List the events of a user.
POST    /api/workspaces/{id}/users/{id}/identities                               List the identities of a user.
GET     /api/workspaces/{id}/users/{id}/traits                                   List the traits of a user.
GET     /api/workspaces/{id}                                                     Get the workspace.
DELETE  /api/workspaces/{id}                                                     Delete the workspace.
GET     /api/workspaces                                                          List the workspaces.
POST    /api/workspaces                                                          Add a new workspace.
PUT     /api/workspaces/{id}                                                     Set the name and the privacy region of the workspace.
POST    /api/workspaces/{id}/identifiers                                         Set the identifiers of the workspace.
POST    /api/workspaces/{id}/connect-warehouse                                   Connect the workspace to a data warehouse.
GET     /api/workspaces/{id}/warehouse-settings                                  Get the settings of the data warehouse for the workspace.
PUT     /api/workspaces/{id}/warehouse-settings                                  Change the settings of the data warehouse for the workspace.
POST    /api/workspaces/{id}/disconnect-warehouse                                Disconnect the data warehouse.
POST    /api/workspaces/{id}/ping-warehouse                                      Ping a data warehouse.
POST    /api/workspaces/{id}/init-warehouse                                      Initialize the data warehouse.
GET     /api/workspaces/{id}/user-schema                                         Get the user schema of the workspace.
GET     /api/workspaces/{id}/identifiers-schema                                  Get a schema whose properties can be used as identifiers for the workspace.                     
POST    /api/workspaces/{id}/run-identity-resolution                             Run the Workspace Identity Resolution on the workspace.
POST    /api/workspaces/{id}/oauth-token                                         Generate an OAuth token not yet associated with a connection.
POST    /api/workspaces/{id}/add-connection                                      Add a new connection.
GET     /api/workspaces/{id}/privacy-region                                      Get the workspace privacy region.
GET     /api/events-schema                                                       Get the events schema (which does not include the GID).
POST    /api/expressions-properties                                              Return the unique properties contained inside a list of expressions.
GET     /api/transformation-languages                                            Return the supported transformation languages.
POST    /api/transform-data                                                      Transform data, using a mapping or a transformation, and returns it.
POST    /api/validate-expression                                                 Validate an expression.
GET     /api/members                                                             List the members.
POST    /api/members/invitations                                                 Create an invited member and send the invitation email.
GET     /api/members/invitations/{token}                                         Get the organization's name and email of the invited member.
PUT     /api/members/invitations/{token}                                         Accept an invitation and set the name and password of the invited member.
POST    /api/members/login                                                       Authenticate a member given his email and password.
POST    /api/members/logout                                                      Authenticate a member given his email and password.
GET     /api/members/current                                                     Get the current member.
PUT     /api/members/current                                                     Update the current member.
DELETE  /api/members/{id}                                                        Delete the member.