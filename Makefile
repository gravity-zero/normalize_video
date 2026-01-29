init:
	sudo apt install mkvtoolnix
	go mod tidy
	go mod download

start:
	go run main.go

build:
	go mod tidy
	go mod download
	go build

test:
	go test ./test/unit/... -v

test-coverage:
	go test ./... -coverprofile=coverage.out -coverpkg=./files/...,./service/...,./types/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration:
	go test ./test/integration/... -v

test-all:
	go test ./test/... -v

clean:
	rm -f coverage.out coverage.html
	go clean