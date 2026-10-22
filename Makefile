.PHONY: build build-dynamic cross cross-dynamic dev install test test-forced fmt vet tidy vendor ensure-vendor examples examples-check clean

# The build script is executed with -mod=mod on purpose: it is the only way to
# run it when the vendor directory is out of sync with go.mod, since Go
# defaults to the vendor directory whenever it is present.
build:
	go run -mod=mod build.go build static

build-dynamic:
	go run -mod=mod build.go build dynamic

cross:
	go run -mod=mod build.go cross static

cross-dynamic:
	go run -mod=mod build.go cross dynamic

dev:
	go run -mod=mod build.go dev

install: ensure-vendor
	go install -mod=vendor ./cmd/pre

test: ensure-vendor
	go test -mod=vendor ./...

test-forced: ensure-vendor
	go test -mod=vendor ./... -count=1

fmt:
	go fmt ./cmd/... ./internal/...
	gofmt -w examples/*.go

vet:
	go vet ./...

tidy:
	go mod tidy
	go mod vendor

ensure-vendor:
	@test -f vendor/modules.txt || $(MAKE) --no-print-directory vendor

vendor:
	go run -mod=mod build.go vendor

examples: build
	@for file in examples/*/*.pre; do \
		printf '==> %s\n' "$$file"; \
		./bin/pre --verbose "$$file" || exit 1; \
	done

examples-check: examples
	./examples/check.sh

clean:
	go run -mod=mod build.go clean
	rm -rf examples/*/generated
