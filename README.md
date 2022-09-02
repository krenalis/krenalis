# chichi

![](https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it)

- [How to compile the server](#how-to-compile-the-server)
- [Install React and other dependencies](#install-react-and-other-dependencies)
- [How to run](#how-to-run)

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
