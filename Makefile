.PHONY: build test lint run

build:
	go build -o bin/aim-q ./main.go

test:
	go test ./... -v -cover

lint:
	golangci-lint run

run:
	go run main.go generate --uri=http://guest:guest@localhost:15672 --out=topology.puml

ci: test lint
