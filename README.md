# chichi

![](https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it)

- [How to execute Chichi](#how-to-execute-chichi)
  - [Install React and other dependencies](#install-react-and-other-dependencies)
  - [Configure and add certificates](#configure-and-add-certificates)
  - [Compile the server](#compile-the-server)
  - [Run and open the browser](#run-and-open-the-browser)
- [Checklist before commit](#checklist-before-commit)
- [Documentation of APIs](#documentation-of-apis)
  - [/admin/api/smart-events.create](#adminapismart-eventscreate)
  - [/admin/api/smart-events.delete](#adminapismart-eventsdelete)
  - [/admin/api/smart-events.find](#adminapismart-eventsfind)
  - [/admin/api/smart-events.get](#adminapismart-eventsget)

## How to execute Chichi

### Install React and other dependencies

```
cd admin
npm install
```

It is recommended to add the `/admin/node_modules/` directory your local `.gitignore` file.

### Configure and add certificates

Add a configuration file `app.ini` (see `app.example.ini`) in the same directory of
the `chichi` executable, as well as a `cert.pem` and `key.pem` certificate files.

### Compile the server

Within the root of this repository execute:

```bash
go build -tags osusergo,netgo -trimpath
```

### Run and open the browser

Launch the server executing `chichi` (or `chichi.exe` on Windows) and visit https://localhost:9090/admin/public/.

## Checklist before commit

1. The `go.sum` and `go.mod` files must be cleared by `go mod tidy`
2. The entire project must be formatted with `go fmt ./...`
3. These features **currently work** and consequentially should be tested:
  * The login page at https://localhost:9090/admin/
  * Every example in the dashboard (https://localhost:9090/admin/dashboard) should work (note that not every example returns a set of results, it depends on the data on the database)

## Documentation of APIs

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