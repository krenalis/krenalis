
# Getting Started

This guide provides clear instructions for installing the JavaScript SDK in web browsers and seamlessly integrating it into both JavaScript and TypeScript applications.

## Step 1: Create a Source JavaScript Connection

To create a source JavaScript connection in ChiChi:

1. Click on **Connections**.

2. Click on **Add a New Connection**.

3. From the list of connectors, select the **JavaScript** connector.

4. Enter the display name and the hostname where the SDK will be installed.

5. Click on **Save**.

## Step 2: Install or Import the SDK

Below are outlined the various alternative methods for installing or importing the SDK to suit your requirements.

### Install the SDK on the Website

This is the simplest method to start collecting events.

1. In the new created JavaScript connection, navigate to **Settings**.

2. Select **Snippet**.

3. Copy the SDK snippet.

4. Paste the snippet into your website between `<head>` and `</head>`.

### Import into an Application using `import`

The JavaScript SDK can be imported with `import` into TypeScript and JavaScript projects, using ES6 modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.

2. Select **Write Key**.

3. Copy the Write Key and the Endpoint.

4. In your project, install the `chichi-javascript-sdk` npm package:

    ```sh
    npm install chichi-javascript-sdk --save
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:

    ```javascript
    import Analytics from 'chichi-javascript-sdk';
   
    const chichiAnalytics = new Analytics('<write key>', '<endpoint>');
    chichiAnalytics.page('home');
    ```

### Import into an Application using `require`

The JavaScript SDK can be imported with `require` into JavaScript projects, using CommonJS modules, that will be bundled to run in the browser.

1. In the new created JavaScript connection, navigate to **Settings**.

2. Select **Write Key**.

3. Copy the Write Key and the Endpoint.

4. In your project, install the `chichi-javascript-sdk` npm package:

    ```sh
    npm install chichi-javascript-sdk --save
    ```

5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:

    ```javascript
    const { Analytics } = require('chichi-javascript-sdk');
   
    const chichiAnalytics = new Analytics('<write key>', '<endpoint>');
    chichiAnalytics.page('home');
    ```

## Step 3: Add an Action

When the JavaScript SDK is installed on your website, using the snippet or imported in your project, you can choose to collect only the events, or import the users, or both:

1. Go to the JavaScript connection you just created and click on **Actions**.

2. Under the **Collect Events** action, click on **Add**.

3. Confirm by clicking **Add**.

4. Enable the action by toggling the switch in the **Enable** column.
