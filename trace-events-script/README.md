# Chichi Analytics

### Run Prettier

```sh
$ npm run prettier
```

### Build `dist/chichi.js`

```sh
$ npm run build
$ npm run bundle
```

These script commands create the `dist/chichi.js` file.

### Add the snippet to an HTML page

Add the content of the `snippet.js` file to the HTML page:

```html
<script type="text/javascript">
  <!-- copy the contents of the snippet.js file here -->
</script>
```

Replace `123456789` with the identifier of the website source connection and replace `'../dist/chichi.js'` with the
URL of the `dist/chichi.js` script.
