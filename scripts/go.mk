BINARY      ?= thermal-tools
CMD_PATH    ?= ./cmd/thermal-tools

.PHONY: build test vet lint clean tidy

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(CMD_PATH)

test:
	go test -race ./...

test-integration:
	go test -race -tags=integration ./...

vet:
	go vet ./...

lint:
	gofmt -l .

tidy:
	go mod tidy

clean:
	rm -rf bin/
