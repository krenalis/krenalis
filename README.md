# chichi

<img src="https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it" width=260px/>

- [Before commit](#before-commit)
- [How to execute Chichi](#how-to-execute-chichi)
  - [Install React and other dependencies](#install-react-and-other-dependencies)
  - [Configure and add certificates](#configure-and-add-certificates)
  - [Compile the server](#compile-the-server)
  - [Run and open the browser](#run-and-open-the-browser)
  - [Expose on the Internet (optional)](#expose-on-the-internet-optional)
- [Interact with Chichi using `chichi-cli`](#interact-with-chichi-using-chichi-cli)

## Before commit

From the root of this repository, run:

```
go run commit/commit.go
```

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

Launch the server executing `chichi` (or `chichi.exe` on Windows) and visit https://localhost:9090/admin/.

### Expose on the Internet (optional)

1. Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
2. Check that it is installed correctly: `cloudflared --version`
3. Run cloudflared: `cloudflared tunnel --url https://localhost:9090`
4. Make a note of the URL listed in the standard output (example:  https://xxxxxxx.trycloudflare.com)
5. Open the URL in a browser

## Interact with Chichi using `chichi-cli`

Refer to the [documentation of the chichi-cli tool](chichi-cli/README.md).

