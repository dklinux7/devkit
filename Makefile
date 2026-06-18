.PHONY: build test lint check update clean install

BINARY := devkit
CMD := .

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) $(CMD)

install:
	CGO_ENABLED=0 go install $(CMD)

test:
	go test ./...

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

licenses:
	go-licenses check ./... --allowed_licenses=MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC

check: test lint vuln licenses

update:
	go get -u ./...
	go mod tidy

clean:
	rm -f $(BINARY)
