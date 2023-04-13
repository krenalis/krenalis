
## Connection

### ServeUI

* Settings
* Firehose
* ClientSecret (only for app connections)
* Resource     (only for app connections)
* AccessToken  (only for app connections)

### SettingsUI

* Settings
* Firehose
* ClientSecret (only for app connections)
* Resource     (only for app connections)
* AccessToken  (only for app connections)

## AppConnection

### EventTypes

* Settings
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### EventTypeSchema

* Settings
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### GroupSchema

* Settings
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### Groups

* Settings
* Firehose
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### ReceiveWebhook

If webhooksPer is "Connector":

* ClientSecret

If webhooksPer is "Resource":

* ClientSecret
* Resource
* AccessToken

If webhooksPer is "Connection":

* Settings
* Firehose
* ClientSecret
* Resource
* AccessToken

### Resource

* ClientSecret
* AccessToken
* PrivacyRegion

### SendEvent

* Settings
* PrivacyRegion

### SetGroups

* Settings
* Firehose
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### SetUsers

* Settings
* Firehose
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

### UserSchema

* Settings
* ClientSecret
* Resource
* AccessToken
* PrivacyRegio

### Users

* Settings
* Firehose
* ClientSecret
* Resource
* AccessToken
* PrivacyRegion

## DatabaseConnection

### Query

* Settings
* Firehose

## StorageConnection

### Reader

* Settings
* Firehose

### Writer

* Settings
* Firehose

## StreamConnection

### Close

* Settings

### Commit

* Settings

### Send

* Settings

### Receive

* Settings

## FileConnection

### Read

* Settings
* Firehose

### Write

* Settings
* Firehose

