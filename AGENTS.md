# AGENTS.md — App Scaffold

## Quick reference

| Task | Command |
|------|---------|
| Dev server | `make dev` (listens on `:8080`) |
| Build all | `make build` |
| Deploy | `make deploy` (runs build, then CDK deploy) |
| Go check | `make check` (`go vet` + `go build ./...`) |
| Clean | `make clean` (removes `dist/*` and `tmp/`) |

## Architecture

Polyglot serverless HTML-over-the-wire app:
- **Go Lambda** (`cmd/api/`) — SSR via [templ](https://templ.guide), HTTP mux, ChatStore interface
- **Python Lambda** (`lambdas/durable/`) — async Bedrock Converse calls, writes HTML fragments to S3
- **TypeScript CDK** (`cdk/`) — separate npm project, defines all infra
- **Frontend** — HTMX + Alpine.js loaded from CDN; only Tailwind CSS needs compile

CloudFront routes: `/static/*` → S3, `/generated/*` → S3, `/*` → Go Lambda.

## Key conventions

- **`templ` is a build prerequisite.** `.templ` files generate `*_templ.go` (gitignored). Run `templ generate` before `go build`, or `make build` which does it for you.
- **`dist/` is the deployment artifact directory.** CDK reads from it. Structure: `dist/api/bootstrap` (Go binary), `dist/durable/main.py` (Python), `dist/static/styles.css` (Tailwind). All gitignored.
- **Dev vs. production** is controlled by `APP_ENV`. `APP_ENV=development` (default) uses local filesystem storage (`data/`) and no AWS. `APP_ENV=production` uses S3 + Lambda invoke.
- **Go package layout:** `cmd/api/` is the Lambda entrypoint (thin). `internal/` holds all business logic (Go convention — unimportable externally). Sub-packages: `config`, `handler`, `models`, `store`, `template`.
- **`ChatStore` interface** (`internal/store/store.go`) has two implementations: `S3Store` (prod) and `FSStore` (dev). Both satisfy the same `GetChat`/`SaveChat`/`GetFragment`/`PutFragment` contract.
- **Single-chat model in v1:** chat ID is hardcoded to `"default"`. No auth. Anyone visiting the URL shares the same conversation.
- **Go Lambda cross-compiles for arm64 Linux:** `GOOS=linux GOARCH=arm64 CGO_ENABLED=0`.
- **CDK bootstrap qualifier** is `"scaffold"` (set in `cdk/cdk.json`). If bootstrapping a new account, run `cdk bootstrap --qualifier scaffold`.
- **Python Lambda runtime** is `provided.al2023` for Go, Python 3.13 for Python Lambda. Both use arm64 (Graviton2).
- **CSS source** is `web/css/input.css`. Tailwind v4 with `@import "tailwindcss"`.
- **Root `package.json`** exists solely for Tailwind CSS (`tailwindcss` + `@tailwindcss/cli`). The CDK is an independent npm project.
- **No tests, no CI.** The repo currently has no `*_test.go` files and no GitHub workflows.

## Important env vars

| Variable | Default | Purpose |
|----------|---------|---------|
| `APP_ENV` | `development` | `"production"` enables S3 + Lambda invoke |
| `BEDROCK_MODEL_ID` | `us.anthropic.claude-sonnet-4-6` | Bedrock model for LLM calls |
| `GENERATED_BUCKET` | `app-scaffold-generated` | S3 bucket name for chat data |
| `DURABLE_LAMBDA_NAME` | `app-scaffold-durable` | Name of the Python Lambda function |

## Making changes

- **Template changes** (`.templ` files): edit the `.templ` file, then run `templ generate` (or `make build/templ`). The generated `*_templ.go` files go alongside the `.templ` source. Never edit generated files.
- **Adding Go dependencies:** `go get <pkg>` from repo root. Module is `github.com/c3d4r/app_scaffold`.
- **Adding Python dependencies:** add to `lambdas/durable/requirements.txt`, then update `cdk/lib/constructs/durable-lambda.ts` to include the layer or bundle.
- **CDK changes:** work in `cdk/`. Run `npm run build` (i.e. `tsc`) to typecheck before deploy. Deploy via `make deploy` from repo root (not from `cdk/` directly).
- **CSS changes:** edit `web/css/input.css`. Rebuild with `make build/css` or `make dev` for watch mode.
- **Adding routes:** handlers live in `internal/handler/`. Routes are registered in `handler.go` → `Routes()`. Templates in `internal/template/`.
