# Chichi Analytics

- [Format files](#format-files)
  - [Formatting files under `website-for-testing`](#formatting-files-under-website-for-testing)
- [Build `dist/chichi.js`](#build-distchichijs)
- [Add the snippet to an HTML page](#add-the-snippet-to-an-html-page)

### Install dependencies

Run:

```sh
npm install
```

## Format files

Run:

```sh
npm run prettier
```

### Formatting files under `website-for-testing`

Run:

```sh
npm run prettier-test-website
```

## Build `dist/chichi.js`

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
* `npm run transpile` transpile the `build/chichi.bundle.js` file to ES5 and creates the `build/chichi.es5.js` file.
* `npm run minify` minifies the `build/chichi.es5.js` file and creates the `dist/chichi.js` file.

## Add the snippet to an HTML page

Add the content of the `snippet.js` file to the HTML page:

```html
<script type="text/javascript">
  // Copy the contents of the snippet.js file here.
</script>
```

Replace `kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc` with a write key of the JavaScript source connection and replace `'../dist/chichi.js'` with the
URL of the `dist/chichi.js` script.
