# chichi

![](https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it)

- [How to compile the server](#how-to-compile-the-server)
- [Install React and other dependencies](#install-react-and-other-dependencies)
- [How to run](#how-to-run)
- [APIs](#apis)
  - [/admin/api/smart-events.create](#adminapismart-eventscreate)
  - [/admin/api/smart-events.delete](#adminapismart-eventsdelete)
  - [/admin/api/smart-events.find](#adminapismart-eventsfind)
  - [/admin/api/smart-events.get](#adminapismart-eventsget)
  - [/admin/api/smart-events.update](#adminapismart-eventsupdate)

## How to compile the server

```bash
go build -tags osusergo,netgo -trimpath
```

## Install React and other dependencies

```
cd admin
npm install
```

It is recommended to add the `/admin/node_modules/` directory your local `.gitignore` file.

## How to run

Add a configuration file `app.ini` (see `app.example.ini`) in the same directory of
the `chichi` executable, as well as a `cert.pem` and `key.pem` certificate files.

Then, launch the server executing `chichi` and visit the URL
https://localhost:9090/admin/public/.

## APIs

TODO(Gianluca): this is just a draft of the documentation. Improve it.

### /admin/api/smart-events.create

Request:

```
<Smart Event>
```

Response:

```
<Smart Event ID>
```

### /admin/api/smart-events.delete

Request:

```
[<id #1, id #2...>]
```

Response: nothing

### /admin/api/smart-events.find

Request: nothing.

Response:

```
[
   <smart event #1>,
   <smart event #2>...
]
```


### /admin/api/smart-events.get

Request:

```
<id>
````

Response:

If ID corresponds to a Smart Event, returns:

```
<smart event #1>
```

### /admin/api/smart-events.update

Request:

```
{
   "ID": <Smart Event ID>,
   "SmartEvent": <Smart Event>
}
```