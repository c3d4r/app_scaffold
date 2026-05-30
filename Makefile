GENERATED_BUCKET=
DURABLE_LAMBDA_NAME=app-scaffold-durable
BEDROCK_MODEL_ID=us.anthropic.claude-sonnet-4-5-20250929-v1:0

GOPATH      := $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

.PHONY: dev build deploy clean check

# ─── Development ────────────────────────────────────────────────────────────────

dev: node_modules
	@echo "Starting dev environment..."
	make -j4 dev/templ dev/tailwind dev/server dev/reload

dev/templ:
	templ generate --watch --proxy="http://localhost:8080" --open-browser=false

dev/tailwind: node_modules
	npx @tailwindcss/cli -i web/css/input.css -o dist/static/styles.css --watch

dev/server:
	go run github.com/air-verse/air@v1.63.0 \
		--build.cmd "go build -o tmp/bin/api cmd/api/*.go" \
		--build.bin "tmp/bin/api" \
		--build.delay "100" \
		--build.exclude_dir "node_modules,cdk,dist,lambdas" \
		--build.include_ext "go,templ" \
		--build.stop_on_error "false" \
		--misc.clean_on_exit true

dev/reload:
	go run github.com/air-verse/air@v1.63.0 \
		--build.cmd "templ generate --notify-proxy" \
		--build.bin "true" \
		--build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "css"

node_modules:
	npm install tailwindcss @tailwindcss/cli

# ─── Build ──────────────────────────────────────────────────────────────────────

build: build/templ build/api build/durable build/css
	@echo "Build complete."

build/templ:
	templ generate

build/api:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/api/bootstrap cmd/api/*.go

build/durable:
	cp lambdas/durable/main.py dist/durable/
	test -f lambdas/durable/requirements.txt && cp lambdas/durable/requirements.txt dist/durable/ || true

build/css: node_modules
	npx @tailwindcss/cli -i web/css/input.css -o dist/static/styles.css --minify

# ─── Deploy ─────────────────────────────────────────────────────────────────────

deploy: build
	cd cdk && npm install && npx cdk deploy --require-approval never

API_LAMBDA_NAME = AppScaffoldStack-ApiLambdaFunction8FC74655-E9NEQkyxGvqQ
DURABLE_LAMBDA_NAME_DEPLOY = AppScaffoldStack-DurableLambdaFunctionB2837D43-dIh4oXXsr6SG
LAMBDA_REGION = us-east-1

deploy/fast: build/templ
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/api/bootstrap cmd/api/*.go
	cd dist/api && zip -q ../bootstrap.zip bootstrap && cd ../..
	aws lambda update-function-code --region $(LAMBDA_REGION) --function-name $(API_LAMBDA_NAME) --zip-file fileb://dist/bootstrap.zip
	rm -f dist/bootstrap.zip

deploy/fast-durable:
	cp lambdas/durable/main.py dist/durable/
	cd dist/durable && zip -q ../durable.zip main.py && cd ../..
	aws lambda update-function-code --region $(LAMBDA_REGION) --function-name $(DURABLE_LAMBDA_NAME_DEPLOY) --zip-file fileb://dist/durable.zip
	rm -f dist/durable.zip

deploy/fast-all: deploy/fast deploy/fast-durable

# ─── Utilities ──────────────────────────────────────────────────────────────────

clean:
	rm -rf dist/* tmp/

check:
	go vet ./cmd/... ./internal/...
	go build ./...
