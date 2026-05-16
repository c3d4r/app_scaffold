# Technology Stack

## Compute

| Component | Runtime | Memory | Timeout | Architecture |
|-----------|---------|--------|---------|--------------|
| Go API Lambda | `provided.al2023` | 256 MB | 10 s | arm64 |
| Python Durable Lambda | Python 3.13 | 256 MB | 30 s | arm64 |

Both Lambdas use Graviton2 (arm64) for lower cost and better performance.

## Storage

| Resource | Purpose | Format |
|----------|---------|--------|
| S3 Static Bucket | CSS, future static assets | files |
| S3 Generated Bucket | Chat JSON + HTML fragments | JSON + HTML |

No DynamoDB or RDS in v1. All state is in S3 as JSON files.

## CDN

| Resource | Purpose |
|----------|---------|
| CloudFront | Single entry point, caching, OAC to S3 |

CloudFront Price Class 100 (North America + Europe) to minimize cost.

## IaC

| Tool | Language | Purpose |
|------|----------|---------|
| AWS CDK | TypeScript | Infrastructure definition and deployment |

CDK constructs define all AWS resources. Deployment via `make deploy`.

## Backend Dependencies

### Go (`go.mod`)

| Package | Purpose |
|---------|---------|
| `github.com/aws/aws-lambda-go` | Lambda runtime adapter |
| `github.com/aws/aws-sdk-go-v2` | AWS SDK core (modular) |
| `github.com/aws/aws-sdk-go-v2/service/s3` | S3 client |
| `github.com/aws/aws-sdk-go-v2/service/lambda` | Lambda invoke client |
| `github.com/aws/aws-sdk-go-v2/config` | AWS config loading |
| `github.com/a-h/templ` | HTML template engine (compile-time) |
| `github.com/google/uuid` | ID generation |

### Python (`lambdas/durable/requirements.txt`)

| Package | Purpose |
|---------|---------|
| (none) | `boto3` is built into the Lambda runtime |

No external Python packages needed in v1. `boto3` is provided by AWS Lambda's
Python runtime by default.

## Frontend Dependencies

| Library | Loaded From | Size | Purpose |
|---------|------------|------|---------|
| HTMX 2.x | `unpkg.com/htmx.org` | ~14 KB | Dynamic HTML, form handling, polling |
| Alpine.js 3.x | `unpkg.com/alpinejs` | ~15 KB | Client-side state, scroll management |
| Tailwind CSS | Compiled to `/static/styles.css` | ~3 KB | Styling |

HTMX and Alpine.js are loaded via CDN (`<script>` tags in the base template).
No npm install, no build step for JavaScript.

## Development Tools

| Tool | Purpose |
|------|---------|
| `templ` CLI | Generate Go code from `.templ` files |
| `@tailwindcss/cli` | Compile Tailwind CSS |
| `air` | Go live reload (dev mode only) |
| `make` | Task runner |

The development loop (`make dev`) runs four watch processes in parallel:
1. `templ generate --watch` — regenerates Go from `.templ` files
2. `tailwindcss --watch` — recompiles CSS
3. `air` — rebuilds and restarts Go server
4. `air` + `templ generate --notify-proxy` — reloads browser on asset changes

## Bedrock

| Setting | Default | Configurable |
|---------|---------|-------------|
| Model ID | `us.anthropic.claude-3-5-sonnet-20241022-v2:0` | `BEDROCK_MODEL_ID` env var |
| API | Converse (`bedrock-runtime.converse`) | N/A |
| Max tokens | 1024 | Code constant |
| Temperature | 0.7 | Code constant |

The Converse API is used for portability across Bedrock models. Any model
supporting the Converse API (Claude, Llama, Command R, etc.) works by
changing the model ID without code changes.
