.PHONY: run build test coverage coverage-html lint

run:
	go run .

build:
	go build -o todoapi .

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	pre-commit run --all-files
