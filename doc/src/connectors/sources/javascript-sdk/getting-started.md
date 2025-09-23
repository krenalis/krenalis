{% extends "/layouts/doc.html" %}
{% macro Title string %}Getting Started{% end %}
{% Article %}

# Getting started

This guide provides clear instructions for installing the JavaScript SDK in web browsers and seamlessly integrating it into both JavaScript and TypeScript applications.

## Using the SDK

- [1. Create a source JavaScript connection](#1-create-a-source-javascript-connection)
- [2. Install or import the SDK](#2-install-or-import-the-sdk)
  - [Install the SDK on the website](#install-the-sdk-on-the-website)
  - [Import into an application using `import`](#import-into-an-application-using-import)
  - [Import into an application using `require`](#import-into-an-application-using-require)
- [3. Add an action](#3-add-an-action)

### 1. Create a source JavaScript connection

To create a source JavaScript connection in Meergo:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **JavaScript** source; you can use the search bar at the top or filter by category.
4. Click on the **JavaScript** connector. A panel will open on the right with information about **JavaScript**.
5. Click on **Add source**. The `Add JavaScript source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Strategy** field, choose the strategy with which anonymous users will be treated.
8. Click **Add**.


### 2. Install or import the SDK

Below are outlined the various alternative methods for installing or importing the SDK to suit your requirements.

#### Install the SDK on the website

This is the simplest method to start collecting events.

1. In the new created JavaScript connection, navigate to **Settings**.
2. Select **Snippet**.
3. Copy the SDK snippet.
4. Paste the snippet into your website between `<head>` and `</head>`.

#### Import into an application using `import`

The JavaScript SDK can be imported with `import` into TypeScript and JavaScript projects, using ES6 modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your project, install the `meergo-javascript-sdk` npm package:
    ```sh
    $ npm install meergo-javascript-sdk --save
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```javascript
    import Meergo from 'meergo-javascript-sdk';
   
    const meergo = new Meergo('<write key>', '<endpoint>');
    meergo.page('home');
    ```

#### Import into an application using `require`

The JavaScript SDK can be imported with `require` into JavaScript projects, using CommonJS modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your project, install the `meergo-javascript-sdk` npm package:
    ```sh
    $ npm install meergo-javascript-sdk --save
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```javascript
    const { Meergo } = require('meergo-javascript-sdk');
   
    const meergo = new Meergo('<write key>', '<endpoint>');
    meergo.page('home');
    ```

### 3. Add an action

When the JavaScript SDK is installed on your website, using the snippet or imported in your project, you can choose to collect only the events, or import the users, or both:

1. Go to the JavaScript connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
5. Enable the action by toggling the switch in the **Enabled** column.
