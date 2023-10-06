# Chichi Analytics

- [Format files](#format-files)
  - [Formatting files under `website-for-testing`](#formatting-files-under-website-for-testing)
- [Build `dist/chichi.js`](#build-distchichijs)
- [Add the snippet to an HTML page](#add-the-snippet-to-an-html-page)


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
npm run bundle
```

These script commands create the `dist/chichi.js` file.

## Add the snippet to an HTML page

Add the content of the `snippet.js` file to the HTML page:

```html
<script type="text/javascript">
  // Copy the contents of the snippet.js file here.
</script>
```

Replace `kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc` with a write key of the JavaScript source connection and replace `'../dist/chichi.js'` with the
URL of the `dist/chichi.js` script.
