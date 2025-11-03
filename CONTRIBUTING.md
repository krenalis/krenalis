# Contributing to Meergo

If you're reading this, we would first like to thank you for taking the time to contribute.

The following is a set of guidelines for contributing to Meergo and its repositories, which are hosted in the [Meergo Organization](https://github.com/meergo) on GitHub. These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

#### Table Of Contents

- [Code of Conduct](#code-of-conduct)
- [Asking questions](#asking-questions)
- [Meergo repositories and packages](#meergo-repositories-and-packages)
  - [SDKs for event collection](#sdks-for-event-collection)
- [Reporting bugs](#reporting-bugs)
- [Contributing to the source code](#contributing-to-the-source-code)
  - [Development setup](#development-setup)
  - [Pull Requests](#pull-requests)
    - [Good commit messages](#good-commit-messages)
    - [Before opening a pull request](#before-opening-a-pull-request)
    - [After opening a pull request](#after-opening-a-pull-request)

## Code of Conduct

By participating, you are expected to adhere to the [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to [dev@meergo.com](mailto:dev@meergo.com).

## Asking questions

If you have a question **about using or developing** Meergo, join our [Slack Community](https://www.meergo.com/slack).

## Meergo repositories and packages

Meergo is an **open development project** and is **freely available**. Connectors, warehouse drivers, and SDKs are MIT-licensed (open source). These are the areas where we expect most community contributions.

Here's a list of Meergo components:

- [meergo](https://github.com/meergo/meergo) - Meergo monorepo (Multiple licenses)
- [meergo/assets](https://github.com/meergo/meergo/tree/main/assets) - Meergo Admin console (Elastic License v2)
- [meergo/core](https://github.com/meergo/meergo/tree/main/core) - Meergo Core (Elastic License v2)
- [meergo/connectors](https://github.com/meergo/meergo/tree/main/connectors) - Meergo connectors (MIT)
- [meergo/warehouses](https://github.com/meergo/meergo/tree/main/warehouses) - Meergo warehouse drivers (MIT)

### SDKs for event collection

All SDKs are released under the MIT License.

Issues for the various SDKs are tracked in the main Meergo repository, using a specific label for each SDK as shown below.

| SDK        | Repository                                                             | Issues Label                                                                                      |
|------------|------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
| JavaScript | [analytics-javascript](https://github.com/meergo/analytics-javascript) | [javascript-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Ajavascript-sdk) |
| Android    | [analytics-android](https://github.com/meergo/analytics-android)       | [android-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Aandroid-sdk)       |
| Python     | [analytics-python](https://github.com/meergo/analytics-python)         | [python-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Apython-sdk)         |
| .NET       | [analytics-dotnet](https://github.com/meergo/analytics-dotnet)         | [dotnet-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Adotnet-sdk)         |
| Java       | [analytics-java](https://github.com/meergo/analytics-java)             | [java-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Ajava-sdk)             |
| Node.js    | [analytics-node](https://github.com/meergo/analytics-java)             | [node-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Anode-sdk)             |
| Go         | [analytics-go](https://github.com/meergo/analytics-go)                 | [go-sdk](https://github.com/meergo/meergo/issues?q=state%3Aopen%20label%3Ago-sdk)                 |

## Reporting bugs

This section explains how to report a bug in **Meergo**. Following these steps helps everyone understand the problem and fix it faster.

If you find a bug in Meergo, please do the following:

1. **Verify that it is actually a bug.**\
   It may sound obvious, but many issues turn out to be configuration problems or environment errors.
   For example, if you encounter an "Internal Server Error" in the Admin console, check Meergo's log file or standard error output first.

2. **Check existing issues.**  
   Look at the [open issues](https://github.com/meergo/meergo/issues) to see if someone already reported it.
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

To build and test Meergo locally see [DEVELOPERS.md](DEVELOPERS.md). 

### Pull Requests

#### Good commit messages

The following is an example of a good commit message:

```
core/http: normalize `Accept-Language` header

Align the header parser with RFC 9110, removing superfluous spaces and
sorting locales by quality. This prevents cache misses when the browser
sends duplicate values.

Fixes #2031
```

* **First line.** Write a concise, one-line summary of the change, starting with the relevant package path as a prefix.
  If multiple packages are affected, separate them with commas or use a common ancestor.
  After the prefix, use the present tense imperative form (e.g., `"fix ..."`, not `"fixed ..."`).

* **Main content.** Explain what the change does and why it was made. Use clear, complete sentences. Avoid HTML and complex Markdown. Basic Markdown (lists, code, quotes) is fine.

* **Line length.** Each line should be no longer than 72 characters.

* **Names.** When referring to identifiers or code elements, enclose them in backticks (like this `).

#### Before opening a pull request

* Open a pull request only if there's an issue and agreement, except for minor fixes.
* AI-assisted code\
  You must be the author of the code. If you used an AI tool to assist you, you are expected to have fully reviewed and deeply understood the code as if you had written it yourself. You must be able to personally address any feedback or requested changes from reviewers.
* Licensing

  | Area                         | License             |
  |------------------------------|---------------------|
  | Core, Admin console          | Elastic License 2.0 |
  | Connectors, Warehouses, SDKs | MIT (open source)   |

  * Code that introduces a connector or driver for a data warehouse must be released under the MIT License.
  * Code that contributes to the Core or any other area covered by the Elastic License v2 must comply with that license and include acceptance of the Contributor License Agreement (CLA).
* Dependencies\
  In general, pull requests should not add new external dependencies. Exceptions apply to new connectors for databases, files, or file storages, which may require dependencies to connect, read, or write specific formats. Dependencies must use MIT, BSD, or Apache 2.0 licenses. Any other license requires prior approval.
* Third-party code\
  If you need to include third-party code, it must have been distributed by its author under an MIT, BSD, or Apache 2.0 license.
  * For a few lines of code, you may include them inline, but clearly mark them and cite the original author and license.
  * For larger portions, place the code in a separate file with the author and license explicitly stated in the file header.
* Run `"go run commit/commit.go"` before opening the pull request.


#### After opening a pull request

* An initial review may be performed by an AI system. You are free to accept, decline, or ignore any AI suggestion.
* A human reviewer will usually start reviewing your pull request within two business days.
* A reviewer will always perform a full review of the code, regardless of any prior AI review.
* Once all requested changes have been addressed, the reviewer will merge your pull request.