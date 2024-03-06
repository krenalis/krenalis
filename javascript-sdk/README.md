# Chichi Analytics

- [Format files](#format-files)
- [Build `dist/chichi.min.js`](#build-distchichiminjs)
- [Add the snippet to an HTML page](#add-the-snippet-to-an-html-page)
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
npm run fmt
```

## Build `dist/chichi.min.js`

Run:

```sh
npm run build
```

As an alternative, you can perform the build in three steps:

```sh
npm run bundle
npm run transpile
npm run minify
```

* `npm run bundle` bundles the `chichi.js` file and creates the `build/chichi.bundle.js` file.
* `npm run transpile` transpiles the `build/chichi.bundle.js` file to ES5 and creates the `build/chichi.es5.js` file.
* `npm run minify` minifies the `build/chichi.es5.js` file and creates the `dist/chichi.min.js` file.

## Add the snippet to an HTML page

Add the content of the `snippet.js` file to the HTML page:

```html
<script type="text/javascript">
  // Copy the contents of the snippet.js file here.
</script>
```

Replace `kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc` with a write key of the JavaScript source connection and replace `'../dist/chichi.min.js'` with the
URL of the `dist/chichi.min.js` script.

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
import Analytics from "../chichi/javascript-sdk";
const analytics = new Analytics("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/batch");
analytics.page("home");
```

To import the JavaScript SDK into a browser as an ES6 module, follow these steps:

1. Build the `dist/chichi.es6.min.js` module file:

   ```sh
   npm run build:es6
   ```

2. Import the module in the browser:

   ```html
   <script type="module">
       import Analytics from "https://example.com/chichi.es6.min.js";
       const analytics = new Analytics("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/batch");
       analytics.page("home");
   </script>
   ```

### ES6 Module Compatibility

Check the browser compatibility for ES6 module usage and dynamic import: https://caniuse.com/es6-module-dynamic-import

## CJS Module

To import the JavaScript SDK into an application using CommonJS (CJS) using the `require` function, follow these steps:

1. Build the CJS module:

   ```sh
   npm run build:cjs
   ```

2. Import the module in the application:

    ```javascript
    const { Analytics } = require("../chichi/javascript-sdk");
    const analytics = new Analytics("kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc", "https://localhost:9090/api/v1/batch");
    analytics.page("home");
    ```

## Execute Tests

Run:

```sh
npm run test
```
