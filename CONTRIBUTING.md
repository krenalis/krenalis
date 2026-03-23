# Contributing to Meergo

If you're reading this, we would first like to thank you for taking the time to contribute.

The following is a set of guidelines for contributing to Meergo and its repositories, which are hosted in the [Meergo Organization](https://github.com/meergo) on GitHub. These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

#### Table Of Contents

- [Code of Conduct](#code-of-conduct)
- [Meergo repositories and packages](#meergo-repositories-and-packages)
  - [SDKs for event collection](#sdks-for-event-collection)
- [Reporting bugs](#reporting-bugs)
- [Report security issues](#report-security-issues)
- [Contributing to the source code](#contributing-to-the-source-code)
  - [Build a new connector](#build-a-new-connector)
  - [Development setup](#development-setup)
  - [Pull Requests](#pull-requests)
    - [Before opening a pull request](#before-opening-a-pull-request)
    - [After opening a pull request](#after-opening-a-pull-request)
  - [Running Meergo from repository using Docker Compose](#running-meergo-from-repository-using-docker-compose)

## Code of Conduct

By participating, you are expected to adhere to the [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to [conduct@krenalis.com](mailto:conduct@krenalis.com).

## Meergo repositories and packages

Meergo is an **open development project** and is **freely available**. Connectors, warehouse integrations, and SDKs are MIT-licensed (open source). These are the areas where we expect most community contributions.

Here's a list of Meergo components:

- [meergo](https://github.com/krenalis/krenalis) - Meergo monorepo (Multiple licenses)
- [meergo/admin](https://github.com/krenalis/krenalis/tree/main/admin) - Meergo Admin console (Elastic License v2)
- [meergo/connectors](https://github.com/krenalis/krenalis/tree/main/connectors) - Meergo connectors (MIT)
- [meergo/core](https://github.com/krenalis/krenalis/tree/main/core) - Meergo Core (Elastic License v2)
- [meergo/tools](https://github.com/krenalis/krenalis/tree/main/tools) - Meergo tools (Elastic License v2)
- [meergo/warehouses](https://github.com/krenalis/krenalis/tree/main/warehouses) - Meergo warehouse integrations (MIT)

### SDKs for event collection

All SDKs are released under the MIT License.

Issues for the various SDKs are tracked in the main Meergo repository, using a specific label for each SDK as shown below.

| SDK        | Repository                                                             | Issues Label                                                                                          |
|------------|------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------|
| JavaScript | [analytics-javascript](https://github.com/meergo/analytics-javascript) | [javascript-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Ajavascript-sdk) |
| Android    | [analytics-android](https://github.com/meergo/analytics-android)       | [android-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Aandroid-sdk)       |
| Python     | [analytics-python](https://github.com/meergo/analytics-python)         | [python-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Apython-sdk)         |
| .NET       | [analytics-dotnet](https://github.com/meergo/analytics-dotnet)         | [dotnet-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Adotnet-sdk)         |
| Java       | [analytics-java](https://github.com/meergo/analytics-java)             | [java-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Ajava-sdk)             |
| Node.js    | [analytics-nodejs](https://github.com/meergo/analytics-nodejs)         | [nodejs-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Anodejs-sdk)         |
| Go         | [analytics-go](https://github.com/meergo/analytics-go)                 | [go-sdk](https://github.com/krenalis/krenalis/issues?q=state%3Aopen%20label%3Ago-sdk)                 |

## Reporting bugs

This section explains how to report a bug in **Meergo**. Following these steps helps everyone understand the problem and fix it faster.

If you find a bug in Meergo, please do the following:

1. **Verify that it is actually a bug.**\
   It may sound obvious, but many issues turn out to be configuration problems or environment errors.
   For example, if you encounter an "Internal Server Error" in the Admin console, check Meergo's log file or standard error output first.

2. **Check existing issues.**  
   Look at the [open issues](https://github.com/krenalis/krenalis/issues) to see if someone already reported it.
    * If you find the same issue, add a comment if you have new or useful information.
    * Otherwise, adding a reaction (👍) is enough to show that it affects you too, but in this case subscribe to the issue to receive notifications.

3. **Check your version.**  
   Make sure you are using the **latest version** of Meergo.
    * If you can update, please do so and see if the problem is still there.
    * If you can't update, but can test with the latest version, that helps a lot too.

4. **Isolate the problem.**  
   Try to reduce the problem to a minimal, reproducible example. The easier it is to reproduce, the faster we can fix it.

5. **Open a new issue on GitHub.**  
   Include these details:
    * A **clear title** that describes the problem.
    * **Steps to reproduce** the issue, including any configuration, inputs, or environment details.
    * What you **expected to happen** and what **actually happened**. This helps others see the issue from your point of view.
    * **Environment details:** which version of Meergo and Go you're using, your operating system, and whether you're running Meergo from source or with Docker Compose.
    * A **screenshot or GIF**, if possible — sometimes an image explains things faster than words.

## Report security issues

For security vulnerability reporting see [SECURITY.md](SECURITY.md).

## Contributing to the source code

### Build a new connector

To build a new connector for Meergo see the [Create new connector](https://www.meergo.com/docs/create-new-connector) documentation.

### Development setup

To build and test Meergo locally see the [contributing guidelines](https://www.meergo.com/docs/contributing-guidelines). 

### Pull Requests

#### Before opening a pull request

* Open a pull request only if there's an issue and agreement, except for minor fixes.

#### After opening a pull request

* A reviewer will usually start reviewing your pull request within two business days.
* Once all requested changes have been addressed, the reviewer will merge your pull request.

### Running Meergo from repository using Docker Compose

If you just want to run Meergo with Docker, see [Install using Docker Compose](https://www.meergo.com/docs/installation/using-docker-compose).

Otherwise, if you are developing Meergo and want to test it with Docker, you can run it with Docker Compose by running, inside the repository:

```
docker compose up --build
```
