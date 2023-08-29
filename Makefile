.PHONY: all
all: lint-fix lint tests install

vet:
	go vet ./...

tests: vet
	@go clean -testcache
	go test -race  ./... -coverpkg=./...  -coverprofile cover.out && go tool cover -func=cover.out

t: vet
	@go clean -testcache
	go test ./...

lint-fix:
	golangci-lint run --fix

lint:
	golangci-lint run --modules-download-mode vendor --timeout=20m -v

install:
	go install ./...