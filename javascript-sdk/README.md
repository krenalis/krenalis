# Meergo JavaScript SDK

> [!NOTE]
> This documentation is for the developers of the Meergo JavaScript SDK.
> 
> If you are looking for documentation for using the SDK, [you can find it here](http://localhost:8080/developers/javascript-sdk).

<h2>Table of contents</h2>

- [Format files](#format-files)
- [Build `dist/meergo.min.js`](#build-distmeergominjs)
- [Minimum Supported Browsers](#minimum-supported-browsers)
- [ES6 Module](#es6-module)
  - [ES6 Module Compatibility](#es6-module-compatibility)
- [CJS Module](#cjs-module)
- [Execute Tests](#execute-tests)

### Install dependencies

The following dependencies are required:

* Node.js with npm
* Deno

Run:

```sh
npm install
```

## Format files

Run:

```sh
deno fmt
```

## Build `dist/meergo.min.js`

Run:

```sh
deno task build
```

As an alternative, you can perform the build in three steps:

```sh
deno task bundle
deno task transpile
deno task minify
```

* `deno task bundle` bundles the `main.js` file and creates the `build/meergo.bundle.js` file.
* `deno task transpile` transpiles the `build/meergo.bundle.js` file to ES5 and creates the `build/meergo.es5.js` file.
* `deno task minify` minifies the `build/meergo.es5.js` file and creates the `dist/meergo.min.js` file.

## Minimum Supported Browsers

* Chrome 23
* Edge 80
* Safari 7
* Firefox 21
* Opera 14
* IE 11

## ES6 Module

To import the JavaScript SDK into an application as an ES6 module:

```javascript
import Meergo from "../meergo/javascript-sdk";
const meergo = new Meergo("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/events");
meergo.page("home");
```

To import the JavaScript SDK into a browser as an ES6 module, follow these steps:

1. Build the `dist/meergo.es6.min.js` module file:

   ```sh
   deno task build:es6
   ```

2. Import the module in the browser:

   ```html
   <script type="module">
       import Meergo from "https://example.com/meergo.es6.min.js";
       const meergo = new Meergo("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/events");
       meergo.page("home");
   </script>
   ```

### ES6 Module Compatibility

Check the browser compatibility for ES6 module usage and dynamic import: https://caniuse.com/es6-module-dynamic-import

## CJS Module

To import the JavaScript SDK into an application using CommonJS (CJS) using the `require` function, follow these steps:

1. Build the CJS module:

   ```sh
   deno task build:cjs
   ```

2. Import the module in the application:

    ```javascript
    const { Meergo } = require("../meergo/javascript-sdk");
    const meergo = new Meergo("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/events");
    meergo.page("home");
    ```

## Execute Tests

Run:

```sh
deno test
```
