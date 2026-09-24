.PHONY: build test vet check e2e-linux

build:
	./scripts/build.sh

test:
	@if command -v go >/dev/null 2>&1; then \
		go test ./...; \
	else \
		docker run --rm -v "$(CURDIR)":/src -w /src golang:1.25 go test ./...; \
	fi

vet:
	@if command -v go >/dev/null 2>&1; then \
		for goos in linux darwin windows; do \
			GOOS=$$goos go vet ./...; \
		done; \
	else \
		for goos in linux darwin windows; do \
			docker run --rm -v "$(CURDIR)":/src -w /src -e GOOS=$$goos golang:1.25 go vet ./...; \
		done; \
	fi

check: test vet
	./scripts/check-text.sh
	./scripts/test-launcher.sh

e2e-linux:
	./scripts/e2e-linux.sh
