.PHONY: build run test clean

build:
	go build -o bin/api cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test ./...

clean:
	rm -rf bin/
