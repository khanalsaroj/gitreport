BINARY  := gitreport
VERSION ?= dev
LDFLAGS := -X github.com/khanalsaroj/gitreport/internal/version.Version=$(VERSION)

.PHONY: build test vet fmt fmt-check check clean

## build: compile the binary into ./$(BINARY)
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run the test suite with the race detector
test:
	go test -race ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go source
fmt:
	gofmt -w .

## fmt-check: fail if any Go source is not formatted
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files need 'make fmt':"; echo "$$unformatted"; exit 1; \
	fi

## check: run vet, format check, and tests
check: vet fmt-check test

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe
