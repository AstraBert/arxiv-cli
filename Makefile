.PHONY: build test clippy clippy-fix format format-check version-bump build-go test-go format-go lint-go

build:
	cargo build --target-dir target/

test:
	$(info ****************** running tests ******************)
	cargo test

clippy:
	$(info ****************** running clippy in check mode ******************)
	cargo clippy

clippy-fix:
	$(info ****************** running clippy in fix mode ******************)
	cargo clippy --fix --bin "arxiv-cli"

format:
	$(info ****************** running rustfmt in fix mode ******************)
	cargo fmt

format-check:
	$(info ****************** running rustfmt in check mode ******************)
	cargo fmt --check

version-bump:
	$(info ****************** bumping version in package.json and Cargo.toml ******************)
	python3 scripts/version_bump.py

npm-publish:
	$(info ****************** login and publish to npm ******************)
	$(info ****************** meant for manual usage ******************)
	bash scripts/login_and_publish_to_npm.sh

build-go:
	$(info ****************** building Go binary ******************)
	go build -C . -o bin/arxiv-cli ./cmd/arxiv-cli

test-go:
	$(info ****************** running Go tests ******************)
	go test ./...

format-go:
	$(info ****************** formatting Go code ******************)
	goimports -w .

lint-go:
	$(info ****************** running golangci-lint ******************)
	golangci-lint run