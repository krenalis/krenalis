This file contains documentation for the REST APIs of Chichi.

GET    /apis/data-sources                       List the data sources
GET    /apis/data-sources/{id}/properties       List the properties of a data source
GET    /apis/data-sources/{id}/transformations  List the transformations of a data source
POST   /apis/data-sources/{id}/import           Import from a data source
POST   /apis/data-sources/{id}/reimport         Import from a data source
PUT    /apis/transformations/                   Create a new transformation
PATCH  /apis/transformations/{id}               Update a transformation
