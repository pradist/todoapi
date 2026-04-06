.PHONY: run build test coverage coverage-html lint hurl

run:
	go run .

build:
	go build -o todoapi .

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out
# 	grep -v "main.go" coverage.out > coverage_filtered.out
# 	go tool cover -func=coverage_filtered.out

coverage-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
# 	grep -v "main.go" coverage.out > coverage_filtered.out
# 	go tool cover -html=coverage_filtered.out

lint:
	pre-commit run --all-files

hurl:
	hurl --variables-file test/vars.env test/01_health.hurl test/02_auth.hurl test/03_todos.hurl
