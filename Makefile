.PHONY: run build test coverage coverage-html lint hurl httpyac

run:
	go run .

build:
	go build -o todoapi .

test:
	go test ./...

coverage:
	go test ./... -v -coverprofile=coverage.out
# 	grep -v "main.go" coverage.out > coverage_filtered.out
# 	go tool cover -func=coverage_filtered.out

coverage-html:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out
# 	grep -v "main.go" coverage.out > coverage_filtered.out
# 	go tool cover -html=coverage_filtered.out

lint:
	pre-commit run --all-files

hurl:
	hurl --variables-file test/vars.env test/hurl/01_health.hurl test/hurl/02_auth.hurl test/hurl/03_todos.hurl

httpyac:
	httpyac send test/httpyac/*.http --all -e local -e local
