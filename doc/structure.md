# Project Structure

```
app_scaffold/
├── cdk/                         # Infrastructure as Code (TypeScript)
│   ├── bin/
│   │   └── app.ts               # CDK app entry point
│   ├── lib/
│   │   ├── app-scaffold-stack.ts       # Main CloudFormation stack
│   │   └── constructs/                 # Reusable CDK constructs
│   │       ├── api-lambda.ts           # Go Lambda + Function URL
│   │       ├── cloudfront.ts           # CloudFront distribution
│   │       ├── durable-lambda.ts       # Python Lambda
│   │       ├── generated-bucket.ts     # S3 bucket for chat data
│   │       └── static-bucket.ts        # S3 bucket for CSS/assets
│   ├── package.json
│   ├── tsconfig.json
│   └── cdk.json
│
├── cmd/
│   └── api/                      # Go API Lambda entry point
│       ├── main.go              # Lambda handler + event adapter
│       ├── server.go            # HTTP mux setup + route registration
│       └── bridge.go            # LambdaFunctionURLRequest → http.Request
│
├── internal/                     # Private Go packages (API Lambda)
│   ├── config/
│   │   └── config.go            # Environment-driven configuration
│   ├── handler/
│   │   ├── home.go              # GET / (redirect) + GET /{chatId} (SSR page)
│   │   └── message.go           # POST /{chatId} (send) + GET /{chatId}/msgs/{msgId} (poll)
│   ├── models/
│   │   └── chat.go              # Chat and Message structs
│   ├── store/
│   │   ├── store.go             # ChatStore interface
│   │   ├── s3.go                # S3Store implementation
│   │   └── fs.go                # FSStore implementation (local dev)
│   └── template/
│       ├── base.templ           # <html>, <head>, <body> shell
│       ├── chat.templ           # Full chat page (message list + input)
│       ├── message.templ        # Single message bubble
│       ├── loader.templ         # Loading placeholder with polling
│       └── input.templ          # Chat input form
│
├── lambdas/
│   └── durable/                  # Python Durable Lambda
│       ├── main.py              # Lambda handler + Bedrock Converse + S3 writes
│       └── requirements.txt     # Empty (boto3 built-in)
│
├── web/
│   └── css/
│       └── input.css            # Tailwind CSS input (@import "tailwindcss")
│
├── dist/                         # Built artifacts (gitignored)
│   ├── api/
│   │   └── bootstrap            # Compiled Go binary for Lambda
│   ├── durable/
│   │   └── *.py                 # Python Lambda source (copied)
│   └── static/
│       └── styles.css           # Compiled Tailwind CSS
│
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

## Directory Rationale

### `cmd/` vs `internal/`

- **`cmd/api/`** — Lambda entry point only. Handles the Lambda runtime
  lifecycle, adapts events to `http.Request`, creates the server, and
  starts listening. Thin by design.

- **`internal/`** — All business logic. The `internal/` directory is a Go
  convention that prevents external packages from importing these packages.
  Contains handlers, store implementations, models, templates, and config.

### `cdk/`

Separate Node.js project for infrastructure. Uses `aws-cdk-lib` (v2).
TypeScript is the most mature CDK language with the broadest construct
library support. Even though the app uses Go and Python, the CDK code
is TypeScript — this is standard in polyglot projects.

### `lambdas/`

Python Lambda source. Kept separate from Go code since it's a different
language and runtime. The CDK construct copies this directory to the
deployment package. Could be reorganised if more Python Lambdas are added.

### `web/`

Frontend source files that require compilation. In v1, this is just
`css/input.css` for Tailwind. If future versions add images, fonts,
or TypeScript, they go here.

### `dist/`

Build output. Gitignored. The CDK deployment reads from `dist/`:
- `dist/api/bootstrap` — compiled Go binary
- `dist/durable/` — Python source (copied verbatim)
- `dist/static/styles.css` — compiled Tailwind CSS

## Naming Conventions

| Convention | Example |
|-----------|---------|
| Go packages | lowercase, single word: `handler`, `store`, `models` |
| Go files | lowercase, snake_case: `chat.go`, `s3_store.go` |
| Lambda entries | `cmd/{name}/main.go` |
| Templ files | lowercase, snake_case: `chat.templ`, `message.templ` |
| CDK constructs | kebab-case filenames: `api-lambda.ts` |
| CDK class names | PascalCase: `ApiLambda`, `CloudFrontConstruct` |
| Python files | lowercase, snake_case: `main.py` |
