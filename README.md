# App Scaffold

A full working example of a serverless LLM chat application on AWS.
Designed as a template for building SaaS apps with:

- **Near-zero cost** at low usage (all services within AWS free tier)
- **Elastic scalability** (Lambda, S3, CloudFront scale automatically)
- **Replatformable** (portable Go packages, static frontend, interface-driven storage)

## Architecture

Serverless HTML-over-the-wire. CloudFront routes to three origins:
S3 (static assets), S3 (generated HTML fragments), and Go Lambda (SSR + API).
A Python Lambda handles async LLM calls to Bedrock.

```
Browser ──HTTPS── CloudFront ──┬── /static/* ──── S3 (CSS)
                               ├── /generated/* ── S3 (chat data + HTML fragments)
                               └── /* ──────────── Go Lambda (templ SSR)
                                                      │
                                                      ├── async invoke
                                                      ▼
                                               Python Lambda ── Bedrock (Claude)
```

## Documentation

- [Architecture](doc/architecture.md) — system design, origins, design decisions
- [Request Flows](doc/flow.md) — sequence diagrams for every request path
- [Technology Stack](doc/stack.md) — runtimes, dependencies, dev tools
- [Project Structure](doc/structure.md) — directory layout and conventions

## Stack

| Layer | Technology |
|-------|-----------|
| CDN | CloudFront |
| Compute | Go Lambda + Python Lambda |
| Storage | S3 (chat JSON + HTML fragments) |
| HTML Templates | [templ](https://templ.guide) (Go) |
| Frontend | HTMX + Alpine.js + Tailwind CSS |
| LLM | Bedrock (Converse API, configurable model) |
| IaC | AWS CDK (TypeScript) |

## Getting Started

### Prerequisites

- [Go](https://go.dev) 1.23+
- [Python](https://python.org) 3.13+
- [Node.js](https://nodejs.org) 22+
- [AWS CDK](https://aws.amazon.com/cdk/) CLI (`npm install -g aws-cdk`)
- [templ](https://templ.guide) CLI (`go install github.com/a-h/templ/cmd/templ@latest`)
- AWS credentials configured
- Bedrock model access enabled in your AWS region

### Development

```sh
# Start dev server with hot reload
make dev

# Open http://localhost:8080
```

Dev mode uses local filesystem storage (no AWS resources needed). All four
watch processes run in parallel: `templ`, `tailwindcss`, Go rebuild, and
browser reload.

### Deploy

```sh
# Set required env vars
export BEDROCK_MODEL_ID="eu.anthropic.claude-3-5-sonnet-20241022-v2:0"

# Build and deploy
make build
make deploy
```

## Project Structure

```
app_scaffold/
├── cdk/          Infrastructure as Code (TypeScript)
├── cmd/api/      Go Lambda entry point
├── internal/     Go business logic (handlers, store, templates, config)
├── lambdas/      Python Lambda source
├── web/          Frontend source (CSS)
├── dist/         Build output (gitignored)
├── doc/          Documentation
└── Makefile
```

See [Project Structure](doc/structure.md) for details.

## License

MIT
