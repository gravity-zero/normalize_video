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