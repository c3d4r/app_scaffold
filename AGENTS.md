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
- **Single-chat model in v1:** chat ID is hardcoded to `"default"`. Per-user sessions are handled via Cognito auth — each authenticated user gets their own session.
- **Auth is session-based:** session cookies (`scaffold_session`) stored client-side. Server-side session storage in S3 (prod) or filesystem (dev). Custom login/signup pages that call Cognito API directly (not hosted UI). Google/GitHub identity providers can be added later via Cognito federation.
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
| `BEDROCK_MODEL_ID` | `us.anthropic.claude-sonnet-4-5-20250929-v1:0` | Bedrock model for LLM calls |
| `GENERATED_BUCKET` | `app-scaffold-generated` | S3 bucket name for chat data |
| `DURABLE_LAMBDA_NAME` | `app-scaffold-durable` | Name of the Python Lambda function |
| `COGNITO_USER_POOL_ID` | (none) | Cognito User Pool ID |
| `COGNITO_CLIENT_ID` | (none) | Cognito app client ID |
| `COGNITO_CLIENT_SECRET` | (none) | Cognito app client secret |
| `COGNITO_DOMAIN` | (none) | Cognito domain prefix (e.g. `app-scaffold`) |
| `COGNITO_REGION` | (none) | AWS region for Cognito (e.g. `us-east-1`) |
| `CALLBACK_URL` | (derived from host) | Full callback URL for OAuth redirect |

## Auth

- **Open routes:** `/about` (public landing page), `/auth/login`, `/auth/signup`, `/auth/confirm`, `/auth/callback`, `/auth/logout`. All require no session.
- **Protected routes:** everything else (`/{chatId}`, etc.) wrapped in `authMiddleware`.
- **Dev mode (`APP_ENV=development`):** Cognito is optional. Without Cognito env vars, the middleware redirects to `/auth/dev-sign-in` which creates a fake dev session.
- **Prod mode (`APP_ENV=production`):** Cognito env vars are required. Middleware redirects to `/auth/login` (custom page) for sign-in.
- **Custom login:** `internal/handler/auth_pages.go` — login/signup/confirm handlers use Cognito API directly (USER_PASSWORD_AUTH flow with SECRET_HASH).
- **Cognito API client:** `internal/handler/cognito_api.go` — `InitiateAuth`, `SignUp`, `ConfirmSignUp` wrappers.
- **Session model:** `internal/auth/session.go` — `SessionStore` interface with S3 and FS implementations.
- **Cognito config:** `internal/auth/cognito.go` — JWT token verification (JWKS from Cognito endpoint). Also supports hosted UI callback for OAuth fallback.
- **Hosted UI:** `https://app-scaffold-336205929843.auth.us-east-1.amazoncognito.com` (User Pool: `us-east-1_jousDyMqY`, Client: `19qmbmf7k7qfj8p8at3d9pud7t`)
- **User management:** self-sign-up is disabled. Users must be created via admin CLI:
  ```
  # Create user (use non-email username — email is an alias)
  aws cognito-idp admin-create-user --region us-east-1 \
    --user-pool-id us-east-1_jousDyMqY \
    --username <username> \
    --user-attributes Name=email,Value=<email> Name=email_verified,Value=true \
    --message-action SUPPRESS

  # Set permanent password
  aws cognito-idp admin-set-user-password --region us-east-1 \
    --user-pool-id us-east-1_jousDyMqY \
    --username <username> \
    --password <password> \
    --permanent
  ```
  Test user: `testuser` (email: `test@ced4r.link`)

## Debugging production issues

- **AWS credentials** are set in the environment for `aws` CLI and CDK operations. Both `aws` and `cdk` commands work without additional configuration.
- **CloudWatch logs:** both Lambdas log to `/aws/lambda/<function-name>`. Check the Durable Lambda logs first for Bedrock errors, then the API Lambda logs for routing/parsing issues.
- **Check Lambda env vars:** `aws lambda get-function-configuration --function-name '<name>' --query 'Environment.Variables'`.
- **List available Bedrock models:** `aws bedrock list-foundation-models --region <region> --query 'modelSummaries[?contains(modelId, \`claude\`)].modelId'`.
- **Model lifecycle matters:** models reach end-of-life; some newer models require AWS Marketplace subscription. Verify status with `modelLifecycle.status` in the list-foundation-models output.
- **Lambda Function URL bodies are base64-encoded.** The bridge in `cmd/api/bridge.go` MUST check `req.IsBase64Encoded` and decode before passing to `http.Request`. Otherwise form values silently parse as empty.

## Making changes

- **Template changes** (`.templ` files): edit the `.templ` file, then run `templ generate` (or `make build/templ`). The generated `*_templ.go` files go alongside the `.templ` source. Never edit generated files.
- **Adding Go dependencies:** `go get <pkg>` from repo root. Module is `github.com/c3d4r/app_scaffold`.
- **Adding Python dependencies:** add to `lambdas/durable/requirements.txt`, then update `cdk/lib/constructs/durable-lambda.ts` to include the layer or bundle.
- **CDK changes:** work in `cdk/`. Run `npm run build` (i.e. `tsc`) to typecheck before deploy. Deploy via `make deploy` from repo root (not from `cdk/` directly).
- **CSS changes:** edit `web/css/input.css`. Rebuild with `make build/css` or `make dev` for watch mode.
- **Adding routes:** handlers live in `internal/handler/`. Routes are registered in `handler.go` → `Routes()`. Templates in `internal/template/`.
