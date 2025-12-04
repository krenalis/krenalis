<h1>Meergo</h1>

[Meergo](https://www.meergo.com) is a Customer Data Platform (CDP) that helps data analysts and the marketing team collect, enrich, unify, and activate user and customer data — including product events, marketing signals, and sales interactions — and syncs it to the tools your teams use every day.


Meergo is open and free to use. It's a lightweight, developer-friendly alternative to Customer Data Platforms such as Segment, RudderStack, and mParticle.  

It is built in Go with a modern TypeScript/React frontend and a comprehensive REST API, and ships as a **single executable** that includes:

- **Real-time** event collection and dispatch
- **Batch data ingestion** from SaaS apps, databases, and files
- **Transformations** with Visual Mapping, JavaScript, or Python
- **Identity resolution** that runs directly inside **your data warehouse**
- A unified, **360-degree customer profile** view
- **Activation** of profiles and events to SaaS apps, databases, and files
- **Snowflake and PostgreSQL** support as primary event and profile data storage

## What Makes Meergo Different

Meergo unifies customer data **across your entire stack** without relying on traditional one-to-one integrations, which often introduce fragmentation and maintenance overhead as systems scale. It provides end-to-end visibility into the customer data pipeline and gives you fine-grained control over data modeling, enrichment logic, and downstream delivery.

Your customer data lives **directly in your data warehouse**, with native support for **Snowflake and PostgreSQL**. Identity resolution also runs **inside the warehouse**, allowing you to maintain a consistent and reproducible customer graph using the infrastructure you already trust.

This architecture is especially well-suited for AI workloads. Both **AI agents** and **models** can consume clean, unified, and continuously updated customer data directly from your warehouse, without additional pipelines.

Meanwhile, **real-time, bidirectional sync** keeps operational tools aligned with warehouse state — reducing operational burden, improving data reliability, and eliminating the need for separate data pipelines or orchestration layers.

> [!WARNING]
> Meergo is under active development. Breaking changes may occur until the project reaches its 1.0.0 release.

## Event Analytics SDKs

The easiest way to collect events is to use one of the official SDKs:

- **JavaScript (Browser)** — https://github.com/meergo/analytics-javascript  
- **Android** — https://github.com/meergo/analytics-android  
- **Apple** — _coming soon_  
- **Python** — https://github.com/meergo/analytics-python  
- **.NET** — https://github.com/meergo/analytics-dotnet  
- **Java** — https://github.com/meergo/analytics-java  
- **Node.js** — https://github.com/meergo/analytics-nodejs  
- **Go** — https://github.com/meergo/analytics-go  

## Get Started

For full documentation, visit **https://www.meergo.com/docs**

### Run Meergo with Docker Compose

To evaluate Meergo locally, you can run a complete instance without setting up the full environment.

It includes the standalone meergo executable, along with Node.js and Python for local transformations, and a PostgreSQL database—provided for convenience as both the internal support store and an optional warehouse for event and profile data.

Navigate to the directory where you want to run Meergo and execute:

```
mkdir -p storage
curl -f "https://raw.githubusercontent.com/meergo/meergo/refs/tags/v0.16.0/tools/docker-compose/compose-release.yaml" -o compose.yaml
docker compose up
```

### Building and running from source

To build the standalone executable, run `go generate && go build` in the root directory of this repository:

1. [Install Go 1.23+](https://go.dev/doc/install) (*if you haven't already*)
2. Clone or download this repository
3. Check out the latest release
4. Run `go generate`
5. Run `go build`
   (*See also [https://go.dev/doc/install/source#environment](https://go.dev/doc/install/source#environment)*)

To run the executable, provide an empty PostgreSQL database on first run, starting the `meergo` command:

1. Set your PostgreSQL credentials using the recommended [environment variables](https://www.meergo.com/docs/configuration/environment-variables#database)
2. Run `./meergo -init-db-if-empty`

## Security

We take the security of Meergo and its ecosystem seriously.

If you discover a potential vulnerability, please report it privately to **[security@meergo.com](mailto:security@meergo.com)** rather than opening a public issue. See the [SECURITY](https://github.com/meergo/meergo/blob/main/SECURITY.md) file for details.

We will acknowledge your report as quickly as possible and keep you updated throughout the resolution process. Valid reports will be credited in the release notes once the fix is published.

## Contributing

Meergo is an open, community-driven project. We welcome contributions from developers of all backgrounds — whether you're fixing a bug, improving documentation, or building a new connector.

### Ways to contribute

* [Contributing to the source code](https://github.com/meergo/meergo/blob/main/CONTRIBUTING.md)
* [Suggesting new features and reporting issues](https://github.com/meergo/meergo/issues)

To keep the project coherent and maintainable, we follow a roadmap and prioritize issues in a specific order.
For this reason, please open an issue or discussion before submitting a pull request for a new feature.

## License

Meergo is licensed under the **MIT License** (for connectors and warehouse drivers) and the **Elastic 2.0 License** (for the core and admin). Both licenses are documented in the [LICENSE](https://github.com/meergo/meergo/blob/main/LICENSE) file.
You are free to use, modify, and integrate the project according to the terms described there.
